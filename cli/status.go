package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type StatusOut struct {
	State            string            `json:"state"`
	Suggestion       string            `json:"suggestion"`
	RecommendedMode  string            `json:"recommended_mode,omitempty"` // ship | amend | gate | review | init | idle
	HasConfig        bool              `json:"has_config"`
	Upstream         string            `json:"upstream,omitempty"`
	Fork             string            `json:"fork,omitempty"`
	UpstreamRepo     string            `json:"upstream_repo,omitempty"`
	CurrentBranch    string            `json:"current_branch,omitempty"`
	HasStaged        bool              `json:"has_staged"`
	TokenOK          bool              `json:"token_ok"`
	Warn             string            `json:"warn,omitempty"`
	Lines            map[string]string `json:"lines,omitempty"`
	LastSeries       string            `json:"last_series,omitempty"`
	LastLines        []string          `json:"last_lines,omitempty"`
	SeriesBranches   []string          `json:"series_branches,omitempty"`
}

func cmdStatus(args []string) error {
	asJSON := flagBool(args, "--json")
	st := StatusOut{}

	remotes, err := listRemotes()
	if err != nil {
		return &ExitError{Code: 2, Message: "not a git repo or git missing: " + err.Error()}
	}
	up, fork, warn := detectRoles(remotes)
	st.Warn = warn
	if up != nil {
		st.Upstream = up.Name
		st.UpstreamRepo = up.OwnerRepo
	}
	if fork != nil {
		st.Fork = fork.Name
	}

	if _, err := gitcodeToken(); err == nil {
		st.TokenOK = true
	}

	if c, err := loadConfig(); err == nil {
		st.HasConfig = true
		st.Upstream = c.Upstream
		st.Fork = c.Fork
		st.UpstreamRepo = c.UpstreamRepo
		st.Lines = c.Lines
	}

	if b, err := currentBranch(); err == nil {
		st.CurrentBranch = b
	}
	if hs, err := hasStaged(); err == nil {
		st.HasStaged = hs
	}

	if last, err := loadLastShip(); err == nil && last != nil && last.Series != "" {
		st.LastSeries = last.Series
		for _, r := range last.Replicas {
			st.LastLines = append(st.LastLines, r.Line)
			if r.Branch != "" {
				st.SeriesBranches = append(st.SeriesBranches, r.Branch)
			}
		}
	}
	// Prefer local branches matching last series / current series-like branch.
	seriesHint := st.LastSeries
	if seriesHint == "" {
		seriesHint = seriesFromBranch(st.CurrentBranch)
	}
	if seriesHint != "" {
		if local := listLocalSeriesBranches(seriesHint); len(local) > 0 {
			st.SeriesBranches = local
			if st.LastSeries == "" {
				st.LastSeries = seriesHint
			}
		}
	}
	hasSeriesBranches := len(st.SeriesBranches) > 0

	switch {
	case !st.HasConfig:
		st.State = "need_init"
		st.RecommendedMode = "init"
		st.Suggestion = "blueship init"
	case st.Upstream == "" || st.Fork == "":
		st.State = "need_remotes"
		st.RecommendedMode = "init"
		st.Suggestion = "add allowlisted upstream + fork remotes, then blueship init"
	case st.HasStaged && hasSeriesBranches && st.LastSeries != "":
		st.State = "ready_amend"
		st.RecommendedMode = "amend"
		st.Suggestion = fmt.Sprintf("blueship ship --amend --series %s --lines %s", st.LastSeries, joinCSV(st.LastLinesOrConfig()))
	case st.HasStaged:
		st.State = "ready_ship"
		st.RecommendedMode = "ship"
		st.Suggestion = "blueship ship --series <name> --lines master,sta,70"
	case hasSeriesBranches && st.LastSeries != "":
		st.State = "in_series"
		st.RecommendedMode = "gate"
		st.Suggestion = fmt.Sprintf("MRs ready → blueship gate --series %s; or stage more changes then amend", st.LastSeries)
	case strings.Contains(st.CurrentBranch, "_"):
		st.State = "in_series"
		st.RecommendedMode = "gate"
		st.Suggestion = "if MRs ready: blueship gate --series <name>; stage changes to amend"
	default:
		st.State = "idle"
		st.RecommendedMode = "idle"
		st.Suggestion = "stage changes, then blueship ship --series <name> --lines ..."
	}

	if asJSON {
		return printJSON(st)
	}
	fmt.Fprintf(os.Stdout, "state: %s\n", st.State)
	fmt.Fprintf(os.Stdout, "recommended_mode: %s\n", st.RecommendedMode)
	fmt.Fprintf(os.Stdout, "suggestion: %s\n", st.Suggestion)
	fmt.Fprintf(os.Stdout, "config: %v  upstream: %s (%s)  fork: %s  staged: %v  token: %v\n",
		st.HasConfig, st.Upstream, st.UpstreamRepo, st.Fork, st.HasStaged, st.TokenOK)
	if st.LastSeries != "" {
		fmt.Fprintf(os.Stdout, "last_series: %s  branches: %s\n", st.LastSeries, strings.Join(st.SeriesBranches, ","))
	}
	if st.Warn != "" {
		fmt.Fprintf(os.Stdout, "warn: %s\n", st.Warn)
	}
	if st.CurrentBranch != "" {
		fmt.Fprintf(os.Stdout, "branch: %s\n", st.CurrentBranch)
	}
	return nil
}

func (st StatusOut) LastLinesOrConfig() []string {
	if len(st.LastLines) > 0 {
		return st.LastLines
	}
	var keys []string
	for k := range st.Lines {
		if k != "weekly" {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return []string{"master", "sta", "70"}
	}
	return keys
}

func loadLastShip() (*ShipOut, error) {
	top, err := gitTopLevel()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filepath.Join(top, ".blueship-last-ship.json"))
	if err != nil {
		return nil, err
	}
	var out ShipOut
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func seriesFromBranch(branch string) string {
	branch = strings.TrimSpace(branch)
	if branch == "" || !strings.Contains(branch, "_") {
		return ""
	}
	// 1024_master / 1024_701 → 1024
	i := strings.LastIndex(branch, "_")
	if i <= 0 {
		return ""
	}
	return branch[:i]
}

func listLocalSeriesBranches(series string) []string {
	out, err := runGit("branch", "--list", series+"_*")
	if err != nil || strings.TrimSpace(out) == "" {
		return nil
	}
	var names []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "*"))
		line = strings.TrimSpace(line)
		if line != "" {
			names = append(names, line)
		}
	}
	return names
}

func joinCSV(parts []string) string {
	return strings.Join(parts, ",")
}
