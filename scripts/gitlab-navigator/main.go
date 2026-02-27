package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' {
			i++
			continue
		}
		if s[i] == '#' {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
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
		return "", netrcEntry{}, fmt.Errorf("no ~/.netrc machine entry matching '%s'", input)
	}
	if len(matches) > 1 {
		msg := fmt.Sprintf("Multiple matches for '%s':\n", input)
		for _, m := range matches {
			msg += fmt.Sprintf("  %s\n", m.Machine)
		}
		return "", netrcEntry{}, fmt.Errorf("%sambiguous host substring '%s'. Use a more specific name or full hostname.", msg, input)
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

// ── HTTP client (PRIVATE-TOKEN) ─────────────────────────────

type apiClient struct {
	baseURL  string
	login    string
	password string
}

func newClient(host string, entry netrcEntry) *apiClient {
	return &apiClient{
		baseURL:  "https://" + host,
		login:    entry.Login,
		password: entry.Password,
	}
}

func (c *apiClient) doRequest(method, fullURL string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.password)
	req.Header.Set("Accept", "application/json")
	if method != "GET" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	return http.DefaultClient.Do(req)
}

func (c *apiClient) get(endpoint string, params url.Values) (json.RawMessage, error) {
	u := c.baseURL + "/api/v4" + endpoint
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	resp, err := c.doRequest("GET", u, nil)
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

func (c *apiClient) getWithHeaders(endpoint string, params url.Values) (json.RawMessage, http.Header, error) {
	u := c.baseURL + "/api/v4" + endpoint
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	resp, err := c.doRequest("GET", u, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, nil, fmt.Errorf("API returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.RawMessage(body), resp.Header, nil
}

func (c *apiClient) post(endpoint string, form url.Values) (json.RawMessage, error) {
	u := c.baseURL + "/api/v4" + endpoint
	resp, err := c.doRequest("POST", u, bytes.NewBufferString(form.Encode()))
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

func defaultProjectPath(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "project"
	}
	return out
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func toStringSlice(arr []any) []string {
	var result []string
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// ── Commands ────────────────────────────────────────────────

func cmdDiscover(args []string) {
	filter := "gitlab"
	if len(args) > 0 {
		filter = args[0]
	}
	hosts := discoverHosts(filter)
	if len(hosts) == 0 {
		fmt.Printf("No ~/.netrc entries matching '%s' found.\n\n", filter)
		fmt.Println("To search with a different substring: gitlab-navigator discover <substring>")
		return
	}
	fmt.Printf("Found %d matching host(s) in ~/.netrc:\n", len(hosts))
	for _, h := range hosts {
		fmt.Printf("  %s\n", h)
	}
	fmt.Println()
	fmt.Println("Usage: gitlab-navigator <hostname-or-substring> <command>")
}

func cmdWhoami(c *apiClient) {
	data, err := c.get("/user", nil)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	out := map[string]any{
		"username": jsonStr(m, "username"),
		"name":     jsonStr(m, "name"),
		"email":    jsonStr(m, "email"),
		"state":    jsonStr(m, "state"),
		"is_admin": m["is_admin"],
		"id":       m["id"],
	}
	printJSON(out)
}

func cmdTest(c *apiClient) {
	fmt.Println("Testing connection...")
	data, err := c.get("/user", nil)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	name := strOr(jsonStr(m, "name"), strOr(jsonStr(m, "username"), "unknown"))
	admin := strOr(jsonStr(m, "is_admin"), "false")
	fmt.Printf("Connected as: %s (admin: %s)\n\n", name, admin)

	fmt.Println("Testing project listing...")
	projects, err := c.get("/projects", url.Values{
		"per_page": {"3"}, "order_by": {"updated_at"}, "sort": {"desc"}, "membership": {"true"},
	})
	if err != nil {
		die("%s", err)
	}
	var pList []any
	json.Unmarshal(projects, &pList)
	for _, p := range pList {
		pm := asMap(p)
		if pm != nil {
			fmt.Printf("  %s\n", jsonStr(pm, "path_with_namespace"))
		}
	}
	fmt.Println()

	vData, err := c.get("/version", nil)
	ver := "unknown"
	if err == nil {
		var vm map[string]any
		json.Unmarshal(vData, &vm)
		ver = strOr(jsonStr(vm, "version"), "unknown")
	}
	fmt.Printf("GitLab version: %s\n", ver)
	fmt.Println("Connection OK.")
}

func cmdStarred(c *apiClient, args []string) {
	limit := "25"
	if len(args) > 0 {
		limit = args[0]
	}
	data, err := c.get("/projects", url.Values{
		"starred": {"true"}, "per_page": {limit}, "order_by": {"updated_at"}, "sort": {"desc"},
	})
	if err != nil {
		die("%s", err)
	}
	var projects []any
	json.Unmarshal(data, &projects)
	for _, p := range projects {
		pm := asMap(p)
		if pm == nil {
			continue
		}
		fmt.Printf("%s\n", jsonStr(pm, "path_with_namespace"))
		fmt.Printf("  Updated: %s  Stars: %s  Forks: %s\n",
			jsonStr(pm, "last_activity_at"),
			strOr(jsonStr(pm, "star_count"), "0"),
			strOr(jsonStr(pm, "forks_count"), "0"))
		fmt.Printf("  URL: %s\n\n", jsonStr(pm, "web_url"))
	}
}

func cmdStarredActivity(c *apiClient, args []string) {
	days := 7
	if len(args) > 0 {
		days, _ = strconv.Atoi(args[0])
		if days <= 0 {
			days = 7
		}
	}
	after := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour).Format(time.RFC3339)
	data, err := c.get("/projects", url.Values{
		"starred": {"true"}, "per_page": {"100"}, "order_by": {"updated_at"}, "sort": {"desc"},
	})
	if err != nil {
		die("%s", err)
	}
	var projects []any
	json.Unmarshal(data, &projects)
	var active []map[string]any
	for _, p := range projects {
		pm := asMap(p)
		if pm == nil {
			continue
		}
		if jsonStr(pm, "last_activity_at") > after {
			active = append(active, pm)
		}
	}
	fmt.Printf("Starred projects with activity in last %d days (%d projects):\n\n", days, len(active))
	for _, pm := range active {
		fmt.Printf("%s\n", jsonStr(pm, "path_with_namespace"))
		fmt.Printf("  Last activity: %s\n", jsonStr(pm, "last_activity_at"))
		fmt.Printf("  Default branch: %s\n\n", strOr(jsonStr(pm, "default_branch"), "main"))
	}
}

func cmdEvents(c *apiClient, args []string) {
	limit := "20"
	if len(args) > 0 {
		limit = args[0]
	}
	data, err := c.get("/events", url.Values{"per_page": {limit}, "sort": {"desc"}})
	if err != nil {
		die("%s", err)
	}
	var events []any
	json.Unmarshal(data, &events)
	for _, e := range events {
		em := asMap(e)
		if em == nil {
			continue
		}
		pushData := jsonMap(em, "push_data")
		targetType := strOr(jsonStr(em, "target_type"), strOr(jsonStr(pushData, "ref_type"), "project"))
		targetTitle := strOr(jsonStr(em, "target_title"), strOr(jsonStr(pushData, "commit_title"), "N/A"))
		fmt.Printf("%s [%s] %s: %s\n",
			jsonStr(em, "created_at"),
			jsonStr(em, "action_name"),
			targetType,
			targetTitle)
		fmt.Printf("  Project: %s  Author: %s\n\n",
			jsonStr(em, "project_id"),
			strOr(jsonStr(em, "author_username"), "unknown"))
	}
}

func cmdProjectEvents(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: project-events <project-id-or-path> [limit]")
	}
	projectRef := args[0]
	limit := "20"
	if len(args) > 1 {
		limit = args[1]
	}
	encoded := url.PathEscape(projectRef)
	data, err := c.get("/projects/"+encoded+"/events", url.Values{"per_page": {limit}, "sort": {"desc"}})
	if err != nil {
		die("%s", err)
	}
	var events []any
	json.Unmarshal(data, &events)
	for _, e := range events {
		em := asMap(e)
		if em == nil {
			continue
		}
		pushData := jsonMap(em, "push_data")
		targetType := strOr(jsonStr(em, "target_type"), strOr(jsonStr(pushData, "ref_type"), "event"))
		targetTitle := strOr(jsonStr(em, "target_title"), strOr(jsonStr(pushData, "commit_title"), "N/A"))
		fmt.Printf("%s [%s] %s: %s\n",
			jsonStr(em, "created_at"),
			jsonStr(em, "action_name"),
			targetType,
			targetTitle)
		fmt.Printf("  Author: %s\n\n", strOr(jsonStr(em, "author_username"), "unknown"))
	}
}

func cmdProjects(c *apiClient, args []string) {
	limit := "25"
	if len(args) > 0 {
		limit = args[0]
	}
	data, headers, err := c.getWithHeaders("/projects", url.Values{
		"per_page": {limit}, "order_by": {"updated_at"}, "sort": {"desc"}, "membership": {"true"},
	})
	if err != nil {
		die("%s", err)
	}
	total := headers.Get("X-Total")
	if total != "" {
		fmt.Printf("Your projects (%s total):\n\n", total)
	} else {
		fmt.Println("Your projects:")
		fmt.Println()
	}
	var projects []any
	json.Unmarshal(data, &projects)
	for _, p := range projects {
		pm := asMap(p)
		if pm == nil {
			continue
		}
		fmt.Printf("%s\n", jsonStr(pm, "path_with_namespace"))
		fmt.Printf("  Updated: %s  Visibility: %s  Default: %s\n\n",
			jsonStr(pm, "last_activity_at"),
			jsonStr(pm, "visibility"),
			strOr(jsonStr(pm, "default_branch"), "main"))
	}
}

func cmdProjectInfo(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: project-info <project-id-or-path>")
	}
	encoded := url.PathEscape(args[0])
	data, err := c.get("/projects/"+encoded, url.Values{"statistics": {"true"}})
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	out := map[string]any{
		"id":                m["id"],
		"name":              jsonStr(m, "name"),
		"path":              jsonStr(m, "path_with_namespace"),
		"description":       strOr(jsonStr(m, "description"), "none"),
		"visibility":        jsonStr(m, "visibility"),
		"default_branch":    strOr(jsonStr(m, "default_branch"), "main"),
		"web_url":           jsonStr(m, "web_url"),
		"created":           jsonStr(m, "created_at"),
		"updated":           jsonStr(m, "last_activity_at"),
		"creator":           m["creator_id"],
		"topics":            m["topics"],
		"star_count":        m["star_count"],
		"forks_count":       m["forks_count"],
		"open_issues_count": m["open_issues_count"],
		"statistics":        m["statistics"],
	}
	printJSON(out)
}

func cmdCreateProject(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: create-project <name> [path] [visibility] [namespace-id]")
	}
	name := strings.TrimSpace(args[0])
	if name == "" {
		die("project name cannot be empty")
	}
	path := defaultProjectPath(name)
	if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
		path = strings.TrimSpace(args[1])
	}
	visibility := "private"
	if len(args) > 2 && strings.TrimSpace(args[2]) != "" {
		visibility = strings.TrimSpace(args[2])
	}

	form := url.Values{
		"name":       {name},
		"path":       {path},
		"visibility": {visibility},
	}
	if len(args) > 3 && strings.TrimSpace(args[3]) != "" {
		form.Set("namespace_id", strings.TrimSpace(args[3]))
	}

	data, err := c.post("/projects", form)
	if err != nil {
		die("%s", err)
	}

	var m map[string]any
	json.Unmarshal(data, &m)
	out := map[string]any{
		"id":                  m["id"],
		"name":                jsonStr(m, "name"),
		"path_with_namespace": jsonStr(m, "path_with_namespace"),
		"visibility":          jsonStr(m, "visibility"),
		"default_branch":      strOr(jsonStr(m, "default_branch"), "main"),
		"http_url_to_repo":    jsonStr(m, "http_url_to_repo"),
		"ssh_url_to_repo":     jsonStr(m, "ssh_url_to_repo"),
		"web_url":             jsonStr(m, "web_url"),
	}
	printJSON(out)
}

