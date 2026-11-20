package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
)

func load(t *testing.T, file, path string) *Page {
	t.Helper()
	f, err := os.Open("testdata/" + file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := html.Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	p, err := Parse(path, doc)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func paths(bs []Block) []string {
	var out []string
	for _, b := range bs {
		out = append(out, b.Path)
	}
	return out
}

func TestSections(t *testing.T) {
	cases := []struct {
		file, path, citation, anchor, official string
		firstPaths                             []string
	}{
		{"usc_42_1983.html", "/uscode/text/42/1983", "42 U.S.C. § 1983", "1983", "https://uscode.house.gov/view.xhtml?req=granuleid:USC-prelim-title42-section1983&num=0&edition=prelim", []string{""}},
		{"usc_42_1988.html", "/uscode/text/42/1988", "42 U.S.C. § 1988", "1988", "https://uscode.house.gov/view.xhtml?req=granuleid:USC-prelim-title42-section1988&num=0&edition=prelim", []string{"(a)", "(b)", "(c)"}},
		{"cfr_45_164.502.html", "/cfr/text/45/164.502", "45 C.F.R. § 164.502", "164-502", "https://www.ecfr.gov/current/title-45/section-164.502", []string{"(a)", "(a)(1)", "(a)(1)(i)"}},
		{"frcp_26.html", "/rules/frcp/rule_26", "Fed. R. Civ. P. 26", "rule-26", "", []string{"(a)", "(a)(1)", "(a)(1)(A)", "(a)(1)(A)(i)"}},
		{"wv_reg_1-1-4.html", "/regulations/west-virginia/W-Va-C-S-R-SS-1-1-4", "W. Va. Code R. § 1-1-4", "1-1-4", "", []string{"4.1.", "4.1.a.", "4.1.b.", "4.1.b.1."}},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			p := load(t, c.file, c.path)
			if p.Kind != "section" || p.Citation != c.citation || p.Anchor != c.anchor || p.Official != c.official {
				t.Fatalf("got kind=%q citation=%q anchor=%q official=%q", p.Kind, p.Citation, p.Anchor, p.Official)
			}
			got := paths(p.Blocks)
			if len(got) < len(c.firstPaths) || strings.Join(got[:len(c.firstPaths)], " ") != strings.Join(c.firstPaths, " ") {
				t.Fatalf("paths start %q, want %q", got, c.firstPaths)
			}
			for _, b := range p.Blocks {
				if b.Text == "" {
					t.Fatalf("empty block %+v", b)
				}
			}
		})
	}
}

func TestUSCTextOnce(t *testing.T) {
	p := load(t, "usc_42_1988.html", "/uscode/text/42/1988")
	for _, b := range p.Blocks {
		if b.Depth == 0 {
			t.Fatalf("subsection text emitted twice: %+v", b)
		}
		if strings.Contains(b.Text, "R.S. §") {
			t.Fatalf("source credit leaked into %s", b.Path)
		}
	}
}

func TestTOC(t *testing.T) {
	p := load(t, "usc_toc.html", "/uscode/text/42/chapter-21/subchapter-I")
	if p.Kind != "toc" || len(p.Links) == 0 || p.Links[0].Path != "/uscode/text/42/1981" {
		t.Fatalf("got %+v", p)
	}
	for _, l := range p.Links {
		if l.Text == "prev" || l.Text == "next" {
			t.Fatalf("prev/next leaked: %+v", l)
		}
	}
	w := load(t, "wv_toc.html", "/regulations/west-virginia/agency-1/title-1/series-1-01")
	if w.Kind != "toc" {
		t.Fatalf("wv toc kind %q", w.Kind)
	}
}

func TestWithin(t *testing.T) {
	bs := []Block{{Path: "(a)"}, {Path: "(a)(1)"}, {Path: "(a)(1)(A)"}, {Path: "(a)(10)"}, {Path: "(b)"}}
	if got := strings.Join(paths(Within(bs, "(a)(1)")), " "); got != "(a)(1) (a)(1)(A)" {
		t.Fatalf("got %q", got)
	}
	rs := []Block{{Path: "4.1."}, {Path: "4.1.b."}, {Path: "4.10."}}
	if got := strings.Join(paths(Within(rs, "4.1.")), " "); got != "4.1. 4.1.b." {
		t.Fatalf("got %q", got)
	}
}

func TestRobots(t *testing.T) {
	r, err := parseRobots([]byte("User-agent: Googlebot\nDisallow: /\n\nUser-agent: *\nUser-agent: other\nCrawl-delay: 10\nAllow: /misc/*.css$\nDisallow: /misc/\nDisallow: /search/\n"))
	if err != nil {
		t.Fatal(err)
	}
	if r.delay != 10*time.Second {
		t.Fatalf("delay %v", r.delay)
	}
	for path, want := range map[string]bool{"/uscode/text/42/1983": true, "/search/site/x": false, "/misc/a.css": true, "/misc/a.js": false} {
		if got := r.allowed(path); got != want {
			t.Errorf("allowed(%s)=%v want %v", path, got, want)
		}
	}
}

func TestNormalize(t *testing.T) {
	for in, ok := range map[string]bool{"/uscode/text/42/1983": true, "https://www.law.cornell.edu/cfr/text/45/164.502": true, "http://www.law.cornell.edu/x": false, "https://example.com/x": false, "https://www.law.cornell.edu/search/site?q=x": false} {
		if _, err := normalize(in); (err == nil) != ok {
			t.Errorf("normalize(%q) err=%v", in, err)
		}
	}
}

func TestCFRInlineChild(t *testing.T) {
	p := load(t, "cfr_45_164.508.html", "/cfr/text/45/164.508")
	got := strings.Join(paths(p.Blocks[:3]), " ")
	if got != "(a) (a)(1) (a)(2)" {
		t.Fatalf("got %q", got)
	}
	if len(Within(p.Blocks, "(c)(1)")) == 0 {
		t.Fatal("(c)(1) missing")
	}
}

func TestRulesIndexIsTOC(t *testing.T) {
	p := load(t, "frcp_toc.html", "/rules/frcp")
	if p.Kind != "toc" || len(p.Links) == 0 {
		t.Fatalf("kind %q, %d links", p.Kind, len(p.Links))
	}
}

func TestRulesNotesSkipped(t *testing.T) {
	p := load(t, "frcp_26.html", "/rules/frcp/rule_26")
	for _, b := range p.Blocks {
		if b.Depth == 0 || strings.HasPrefix(b.Text, "(As amended") {
			t.Fatalf("note leaked: %.80s", b.Text)
		}
	}
}

func TestRobotsDelayCapped(t *testing.T) {
	r, _ := parseRobots([]byte("User-agent: *\nCrawl-delay: 999999999\n"))
	if r.delay != maxDelay {
		t.Fatalf("delay %v", r.delay)
	}
}

func TestSoftNotFound(t *testing.T) {
	f, _ := os.Open("testdata/wv_toc.html")
	defer f.Close()
	doc, _ := html.Parse(f)
	if _, err := Parse("/regulations/west-virginia/W-Va-Code-St-R-SS-78-3-1", doc); err == nil {
		t.Fatal("index page at a section address was accepted")
	}
}
