package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const gitcodeAPI = "https://gitcode.com/api/v5"

func gitcodeToken() (string, error) {
	if t := strings.TrimSpace(os.Getenv("GITCODE_TOKEN")); t != "" {
		return t, nil
	}
	out, err := exec.Command("git", "config", "--get", "gitcode.token").Output()
	if err == nil {
		t := strings.TrimSpace(string(out))
		if t != "" {
			return t, nil
		}
	}
	return "", &ExitError{Code: 2, Message: "missing GitCode token: set git config gitcode.token or GITCODE_TOKEN"}
}

func apiDo(method, path string, body any) (int, []byte, error) {
	token, err := gitcodeToken()
	if err != nil {
		return 0, nil, err
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, gitcodeAPI+path, rdr)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return resp.StatusCode, data, err
}

type issueResp struct {
	Number  any    `json:"number"`
	HTMLURL string `json:"html_url"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

func createBlankIssue(owner, repo, title string) (*issueResp, error) {
	code, data, err := apiDo("POST", fmt.Sprintf("/repos/%s/%s/issues", owner, repo), map[string]string{
		"title": title,
		"body":  "-",
	})
	if err != nil {
		return nil, err
	}
	var ir issueResp
	_ = json.Unmarshal(data, &ir)
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf("create issue HTTP %d: %s", code, strings.TrimSpace(string(data)))
	}
	return &ir, nil
}

func issueNumberString(n any) string {
	switch v := n.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	default:
		return fmt.Sprint(v)
	}
}

type prBrief struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
	State   string `json:"state"`
	Head    struct {
		Ref  string `json:"ref"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"head"`
}

func listOpenPRs(owner, repo string) ([]prBrief, error) {
	var all []prBrief
	for page := 1; page <= 5; page++ {
		code, data, err := apiDo("GET", fmt.Sprintf("/repos/%s/%s/pulls?state=open&per_page=100&page=%d", owner, repo, page), nil)
		if err != nil {
			return nil, err
		}
		if code != 200 {
			return nil, fmt.Errorf("list PRs HTTP %d: %s", code, string(data))
		}
		var batch []prBrief
		if err := json.Unmarshal(data, &batch); err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
	}
	return all, nil
}

func postStartBuild(owner, repo string, mr int) error {
	code, data, err := apiDo("POST", fmt.Sprintf("/repos/%s/%s/pulls/%d/comments", owner, repo, mr), map[string]string{
		"body": "start build",
	})
	if err != nil {
		return err
	}
	if code != 201 && code != 200 {
		return fmt.Errorf("start build on !%d HTTP %d: %s", mr, code, strings.TrimSpace(string(data)))
	}
	return nil
}
