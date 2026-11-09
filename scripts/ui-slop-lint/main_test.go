package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func lint(t *testing.T, files map[string]string, mod ...func(*Options)) *Report {
	t.Helper()
	dir := writeFiles(t, files)
	opts := Options{Paths: []string{dir}, MaxRawHex: 12, Ignore: map[string]bool{}, Cwd: dir}
	for _, m := range mod {
		m(&opts)
	}
	rep, err := Scan(opts)
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func count(rep *Report, rule string) int {
	n := 0
	for _, f := range rep.Findings {
		if f.Rule == rule {
			n++
		}
	}
	return n
}

func find(rep *Report, rule string) *Finding {
	for i := range rep.Findings {
		if rep.Findings[i].Rule == rule {
			return &rep.Findings[i]
		}
	}
	return nil
}

type ruleCase struct {
	name string
	rule string
	file string
	src  string
	want int
}

// Cases here are single-file; project-level behavior has its own tests.
// Fixtures include a focus-visible and reduced-motion marker only where a
// project-gated rule is under test.
func TestLineRules(t *testing.T) {
	cases := []ruleCase{
		// 1 transition-all
		{"transition-all css", "transition-all", "a.css", `a{transition: all .3s}`, 1},
		{"transition-all property", "transition-all", "a.css", `a{transition-property: all}`, 1},
		{"transition-all tailwind variant", "transition-all", "a.html", `<div class="hover:transition-all duration-200">`, 1},
		{"transition-all negative", "transition-all", "a.css", `a{transition: opacity .2s}`, 0},
		{"transition-all token boundary", "transition-all", "a.html", `<div class="transition-allowed">`, 0},

		// 2 outline-removed
		{"outline css", "outline-removed", "a.css", `a:focus{outline:none}`, 1},
		{"outline zero", "outline-removed", "a.css", `a{outline: 0}`, 1},
		{"outline tailwind", "outline-removed", "a.html", `<input class="focus:outline-none">`, 1},
		{"outline replaced by focus-visible", "outline-removed", "a.css", "a{outline:none}\na:focus-visible{outline:2px solid red}", 0},
		{"outline replaced by tw focus-visible", "outline-removed", "a.html", `<a class="outline-none focus-visible:ring-2">`, 0},
		{"outline non-zero width", "outline-removed", "a.css", `a{outline: 0.5px solid red}`, 0},

		// 4 placeholder-copy
		{"lorem", "placeholder-copy", "a.html", `<p>Lorem ipsum</p>`, 1},
		{"bracket placeholder", "placeholder-copy", "a.html", `<h1>[Company Name]</h1>`, 1},
		{"placeholder css ignored", "placeholder-copy", "a.css", `/* lorem ipsum */ a{}`, 0},
		{"placeholder negative", "placeholder-copy", "a.html", `<p>Real copy</p>`, 0},

		// 5 placeholder-image
		{"via placeholder", "placeholder-image", "a.html", `<img src="https://via.placeholder.com/300" alt="">`, 1},
		{"picsum in css", "placeholder-image", "a.css", `a{background:url(https://picsum.photos/200)}`, 1},
		{"placeholder image negative", "placeholder-image", "a.html", `<img src="https://cdn.example.com/a.png" alt="">`, 0},
		{"placeholder image prefix word", "placeholder-image", "a.html", `<img src="https://myplaceholder.com/a.png" alt="">`, 0},

		// 6 viewport-100vw
		{"100vw width", "viewport-100vw", "a.css", `.a{width:100vw}`, 1},
		{"100vw calc min-width", "viewport-100vw", "a.css", `.a{min-width: calc(100vw - 2rem)}`, 1},
		{"w-screen", "viewport-100vw", "a.html", `<div class="w-screen">`, 1},
		{"w-[100vw]", "viewport-100vw", "a.html", `<div class="max-w-[100vw]">`, 1},
		{"100vw negative width percent", "viewport-100vw", "a.css", `.a{width:100%}`, 0},
		{"100vw negative max-w-screen-lg", "viewport-100vw", "a.html", `<div class="max-w-screen-lg">`, 0},
		{"100vw negative non-width", "viewport-100vw", "a.css", `.a{margin-left: calc(50% - 50vw); padding: 100vw}`, 0},

		// 7 zoom-disabled
		{"user-scalable no", "zoom-disabled", "a.html", `<meta name="viewport" content="width=device-width, user-scalable=no">`, 1},
		{"maximum-scale 1.0", "zoom-disabled", "a.html", `<meta name="viewport" content="width=device-width, maximum-scale=1.0">`, 1},
		{"zoom allowed", "zoom-disabled", "a.html", `<meta name="viewport" content="width=device-width, initial-scale=1">`, 0},
		{"maximum-scale 1.5", "zoom-disabled", "a.html", `<meta name="viewport" content="width=device-width, maximum-scale=1.5">`, 0},
		{"maximum-scale 5", "zoom-disabled", "a.html", `<meta name="viewport" content="maximum-scale=5">`, 0},
		{"non-viewport meta", "zoom-disabled", "a.html", `<meta name="description" content="user-scalable=no">`, 0},

		// 8 img-missing-alt
		{"img no alt", "img-missing-alt", "a.html", `<img src="a.png">`, 1},
		{"Image arrow handler no alt", "img-missing-alt", "a.tsx", `<Image src="a" onLoad={() => go()} />`, 1},
		{"img alt empty", "img-missing-alt", "a.html", `<img src="a.png" alt="">`, 0},
		{"img spread", "img-missing-alt", "a.tsx", `<img {...props} />`, 0},
		{"Image alt after arrow", "img-missing-alt", "a.tsx", `<Image src={x} onLoad={() => y()} alt="z"/>`, 0},
		{"svg image element", "img-missing-alt", "a.html", `<svg><image href="#a"></image></svg>`, 0},
		{"vue bound alt", "img-missing-alt", "a.vue", `<img :src="a" :alt="b">`, 0},

		// 9 purple-blue-gradient
		{"gradient hex pair", "purple-blue-gradient", "a.css", `a{background: linear-gradient(135deg, #6366f1, #8b5cf6)}`, 1},
		{"gradient names", "purple-blue-gradient", "a.css", `a{background: linear-gradient(to right, purple, blue)}`, 1},
		{"gradient tailwind", "purple-blue-gradient", "a.html", `<div class="bg-gradient-to-r from-purple-500 to-blue-500">`, 1},
		{"gradient neutral", "purple-blue-gradient", "a.css", `a{background: linear-gradient(#fff, #eee)}`, 0},
		{"gradient single family color", "purple-blue-gradient", "a.css", `a{background: linear-gradient(to right, #6366f1, #ffffff)}`, 0},
		{"gradient tailwind other end", "purple-blue-gradient", "a.html", `<div class="from-purple-500 to-red-500">`, 0},

		// 10 gradient-text
		{"background-clip text", "gradient-text", "a.css", `a{background-clip: text}`, 1},
		{"webkit and standard", "gradient-text", "a.css", "a{-webkit-background-clip:text;\nbackground-clip:text}", 2},
		{"bg-clip-text", "gradient-text", "a.html", `<h1 class="bg-clip-text text-transparent">`, 1},
		{"clip negative", "gradient-text", "a.css", `a{background-clip: padding-box}`, 0},
		{"bg-clip-border", "gradient-text", "a.html", `<div class="bg-clip-border">`, 0},

		// 11 default-indigo
		{"indigo bg", "default-indigo", "a.html", `<button class="bg-indigo-600">`, 1},
		{"violet hover text", "default-indigo", "a.html", `<a class="hover:text-violet-500">`, 1},
		{"indigo hex upper", "default-indigo", "a.css", `a{color:#6366F1}`, 1},
		{"indigo light shade", "default-indigo", "a.html", `<div class="bg-indigo-100">`, 0},
		{"blue not indigo", "default-indigo", "a.html", `<div class="bg-blue-600">`, 0},
		{"other hex", "default-indigo", "a.css", `a{color:#123456}`, 0},
		{"indigo hex not double reported in gradient", "default-indigo", "a.css", `a{background: linear-gradient(#6366f1, #8b5cf6)}`, 0},

		// 12 emoji-icon
		{"emoji", "emoji-icon", "a.html", `<button>🚀 Launch</button>`, 1},
		{"emoji run counted once", "emoji-icon", "a.html", `<p>✨🎉</p>`, 1},
		{"emoji separate", "emoji-icon", "a.html", `<p>🚀 and 🎉</p>`, 2},
		{"emoji negative typography", "emoji-icon", "a.html", `<p>Café — “quotes” → arrows</p>`, 0},
		{"emoji in css ignored", "emoji-icon", "a.css", `a::before{content:"🚀"}`, 0},

		// 15 ease-in-ui
		{"ease-in css", "ease-in-ui", "a.css", `a{transition: opacity .2s ease-in}`, 1},
		{"ease-in tailwind", "ease-in-ui", "a.html", `<div class="ease-in">`, 1},
		{"ease-in-out css", "ease-in-ui", "a.css", `a{transition: opacity .2s ease-in-out}`, 0},
		{"ease-in-out tailwind", "ease-in-ui", "a.html", `<div class="ease-in-out">`, 0},
		{"ease-out", "ease-in-ui", "a.css", `a{transition: opacity .2s ease-out}`, 0},

		// 16 scale-zero-entry
		{"scale(0)", "scale-zero-entry", "a.css", `a{transform: scale(0)}`, 1},
		{"scale prop zero", "scale-zero-entry", "a.css", `a{scale: 0}`, 1},
		{"scale-0 tailwind", "scale-zero-entry", "a.html", `<div class="scale-0">`, 1},
		{"scale 0.95", "scale-zero-entry", "a.css", `a{transform: scale(0.95)}`, 0},
		{"scale prop fraction", "scale-zero-entry", "a.css", `a{scale: 0.5}`, 0},
		{"scale-95", "scale-zero-entry", "a.html", `<div class="scale-95">`, 0},
		{"scale custom property", "scale-zero-entry", "a.css", `:root{--scale: 0}`, 0},

		// 17 long-ui-duration
		{"transition 600ms", "long-ui-duration", "a.css", `a{transition: opacity 600ms ease}`, 1},
		{"transition-duration .5s", "long-ui-duration", "a.css", `a{transition-duration: 0.5s}`, 1},
		{"duration-700", "long-ui-duration", "a.html", `<div class="duration-700">`, 1},
		{"long second layer", "long-ui-duration", "a.css", `a{transition: color .2s, transform 1s}`, 1},
		{"short transition", "long-ui-duration", "a.css", `a{transition: opacity 200ms}`, 0},
		{"long delay only", "long-ui-duration", "a.css", `a{transition: all .2s ease .8s}`, 0},
		{"keyframe animation excluded", "long-ui-duration", "a.css", `a{animation: spin 2s linear infinite}`, 0},
		{"duration-300", "long-ui-duration", "a.html", `<div class="duration-300">`, 0},
		{"bezier commas", "long-ui-duration", "a.css", `a{transition: transform .3s cubic-bezier(.1,.7,.9,1)}`, 0},

		// 18 eyebrow-label
		{"eyebrow tailwind", "eyebrow-label", "a.html", `<p class="text-xs uppercase tracking-widest">New</p>`, 1},
		{"eyebrow tailwind arbitrary tracking", "eyebrow-label", "a.tsx", `<p className="text-sm uppercase tracking-[0.15em]">New</p>`, 1},
		{"eyebrow css", "eyebrow-label", "a.css", `.eyebrow{text-transform:uppercase;letter-spacing:.12em}`, 1},
		{"eyebrow missing tracking", "eyebrow-label", "a.html", `<p class="text-xs uppercase">New</p>`, 0},
		{"eyebrow big text", "eyebrow-label", "a.html", `<p class="text-base uppercase tracking-wide">New</p>`, 0},
		{"eyebrow css split rules", "eyebrow-label", "a.css", `.a{text-transform:uppercase} .b{letter-spacing:.2em}`, 0},
		{"eyebrow css subtle tracking", "eyebrow-label", "a.css", `.a{text-transform:uppercase;letter-spacing:.01em}`, 0},

		// 19 colored-side-border
		{"side border css", "colored-side-border", "a.css", `a{border-left: 4px solid #3b82f6}`, 1},
		{"side border width only", "colored-side-border", "a.css", `a{border-right-width: 3px}`, 1},
		{"side border tailwind", "colored-side-border", "a.html", `<div class="border-l-4 border-l-blue-500">`, 1},
		{"side border hairline", "colored-side-border", "a.css", `a{border-left: 1px solid #ccc}`, 0},
		{"bottom border", "colored-side-border", "a.css", `a{border-bottom: 4px solid red}`, 0},
		{"side border tailwind uncolored", "colored-side-border", "a.html", `<div class="border-l-4">`, 0},
		{"side border tailwind thin", "colored-side-border", "a.html", `<div class="border-l-blue-500 border">`, 0},

		// 20 glass-blur
		{"backdrop css", "glass-blur", "a.css", `a{backdrop-filter: blur(8px)}`, 1},
		{"backdrop tailwind", "glass-blur", "a.html", `<div class="backdrop-blur-md">`, 1},
		{"plain filter", "glass-blur", "a.css", `a{filter: blur(4px)}`, 0},
		{"backdrop none", "glass-blur", "a.css", `a{backdrop-filter: none}`, 0},
		{"backdrop opacity", "glass-blur", "a.html", `<div class="backdrop-opacity-50">`, 0},

		// 21 copy-tells
		{"copy two words", "copy-tells", "a.html", `<p>Unlock seamless experiences</p>`, 2},
		{"copy phrase", "copy-tells", "a.html", `<p>Take your business to the next level</p>`, 1},
		{"copy join thousands", "copy-tells", "a.html", `<p>Join thousands of teams</p>`, 1},
		{"copy hyphenated", "copy-tells", "a.html", `<p>A game-changer</p>`, 1},
		{"copy inside style", "copy-tells", "a.html", `<style>/* robust seamless */ a{}</style><p>Hello</p>`, 0},
		{"copy inflected", "copy-tells", "a.html", `<p>Unlocked doors</p>`, 0},
		{"copy identifier", "copy-tells", "a.tsx", `const unlockAccount = () => 1`, 0},
		{"copy css ignored", "copy-tells", "a.css", `/* seamless */ a{}`, 0},

		// 23 will-change
		{"will-change", "will-change", "a.css", `a{will-change: transform}`, 1},
		{"will-change negative", "will-change", "a.css", `a{transition: transform .2s}`, 0},
		{"will-change custom property", "will-change", "a.css", `a{--will-change: 1}`, 0},

		// 24 pure-black-white
		{"black and white", "pure-black-white", "a.css", `a{color:#000; background:#FFF}`, 2},
		{"long form", "pure-black-white", "a.css", `a{color:#000000}`, 1},
		{"tinted", "pure-black-white", "a.css", `a{color:#111; background:#fafafa}`, 0},
		{"alpha variant", "pure-black-white", "a.css", `a{color:#0001}`, 0},
		{"black in token block", "pure-black-white", "a.css", `:root{--ink:#000}`, 0},
		{"fragment link", "pure-black-white", "a.html", `<a href="#fff">x</a>`, 0},

		// 25 tiny-text
		{"tiny css", "tiny-text", "a.css", `a{font-size: 10px}`, 1},
		{"tiny fractional", "tiny-text", "a.css", `a{font-size:11.5px}`, 1},
		{"tiny tailwind", "tiny-text", "a.html", `<span class="text-[10px]">`, 1},
		{"12px ok", "tiny-text", "a.css", `a{font-size: 12px}`, 0},
		{"110px ok", "tiny-text", "a.css", `a{font-size: 110px}`, 0},
		{"rem ok", "tiny-text", "a.css", `a{font-size: 1rem}`, 0},
		{"text-[12px] ok", "tiny-text", "a.html", `<span class="text-[12px]">`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.rule+"/"+tc.name, func(t *testing.T) {
			rep := lint(t, map[string]string{tc.file: tc.src})
			if got := count(rep, tc.rule); got != tc.want {
				t.Errorf("%s: got %d findings, want %d\n%+v", tc.rule, got, tc.want, rep.Findings)
			}
		})
	}
}

func TestEyebrowCSSReportsOpeningLine(t *testing.T) {
	rep := lint(t, map[string]string{"a.css": "/* heading */\n.eyebrow {\n  text-transform: uppercase;\n  letter-spacing: 0.12em;\n}\n"})
	f := find(rep, "eyebrow-label")
	if f == nil || f.Line != 2 || f.Col != 1 {
		t.Fatalf("want eyebrow-label at 2:1, got %+v", f)
	}
}

func TestMissingReducedMotion(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{"keyframes", map[string]string{"a.css": `@keyframes spin{}`}, 1},
		{"transition", map[string]string{"a.css": `a{transition: color .2s}`}, 1},
		{"tailwind animate", map[string]string{"a.html": `<div class="animate-spin">`}, 1},
		{"tailwind transition class", map[string]string{"a.html": `<div class="transition duration-200">`}, 1},
		{"transition none", map[string]string{"a.css": `a{transition: none}`}, 0},
		{"animation none", map[string]string{"a.css": `a{animation: none}`}, 0},
		{"no motion", map[string]string{"a.css": `a{color:red}`}, 0},
		{"reduced motion elsewhere", map[string]string{
			"a.css": `@keyframes spin{}`,
			"b.css": `@media (prefers-reduced-motion: reduce){*{animation:none}}`,
		}, 0},
		{"tailwind motion-reduce", map[string]string{
			"a.html": `<div class="animate-spin motion-reduce:animate-none">`,
		}, 0},
		{"tailwind motion-safe", map[string]string{
			"a.html": `<div class="motion-safe:animate-spin">`,
		}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rep := lint(t, tc.files)
			if got := count(rep, "missing-reduced-motion"); got != tc.want {
				t.Fatalf("got %d, want %d: %+v", got, tc.want, rep.Findings)
			}
			if f := find(rep, "missing-reduced-motion"); f != nil && (f.File != "" || f.Line != 0 || f.Tier != TierFloor) {
				t.Errorf("project-level finding should have empty file and line 0: %+v", f)
			}
		})
	}
}