func cmdMyMRs(c *apiClient, args []string) {
	state := "opened"
	limit := "25"
	if len(args) > 0 {
		state = args[0]
	}
	if len(args) > 1 {
		limit = args[1]
	}
	data, err := c.get("/merge_requests", url.Values{
		"state": {state}, "scope": {"assigned_to_me"}, "per_page": {limit},
		"order_by": {"updated_at"}, "sort": {"desc"},
	})
	if err != nil {
		die("%s", err)
	}
	var mrs []any
	json.Unmarshal(data, &mrs)
	fmt.Printf("Merge requests assigned to you (%s, %d shown):\n\n", state, len(mrs))
	for _, mr := range mrs {
		mm := asMap(mr)
		if mm == nil {
			continue
		}
		author := jsonMap(mm, "author")
		refs := jsonMap(mm, "references")
		ref := jsonStr(refs, "full")
		if ref == "" {
			ref = jsonStr(mm, "web_url")
		}
		fmt.Printf("!%s [%s] %s\n", jsonStr(mm, "iid"), jsonStr(mm, "state"), jsonStr(mm, "title"))
		fmt.Printf("  Project: %s  Author: %s\n", ref, strOr(jsonStr(author, "username"), "unknown"))
		fmt.Printf("  Updated: %s  Target: %s\n\n", jsonStr(mm, "updated_at"), jsonStr(mm, "target_branch"))
	}
}

