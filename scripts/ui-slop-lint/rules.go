package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type RuleKind int

const (
	KindLine RuleKind = iota
	KindProject
)

// Hit is one match inside a file; Tier overrides the rule's tier when set.
type Hit struct {
	span
	Tier Tier
}

// Rule is one table entry. Line rules run per file; project rules run once
// over all files. Desc doubles as the finding message.
type Rule struct {
	ID      string
	Tier    Tier
	Desc    string
	Kind    RuleKind
	Applies func(FileKind) bool
	Line    func(p *Project, f *File) []Hit
	Project func(p *Project) []Finding
}

func (r *Rule) appliesTo(k FileKind) bool { return r.Applies == nil || r.Applies(k) }

func markupOnly(k FileKind) bool { return k == FileMarkup }

const ruleWaiverMissingReason = "waiver-missing-reason"

var rules = []Rule{
	{ID: "transition-all", Tier: TierFloor, Desc: "Animate specific properties (transform, opacity), not all.", Line: checkTransitionAll},
	{ID: "outline-removed", Tier: TierFloor, Desc: "Focus outline removed with no :focus-visible replacement.", Line: checkOutlineRemoved},
	{ID: "missing-reduced-motion", Tier: TierFloor, Kind: KindProject, Desc: "Motion present without a prefers-reduced-motion path.", Project: checkReducedMotion},
	{ID: "placeholder-copy", Tier: TierFloor, Desc: "Placeholder copy shipped.", Applies: markupOnly, Line: regexRule(rePlaceholderCopy)},
	{ID: "placeholder-image", Tier: TierFloor, Desc: "Placeholder image service; use a real asset or an honest empty slot.", Line: regexRule(rePlaceholderImage)},
	{ID: "viewport-100vw", Tier: TierFloor, Desc: "100vw includes scrollbar width and causes horizontal scroll; use 100% or overflow-x: clip.", Line: checkViewport100vw},
	{ID: "zoom-disabled", Tier: TierFloor, Desc: "Pinch-zoom disabled.", Applies: markupOnly, Line: checkZoomDisabled},
	{ID: "img-missing-alt", Tier: TierFloor, Desc: `Image without alt (use alt="" if decorative).`, Applies: markupOnly, Line: checkImgAlt},

	{ID: "purple-blue-gradient", Tier: TierReflex, Desc: "Two-stop purple/blue gradient is the #1 AI-UI tell.", Line: checkPurpleBlueGradient},
	{ID: "gradient-text", Tier: TierReflex, Desc: "Gradient text.", Line: checkGradientText},
	{ID: "default-indigo", Tier: TierReflex, Desc: "Framework-default indigo/violet accent.", Line: checkDefaultIndigo},
	{ID: "emoji-icon", Tier: TierReflex, Desc: "Emoji used as icon; use one consistent icon set (SVG).", Applies: markupOnly, Line: checkEmojiIcon},
	{ID: "raw-hex-sprawl", Tier: TierReflex, Kind: KindProject, Desc: "Raw hex values outside token blocks; route colors through tokens.", Project: checkRawHexSprawl},
	{ID: "off-palette-color", Tier: TierReflex, Desc: "Color not in DESIGN.md palette.", Line: checkOffPalette},
	{ID: "ease-in-ui", Tier: TierReflex, Desc: "ease-in feels sluggish on UI; use ease-out for enter/exit.", Line: tokenRule(reEaseIn)},
	{ID: "scale-zero-entry", Tier: TierReflex, Desc: "Animate from scale(0.95)+opacity, not scale(0).", Line: checkScaleZero},
	{ID: "long-ui-duration", Tier: TierReflex, Desc: "UI transitions should stay under ~300ms.", Line: checkLongDuration},
	{ID: "eyebrow-label", Tier: TierReflex, Desc: "Tracked-caps eyebrow label above heading (waive if intentional).", Line: checkEyebrow},
	{ID: "colored-side-border", Tier: TierReflex, Desc: "Colored side-stripe callout/card.", Line: checkColoredSideBorder},
	{ID: "glass-blur", Tier: TierReflex, Desc: "Decorative glass; justify or remove.", Line: checkGlassBlur},
	{ID: "copy-tells", Tier: TierReflex, Desc: "Generic marketing copy tell.", Applies: markupOnly, Line: checkCopyTells},
	{ID: ruleWaiverMissingReason, Tier: TierReflex, Desc: "Waiver has no reason; use `ui-slop-lint: allow <rule> -- <reason>`.", Line: checkWaiverMissingReason},

	{ID: "will-change", Tier: TierFinish, Desc: "will-change is a last resort.", Line: regexRule(reWillChange)},
	{ID: "pure-black-white", Tier: TierFinish, Desc: "Use tinted neutrals via tokens.", Line: checkPureBlackWhite},
	{ID: "tiny-text", Tier: TierFinish, Desc: "Text below 12px.", Line: checkTinyText},
}

