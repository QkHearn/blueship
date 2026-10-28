package main

import (
	"fmt"
	"os"
	"strings"
)

type StatusOut struct {
	State       string            `json:"state"`
	Suggestion  string            `json:"suggestion"`
	HasConfig   bool              `json:"has_config"`
	Upstream    string            `json:"upstream,omitempty"`
	Fork        string            `json:"fork,omitempty"`
	UpstreamRepo string           `json:"upstream_repo,omitempty"`
	CurrentBranch string          `json:"current_branch,omitempty"`
	HasStaged   bool              `json:"has_staged"`
	TokenOK     bool              `json:"token_ok"`
	Warn        string            `json:"warn,omitempty"`
	Lines       map[string]string `json:"lines,omitempty"`
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

	switch {
	case !st.HasConfig:
		st.State = "need_init"
		st.Suggestion = "blueship init"
	case st.Upstream == "" || st.Fork == "":
		st.State = "need_remotes"
		st.Suggestion = "add allowlisted upstream + fork remotes, then blueship init"
	case st.HasStaged:
		st.State = "ready_ship"
		st.Suggestion = "blueship review (via skill) then blueship ship --series <name> --lines master,sta,70"
	case strings.Contains(st.CurrentBranch, "_"):
		st.State = "in_series"
		st.Suggestion = "if amending: blueship ship --amend --series <name>; if MRs ready: blueship gate --series <name>"
	default:
		st.State = "idle"
		st.Suggestion = "stage changes, then blueship ship --series <name> --lines ..."
	}

	if asJSON {
		return printJSON(st)
	}
	fmt.Fprintf(os.Stdout, "state: %s\n", st.State)
	fmt.Fprintf(os.Stdout, "suggestion: %s\n", st.Suggestion)
	fmt.Fprintf(os.Stdout, "config: %v  upstream: %s (%s)  fork: %s  staged: %v  token: %v\n",
		st.HasConfig, st.Upstream, st.UpstreamRepo, st.Fork, st.HasStaged, st.TokenOK)
	if st.Warn != "" {
		fmt.Fprintf(os.Stdout, "warn: %s\n", st.Warn)
	}
	if st.CurrentBranch != "" {
		fmt.Fprintf(os.Stdout, "branch: %s\n", st.CurrentBranch)
	}
	return nil
}