func hexList(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, ".c%d{color:#%02x33aa}\n", i, i+1)
	}
	return b.String()
}

func TestRawHexSprawl(t *testing.T) {
	t.Run("over threshold lists top files", func(t *testing.T) {
		rep := lint(t, map[string]string{"big.css": hexList(10), "small.css": hexList(3)})
		f := find(rep, "raw-hex-sprawl")
		if f == nil {
			t.Fatalf("want a finding: %+v", rep.Findings)
		}
		if !strings.Contains(f.Message, "13 raw hex") || !strings.Contains(f.Message, "big.css (10)") || !strings.Contains(f.Message, "small.css (3)") {
			t.Errorf("message = %q", f.Message)
		}
		if f.File != "" || f.Line != 0 {
			t.Errorf("project-level finding should have empty file: %+v", f)
		}
	})
	t.Run("at threshold is fine", func(t *testing.T) {
		if rep := lint(t, map[string]string{"a.css": hexList(12)}); count(rep, "raw-hex-sprawl") != 0 {
			t.Fatalf("unexpected finding: %+v", rep.Findings)
		}
	})
	t.Run("custom threshold", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.css": hexList(2)}, func(o *Options) { o.MaxRawHex = 1 })
		if count(rep, "raw-hex-sprawl") != 1 {
			t.Fatalf("want finding: %+v", rep.Findings)
		}
	})
	t.Run("token blocks are excluded", func(t *testing.T) {
		var b strings.Builder
		for _, sel := range []string{":root", "@theme", `[data-theme="dark"]`, ".dark"} {
			fmt.Fprintf(&b, "%s {\n%s}\n", sel, strings.ReplaceAll(hexList(5), ".c", "--c"))
		}
		b.WriteString(".x{color:#112233}")
		rep := lint(t, map[string]string{"a.css": b.String()}, func(o *Options) { o.MaxRawHex = 1 })
		if count(rep, "raw-hex-sprawl") != 0 {
			t.Fatalf("hex in token blocks should not count: %+v", rep.Findings)
		}
	})
	t.Run("hex after token block closes counts", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.css": ":root{--a:#112233}\n.x{color:#223344;background:#334455}"},
			func(o *Options) { o.MaxRawHex = 1 })
		if f := find(rep, "raw-hex-sprawl"); f == nil || !strings.Contains(f.Message, "2 raw hex") {
			t.Fatalf("want 2 counted: %+v", rep.Findings)
		}
	})
	t.Run("fragments and entities are not colors", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.html": `<a href="#add">x</a><a href="/p#bee">y</a><p>&#039;</p><a href="#add-to-cart">z</a>`},
			func(o *Options) { o.MaxRawHex = 0 })
		if count(rep, "raw-hex-sprawl") != 0 {
			t.Fatalf("unexpected: %+v", rep.Findings)
		}
	})
}