// ---- regex helpers --------------------------------------------------------

func ci(p string) *regexp.Regexp { return regexp.MustCompile("(?i)" + p) }

// declRe matches a CSS property at a declaration boundary; a leading "-" is
// excluded so vendor prefixes and custom properties do not double-report.
func declRe(body string) *regexp.Regexp {
	return regexp.MustCompile(`(?im)(?:^|[^\w-])(` + body + `)`)
}

// regexHits reports group 1 when the pattern has one, else the whole match.
func regexHits(re *regexp.Regexp, s string) []Hit {
	var hits []Hit
	for _, m := range re.FindAllStringSubmatchIndex(s, -1) {
		a, b := m[0], m[1]
		if re.NumSubexp() >= 1 && m[2] >= 0 {
			a, b = m[2], m[3]
		}
		hits = append(hits, Hit{span: span{a, b}})
	}
	return hits
}

func isTokenChar(c byte) bool {
	return c == '-' || c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// tokenHits matches whole class tokens: "hover:ease-in" counts, "ease-in-out"
// and "--ease-in" do not.
func tokenHits(re *regexp.Regexp, s string) []Hit {
	var hits []Hit
	for _, m := range re.FindAllStringIndex(s, -1) {
		if m[0] > 0 && isTokenChar(s[m[0]-1]) {
			continue
		}
		if m[1] < len(s) && isTokenChar(s[m[1]]) {
			continue
		}
		hits = append(hits, Hit{span: span{m[0], m[1]}})
	}
	return hits
}

func hasToken(re *regexp.Regexp, s string) bool { return len(tokenHits(re, s)) > 0 }

func regexRule(re *regexp.Regexp) func(*Project, *File) []Hit {
	return func(_ *Project, f *File) []Hit { return regexHits(re, f.Scan) }
}

func tokenRule(re *regexp.Regexp) func(*Project, *File) []Hit {
	return func(_ *Project, f *File) []Hit { return tokenHits(re, f.Scan) }
}

// ---- floor ---------------------------------------------------------------

var (
	reTransitionAllCSS = declRe(`transition(?:-property)?\s*:\s*all\b`)
	reTransitionAllTW  = ci(`transition-all`)
)

func checkTransitionAll(_ *Project, f *File) []Hit {
	return append(regexHits(reTransitionAllCSS, f.Scan), tokenHits(reTransitionAllTW, f.Scan)...)
}

var (
	reOutlineCSS = declRe(`outline\s*:\s*["']?(?:none|0(?:px)?)["']?\s*(?:[;},!]|$)`)
	reOutlineTW  = ci(`outline-none`)
)

func checkOutlineRemoved(p *Project, f *File) []Hit {
	if p.hasFocusVisible {
		return nil
	}
	return append(regexHits(reOutlineCSS, f.Scan), tokenHits(reOutlineTW, f.Scan)...)
}

var (
	reKeyframes   = ci(`@keyframes\b`)
	reMotionDecl  = declRe(`(?:animation|transition)\s*:\s*([a-z-]*)`)
	reTWAnimate   = ci(`animate-(?:[a-z][a-z0-9-]*|\[[^\]\s]*\])`)
	reTWTransInfo = ci(`transition(?:-(?:all|colors|opacity|shadow|transform))?`)
)

// fileHasMotion is deliberately generous: any keyframes, non-"none" animation
// or transition declaration, or Tailwind animate-/transition class.
func fileHasMotion(f *File) bool {
	s := f.Scan
	if reKeyframes.MatchString(s) {
		return true
	}
	for _, m := range reMotionDecl.FindAllStringSubmatch(s, -1) {
		if !strings.EqualFold(m[2], "none") {
			return true
		}
	}
	for _, h := range tokenHits(reTWAnimate, s) {
		if !strings.EqualFold(s[h.Start:h.End], "animate-none") {
			return true
		}
	}
	for _, h := range tokenHits(reTWTransInfo, s) {
		if s[h.Start:h.End] != "transition" || f.inClassAttr(h.Start) {
			return true
		}
	}
	return false
}

func checkReducedMotion(p *Project) []Finding {
	if !p.hasMotion || p.hasReducedMotion {
		return nil
	}
	return []Finding{{}}
}

var (
	rePlaceholderCopy  = ci(`lorem\s+ipsum|dolor\s+sit\s+amet|\[(?:company|product|your)\s+name\]`)
	rePlaceholderImage = ci(`\b(?:placehold\.co|placeholder\.com|placekitten\.com|picsum\.photos|dummyimage\.com|loremflickr\.com)\b`)
)

var (
	reViewportCSS = regexp.MustCompile(`(?i)(?:^|[\s;{"'(])((?:min-|max-)?width["']?\s*:\s*["']?[^;{}"']*\b100vw\b)`)
	reViewportTW  = ci(`(?:min-|max-)?w-(?:screen|\[100vw\])`)
)

func checkViewport100vw(_ *Project, f *File) []Hit {
	hits := regexHits(reViewportCSS, f.Scan)
	for _, h := range tokenHits(reViewportTW, f.Scan) {
		if t := strings.ToLower(f.Scan[h.Start:h.End]); t == "w-screen" || strings.HasSuffix(t, "w-[100vw]") {
			hits = append(hits, h)
		}
	}
	return hits
}

var (
	reMetaTag  = regexp.MustCompile(`(?is)<meta\b[^>]*>`)
	reViewport = ci(`name\s*=\s*["']?viewport\b`)
	reNoZoom   = ci(`((?:user-scalable\s*=\s*(?:no|0)\b)|(?:maximum-scale\s*=\s*1(?:\.0+)?))(?:[^\d.]|$)`)
)

func checkZoomDisabled(_ *Project, f *File) []Hit {
	var hits []Hit
	for _, m := range reMetaTag.FindAllStringIndex(f.Scan, -1) {
		tag := f.Scan[m[0]:m[1]]
		if !reViewport.MatchString(tag) {
			continue
		}
		if z := reNoZoom.FindStringSubmatchIndex(tag); z != nil {
			hits = append(hits, Hit{span: span{m[0] + z[2], m[0] + z[3]}})
		}
	}
	return hits
}

var (
	reImgOpen = regexp.MustCompile(`(?:(?i:<img)|<Image)[\s/>]`)
	reAlt     = ci(`[\s:]alt(?:\s*=|[\s/>])`)
	reSpread  = ci(`\{\s*\.\.\.|v-bind\s*=`)
)

// tagEnd returns the index just past the tag's closing ">", honoring quotes
// and {} so JSX like onLoad={() => x} does not end the tag early.
func tagEnd(s string, i int) int {
	var quote byte
	depth := 0
	limit := i + 4000
	if limit > len(s) {
		limit = len(s)
	}
	for ; i < limit; i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'' || c == '`':
			quote = c
		case c == '{':
			depth++
		case c == '}':
			depth--
		case c == '>' && depth <= 0:
			return i + 1
		}
	}
	return -1
}

