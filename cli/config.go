package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var upstreamAllowlist = []string{
	"openharmony/docs",
	"openharmony/interface_sdk-js",
	"openharmony/interface_sdk_c",
	"HarmonyOS_Samples/guide-snippets",
}

type Config struct {
	Upstream    string            `json:"upstream"`
	Fork        string            `json:"fork"`
	UpstreamRepo string           `json:"upstream_repo"`
	Lines       map[string]string `json:"lines"`
	CreateIssue bool              `json:"create_issue"`
}

func configPath() (string, error) {
	root, err := gitTopLevel()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".blueship.json"), nil
}

func loadConfig() (*Config, error) {
	p, err := configPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func saveConfig(c *Config) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(b, '\n'), 0644)
}

func isAllowlisted(ownerRepo string) bool {
	for _, a := range upstreamAllowlist {
		if strings.EqualFold(a, ownerRepo) {
			return true
		}
	}
	return false
}

func defaultLinesForRepo(repo string) map[string]string {
	switch repo {
	case "docs":
		return map[string]string{
			"master": "master",
			"sta":    "OpenHarmony_feature_sta_20260331",
			"70":     "OpenHarmony-7.0-Release",
			"weekly": "weekly_{date}",
		}
	default:
		return map[string]string{
			"master": "master",
		}
	}
}

func resolveLineBase(c *Config, line, weeklyDate string) (string, error) {
	tmpl, ok := c.Lines[line]
	if !ok {
		return "", fmt.Errorf("unknown line %q (not in config.lines)", line)
	}
	base := tmpl
	if strings.Contains(base, "{date}") {
		if weeklyDate == "" {
			return "", &ExitError{Code: 3, Message: "line " + line + " needs --weekly-date YYYYMMDD"}
		}
		base = strings.ReplaceAll(base, "{date}", weeklyDate)
	}
	return base, nil
}

func branchName(series, line string) string {
	if line == "weekly" {
		return series + "_701"
	}
	return series + "_" + line
}
