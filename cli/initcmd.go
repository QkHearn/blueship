package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func cmdInit(args []string) error {
	remotes, err := listRemotes()
	if err != nil {
		return &ExitError{Code: 2, Message: err.Error()}
	}
	up, fork, warn := detectRoles(remotes)
	if warn != "" {
		fmt.Fprintln(os.Stderr, "detect:", warn)
	}
	fmt.Println("Remotes:")
	for _, r := range remotes {
		mark := ""
		if up != nil && r.Name == up.Name {
			mark = " [candidate upstream]"
		}
		if fork != nil && r.Name == fork.Name {
			mark = " [candidate fork]"
		}
		fmt.Printf("  %s\t%s%s\n", r.Name, r.URL, mark)
	}

	if up == nil || fork == nil {
		fmt.Println("\nNeed both:")
		fmt.Println("  - upstream (allowlist): openharmony/docs | interface_sdk-js | interface_sdk_c | HarmonyOS_Samples/guide-snippets")
		fmt.Println("  - fork: your account clone of the same repo")
		fmt.Println("Example:")
		fmt.Println("  git remote add upstream git@gitcode.com:openharmony/docs.git")
		fmt.Println("  git remote add myfork git@gitcode.com:<you>/docs.git")
		return &ExitError{Code: 2, Message: "fix remotes then re-run blueship init"}
	}

	reader := bufio.NewReader(os.Stdin)
	ask := func(prompt, def string) string {
		if def != "" {
			fmt.Printf("%s [%s]: ", prompt, def)
		} else {
			fmt.Printf("%s: ", prompt)
		}
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return def
		}
		return line
	}

	upName := ask("Upstream remote name", up.Name)
	forkName := ask("Fork remote name", fork.Name)
	createIssue := strings.ToLower(ask("Create blank issues on ship? (y/n)", "y"))
	lines := defaultLinesForRepo(up.Repo)

	c := &Config{
		Upstream:     upName,
		Fork:         forkName,
		UpstreamRepo: up.OwnerRepo,
		Lines:        lines,
		CreateIssue:  createIssue == "y" || createIssue == "yes",
	}
	// refresh OwnerRepo from selected upstream remote
	for _, r := range remotes {
		if r.Name == upName && r.OwnerRepo != "" {
			if !isAllowlisted(r.OwnerRepo) {
				return &ExitError{Code: 2, Message: "upstream not in allowlist: " + r.OwnerRepo}
			}
			c.UpstreamRepo = r.OwnerRepo
		}
	}
	if err := saveConfig(c); err != nil {
		return err
	}
	p, _ := configPath()
	fmt.Println("wrote", p)
	return printJSON(c)
}
