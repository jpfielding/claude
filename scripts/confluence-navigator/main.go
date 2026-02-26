package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

// ── Netrc parsing ───────────────────────────────────────────

type netrcEntry struct {
	Machine  string
	Login    string
	Password string
}

func parseNetrc() []netrcEntry {
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".netrc"))
	if err != nil {
		return nil
	}
	tokens := tokenize(string(data))
	var entries []netrcEntry
	var cur *netrcEntry
	for i := 0; i < len(tokens); i++ {
		switch strings.ToLower(tokens[i]) {
		case "machine":
			if cur != nil {
				entries = append(entries, *cur)
			}
			cur = &netrcEntry{}
			if i+1 < len(tokens) {
				i++
				cur.Machine = tokens[i]
			}
		case "login":
			if cur != nil && i+1 < len(tokens) {
				i++
				cur.Login = tokens[i]
			}
		case "password":
			if cur != nil && i+1 < len(tokens) {
				i++
				cur.Password = tokens[i]
			}
		}
	}
	if cur != nil {
		entries = append(entries, *cur)
	}
	return entries
}

func tokenize(s string) []string {
	var tokens []string
	i := 0
	for i < len(s) {
		// skip whitespace
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' {
			i++
			continue
		}
		// skip comments
		if s[i] == '#' {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		// quoted string
		if s[i] == '"' {
			i++
			start := i
			for i < len(s) && s[i] != '"' {
				i++
			}
			tokens = append(tokens, s[start:i])
			if i < len(s) {
				i++
			}
			continue
		}
		// unquoted token
		start := i
		for i < len(s) && s[i] != ' ' && s[i] != '\t' && s[i] != '\n' && s[i] != '\r' {
			i++
		}
		tokens = append(tokens, s[start:i])
	}
	return tokens
}

func lookupNetrc(machine string) (netrcEntry, error) {
	for _, e := range parseNetrc() {
		if e.Machine == machine {
			return e, nil
		}
	}
	return netrcEntry{}, fmt.Errorf("no entry for machine '%s' in ~/.netrc", machine)
}

func resolveHost(input, filter string) (string, netrcEntry, error) {
	if strings.Contains(input, ".") {
		entry, err := lookupNetrc(input)
		if err != nil {
			return "", netrcEntry{}, err
		}
		return input, entry, nil
	}
	entries := parseNetrc()
	var matches []netrcEntry
	for _, e := range entries {
		if strings.Contains(e.Machine, input) && strings.Contains(e.Machine, filter) {
			matches = append(matches, e)
		}
	}
	if len(matches) == 0 {
		return "", netrcEntry{}, fmt.Errorf("no ~/.netrc machine entries matching '%s'", input)
	}
	if len(matches) > 1 {
		msg := fmt.Sprintf("Multiple matches for '%s':\n", input)
		for _, m := range matches {
			msg += fmt.Sprintf("  %s\n", m.Machine)
		}
		return "", netrcEntry{}, fmt.Errorf("%sambiguous host substring '%s' — be more specific or use the full hostname", msg, input)
	}
	return matches[0].Machine, matches[0], nil
}

func discoverHosts(filter string) []string {
	var hosts []string
	for _, e := range parseNetrc() {
		if strings.Contains(e.Machine, filter) {
			hosts = append(hosts, e.Machine)
		}
	}
	return hosts
}

// ── HTTP client ─────────────────────────────────────────────

type apiClient struct {
	baseURL  string
	password string
}

func newClient(host string, entry netrcEntry) *apiClient {
	return &apiClient{
		baseURL:  "https://" + host,
		password: entry.Password,
	}
}

func (c *apiClient) get(endpoint string, params url.Values) (json.RawMessage, error) {
	u := c.baseURL + "/rest/api" + endpoint
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.RawMessage(body), nil
}