const designFrontMatter = `---
name: Acme
colors:
  primary: "#336699"
  bg: '#abc'
  ink: #111111
  withalpha: "#11223344"
typography:
  body: "#ffffff"
---
# Prose
`

const designFenced = "# Design\n\n```yaml\ncolors:\n  accent: \"#FF0000\"\n  # comment\n  muted: '#00ff00'\nspacing:\n  s: \"#123456\"\n```\n"

func TestParseDesignColors(t *testing.T) {
	cases := []struct {
		name string
		text string
		want map[string]bool
	}{
		{"front matter", designFrontMatter, map[string]bool{"#336699": true, "#aabbcc": true, "#111111": true, "#112233": true}},
		{"fenced yaml", designFenced, map[string]bool{"#ff0000": true, "#00ff00": true}},
		{"both", designFrontMatter + designFenced, map[string]bool{
			"#336699": true, "#aabbcc": true, "#111111": true, "#112233": true, "#ff0000": true, "#00ff00": true}},
		{"no colors key", "---\nname: x\n---\n", map[string]bool{}},
		{"colors outside yaml ignored", "colors:\n  a: \"#123456\"\n", map[string]bool{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseDesignColors(tc.text); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestOffPalette(t *testing.T) {
	dir := writeFiles(t, map[string]string{"DESIGN.md": designFrontMatter})
	withDesign := func(o *Options) { o.DesignFile = filepath.Join(dir, "DESIGN.md") }

	t.Run("flags colors outside the palette", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.css": `a{color:#336699;background:#AABBCC;border-color:#222222}`}, withDesign)
		if got := count(rep, "off-palette-color"); got != 1 {
			t.Fatalf("got %d: %+v", got, rep.Findings)
		}
		if f := find(rep, "off-palette-color"); f.Tier != TierReflex || f.Match != "#222222" {
			t.Errorf("unexpected finding %+v", f)
		}
	})
	t.Run("black and white are finish tier", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.css": `a{color:#000;background:#fff}`}, withDesign)
		n := 0
		for _, f := range rep.Findings {
			if f.Rule == "off-palette-color" {
				n++
				if f.Tier != TierFinish {
					t.Errorf("want finish tier: %+v", f)
				}
			}
		}
		if n != 2 {
			t.Fatalf("got %d, want 2", n)
		}
	})
	t.Run("token blocks excluded", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.css": `:root{--x:#222222}`}, withDesign)
		if count(rep, "off-palette-color") != 0 {
			t.Fatalf("%+v", rep.Findings)
		}
	})
	t.Run("disabled without -design", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.css": `a{color:#222222}`})
		if count(rep, "off-palette-color") != 0 {
			t.Fatalf("%+v", rep.Findings)
		}
	})
	t.Run("disabled when palette is empty", func(t *testing.T) {
		empty := writeFiles(t, map[string]string{"DESIGN.md": "# nothing\n"})
		rep := lint(t, map[string]string{"a.css": `a{color:#222222}`}, func(o *Options) { o.DesignFile = filepath.Join(empty, "DESIGN.md") })
		if count(rep, "off-palette-color") != 0 {
			t.Fatalf("%+v", rep.Findings)
		}
	})
}

