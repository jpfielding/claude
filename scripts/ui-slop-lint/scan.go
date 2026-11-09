package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// Tier orders findings by severity: floor findings fail the run.
type Tier string

const (
	TierFloor  Tier = "floor"
	TierReflex Tier = "reflex"
	TierFinish Tier = "finish"
)

var tierRank = map[Tier]int{TierFloor: 0, TierReflex: 1, TierFinish: 2}

// FileKind separates stylesheets from everything that may embed markup.
type FileKind int

const (
	FileStyle FileKind = iota
	FileMarkup
)

type Finding struct {
	Rule           string `json:"rule"`
	Tier           Tier   `json:"tier"`
	File           string `json:"file"`
	Line           int    `json:"line"`
	Col            int    `json:"col"`
	Match          string `json:"match"`
	Message        string `json:"message"`
	Waived         bool   `json:"waived,omitempty"`
	WaiverReason   string `json:"waiver_reason,omitempty"`
	WaiverRejected bool   `json:"waiver_rejected,omitempty"`
}

type Summary struct {
	FilesScanned int `json:"files_scanned"`
	Floor        int `json:"floor"`
	Reflex       int `json:"reflex"`
	Finish       int `json:"finish"`
	Waived       int `json:"waived"`
}

type Report struct {
	Findings []Finding `json:"findings"`
	Summary  Summary   `json:"summary"`
}

type Options struct {
	Paths      []string
	DesignFile string
	MaxRawHex  int
	Ignore     map[string]bool
	// Cwd anchors relative output paths; empty means os.Getwd.
	Cwd string
}

var (
	scanExts = map[string]FileKind{
		".css": FileStyle, ".scss": FileStyle,
		".html": FileMarkup, ".htm": FileMarkup, ".tsx": FileMarkup, ".jsx": FileMarkup,
		".vue": FileMarkup, ".svelte": FileMarkup, ".templ": FileMarkup, ".tmpl": FileMarkup,
		".gohtml": FileMarkup, ".astro": FileMarkup,
	}
	skipDirs = map[string]bool{
		".git": true, "node_modules": true, "dist": true, "build": true,
		"out": true, "vendor": true, ".next": true, "coverage": true,
	}
)

// span is a half-open byte range into a file's content.
type span struct{ Start, End int }

type waiver struct {
	ids       []string
	reason    string
	span      span
	hasReason bool
}

type hexHit struct {
	span
	raw     string // lowercased as written
	norm    string // lowercase #rrggbb, alpha dropped
	inToken bool
}

// File is one scanned source file plus the per-file indexes rules share.
type File struct {
	Rel  string
	Kind FileKind
	Raw  string
	// Scan equals Raw with waiver comments blanked (same byte offsets), so a
	// waiver that names a rule cannot itself trigger that rule.
	Scan string

	lineStarts  []int
	waivers     map[int]waiver // 1-based line -> waiver
	tokenBlocks []span
	hexes       []hexHit
	classAttrs  []span
}

// Project is the cross-file context handed to every rule.
type Project struct {
	Opts    Options
	Files   []*File
	Palette map[string]bool

	hasFocusVisible  bool
	hasMotion        bool
	hasReducedMotion bool
}

var (
	reWaiver = regexp.MustCompile(`(?i)ui-slop-lint:\s*allow\s+([a-z0-9-]+(?:\s*,\s*[a-z0-9-]+)*)(?:\s+--(?:\s+(.*))?)?`)
	// RE2 has no lookbehind, so the hex filter checks context after matching.
	reHex          = regexp.MustCompile(`#(?:[0-9a-fA-F]{8}|[0-9a-fA-F]{6}|[0-9a-fA-F]{3,4})\b`)
	reHexPrefix    = regexp.MustCompile(`(?i)(?:\b(?:href|to|src|action|id|for|name)\s*=|url\()\s*["']?$`)
	reTokenBlock   = regexp.MustCompile(`(?i)(?::root|@theme|\[data-theme[^\]]*\]|\.dark)(?:[\s,:>.\[+~][^{};]*)?\{`)
	reClassAttr    = regexp.MustCompile("(?i)\\b(?:class|className)\\s*=\\s*(?:\"([^\"]*)\"|'([^']*)'|`([^`]*)`|\\{((?:[^{}]|\\{[^{}]*\\})*)\\})")
	waiverClosers  = []string{"*/}}", "*/}", "*/", "-->", "--}}", "%}"}
	waiverEndMarks = []string{"*/", "-->"}
)