func (c *apiClient) post(endpoint string, payload any) (json.RawMessage, error) {
	u := c.baseURL + "/rest/api" + endpoint
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	req, err := http.NewRequest("POST", u, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.RawMessage(body), nil
}

func (c *apiClient) put(endpoint string, payload any) (json.RawMessage, error) {
	u := c.baseURL + "/rest/api" + endpoint
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	req, err := http.NewRequest("PUT", u, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.RawMessage(body), nil
}

func (c *apiClient) delete(endpoint string) error {
	u := c.baseURL + "/rest/api" + endpoint
	req, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ── Calendar API client methods ─────────────────────────────
// The Team Calendars plugin uses a different base path: /rest/calendar-services/1.0/calendar

func (c *apiClient) getCalendar(endpoint string, params url.Values) (json.RawMessage, error) {
	u := c.baseURL + "/rest/calendar-services/1.0/calendar" + endpoint
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("calendar plugin may not be installed on this Confluence instance (HTTP 404)")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.RawMessage(body), nil
}

func (c *apiClient) postCalendar(endpoint string, payload any) (json.RawMessage, error) {
	u := c.baseURL + "/rest/calendar-services/1.0/calendar" + endpoint
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	req, err := http.NewRequest("POST", u, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("calendar plugin may not be installed on this Confluence instance (HTTP 404)")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.RawMessage(body), nil
}

func (c *apiClient) putCalendar(endpoint string, payload any) (json.RawMessage, error) {
	u := c.baseURL + "/rest/calendar-services/1.0/calendar" + endpoint
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	req, err := http.NewRequest("PUT", u, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("calendar plugin may not be installed on this Confluence instance (HTTP 404)")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.RawMessage(body), nil
}

func (c *apiClient) deleteCalendar(endpoint string) error {
	u := c.baseURL + "/rest/calendar-services/1.0/calendar" + endpoint
	req, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 404 {
		return fmt.Errorf("calendar plugin may not be installed on this Confluence instance (HTTP 404)")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ── Output helpers ──────────────────────────────────────────

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func jsonStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok && v != nil {
		switch t := v.(type) {
		case string:
			return t
		case float64:
			return strconv.FormatFloat(t, 'f', -1, 64)
		case bool:
			return strconv.FormatBool(t)
		}
	}
	return ""
}

func jsonFloat(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok && v != nil {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

func jsonMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok && v != nil {
		if mm, ok := v.(map[string]any); ok {
			return mm
		}
	}
	return nil
}

func jsonArr(m map[string]any, key string) []any {
	if v, ok := m[key]; ok && v != nil {
		if arr, ok := v.([]any); ok {
			return arr
		}
	}
	return nil
}

func strOr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

// ── Commands ────────────────────────────────────────────────

func cmdDiscover(args []string) {
	filter := "confluence"
	if len(args) > 0 {
		filter = args[0]
	}
	hosts := discoverHosts(filter)
	if len(hosts) == 0 {
		fmt.Printf("No ~/.netrc entries matching '%s' found.\n", filter)
		fmt.Println()
		fmt.Println("To search with a different substring: confluence-navigator discover <substring>")
		return
	}
	fmt.Printf("Found ~/.netrc entries matching '%s':\n\n", filter)
	for i, h := range hosts {
		fmt.Printf("  [%d] %s\n", i+1, h)
	}
	fmt.Println()
	fmt.Println("Usage: confluence-navigator <hostname-or-substring> <command>")
	fmt.Println()
	fmt.Printf("Example using the first match:\n  confluence-navigator %s test\n", hosts[0])
}

func cmdWhoami(c *apiClient) {
	data, err := c.get("/user/current", nil)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	out := map[string]any{
		"username":    jsonStr(m, "username"),
		"displayName": jsonStr(m, "displayName"),
		"email":       jsonStr(m, "email"),
		"userKey":     strOr(jsonStr(m, "userKey"), jsonStr(m, "key")),
	}
	printJSON(out)
}

func cmdTest(c *apiClient) {
	fmt.Println("Testing connection...")
	data, err := c.get("/user/current", nil)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	name := strOr(jsonStr(m, "displayName"), strOr(jsonStr(m, "username"), "unknown"))
	fmt.Printf("Connected as: %s\n\n", name)

	fmt.Println("Testing space listing...")
	params := url.Values{"limit": {"3"}}
	data, err = c.get("/space", params)
	if err != nil {
		die("%s", err)
	}
	var sm map[string]any
	json.Unmarshal(data, &sm)
	for _, r := range jsonArr(sm, "results") {
		rm := asMap(r)
		if rm != nil {
			fmt.Printf("  %s - %s\n", jsonStr(rm, "key"), jsonStr(rm, "name"))
		}
	}
	fmt.Println()
	fmt.Println("Connection OK.")
}

func cmdRecent(c *apiClient, args []string) {
	limit := "20"
	if len(args) > 0 {
		limit = args[0]
	}
	params := url.Values{
		"orderby": {"history.lastUpdated desc"},
		"limit":   {limit},
		"expand":  {"history.lastUpdated,space,version"},
	}
	data, err := c.get("/content", params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	for _, r := range jsonArr(m, "results") {
		rm := asMap(r)
		if rm == nil {
			continue
		}
		space := jsonMap(rm, "space")
		version := jsonMap(rm, "version")
		by := jsonMap(version, "by")
		links := jsonMap(rm, "_links")
		updated := strOr(jsonStr(version, "when"), jsonStr(jsonMap(jsonMap(rm, "history"), "lastUpdated"), "when"))
		byName := "unknown"
		if by != nil {
			byName = strOr(jsonStr(by, "displayName"), "unknown")
		}
		fmt.Printf("[%s] %s\n", jsonStr(space, "key"), jsonStr(rm, "title"))
		fmt.Printf("  Updated: %s by %s\n", updated, byName)
		fmt.Printf("  URL: %s\n\n", jsonStr(links, "self"))
	}
}

func cmdWatched(c *apiClient) {
	params := url.Values{"limit": {"50"}}
	data, err := c.get("/user/watch", params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	if jsonArr(m, "results") != nil {
		for _, r := range jsonArr(m, "results") {
			rm := asMap(r)
			if rm == nil {
				continue
			}
			content := jsonMap(rm, "content")
			space := jsonMap(content, "space")
			spaceKey := "?"
			if space != nil {
				spaceKey = strOr(jsonStr(space, "key"), "?")
			}
			title := strOr(jsonStr(content, "title"), strOr(jsonStr(jsonMap(rm, "space"), "name"), "unknown"))
			fmt.Printf("[%s] %s\n", spaceKey, title)
			fmt.Printf("  Type: %s\n\n", strOr(jsonStr(rm, "type"), "unknown"))
		}
	} else {
		fmt.Println("Direct watch API not available. Trying alternative approach...")
		userData, err := c.get("/user/current", nil)
		if err != nil {
			die("%s", err)
		}
		var um map[string]any
		json.Unmarshal(userData, &um)
		userKey := strOr(jsonStr(um, "userKey"), jsonStr(um, "key"))
		if userKey != "" {
			fmt.Printf("Current user: %s\n\n", strOr(jsonStr(um, "displayName"), jsonStr(um, "username")))
			fmt.Println("Use 'search' command with CQL to find recently modified content in your watched spaces.")
			fmt.Println("Example: confluence-navigator <instance> search 'watcher = currentUser() order by lastModified desc'")
		} else {
			fmt.Println("Could not determine current user. Check authentication.")
		}
	}
}

func cmdWatchChanges(c *apiClient, args []string) {
	days := "7"
	if len(args) > 0 {
		days = args[0]
	}
	cql := fmt.Sprintf("watcher = currentUser() AND lastModified >= now(\"-%sd\") ORDER BY lastModified DESC", days)
	params := url.Values{
		"cql":    {cql},
		"limit":  {"25"},
		"expand": {"history.lastUpdated,space,version"},
	}
	data, err := c.get("/content/search", params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	results := jsonArr(m, "results")
	if results != nil {
		fmt.Printf("Changes to watched content in last %s days (%d results):\n\n", days, len(results))
		for _, r := range results {
			rm := asMap(r)
			if rm == nil {
				continue
			}
			space := jsonMap(rm, "space")
			version := jsonMap(rm, "version")
			by := jsonMap(version, "by")
			spaceKey := "?"
			if space != nil {
				spaceKey = strOr(jsonStr(space, "key"), "?")
			}
			updated := strOr(jsonStr(version, "when"), "unknown")
			byName := "unknown"
			if by != nil {
				byName = strOr(jsonStr(by, "displayName"), "unknown")
			}
			fmt.Printf("[%s] %s\n", spaceKey, jsonStr(rm, "title"))
			fmt.Printf("  Updated: %s by %s\n", updated, byName)
			fmt.Printf("  ID: %s\n\n", jsonStr(rm, "id"))
		}
	} else {
		fmt.Println("No results or CQL not supported. Raw response:")
		printJSON(m)
	}
}

func cmdSearch(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: search <CQL query>")
	}
	cql := args[0]
	limit := "20"
	if len(args) > 1 {
		limit = args[1]
	}
	params := url.Values{
		"cql":    {cql},
		"limit":  {limit},
		"expand": {"space,version"},
	}
	data, err := c.get("/content/search", params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	for _, r := range jsonArr(m, "results") {
		rm := asMap(r)
		if rm == nil {
			continue
		}
		space := jsonMap(rm, "space")
		version := jsonMap(rm, "version")
		by := jsonMap(version, "by")
		spaceKey := "?"
		if space != nil {
			spaceKey = strOr(jsonStr(space, "key"), "?")
		}
		updated := strOr(jsonStr(version, "when"), "unknown")
		byName := "unknown"
		if by != nil {
			byName = strOr(jsonStr(by, "displayName"), "unknown")
		}
		fmt.Printf("[%s] %s (id: %s)\n", spaceKey, jsonStr(rm, "title"), jsonStr(rm, "id"))
		fmt.Printf("  Updated: %s by %s\n\n", updated, byName)
	}
}

func cmdSpaces(c *apiClient, args []string) {
	limit := "50"
	if len(args) > 0 {
		limit = args[0]
	}
	params := url.Values{
		"limit":  {limit},
		"expand": {"description.plain"},
	}
	data, err := c.get("/space", params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	for _, r := range jsonArr(m, "results") {
		rm := asMap(r)
		if rm == nil {
			continue
		}
		desc := ""
		if d := jsonMap(rm, "description"); d != nil {
			if p := jsonMap(d, "plain"); p != nil {
				desc = jsonStr(p, "value")
			}
		}
		if desc == "" {
			desc = "none"
		}
		desc = strings.ReplaceAll(desc, "\n", " ")
		if len(desc) > 120 {
			desc = desc[:120]
		}
		fmt.Printf("%s - %s\n", jsonStr(rm, "key"), jsonStr(rm, "name"))
		fmt.Printf("  Type: %s\n", jsonStr(rm, "type"))
		fmt.Printf("  Description: %s\n\n", desc)
	}
}

func cmdSpacePages(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: space-pages <space-key> [limit]")
	}
	spaceKey := args[0]
	limit := "25"
	if len(args) > 1 {
		limit = args[1]
	}
	params := url.Values{
		"limit":   {limit},
		"expand":  {"version"},
		"orderby": {"history.lastUpdated desc"},
	}
	data, err := c.get("/space/"+spaceKey+"/content/page", params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	for _, r := range jsonArr(m, "results") {
		rm := asMap(r)
		if rm == nil {
			continue
		}
		version := jsonMap(rm, "version")
		by := jsonMap(version, "by")
		byName := "unknown"
		if by != nil {
			byName = strOr(jsonStr(by, "displayName"), "unknown")
		}
		fmt.Printf("%s (id: %s)\n", jsonStr(rm, "title"), jsonStr(rm, "id"))
		fmt.Printf("  Version: %s by %s at %s\n\n",
			jsonStr(version, "number"), byName, strOr(jsonStr(version, "when"), "unknown"))
	}
}

func cmdPage(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: page <page-id> [format: storage|view]")
	}
	pageID := args[0]
	format := "view"
	if len(args) > 1 {
		format = args[1]
	}
	params := url.Values{
		"expand": {"body." + format + ",space,version,ancestors"},
	}
	data, err := c.get("/content/"+pageID, params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)

	space := jsonMap(m, "space")
	version := jsonMap(m, "version")
	by := jsonMap(version, "by")

	// ancestors chain
	var ancestorTitles []string
	for _, a := range jsonArr(m, "ancestors") {
		am := asMap(a)
		if am != nil {
			ancestorTitles = append(ancestorTitles, jsonStr(am, "title"))
		}
	}

	content := ""
	if body := jsonMap(m, "body"); body != nil {
		if fmtBody := jsonMap(body, format); fmtBody != nil {
			content = strOr(jsonStr(fmtBody, "value"), "No content")
		}
	}
	if content == "" {
		content = "No content"
	}

	fmt.Printf("Title: %s\n", jsonStr(m, "title"))
	fmt.Printf("Space: %s - %s\n", jsonStr(space, "key"), jsonStr(space, "name"))
	fmt.Printf("Version: %s by %s at %s\n",
		jsonStr(version, "number"), strOr(jsonStr(by, "displayName"), "unknown"), jsonStr(version, "when"))
	fmt.Printf("Ancestors: %s\n", strings.Join(ancestorTitles, " > "))
	fmt.Printf("\n--- Content ---\n%s\n", content)
}

func cmdPageInfo(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: page-info <page-id>")
	}
	pageID := args[0]
	params := url.Values{
		"expand": {"space,version,ancestors,children.page,history,metadata.labels"},
	}
	data, err := c.get("/content/"+pageID, params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)

	space := jsonMap(m, "space")
	version := jsonMap(m, "version")
	by := jsonMap(version, "by")
	history := jsonMap(m, "history")
	createdBy := jsonMap(history, "createdBy")

	var ancestors []map[string]any
	for _, a := range jsonArr(m, "ancestors") {
		am := asMap(a)
		if am != nil {
			ancestors = append(ancestors, map[string]any{"id": jsonStr(am, "id"), "title": jsonStr(am, "title")})
		}
	}

	var children []map[string]any
	if ch := jsonMap(m, "children"); ch != nil {
		if pg := jsonMap(ch, "page"); pg != nil {
			for _, c := range jsonArr(pg, "results") {
				cm := asMap(c)
				if cm != nil {
					children = append(children, map[string]any{"id": jsonStr(cm, "id"), "title": jsonStr(cm, "title")})
				}
			}
		}
	}

	var labels []string
	if meta := jsonMap(m, "metadata"); meta != nil {
		if lbls := jsonMap(meta, "labels"); lbls != nil {
			for _, l := range jsonArr(lbls, "results") {
				lm := asMap(l)
				if lm != nil {
					labels = append(labels, jsonStr(lm, "name"))
				}
			}
		}
	}

	out := map[string]any{
		"id":    jsonStr(m, "id"),
		"title": jsonStr(m, "title"),
		"space": map[string]any{
			"key":  jsonStr(space, "key"),
			"name": jsonStr(space, "name"),
		},
		"version": map[string]any{
			"number": jsonStr(version, "number"),
			"by":     strOr(jsonStr(by, "displayName"), "unknown"),
			"when":   jsonStr(version, "when"),
		},
		"ancestors": ancestors,
		"children":  children,
		"labels":    labels,
		"created":   jsonStr(history, "createdDate"),
		"creator":   strOr(jsonStr(createdBy, "displayName"), "unknown"),
	}
	printJSON(out)
}

func cmdChildren(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: children <page-id>")
	}
	pageID := args[0]
	params := url.Values{
		"limit":  {"50"},
		"expand": {"version"},
	}
	data, err := c.get("/content/"+pageID+"/child/page", params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	for _, r := range jsonArr(m, "results") {
		rm := asMap(r)
		if rm == nil {
			continue
		}
		version := jsonMap(rm, "version")
		by := jsonMap(version, "by")
		byName := "unknown"
		if by != nil {
			byName = strOr(jsonStr(by, "displayName"), "unknown")
		}
		fmt.Printf("%s (id: %s)\n", jsonStr(rm, "title"), jsonStr(rm, "id"))
		fmt.Printf("  Version: %s by %s\n\n", jsonStr(version, "number"), byName)
	}
}

func cmdLabels(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: labels <page-id>")
	}
	pageID := args[0]
	data, err := c.get("/content/"+pageID+"/label", nil)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	for _, r := range jsonArr(m, "results") {
		rm := asMap(r)
		if rm != nil {
			fmt.Printf("%s:%s\n", jsonStr(rm, "prefix"), jsonStr(rm, "name"))
		}
	}
}

func cmdHistory(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: history <page-id> [limit]")
	}
	pageID := args[0]
	limit := "10"
	if len(args) > 1 {
		limit = args[1]
	}
	params := url.Values{"limit": {limit}}
	data, err := c.get("/content/"+pageID+"/version", params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	for _, r := range jsonArr(m, "results") {
		rm := asMap(r)
		if rm == nil {
			continue
		}
		by := jsonMap(rm, "by")
		byName := "unknown"
		if by != nil {
			byName = strOr(jsonStr(by, "displayName"), "unknown")
		}
		msg := strOr(jsonStr(rm, "message"), "<no message>")
		fmt.Printf("v%s - %s by %s\n", jsonStr(rm, "number"), jsonStr(rm, "when"), byName)
		fmt.Printf("  Message: %s\n\n", msg)
	}
}

func cmdCreatePage(c *apiClient, args []string) {
	if len(args) < 2 {
		die("Usage: create-page <space-key> <page-title> [parent-page-id] [content]")
	}
	spaceKey := args[0]
	pageTitle := args[1]
	parentID := ""
	content := ""
	if len(args) > 2 {
		parentID = args[2]
	}
	if len(args) > 3 {
		content = args[3]
	}

	payload := map[string]any{
		"type":  "page",
		"title": pageTitle,
		"space": map[string]any{
			"key": spaceKey,
		},
		"body": map[string]any{
			"storage": map[string]any{
				"value":          content,
				"representation": "storage",
			},
		},
	}
	if parentID != "" {
		payload["ancestors"] = []map[string]any{
			{"id": parentID},
		}
	}

	data, err := c.post("/content", payload)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)

	space := jsonMap(m, "space")
	links := jsonMap(m, "_links")
	webUI := ""
	if links != nil {
		webUI = jsonStr(links, "webui")
		if webUI != "" && !strings.HasPrefix(webUI, "http") {
			webUI = c.baseURL + webUI
		}
	}

	out := map[string]any{
		"id":    jsonStr(m, "id"),
		"title": jsonStr(m, "title"),
		"space": map[string]any{
			"key":  jsonStr(space, "key"),
			"name": jsonStr(space, "name"),
		},
		"url": webUI,
	}
	printJSON(out)
}

func cmdUpdatePage(c *apiClient, args []string) {
	if len(args) < 2 {
		die("Usage: update-page <page-id> <content> [mode: replace|append]")
	}
	pageID := args[0]
	content := args[1]
	mode := "replace"
	if len(args) > 2 {
		mode = args[2]
	}
	if mode != "replace" && mode != "append" {
		die("Usage: update-page <page-id> <content> [mode: replace|append]")
	}

	params := url.Values{
		"expand": {"title,type,space,version,body.storage"},
	}
	data, err := c.get("/content/"+pageID, params)
	if err != nil {
		die("%s", err)
	}
	var page map[string]any
	json.Unmarshal(data, &page)

	currentVersion, err := strconv.Atoi(jsonStr(jsonMap(page, "version"), "number"))
	if err != nil {
		die("failed to parse current page version: %v", err)
	}

	finalContent := content
	if mode == "append" {
		existing := ""
		if body := jsonMap(page, "body"); body != nil {
			if storage := jsonMap(body, "storage"); storage != nil {
				existing = jsonStr(storage, "value")
			}
		}
		finalContent = existing + content
	}

	payload := map[string]any{
		"id":    pageID,
		"type":  strOr(jsonStr(page, "type"), "page"),
		"title": jsonStr(page, "title"),
		"space": map[string]any{
			"key": jsonStr(jsonMap(page, "space"), "key"),
		},
		"body": map[string]any{
			"storage": map[string]any{
				"value":          finalContent,
				"representation": "storage",
			},
		},
		"version": map[string]any{
			"number": currentVersion + 1,
		},
	}

	updatedData, err := c.put("/content/"+pageID, payload)
	if err != nil {
		die("%s", err)
	}
	var updated map[string]any
	json.Unmarshal(updatedData, &updated)

	links := jsonMap(updated, "_links")
	webUI := ""
	if links != nil {
		webUI = jsonStr(links, "webui")
		if webUI != "" && !strings.HasPrefix(webUI, "http") {
			webUI = c.baseURL + webUI
		}
	}

	out := map[string]any{
		"id":      jsonStr(updated, "id"),
		"title":   jsonStr(updated, "title"),
		"version": jsonStr(jsonMap(updated, "version"), "number"),
		"url":     webUI,
	}
	printJSON(out)
}

func cmdCreateSpace(c *apiClient, args []string) {
	if len(args) < 2 {
		die("Usage: create-space <space-key> <space-name> [description]")
	}
	spaceKey := args[0]
	spaceName := args[1]
	description := ""
	if len(args) > 2 {
		description = args[2]
	}

	payload := map[string]any{
		"key":  spaceKey,
		"name": spaceName,
		"type": "personal",
	}
	if description != "" {
		payload["description"] = map[string]any{
			"plain": map[string]any{
				"value":          description,
				"representation": "plain",
			},
		}
	}

	data, err := c.post("/space", payload)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)

	links := jsonMap(m, "_links")
	webUI := ""
	if links != nil {
		webUI = jsonStr(links, "webui")
		if webUI != "" && !strings.HasPrefix(webUI, "http") {
			webUI = c.baseURL + webUI
		}
	}

	out := map[string]any{
		"key":  jsonStr(m, "key"),
		"name": jsonStr(m, "name"),
		"id":   jsonStr(m, "id"),
		"type": jsonStr(m, "type"),
		"url":  webUI,
	}
	printJSON(out)
}

// ── Comments Management ─────────────────────────────────────

func cmdComments(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: comments <page-id>")
	}
	pageID := args[0]
	params := url.Values{
		"limit":  {"50"},
		"expand": {"body.view,version"},
	}
	data, err := c.get("/content/"+pageID+"/child/comment", params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)

	results := jsonArr(m, "results")
	if len(results) == 0 {
		fmt.Println("No comments on this page.")
		return
	}

	for _, r := range results {
		rm := asMap(r)
		if rm == nil {
			continue
		}
		version := jsonMap(rm, "version")
		by := jsonMap(version, "by")
		byName := "unknown"
		if by != nil {
			byName = strOr(jsonStr(by, "displayName"), "unknown")
		}

		content := ""
		if body := jsonMap(rm, "body"); body != nil {
			if view := jsonMap(body, "view"); view != nil {
				content = jsonStr(view, "value")
			}
		}
		// Strip HTML tags for display
		content = stripHTML(content)
		if len(content) > 200 {
			content = content[:200] + "..."
		}

		fmt.Printf("Comment ID: %s\n", jsonStr(rm, "id"))
		fmt.Printf("  By: %s at %s\n", byName, strOr(jsonStr(version, "when"), "unknown"))
		fmt.Printf("  Content: %s\n\n", content)
	}
}

func cmdCommentAdd(c *apiClient, args []string) {
	if len(args) < 2 {
		die("Usage: comment-add <page-id> <comment-text>")
	}
	pageID := args[0]
	commentText := args[1]

	payload := map[string]any{
		"type": "comment",
		"container": map[string]any{
			"id":   pageID,
			"type": "page",
		},
		"body": map[string]any{
			"storage": map[string]any{
				"value":          commentText,
				"representation": "storage",
			},
		},
	}

	data, err := c.post("/content", payload)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)

	out := map[string]any{
		"id":      jsonStr(m, "id"),
		"type":    jsonStr(m, "type"),
		"pageId":  pageID,
		"message": "Comment added successfully",
	}
	printJSON(out)
}

func cmdCommentUpdate(c *apiClient, args []string) {
	if len(args) < 2 {
		die("Usage: comment-update <comment-id> <new-text>")
	}
	commentID := args[0]
	newText := args[1]

	// Get current comment to find version number and container
	params := url.Values{
		"expand": {"version,container"},
	}
	data, err := c.get("/content/"+commentID, params)
	if err != nil {
		die("%s", err)
	}
	var comment map[string]any
	json.Unmarshal(data, &comment)

	currentVersion, err := strconv.Atoi(jsonStr(jsonMap(comment, "version"), "number"))
	if err != nil {
		die("failed to parse current comment version: %v", err)
	}

	container := jsonMap(comment, "container")

	payload := map[string]any{
		"id":   commentID,
		"type": "comment",
		"container": map[string]any{
			"id":   jsonStr(container, "id"),
			"type": jsonStr(container, "type"),
		},
		"body": map[string]any{
			"storage": map[string]any{
				"value":          newText,
				"representation": "storage",
			},
		},
		"version": map[string]any{
			"number": currentVersion + 1,
		},
	}

	updatedData, err := c.put("/content/"+commentID, payload)
	if err != nil {
		die("%s", err)
	}
	var updated map[string]any
	json.Unmarshal(updatedData, &updated)

	out := map[string]any{
		"id":      jsonStr(updated, "id"),
		"version": jsonStr(jsonMap(updated, "version"), "number"),
		"message": "Comment updated successfully",
	}
	printJSON(out)
}

// ── Watch/Unwatch ───────────────────────────────────────────

func cmdWatch(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: watch <page-id>")
	}
	pageID := args[0]

	_, err := c.post("/user/watch/content/"+pageID, nil)
	if err != nil {
		die("%s", err)
	}
	fmt.Printf("Now watching page %s\n", pageID)
}

func cmdUnwatch(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: unwatch <page-id>")
	}
	pageID := args[0]

	err := c.delete("/user/watch/content/" + pageID)
	if err != nil {
		die("%s", err)
	}
	fmt.Printf("Stopped watching page %s\n", pageID)
}

// ── Reading List (Save for Later) ───────────────────────────

func cmdReadLaterAdd(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: read-later-add <page-id>")
	}
	pageID := args[0]

	// Try the standard reading list endpoint
	_, err := c.post("/user/current/content/reading-list/"+pageID, nil)
	if err != nil {
		// Try alternative endpoint format
		payload := map[string]any{
			"contentId": pageID,
		}
		_, err = c.post("/user/current/content/reading-list", payload)
		if err != nil {
			die("Failed to add to reading list (endpoint may not be available in this Confluence version): %s", err)
		}
	}
	fmt.Printf("Added page %s to reading list\n", pageID)
}

func cmdReadLaterRemove(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: read-later-remove <page-id>")
	}
	pageID := args[0]

	err := c.delete("/user/current/content/reading-list/" + pageID)
	if err != nil {
		die("Failed to remove from reading list (endpoint may not be available in this Confluence version): %s", err)
	}
	fmt.Printf("Removed page %s from reading list\n", pageID)
}