func cmdMRReview(c *apiClient, args []string) {
	state := "opened"
	limit := "25"
	if len(args) > 0 {
		state = args[0]
	}
	if len(args) > 1 {
		limit = args[1]
	}
	// Get current username
	userData, err := c.get("/user", nil)
	if err != nil {
		die("%s", err)
	}
	var um map[string]any
	json.Unmarshal(userData, &um)
	username := jsonStr(um, "username")

	data, err := c.get("/merge_requests", url.Values{
		"state": {state}, "scope": {"all"}, "reviewer_username": {username},
		"per_page": {limit}, "order_by": {"updated_at"}, "sort": {"desc"},
	})
	if err != nil {
		// Fallback
		data, err = c.get("/merge_requests", url.Values{
			"state": {state}, "scope": {"all"}, "per_page": {limit},
			"order_by": {"updated_at"}, "sort": {"desc"},
		})
		if err != nil {
			die("%s", err)
		}
	}
	var mrs []any
	json.Unmarshal(data, &mrs)
	fmt.Printf("Merge requests awaiting your review (%s, %d shown):\n\n", state, len(mrs))
	for _, mr := range mrs {
		mm := asMap(mr)
		if mm == nil {
			continue
		}
		author := jsonMap(mm, "author")
		refs := jsonMap(mm, "references")
		ref := jsonStr(refs, "full")
		if ref == "" {
			ref = jsonStr(mm, "web_url")
		}
		fmt.Printf("!%s [%s] %s\n", jsonStr(mm, "iid"), jsonStr(mm, "state"), jsonStr(mm, "title"))
		fmt.Printf("  Project: %s  Author: %s\n", ref, strOr(jsonStr(author, "username"), "unknown"))
		fmt.Printf("  Updated: %s  Target: %s\n\n", jsonStr(mm, "updated_at"), jsonStr(mm, "target_branch"))
	}
}

