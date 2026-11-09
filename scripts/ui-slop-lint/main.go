// ui-slop-lint is a deterministic, read-only lint for AI-generated UI tells
// and basic UI-quality floor violations in frontend source.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	exitOK    = 0
	exitFloor = 1
	exitUsage = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ui-slop-lint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		format    = fs.String("format", "text", "output format: text or json")
		design    = fs.String("design", "", "DESIGN.md whose `colors:` map defines the palette for off-palette-color")
		maxHex    = fs.Int("max-raw-hex", 12, "raw-hex-sprawl threshold")
		ignore    = fs.String("ignore", "", "comma-separated rule ids to disable entirely")
		listRules = fs.Bool("list-rules", false, "print the rule table and exit")
	)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: ui-slop-lint [flags] PATH...  (default PATH is \".\"; flags may follow paths)")
		fs.PrintDefaults()
	}

	// The stdlib flag package stops at the first positional; loop so that
	// `ui-slop-lint . -format json` works.
	var paths []string
	rest := args
	for {
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return exitOK
			}
			return exitUsage
		}
		rest = fs.Args()
		if len(rest) == 0 {
			break
		}
		paths = append(paths, rest[0])
		rest = rest[1:]
	}

	if *format != "text" && *format != "json" {
		fmt.Fprintf(stderr, "ui-slop-lint: unsupported -format %q (want text or json)\n", *format)
		return exitUsage
	}
	if *maxHex < 0 {
		fmt.Fprintln(stderr, "ui-slop-lint: -max-raw-hex must be >= 0")
		return exitUsage
	}
	if *listRules {
		if err := writeRules(stdout, *format); err != nil {
			fmt.Fprintf(stderr, "ui-slop-lint: %v\n", err)
			return exitUsage
		}
		return exitOK
	}

	ignored, err := parseIgnore(*ignore)
	if err != nil {
		fmt.Fprintf(stderr, "ui-slop-lint: %v\n", err)
		return exitUsage
	}
	rep, err := Scan(Options{Paths: paths, DesignFile: *design, MaxRawHex: *maxHex, Ignore: ignored})
	if err != nil {
		fmt.Fprintf(stderr, "ui-slop-lint: %v\n", err)
		return exitUsage
	}
	if err := writeReport(stdout, *format, rep); err != nil {
		fmt.Fprintf(stderr, "ui-slop-lint: %v\n", err)
		return exitUsage
	}
	if rep.Summary.Floor > 0 {
		return exitFloor
	}
	return exitOK
}

func parseIgnore(s string) (map[string]bool, error) {
	ignored := map[string]bool{}
	if strings.TrimSpace(s) == "" {
		return ignored, nil
	}
	known := map[string]bool{}
	for _, r := range rules {
		known[r.ID] = true
	}
	for _, id := range strings.Split(s, ",") {
		id = strings.TrimSpace(id)
		if !known[id] {
			return nil, fmt.Errorf("unknown rule id %q in -ignore (see -list-rules)", id)
		}
		ignored[id] = true
	}
	return ignored, nil
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeRules(w io.Writer, format string) error {
	type row struct {
		ID          string `json:"id"`
		Tier        Tier   `json:"tier"`
		Description string `json:"description"`
	}
	rows := make([]row, 0, len(rules))
	for _, r := range rules {
		rows = append(rows, row{r.ID, r.Tier, r.Desc})
	}
	if format == "json" {
		return writeJSON(w, map[string]any{"rules": rows})
	}
	for _, r := range rows {
		if _, err := fmt.Fprintf(w, "%-24s %-7s %s\n", r.ID, r.Tier, r.Description); err != nil {
			return err
		}
	}
	return nil
}

func writeReport(w io.Writer, format string, rep *Report) error {
	if format == "json" {
		return writeJSON(w, rep)
	}
	var b strings.Builder
	for _, f := range rep.Findings {
		file := f.File
		if file == "" {
			file = "(project)"
		}
		fmt.Fprintf(&b, "%s:%d:%d: [%s] %s: %s", file, f.Line, f.Col, f.Tier, f.Rule, f.Message)
		if f.Match != "" {
			fmt.Fprintf(&b, " — %q", f.Match)
		}
		switch {
		case f.Waived:
			fmt.Fprintf(&b, " [waived: %s]", f.WaiverReason)
		case f.WaiverRejected:
			fmt.Fprintf(&b, " [waiver rejected: floor rules cannot be waived (%s)]", f.WaiverReason)
		}
		b.WriteByte('\n')
	}
	s := rep.Summary
	fmt.Fprintf(&b, "%d files · floor %d · reflex %d · finish %d · waived %d\n",
		s.FilesScanned, s.Floor, s.Reflex, s.Finish, s.Waived)
	_, err := io.WriteString(w, b.String())
	return err
}