func cmdReadLaterList(c *apiClient) {
	params := url.Values{
		"limit":  {"50"},
		"expand": {"space,version"},
	}
	data, err := c.get("/user/current/content/reading-list", params)
	if err != nil {
		die("Failed to get reading list (endpoint may not be available in this Confluence version): %s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)

	results := jsonArr(m, "results")
	if len(results) == 0 {
		fmt.Println("Reading list is empty.")
		return
	}

	fmt.Printf("Reading list (%d items):\n\n", len(results))
	for _, r := range results {
		rm := asMap(r)
		if rm == nil {
			continue
		}
		space := jsonMap(rm, "space")
		spaceKey := "?"
		if space != nil {
			spaceKey = strOr(jsonStr(space, "key"), "?")
		}
		fmt.Printf("[%s] %s (id: %s)\n", spaceKey, jsonStr(rm, "title"), jsonStr(rm, "id"))
	}
}

// ── Page Analytics ──────────────────────────────────────────

func cmdAnalytics(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: analytics <page-id>")
	}
	pageID := args[0]

	// Try the analytics endpoint - this varies by Confluence version
	// Cloud uses /wiki/rest/api/analytics/content/{id}/views
	// Server/Data Center may use different endpoints
	data, err := c.get("/analytics/content/"+pageID+"/views", nil)
	if err != nil {
		// Try alternative endpoint for Server/DC
		data, err = c.get("/content/"+pageID+"/views", nil)
		if err != nil {
			// Fall back to getting basic page info with history
			fmt.Println("Analytics endpoint not available. Showing basic page statistics:")
			fmt.Println()
			showBasicPageStats(c, pageID)
			return
		}
	}

	var m map[string]any
	json.Unmarshal(data, &m)

	// Render analytics as a table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "METRIC\tVALUE")
	fmt.Fprintln(w, "------\t-----")

	if count := jsonFloat(m, "count"); count > 0 {
		fmt.Fprintf(w, "Total Views\t%.0f\n", count)
	} else if count := jsonStr(m, "count"); count != "" {
		fmt.Fprintf(w, "Total Views\t%s\n", count)
	}

	if uniqueCount := jsonFloat(m, "uniqueCount"); uniqueCount > 0 {
		fmt.Fprintf(w, "Unique Viewers\t%.0f\n", uniqueCount)
	} else if uniqueCount := jsonStr(m, "uniqueCount"); uniqueCount != "" {
		fmt.Fprintf(w, "Unique Viewers\t%s\n", uniqueCount)
	}

	if lastViewed := jsonStr(m, "lastViewed"); lastViewed != "" {
		fmt.Fprintf(w, "Last Viewed\t%s\n", lastViewed)
	}

	if fromDate := jsonStr(m, "fromDate"); fromDate != "" {
		fmt.Fprintf(w, "Analytics From\t%s\n", fromDate)
	}

	if toDate := jsonStr(m, "toDate"); toDate != "" {
		fmt.Fprintf(w, "Analytics To\t%s\n", toDate)
	}

	w.Flush()

	// If there are viewers breakdown
	if viewers := jsonArr(m, "viewers"); len(viewers) > 0 {
		fmt.Println()
		fmt.Println("Recent Viewers:")
		fmt.Println()
		vw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(vw, "USER\tVIEWS\tLAST VIEWED")
		fmt.Fprintln(vw, "----\t-----\t-----------")
		for _, v := range viewers {
			vm := asMap(v)
			if vm == nil {
				continue
			}
			user := jsonMap(vm, "user")
			userName := "unknown"
			if user != nil {
				userName = strOr(jsonStr(user, "displayName"), jsonStr(user, "username"))
			}
			viewCount := jsonStr(vm, "viewCount")
			lastView := jsonStr(vm, "lastViewed")
			fmt.Fprintf(vw, "%s\t%s\t%s\n", userName, viewCount, lastView)
		}
		vw.Flush()
	}
}

func showBasicPageStats(c *apiClient, pageID string) {
	params := url.Values{
		"expand": {"version,history"},
	}
	data, err := c.get("/content/"+pageID, params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)

	version := jsonMap(m, "version")
	history := jsonMap(m, "history")
	createdBy := jsonMap(history, "createdBy")
	lastUpdatedBy := jsonMap(version, "by")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "METRIC\tVALUE")
	fmt.Fprintln(w, "------\t-----")
	fmt.Fprintf(w, "Page ID\t%s\n", jsonStr(m, "id"))
	fmt.Fprintf(w, "Title\t%s\n", jsonStr(m, "title"))
	fmt.Fprintf(w, "Version\t%s\n", jsonStr(version, "number"))
	fmt.Fprintf(w, "Created\t%s\n", jsonStr(history, "createdDate"))
	fmt.Fprintf(w, "Creator\t%s\n", strOr(jsonStr(createdBy, "displayName"), "unknown"))
	fmt.Fprintf(w, "Last Updated\t%s\n", jsonStr(version, "when"))
	fmt.Fprintf(w, "Updated By\t%s\n", strOr(jsonStr(lastUpdatedBy, "displayName"), "unknown"))
	w.Flush()
}