func TestWaivers(t *testing.T) {
	t.Run("valid reason on preceding line", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.css": "/* ui-slop-lint: allow glass-blur -- hero sits over a photo */\n.h{backdrop-filter: blur(8px)}\n"})
		f := find(rep, "glass-blur")
		if f == nil || !f.Waived || f.WaiverReason != "hero sits over a photo" {
			t.Fatalf("want waived with reason: %+v", f)
		}
		if rep.Summary.Waived != 1 || rep.Summary.Reflex != 0 {
			t.Errorf("summary = %+v", rep.Summary)
		}
	})
	t.Run("same line", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.css": ".h{backdrop-filter: blur(8px)} /* ui-slop-lint: allow glass-blur -- because */\n"})
		if f := find(rep, "glass-blur"); f == nil || !f.Waived {
			t.Fatalf("%+v", rep.Findings)
		}
	})
	t.Run("html comment trims closer", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.html": "<!-- ui-slop-lint: allow emoji-icon -- brand mascot -->\n<p>🦊</p>\n"})
		f := find(rep, "emoji-icon")
		if f == nil || !f.Waived || f.WaiverReason != "brand mascot" {
			t.Fatalf("%+v", f)
		}
	})
	t.Run("jsx comment and multiple ids", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.tsx": "{/* ui-slop-lint: allow glass-blur,will-change -- modal scrim */}\n<div style=\"backdrop-filter: blur(2px); will-change: opacity\"/>\n"})
		if !find(rep, "glass-blur").Waived || !find(rep, "will-change").Waived {
			t.Fatalf("%+v", rep.Findings)
		}
		if rep.Summary.Waived != 2 {
			t.Errorf("summary = %+v", rep.Summary)
		}
	})
	t.Run("two lines above does not apply", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.css": "/* ui-slop-lint: allow glass-blur -- why */\n\n.h{backdrop-filter: blur(8px)}\n"})
		if f := find(rep, "glass-blur"); f == nil || f.Waived {
			t.Fatalf("%+v", f)
		}
	})
	t.Run("other rule id does not apply", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.css": "/* ui-slop-lint: allow will-change -- why */\n.h{backdrop-filter: blur(8px)}\n"})
		if f := find(rep, "glass-blur"); f == nil || f.Waived {
			t.Fatalf("%+v", f)
		}
	})
	t.Run("missing reason does not apply and is reported", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.css": "/* ui-slop-lint: allow glass-blur */\n.h{backdrop-filter: blur(8px)}\n"})
		if f := find(rep, "glass-blur"); f == nil || f.Waived {
			t.Fatalf("%+v", f)
		}
		w := find(rep, "waiver-missing-reason")
		if w == nil || w.Tier != TierReflex || w.Line != 1 {
			t.Fatalf("want reflex waiver-missing-reason on line 1: %+v", w)
		}
		if rep.Summary.Waived != 0 || rep.Summary.Reflex != 2 {
			t.Errorf("summary = %+v", rep.Summary)
		}
	})
	t.Run("empty reason before closer", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.css": "/* ui-slop-lint: allow glass-blur -- */\n.h{backdrop-filter: blur(8px)}\n"})
		if count(rep, "waiver-missing-reason") != 1 || find(rep, "glass-blur").Waived {
			t.Fatalf("%+v", rep.Findings)
		}
	})
	t.Run("html closer without reason", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.html": "<!-- ui-slop-lint: allow emoji-icon -->\n<p>🦊</p>\n"})
		if count(rep, "waiver-missing-reason") != 1 || find(rep, "emoji-icon").Waived {
			t.Fatalf("%+v", rep.Findings)
		}
	})
	t.Run("floor rules are rejected but reported", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.css": "/* ui-slop-lint: allow transition-all -- it is fine */\na{transition: all 1s}\n@media (prefers-reduced-motion: reduce){a{transition:none}}\n"})
		if n := count(rep, "transition-all"); n != 1 {
			t.Fatalf("waiver comment must not self-trigger; got %d: %+v", n, rep.Findings)
		}
		f := find(rep, "transition-all")
		if f.Waived || !f.WaiverRejected {
			t.Fatalf("want waiver_rejected, not waived: %+v", f)
		}
		if rep.Summary.Floor != 1 || rep.Summary.Waived != 0 {
			t.Errorf("summary = %+v", rep.Summary)
		}
	})
	t.Run("no waiver for waiver-missing-reason", func(t *testing.T) {
		rep := lint(t, map[string]string{"a.css": "/* ui-slop-lint: allow waiver-missing-reason -- nope */\n/* ui-slop-lint: allow glass-blur */\n"})
		if w := find(rep, "waiver-missing-reason"); w == nil || w.Waived {
			t.Fatalf("%+v", rep.Findings)
		}
	})
}