func cmdProjectMRs(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: project-mrs <project-id-or-path> [state] [limit]")
	}
	encoded := url.PathEscape(args[0])
	state := "opened"
	limit := "25"
	if len(args) > 1 {
		state = args[1]
	}
	if len(args) > 2 {
		limit = args[2]
	}
	data, err := c.get("/projects/"+encoded+"/merge_requests", url.Values{
		"state": {state}, "per_page": {limit}, "order_by": {"updated_at"}, "sort": {"desc"},
	})
	if err != nil {
		die("%s", err)
	}
	var mrs []any
	json.Unmarshal(data, &mrs)
	for _, mr := range mrs {
		mm := asMap(mr)
		if mm == nil {
			continue
		}
		author := jsonMap(mm, "author")
		fmt.Printf("!%s [%s] %s\n", jsonStr(mm, "iid"), jsonStr(mm, "state"), jsonStr(mm, "title"))
		fmt.Printf("  Author: %s  Updated: %s\n",
			strOr(jsonStr(author, "username"), "unknown"), jsonStr(mm, "updated_at"))
		fmt.Printf("  Source: %s -> %s  Approvals: %s\n\n",
			jsonStr(mm, "source_branch"), jsonStr(mm, "target_branch"),
			strOr(jsonStr(mm, "upvotes"), "0"))
	}
}

func cmdMR(c *apiClient, args []string) {
	if len(args) < 2 {
		die("Usage: mr <project-id-or-path> <mr-iid>")
	}
	encoded := url.PathEscape(args[0])
	mrIID := args[1]
	data, err := c.get("/projects/"+encoded+"/merge_requests/"+mrIID, nil)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	author := jsonMap(m, "author")
	mergedBy := jsonMap(m, "merged_by")

	var assigneeNames []string
	for _, a := range jsonArr(m, "assignees") {
		if am := asMap(a); am != nil {
			assigneeNames = append(assigneeNames, jsonStr(am, "username"))
		}
	}
	var reviewerNames []string
	for _, r := range jsonArr(m, "reviewers") {
		if rm := asMap(r); rm != nil {
			reviewerNames = append(reviewerNames, jsonStr(rm, "username"))
		}
	}
	milestone := jsonMap(m, "milestone")

	out := map[string]any{
		"iid":           m["iid"],
		"title":         jsonStr(m, "title"),
		"state":         jsonStr(m, "state"),
		"author":        jsonStr(author, "username"),
		"assignees":     assigneeNames,
		"reviewers":     reviewerNames,
		"source_branch": jsonStr(m, "source_branch"),
		"target_branch": jsonStr(m, "target_branch"),
		"created":       jsonStr(m, "created_at"),
		"updated":       jsonStr(m, "updated_at"),
		"merged_by":     nil,
		"merged_at":     nil,
		"labels":        m["labels"],
		"milestone":     nil,
		"draft":         m["draft"],
		"merge_status":  strOr(jsonStr(m, "merge_status"), "unknown"),
		"has_conflicts": m["has_conflicts"],
		"changes_count": strOr(jsonStr(m, "changes_count"), "unknown"),
		"web_url":       jsonStr(m, "web_url"),
		"description":   strOr(jsonStr(m, "description"), "none"),
	}
	if mergedBy != nil {
		out["merged_by"] = jsonStr(mergedBy, "username")
	}
	if jsonStr(m, "merged_at") != "" {
		out["merged_at"] = jsonStr(m, "merged_at")
	}
	if milestone != nil {
		out["milestone"] = jsonStr(milestone, "title")
	}
	printJSON(out)
}

func cmdMRChanges(c *apiClient, args []string) {
	if len(args) < 2 {
		die("Usage: mr-changes <project-id-or-path> <mr-iid>")
	}
	encoded := url.PathEscape(args[0])
	mrIID := args[1]
	data, err := c.get("/projects/"+encoded+"/merge_requests/"+mrIID+"/changes", nil)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	for _, change := range jsonArr(m, "changes") {
		cm := asMap(change)
		if cm == nil {
			continue
		}
		status := "modified"
		if jsonStr(cm, "new_file") == "true" {
			status = "added"
		} else if jsonStr(cm, "deleted_file") == "true" {
			status = "deleted"
		} else if jsonStr(cm, "renamed_file") == "true" {
			status = fmt.Sprintf("renamed from %s", jsonStr(cm, "old_path"))
		}
		fmt.Printf("%s (%s)\n\n", jsonStr(cm, "new_path"), status)
	}
}

func cmdMyIssues(c *apiClient, args []string) {
	state := "opened"
	limit := "25"
	if len(args) > 0 {
		state = args[0]
	}
	if len(args) > 1 {
		limit = args[1]
	}
	data, err := c.get("/issues", url.Values{
		"state": {state}, "scope": {"assigned_to_me"}, "per_page": {limit},
		"order_by": {"updated_at"}, "sort": {"desc"},
	})
	if err != nil {
		die("%s", err)
	}
	var issues []any
	json.Unmarshal(data, &issues)
	fmt.Printf("Issues assigned to you (%s, %d shown):\n\n", state, len(issues))
	for _, issue := range issues {
		im := asMap(issue)
		if im == nil {
			continue
		}
		refs := jsonMap(im, "references")
		labels := toStringSlice(jsonArr(im, "labels"))
		labelStr := "none"
		if len(labels) > 0 {
			labelStr = strings.Join(labels, ", ")
		}
		fmt.Printf("#%s [%s] %s\n", jsonStr(im, "iid"), jsonStr(im, "state"), jsonStr(im, "title"))
		fmt.Printf("  Project: %s  Labels: %s\n",
			strOr(jsonStr(refs, "full"), "unknown"), labelStr)
		fmt.Printf("  Updated: %s\n\n", jsonStr(im, "updated_at"))
	}
}

