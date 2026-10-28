package main

import (
	"fmt"
	"strconv"
	"strings"
)

type GateResult struct {
	MR      int    `json:"mr"`
	Branch  string `json:"branch,omitempty"`
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

type GateOut struct {
	Results []GateResult `json:"results"`
}

func cmdGate(args []string) error {
	series := flagValue(args, "--series")
	mrCSV := flagValue(args, "--mr")
	c, err := loadConfig()
	if err != nil {
		return &ExitError{Code: 2, Message: "missing .blueship.json — run blueship init"}
	}
	parts := strings.Split(c.UpstreamRepo, "/")
	if len(parts) != 2 {
		return &ExitError{Code: 2, Message: "bad upstream_repo"}
	}
	owner, repo := parts[0], parts[1]

	out := GateOut{}
	var mrs []int
	var branches map[int]string

	if mrCSV != "" {
		for _, p := range splitCSV(mrCSV) {
			n, err := strconv.Atoi(p)
			if err != nil {
				return &ExitError{Code: 3, Message: "bad --mr value: " + p}
			}
			mrs = append(mrs, n)
		}
	} else if series != "" {
		prs, err := listOpenPRs(owner, repo)
		if err != nil {
			return err
		}
		prefix := series + "_"
		branches = map[int]string{}
		for _, pr := range prs {
			ref := pr.Head.Ref
			if strings.HasPrefix(ref, prefix) {
				mrs = append(mrs, pr.Number)
				branches[pr.Number] = ref
			}
		}
		if len(mrs) == 0 {
			return &ExitError{Code: 3, Message: fmt.Sprintf("no open MRs with head branch prefix %q; pass --mr or create MRs first", prefix)}
		}
	} else {
		return &ExitError{Code: 3, Message: "required: --series NAME or --mr N,N"}
	}

	for _, mr := range mrs {
		gr := GateResult{MR: mr}
		if branches != nil {
			gr.Branch = branches[mr]
		}
		if err := postStartBuild(owner, repo, mr); err != nil {
			gr.Message = err.Error()
		} else {
			gr.OK = true
		}
		out.Results = append(out.Results, gr)
	}
	return printJSON(out)
}