func TestWaiverMissingReasonNegative(t *testing.T) {
	rep := lint(t, map[string]string{"a.css": "/* just a normal comment */\na{color:red}\n"})
	if count(rep, "waiver-missing-reason") != 0 {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestWalking(t *testing.T) {
	rep := lint(t, map[string]string{
		"src/a.css":              `a{transition: all 1s}`,
		"node_modules/x/b.css":   `a{transition: all 1s}`,
		"dist/c.css":             `a{transition: all 1s}`,
		"vendor/d.css":           `a{transition: all 1s}`,
		".git/e.css":             `a{transition: all 1s}`,
		"src/app.min.css":        `a{transition: all 1s}`,
		"src/page_templ.go":      `transition: all 1s`,
		"src/readme.txt":         `transition: all 1s`,
		"src/nested/page.gohtml": `<div class="transition-all">`,
	})
	if rep.Summary.FilesScanned != 2 || count(rep, "transition-all") != 2 {
		t.Fatalf("scanned %d, findings %+v", rep.Summary.FilesScanned, rep.Findings)
	}
}

func TestScanDedupesPathsAndSortsFindings(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"b.css": "a{will-change:opacity}\n",
		"a.css": "a{transition: all 1s; will-change: opacity}\n",
	})
	rep, err := Scan(Options{Paths: []string{dir, filepath.Join(dir, "a.css")}, MaxRawHex: 12, Ignore: map[string]bool{}, Cwd: dir})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.FilesScanned != 2 {
		t.Errorf("files scanned = %d", rep.Summary.FilesScanned)
	}
	var order []string
	for _, f := range rep.Findings {
		order = append(order, fmt.Sprintf("%s:%s:%s", f.Tier, f.File, f.Rule))
	}
	if !strings.HasPrefix(order[0], "floor:") || order[len(order)-1] != "finish:b.css:will-change" {
		t.Errorf("order = %v", order)
	}
	if order[len(order)-2] != "finish:a.css:will-change" {
		t.Errorf("finish findings must sort by file: %v", order)
	}
}