func newFile(rel string, kind FileKind, raw string) *File {
	f := &File{Rel: rel, Kind: kind, Raw: raw}
	f.lineStarts = []int{0}
	for i := 0; i < len(raw); i++ {
		if raw[i] == '\n' {
			f.lineStarts = append(f.lineStarts, i+1)
		}
	}
	f.parseWaivers()
	f.tokenBlocks = findTokenBlocks(f.Scan)
	f.hexes = f.findHexes()
	f.classAttrs = findClassAttrs(f.Scan)
	return f
}

func (f *File) lineAt(off int) int {
	return sort.Search(len(f.lineStarts), func(i int) bool { return f.lineStarts[i] > off })
}

func (f *File) colAt(line, off int) int {
	return utf8.RuneCountInString(f.Raw[f.lineStarts[line-1]:off]) + 1
}

func (f *File) lineText(off int) string {
	line := f.lineAt(off)
	end := len(f.Scan)
	if line < len(f.lineStarts) {
		end = f.lineStarts[line]
	}
	return f.Scan[f.lineStarts[line-1]:end]
}

func (f *File) parseWaivers() {
	f.waivers = map[int]waiver{}
	buf := []byte(f.Raw)
	for i, start := range f.lineStarts {
		end := len(f.Raw)
		if i+1 < len(f.lineStarts) {
			end = f.lineStarts[i+1] - 1
		}
		line := strings.TrimSuffix(f.Raw[start:end], "\r")
		m := reWaiver.FindStringSubmatchIndex(line)
		if m == nil {
			continue
		}
		w := waiver{}
		for _, id := range strings.Split(line[m[2]:m[3]], ",") {
			w.ids = append(w.ids, strings.ToLower(strings.TrimSpace(id)))
		}
		if m[4] >= 0 {
			w.reason = trimReason(line[m[4]:m[5]])
		}
		w.hasReason = w.reason != ""
		stop := len(line)
		for _, mark := range waiverEndMarks {
			if j := strings.Index(line[m[0]:], mark); j >= 0 && m[0]+j < stop {
				stop = m[0] + j
			}
		}
		w.span = span{start + m[0], start + stop}
		for j := w.span.Start; j < w.span.End; j++ {
			buf[j] = ' '
		}
		f.waivers[i+1] = w
	}
	f.Scan = string(buf)
}

func trimReason(r string) string {
	for {
		r = strings.TrimSpace(r)
		trimmed := false
		for _, c := range waiverClosers {
			if strings.HasSuffix(r, c) {
				r = strings.TrimSuffix(r, c)
				trimmed = true
			}
		}
		if !trimmed {
			return r
		}
	}
}

// findTokenBlocks locates :root, @theme, [data-theme...] and .dark bodies.
func findTokenBlocks(s string) []span {
	var blocks []span
	for _, m := range reTokenBlock.FindAllStringIndex(s, -1) {
		open := m[1] - 1
		depth := 0
		end := len(s)
		for i := open; i < len(s); i++ {
			if s[i] == '{' {
				depth++
			} else if s[i] == '}' {
				depth--
				if depth == 0 {
					end = i
					break
				}
			}
		}
		blocks = append(blocks, span{open, end})
	}
	return blocks
}

func findClassAttrs(s string) []span {
	var attrs []span
	for _, m := range reClassAttr.FindAllStringSubmatchIndex(s, -1) {
		for g := 1; g <= 4; g++ {
			if m[2*g] >= 0 {
				attrs = append(attrs, span{m[2*g], m[2*g+1]})
				break
			}
		}
	}
	return attrs
}

func (f *File) inClassAttr(off int) bool {
	for _, a := range f.classAttrs {
		if off >= a.Start && off < a.End {
			return true
		}
	}
	return false
}

func normalizeHex(raw string) (string, bool) {
	h := strings.ToLower(strings.TrimPrefix(raw, "#"))
	switch len(h) {
	case 3, 4:
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	case 6:
	case 8:
		h = h[:6]
	default:
		return "", false
	}
	return "#" + h, true
}

