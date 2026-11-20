package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// Page is what one LII page yields: a table of contents (Links) or a
// section broken into labelled blocks.
type Page struct {
	URL      string  `json:"url"`
	Kind     string  `json:"kind"` // section | toc | unknown
	Title    string  `json:"title"`
	Citation string  `json:"citation,omitempty"`
	Heading  string  `json:"heading,omitempty"`
	Anchor   string  `json:"anchor,omitempty"`
	Official string  `json:"official,omitempty"`
	Links    []Link  `json:"links,omitempty"`
	Blocks   []Block `json:"blocks,omitempty"`
}

// Link is one entry of a table of contents.
type Link struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

// Block is one subdivision of a section. Path is the full designation, such
// as "(a)(1)(A)" or "4.1.b.1."; Depth is 1 for the top level, 0 for
// unlabelled text.
type Block struct {
	Path  string `json:"path,omitempty"`
	Label string `json:"label,omitempty"`
	Depth int    `json:"depth"`
	Text  string `json:"text"`
}

var (
	uscBlock   = map[string]bool{"subsection": true, "paragraph": true, "subparagraph": true, "clause": true, "subclause": true, "item": true, "subitem": true}
	psection   = regexp.MustCompile(`^psection-(\d+)$`)
	statBody   = regexp.MustCompile(`^statutory-body-(\d+)em$`)
	indentN    = regexp.MustCompile(`^indent(\d+)$`)
	leadLabel  = regexp.MustCompile(`^\(([^)\s]{1,8})\)`)
	spaces     = regexp.MustCompile(`\s+`)
	nonAnchor  = regexp.MustCompile(`[^a-z0-9]+`)
	uscPath    = regexp.MustCompile(`^/uscode/text/(\d+[a-zA-Z]?)/([0-9][0-9a-zA-Z.\-]*)$`)
	cfrPath    = regexp.MustCompile(`^/cfr/text/(\d+)/([0-9][0-9a-zA-Z.\-]*)$`)
	rulePath   = regexp.MustCompile(`^/rules/(frcp|frcrmp|fre|frap|frbp)/rule_([0-9a-zA-Z.\-]+)$`)
	ruleTitle  = regexp.MustCompile(`^Rule\s+([0-9a-zA-Z.\-]+)\.\s*(.*)$`)
	ruleCiteOf = map[string]string{"frcp": "Fed. R. Civ. P.", "frcrmp": "Fed. R. Crim. P.", "fre": "Fed. R. Evid.", "frap": "Fed. R. App. P.", "frbp": "Fed. R. Bankr. P."}
)

