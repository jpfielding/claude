package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
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
		return "", netrcEntry{}, fmt.Errorf("no ~/.netrc machine entries matching '%s' (filtered for jira hosts)", input)
	}
	if len(matches) > 1 {
		msg := fmt.Sprintf("Multiple ~/.netrc entries match '%s':\n", input)
		for _, m := range matches {
			msg += fmt.Sprintf("  %s\n", m.Machine)
		}
		return "", netrcEntry{}, fmt.Errorf("%sambiguous host substring '%s'", msg, input)
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

func (c *apiClient) doGet(fullURL string) (json.RawMessage, error) {
	req, err := http.NewRequest("GET", fullURL, nil)
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

func (c *apiClient) get(endpoint string, params url.Values) (json.RawMessage, error) {
	u := c.baseURL + "/rest/api/2" + endpoint
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	return c.doGet(u)
}

func (c *apiClient) getAgile(endpoint string, params url.Values) (json.RawMessage, error) {
	u := c.baseURL + "/rest/agile/1.0" + endpoint
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	return c.doGet(u)
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

func joinNames(arr []any, key string) string {
	var names []string
	for _, item := range arr {
		if m := asMap(item); m != nil {
			if n := jsonStr(m, key); n != "" {
				names = append(names, n)
			}
		}
	}
	return strings.Join(names, ", ")
}

// ── Commands ────────────────────────────────────────────────

func cmdDiscover(args []string) {
	filter := "jira"
	if len(args) > 0 {
		filter = args[0]
	}
	hosts := discoverHosts(filter)
	if len(hosts) == 0 {
		fmt.Printf("No ~/.netrc entries matching '%s' found.\n\n", filter)
		fmt.Println("To search with a different substring: jira-navigator discover <substring>")
		return
	}
	fmt.Printf("Found ~/.netrc entries matching '%s':\n\n", filter)
	for i, h := range hosts {
		fmt.Printf("  [%d] %s\n", i+1, h)
	}
	fmt.Println()
	fmt.Println("Usage: jira-navigator <hostname-or-substring> <command>")
}

func cmdWhoami(c *apiClient) {
	data, err := c.get("/myself", nil)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	out := map[string]any{
		"name":         jsonStr(m, "name"),
		"displayName":  jsonStr(m, "displayName"),
		"emailAddress": jsonStr(m, "emailAddress"),
		"key":          jsonStr(m, "key"),
		"timeZone":     jsonStr(m, "timeZone"),
	}
	printJSON(out)
}

func cmdTest(c *apiClient) {
	fmt.Println("Testing connection...")
	data, err := c.get("/myself", nil)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	name := strOr(jsonStr(m, "displayName"), strOr(jsonStr(m, "name"), "unknown"))
	fmt.Printf("Connected as: %s\n\n", name)

	fmt.Println("Testing project listing...")
	data, err = c.get("/project", url.Values{"maxResults": {"3"}})
	if err != nil {
		die("%s", err)
	}
	var projects []any
	json.Unmarshal(data, &projects)
	for _, p := range projects {
		pm := asMap(p)
		if pm != nil {
			fmt.Printf("  %s - %s\n", jsonStr(pm, "key"), jsonStr(pm, "name"))
		}
	}
	fmt.Println()
	fmt.Println("Connection OK.")
}

func printIssueList(m map[string]any) {
	for _, issue := range jsonArr(m, "issues") {
		im := asMap(issue)
		if im == nil {
			continue
		}
		fields := jsonMap(im, "fields")
		project := jsonMap(fields, "project")
		status := jsonMap(fields, "status")
		priority := jsonMap(fields, "priority")
		issuetype := jsonMap(fields, "issuetype")
		assignee := jsonMap(fields, "assignee")
		fmt.Printf("[%s] %s: %s\n", jsonStr(project, "key"), jsonStr(im, "key"), jsonStr(fields, "summary"))
		fmt.Printf("  Status: %s  Priority: %s  Type: %s\n",
			jsonStr(status, "name"),
			strOr(jsonStr(priority, "name"), "None"),
			jsonStr(issuetype, "name"))
		fmt.Printf("  Assignee: %s  Updated: %s\n\n",
			strOr(jsonStr(assignee, "displayName"), "Unassigned"),
			jsonStr(fields, "updated"))
	}
}

func cmdRecent(c *apiClient, args []string) {
	limit := "20"
	if len(args) > 0 {
		limit = args[0]
	}
	jql := "ORDER BY updated DESC"
	params := url.Values{
		"jql":        {jql},
		"maxResults": {limit},
		"fields":     {"summary,status,assignee,updated,priority,issuetype,project"},
	}
	data, err := c.get("/search", params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	printIssueList(m)
}

func cmdMyIssues(c *apiClient, args []string) {
	limit := "25"
	if len(args) > 0 {
		limit = args[0]
	}
	jql := "assignee = currentUser() AND resolution = Unresolved ORDER BY updated DESC"
	params := url.Values{
		"jql":        {jql},
		"maxResults": {limit},
		"fields":     {"summary,status,priority,issuetype,project,updated"},
	}
	data, err := c.get("/search", params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	fmt.Printf("Open issues assigned to you (%s total):\n\n", jsonStr(m, "total"))
	printIssueList(m)
}

func cmdWatched(c *apiClient, args []string) {
	limit := "25"
	if len(args) > 0 {
		limit = args[0]
	}
	jql := "watcher = currentUser() AND resolution = Unresolved ORDER BY updated DESC"
	params := url.Values{
		"jql":        {jql},
		"maxResults": {limit},
		"fields":     {"summary,status,assignee,priority,issuetype,project,updated"},
	}
	data, err := c.get("/search", params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	fmt.Printf("Unresolved watched issues (%s total):\n\n", jsonStr(m, "total"))
	printIssueList(m)
}

func cmdWatchChanges(c *apiClient, args []string) {
	days := "7"
	if len(args) > 0 {
		days = args[0]
	}
	jql := fmt.Sprintf("watcher = currentUser() AND updated >= -%sd ORDER BY updated DESC", days)
	params := url.Values{
		"jql":        {jql},
		"maxResults": {"50"},
		"fields":     {"summary,status,assignee,priority,issuetype,project,updated"},
	}
	data, err := c.get("/search", params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	fmt.Printf("Watched issues updated in last %s days (%s results):\n\n", days, jsonStr(m, "total"))
	printIssueList(m)
}

func cmdSearch(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: search <JQL query> [limit]")
	}
	jql := args[0]
	limit := "20"
	if len(args) > 1 {
		limit = args[1]
	}
	params := url.Values{
		"jql":        {jql},
		"maxResults": {limit},
		"fields":     {"summary,status,assignee,priority,issuetype,project,updated"},
	}
	data, err := c.get("/search", params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	fmt.Printf("Results (%s total):\n\n", jsonStr(m, "total"))
	printIssueList(m)
}

func cmdIssue(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: issue <issue-key>")
	}
	issueKey := args[0]
	params := url.Values{"expand": {"renderedFields,names,changelog"}}
	data, err := c.get("/issue/"+issueKey, params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	fields := jsonMap(m, "fields")
	rendered := jsonMap(m, "renderedFields")
	project := jsonMap(fields, "project")
	status := jsonMap(fields, "status")
	priority := jsonMap(fields, "priority")
	issuetype := jsonMap(fields, "issuetype")
	assignee := jsonMap(fields, "assignee")
	reporter := jsonMap(fields, "reporter")
	resolution := jsonMap(fields, "resolution")

	description := ""
	if rendered != nil {
		description = jsonStr(rendered, "description")
	}
	if description == "" {
		description = jsonStr(fields, "description")
	}
	if description == "" {
		description = "No description"
	}

	fmt.Printf("Key: %s\n", jsonStr(m, "key"))
	fmt.Printf("Summary: %s\n", jsonStr(fields, "summary"))
	fmt.Printf("Type: %s\n", jsonStr(issuetype, "name"))
	fmt.Printf("Status: %s\n", jsonStr(status, "name"))
	fmt.Printf("Priority: %s\n", strOr(jsonStr(priority, "name"), "None"))
	fmt.Printf("Project: %s - %s\n", jsonStr(project, "key"), jsonStr(project, "name"))
	fmt.Printf("Assignee: %s\n", strOr(jsonStr(assignee, "displayName"), "Unassigned"))
	fmt.Printf("Reporter: %s\n", strOr(jsonStr(reporter, "displayName"), "Unknown"))
	fmt.Printf("Created: %s\n", jsonStr(fields, "created"))
	fmt.Printf("Updated: %s\n", jsonStr(fields, "updated"))
	fmt.Printf("Resolution: %s\n", strOr(jsonStr(resolution, "name"), "Unresolved"))
	fmt.Printf("Labels: %s\n", strings.Join(toStringSlice(jsonArr(fields, "labels")), ", "))
	fmt.Printf("Components: %s\n", joinNames(jsonArr(fields, "components"), "name"))
	fmt.Printf("Fix Versions: %s\n", joinNames(jsonArr(fields, "fixVersions"), "name"))
	fmt.Printf("\n--- Description ---\n%s\n", description)
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

func cmdIssueInfo(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: issue-info <issue-key>")
	}
	issueKey := args[0]
	params := url.Values{
		"fields": {"summary,status,priority,issuetype,project,assignee,reporter,created,updated,resolution,labels,components,fixVersions,subtasks,issuelinks,parent"},
	}
	data, err := c.get("/issue/"+issueKey, params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	fields := jsonMap(m, "fields")
	project := jsonMap(fields, "project")
	status := jsonMap(fields, "status")
	priority := jsonMap(fields, "priority")
	issuetype := jsonMap(fields, "issuetype")
	assignee := jsonMap(fields, "assignee")
	reporter := jsonMap(fields, "reporter")
	resolution := jsonMap(fields, "resolution")
	parent := jsonMap(fields, "parent")

	var parentOut any
	if parent != nil {
		pf := jsonMap(parent, "fields")
		parentOut = map[string]any{
			"key":     jsonStr(parent, "key"),
			"summary": jsonStr(pf, "summary"),
		}
	}

	var subtasks []any
	for _, st := range jsonArr(fields, "subtasks") {
		stm := asMap(st)
		if stm == nil {
			continue
		}
		stf := jsonMap(stm, "fields")
		sts := jsonMap(stf, "status")
		subtasks = append(subtasks, map[string]any{
			"key":     jsonStr(stm, "key"),
			"summary": jsonStr(stf, "summary"),
			"status":  jsonStr(sts, "name"),
		})
	}

	var links []any
	for _, link := range jsonArr(fields, "issuelinks") {
		lm := asMap(link)
		if lm == nil {
			continue
		}
		lt := jsonMap(lm, "type")
		direction := "outward"
		linkedIssue := jsonMap(lm, "outwardIssue")
		if linkedIssue == nil {
			direction = "inward"
			linkedIssue = jsonMap(lm, "inwardIssue")
		}
		if linkedIssue == nil {
			continue
		}
		lif := jsonMap(linkedIssue, "fields")
		lis := jsonMap(lif, "status")
		links = append(links, map[string]any{
			"type":      jsonStr(lt, "name"),
			"direction": direction,
			"issue": map[string]any{
				"key":     jsonStr(linkedIssue, "key"),
				"summary": jsonStr(lif, "summary"),
				"status":  jsonStr(lis, "name"),
			},
		})
	}

	var compNames []string
	for _, c := range jsonArr(fields, "components") {
		if cm := asMap(c); cm != nil {
			compNames = append(compNames, jsonStr(cm, "name"))
		}
	}

	var fvNames []string
	for _, fv := range jsonArr(fields, "fixVersions") {
		if fvm := asMap(fv); fvm != nil {
			fvNames = append(fvNames, jsonStr(fvm, "name"))
		}
	}

	out := map[string]any{
		"key":         jsonStr(m, "key"),
		"summary":     jsonStr(fields, "summary"),
		"type":        jsonStr(issuetype, "name"),
		"status":      jsonStr(status, "name"),
		"priority":    strOr(jsonStr(priority, "name"), "None"),
		"project":     map[string]any{"key": jsonStr(project, "key"), "name": jsonStr(project, "name")},
		"assignee":    strOr(jsonStr(assignee, "displayName"), "Unassigned"),
		"reporter":    strOr(jsonStr(reporter, "displayName"), "Unknown"),
		"created":     jsonStr(fields, "created"),
		"updated":     jsonStr(fields, "updated"),
		"resolution":  strOr(jsonStr(resolution, "name"), "Unresolved"),
		"labels":      toStringSlice(jsonArr(fields, "labels")),
		"components":  compNames,
		"fixVersions": fvNames,
		"parent":      parentOut,
		"subtasks":    subtasks,
		"links":       links,
	}
	printJSON(out)
}

func cmdComments(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: comments <issue-key> [limit]")
	}
	issueKey := args[0]
	limit := "20"
	if len(args) > 1 {
		limit = args[1]
	}
	params := url.Values{
		"maxResults": {limit},
		"orderBy":    {"-created"},
	}
	data, err := c.get("/issue/"+issueKey+"/comment", params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	for _, comment := range jsonArr(m, "comments") {
		cm := asMap(comment)
		if cm == nil {
			continue
		}
		author := jsonMap(cm, "author")
		fmt.Printf("%s (%s):\n%s\n---\n\n",
			strOr(jsonStr(author, "displayName"), "unknown"),
			jsonStr(cm, "created"),
			jsonStr(cm, "body"))
	}
}

func cmdTransitions(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: transitions <issue-key>")
	}
	issueKey := args[0]
	data, err := c.get("/issue/"+issueKey+"/transitions", nil)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	fmt.Printf("Available transitions for %s:\n\n", issueKey)
	for _, t := range jsonArr(m, "transitions") {
		tm := asMap(t)
		if tm == nil {
			continue
		}
		to := jsonMap(tm, "to")
		fmt.Printf("  [%s] %s -> %s\n", jsonStr(tm, "id"), jsonStr(tm, "name"), jsonStr(to, "name"))
	}
}

func cmdChangelog(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: changelog <issue-key> [limit]")
	}
	issueKey := args[0]
	limit := 10
	if len(args) > 1 {
		limit, _ = strconv.Atoi(args[1])
		if limit <= 0 {
			limit = 10
		}
	}
	params := url.Values{
		"expand": {"changelog"},
		"fields": {"summary"},
	}
	data, err := c.get("/issue/"+issueKey, params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	fields := jsonMap(m, "fields")
	fmt.Printf("Changelog for %s: %s\n\n", issueKey, jsonStr(fields, "summary"))

	changelog := jsonMap(m, "changelog")
	histories := jsonArr(changelog, "histories")

	// Sort by created descending
	sort.Slice(histories, func(i, j int) bool {
		hi := asMap(histories[i])
		hj := asMap(histories[j])
		return jsonStr(hi, "created") > jsonStr(hj, "created")
	})

	if len(histories) > limit {
		histories = histories[:limit]
	}

	for _, h := range histories {
		hm := asMap(h)
		if hm == nil {
			continue
		}
		author := jsonMap(hm, "author")
		fmt.Printf("%s (%s):\n", strOr(jsonStr(author, "displayName"), "unknown"), jsonStr(hm, "created"))
		for _, item := range jsonArr(hm, "items") {
			im := asMap(item)
			if im == nil {
				continue
			}
			fmt.Printf("  %s: %s -> %s\n",
				jsonStr(im, "field"),
				strOr(jsonStr(im, "fromString"), "none"),
				strOr(jsonStr(im, "toString"), "none"))
		}
		fmt.Println()
	}
}

func cmdProjects(c *apiClient) {
	data, err := c.get("/project", nil)
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
		lead := jsonMap(pm, "lead")
		fmt.Printf("%s - %s\n", jsonStr(pm, "key"), jsonStr(pm, "name"))
		fmt.Printf("  Lead: %s  Type: %s\n\n",
			strOr(jsonStr(lead, "displayName"), "Unknown"),
			strOr(jsonStr(pm, "projectTypeKey"), "unknown"))
	}
}

func cmdProjectInfo(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: project-info <project-key>")
	}
	projectKey := args[0]
	params := url.Values{"expand": {"description,lead,issueTypes,components,versions"}}
	data, err := c.get("/project/"+projectKey, params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	lead := jsonMap(m, "lead")

	var issueTypeNames []string
	for _, it := range jsonArr(m, "issueTypes") {
		if itm := asMap(it); itm != nil {
			issueTypeNames = append(issueTypeNames, jsonStr(itm, "name"))
		}
	}

	var components []any
	for _, comp := range jsonArr(m, "components") {
		if cm := asMap(comp); cm != nil {
			cl := jsonMap(cm, "lead")
			entry := map[string]any{"name": jsonStr(cm, "name")}
			if cl != nil {
				entry["lead"] = jsonStr(cl, "displayName")
			}
			components = append(components, entry)
		}
	}

	var versions []any
	for _, ver := range jsonArr(m, "versions") {
		if vm := asMap(ver); vm != nil {
			entry := map[string]any{
				"name":        jsonStr(vm, "name"),
				"released":    vm["released"],
				"releaseDate": vm["releaseDate"],
			}
			versions = append(versions, entry)
		}
	}

	out := map[string]any{
		"key":         jsonStr(m, "key"),
		"name":        jsonStr(m, "name"),
		"description": strOr(jsonStr(m, "description"), "None"),
		"lead":        strOr(jsonStr(lead, "displayName"), "Unknown"),
		"projectType": strOr(jsonStr(m, "projectTypeKey"), "unknown"),
		"issueTypes":  issueTypeNames,
		"components":  components,
		"versions":    versions,
	}
	printJSON(out)
}

func cmdStatuses(c *apiClient, args []string) {
	if len(args) > 0 {
		projectKey := args[0]
		data, err := c.get("/project/"+projectKey+"/statuses", nil)
		if err != nil {
			die("%s", err)
		}
		var issueTypes []any
		json.Unmarshal(data, &issueTypes)
		for _, it := range issueTypes {
			itm := asMap(it)
			if itm == nil {
				continue
			}
			fmt.Printf("%s:\n", jsonStr(itm, "name"))
			for _, s := range jsonArr(itm, "statuses") {
				sm := asMap(s)
				if sm == nil {
					continue
				}
				cat := jsonMap(sm, "statusCategory")
				fmt.Printf("  [%s] %s (%s)\n", jsonStr(sm, "id"), jsonStr(sm, "name"), jsonStr(cat, "name"))
			}
			fmt.Println()
		}
	} else {
		data, err := c.get("/status", nil)
		if err != nil {
			die("%s", err)
		}
		var statuses []any
		json.Unmarshal(data, &statuses)
		for _, s := range statuses {
			sm := asMap(s)
			if sm == nil {
				continue
			}
			cat := jsonMap(sm, "statusCategory")
			fmt.Printf("[%s] %s (%s)\n", jsonStr(sm, "id"), jsonStr(sm, "name"), jsonStr(cat, "name"))
		}
	}
}

func cmdFilters(c *apiClient) {
	data, err := c.get("/filter/favourite", nil)
	if err != nil {
		die("%s", err)
	}
	var filters []any
	json.Unmarshal(data, &filters)
	fmt.Println("Favourite filters:")
	fmt.Println()
	for _, f := range filters {
		fm := asMap(f)
		if fm == nil {
			continue
		}
		owner := jsonMap(fm, "owner")
		fmt.Printf("[%s] %s\n", jsonStr(fm, "id"), jsonStr(fm, "name"))
		fmt.Printf("  JQL: %s\n", jsonStr(fm, "jql"))
		fmt.Printf("  Owner: %s\n\n", strOr(jsonStr(owner, "displayName"), "Unknown"))
	}
}

func cmdBoards(c *apiClient) {
	data, err := c.getAgile("/board", url.Values{"maxResults": {"50"}})
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	for _, b := range jsonArr(m, "values") {
		bm := asMap(b)
		if bm == nil {
			continue
		}
		location := jsonMap(bm, "location")
		projectKey := "N/A"
		if location != nil {
			projectKey = strOr(jsonStr(location, "projectKey"), "N/A")
		}
		fmt.Printf("[%s] %s\n", jsonStr(bm, "id"), jsonStr(bm, "name"))
		fmt.Printf("  Type: %s  Project: %s\n\n", jsonStr(bm, "type"), projectKey)
	}
}

func cmdSprints(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: sprints <board-id> [state: active|closed|future]")
	}
	boardID := args[0]
	state := "active"
	if len(args) > 1 {
		state = args[1]
	}
	params := url.Values{
		"state":      {state},
		"maxResults": {"10"},
	}
	data, err := c.getAgile("/board/"+boardID+"/sprint", params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	for _, s := range jsonArr(m, "values") {
		sm := asMap(s)
		if sm == nil {
			continue
		}
		fmt.Printf("[%s] %s\n", jsonStr(sm, "id"), jsonStr(sm, "name"))
		fmt.Printf("  State: %s  Start: %s  End: %s\n\n",
			jsonStr(sm, "state"),
			strOr(jsonStr(sm, "startDate"), "N/A"),
			strOr(jsonStr(sm, "endDate"), "N/A"))
	}
}

func cmdSprintIssues(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: sprint-issues <sprint-id> [limit]")
	}
	sprintID := args[0]
	limit := "50"
	if len(args) > 1 {
		limit = args[1]
	}
	params := url.Values{
		"maxResults": {limit},
		"fields":     {"summary,status,assignee,priority,issuetype,project"},
	}
	data, err := c.getAgile("/sprint/"+sprintID+"/issue", params)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	for _, issue := range jsonArr(m, "issues") {
		im := asMap(issue)
		if im == nil {
			continue
		}
		fields := jsonMap(im, "fields")
		project := jsonMap(fields, "project")
		status := jsonMap(fields, "status")
		priority := jsonMap(fields, "priority")
		assignee := jsonMap(fields, "assignee")
		fmt.Printf("[%s] %s: %s\n", jsonStr(project, "key"), jsonStr(im, "key"), jsonStr(fields, "summary"))
		fmt.Printf("  Status: %s  Priority: %s  Assignee: %s\n\n",
			jsonStr(status, "name"),
			strOr(jsonStr(priority, "name"), "None"),
			strOr(jsonStr(assignee, "displayName"), "Unassigned"))
	}
}

// ── Help ────────────────────────────────────────────────────

func printHelp() {
	fmt.Println(`Usage: jira-navigator <host> <command> [args...]

Discovery:
  discover [substring]                  Find Jira hosts in ~/.netrc

Commands (first arg is hostname or substring matching ~/.netrc):
  <host> whoami                         Show current user
  <host> test                           Test connection
  <host> recent [limit]                 Recently updated issues
  <host> my-issues [limit]              Issues assigned to you
  <host> watched [limit]                Unresolved watched issues
  <host> watch-changes [days]           Watched issues updated recently (default: 7d)
  <host> search <JQL> [limit]           Search via JQL
  <host> issue <key>                    Full issue details + description
  <host> issue-info <key>               Compact issue metadata (JSON)
  <host> comments <key> [limit]         Issue comments
  <host> transitions <key>              Available status transitions
  <host> changelog <key> [limit]        Issue change history
  <host> projects                       List all projects
  <host> project-info <key>             Project details
  <host> statuses [project-key]         List statuses
  <host> filters                        Favourite/saved filters
  <host> boards                         List agile boards
  <host> sprints <board-id> [state]     List sprints (active|closed|future)
  <host> sprint-issues <sprint-id>      Issues in a sprint`)
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

	hostname, entry, err := resolveHost(host, "jira")
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
	case "my-issues":
		cmdMyIssues(client, cmdArgs)
	case "watched":
		cmdWatched(client, cmdArgs)
	case "watch-changes":
		cmdWatchChanges(client, cmdArgs)
	case "search":
		cmdSearch(client, cmdArgs)
	case "issue":
		cmdIssue(client, cmdArgs)
	case "issue-info":
		cmdIssueInfo(client, cmdArgs)
	case "comments":
		cmdComments(client, cmdArgs)
	case "transitions":
		cmdTransitions(client, cmdArgs)
	case "changelog":
		cmdChangelog(client, cmdArgs)
	case "projects":
		cmdProjects(client)
	case "project-info":
		cmdProjectInfo(client, cmdArgs)
	case "statuses":
		cmdStatuses(client, cmdArgs)
	case "filters":
		cmdFilters(client)
	case "boards":
		cmdBoards(client)
	case "sprints":
		cmdSprints(client, cmdArgs)
	case "sprint-issues":
		cmdSprintIssues(client, cmdArgs)
	case "help":
		printHelp()
	default:
		die("Unknown command: %s (run with 'help')", command)
	}
}