func cmdProjectIssues(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: project-issues <project-id-or-path> [state] [limit]")
	}
	encoded := url.PathEscape(args[0])
	state := "opened"
	limit := "25"
	if len(args) > 1 {
		state = args[1]
	}
	if len(args) > 2 {
		limit = args[2]
	}
	data, err := c.get("/projects/"+encoded+"/issues", url.Values{
		"state": {state}, "per_page": {limit}, "order_by": {"updated_at"}, "sort": {"desc"},
	})
	if err != nil {
		die("%s", err)
	}
	var issues []any
	json.Unmarshal(data, &issues)
	for _, issue := range issues {
		im := asMap(issue)
		if im == nil {
			continue
		}
		author := jsonMap(im, "author")
		assignee := jsonMap(im, "assignee")
		labels := toStringSlice(jsonArr(im, "labels"))
		labelStr := "none"
		if len(labels) > 0 {
			labelStr = strings.Join(labels, ", ")
		}
		fmt.Printf("#%s [%s] %s\n", jsonStr(im, "iid"), jsonStr(im, "state"), jsonStr(im, "title"))
		fmt.Printf("  Author: %s  Assignee: %s  Labels: %s\n",
			strOr(jsonStr(author, "username"), "unknown"),
			strOr(jsonStr(assignee, "username"), "unassigned"),
			labelStr)
		fmt.Printf("  Updated: %s\n\n", jsonStr(im, "updated_at"))
	}
}

func cmdIssue(c *apiClient, args []string) {
	if len(args) < 2 {
		die("Usage: issue <project-id-or-path> <issue-iid>")
	}
	encoded := url.PathEscape(args[0])
	issueIID := args[1]
	data, err := c.get("/projects/"+encoded+"/issues/"+issueIID, nil)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	author := jsonMap(m, "author")
	milestone := jsonMap(m, "milestone")

	var assigneeNames []string
	for _, a := range jsonArr(m, "assignees") {
		if am := asMap(a); am != nil {
			assigneeNames = append(assigneeNames, jsonStr(am, "username"))
		}
	}

	out := map[string]any{
		"iid":         m["iid"],
		"title":       jsonStr(m, "title"),
		"state":       jsonStr(m, "state"),
		"author":      jsonStr(author, "username"),
		"assignees":   assigneeNames,
		"labels":      m["labels"],
		"milestone":   nil,
		"created":     jsonStr(m, "created_at"),
		"updated":     jsonStr(m, "updated_at"),
		"closed_at":   m["closed_at"],
		"due_date":    m["due_date"],
		"weight":      m["weight"],
		"web_url":     jsonStr(m, "web_url"),
		"description": strOr(jsonStr(m, "description"), "none"),
	}
	if milestone != nil {
		out["milestone"] = jsonStr(milestone, "title")
	}
	printJSON(out)
}

func cmdPipelines(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: pipelines <project-id-or-path> [limit]")
	}
	encoded := url.PathEscape(args[0])
	limit := "15"
	if len(args) > 1 {
		limit = args[1]
	}
	data, err := c.get("/projects/"+encoded+"/pipelines", url.Values{
		"per_page": {limit}, "order_by": {"updated_at"}, "sort": {"desc"},
	})
	if err != nil {
		die("%s", err)
	}
	var pipelines []any
	json.Unmarshal(data, &pipelines)
	for _, p := range pipelines {
		pm := asMap(p)
		if pm == nil {
			continue
		}
		fmt.Printf("[#%s] %s  Ref: %s  Source: %s\n",
			jsonStr(pm, "id"), jsonStr(pm, "status"),
			jsonStr(pm, "ref"), strOr(jsonStr(pm, "source"), "unknown"))
		fmt.Printf("  Created: %s  Updated: %s\n",
			jsonStr(pm, "created_at"), jsonStr(pm, "updated_at"))
		fmt.Printf("  URL: %s\n\n", jsonStr(pm, "web_url"))
	}
}

func cmdPipeline(c *apiClient, args []string) {
	if len(args) < 2 {
		die("Usage: pipeline <project-id-or-path> <pipeline-id>")
	}
	encoded := url.PathEscape(args[0])
	pipelineID := args[1]
	data, err := c.get("/projects/"+encoded+"/pipelines/"+pipelineID, nil)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	user := jsonMap(m, "user")
	out := map[string]any{
		"id":       m["id"],
		"status":   jsonStr(m, "status"),
		"ref":      jsonStr(m, "ref"),
		"sha":      jsonStr(m, "sha"),
		"source":   strOr(jsonStr(m, "source"), "unknown"),
		"created":  jsonStr(m, "created_at"),
		"updated":  jsonStr(m, "updated_at"),
		"started":  m["started_at"],
		"finished": m["finished_at"],
		"duration": m["duration"],
		"user":     jsonStr(user, "username"),
		"web_url":  jsonStr(m, "web_url"),
	}
	printJSON(out)

	fmt.Println()
	fmt.Println("Jobs:")
	jobs, err := c.get("/projects/"+encoded+"/pipelines/"+pipelineID+"/jobs", url.Values{"per_page": {"50"}})
	if err != nil {
		die("%s", err)
	}
	var jobList []any
	json.Unmarshal(jobs, &jobList)
	for _, j := range jobList {
		jm := asMap(j)
		if jm == nil {
			continue
		}
		runner := jsonMap(jm, "runner")
		runnerDesc := "N/A"
		if runner != nil {
			runnerDesc = strOr(jsonStr(runner, "description"), "N/A")
		}
		fmt.Printf("  [%s] %s  Stage: %s  Duration: %ss  Runner: %s\n",
			jsonStr(jm, "status"), jsonStr(jm, "name"),
			jsonStr(jm, "stage"),
			strOr(jsonStr(jm, "duration"), "0"),
			runnerDesc)
	}
}

