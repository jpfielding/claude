// context-audit walks ~/.claude/ read-only and flags context-management
// hygiene issues: oversized per-project MEMORY.md indexes
// (projects/*/memory/MEMORY.md), dead/orphan memory pointers, bloated
// SKILL.md bodies (including symlinked skills), reference files without a
// TOC, oversized agent descriptions, etc. Stdlib only.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Thresholds tuned to Claude Code's documented loading behavior.
const (
	memoryTruncateLines   = 200  // MEMORY.md truncates after 200 lines
	memoryIndexMaxLineLen = 150  // per auto-memory guidance
	skillBodySoftLimit    = 500  // skill-creator soft cap
	skillBodyHardLimit    = 750  // flag loudly above this
	referenceTocThreshold = 100  // reference files > this should have a TOC
	claudeMdSoftLimit     = 400  // keep global CLAUDE.md lean
	agentDescSoftLimit    = 1000 // agent descriptions load into every session's Agent tool listing
)

type severity int

const (
	sevHigh severity = iota
	sevMed
	sevLow
)

func (s severity) String() string {
	switch s {
	case sevHigh:
		return "HIGH"
	case sevMed:
		return "MED"
	default:
		return "LOW"
	}
}

type finding struct {
	sev      severity
	category string
	path     string
	message  string
}

type report struct {
	findings []finding
	stats    []kv
}

type kv struct {
	k, v string
}

func (r *report) add(sev severity, cat, path, msg string) {
	r.findings = append(r.findings, finding{sev, cat, path, msg})
}

func (r *report) stat(k, v string) {
	r.stats = append(r.stats, kv{k, v})
}

// countLines returns the line count of a file, or 0 on any error.
func countLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	n := 0
	for sc.Scan() {
		n++
	}
	return n
}

