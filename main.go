package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"text/template"
	"time"
)

var (
	// cwd is used to set the current working directory
	// of all the shell commands run.
	cwd                   string
	DateTimeLayout        = time.DateTime
	PackageName           = "go-autosave"
	DryRun                = flag.Bool("dry", false, "perform a dry run")
	DateTimeVerboseLayout = "Monday, January 02 2006 15:04:05 MST"
	CommitMessageFormat   = `{{ .Datetime }} autosave

Autosaved by {{ .Author }}
{{ .DatetimeVerbose }}
`
	CommitMessageTemplate = template.Must(
		template.New("commit_message").Parse(CommitMessageFormat),
	)
)

type CommitMessageData struct {
	Datetime        string
	Author          string
	DatetimeVerbose string
}

func init() {
	flag.Parse()
	cwd = os.Getenv("NOTES")
	if cwd == "" {
		fmt.Fprintln(os.Stderr, "error: $NOTES is not set")
		os.Exit(1)
	}
	info, ok := debug.ReadBuildInfo()
	if ok {
		PackageName = info.Path
	}
	fmt.Fprintf(os.Stderr, "%s Autosaving notes at: %s\n", time.Now().Format(time.DateTime), cwd)
}

func main() {
	status, err := runGitStatus()
	if err != nil {
		fmt.Fprintln(os.Stderr, "git status failed", err)
		os.Exit(1)
	}

	if !status.HasChanges() {
		fmt.Fprintln(os.Stderr, "no changes to commit: exiting")
		os.Exit(0)
	}

	if *DryRun {
		fmt.Fprintln(os.Stderr, "skipping git-add in dry run")
	} else if err := runGitAddAll(); err != nil {
		fmt.Fprintln(os.Stderr, "git add failed", "error", err)
		os.Exit(1)
	}

	if *DryRun {
		fmt.Fprintln(os.Stderr, "skipping git-commit in dry run")
	} else if err := runGitCommit(); err != nil {
		fmt.Fprintln(os.Stderr, "no changes committed", "error", err)
		os.Exit(1)
	}
}

func runGitCommit() error {
	now := time.Now()
	data := CommitMessageData{
		Datetime:        now.Format(DateTimeLayout),
		DatetimeVerbose: now.Format(DateTimeVerboseLayout),
		Author:          PackageName,
	}
	messageWriter := strings.Builder{}
	if err := CommitMessageTemplate.Execute(&messageWriter, data); err != nil {
		return err
	}

	cmd := exec.Command("git", "commit", "--message", messageWriter.String())
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	fmt.Fprintln(os.Stderr, cmd.String())
	fmt.Fprintln(os.Stderr, string(out))
	if err != nil {
		return err
	}

	return nil
}

func runGitAddAll() error {
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	fmt.Fprintln(os.Stderr, cmd.String())
	fmt.Fprintln(os.Stderr, string(out))
	if err != nil {
		return err
	}

	return nil
}

type gitStatus struct {
	Lines []string
}

func (self *gitStatus) HasChanges() bool {
	return len(self.Lines) > 0
}

func runGitStatus() (*gitStatus, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	fmt.Fprintln(os.Stderr, cmd.String())
	fmt.Fprintln(os.Stderr, string(out))
	if err != nil {
		return nil, err
	}

	status := &gitStatus{}
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		status.Lines = append(status.Lines, line)
	}

	return status, nil
}
