// courtlistener is a small client for CourtListener's REST v4 API. It reads the
// token from ~/.netrc (machine www.courtlistener.com), sends it as
// "Authorization: Token", and keeps a local rolling-window budget so a session
// cannot spend the account's 5/min, 50/hour, 125/day allowance blindly.
// Stdlib only.
//
//	courtlistener get <endpoint> [key=value ...]   GET /api/rest/v4/<endpoint>/
//	courtlistener lookup <volume> <reporter> <page> POST citation-lookup/
//	courtlistener lookup -text "<text>"             POST citation-lookup/ with text
//	courtlistener budget                            show local spend; no request
//
// Flags (before the subcommand): -dry-run prints the request without sending
// or spending; -netrc overrides the netrc path (env COURTLISTENER_NETRC).
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	baseURL   = "https://www.courtlistener.com/api/rest/v4/"
	netrcHost = "www.courtlistener.com"
	userAgent = "arca-law-maintainer/1 (courtlistener script)"
)

// limits mirror CourtListener's documented authenticated throttle (read
// 2026-11-08). The account's other users (an admin server sharing the token)
// are invisible here, so this is a ceiling, not a guarantee.
var limits = []struct {
	window time.Duration
	max    int
}{
	{time.Minute, 5},
	{time.Hour, 50},
	{24 * time.Hour, 125},
}

// Exit codes let a caller tell a refused budget from a failed request.
const (
	exitUsage  = 2
	exitBudget = 3
	exitHTTP   = 4
)

func main() {
	dryRun := flag.Bool("dry-run", false, "print the request; send nothing, spend nothing")
	netrc := flag.String("netrc", envOr("COURTLISTENER_NETRC", filepath.Join(home(), ".netrc")), "netrc holding the token")
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
		os.Exit(exitUsage)
	}
	ledger := filepath.Join(cacheDir(), "requests.log")

	var req *http.Request
	var err error
	switch cmd, args := flag.Arg(0), flag.Args()[1:]; cmd {
	case "budget":
		err = printBudget(os.Stdout, ledger, time.Now())
		exitOn(err, 1)
		return
	case "get":
		req, err = buildGet(args)
	case "lookup":
		req, err = buildLookup(args)
	default:
		err = fmt.Errorf("unknown command %q", cmd)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "courtlistener:", err)
		usage()
		os.Exit(exitUsage)
	}

	if *dryRun {
		fmt.Println(req.Method, req.URL.String())
		return
	}
	token, err := readToken(*netrc)
	exitOn(err, 1)
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	exitOn(spend(ledger, time.Now()), exitBudget)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	code := send(req.WithContext(ctx), os.Stdout, os.Stderr)
	cancel()
	os.Exit(code)
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: courtlistener [-dry-run] [-netrc path] <command>
  get <endpoint> [key=value ...]     e.g. get search type=o 'q=cites:(123)' fields=cluster_id
  lookup <volume> <reporter> <page>  e.g. lookup 576 U.S. 644
  lookup -text "<text with citations>"
  budget                             local spend in the last minute/hour/day