// readLines returns all lines of a file (nil on error).
func readLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	// strings.Split keeps an empty trailing element if file ends with \n; trim it.
	parts := strings.Split(string(data), "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// splitFrontmatter extracts the YAML frontmatter and body from a markdown file.
// Returns empty frontmatter if the file does not open with "---\n".
func splitFrontmatter(text string) (frontmatter, body string) {
	if !strings.HasPrefix(text, "---\n") {
		return "", text
	}
	rest := text[4:]
	end := strings.Index(rest, "\n---\n")
	if end == -1 {
		return "", text
	}
	return rest[:end], rest[end+5:]
}

var (
	linkRe = regexp.MustCompile(`\[[^\]]+\]\(([^)]+\.md)\)`)
	descRe = regexp.MustCompile(`(?ms)^description:\s*(.+?)(?:^\w|\z)`)
	tocRe  = regexp.MustCompile(`(?m)^- \[.+\]\(#`)
)

func auditMemory(root string, rep *report) {
	type idx struct{ index, memDir string }
	var indexes []idx

	// Legacy/global index at the root, if present.
	rootIndex := filepath.Join(root, "MEMORY.md")
	if _, err := os.Stat(rootIndex); err == nil {
		indexes = append(indexes, idx{rootIndex, filepath.Join(root, "memory")})
	}

	// Per-project auto-memory: <root>/projects/<slug>/memory/MEMORY.md.
	matches, _ := filepath.Glob(filepath.Join(root, "projects", "*", "memory", "MEMORY.md"))
	sort.Strings(matches)
	for _, m := range matches {
		indexes = append(indexes, idx{m, filepath.Dir(m)})
	}

	if len(indexes) == 0 {
		rep.add(sevLow, "MEMORY", filepath.Join(root, "projects", "*", "memory", "MEMORY.md"),
			"MEMORY.md not found — memory system unused or relocated")
		return
	}

	totalFiles, totalEntries := 0, 0
	for _, ix := range indexes {
		files, entries := auditMemoryIndex(ix.index, ix.memDir, rep)
		totalFiles += files
		totalEntries += entries
	}

	rep.stat("memory indexes", fmt.Sprintf("%d", len(indexes)))
	rep.stat("memory files on disk", fmt.Sprintf("%d", totalFiles))
	rep.stat("memory entries indexed", fmt.Sprintf("%d", totalEntries))
}

// auditMemoryIndex checks one MEMORY.md index against its memory directory
// and returns the number of memory files on disk and entries indexed.
func auditMemoryIndex(index, memDir string, rep *report) (files, entries int) {
	lines := readLines(index)

	switch {
	case len(lines) > memoryTruncateLines:
		rep.add(sevHigh, "MEMORY", index,
			fmt.Sprintf("%d lines > %d (system prompt truncates after %d) — entries past that line are invisible to Claude",
				len(lines), memoryTruncateLines, memoryTruncateLines))
	case len(lines) > int(float64(memoryTruncateLines)*0.8):
		rep.add(sevMed, "MEMORY", index,
			fmt.Sprintf("%d lines — approaching %d-line truncation limit", len(lines), memoryTruncateLines))
	}

	// Long lines in MEMORY.md — the index should hold one-liner pointers.
	var long []struct {
		line int
		size int
	}
	for i, ln := range lines {
		if len(ln) > memoryIndexMaxLineLen {
			long = append(long, struct {
				line int
				size int
			}{i + 1, len(ln)})
		}
	}
	if len(long) > 0 {
		var preview []string
		for i, ll := range long {
			if i >= 5 {
				break
			}
			preview = append(preview, fmt.Sprintf("L%d(%dc)", ll.line, ll.size))
		}
		extra := ""
		if len(long) > 5 {
			extra = fmt.Sprintf(" + %d more", len(long)-5)
		}
		rep.add(sevMed, "MEMORY", index,
			fmt.Sprintf("%d line(s) exceed %d chars — MEMORY.md should be a thin index, not hold content: %s%s",
				len(long), memoryIndexMaxLineLen, strings.Join(preview, ", "), extra))
	}

	// Cross-reference pointers against memory/ directory.
	referenced := map[string]struct{}{}
	for _, ln := range lines {
		for _, m := range linkRe.FindAllStringSubmatch(ln, -1) {
			referenced[m[1]] = struct{}{}
		}
	}

	existing := map[string]struct{}{}
	if dirEntries, err := os.ReadDir(memDir); err == nil {
		for _, e := range dirEntries {
			// Exclude the index itself, or it reports as its own orphan.
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && e.Name() != "MEMORY.md" {
				existing[e.Name()] = struct{}{}
			}
		}
	}

	var orphans, dead []string
	for name := range existing {
		if _, ok := referenced[name]; !ok {
			orphans = append(orphans, name)
		}
	}
	for name := range referenced {
		if _, ok := existing[name]; !ok {
			dead = append(dead, name)
		}
	}
	sort.Strings(orphans)
	sort.Strings(dead)

	if len(orphans) > 0 {
		preview := strings.Join(orphans[:min(6, len(orphans))], ", ")
		if len(orphans) > 6 {
			preview += "..."
		}
		rep.add(sevMed, "MEMORY", memDir,
			fmt.Sprintf("%d orphan memory file(s) — present on disk, not linked from MEMORY.md: %s",
				len(orphans), preview))
	}
	if len(dead) > 0 {
		preview := strings.Join(dead[:min(6, len(dead))], ", ")
		rep.add(sevHigh, "MEMORY", index,
			fmt.Sprintf("%d dead pointer(s) — linked from MEMORY.md but file missing: %s", len(dead), preview))
	}

	return len(existing), len(referenced)
}

func auditClaudeMd(root string, rep *report) {
	path := filepath.Join(root, "CLAUDE.md")
	if _, err := os.Stat(path); err != nil {
		return
	}
	n := countLines(path)
	rep.stat("CLAUDE.md lines", fmt.Sprintf("%d", n))
	if n > claudeMdSoftLimit {
		rep.add(sevMed, "CLAUDE_MD", path,
			fmt.Sprintf("%d lines > %d — global CLAUDE.md loads every session; move project-specific rules to per-project CLAUDE.md or skill references",
				n, claudeMdSoftLimit))
	}
}

// isDir reports whether path is a directory, following symlinks
// (os.DirEntry.IsDir is false for a symlink to a directory).
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func auditSkills(root string, rep *report) {
	skillsDir := filepath.Join(root, "skills")
	if _, err := os.Stat(skillsDir); err != nil {
		return
	}

	var skillDirs []string
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		top := filepath.Join(skillsDir, e.Name())
		if !isDir(top) {
			continue
		}
		if _, err := os.Stat(filepath.Join(top, "SKILL.md")); err == nil {
			skillDirs = append(skillDirs, top)
		}
		// One level deeper — plugin namespaces like .system/imagegen.
		subs, err := os.ReadDir(top)
		if err != nil {
			continue
		}
		for _, s := range subs {
			sub := filepath.Join(top, s.Name())
			if !isDir(sub) {
				continue
			}
			if _, err := os.Stat(filepath.Join(sub, "SKILL.md")); err == nil {
				skillDirs = append(skillDirs, sub)
			}
		}
	}
	sort.Strings(skillDirs)
	rep.stat("skills installed", fmt.Sprintf("%d", len(skillDirs)))

	for _, sd := range skillDirs {
		skillMd := filepath.Join(sd, "SKILL.md")
		data, err := os.ReadFile(skillMd)
		if err != nil {
			continue
		}
		text := string(data)
		frontmatter, body := splitFrontmatter(text)
		bodyLines := strings.Count(body, "\n")

		// Description checks.
		if m := descRe.FindStringSubmatch(frontmatter); m != nil {
			desc := strings.TrimSpace(m[1])
			if len(desc) < 60 {
				rep.add(sevMed, "SKILLS", skillMd,
					fmt.Sprintf("description is only %d chars — add concrete triggers/examples so Claude knows when to invoke", len(desc)))
			}
		} else {
			rep.add(sevHigh, "SKILLS", skillMd, "missing description in frontmatter — skill won't trigger correctly")
		}

		switch {
		case bodyLines > skillBodyHardLimit:
			rep.add(sevHigh, "SKILLS", skillMd,
				fmt.Sprintf("SKILL.md body is %d lines (> %d) — split into references/*.md (progressive disclosure)",
					bodyLines, skillBodyHardLimit))
		case bodyLines > skillBodySoftLimit:
			rep.add(sevMed, "SKILLS", skillMd,
				fmt.Sprintf("SKILL.md body is %d lines (> %d) — consider moving detail into references/",
					bodyLines, skillBodySoftLimit))
		}

		// Reference files over threshold should have a TOC.
		refsDir := filepath.Join(sd, "references")
		refEntries, err := os.ReadDir(refsDir)
		if err != nil {
			continue
		}
		for _, re := range refEntries {
			refPath := filepath.Join(refsDir, re.Name())
			if !strings.HasSuffix(re.Name(), ".md") || isDir(refPath) {
				continue
			}
			rlines := readLines(refPath)
			if len(rlines) <= referenceTocThreshold {
				continue
			}
			head := strings.ToLower(strings.Join(rlines[:min(30, len(rlines))], "\n"))
			if !strings.Contains(head, "table of contents") &&
				!strings.Contains(head, "## contents") &&
				!tocRe.MatchString(strings.Join(rlines[:min(30, len(rlines))], "\n")) {
				rep.add(sevLow, "SKILLS", refPath,
					fmt.Sprintf("reference is %d lines without a visible TOC in the first 30 lines — add one so Claude can preview scope",
						len(rlines)))
			}
		}
	}
}