func cmdBranches(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: branches <project-id-or-path> [limit]")
	}
	encoded := url.PathEscape(args[0])
	limit := "25"
	if len(args) > 1 {
		limit = args[1]
	}
	data, err := c.get("/projects/"+encoded+"/repository/branches", url.Values{
		"per_page": {limit}, "order_by": {"updated"}, "sort": {"desc"},
	})
	if err != nil {
		die("%s", err)
	}
	var branches []any
	json.Unmarshal(data, &branches)
	for _, b := range branches {
		bm := asMap(b)
		if bm == nil {
			continue
		}
		flags := ""
		if jsonStr(bm, "default") == "true" {
			flags += " [default]"
		}
		if jsonStr(bm, "protected") == "true" {
			flags += " [protected]"
		}
		commit := jsonMap(bm, "commit")
		fmt.Printf("%s%s\n", jsonStr(bm, "name"), flags)
		fmt.Printf("  Last commit: %s %s (%s, %s)\n\n",
			jsonStr(commit, "short_id"),
			strOr(jsonStr(commit, "title"), "no message"),
			strOr(jsonStr(commit, "author_name"), "unknown"),
			jsonStr(commit, "committed_date"))
	}
}

func cmdCommits(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: commits <project-id-or-path> [ref] [limit]")
	}
	encoded := url.PathEscape(args[0])
	ref := ""
	limit := "15"
	if len(args) > 1 {
		ref = args[1]
	}
	if len(args) > 2 {
		limit = args[2]
	}
	params := url.Values{"per_page": {limit}}
	if ref != "" {
		params.Set("ref_name", ref)
	}
	data, err := c.get("/projects/"+encoded+"/repository/commits", params)
	if err != nil {
		die("%s", err)
	}
	var commits []any
	json.Unmarshal(data, &commits)
	for _, co := range commits {
		cm := asMap(co)
		if cm == nil {
			continue
		}
		fmt.Printf("%s %s\n", jsonStr(cm, "short_id"), jsonStr(cm, "title"))
		fmt.Printf("  Author: %s  Date: %s\n\n", jsonStr(cm, "author_name"), jsonStr(cm, "committed_date"))
	}
}

func cmdTree(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: tree <project-id-or-path> [path] [ref]")
	}
	encoded := url.PathEscape(args[0])
	path := "."
	ref := ""
	if len(args) > 1 {
		path = args[1]
	}
	if len(args) > 2 {
		ref = args[2]
	}
	params := url.Values{"per_page": {"100"}, "recursive": {"false"}}
	if path != "." {
		params.Set("path", path)
	}
	if ref != "" {
		params.Set("ref", ref)
	}
	data, err := c.get("/projects/"+encoded+"/repository/tree", params)
	if err != nil {
		die("%s", err)
	}
	var items []any
	json.Unmarshal(data, &items)
	for _, item := range items {
		im := asMap(item)
		if im == nil {
			continue
		}
		if jsonStr(im, "type") == "tree" {
			fmt.Printf("📁 %s/\n", jsonStr(im, "name"))
		} else {
			fmt.Printf("   %s\n", jsonStr(im, "name"))
		}
	}
}

func cmdFile(c *apiClient, args []string) {
	if len(args) < 2 {
		die("Usage: file <project-id-or-path> <file-path> [ref]")
	}
	encodedProject := url.PathEscape(args[0])
	encodedFile := url.PathEscape(args[1])
	ref := ""
	if len(args) > 2 {
		ref = args[2]
	}
	params := url.Values{}
	if ref != "" {
		params.Set("ref", ref)
	}
	data, err := c.get("/projects/"+encodedProject+"/repository/files/"+encodedFile, params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	content := jsonStr(m, "content")
	encoding := strOr(jsonStr(m, "encoding"), "base64")
	lastCommit := jsonStr(m, "last_commit_id")
	if len(lastCommit) > 8 {
		lastCommit = lastCommit[:8]
	}

	fmt.Printf("File: %s\n", jsonStr(m, "file_path"))
	fmt.Printf("Size: %s bytes  Ref: %s\n", jsonStr(m, "size"), jsonStr(m, "ref"))
	fmt.Printf("Last commit: %s\n", lastCommit)
	fmt.Println("---")

	if encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			fmt.Println("[binary content]")
		} else {
			fmt.Print(string(decoded))
		}
	} else {
		fmt.Print(content)
	}
}

