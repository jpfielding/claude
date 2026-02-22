package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
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
		return "", netrcEntry{}, fmt.Errorf("no ~/.netrc machine entries matching '%s' (filtered for harbor hosts)", input)
	}
	if len(matches) > 1 {
		msg := fmt.Sprintf("Multiple ~/.netrc entries match '%s':\n", input)
		for _, m := range matches {
			msg += fmt.Sprintf("  %s\n", m.Machine)
		}
		return "", netrcEntry{}, fmt.Errorf("%sambiguous host substring '%s'. Use a more specific substring or full hostname.", msg, input)
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

// ── HTTP client (no auth) ───────────────────────────────────

type apiClient struct {
	baseURL string
}

func newClient(host string) *apiClient {
	return &apiClient{baseURL: "https://" + host}
}

func (c *apiClient) get(endpoint string, params url.Values) (json.RawMessage, error) {
	u := c.baseURL + "/api/v2.0" + endpoint
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
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

func (c *apiClient) getWithHeaders(endpoint string, params url.Values) (json.RawMessage, http.Header, error) {
	u := c.baseURL + "/api/v2.0" + endpoint
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
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

func splitProjectRepo(ref string) (string, string) {
	idx := strings.Index(ref, "/")
	if idx < 0 {
		return ref, ""
	}
	return ref[:idx], ref[idx+1:]
}

// ── Commands ────────────────────────────────────────────────

func cmdDiscover(args []string) {
	filter := "harbor"
	if len(args) > 0 {
		filter = args[0]
	}
	hosts := discoverHosts(filter)
	if len(hosts) == 0 {
		fmt.Printf("No ~/.netrc entries matching '%s' found.\n\n", filter)
		fmt.Println("To search with a different substring: harbor-navigator discover <substring>")
		return
	}
	fmt.Printf("Found ~/.netrc entries matching '%s':\n\n", filter)
	for i, h := range hosts {
		fmt.Printf("  [%d] %s\n", i+1, h)
	}
	fmt.Println()
	fmt.Println("Usage: harbor-navigator <hostname-or-substring> <command>")
}

func cmdWhoami(c *apiClient) {
	data, err := c.get("/users/current", nil)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	out := map[string]any{
		"username": jsonStr(m, "username"),
		"realname": jsonStr(m, "realname"),
		"email":    jsonStr(m, "email"),
		"admin":    m["sysadmin_flag"],
		"user_id":  m["user_id"],
	}
	printJSON(out)
}

func cmdTest(c *apiClient) {
	fmt.Printf("Testing connection to %s...\n\n", c.baseURL)

	sysinfo, err := c.get("/systeminfo", nil)
	version := "unknown"
	authMode := "unknown"
	if err == nil {
		var m map[string]any
		json.Unmarshal(sysinfo, &m)
		version = strOr(jsonStr(m, "harbor_version"), "unknown")
		authMode = strOr(jsonStr(m, "auth_mode"), "unknown")
	}
	fmt.Printf("Harbor version: %s (auth_mode: %s)\n\n", version, authMode)

	fmt.Println("Testing project listing...")
	data, err := c.get("/projects", url.Values{"page_size": {"5"}})
	if err != nil {
		die("%s", err)
	}
	var projects []any
	json.Unmarshal(data, &projects)
	for _, p := range projects {
		pm := asMap(p)
		if pm != nil {
			fmt.Printf("  %s (repos: %s)\n", jsonStr(pm, "name"), strOr(jsonStr(pm, "repo_count"), "0"))
		}
	}
	fmt.Println()

	health, err := c.get("/health", nil)
	status := "unknown"
	if err == nil {
		var hm map[string]any
		json.Unmarshal(health, &hm)
		status = strOr(jsonStr(hm, "status"), "unknown")
	}
	fmt.Printf("System health: %s\n", status)
	fmt.Println("Connection OK.")
}

func cmdSystemInfo(c *apiClient) {
	data, err := c.get("/systeminfo", nil)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	out := map[string]any{
		"harbor_version":                 jsonStr(m, "harbor_version"),
		"registry_url":                   jsonStr(m, "registry_url"),
		"with_notary":                    m["with_notary"],
		"auth_mode":                      jsonStr(m, "auth_mode"),
		"project_creation_restriction":   jsonStr(m, "project_creation_restriction"),
		"self_registration":              m["self_registration"],
		"has_ca_root":                    m["has_ca_root"],
		"registry_storage_provider":      strOr(jsonStr(m, "storage_provider_name"), "unknown"),
	}
	printJSON(out)
}

func cmdHealth(c *apiClient) {
	data, err := c.get("/health", nil)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	fmt.Printf("Overall: %s\n\nComponents:\n", strOr(jsonStr(m, "status"), "unknown"))
	for _, comp := range jsonArr(m, "components") {
		cm := asMap(comp)
		if cm != nil {
			fmt.Printf("  %s: %s\n", jsonStr(cm, "name"), jsonStr(cm, "status"))
		}
	}
}

func cmdProjects(c *apiClient, args []string) {
	limit := "25"
	if len(args) > 0 {
		limit = args[0]
	}
	data, err := c.get("/projects", url.Values{"page_size": {limit}})
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
		meta := jsonMap(pm, "metadata")
		pub := "false"
		if meta != nil {
			pub = strOr(jsonStr(meta, "public"), "false")
		}
		fmt.Printf("%s\n  Repos: %s  Public: %s  Created: %s\n\n",
			jsonStr(pm, "name"),
			strOr(jsonStr(pm, "repo_count"), "0"),
			pub,
			jsonStr(pm, "creation_time"))
	}
}

func cmdProjectInfo(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: project-info <project-name>")
	}
	data, err := c.get("/projects/"+args[0], nil)
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	meta := jsonMap(m, "metadata")
	out := map[string]any{
		"name":                       jsonStr(m, "name"),
		"project_id":                 m["project_id"],
		"public":                     strOr(jsonStr(meta, "public"), "false"),
		"repo_count":                 m["repo_count"],
		"owner":                      strOr(jsonStr(m, "owner_name"), "unknown"),
		"creation_time":              jsonStr(m, "creation_time"),
		"update_time":                jsonStr(m, "update_time"),
		"auto_scan":                  strOr(jsonStr(meta, "auto_scan"), "unknown"),
		"severity":                   strOr(jsonStr(meta, "severity"), "unknown"),
		"reuse_sys_cve_allowlist":    strOr(jsonStr(meta, "reuse_sys_cve_allowlist"), "unknown"),
	}
	printJSON(out)
}

func cmdRepos(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: repos <project-name> [limit]")
	}
	project := args[0]
	limit := "25"
	if len(args) > 1 {
		limit = args[1]
	}
	data, headers, err := c.getWithHeaders("/projects/"+project+"/repositories", url.Values{"page_size": {limit}})
	if err != nil {
		die("%s", err)
	}
	total := headers.Get("X-Total-Count")
	if total != "" {
		fmt.Printf("Repositories in %s (%s total):\n\n", project, total)
	} else {
		fmt.Printf("Repositories in %s:\n\n", project)
	}
	var repos []any
	json.Unmarshal(data, &repos)
	for _, r := range repos {
		rm := asMap(r)
		if rm == nil {
			continue
		}
		fmt.Printf("%s\n  Artifacts: %s  Pulls: %s  Updated: %s\n\n",
			jsonStr(rm, "name"),
			strOr(jsonStr(rm, "artifact_count"), "0"),
			strOr(jsonStr(rm, "pull_count"), "0"),
			jsonStr(rm, "update_time"))
	}
}

