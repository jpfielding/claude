// Command lii reads pages of the Legal Information Institute (Cornell) for
// discovery: tables of contents, and sections split into their labelled
// subdivisions, with each section's citation, anchor and official address.
// It is polite: it obeys robots.txt and its Crawl-delay, sends one request at
// a time, and caches every page it fetches.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	baseURL   = "https://www.law.cornell.edu"
	userAgent = "lii-discovery/0.1 (research tool; one request at a time, obeys robots.txt)"
	pageTTL   = 7 * 24 * time.Hour
	robotsTTL = 24 * time.Hour
	maxBody   = 8 << 20
)

// Exit codes.
const (
	exitOK      = 0
	exitSetup   = 1
	exitUsage   = 2
	exitRefused = 3 // robots.txt disallows the path
	exitHTTP    = 4
)

type refusedError struct{ msg string }

func (e refusedError) Error() string { return e.msg }

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("lii", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cacheDir := fs.String("cache", os.Getenv("LII_CACHE"), "cache directory (default: user cache dir/lii)")
	refresh := fs.Bool("refresh", false, "fetch even when the page is cached")
	dryRun := fs.Bool("dry-run", false, "print the address and whether it is cached; fetch nothing")
	sub := fs.String("sub", "", `keep only this subdivision and what is under it, e.g. "(a)(1)" or "4.1.b."`)
	paths := fs.Bool("paths", false, "print only the block paths (an outline), not their text")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: lii [flags] get <path or https://www.law.cornell.edu/... url>")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitUsage
	}
	if fs.NArg() != 2 || fs.Arg(0) != "get" {
		fs.Usage()
		return exitUsage
	}
	path, err := normalize(fs.Arg(1))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	dir := *cacheDir
	if dir == "" {
		d, err := os.UserCacheDir()
		if err != nil {
			fmt.Fprintln(stderr, "no cache dir:", err)
			return exitSetup
		}
		dir = filepath.Join(d, "lii")
	}
	if err := os.MkdirAll(filepath.Join(dir, "pages"), 0o755); err != nil {
		fmt.Fprintln(stderr, err)
		return exitSetup
	}
	f := &fetcher{dir: dir, client: &http.Client{
		Timeout: 30 * time.Second,
		// Never follow a redirect: it would be fetched without the robots
		// check or the delay. A 3xx is reported instead.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}

	if *dryRun {
		_, cached := f.cached(path)
		fmt.Fprintf(stdout, "GET %s%s (cached: %v)\n", baseURL, path, cached && !*refresh)
		return exitOK
	}
	body, err := f.get(path, *refresh)
	if err != nil {
		fmt.Fprintln(stderr, err)
		var r refusedError
		if errors.As(err, &r) {
			return exitRefused
		}
		return exitHTTP
	}
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		fmt.Fprintln(stderr, "parse:", err)
		return exitHTTP
	}
	page, err := Parse(path, doc)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitHTTP
	}
	if *sub != "" {
		page.Blocks = Within(page.Blocks, *sub)
		if len(page.Blocks) == 0 {
			fmt.Fprintf(stderr, "no subdivision %q on %s\n", *sub, path)
			return exitUsage
		}
	}
	if *paths {
		fmt.Fprintf(stdout, "%s\t%s\t%s\n", page.Citation, page.Anchor, page.Heading)
		for _, b := range page.Blocks {
			fmt.Fprintf(stdout, "%s%s  %s\n", strings.Repeat("  ", b.Depth), b.Path, truncate([]byte(b.Text), 90))
		}
		for _, l := range page.Links {
			fmt.Fprintf(stdout, "%s\t%s\n", l.Path, l.Text)
		}
		return exitOK
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(page); err != nil {
		fmt.Fprintln(stderr, err)
		return exitSetup
	}
	return exitOK
}

// normalize accepts a path or a full LII address and returns the path. Other
// hosts are refused: this tool reads LII only.
func normalize(s string) (string, error) {
	if strings.HasPrefix(s, "/") {
		s = baseURL + s
	}
	u, err := url.Parse(s)
	if err != nil || u.Scheme != "https" || u.Host != "www.law.cornell.edu" || u.Path == "" {
		return "", fmt.Errorf("not an https://www.law.cornell.edu address: %q", s)
	}
	if u.RawQuery != "" {
		return "", fmt.Errorf("query strings are not supported (LII search is disallowed to programs): %q", s)
	}
	return u.EscapedPath(), nil
}

type fetcher struct {
	dir    string
	client *http.Client
}

func (f *fetcher) pageFile(path string) string {
	sum := sha256.Sum256([]byte(path))
	return filepath.Join(f.dir, "pages", hex.EncodeToString(sum[:8])+".html")
}