func cmdGroups(c *apiClient, args []string) {
	limit := "25"
	if len(args) > 0 {
		limit = args[0]
	}
	data, err := c.get("/groups", url.Values{
		"per_page": {limit}, "order_by": {"name"}, "sort": {"asc"}, "min_access_level": {"10"},
	})
	if err != nil {
		die("%s", err)
	}
	var groups []any
	json.Unmarshal(data, &groups)
	for _, g := range groups {
		gm := asMap(g)
		if gm == nil {
			continue
		}
		projectCount := len(jsonArr(gm, "projects"))
		subgroupCount := len(jsonArr(gm, "subgroups"))
		fmt.Printf("%s\n", jsonStr(gm, "full_path"))
		fmt.Printf("  Visibility: %s  Projects: %d  Subgroups: %d\n",
			jsonStr(gm, "visibility"), projectCount, subgroupCount)
		fmt.Printf("  URL: %s\n\n", jsonStr(gm, "web_url"))
	}
}

func cmdGroupProjects(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: group-projects <group-id-or-path> [limit]")
	}
	encoded := url.PathEscape(args[0])
	limit := "25"
	if len(args) > 1 {
		limit = args[1]
	}
	data, err := c.get("/groups/"+encoded+"/projects", url.Values{
		"per_page": {limit}, "order_by": {"updated_at"}, "sort": {"desc"}, "include_subgroups": {"true"},
	})
	if err != nil {
		die("%s", err)
	}
	var projects []any
	json.Unmarshal(data, &projects)
	for _, p := range projects {
		pm := asMap(p)
		if pm == nil {
			continue
		}
		fmt.Printf("%s\n", jsonStr(pm, "path_with_namespace"))
		fmt.Printf("  Updated: %s  Visibility: %s  Stars: %s\n\n",
			jsonStr(pm, "last_activity_at"),
			jsonStr(pm, "visibility"),
			strOr(jsonStr(pm, "star_count"), "0"))
	}
}

func cmdSearch(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: search <query> [scope: projects|issues|merge_requests|milestones|blobs]")
	}
	query := args[0]
	scope := "projects"
	limit := "20"
	if len(args) > 1 {
		scope = args[1]
	}
	if len(args) > 2 {
		limit = args[2]
	}
	data, err := c.get("/search", url.Values{
		"search": {query}, "scope": {scope}, "per_page": {limit},
	})
	if err != nil {
		die("%s", err)
	}
	var results []any
	json.Unmarshal(data, &results)

	switch scope {
	case "projects":
		for _, r := range results {
			rm := asMap(r)
			if rm == nil {
				continue
			}
			desc := strOr(jsonStr(rm, "description"), "none")
			if len(desc) > 120 {
				desc = desc[:120]
			}
			fmt.Printf("%s\n", jsonStr(rm, "path_with_namespace"))
			fmt.Printf("  Description: %s\n", desc)
			fmt.Printf("  Updated: %s\n\n", jsonStr(rm, "last_activity_at"))
		}
	case "issues":
		for _, r := range results {
			rm := asMap(r)
			if rm == nil {
				continue
			}
			refs := jsonMap(rm, "references")
			fmt.Printf("#%s [%s] %s\n", jsonStr(rm, "iid"), jsonStr(rm, "state"), jsonStr(rm, "title"))
			fmt.Printf("  Project: %s  Updated: %s\n\n",
				strOr(jsonStr(refs, "full"), "unknown"), jsonStr(rm, "updated_at"))
		}
	case "merge_requests":
		for _, r := range results {
			rm := asMap(r)
			if rm == nil {
				continue
			}
			refs := jsonMap(rm, "references")
			fmt.Printf("!%s [%s] %s\n", jsonStr(rm, "iid"), jsonStr(rm, "state"), jsonStr(rm, "title"))
			fmt.Printf("  Project: %s  Updated: %s\n\n",
				strOr(jsonStr(refs, "full"), "unknown"), jsonStr(rm, "updated_at"))
		}
	case "blobs":
		for _, r := range results {
			rm := asMap(r)
			if rm == nil {
				continue
			}
			fmt.Printf("%s (project: %s)\n", jsonStr(rm, "path"), jsonStr(rm, "project_id"))
			fmt.Printf("  Ref: %s  Filename: %s\n\n", jsonStr(rm, "ref"), jsonStr(rm, "filename"))
		}
	default:
		printJSON(results)
	}
}

func cmdProjectSearch(c *apiClient, args []string) {
	if len(args) < 2 {
		die("Usage: project-search <project-id-or-path> <query> [scope: blobs|commits|issues|merge_requests]")
	}
	encoded := url.PathEscape(args[0])
	query := args[1]
	scope := "blobs"
	if len(args) > 2 {
		scope = args[2]
	}
	data, err := c.get("/projects/"+encoded+"/search", url.Values{
		"search": {query}, "scope": {scope}, "per_page": {"20"},
	})
	if err != nil {
		die("%s", err)
	}
	var results []any
	json.Unmarshal(data, &results)

	switch scope {
	case "blobs":
		for _, r := range results {
			rm := asMap(r)
			if rm == nil {
				continue
			}
			d := strings.ReplaceAll(jsonStr(rm, "data"), "\n", " ")
			if len(d) > 200 {
				d = d[:200]
			}
			fmt.Printf("%s:%s\n", jsonStr(rm, "path"), jsonStr(rm, "startline"))
			fmt.Printf("  %s\n\n", d)
		}
	case "commits":
		for _, r := range results {
			rm := asMap(r)
			if rm == nil {
				continue
			}
			fmt.Printf("%s %s\n", jsonStr(rm, "short_id"), jsonStr(rm, "title"))
			fmt.Printf("  Author: %s  Date: %s\n\n", jsonStr(rm, "author_name"), jsonStr(rm, "committed_date"))
		}
	default:
		printJSON(results)
	}
}