func checkImgAlt(_ *Project, f *File) []Hit {
	var hits []Hit
	for _, m := range reImgOpen.FindAllStringIndex(f.Scan, -1) {
		end := tagEnd(f.Scan, m[0]+1)
		if end < 0 {
			continue
		}
		tag := f.Scan[m[0]:end]
		if reAlt.MatchString(tag) || reSpread.MatchString(tag) {
			continue
		}
		hits = append(hits, Hit{span: span{m[0], end}})
	}
	return hits
}

// ---- reflex --------------------------------------------------------------

var (
	reGradient   = ci(`\b(?:linear|radial|conic)-gradient\(`)
	reColorName  = ci(`\b(?:purple|violet|indigo|blue)\b`)
	reTWFrom     = ci(`from-(?:purple|violet|indigo|fuchsia)-\d{3}`)
	reTWTo       = ci(`to-(?:blue|indigo|purple|cyan|pink|violet)-\d{3}`)
	purpleFamily = map[string]bool{
		"#6366f1": true, "#4f46e5": true, "#4338ca": true, "#8b5cf6": true,
		"#7c3aed": true, "#6d28d9": true, "#a855f7": true, "#9333ea": true,
		"#3b82f6": true, "#2563eb": true, "#06b6d4": true, "#ec4899": true,
	}
)