// ── Tree View Navigation ────────────────────────────────────

type pageNode struct {
	ID       string
	Title    string
	Children []*pageNode
}

func cmdTree(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: tree <space-key> [root-page-id]")
	}
	spaceKey := args[0]
	rootPageID := ""
	if len(args) > 1 {
		rootPageID = args[1]
	}

	if rootPageID != "" {
		// Build tree from specific root page
		root := fetchPageTree(c, rootPageID, 0, 5) // limit depth to 5 levels
		if root == nil {
			die("Failed to fetch page tree")
		}
		fmt.Printf("Page tree for: %s\n\n", root.Title)
		printTree(root, "", true)
	} else {
		// Get all root pages in the space and build tree
		fmt.Printf("Page tree for space: %s\n\n", spaceKey)
		params := url.Values{
			"limit": {"100"},
			"depth": {"root"},
		}
		data, err := c.get("/space/"+spaceKey+"/content/page", params)
		if err != nil {
			die("%s", err)
		}
		var m map[string]any
		json.Unmarshal(data, &m)

		results := jsonArr(m, "results")
		if len(results) == 0 {
			fmt.Println("No pages found in this space.")
			return
		}

		for i, r := range results {
			rm := asMap(r)
			if rm == nil {
				continue
			}
			pageID := jsonStr(rm, "id")
			root := fetchPageTree(c, pageID, 0, 4)
			if root != nil {
				isLast := i == len(results)-1
				printTree(root, "", isLast)
			}
		}
	}
}

