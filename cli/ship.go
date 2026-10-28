package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ShipReplica struct {
	Line      string `json:"line"`
	Branch    string `json:"branch"`
	RemoteRef string `json:"remote_ref,omitempty"` // e.g. hearn/1028_master
	RemoteURL string `json:"remote_url,omitempty"` // browser URL on fork
	SHA       string `json:"sha"`
	Issue     string `json:"issue,omitempty"`
	Pushed    bool   `json:"pushed"`
	Message   string `json:"message,omitempty"`
}

type ShipOut struct {
	Series   string        `json:"series"`
	Fork     string        `json:"fork,omitempty"`
	Mode     string        `json:"mode"` // ship | amend
	Replicas []ShipReplica `json:"replicas"`
}

func cmdShip(args []string) error {
	series := flagValue(args, "--series")
	linesCSV := flagValue(args, "--lines")
	weekly := flagValue(args, "--weekly-date")
	msg := flagValue(args, "--message")
	amend := flagBool(args, "--amend")
	noIssue := flagBool(args, "--no-issue")
	if msg == "" {
		msg = "fix bug"
	}
	if series == "" {
		return &ExitError{Code: 3, Message: "required: --series NAME"}
	}
	if linesCSV == "" {
		return &ExitError{Code: 3, Message: "required: --lines master,sta,70"}
	}
	lines := splitCSV(linesCSV)

	c, err := loadConfig()
	if err != nil {
		return &ExitError{Code: 2, Message: "missing .blueship.json — run blueship init"}
	}
	createIssue := c.CreateIssue && !noIssue

	hs, err := hasStaged()
	if err != nil {
		return err
	}
	if !hs {
		return &ExitError{Code: 3, Message: "no staged changes; stage files first"}
	}
	patch, err := stagedDiff()
	if err != nil {
		return err
	}
	if strings.TrimSpace(patch) == "" {
		return &ExitError{Code: 3, Message: "staged diff empty"}
	}

	tmpDir, err := os.MkdirTemp("", "blueship-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	patchFile := filepath.Join(tmpDir, "change.patch")
	if err := os.WriteFile(patchFile, []byte(patch+"\n"), 0644); err != nil {
		return err
	}

	orig, _ := currentBranch()
	mode := "ship"
	if amend {
		mode = "amend"
	}
	out := ShipOut{Series: series, Fork: c.Fork, Mode: mode}

	parts := strings.Split(c.UpstreamRepo, "/")
	if len(parts) != 2 {
		return &ExitError{Code: 2, Message: "bad upstream_repo in config"}
	}
	owner, repo := parts[0], parts[1]
	forkBrowse := forkBrowseBase(c.Fork)

	for _, line := range lines {
		rep := ShipReplica{Line: line, Branch: branchName(series, line)}
		baseRef, err := resolveLineBase(c, line, weekly)
		if err != nil {
			rep.Message = err.Error()
			out.Replicas = append(out.Replicas, rep)
			continue
		}
		remoteBase := c.Upstream + "/" + baseRef

		if _, err := runGit("fetch", c.Upstream, baseRef); err != nil {
			rep.Message = "fetch failed: " + err.Error()
			out.Replicas = append(out.Replicas, rep)
			continue
		}

		if amend {
			if _, err := runGit("rev-parse", "--verify", rep.Branch); err != nil {
				if _, err2 := runGit("fetch", c.Fork, rep.Branch); err2 == nil {
					if _, err3 := runGit("checkout", "-B", rep.Branch, c.Fork+"/"+rep.Branch); err3 != nil {
						rep.Message = err3.Error()
						out.Replicas = append(out.Replicas, rep)
						continue
					}
				} else {
					rep.Message = "amend: branch not found locally or on fork"
					out.Replicas = append(out.Replicas, rep)
					continue
				}
			} else if _, err := runGit("checkout", rep.Branch); err != nil {
				rep.Message = err.Error()
				out.Replicas = append(out.Replicas, rep)
				continue
			}
		} else if _, err := runGit("checkout", "-B", rep.Branch, remoteBase); err != nil {
			rep.Message = err.Error()
			out.Replicas = append(out.Replicas, rep)
			continue
		}

		cmdApply := exec.Command("git", "apply", "--3way", patchFile)
		if err := cmdApply.Run(); err != nil {
			cmdApply2 := exec.Command("git", "apply", patchFile)
			if err2 := cmdApply2.Run(); err2 != nil {
				rep.Message = "patch apply failed (bases differ?); resolve manually"
				out.Replicas = append(out.Replicas, rep)
				continue
			}
		}
		_, _ = runGit("add", "-A")

		var commitArgs []string
		if amend {
			commitArgs = []string{"commit", "--amend", "--no-edit", "-s"}
		} else {
			commitArgs = []string{"commit", "-s", "-m", msg}
		}
		if _, err := runGit(commitArgs...); err != nil {
			rep.Message = "commit failed: " + err.Error()
			out.Replicas = append(out.Replicas, rep)
			continue
		}
		sha, _ := runGit("rev-parse", "HEAD")
		rep.SHA = sha

		var pushArgs []string
		if amend {
			pushArgs = []string{"push", "--force-with-lease", c.Fork, rep.Branch + ":" + rep.Branch}
		} else {
			pushArgs = []string{"push", c.Fork, rep.Branch + ":" + rep.Branch}
		}
		if _, err := runGit(pushArgs...); err != nil {
			rep.Message = "push failed: " + err.Error()
			out.Replicas = append(out.Replicas, rep)
			continue
		}
		rep.Pushed = true
		rep.RemoteRef = c.Fork + "/" + rep.Branch
		if forkBrowse != "" {
			rep.RemoteURL = strings.TrimRight(forkBrowse, "/") + "/tree/" + rep.Branch
		}

		if createIssue && !amend {
			title := fmt.Sprintf("%s [%s / %s]", msg, baseRef, rep.Branch)
			ir, err := createBlankIssue(owner, repo, title)
			if err != nil {
				rep.Message = "issue failed: " + err.Error()
			} else {
				rep.Issue = ir.HTMLURL
				if rep.Issue == "" {
					rep.Issue = fmt.Sprintf("https://gitcode.com/%s/%s/issues/%s", owner, repo, issueNumberString(ir.Number))
				}
			}
		}
		out.Replicas = append(out.Replicas, rep)
	}

	if orig != "" {
		_, _ = runGit("checkout", orig)
	}

	if b, err := json.MarshalIndent(out, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(mustGitTop(), ".blueship-last-ship.json"), append(b, '\n'), 0644)
	}
	// human-readable remote refs (always print so Agent/user see fork branches)
	fmt.Fprintf(os.Stderr, "\n[blueship] %s series=%s → fork remote %q\n", out.Mode, out.Series, out.Fork)
	for _, r := range out.Replicas {
		if r.Pushed {
			fmt.Fprintf(os.Stderr, "  pushed  %s  %s\n", r.RemoteRef, shortSHA(r.SHA))
			if r.RemoteURL != "" {
				fmt.Fprintf(os.Stderr, "          %s\n", r.RemoteURL)
			}
		} else {
			fmt.Fprintf(os.Stderr, "  failed  %s  %s\n", r.Branch, r.Message)
		}
		if r.Issue != "" {
			fmt.Fprintf(os.Stderr, "          issue %s\n", r.Issue)
		}
	}
	fmt.Fprintln(os.Stderr)
	return printJSON(out)
}

func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// forkBrowseBase returns https://gitcode.com/<owner>/<repo> for the fork remote.
func forkBrowseBase(forkRemote string) string {
	out, err := runGit("remote", "get-url", forkRemote)
	if err != nil {
		return ""
	}
	o, r, ok := parseOwnerRepo(out)
	if !ok {
		return ""
	}
	return fmt.Sprintf("https://gitcode.com/%s/%s", o, r)
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func mustGitTop() string {
	t, err := gitTopLevel()
	if err != nil {
		return "."
	}
	return t
}
