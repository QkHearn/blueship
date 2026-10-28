package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	var err error
	switch cmd {
	case "version", "-v", "--version":
		fmt.Println(version)
	case "help", "-h", "--help":
		printHelp()
	case "init":
		err = cmdInit(args)
	case "status":
		err = cmdStatus(args)
	case "ship":
		err = cmdShip(args)
	case "gate":
		err = cmdGate(args)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printHelp()
		os.Exit(1)
	}
	if err != nil {
		var ee *ExitError
		if errors.As(err, &ee) {
			fmt.Fprintln(os.Stderr, ee.Message)
			os.Exit(ee.Code)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Print(`blueship — multi-line doc publish helper (fork push + issues + start build)

Usage:
  blueship init
  blueship status [--json]
  blueship ship --series NAME --lines master,sta,70 [--weekly-date YYYYMMDD] [--amend] [--no-issue] [--message MSG]
  blueship gate --series NAME | --mr N,N,...
  blueship version

Exit codes:
  0 ok
  2 missing dependency / config
  3 needs user confirmation
`)
}

type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string { return e.Message }

func flagBool(args []string, name string) bool {
	for _, a := range args {
		if a == name {
			return true
		}
	}
	return false
}

func flagValue(args []string, name string) string {
	for i, a := range args {
		if a == name && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, name+"=") {
			return strings.TrimPrefix(a, name+"=")
		}
	}
	return ""
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