func fetchPageTree(c *apiClient, pageID string, depth, maxDepth int) *pageNode {
	if depth > maxDepth {
		return nil
	}

	params := url.Values{
		"expand": {"title"},
	}
	data, err := c.get("/content/"+pageID, params)
	if err != nil {
		return nil
	}
	var m map[string]any
	json.Unmarshal(data, &m)

	node := &pageNode{
		ID:    jsonStr(m, "id"),
		Title: jsonStr(m, "title"),
	}

	// Get children
	childParams := url.Values{
		"limit": {"100"},
	}
	childData, err := c.get("/content/"+pageID+"/child/page", childParams)
	if err != nil {
		return node
	}
	var cm map[string]any
	json.Unmarshal(childData, &cm)

	for _, r := range jsonArr(cm, "results") {
		rm := asMap(r)
		if rm == nil {
			continue
		}
		childID := jsonStr(rm, "id")
		childNode := fetchPageTree(c, childID, depth+1, maxDepth)
		if childNode != nil {
			node.Children = append(node.Children, childNode)
		}
	}

	return node
}

func printTree(node *pageNode, prefix string, isLast bool) {
	if node == nil {
		return
	}

	connector := "|-- "
	if isLast {
		connector = "`-- "
	}

	if prefix == "" {
		fmt.Printf("%s (id: %s)\n", node.Title, node.ID)
	} else {
		fmt.Printf("%s%s%s (id: %s)\n", prefix, connector, node.Title, node.ID)
	}

	childPrefix := prefix
	if prefix != "" {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "|   "
		}
	}

	for i, child := range node.Children {
		isChildLast := i == len(node.Children)-1
		printTree(child, childPrefix, isChildLast)
	}
}

