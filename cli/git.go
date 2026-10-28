package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return out, fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return out, nil
}

func gitTopLevel() (string, error) {
	return runGit("rev-parse", "--show-toplevel")
}

type Remote struct {
	Name string
	URL  string
	OwnerRepo string // owner/repo
	Owner string
	Repo  string
}

var ownerRepoRe = regexp.MustCompile(`(?i)(?:gitcode\.com[:/]|github\.com[:/])([^/]+)/([^/.]+)(?:\.git)?$`)

func parseOwnerRepo(url string) (owner, repo string, ok bool) {
	m := ownerRepoRe.FindStringSubmatch(strings.TrimSpace(url))
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

func listRemotes() ([]Remote, error) {
	out, err := runGit("remote", "-v")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var remotes []Remote
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name, url := fields[0], fields[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		r := Remote{Name: name, URL: url}
		if o, repo, ok := parseOwnerRepo(url); ok {
			r.Owner, r.Repo, r.OwnerRepo = o, repo, o+"/"+repo
		}
		remotes = append(remotes, r)
	}
	return remotes, nil
}

func detectRoles(remotes []Remote) (upstream, fork *Remote, warn string) {
	var ups, forks []*Remote
	for i := range remotes {
		r := &remotes[i]
		if r.OwnerRepo == "" {
			continue
		}
		if isAllowlisted(r.OwnerRepo) {
			ups = append(ups, r)
		} else {
			forks = append(forks, r)
		}
	}
	if len(ups) == 1 {
		upstream = ups[0]
	}
	if len(forks) >= 1 && upstream != nil {
		// prefer same repo name as upstream
		for _, f := range forks {
			if strings.EqualFold(f.Repo, upstream.Repo) {
				fork = f
				break
			}
		}
		if fork == nil {
			fork = forks[0]
		}
	}
	if len(ups) == 0 {
		warn = "no allowlisted upstream remote found"
	} else if len(ups) > 1 {
		warn = "multiple allowlisted upstream remotes; pick one in init"
	}
	if upstream != nil && fork == nil {
		if warn != "" {
			warn += "; "
		}
		warn += "no fork remote found for same repo"
	}
	return upstream, fork, warn
}

func stagedDiff() (string, error) {
	return runGit("diff", "--cached")
}

func hasStaged() (bool, error) {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	err := cmd.Run()
	if err == nil {
		return false, nil
	}
	if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
		return true, nil
	}
	return false, err
}

func currentBranch() (string, error) {
	return runGit("branch", "--show-current")
}