func (f *fetcher) cached(path string) ([]byte, bool) {
	fi, err := os.Stat(f.pageFile(path))
	if err != nil || time.Since(fi.ModTime()) > pageTTL {
		return nil, false
	}
	b, err := os.ReadFile(f.pageFile(path))
	return b, err == nil
}

func (f *fetcher) get(path string, refresh bool) ([]byte, error) {
	if !refresh {
		if b, ok := f.cached(path); ok {
			return b, nil
		}
	}
	rules, err := f.robots()
	if err != nil {
		return nil, err
	}
	if !rules.allowed(path) {
		return nil, refusedError{fmt.Sprintf("robots.txt disallows %s; not fetched", path)}
	}
	b, err := f.fetch(path, rules.delay)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(f.pageFile(path), b, 0o644); err != nil {
		return nil, err
	}
	return b, nil
}

// fetch waits until delay has passed since the last request to LII (recorded
// in the cache, so separate runs share it), then sends one GET.
func (f *fetcher) fetch(path string, delay time.Duration) ([]byte, error) {
	last := filepath.Join(f.dir, "last-request")
	if fi, err := os.Stat(last); err == nil {
		if wait := delay - time.Since(fi.ModTime()); wait > 0 {
			time.Sleep(wait)
		}
	}
	if err := os.WriteFile(last, nil, 0o644); err != nil {
		return nil, err
	}
	now := time.Now()
	_ = os.Chtimes(last, now, now)
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", path, err)
	}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		// LII sends unknown paths to an index; say so rather than parse the wrong page.
		return nil, fmt.Errorf("GET %s: %s to %s; check the path (walk the TOC from its parent)", path, resp.Status, resp.Header.Get("Location"))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s\n%s", path, resp.Status, truncate(b, 500))
	}
	return b, nil
}

func truncate(b []byte, n int) string {
	r := []rune(string(b))
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return string(r)
}

type robotsRules struct {
	disallow []string
	allow    []string
	delay    time.Duration
}

// allowed applies the longest matching rule, as RFC 9309 does: '*' matches
// any run of characters and a trailing '$' anchors the end; on a tie, allow
// wins.
func (r robotsRules) allowed(path string) bool {
	best, ok := -1, true
	match := func(rules []string, verdict bool) {
		for _, p := range rules {
			if robotsMatch(p, path) && (len(p) > best || (len(p) == best && verdict)) {
				best, ok = len(p), verdict
			}
		}
	}
	match(r.disallow, false)
	match(r.allow, true)
	return ok
}

func robotsMatch(pattern, path string) bool {
	expr := "^" + regexp.QuoteMeta(strings.TrimSuffix(pattern, "$"))
	expr = strings.ReplaceAll(expr, `\*`, ".*")
	if strings.HasSuffix(pattern, "$") {
		expr += "$"
	}
	re, err := regexp.Compile(expr)
	return err == nil && re.MatchString(path)
}

const maxDelay = 60 * time.Second

func (f *fetcher) robots() (robotsRules, error) {
	file := filepath.Join(f.dir, "robots.txt")
	fi, err := os.Stat(file)
	if err != nil || time.Since(fi.ModTime()) > robotsTTL {
		b, err := f.fetch("/robots.txt", 10*time.Second)
		if err != nil {
			return robotsRules{}, fmt.Errorf("robots.txt: %w", err)
		}
		if err := os.WriteFile(file, b, 0o644); err != nil {
			return robotsRules{}, err
		}
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return robotsRules{}, err
	}
	return parseRobots(b)
}

// parseRobots reads the group for User-agent: * only. Consecutive
// User-agent lines form one group. An unreadable file is an error, so the
// rules never fail open.
func parseRobots(b []byte) (robotsRules, error) {
	r := robotsRules{delay: time.Second}
	inStar, prevAgent := false, false
	sc := bufio.NewScanner(bytes.NewReader(bytes.TrimPrefix(b, []byte("\xef\xbb\xbf"))))
	for sc.Scan() {
		line, _, _ := strings.Cut(sc.Text(), "#")
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k, v = strings.ToLower(strings.TrimSpace(k)), strings.TrimSpace(v)
		if k == "user-agent" {
			inStar = (prevAgent && inStar) || v == "*"
			prevAgent = true
			continue
		}
		prevAgent = false
		if !inStar {
			continue
		}
		switch k {
		case "disallow":
			if v != "" {
				r.disallow = append(r.disallow, v)
			}
		case "allow":
			if v != "" {
				r.allow = append(r.allow, v)
			}
		case "crawl-delay":
			if s, err := strconv.ParseFloat(v, 64); err == nil && s > 0 {
				r.delay = min(time.Duration(s*float64(time.Second)), maxDelay)
			}
		}
	}
	if err := sc.Err(); err != nil {
		return robotsRules{}, fmt.Errorf("robots.txt: %w", err)
	}
	return r, nil
}