// ── Utilities ───────────────────────────────────────────────

func stripHTML(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			result.WriteRune(' ')
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	// Collapse multiple spaces
	out := result.String()
	for strings.Contains(out, "  ") {
		out = strings.ReplaceAll(out, "  ", " ")
	}
	return strings.TrimSpace(out)
}

// ── Date/Time Utilities for Calendar ────────────────────────

// parseDatetime parses a date or datetime string into Unix milliseconds.
// Supported formats:
//   - Unix timestamp (milliseconds): "1709251200000"
//   - Date only: "2024-02-29" (interpreted as all-day, returns start of day)
//   - Date and time: "2024-02-29 14:30"
//
// Returns (timestamp_ms, is_all_day, error)
func parseDatetime(s string) (int64, bool, error) {
	s = strings.TrimSpace(s)

	// Check if it's already a Unix timestamp (milliseconds)
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil && ts > 1000000000000 {
		return ts, false, nil
	}

	// Try date-only format (all-day event)
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return t.UnixMilli(), true, nil
	}

	// Try date with time format
	if t, err := time.ParseInLocation("2006-01-02 15:04", s, time.Local); err == nil {
		return t.UnixMilli(), false, nil
	}

	return 0, false, fmt.Errorf("invalid date format '%s': use 'YYYY-MM-DD' or 'YYYY-MM-DD HH:MM'", s)
}

// parseDate parses a date string (YYYY-MM-DD) into Unix milliseconds (start of day).
func parseDate(s string) (int64, error) {
	s = strings.TrimSpace(s)
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return 0, fmt.Errorf("invalid date format '%s': use 'YYYY-MM-DD'", s)
	}
	return t.UnixMilli(), nil
}

// formatTimestamp formats a Unix millisecond timestamp to a human-readable string.
func formatTimestamp(ms int64) string {
	if ms == 0 {
		return "unknown"
	}
	t := time.UnixMilli(ms)
	return t.Format("2006-01-02 15:04")
}

// formatTimestampFromAny handles JSON number or string timestamps.
func formatTimestampFromAny(v any) string {
	switch t := v.(type) {
	case float64:
		return formatTimestamp(int64(t))
	case int64:
		return formatTimestamp(t)
	case string:
		if ms, err := strconv.ParseInt(t, 10, 64); err == nil {
			return formatTimestamp(ms)
		}
		return t
	}
	return "unknown"
}

// getMonthBounds returns the start and end timestamps (in ms) for the current month.
func getMonthBounds() (int64, int64) {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)
	return startOfMonth.UnixMilli(), endOfMonth.UnixMilli()
}

// ── Calendar Commands ───────────────────────────────────────

func cmdCalendars(c *apiClient) {
	// Try the user's calendars endpoint first
	data, err := c.getCalendar("/user/current/calendars", nil)
	if err != nil {
		// Try alternative endpoint for subcalendars
		data, err = c.getCalendar("/subcalendars", nil)
		if err != nil {
			die("%s", err)
		}
	}

	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		die("Failed to parse calendar response: %s", err)
	}

	// Handle different response formats
	var calendars []any
	switch v := result.(type) {
	case []any:
		calendars = v
	case map[string]any:
		if arr := jsonArr(v, "results"); arr != nil {
			calendars = arr
		} else if arr := jsonArr(v, "calendars"); arr != nil {
			calendars = arr
		} else if arr := jsonArr(v, "payload"); arr != nil {
			calendars = arr
		} else {
			// Single calendar object or wrapped response
			calendars = []any{v}
		}
	}

	if len(calendars) == 0 {
		fmt.Println("No calendars found.")
		return
	}

	fmt.Printf("Available calendars (%d):\n\n", len(calendars))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tCOLOR")
	fmt.Fprintln(w, "--\t----\t----\t-----")

	for _, cal := range calendars {
		cm := asMap(cal)
		if cm == nil {
			continue
		}
		id := strOr(jsonStr(cm, "id"), jsonStr(cm, "subCalendarId"))
		name := strOr(jsonStr(cm, "name"), jsonStr(cm, "title"))
		calType := strOr(jsonStr(cm, "type"), jsonStr(cm, "calendarType"))
		color := strOr(jsonStr(cm, "color"), jsonStr(cm, "colour"))
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", id, name, calType, color)
	}
	w.Flush()
}