func (f *File) findHexes() []hexHit {
	var hexes []hexHit
	s := f.Scan
	for _, m := range reHex.FindAllStringIndex(s, -1) {
		a, b := m[0], m[1]
		if a > 0 {
			p := s[a-1]
			if isTokenChar(p) || p == '&' || p == '/' || p == '.' || p == '%' || p == '?' || p == '#' {
				continue
			}
		}
		if b < len(s) && (s[b] == '-' || s[b] == ':' || s[b] == '.' || s[b] == '[') {
			continue
		}
		rest := strings.TrimLeft(s[b:], " \t")
		if strings.HasPrefix(rest, "{") {
			continue // an #id selector, not a color
		}
		from := a - 16
		if from < 0 {
			from = 0
		}
		if reHexPrefix.MatchString(s[from:a]) {
			continue
		}
		norm, ok := normalizeHex(s[a:b])
		if !ok {
			continue
		}
		in := false
		for _, blk := range f.tokenBlocks {
			if a >= blk.Start && a < blk.End {
				in = true
				break
			}
		}
		hexes = append(hexes, hexHit{span: span{a, b}, raw: strings.ToLower(s[a:b]), norm: norm, inToken: in})
	}
	return hexes
}

func newProject(files []*File, opts Options, palette map[string]bool) *Project {
	p := &Project{Opts: opts, Files: files, Palette: palette}
	for _, f := range files {
		low := strings.ToLower(f.Scan)
		if strings.Contains(low, ":focus-visible") || strings.Contains(low, "focus-visible:") {
			p.hasFocusVisible = true
		}
		if strings.Contains(low, "prefers-reduced-motion") ||
			strings.Contains(low, "motion-reduce:") || strings.Contains(low, "motion-safe:") {
			p.hasReducedMotion = true
		}
		if !p.hasMotion && fileHasMotion(f) {
			p.hasMotion = true
		}
	}
	return p
}

// collectFiles expands paths into deduplicated, sorted scan targets.
func collectFiles(paths []string, cwd string) ([]*File, error) {
	seen := map[string]bool{}
	var files []*File
	add := func(path string) error {
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		kind, ok := kindFor(abs)
		if !ok || seen[abs] {
			return nil
		}
		seen[abs] = true
		b, err := os.ReadFile(abs)
		if err != nil {
			return err
		}
		files = append(files, newFile(relPath(cwd, abs), kind, string(b)))
		return nil
	}
	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if err := add(root); err != nil {
				return nil, err
			}
			continue
		}
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if path != root && skipDirs[d.Name()] {
					return fs.SkipDir
				}
				return nil
			}
			if !d.Type().IsRegular() {
				return nil
			}
			return add(path)
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Rel < files[j].Rel })
	return files, nil
}

func kindFor(path string) (FileKind, bool) {
	name := strings.ToLower(filepath.Base(path))
	if strings.HasSuffix(name, "_templ.go") || strings.HasSuffix(name, ".min.css") {
		return 0, false
	}
	k, ok := scanExts[filepath.Ext(name)]
	return k, ok
}

