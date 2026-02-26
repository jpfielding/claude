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

// ── Help ────────────────────────────────────────────────────

func printHelp() {
	fmt.Println(`Usage: confluence-navigator <host> <command> [args...]

Discovery:
  discover [substring]              Find Confluence hosts in ~/.netrc

Commands (first arg is hostname or unique substring from ~/.netrc):
  <host> whoami                     Show current user
  <host> test                       Test connection
  <host> recent [limit]             Recent content changes
  <host> watched                    List watched content
  <host> watch-changes [days]       Changes to watched content (default: 7 days)
  <host> search <CQL> [limit]       Search via CQL
  <host> spaces [limit]             List spaces
  <host> create-space <key> <name> [desc]  Create a personal space
  <host> create-page <space-key> <title> [parent-id] [content]  Create a page
  <host> update-page <id> <content> [replace|append]  Update page content
  <host> space-pages <key> [limit]  Pages in a space
  <host> page <id> [format]         Get page content
  <host> page-info <id>             Get page metadata
  <host> children <id>              List child pages
  <host> labels <id>                List page labels
  <host> history <id> [limit]       Page version history`)
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
	case "help":
		printHelp()
	default:
		die("Unknown command: %s (run with 'help')", command)
	}
}