func TestRelativePathsAndColumns(t *testing.T) {
	dir := writeFiles(t, map[string]string{"sub/a.html": "<p>héllo 🚀</p>\n"})
	t.Chdir(dir)
	rep, err := Scan(Options{Paths: []string{"."}, MaxRawHex: 12, Ignore: map[string]bool{}})
	if err != nil {
		t.Fatal(err)
	}
	f := find(rep, "emoji-icon")
	if f == nil || f.File != "sub/a.html" || f.Line != 1 || f.Col != 10 {
		t.Fatalf("want sub/a.html:1:10 (rune columns), got %+v", f)
	}
}

func TestMatchTrimmedTo80(t *testing.T) {
	long := strings.Repeat("x", 200)
	rep := lint(t, map[string]string{"a.html": `<img src="` + long + `">`})
	f := find(rep, "img-missing-alt")
	if f == nil || len([]rune(f.Match)) != 80 {
		t.Fatalf("match length: %+v", f)
	}
}

func runCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestRunExitCodesAndOutput(t *testing.T) {
	floorDir := writeFiles(t, map[string]string{"a.css": "a{transition: all 1s}\n@media (prefers-reduced-motion: reduce){a{transition:none}}\n"})
	reflexDir := writeFiles(t, map[string]string{"a.css": "a{backdrop-filter: blur(2px)}\n"})
	cleanDir := writeFiles(t, map[string]string{"a.css": "a{color:#111}\n"})

	t.Run("floor finding exits 1", func(t *testing.T) {
		code, out, _ := runCLI(t, floorDir)
		if code != 1 {
			t.Fatalf("exit %d", code)
		}
		if !strings.Contains(out, "a.css:1:") || !strings.Contains(out, "[floor] transition-all:") {
			t.Errorf("text output: %q", out)
		}
		if !strings.Contains(out, "1 files · floor 1 · reflex ") || !strings.Contains(out, "· finish 0 · waived 0") {
			t.Errorf("summary line missing: %q", out)
		}
	})
	t.Run("reflex only exits 0", func(t *testing.T) {
		if code, out, _ := runCLI(t, reflexDir); code != 0 || !strings.Contains(out, "[reflex] glass-blur") {
			t.Fatalf("exit %d: %q", code, out)
		}
	})
	t.Run("clean exits 0", func(t *testing.T) {
		if code, out, _ := runCLI(t, cleanDir); code != 0 || !strings.Contains(out, "1 files · floor 0 · reflex 0 · finish 0 · waived 0") {
			t.Fatalf("exit %d: %q", code, out)
		}
	})
	t.Run("ignore disables a floor rule", func(t *testing.T) {
		if code, _, _ := runCLI(t, "-ignore", "transition-all,missing-reduced-motion", floorDir); code != 0 {
			t.Fatalf("exit %d", code)
		}
	})
	t.Run("flags may follow paths", func(t *testing.T) {
		code, out, _ := runCLI(t, floorDir, "-format", "json")
		if code != 1 || !strings.HasPrefix(out, "{") {
			t.Fatalf("exit %d: %q", code, out)
		}
	})
	t.Run("json shape", func(t *testing.T) {
		_, out, _ := runCLI(t, "-format", "json", floorDir)
		var got struct {
			Findings []map[string]any `json:"findings"`
			Summary  map[string]int   `json:"summary"`
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("invalid json: %v\n%s", err, out)
		}
		if got.Summary["files_scanned"] != 1 || got.Summary["floor"] < 1 {
			t.Errorf("summary = %v", got.Summary)
		}
		for _, k := range []string{"rule", "tier", "file", "line", "col", "match", "message"} {
			if _, ok := got.Findings[0][k]; !ok {
				t.Errorf("finding missing key %q: %v", k, got.Findings[0])
			}
		}
		if _, ok := got.Findings[0]["waived"]; ok {
			t.Errorf("waived should be omitted when false")
		}
	})
	t.Run("empty findings is an array", func(t *testing.T) {
		_, out, _ := runCLI(t, "-format", "json", cleanDir)
		if !strings.Contains(out, `"findings": []`) {
			t.Errorf("want empty array: %s", out)
		}
	})
	t.Run("waived finding printed in json", func(t *testing.T) {
		d := writeFiles(t, map[string]string{"a.css": "/* ui-slop-lint: allow glass-blur -- hero */\na{backdrop-filter: blur(2px)}\n"})
		code, out, _ := runCLI(t, "-format", "json", d)
		if code != 0 || !strings.Contains(out, `"waived": true`) || !strings.Contains(out, `"waiver_reason": "hero"`) {
			t.Fatalf("exit %d: %s", code, out)
		}
	})
	t.Run("rejected waiver in json", func(t *testing.T) {
		d := writeFiles(t, map[string]string{"a.css": "/* ui-slop-lint: allow transition-all -- ok */\na{transition: all 1s}\n"})
		code, out, _ := runCLI(t, "-format", "json", d)
		if code != 1 || !strings.Contains(out, `"waiver_rejected": true`) {
			t.Fatalf("exit %d: %s", code, out)
		}
	})
	t.Run("missing path is a usage error", func(t *testing.T) {
		if code, _, errOut := runCLI(t, filepath.Join(cleanDir, "nope")); code != 2 || errOut == "" {
			t.Fatalf("exit %d stderr %q", code, errOut)
		}
	})
	t.Run("bad format", func(t *testing.T) {
		if code, _, _ := runCLI(t, "-format", "xml", cleanDir); code != 2 {
			t.Fatalf("exit %d", code)
		}
	})
	t.Run("unknown flag", func(t *testing.T) {
		if code, _, _ := runCLI(t, "-bogus", cleanDir); code != 2 {
			t.Fatalf("exit %d", code)
		}
	})
	t.Run("unknown ignore id", func(t *testing.T) {
		if code, _, errOut := runCLI(t, "-ignore", "nope", cleanDir); code != 2 || !strings.Contains(errOut, "nope") {
			t.Fatalf("exit %d stderr %q", code, errOut)
		}
	})
	t.Run("missing design file", func(t *testing.T) {
		if code, _, _ := runCLI(t, "-design", filepath.Join(cleanDir, "DESIGN.md"), cleanDir); code != 2 {
			t.Fatalf("exit %d", code)
		}
	})
	t.Run("negative max-raw-hex", func(t *testing.T) {
		if code, _, _ := runCLI(t, "-max-raw-hex", "-1", cleanDir); code != 2 {
			t.Fatalf("exit %d", code)
		}
	})
	t.Run("max-raw-hex flag", func(t *testing.T) {
		d := writeFiles(t, map[string]string{"a.css": hexList(3)})
		_, out, _ := runCLI(t, "-max-raw-hex", "2", "-format", "json", d)
		if !strings.Contains(out, "raw-hex-sprawl") {
			t.Errorf("want raw-hex-sprawl: %s", out)
		}
	})
	t.Run("design flag", func(t *testing.T) {
		d := writeFiles(t, map[string]string{"a.css": "a{color:#222222}\n", "DESIGN.md": designFrontMatter})
		_, out, _ := runCLI(t, "-design", filepath.Join(d, "DESIGN.md"), "-format", "json", filepath.Join(d, "a.css"))
		if !strings.Contains(out, "off-palette-color") {
			t.Errorf("want off-palette-color: %s", out)
		}
	})
	t.Run("help exits 0", func(t *testing.T) {
		if code, _, _ := runCLI(t, "-h"); code != 0 {
			t.Fatalf("exit %d", code)
		}
	})
}