func classes(n *html.Node) []string {
	for _, a := range n.Attr {
		if a.Key == "class" {
			return strings.Fields(a.Val)
		}
	}
	return nil
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasClass(n *html.Node, c string) bool {
	for _, x := range classes(n) {
		if x == c {
			return true
		}
	}
	return false
}

func find(n *html.Node, pred func(*html.Node) bool) *html.Node {
	if n.Type == html.ElementNode && pred(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if f := find(c, pred); f != nil {
			return f
		}
	}
	return nil
}

func clean(s string) string { return strings.TrimSpace(spaces.ReplaceAllString(s, " ")) }

// text returns n's text, skipping any descendant for which skip is true.
func text(n *html.Node, skip func(*html.Node) bool) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			return
		}
		if n.Type == html.ElementNode {
			if skip != nil && skip(n) {
				return
			}
			switch n.Data {
			case "script", "style":
				return
			case "p", "div", "br", "li":
				b.WriteByte(' ')
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return clean(b.String())
}

// blockKind says whether n starts a block, and its depth (0 when the depth
// comes from nesting rather than a class).
func blockKind(n *html.Node) (kind string, depth int) {
	if n.Type != html.ElementNode {
		return "", 0
	}
	for _, c := range classes(n) {
		switch {
		case n.Data == "div" && uscBlock[c]:
			return "usc", 0
		case n.Data == "p" && psection.MatchString(c):
			d, _ := strconv.Atoi(psection.FindStringSubmatch(c)[1])
			return "cfr", d
		case n.Data == "div" && c == "subsect":
			for _, c2 := range classes(n) {
				if m := indentN.FindStringSubmatch(c2); m != nil {
					d, _ := strconv.Atoi(m[1])
					return "reg", d + 1
				}
			}
			return "reg", 1
		case n.Data == "p" && statBody.MatchString(c):
			d, _ := strconv.Atoi(statBody.FindStringSubmatch(c)[1])
			return "rule", d + 1
		}
	}
	return "", 0
}

func isBlock(n *html.Node) bool { k, _ := blockKind(n); return k != "" }

// skipped is publisher material, not text of the law: source credits and
// editorial notes.
func skipped(n *html.Node) bool {
	for _, c := range []string{"sourceCredit", "source-credit", "note", "notes", "note-body", "note-head"} {
		if hasClass(n, c) {
			return true
		}
	}
	return attr(n, "id") == "prevnext"
}

func parseBlocks(root *html.Node) []Block {
	var out []Block
	var stack []string // labels by depth, for kinds whose path is built from nesting
	push := func(depth int, label string) string {
		if depth < 1 {
			return ""
		}
		if len(stack) >= depth {
			stack = stack[:depth-1]
		}
		for len(stack) < depth-1 {
			stack = append(stack, "")
		}
		stack = append(stack, label)
		return strings.Join(stack, "")
	}
	var walk func(n *html.Node, uscDepth int)
	// uscDepth > 0 means n is inside a U.S. Code block, whose own text that
	// block already holds; only nested blocks are emitted from there.
	stopped := false
	walk = func(n *html.Node, uscDepth int) {
		if stopped {
			return
		}
		// Publisher notes follow the text of the law, and not every note
		// paragraph is classed: stop at the first note heading.
		if n.Type == html.ElementNode && hasClass(n, "note-head") {
			stopped = true
			return
		}
		if n.Type == html.ElementNode && skipped(n) {
			return
		}
		kind, depth := blockKind(n)
		switch kind {
		case "usc":
			uscDepth++
			label := ""
			if num := find(n, func(x *html.Node) bool { return hasClass(x, "num") }); num != nil {
				label = text(num, nil)
			}
			own := text(n, func(x *html.Node) bool { return x != n && (isBlock(x) || skipped(x)) })
			out = append(out, Block{Path: push(uscDepth, label), Label: label, Depth: uscDepth, Text: own})
		case "cfr":
			label, path := "", ""
			if e := find(n, func(x *html.Node) bool { return hasClass(x, "enumxml") }); e != nil {
				label = text(e, nil)
				if id := attr(e, "id"); id != "" {
					path = "(" + strings.ReplaceAll(id, "_", ")(") + ")"
				}
			}
			t := text(n, nil)
			// The CFR runs first children into their parent's paragraph:
			// "(a) Standard—(1) Rule—(i) Case. Text". Split each level off.
			for path != "" {
				head, rest, ok := strings.Cut(t, "—")
				child := leadLabel.FindString(rest)
				if !ok || child == "" {
					break
				}
				out = append(out, Block{Path: path, Label: label, Depth: depth, Text: head})
				path, label, depth, t = path+child, child, depth+1, rest
			}
			out = append(out, Block{Path: path, Label: label, Depth: depth, Text: t})
			return
		case "reg":
			label := ""
			if d := find(n, func(x *html.Node) bool { return hasClass(x, "designator") }); d != nil {
				label = text(d, nil)
			}
			own := text(n, func(x *html.Node) bool { return x != n && isBlock(x) })
			out = append(out, Block{Path: label, Label: label, Depth: depth, Text: own})
		case "rule":
			t := text(n, nil)
			label := ""
			if m := leadLabel.FindString(t); m != "" {
				label = m
			}
			out = append(out, Block{Path: push(depth, label), Label: label, Depth: depth, Text: t})
			return
		default:
			if n.Type == html.ElementNode && n.Data == "p" {
				if uscDepth > 0 {
					return
				}
				t := text(n, nil)
				if t == "" {
					return
				}
				// A plain <p> under a rule is its top level when it starts with
				// a label; otherwise it is unlabelled text.
				if m := leadLabel.FindString(t); m != "" {
					out = append(out, Block{Path: push(1, m), Label: m, Depth: 1, Text: t})
				} else {
					out = append(out, Block{Depth: 0, Text: t})
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, uscDepth)
		}
	}
	walk(root, 0)
	return out
}

func parseLinks(root *html.Node) []Link {
	var out []Link
	seen := map[string]bool{}
	var walk func(*html.Node, bool)
	walk = func(n *html.Node, inLi bool) {
		if n.Type == html.ElementNode {
			if skipped(n) {
				return
			}
			if n.Data == "li" {
				inLi = true
			}
			if n.Data == "a" && inLi {
				href := attr(n, "href")
				if strings.HasPrefix(href, "/") && !seen[href] {
					seen[href] = true
					out = append(out, Link{Path: href, Text: text(n, nil)})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, inLi)
		}
	}
	walk(root, false)
	return out
}

// Parse reads one LII page fetched from path.
func Parse(path string, doc *html.Node) (*Page, error) {
	p := &Page{URL: baseURL + path, Kind: "unknown"}
	if h1 := find(doc, func(n *html.Node) bool {
		id := attr(n, "id")
		return n.Data == "h1" && (id == "page_title" || id == "page-title")
	}); h1 != nil {
		p.Title = text(h1, nil)
	}
	root := find(doc, func(n *html.Node) bool { return attr(n, "id") == "tab_default_1" })
	if root == nil {
		root = find(doc, func(n *html.Node) bool { return attr(n, "id") == "content1" })
	}
	if root == nil {
		if mc := find(doc, func(n *html.Node) bool { return attr(n, "id") == "main-content" }); mc != nil {
			root = find(mc, func(n *html.Node) bool { return n.Data == "article" })
		}
	}
	if root == nil {
		return nil, fmt.Errorf("no content area found on %s (LII layout changed?)", path)
	}
	p.Blocks = parseBlocks(root)
	labelled := false
	for _, b := range p.Blocks {
		labelled = labelled || b.Depth > 0
	}
	// An index page may open with unlabelled prose; with links and no
	// labelled subdivisions it is a table of contents.
	if links := parseLinks(root); !labelled && len(links) > 0 {
		p.Kind, p.Links, p.Blocks = "toc", links, nil
	} else if len(p.Blocks) > 0 {
		p.Kind = "section"
	}
	if p.Kind == "section" {
		describe(path, p)
	} else if sectionPath(path) {
		// LII answers an unknown section with an index page and 200.
		return nil, fmt.Errorf("%s is a section address but LII answered %q (%s): no such section; walk the TOC from its parent", path, p.Title, p.Kind)
	}
	return p, nil
}

// describe fills the citation, heading, anchor and the official source's
// address for the collections whose official host is known.
func describe(path string, p *Page) {
	cite, head, _ := strings.Cut(p.Title, " - ")
	p.Citation, p.Heading = clean(cite), clean(head)
	switch {
	case uscPath.MatchString(path):
		m := uscPath.FindStringSubmatch(path)
		p.Citation = m[1] + " U.S.C. § " + m[2]
		p.Official = fmt.Sprintf("https://uscode.house.gov/view.xhtml?req=granuleid:USC-prelim-title%s-section%s&num=0&edition=prelim", m[1], m[2])
		p.Anchor = anchor(m[2])
	case cfrPath.MatchString(path):
		m := cfrPath.FindStringSubmatch(path)
		p.Citation = m[1] + " C.F.R. § " + m[2]
		p.Official = fmt.Sprintf("https://www.ecfr.gov/current/title-%s/section-%s", m[1], m[2])
		p.Anchor = anchor(m[2])
	case rulePath.MatchString(path):
		m := rulePath.FindStringSubmatch(path)
		p.Citation = ruleCiteOf[m[1]] + " " + m[2]
		if t := ruleTitle.FindStringSubmatch(p.Title); t != nil {
			p.Heading = clean(t[2])
		}
		p.Anchor = anchor("rule-" + m[2])
	default:
		if _, num, ok := strings.Cut(p.Citation, "§"); ok {
			p.Anchor = anchor(num)
		}
	}
}

func sectionPath(path string) bool {
	return uscPath.MatchString(path) || cfrPath.MatchString(path) || rulePath.MatchString(path) ||
		(strings.HasPrefix(path, "/regulations/") && strings.Contains(path, "-SS-"))
}

// anchor follows arca's grammar: lowercase letters, digits, hyphens.
func anchor(s string) string {
	return strings.Trim(nonAnchor.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

// Within keeps the blocks at or under sub: "(a)(1)" keeps "(a)(1)" and
// "(a)(1)(A)" but not "(a)(10)"; "4.1." keeps "4.1.b." but not "4.10.".
func Within(blocks []Block, sub string) []Block {
	var out []Block
	for _, b := range blocks {
		if b.Path == sub || (strings.HasPrefix(b.Path, sub) && (strings.HasSuffix(sub, ".") || strings.HasPrefix(b.Path[len(sub):], "("))) {
			out = append(out, b)
		}
	}
	return out
}