func relPath(cwd, abs string) string {
	if cwd != "" {
		if rel, err := filepath.Rel(cwd, abs); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(abs)
}

// Scan runs every enabled rule over the paths and returns a sorted report.
func Scan(opts Options) (*Report, error) {
	if len(opts.Paths) == 0 {
		opts.Paths = []string{"."}
	}
	cwd := opts.Cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	var palette map[string]bool
	if opts.DesignFile != "" {
		b, err := os.ReadFile(opts.DesignFile)
		if err != nil {
			return nil, err
		}
		palette = parseDesignColors(string(b))
		if len(palette) == 0 {
			fmt.Fprintf(os.Stderr, "ui-slop-lint: warning: no colors parsed from %s; off-palette-color disabled\n", opts.DesignFile)
		}
	}
	files, err := collectFiles(opts.Paths, cwd)
	if err != nil {
		return nil, err
	}
	return analyze(newProject(files, opts, palette)), nil
}

func analyze(p *Project) *Report {
	rep := &Report{Findings: []Finding{}}
	rep.Summary.FilesScanned = len(p.Files)
	for _, f := range p.Files {
		for i := range rules {
			r := &rules[i]
			if p.Opts.Ignore[r.ID] || r.Line == nil || !r.appliesTo(f.Kind) {
				continue
			}
			seen := map[span]bool{}
			for _, h := range r.Line(p, f) {
				if h.End > len(f.Raw) || h.Start < 0 || h.Start >= h.End || seen[h.span] {
					continue
				}
				seen[h.span] = true
				fd := f.finding(r, h)
				f.applyWaiver(&fd)
				rep.Findings = append(rep.Findings, fd)
			}
		}
	}
	for i := range rules {
		r := &rules[i]
		if p.Opts.Ignore[r.ID] || r.Project == nil {
			continue
		}
		for _, fd := range r.Project(p) {
			fd.Rule, fd.Tier = r.ID, r.Tier
			if fd.Message == "" {
				fd.Message = r.Desc
			}
			rep.Findings = append(rep.Findings, fd)
		}
	}
	sort.SliceStable(rep.Findings, func(i, j int) bool {
		a, b := rep.Findings[i], rep.Findings[j]
		switch {
		case tierRank[a.Tier] != tierRank[b.Tier]:
			return tierRank[a.Tier] < tierRank[b.Tier]
		case a.File != b.File:
			return a.File < b.File
		case a.Line != b.Line:
			return a.Line < b.Line
		case a.Col != b.Col:
			return a.Col < b.Col
		}
		return a.Rule < b.Rule
	})
	for _, fd := range rep.Findings {
		switch {
		case fd.Waived:
			rep.Summary.Waived++
		case fd.Tier == TierFloor:
			rep.Summary.Floor++
		case fd.Tier == TierReflex:
			rep.Summary.Reflex++
		case fd.Tier == TierFinish:
			rep.Summary.Finish++
		}
	}
	return rep
}

func (f *File) finding(r *Rule, h Hit) Finding {
	line := f.lineAt(h.Start)
	tier := r.Tier
	if h.Tier != "" {
		tier = h.Tier
	}
	return Finding{
		Rule: r.ID, Tier: tier, File: f.Rel, Line: line, Col: f.colAt(line, h.Start),
		Match: snippet(f.Raw[h.Start:h.End]), Message: r.Desc,
	}
}

// applyWaiver honors a waiver on the finding's line or the line above.
// A reasonless waiver never applies; floor rules reject even valid ones.
func (f *File) applyWaiver(fd *Finding) {
	if fd.Rule == ruleWaiverMissingReason {
		return
	}
	for _, line := range []int{fd.Line, fd.Line - 1} {
		w, ok := f.waivers[line]
		if !ok || !w.hasReason {
			continue
		}
		for _, id := range w.ids {
			if id != fd.Rule {
				continue
			}
			fd.WaiverReason = w.reason
			if fd.Tier == TierFloor {
				fd.WaiverRejected = true
			} else {
				fd.Waived = true
			}
			return
		}
	}
}

func snippet(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) > 80 {
		return string([]rune(s)[:80])
	}
	return s
}

var (
	reYAMLColorsKey = regexp.MustCompile(`^(\s*)colors:\s*(?:#.*)?$`)
	reYAMLColorLine = regexp.MustCompile(`^\s*(?:"[^"]*"|'[^']*'|[^\s:]+)\s*:\s*["']?(#[0-9a-fA-F]+)["']?\s*(?:#.*)?$`)
	reFenceOpen     = regexp.MustCompile("^\\s*```\\s*(?:ya?ml)\\s*$")
)

// parseDesignColors extracts only the `colors:` map from a DESIGN.md's front
// matter and ```yaml fences, returning normalized #rrggbb values.
func parseDesignColors(text string) map[string]bool {
	palette := map[string]bool{}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var regions [][]string
	i := 0
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i < len(lines) && strings.TrimSpace(lines[i]) == "---" {
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "---" {
				regions = append(regions, lines[i+1:j])
				i = j
				break
			}
		}
	}
	for ; i < len(lines); i++ {
		if !reFenceOpen.MatchString(lines[i]) {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			if strings.HasPrefix(strings.TrimSpace(lines[j]), "```") {
				regions = append(regions, lines[i+1:j])
				i = j
				break
			}
		}
	}
	for _, region := range regions {
		collectColors(region, palette)
	}
	return palette
}

func collectColors(region []string, palette map[string]bool) {
	for i := 0; i < len(region); i++ {
		m := reYAMLColorsKey.FindStringSubmatch(region[i])
		if m == nil {
			continue
		}
		indent := len(m[1])
		for i++; i < len(region); i++ {
			line := region[i]
			trim := strings.TrimSpace(line)
			if trim == "" || strings.HasPrefix(trim, "#") {
				continue
			}
			if len(line)-len(strings.TrimLeft(line, " \t")) <= indent {
				i--
				break
			}
			if c := reYAMLColorLine.FindStringSubmatch(line); c != nil {
				if norm, ok := normalizeHex(c[1]); ok {
					palette[norm] = true
				}
			}
		}
	}
}