func cmdCalendarEvents(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: calendar-events <calendar-id> [start-date] [end-date]")
	}
	calendarID := args[0]

	// Default to current month if no dates provided
	var startMS, endMS int64
	if len(args) >= 3 {
		var err error
		startMS, err = parseDate(args[1])
		if err != nil {
			die("%s", err)
		}
		endMS, err = parseDate(args[2])
		if err != nil {
			die("%s", err)
		}
		// End date should be end of day
		endMS += 24*60*60*1000 - 1
	} else if len(args) == 2 {
		var err error
		startMS, err = parseDate(args[1])
		if err != nil {
			die("%s", err)
		}
		// Single date: show that whole day
		endMS = startMS + 24*60*60*1000 - 1
	} else {
		startMS, endMS = getMonthBounds()
	}

	params := url.Values{
		"subCalendarId": {calendarID},
		"start":         {strconv.FormatInt(startMS, 10)},
		"end":           {strconv.FormatInt(endMS, 10)},
	}

	data, err := c.getCalendar("/events.json", params)
	if err != nil {
		die("%s", err)
	}

	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		die("Failed to parse events response: %s", err)
	}

	// Handle different response formats
	var events []any
	switch v := result.(type) {
	case []any:
		events = v
	case map[string]any:
		if arr := jsonArr(v, "events"); arr != nil {
			events = arr
		} else if arr := jsonArr(v, "results"); arr != nil {
			events = arr
		} else if arr := jsonArr(v, "payload"); arr != nil {
			events = arr
		}
	}

	if len(events) == 0 {
		fmt.Printf("No events found in calendar %s for the specified period.\n", calendarID)
		fmt.Printf("Period: %s to %s\n", formatTimestamp(startMS), formatTimestamp(endMS))
		return
	}

	fmt.Printf("Events in calendar %s (%d found):\n", calendarID, len(events))
	fmt.Printf("Period: %s to %s\n\n", formatTimestamp(startMS), formatTimestamp(endMS))

	for _, ev := range events {
		em := asMap(ev)
		if em == nil {
			continue
		}

		id := strOr(jsonStr(em, "id"), jsonStr(em, "eventId"))
		title := strOr(jsonStr(em, "title"), jsonStr(em, "summary"))
		description := strOr(jsonStr(em, "description"), "")
		allDay := false
		if v, ok := em["allDay"]; ok {
			if b, ok := v.(bool); ok {
				allDay = b
			}
		}

		// Format times
		startTime := formatTimestampFromAny(em["start"])
		endTime := formatTimestampFromAny(em["end"])

		fmt.Printf("Event ID: %s\n", id)
		fmt.Printf("  Title: %s\n", title)
		if allDay {
			fmt.Printf("  Time: All day\n")
		} else {
			fmt.Printf("  Start: %s\n", startTime)
			fmt.Printf("  End: %s\n", endTime)
		}
		if description != "" {
			desc := description
			if len(desc) > 100 {
				desc = desc[:100] + "..."
			}
			fmt.Printf("  Description: %s\n", desc)
		}
		fmt.Println()
	}
}

func cmdCalendarEvent(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: calendar-event <event-id>")
	}
	eventID := args[0]

	data, err := c.getCalendar("/events/"+eventID+".json", nil)
	if err != nil {
		die("%s", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		die("Failed to parse event response: %s", err)
	}

	// Check if response has a nested event object
	if ev := jsonMap(m, "event"); ev != nil {
		m = ev
	}

	id := strOr(jsonStr(m, "id"), jsonStr(m, "eventId"))
	title := strOr(jsonStr(m, "title"), jsonStr(m, "summary"))
	description := jsonStr(m, "description")
	calendarID := strOr(jsonStr(m, "subCalendarId"), jsonStr(m, "calendarId"))
	allDay := false
	if v, ok := m["allDay"]; ok {
		if b, ok := v.(bool); ok {
			allDay = b
		}
	}

	fmt.Printf("Event Details\n")
	fmt.Printf("=============\n\n")
	fmt.Printf("ID: %s\n", id)
	fmt.Printf("Title: %s\n", title)
	fmt.Printf("Calendar ID: %s\n", calendarID)
	if allDay {
		fmt.Printf("Time: All day\n")
	} else {
		fmt.Printf("Start: %s\n", formatTimestampFromAny(m["start"]))
		fmt.Printf("End: %s\n", formatTimestampFromAny(m["end"]))
	}
	if description != "" {
		fmt.Printf("\nDescription:\n%s\n", description)
	}

	// Show additional metadata if available
	if location := jsonStr(m, "location"); location != "" {
		fmt.Printf("\nLocation: %s\n", location)
	}
	if organizer := jsonMap(m, "organizer"); organizer != nil {
		fmt.Printf("Organizer: %s\n", strOr(jsonStr(organizer, "displayName"), jsonStr(organizer, "name")))
	}
}