func cmdArtifacts(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: artifacts <project/repo-name> [limit]")
	}
	project, repo := splitProjectRepo(args[0])
	if repo == "" {
		die("Provide full path: <project>/<repo-name>")
	}
	limit := "25"
	if len(args) > 1 {
		limit = args[1]
	}
	encodedRepo := url.PathEscape(repo)
	params := url.Values{
		"page_size":          {limit},
		"with_tag":           {"true"},
		"with_scan_overview": {"true"},
		"with_label":         {"true"},
	}
	data, err := c.get("/projects/"+project+"/repositories/"+encodedRepo+"/artifacts", params)
	if err != nil {
		die("%s", err)
	}
	var artifacts []any
	json.Unmarshal(data, &artifacts)
	for _, a := range artifacts {
		am := asMap(a)
		if am == nil {
			continue
		}
		digest := jsonStr(am, "digest")
		if len(digest) > 19 {
			digest = digest[:19] + "..."
		}
		// tags
		var tagNames []string
		for _, t := range jsonArr(am, "tags") {
			if tm := asMap(t); tm != nil {
				tagNames = append(tagNames, jsonStr(tm, "name"))
			}
		}
		tags := "<none>"
		if len(tagNames) > 0 {
			tags = strings.Join(tagNames, ", ")
		}
		// size in MB
		sizeMB := math.Floor(jsonFloat(am, "size") / 1048576)
		// scan overview
		scan := "not scanned"
		if so := jsonMap(am, "scan_overview"); so != nil {
			var parts []string
			for k, v := range so {
				vm := asMap(v)
				if vm != nil {
					if summary := jsonMap(vm, "summary"); summary != nil {
						sj, _ := json.Marshal(summary)
						parts = append(parts, fmt.Sprintf("%s: %s", k, string(sj)))
					}
				}
			}
			if len(parts) > 0 {
				scan = strings.Join(parts, ", ")
			}
		}
		fmt.Printf("Digest: %s\n", digest)
		fmt.Printf("  Tags: %s\n", tags)
		fmt.Printf("  Type: %s  Size: %.0fMB  Pushed: %s\n",
			strOr(jsonStr(am, "type"), "unknown"), sizeMB, jsonStr(am, "push_time"))
		fmt.Printf("  Scan: %s\n\n", scan)
	}
}