// purpleGradients returns the full span of each gradient( ... ) call that has
// at least two purple/blue-family stops.
func purpleGradients(f *File) []span {
	var out []span
	s := f.Scan
	for _, m := range reGradient.FindAllStringIndex(s, -1) {
		depth := 0
		end := -1
		for i := m[1] - 1; i < len(s) && i < m[1]+600; i++ {
			if s[i] == '(' {
				depth++
			} else if s[i] == ')' {
				if depth--; depth == 0 {
					end = i + 1
					break
				}
			}
		}
		if end < 0 {
			continue
		}
		args := s[m[1]:end]
		n := len(reColorName.FindAllString(args, -1))
		for _, h := range reHex.FindAllString(args, -1) {
			if norm, ok := normalizeHex(h); ok && purpleFamily[norm] {
				n++
			}
		}
		if n >= 2 {
			out = append(out, span{m[0], end})
		}
	}
	return out
}

func checkPurpleBlueGradient(_ *Project, f *File) []Hit {
	var hits []Hit
	for _, sp := range purpleGradients(f) {
		hits = append(hits, Hit{span: sp})
	}
	for _, h := range tokenHits(reTWFrom, f.Scan) {
		if hasToken(reTWTo, f.lineText(h.Start)) {
			hits = append(hits, h)
		}
	}
	return hits
}

var reGradientClip = declRe(`(?:-webkit-)?background-clip\s*:\s*text\b`)
var reTWClipText = ci(`bg-clip-text`)

func checkGradientText(_ *Project, f *File) []Hit {
	return append(regexHits(reGradientClip, f.Scan), tokenHits(reTWClipText, f.Scan)...)
}

var (
	reTWIndigo = ci(`(?:bg|text|border|ring|from|via|to|fill|stroke|outline|decoration|accent|shadow)-(?:indigo|violet|purple)-(?:400|500|600|700)`)
	indigoHex  = map[string]bool{"#6366f1": true, "#4f46e5": true, "#8b5cf6": true, "#7c3aed": true, "#a855f7": true}
)

func checkDefaultIndigo(_ *Project, f *File) []Hit {
	hits := tokenHits(reTWIndigo, f.Scan)
	grads := purpleGradients(f)
hexes:
	for _, h := range f.hexes {
		if !indigoHex[h.norm] {
			continue
		}
		for _, g := range grads {
			if h.Start >= g.Start && h.End <= g.End {
				continue hexes // already reported by purple-blue-gradient
			}
		}
		hits = append(hits, Hit{span: h.span})
	}
	return hits
}

func isEmoji(r rune) bool {
	return r >= 0x1F300 && r <= 0x1FAFF || r >= 0x2600 && r <= 0x27BF || r >= 0x1F000 && r <= 0x1F2FF
}

// checkEmojiIcon reports each run of adjacent emoji (with joiners/variation
// selectors) once.
func checkEmojiIcon(_ *Project, f *File) []Hit {
	var hits []Hit
	start := -1
	last := 0
	for i, r := range f.Scan {
		switch {
		case isEmoji(r):
			if start < 0 {
				start = i
			}
			last = i + len(string(r))
		case start >= 0 && (r == 0xFE0F || r == 0x200D):
			last = i + len(string(r))
		default:
			if start >= 0 {
				hits = append(hits, Hit{span: span{start, last}})
				start = -1
			}
		}
	}
	if start >= 0 {
		hits = append(hits, Hit{span: span{start, last}})
	}
	return hits
}

func checkRawHexSprawl(p *Project) []Finding {
	type count struct {
		file string
		n    int
	}
	var counts []count
	total := 0
	for _, f := range p.Files {
		n := 0
		for _, h := range f.hexes {
			if !h.inToken {
				n++
			}
		}
		if n > 0 {
			counts = append(counts, count{f.Rel, n})
			total += n
		}
	}
	if total <= p.Opts.MaxRawHex {
		return nil
	}
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].n != counts[j].n {
			return counts[i].n > counts[j].n
		}
		return counts[i].file < counts[j].file
	})
	var top []string
	for i := 0; i < len(counts) && i < 3; i++ {
		top = append(top, fmt.Sprintf("%s (%d)", counts[i].file, counts[i].n))
	}
	return []Finding{{Message: fmt.Sprintf(
		"%d raw hex values outside token blocks (max %d); route colors through tokens. Top files: %s",
		total, p.Opts.MaxRawHex, strings.Join(top, ", "))}}
}