func cmdCalendarEventAdd(c *apiClient, args []string) {
	if len(args) < 4 {
		die("Usage: calendar-event-add <calendar-id> <title> <start-datetime> <end-datetime> [description]")
	}
	calendarID := args[0]
	title := args[1]
	startStr := args[2]
	endStr := args[3]
	description := ""
	if len(args) > 4 {
		description = args[4]
	}

	startMS, startAllDay, err := parseDatetime(startStr)
	if err != nil {
		die("%s", err)
	}
	endMS, endAllDay, err := parseDatetime(endStr)
	if err != nil {
		die("%s", err)
	}

	// If both dates are date-only (no time), it's an all-day event
	allDay := startAllDay && endAllDay

	// For all-day events, ensure end is after start
	if allDay && endMS <= startMS {
		endMS = startMS + 24*60*60*1000 // Add one day
	}

	payload := map[string]any{
		"subCalendarId": calendarID,
		"title":         title,
		"start":         startMS,
		"end":           endMS,
		"allDay":        allDay,
	}
	if description != "" {
		payload["description"] = description
	}

	data, err := c.postCalendar("/events.json", payload)
	if err != nil {
		die("%s", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		die("Failed to parse response: %s", err)
	}

	// Check for nested event object
	if ev := jsonMap(m, "event"); ev != nil {
		m = ev
	}

	eventID := strOr(jsonStr(m, "id"), jsonStr(m, "eventId"))
	fmt.Printf("Event created successfully!\n\n")
	fmt.Printf("Event ID: %s\n", eventID)
	fmt.Printf("Title: %s\n", title)
	fmt.Printf("Calendar: %s\n", calendarID)
	if allDay {
		fmt.Printf("Time: All day\n")
	} else {
		fmt.Printf("Start: %s\n", formatTimestamp(startMS))
		fmt.Printf("End: %s\n", formatTimestamp(endMS))
	}
}

func cmdCalendarEventUpdate(c *apiClient, args []string) {
	if len(args) < 4 {
		die("Usage: calendar-event-update <event-id> <title> <start-datetime> <end-datetime> [description]")
	}
	eventID := args[0]
	title := args[1]
	startStr := args[2]
	endStr := args[3]
	description := ""
	if len(args) > 4 {
		description = args[4]
	}

	startMS, startAllDay, err := parseDatetime(startStr)
	if err != nil {
		die("%s", err)
	}
	endMS, endAllDay, err := parseDatetime(endStr)
	if err != nil {
		die("%s", err)
	}

	allDay := startAllDay && endAllDay

	// For all-day events, ensure end is after start
	if allDay && endMS <= startMS {
		endMS = startMS + 24*60*60*1000
	}

	// First, get the current event to find the calendar ID
	existingData, err := c.getCalendar("/events/"+eventID+".json", nil)
	if err != nil {
		die("Failed to fetch existing event: %s", err)
	}
	var existing map[string]any
	json.Unmarshal(existingData, &existing)
	if ev := jsonMap(existing, "event"); ev != nil {
		existing = ev
	}

	calendarID := strOr(jsonStr(existing, "subCalendarId"), jsonStr(existing, "calendarId"))

	payload := map[string]any{
		"id":            eventID,
		"subCalendarId": calendarID,
		"title":         title,
		"start":         startMS,
		"end":           endMS,
		"allDay":        allDay,
	}
	if description != "" {
		payload["description"] = description
	}

	data, err := c.putCalendar("/events/"+eventID+".json", payload)
	if err != nil {
		die("%s", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		die("Failed to parse response: %s", err)
	}

	fmt.Printf("Event updated successfully!\n\n")
	fmt.Printf("Event ID: %s\n", eventID)
	fmt.Printf("Title: %s\n", title)
	if allDay {
		fmt.Printf("Time: All day\n")
	} else {
		fmt.Printf("Start: %s\n", formatTimestamp(startMS))
		fmt.Printf("End: %s\n", formatTimestamp(endMS))
	}
}

func cmdCalendarEventDelete(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: calendar-event-delete <event-id>")
	}
	eventID := args[0]

	err := c.deleteCalendar("/events/" + eventID + ".json")
	if err != nil {
		die("%s", err)
	}

	fmt.Printf("Event %s deleted successfully.\n", eventID)
}

// ── Help ────────────────────────────────────────────────────

func printHelp() {
	fmt.Println(`Usage: confluence-navigator <host> <command> [args...]

Discovery:
  discover [substring]              Find Confluence hosts in ~/.netrc

Commands (first arg is hostname or unique substring from ~/.netrc):

  Connection:
    <host> whoami                   Show current user
    <host> test                     Test connection

  Content Discovery:
    <host> recent [limit]           Recent content changes
    <host> watched                  List watched content
    <host> watch-changes [days]     Changes to watched content (default: 7 days)
    <host> search <CQL> [limit]     Search via CQL

  Spaces:
    <host> spaces [limit]           List spaces
    <host> create-space <key> <name> [desc]  Create a personal space
    <host> space-pages <key> [limit]  Pages in a space

  Pages:
    <host> page <id> [format]       Get page content (format: storage|view)
    <host> page-info <id>           Get page metadata
    <host> children <id>            List child pages
    <host> labels <id>              List page labels
    <host> history <id> [limit]     Page version history
    <host> create-page <space-key> <title> [parent-id] [content]  Create a page
    <host> update-page <id> <content> [replace|append]  Update page content
    <host> tree <space-key> [root-page-id]  Show page hierarchy tree

  Comments:
    <host> comments <page-id>       List comments on a page
    <host> comment-add <page-id> <comment-text>  Add a comment to a page
    <host> comment-update <comment-id> <new-text>  Update an existing comment

  Watch/Unwatch:
    <host> watch <page-id>          Start watching a page
    <host> unwatch <page-id>        Stop watching a page

  Reading List:
    <host> read-later-add <page-id>     Add page to reading list
    <host> read-later-remove <page-id>  Remove page from reading list
    <host> read-later-list              List pages in reading list

  Analytics:
    <host> analytics <page-id>      Get page view statistics

  Calendars (Team Calendars plugin):
    <host> calendars                List available calendars
    <host> calendar-events <id> [start] [end]  List events in a calendar
                                    Dates: YYYY-MM-DD (default: current month)
    <host> calendar-event <event-id>  Get event details
    <host> calendar-event-add <cal-id> <title> <start> <end> [desc]
                                    Create a calendar event
                                    Datetime: "YYYY-MM-DD HH:MM" or "YYYY-MM-DD" (all-day)
    <host> calendar-event-update <event-id> <title> <start> <end> [desc]
                                    Update a calendar event
    <host> calendar-event-delete <event-id>  Delete a calendar event`)
}

// ── Main ────────────────────────────────────────────────────

func main() {
	args := os.Args[1:]
	if len(args) == 0 || args[0] == "help" {
		printHelp()
		return
	}

	if args[0] == "discover" {
		cmdDiscover(args[1:])
		return
	}

	if len(args) < 2 {
		printHelp()
		return
	}

	host := args[0]
	command := args[1]
	cmdArgs := args[2:]

	hostname, entry, err := resolveHost(host, "confluence")
	if err != nil {
		die("%s", err)
	}
	client := newClient(hostname, entry)

	switch command {
	case "whoami":
		cmdWhoami(client)
	case "test":
		cmdTest(client)
	case "recent":
		cmdRecent(client, cmdArgs)
	case "watched":
		cmdWatched(client)
	case "watch-changes":
		cmdWatchChanges(client, cmdArgs)
	case "search":
		cmdSearch(client, cmdArgs)
	case "spaces":
		cmdSpaces(client, cmdArgs)
	case "create-space":
		cmdCreateSpace(client, cmdArgs)
	case "create-page":
		cmdCreatePage(client, cmdArgs)
	case "update-page":
		cmdUpdatePage(client, cmdArgs)
	case "space-pages":
		cmdSpacePages(client, cmdArgs)
	case "page":
		cmdPage(client, cmdArgs)
	case "page-info":
		cmdPageInfo(client, cmdArgs)
	case "children":
		cmdChildren(client, cmdArgs)
	case "labels":
		cmdLabels(client, cmdArgs)
	case "history":
		cmdHistory(client, cmdArgs)
	// New commands
	case "comments":
		cmdComments(client, cmdArgs)
	case "comment-add":
		cmdCommentAdd(client, cmdArgs)
	case "comment-update":
		cmdCommentUpdate(client, cmdArgs)
	case "watch":
		cmdWatch(client, cmdArgs)
	case "unwatch":
		cmdUnwatch(client, cmdArgs)
	case "read-later-add":
		cmdReadLaterAdd(client, cmdArgs)
	case "read-later-remove":
		cmdReadLaterRemove(client, cmdArgs)
	case "read-later-list":
		cmdReadLaterList(client)
	case "analytics":
		cmdAnalytics(client, cmdArgs)
	case "tree":
		cmdTree(client, cmdArgs)
	// Calendar commands
	case "calendars":
		cmdCalendars(client)
	case "calendar-events":
		cmdCalendarEvents(client, cmdArgs)
	case "calendar-event":
		cmdCalendarEvent(client, cmdArgs)
	case "calendar-event-add":
		cmdCalendarEventAdd(client, cmdArgs)
	case "calendar-event-update":
		cmdCalendarEventUpdate(client, cmdArgs)
	case "calendar-event-delete":
		cmdCalendarEventDelete(client, cmdArgs)
	case "help":
		printHelp()
	default:
		die("Unknown command: %s (run with 'help')", command)
	}
}