func cmdTags(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: tags <project/repo-name> [digest-or-tag]")
	}
	project, repo := splitProjectRepo(args[0])
	if repo == "" {
		die("Provide full path: <project>/<repo-name>")
	}
	encodedRepo := url.PathEscape(repo)
	reference := ""
	if len(args) > 1 {
		reference = args[1]
	}

	if reference != "" {
		data, err := c.get("/projects/"+project+"/repositories/"+encodedRepo+"/artifacts/"+reference+"/tags", nil)
		if err != nil {
			die("%s", err)
		}
		var tags []any
		json.Unmarshal(data, &tags)
		for _, t := range tags {
			tm := asMap(t)
			if tm != nil {
				fmt.Printf("%s  Created: %s\n", jsonStr(tm, "name"), jsonStr(tm, "push_time"))
			}
		}
	} else {
		data, err := c.get("/projects/"+project+"/repositories/"+encodedRepo+"/artifacts",
			url.Values{"page_size": {"50"}, "with_tag": {"true"}})
		if err != nil {
			die("%s", err)
		}
		var artifacts []any
		json.Unmarshal(data, &artifacts)
		for _, a := range artifacts {
			am := asMap(a)
			if am == nil {
				continue
			}
			tags := jsonArr(am, "tags")
			if len(tags) == 0 {
				continue
			}
			for _, t := range tags {
				tm := asMap(t)
				if tm != nil {
					fmt.Printf("%s  Pushed: %s\n", jsonStr(tm, "name"), jsonStr(tm, "push_time"))
				}
			}
		}
	}
}