func cmdRegistries(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: registries <project-id-or-path>")
	}
	encoded := url.PathEscape(args[0])
	data, err := c.get("/projects/"+encoded+"/registry/repositories", url.Values{"per_page": {"50"}})
	if err != nil {
		die("%s", err)
	}
	var repos []any
	json.Unmarshal(data, &repos)
	for _, r := range repos {
		rm := asMap(r)
		if rm == nil {
			continue
		}
		fmt.Printf("[%s] %s\n", jsonStr(rm, "id"), jsonStr(rm, "path"))
		fmt.Printf("  Tags: %s  Created: %s\n\n",
			strOr(jsonStr(rm, "tags_count"), "0"), jsonStr(rm, "created_at"))
	}
}

// ── Help ────────────────────────────────────────────────────

func printHelp() {
	fmt.Println(`Usage: gitlab-navigator <host> <command> [args...]

Discovery:
  discover [substring]                             Find GitLab hosts in ~/.netrc

Query commands (host = hostname or unique substring from ~/.netrc):
  <host> whoami                                    Current user
  <host> test                                      Test connection

Activity & starred:
  <host> starred [limit]                           Starred projects
  <host> starred-activity [days] [limit]           Starred projects with recent activity
  <host> events [limit]                            Your recent activity feed
  <host> project-events <project> [limit]          Project activity

Projects:
  <host> projects [limit]                          Your projects (by membership)
  <host> project-info <project>                    Project details + statistics
  <host> create-project <name> [path] [visibility] [namespace-id]
                                                    Create a new project

Merge requests:
  <host> my-mrs [state] [limit]                    MRs assigned to you
  <host> mr-review [state] [limit]                 MRs awaiting your review
  <host> project-mrs <project> [state] [limit]     MRs in a project
  <host> mr <project> <iid>                        MR details
  <host> mr-changes <project> <iid>                MR changed files

Issues:
  <host> my-issues [state] [limit]                 Issues assigned to you
  <host> project-issues <project> [state] [limit]  Issues in a project
  <host> issue <project> <iid>                     Issue details

Pipelines:
  <host> pipelines <project> [limit]               Recent pipelines
  <host> pipeline <project> <id>                   Pipeline details + jobs

Code:
  <host> branches <project> [limit]                List branches
  <host> commits <project> [ref] [limit]           Recent commits
  <host> tree <project> [path] [ref]               Directory listing
  <host> file <project> <path> [ref]               Read file content

Groups:
  <host> groups [limit]                            Your groups
  <host> group-projects <group> [limit]            Projects in a group

Search:
  <host> search <query> [scope] [limit]            Global search
  <host> project-search <project> <query> [scope]  Project-scoped search

Registry:
  <host> registries <project>                      Container registry repos

Search scopes: projects, issues, merge_requests, milestones, blobs
Project refs: use ID (numeric) or URL-encoded path (group%2Fproject)`)
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

	hostname, entry, err := resolveHost(host, "gitlab")
	if err != nil {
		die("%s", err)
	}
	client := newClient(hostname, entry)

	switch command {
	case "whoami":
		cmdWhoami(client)
	case "test":
		cmdTest(client)
	case "starred":
		cmdStarred(client, cmdArgs)
	case "starred-activity":
		cmdStarredActivity(client, cmdArgs)
	case "events":
		cmdEvents(client, cmdArgs)
	case "project-events":
		cmdProjectEvents(client, cmdArgs)
	case "projects":
		cmdProjects(client, cmdArgs)
	case "project-info":
		cmdProjectInfo(client, cmdArgs)
	case "create-project":
		cmdCreateProject(client, cmdArgs)
	case "my-mrs":
		cmdMyMRs(client, cmdArgs)
	case "mr-review":
		cmdMRReview(client, cmdArgs)
	case "project-mrs":
		cmdProjectMRs(client, cmdArgs)
	case "mr":
		cmdMR(client, cmdArgs)
	case "mr-changes":
		cmdMRChanges(client, cmdArgs)
	case "my-issues":
		cmdMyIssues(client, cmdArgs)
	case "project-issues":
		cmdProjectIssues(client, cmdArgs)
	case "issue":
		cmdIssue(client, cmdArgs)
	case "pipelines":
		cmdPipelines(client, cmdArgs)
	case "pipeline":
		cmdPipeline(client, cmdArgs)
	case "branches":
		cmdBranches(client, cmdArgs)
	case "commits":
		cmdCommits(client, cmdArgs)
	case "tree":
		cmdTree(client, cmdArgs)
	case "file":
		cmdFile(client, cmdArgs)
	case "groups":
		cmdGroups(client, cmdArgs)
	case "group-projects":
		cmdGroupProjects(client, cmdArgs)
	case "search":
		cmdSearch(client, cmdArgs)
	case "project-search":
		cmdProjectSearch(client, cmdArgs)
	case "registries":
		cmdRegistries(client, cmdArgs)
	case "help":
		printHelp()
	default:
		die("Unknown command: %s (run with 'help')", command)
	}
}