func checkOffPalette(p *Project, f *File) []Hit {
	if len(p.Palette) == 0 {
		return nil
	}
	var hits []Hit
	for _, h := range f.hexes {
		if h.inToken || p.Palette[h.norm] {
			continue
		}
		hit := Hit{span: h.span}
		if h.norm == "#000000" || h.norm == "#ffffff" {
			hit.Tier = TierFinish
		}
		hits = append(hits, hit)
	}
	return hits
}

var reEaseIn = ci(`ease-in`)

var (
	reScaleFn   = ci(`\bscale\(\s*0\s*\)`)
	reScaleProp = regexp.MustCompile(`(?i)(?:^|[^\w-])(scale\s*:\s*0)(?:[^\d.\w%]|$)`)
	reTWScale0  = ci(`scale-0`)
)

func checkScaleZero(_ *Project, f *File) []Hit {
	hits := regexHits(reScaleFn, f.Scan)
	hits = append(hits, regexHits(reScaleProp, f.Scan)...)
	return append(hits, tokenHits(reTWScale0, f.Scan)...)
}

var (
	reDurationDecl = declRe(`transition(?:-duration)?\s*:[^;{}]{0,300}`)
	reParens       = regexp.MustCompile(`\([^()]*\)`)
	reTime         = ci(`(?:^|[\s,'":])(\d*\.?\d+)(ms|s)\b`)
	reTWLong       = ci(`duration-(?:500|700|1000)`)
)

// longTransition flags a duration >= 500ms. In the `transition` shorthand the
// second time per layer is a delay, so only the first counts.
func longTransition(durationsOnly bool, value string) bool {
	for i := 0; i < 3; i++ {
		value = reParens.ReplaceAllString(value, " ")
	}
	for _, layer := range strings.Split(value, ",") {
		times := reTime.FindAllStringSubmatch(layer, -1)
		if !durationsOnly && len(times) > 1 {
			times = times[:1]
		}
		for _, m := range times {
			v, err := strconv.ParseFloat(m[1], 64)
			if err != nil {
				continue
			}
			if strings.EqualFold(m[2], "s") {
				v *= 1000
			}
			if v >= 500 {
				return true
			}
		}
	}
	return false
}

func checkLongDuration(_ *Project, f *File) []Hit {
	var hits []Hit
	for _, m := range reDurationDecl.FindAllStringSubmatchIndex(f.Scan, -1) {
		decl := f.Scan[m[2]:m[3]]
		colon := strings.IndexByte(decl, ':')
		durationsOnly := strings.HasPrefix(strings.ToLower(decl), "transition-duration")
		if longTransition(durationsOnly, decl[colon+1:]) {
			hits = append(hits, Hit{span: span{m[2], m[3]}})
		}
	}
	return append(hits, tokenHits(reTWLong, f.Scan)...)
}

var (
	reTWUpper    = ci(`uppercase`)
	reTWTracking = ci(`tracking-(?:widest|wider|wide|\[0?\.1\d*em\])`)
	reTWSmall    = ci(`text-(?:xs|sm)`)
	reCSSUpper   = ci(`text-transform\s*:\s*uppercase`)
	reCSSTrack   = ci(`letter-spacing\s*:\s*(?:0?\.[12]\d*em|[2-9](?:\.\d+)?px)`)
)

// leafBlocks returns the {open, close} indexes of braces with no nested braces.
func leafBlocks(s string) []span {
	type frame struct {
		open  int
		child bool
	}
	var stack []frame
	var out []span
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			if n := len(stack); n > 0 {
				stack[n-1].child = true
			}
			stack = append(stack, frame{open: i})
		case '}':
			if n := len(stack); n > 0 {
				top := stack[n-1]
				stack = stack[:n-1]
				if !top.child {
					out = append(out, span{top.open, i})
				}
			}
		}
	}
	return out
}

// selectorStart walks back from a block's "{" to the first non-comment,
// non-space byte of its selector so the finding lands on the opening line.
func selectorStart(s string, open int) int {
	j := open
	for j > 0 && !strings.ContainsRune(";{}", rune(s[j-1])) {
		j--
	}
	for j < open {
		switch {
		case s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r':
			j++
		case strings.HasPrefix(s[j:open], "/*"):
			end := strings.Index(s[j:open], "*/")
			if end < 0 {
				return j
			}
			j += end + 2
		default:
			return j
		}
	}
	return j
}