func cmdSearch(c *apiClient, args []string) {
	if len(args) == 0 {
		die("Usage: search <query>")
	}
	query := args[0]
	data, err := c.get("/search", url.Values{"q": {query}})
	if err != nil {
		die("%s", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	fmt.Println("Projects:")
	for _, p := range jsonArr(m, "project") {
		pm := asMap(p)
		if pm != nil {
			meta := jsonMap(pm, "metadata")
			pub := "false"
			if meta != nil {
				pub = strOr(jsonStr(meta, "public"), "false")
			}
			fmt.Printf("  %s (repos: %s, public: %s)\n",
				jsonStr(pm, "name"), strOr(jsonStr(pm, "repo_count"), "0"), pub)
		}
	}
	fmt.Println()
	fmt.Println("Repositories:")
	for _, r := range jsonArr(m, "repository") {
		rm := asMap(r)
		if rm != nil {
			fmt.Printf("  %s  Project: %s  Pulls: %s\n",
				jsonStr(rm, "repository_name"),
				jsonStr(rm, "project_name"),
				strOr(jsonStr(rm, "pull_count"), "0"))
		}
	}
}

func cmdRecentPushes(c *apiClient, args []string) {
	projectName := ""
	limit := 20
	if len(args) > 0 {
		projectName = args[0]
	}
	if len(args) > 1 {
		limit, _ = strconv.Atoi(args[1])
		if limit <= 0 {
			limit = 20
		}
	}

	if projectName != "" {
		data, err := c.get("/projects/"+projectName+"/repositories",
			url.Values{"page_size": {strconv.Itoa(limit)}, "sort": {"-update_time"}})
		if err != nil {
			die("%s", err)
		}
		fmt.Printf("Recently updated repositories in %s:\n\n", projectName)
		var repos []any
		json.Unmarshal(data, &repos)
		for _, r := range repos {
			rm := asMap(r)
			if rm != nil {
				fmt.Printf("%s\n  Artifacts: %s  Pulls: %s  Updated: %s\n\n",
					jsonStr(rm, "name"),
					strOr(jsonStr(rm, "artifact_count"), "0"),
					strOr(jsonStr(rm, "pull_count"), "0"),
					jsonStr(rm, "update_time"))
			}
		}
	} else {
		fmt.Println("Recently updated repositories across all projects:")
		fmt.Println()
		data, err := c.get("/projects", url.Values{"page_size": {"100"}})
		if err != nil {
			die("%s", err)
		}
		var projects []any
		json.Unmarshal(data, &projects)
		count := 0
		for _, p := range projects {
			pm := asMap(p)
			if pm == nil {
				continue
			}
			pname := jsonStr(pm, "name")
			if pname == "" {
				continue
			}
			repos, err := c.get("/projects/"+pname+"/repositories",
				url.Values{"page_size": {"5"}, "sort": {"-update_time"}})
			if err != nil {
				continue
			}
			var repoList []any
			json.Unmarshal(repos, &repoList)
			for _, r := range repoList {
				rm := asMap(r)
				if rm != nil {
					fmt.Printf("%s\n  Artifacts: %s  Pulls: %s  Updated: %s\n\n",
						jsonStr(rm, "name"),
						strOr(jsonStr(rm, "artifact_count"), "0"),
						strOr(jsonStr(rm, "pull_count"), "0"),
						jsonStr(rm, "update_time"))
					count++
				}
			}
			if count >= limit {
				break
			}
		}
	}
}

func cmdVulns(c *apiClient, args []string) {
	if len(args) < 2 {
		die("Usage: vulns <project/repo-name> <tag-or-digest>")
	}
	project, repo := splitProjectRepo(args[0])
	if repo == "" {
		die("Provide full path: <project>/<repo-name>")
	}
	reference := args[1]
	encodedRepo := url.PathEscape(repo)
	data, err := c.get("/projects/"+project+"/repositories/"+encodedRepo+"/artifacts/"+reference+"/additions/vulnerabilities", nil)
	if err != nil {
		die("%s", err)
	}
	var scanners map[string]any
	json.Unmarshal(data, &scanners)
	for scanner, report := range scanners {
		rm := asMap(report)
		if rm == nil {
			continue
		}
		fmt.Printf("Scanner: %s\n", scanner)
		fmt.Printf("Generated: %s\n", strOr(jsonStr(rm, "generated_at"), "unknown"))
		fmt.Printf("Severity: %s\n\n", strOr(jsonStr(rm, "severity"), "unknown"))

		fmt.Println("Summary:")
		if summary := jsonMap(rm, "summary"); summary != nil {
			if summaryMap := jsonMap(summary, "summary"); summaryMap != nil {
				for k, v := range summaryMap {
					fmt.Printf("  %s: %v\n", k, v)
				}
			} else {
				for k, v := range summary {
					fmt.Printf("  %s: %v\n", k, v)
				}
			}
		}

		vulns := jsonArr(rm, "vulnerabilities")
		fmt.Printf("\nVulnerabilities (%d total):\n", len(vulns))
		for _, v := range vulns {
			vm := asMap(v)
			if vm == nil {
				continue
			}
			fixVer := strOr(jsonStr(vm, "fix_version"), "no fix")
			link := "N/A"
			links := jsonArr(vm, "links")
			if len(links) > 0 {
				if s, ok := links[0].(string); ok {
					link = s
				}
			}
			fmt.Printf("  [%s] %s - %s:%s\n",
				jsonStr(vm, "severity"),
				jsonStr(vm, "id"),
				jsonStr(vm, "package"),
				jsonStr(vm, "version"))
			fmt.Printf("    Fixed: %s  Link: %s\n", fixVer, link)
		}
	}
}

func cmdScan(c *apiClient, args []string) {
	if len(args) < 2 {
		die("Usage: scan <project/repo-name> <tag-or-digest>")
	}
	die("scan command requires authentication which is not supported in unauthenticated mode")
}

func cmdLabels(c *apiClient, args []string) {
	scope := "g"
	if len(args) > 0 {
		scope = args[0]
	}
	data, err := c.get("/labels", url.Values{"scope": {scope}, "page_size": {"50"}})
	if err != nil {
		die("%s", err)
	}
	var labels []any
	json.Unmarshal(data, &labels)
	for _, l := range labels {
		lm := asMap(l)
		if lm == nil {
			continue
		}
		fmt.Printf("[%s] %s\n", jsonStr(lm, "id"), jsonStr(lm, "name"))
		fmt.Printf("  Scope: %s  Color: %s  Description: %s\n\n",
			jsonStr(lm, "scope"),
			strOr(jsonStr(lm, "color"), "none"),
			strOr(jsonStr(lm, "description"), "none"))
	}
}

func cmdReplicationPolicies(c *apiClient) {
	data, err := c.get("/replication/policies", url.Values{"page_size": {"25"}})
	if err != nil {
		die("%s", err)
	}
	var policies []any
	json.Unmarshal(data, &policies)
	for _, p := range policies {
		pm := asMap(p)
		if pm == nil {
			continue
		}
		trigger := jsonMap(pm, "trigger")
		srcReg := jsonMap(pm, "src_registry")
		destReg := jsonMap(pm, "dest_registry")
		// filters
		var filterStrs []string
		for _, f := range jsonArr(pm, "filters") {
			fm := asMap(f)
			if fm != nil {
				filterStrs = append(filterStrs, fmt.Sprintf("%s=%s", jsonStr(fm, "type"), jsonStr(fm, "value")))
			}
		}
		filters := "none"
		if len(filterStrs) > 0 {
			filters = strings.Join(filterStrs, ", ")
		}
		fmt.Printf("[%s] %s\n", jsonStr(pm, "id"), jsonStr(pm, "name"))
		fmt.Printf("  Enabled: %s  Trigger: %s\n",
			jsonStr(pm, "enabled"),
			strOr(jsonStr(trigger, "type"), "manual"))
		fmt.Printf("  Src: %s -> Dest: %s\n",
			strOr(jsonStr(srcReg, "name"), "local"),
			strOr(jsonStr(destReg, "name"), "local"))
		fmt.Printf("  Filters: %s\n\n", filters)
	}
}

func cmdReplicationRuns(c *apiClient, args []string) {
	policyID := ""
	limit := "10"
	if len(args) > 0 {
		policyID = args[0]
	}
	if len(args) > 1 {
		limit = args[1]
	}
	params := url.Values{
		"page_size": {limit},
		"sort":      {"-start_time"},
	}
	if policyID != "" {
		params.Set("policy_id", policyID)
	}
	data, err := c.get("/replication/executions", params)
	if err != nil {
		die("%s", err)
	}
	var runs []any
	json.Unmarshal(data, &runs)
	for _, r := range runs {
		rm := asMap(r)
		if rm == nil {
			continue
		}
		fmt.Printf("[%s] Policy: %s  Status: %s  Trigger: %s\n",
			jsonStr(rm, "id"),
			jsonStr(rm, "policy_id"),
			jsonStr(rm, "status"),
			strOr(jsonStr(rm, "trigger"), "unknown"))
		fmt.Printf("  Started: %s  Ended: %s\n",
			jsonStr(rm, "start_time"),
			strOr(jsonStr(rm, "end_time"), "running"))
		fmt.Printf("  Success: %s  Failed: %s  In-progress: %s\n\n",
			strOr(jsonStr(rm, "succeed"), "0"),
			strOr(jsonStr(rm, "failed"), "0"),
			strOr(jsonStr(rm, "in_progress"), "0"))
	}
}

func cmdRegistries(c *apiClient) {
	data, err := c.get("/registries", url.Values{"page_size": {"25"}})
	if err != nil {
		die("%s", err)
	}
	var registries []any
	json.Unmarshal(data, &registries)
	for _, r := range registries {
		rm := asMap(r)
		if rm == nil {
			continue
		}
		fmt.Printf("[%s] %s\n", jsonStr(rm, "id"), jsonStr(rm, "name"))
		fmt.Printf("  URL: %s  Type: %s  Status: %s\n\n",
			jsonStr(rm, "url"),
			jsonStr(rm, "type"),
			strOr(jsonStr(rm, "status"), "unknown"))
	}
}

func cmdGC(c *apiClient) {
	data, err := c.get("/system/gc/schedule", nil)
	if err != nil {
		die("%s", err)
	}
	fmt.Println("GC Schedule:")
	var schedule any
	json.Unmarshal(data, &schedule)
	printJSON(schedule)
	fmt.Println()
	fmt.Println("Recent GC runs:")
	history, err := c.get("/system/gc", url.Values{"page_size": {"5"}, "sort": {"-creation_time"}})
	if err != nil {
		die("%s", err)
	}
	var runs []any
	json.Unmarshal(history, &runs)
	for _, r := range runs {
		rm := asMap(r)
		if rm == nil {
			continue
		}
		fmt.Printf("[%s] Status: %s  Created: %s\n",
			jsonStr(rm, "id"),
			jsonStr(rm, "job_status"),
			jsonStr(rm, "creation_time"))
		fmt.Printf("  Updated: %s  Deleted: %s\n\n",
			jsonStr(rm, "update_time"),
			strOr(jsonStr(rm, "delete"), "false"))
	}
}

func cmdQuotas(c *apiClient) {
	data, err := c.get("/quotas", url.Values{"page_size": {"50"}})
	if err != nil {
		die("%s", err)
	}
	var quotas []any
	json.Unmarshal(data, &quotas)
	for _, q := range quotas {
		qm := asMap(q)
		if qm == nil {
			continue
		}
		ref := jsonMap(qm, "ref")
		refName := "unknown"
		if ref != nil {
			refName = strOr(jsonStr(ref, "name"), strOr(jsonStr(ref, "id"), "unknown"))
		}
		used := jsonMap(qm, "used")
		hard := jsonMap(qm, "hard")
		usedStorage := jsonFloat(used, "storage")
		hardStorage := jsonFloat(hard, "storage")
		usedMB := math.Floor(usedStorage / 1048576)
		hardStr := "unlimited"
		if hardStorage >= 0 {
			hardStr = fmt.Sprintf("%.0fMB", math.Floor(hardStorage/1048576))
		}
		fmt.Printf("[%s] Ref: %s\n", jsonStr(qm, "id"), refName)
		fmt.Printf("  Storage used: %.0fMB / %s\n\n", usedMB, hardStr)
	}
}

func cmdRobotAccounts(c *apiClient) {
	data, err := c.get("/robots", url.Values{"page_size": {"50"}})
	if err != nil {
		die("%s", err)
	}
	var robots []any
	json.Unmarshal(data, &robots)
	for _, r := range robots {
		rm := asMap(r)
		if rm == nil {
			continue
		}
		expiresAt := jsonFloat(rm, "expires_at")
		expires := "never"
		if expiresAt > 0 {
			expires = jsonStr(rm, "expires_at")
		}
		fmt.Printf("[%s] %s\n", jsonStr(rm, "id"), jsonStr(rm, "name"))
		fmt.Printf("  Level: %s  Disabled: %s  Expires: %s\n",
			jsonStr(rm, "level"),
			jsonStr(rm, "disable"),
			expires)
		fmt.Printf("  Created: %s\n\n", jsonStr(rm, "creation_time"))
	}
}

func cmdAuditLog(c *apiClient, args []string) {
	limit := "25"
	if len(args) > 0 {
		limit = args[0]
	}
	data, err := c.get("/audit-logs", url.Values{"page_size": {limit}, "sort": {"-op_time"}})
	if err != nil {
		die("%s", err)
	}
	var logs []any
	json.Unmarshal(data, &logs)
	for _, l := range logs {
		lm := asMap(l)
		if lm == nil {
			continue
		}
		fmt.Printf("%s [%s] %s: %s\n",
			jsonStr(lm, "op_time"),
			jsonStr(lm, "operation"),
			jsonStr(lm, "resource_type"),
			jsonStr(lm, "resource"))
		fmt.Printf("  User: %s\n\n", jsonStr(lm, "username"))
	}
}

// ── Help ────────────────────────────────────────────────────

func printHelp() {
	fmt.Println(`Usage: harbor-navigator <host> <command> [args...]

Global commands:
  discover [substring]                                 Find Harbor hosts in ~/.netrc
  help                                                 Show this help

Query commands (<host> is a hostname or ~/.netrc substring):
  <host> whoami                                Current user
  <host> test                                  Test connection
  <host> system-info                           Harbor version and config
  <host> health                                Component health status

  <host> projects [limit]                      List projects
  <host> project-info <name>                   Project details
  <host> repos <project> [limit]               Repositories in a project
  <host> artifacts <project/repo> [limit]      Artifacts (images) in a repo
  <host> tags <project/repo> [ref]             Tags on repo or specific artifact
  <host> search <query>                        Search projects and repos
  <host> recent-pushes [project] [limit]       Recently pushed repos

  <host> vulns <project/repo> <tag|digest>     Vulnerability report
  <host> scan <project/repo> <tag|digest>      Trigger vulnerability scan

  <host> labels [scope: g|p]                   List labels (g=global, p=project)
  <host> replication-policies                  Replication policies
  <host> replication-runs [policy-id] [limit]  Replication execution history
  <host> registries                            Connected registries
  <host> gc                                    Garbage collection schedule/history
  <host> quotas                                Storage quotas
  <host> robot-accounts                        Robot accounts
  <host> audit-log [limit]                     Recent audit log entries`)
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

	hostname, _, err := resolveHost(host, "harbor")
	if err != nil {
		die("%s", err)
	}
	client := newClient(hostname)

	switch command {
	case "whoami":
		cmdWhoami(client)
	case "test":
		cmdTest(client)
	case "system-info":
		cmdSystemInfo(client)
	case "health":
		cmdHealth(client)
	case "projects":
		cmdProjects(client, cmdArgs)
	case "project-info":
		cmdProjectInfo(client, cmdArgs)
	case "repos":
		cmdRepos(client, cmdArgs)
	case "artifacts":
		cmdArtifacts(client, cmdArgs)
	case "tags":
		cmdTags(client, cmdArgs)
	case "search":
		cmdSearch(client, cmdArgs)
	case "recent-pushes":
		cmdRecentPushes(client, cmdArgs)
	case "vulns":
		cmdVulns(client, cmdArgs)
	case "scan":
		cmdScan(client, cmdArgs)
	case "labels":
		cmdLabels(client, cmdArgs)
	case "replication-policies":
		cmdReplicationPolicies(client)
	case "replication-runs":
		cmdReplicationRuns(client, cmdArgs)
	case "registries":
		cmdRegistries(client)
	case "gc":
		cmdGC(client)
	case "quotas":
		cmdQuotas(client)
	case "robot-accounts":
		cmdRobotAccounts(client)
	case "audit-log":
		cmdAuditLog(client, cmdArgs)
	case "help":
		printHelp()
	default:
		die("Unknown command: %s (run with 'help')", command)
	}
}