`)
}

// buildGet maps "get clusters/123 fields=id,case_name" to a GET on the v4 base.
func buildGet(args []string) (*http.Request, error) {
	if len(args) < 1 {
		return nil, errors.New("get needs an endpoint")
	}
	ep := strings.Trim(args[0], "/")
	if ep == "" || strings.Contains(ep, "..") || strings.Contains(ep, "://") {
		return nil, fmt.Errorf("bad endpoint %q", args[0])
	}
	q := url.Values{}
	for _, kv := range args[1:] {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("query argument %q is not key=value", kv)
		}
		q.Add(k, v)
	}
	u := baseURL + ep + "/"
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	return http.NewRequest(http.MethodGet, u, nil)
}

func buildLookup(args []string) (*http.Request, error) {
	form := url.Values{}
	switch {
	case len(args) == 2 && args[0] == "-text":
		form.Set("text", args[1])
	case len(args) == 3:
		form.Set("volume", args[0])
		form.Set("reporter", args[1])
		form.Set("page", args[2])
	default:
		return nil, errors.New("lookup needs <volume> <reporter> <page> or -text <text>")
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+"citation-lookup/", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

// send performs one request, never retrying: a 429 ends the call so the
// caller decides, rather than a loop burning the day's budget.
func send(req *http.Request, out, errOut io.Writer) int {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(errOut, "courtlistener:", req.Method, req.URL.Path+":", err)
		return exitHTTP
	}
	defer resp.Body.Close()
	const maxBody = 32 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		fmt.Fprintln(errOut, "courtlistener: read response:", err)
		return exitHTTP
	}
	if len(body) > maxBody {
		fmt.Fprintln(errOut, "courtlistener: response over 32 MiB; narrow it with fields=")
		return exitHTTP
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, body, "", "  ") == nil {
		body = append(pretty.Bytes(), '\n')
	}
	if resp.StatusCode >= 300 {
		fmt.Fprintf(errOut, "courtlistener: %s %s: %s\n", req.Method, req.URL.Path, resp.Status)
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			fmt.Fprintln(errOut, "courtlistener: Retry-After:", ra)
		}
		errOut.Write(body)
		return exitHTTP
	}
	out.Write(body)
	return 0
}

// readToken returns the password for netrcHost, refusing a netrc that group
// or others can read: a leaked token spends the shared daily budget.
func readToken(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("netrc: %w", err)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("netrc %s is mode %o; run: chmod 600 %s", path, fi.Mode().Perm(), path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("netrc: %w", err)
	}
	token, ok := netrcPassword(string(data), netrcHost)
	if !ok {
		return "", fmt.Errorf("netrc %s has no 'machine %s ... password <token>' entry", path, netrcHost)
	}
	return token, nil
}

// netrcPassword scans netrc tokens; entries may span lines. Macros are not
// supported, which is fine for a token file.
func netrcPassword(data, host string) (string, bool) {
	f := strings.Fields(data)
	inHost := false
	for i := 0; i < len(f); i++ {
		switch f[i] {
		case "machine":
			if i+1 < len(f) {
				i++
				inHost = f[i] == host
			}
		case "default":
			inHost = false
		case "password":
			if inHost && i+1 < len(f) {
				return f[i+1], true
			}
			i++
		case "login", "account":
			i++
		}
	}
	return "", false
}

// spend records one request in the ledger, or refuses if any window is full.
// The read, check and write happen under one exclusive lock so concurrent
// callers cannot both pass the check.
func spend(ledger string, now time.Time) error {
	if err := os.MkdirAll(filepath.Dir(ledger), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(ledger, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock %s: %w", ledger, err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	times, err := readLedger(f, now)
	if err != nil {
		return err
	}
	for _, l := range limits {
		if n, oldest := countSince(times, now.Add(-l.window)); n >= l.max {
			return fmt.Errorf("budget: %d of %d requests used in the last %s; next allowed at %s",
				n, l.max, l.window, oldest.Add(l.window).Local().Format("2006-01-02 15:04:05"))
		}
	}
	return writeLedger(f, append(times, now))
}

// readLedger returns the request times within the longest window.
func readLedger(f *os.File, now time.Time) ([]time.Time, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	cutoff := now.Add(-limits[len(limits)-1].window)
	var times []time.Time
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(sc.Text()))
		if err == nil && t.After(cutoff) {
			times = append(times, t)
		}
	}
	return times, sc.Err()
}

func writeLedger(f *os.File, times []time.Time) error {
	var b strings.Builder
	for _, t := range times {
		b.WriteString(t.UTC().Format(time.RFC3339) + "\n")
	}
	// write before truncating: a crash in between leaves extra old lines,
	// which over-count, never an empty ledger that resets the budget
	data := []byte(b.String())
	if _, err := f.WriteAt(data, 0); err != nil {
		return err
	}
	return f.Truncate(int64(len(data)))
}

// countSince counts times after since, returning the oldest of them.
func countSince(times []time.Time, since time.Time) (int, time.Time) {
	n, oldest := 0, time.Time{}
	for _, t := range times {
		if t.After(since) {
			if n == 0 || t.Before(oldest) {
				oldest = t
			}
			n++
		}
	}
	return n, oldest
}

func printBudget(w io.Writer, ledger string, now time.Time) error {
	f, err := os.Open(ledger)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(w, "no requests recorded")
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	times, err := readLedger(f, now)
	if err != nil {
		return err
	}
	for _, l := range limits {
		n, _ := countSince(times, now.Add(-l.window))
		fmt.Fprintf(w, "last %-8s %3d / %d\n", l.window, n, l.max)
	}
	fmt.Fprintln(w, "local spend only; other users of the same token are not counted")
	return nil
}

func exitOn(err error, code int) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "courtlistener:", err)
		os.Exit(code)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}

func cacheDir() string {
	if d, err := os.UserCacheDir(); err == nil {
		return filepath.Join(d, "courtlistener")
	}
	return filepath.Join(home(), ".cache", "courtlistener")
}