func checkEyebrow(_ *Project, f *File) []Hit {
	var hits []Hit
	for _, a := range f.classAttrs {
		v := f.Scan[a.Start:a.End]
		if hasToken(reTWUpper, v) && hasToken(reTWTracking, v) && hasToken(reTWSmall, v) {
			hits = append(hits, Hit{span: a})
		}
	}
	for _, blk := range leafBlocks(f.Scan) {
		body := f.Scan[blk.Start+1 : blk.End]
		if reCSSUpper.MatchString(body) && reCSSTrack.MatchString(body) {
			hits = append(hits, Hit{span: span{selectorStart(f.Scan, blk.Start), blk.Start + 1}})
		}
	}
	return hits
}

var (
	reSideBorderCSS = declRe(`border-(?:left|right)(?:-width)?\s*:\s*[2-9]px`)
	reTWSideWidth   = ci(`border-[lr]-(?:2|4|8)`)
	reTWSideColor   = ci(`border-[lr]-[a-z]+-\d{3}`)
)

func checkColoredSideBorder(_ *Project, f *File) []Hit {
	hits := regexHits(reSideBorderCSS, f.Scan)
	for _, a := range f.classAttrs {
		v := f.Scan[a.Start:a.End]
		if !hasToken(reTWSideColor, v) {
			continue
		}
		for _, h := range tokenHits(reTWSideWidth, v) {
			hits = append(hits, Hit{span: span{a.Start + h.Start, a.Start + h.End}})
		}
	}
	return hits
}

var (
	reBackdropCSS = declRe(`backdrop-filter\s*:\s*blur`)
	reTWBackdrop  = ci(`backdrop-blur(?:-(?:[a-z0-9]+|\[[^\]\s]*\]))?`)
)

func checkGlassBlur(_ *Project, f *File) []Hit {
	return append(regexHits(reBackdropCSS, f.Scan), tokenHits(reTWBackdrop, f.Scan)...)
}

var (
	reCopyTells = ci(`\b(?:in today['’]s fast-paced|unlock the power|take your .{1,30} to the next level|look no further|join thousands|` +
		`delves?|seamless(?:ly)?|unlocks?|elevates?|revolutionize|revolutionary|game-changer|game-changing|cutting-edge|` +
		`supercharge|effortless(?:ly)?|tapestry|synergy|empowers?|leverage|robust)\b`)
	reStyleBlock  = regexp.MustCompile(`(?is)<style\b.*?</style>`)
	reScriptBlock = regexp.MustCompile(`(?is)<script\b.*?</script>`)
)

func checkCopyTells(_ *Project, f *File) []Hit {
	var skip []span
	for _, re := range []*regexp.Regexp{reStyleBlock, reScriptBlock} {
		for _, m := range re.FindAllStringIndex(f.Scan, -1) {
			skip = append(skip, span{m[0], m[1]})
		}
	}
	var hits []Hit
next:
	for _, h := range regexHits(reCopyTells, f.Scan) {
		for _, s := range skip {
			if h.Start >= s.Start && h.Start < s.End {
				continue next
			}
		}
		hits = append(hits, h)
	}
	return hits
}

func checkWaiverMissingReason(_ *Project, f *File) []Hit {
	var lines []int
	for line, w := range f.waivers {
		if !w.hasReason {
			lines = append(lines, line)
		}
	}
	sort.Ints(lines)
	var hits []Hit
	for _, line := range lines {
		hits = append(hits, Hit{span: f.waivers[line].span})
	}
	return hits
}

// ---- finish --------------------------------------------------------------

var reWillChange = declRe(`will-change\s*:`)

func checkPureBlackWhite(_ *Project, f *File) []Hit {
	var hits []Hit
	for _, h := range f.hexes {
		switch h.raw {
		case "#000", "#000000", "#fff", "#ffffff":
			if !h.inToken {
				hits = append(hits, Hit{span: h.span})
			}
		}
	}
	return hits
}

var (
	reTinyCSS = declRe(`font-size\s*:\s*(?:9|10|11)(?:\.\d+)?px`)
	reTinyTW  = ci(`text-\[(?:9|10|11)px\]`)
)

func checkTinyText(_ *Project, f *File) []Hit {
	return append(regexHits(reTinyCSS, f.Scan), tokenHits(reTinyTW, f.Scan)...)
}