func TestListRules(t *testing.T) {
	code, out, _ := runCLI(t, "-list-rules")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if lines := strings.Split(strings.TrimSpace(out), "\n"); len(lines) != 25 {
		t.Errorf("want 25 rules, got %d", len(lines))
	}
	if !strings.Contains(out, "transition-all") || !strings.Contains(out, "floor") {
		t.Errorf("text: %s", out)
	}

	code, out, _ = runCLI(t, "-list-rules", "-format", "json")
	var got struct {
		Rules []struct{ ID, Tier, Description string } `json:"rules"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil || code != 0 || len(got.Rules) != 25 {
		t.Fatalf("json list: code %d err %v n=%d", code, err, len(got.Rules))
	}
	if got.Rules[0].ID != "transition-all" || got.Rules[0].Tier != "floor" || got.Rules[0].Description == "" {
		t.Errorf("first rule = %+v", got.Rules[0])
	}
}

func TestRuleTableIntegrity(t *testing.T) {
	seen := map[string]bool{}
	tiers := map[Tier]int{}
	for _, r := range rules {
		if seen[r.ID] {
			t.Errorf("duplicate rule %q", r.ID)
		}
		seen[r.ID] = true
		tiers[r.Tier]++
		if (r.Kind == KindLine) != (r.Line != nil) || (r.Kind == KindProject) != (r.Project != nil) {
			t.Errorf("rule %q: Kind does not match its check func", r.ID)
		}
		if r.Desc == "" {
			t.Errorf("rule %q has no description", r.ID)
		}
	}
	if tiers[TierFloor] != 8 || tiers[TierReflex] != 14 || tiers[TierFinish] != 3 {
		t.Errorf("tier counts = %v", tiers)
	}
}

func TestIgnoreRemovesRuleEntirely(t *testing.T) {
	rep := lint(t, map[string]string{"a.css": "a{transition: all 1s; will-change: opacity}"},
		func(o *Options) { o.Ignore = map[string]bool{"transition-all": true, "missing-reduced-motion": true} })
	if rep.Summary.Floor != 0 || count(rep, "will-change") != 1 {
		t.Fatalf("%+v", rep.Findings)
	}
}