func auditAgents(root string, rep *report) {
	dir := filepath.Join(root, "agents")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	n, descTotal := 0, 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		n++
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		frontmatter, _ := splitFrontmatter(string(data))
		m := descRe.FindStringSubmatch(frontmatter)
		if m == nil {
			rep.add(sevMed, "AGENTS", path, "missing description in frontmatter — agent won't be selected correctly")
			continue
		}
		desc := strings.TrimSpace(m[1])
		if len(desc) >= 2 && strings.HasPrefix(desc, `"`) && strings.HasSuffix(desc, `"`) {
			desc = desc[1 : len(desc)-1]
		}
		descTotal += len(desc)
		if len(desc) > agentDescSoftLimit {
			rep.add(sevMed, "AGENTS", path,
				fmt.Sprintf("description is %d chars (> %d) — it loads into every session's Agent tool listing; move examples/detail into the body",
					len(desc), agentDescSoftLimit))
		}
	}
	rep.stat("agents installed", fmt.Sprintf("%d", n))
	rep.stat("agent description chars (total)", fmt.Sprintf("%d", descTotal))
}

func auditSettings(root string, rep *report) {
	for _, name := range []string{"settings.json", "settings.local.json"} {
		p := filepath.Join(root, name)
		if _, err := os.Stat(p); err != nil {
			continue
		}
		rep.stat(name+" lines", fmt.Sprintf("%d", countLines(p)))
	}
}

func render(rep *report) string {
	var b strings.Builder
	b.WriteString(strings.Repeat("=", 60))
	b.WriteString("\nCONTEXT AUDIT\n")
	b.WriteString(strings.Repeat("=", 60))
	b.WriteByte('\n')

	if len(rep.stats) > 0 {
		b.WriteString("\nInventory:\n")
		for _, s := range rep.stats {
			fmt.Fprintf(&b, "  %s: %s\n", s.k, s.v)
		}
	}

	if len(rep.findings) == 0 {
		b.WriteString("\nNo findings. Context artifacts within expected thresholds.\n")
		return b.String()
	}

	sort.SliceStable(rep.findings, func(i, j int) bool {
		a, b := rep.findings[i], rep.findings[j]
		if a.sev != b.sev {
			return a.sev < b.sev
		}
		if a.category != b.category {
			return a.category < b.category
		}
		return a.path < b.path
	})

	var current severity = -1
	for _, f := range rep.findings {
		if f.sev != current {
			current = f.sev
			fmt.Fprintf(&b, "\n[%s]\n", f.sev)
		}
		fmt.Fprintf(&b, "  %s  %s\n    -> %s\n", f.category, f.path, f.message)
	}
	b.WriteByte('\n')
	return b.String()
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, p[1:])
		}
	}
	return p
}

func main() {
	root := flag.String("root", "~/.claude", "root directory to audit")
	flag.Parse()

	resolved := expandHome(*root)
	abs, err := filepath.Abs(resolved)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: resolving path: %v\n", err)
		os.Exit(2)
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "error: root not found: %s\n", abs)
		os.Exit(2)
	}

	var rep report
	auditMemory(abs, &rep)
	auditClaudeMd(abs, &rep)
	auditSkills(abs, &rep)
	auditAgents(abs, &rep)
	auditSettings(abs, &rep)

	fmt.Print(render(&rep))
}
