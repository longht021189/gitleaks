package git

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gitleaks/go-gitdiff/gitdiff"
	"github.com/rs/zerolog/log"
	f "github.com/zricethezav/gitleaks/v8/flags"
)

type stdResult struct {
	out io.Reader
	err io.Reader
}

func getCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	listenForStdErr(stderr)

	bytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func getAllCommit(dir, source, target string) ([]string, error) {
	sourceClean := filepath.Clean(dir)
	args := []string{
		"-C", sourceClean,
		"log", "--format=oneline", "--right-only", fmt.Sprintf("%s..%s", target, source)}

	cmd := exec.Command("git", args...)
	log.Debug().Msgf("executing: %s", cmd.String())

	bytes, err := getCmdOutput(cmd)
	if err != nil {
		return nil, err
	}

	var commits []string

	content := string(bytes)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		words := strings.Split(line, " ")
		if words[0] != "" {
			commits = append(commits, words[0])
		}
	}

	return commits, nil
}

func getCommitDetail(dir, commit string) (string, error) {
	sourceClean := filepath.Clean(dir)
	args := []string{"-C", sourceClean, "show", "-p", "-U0", commit}
	cmd := exec.Command("git", args...)
	log.Warn().Msgf("executing: %s", cmd.String())

	bytes, err := getCmdOutput(cmd)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func gitLogGitRequest(source string, flags *f.GitRequestFlags, commitsFile string) (*stdResult, error) {
	var (
		commits []string
		err     error
		enabled = false
	)
	if flags != nil && flags.SourceBranch != "" && flags.TargetBranch != "" {
		commits, err = getAllCommit(source, flags.SourceBranch, flags.TargetBranch)
		if err != nil {
			return nil, err
		}
		enabled = true
	}
	if commitsFile != "" {
		file, err := os.Open(commitsFile)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		bytes, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
		content := string(bytes)
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			words := strings.Split(line, " ")
			if words[0] != "" {
				commits = append(commits, words[0])
			}
		}
		enabled = true
	}
	if !enabled {
		return nil, nil
	}

	log.Warn().Msgf("Commits Count: %d", len(commits))

	changesDetail := ""
	isFirst := true
	for _, c := range commits {
		d, err := getCommitDetail(source, c)
		if err != nil {
			return nil, err
		}
		log.Warn().Msgf("-- Get Commit: %s", c)
		if !isFirst {
			changesDetail += "\n"
		}
		isFirst = false
		changesDetail += d
	}

	return &stdResult{
		err: nil,
		out: strings.NewReader(changesDetail),
	}, nil
}

func gitLog(source string, logOpts string) (*stdResult, error) {
	sourceClean := filepath.Clean(source)
	var cmd *exec.Cmd
	if logOpts != "" {
		args := []string{"-C", sourceClean, "log", "-p", "-U0"}
		args = append(args, strings.Split(logOpts, " ")...)
		cmd = exec.Command("git", args...)
	} else {
		cmd = exec.Command("git", "-C", sourceClean, "log", "-p", "-U0",
			"--full-history", "--all")
	}

	log.Debug().Msgf("executing: %s", cmd.String())

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &stdResult{out: stdout, err: stderr}, nil
}

// GitLog returns a channel of gitdiff.File objects from the
// git log -p command for the given source.
func GitLog(source string, logOpts string) (<-chan *gitdiff.File, error) {
	var (
		std *stdResult
		err error
	)

	std, err = gitLogGitRequest(source, f.GetGitRequestFlags(), f.GetCommitsFlags().File)
	if err != nil {
		return nil, err
	}
	if std == nil {
		std, err = gitLog(source, logOpts)
		if err != nil {
			return nil, err
		}
	}

	go listenForStdErr(std.err)
	// HACK: to avoid https://github.com/zricethezav/gitleaks/issues/722
	time.Sleep(50 * time.Millisecond)

	return gitdiff.Parse(std.out)
}

// GitDiff returns a channel of gitdiff.File objects from
// the git diff command for the given source.
func GitDiff(source string, staged bool) (<-chan *gitdiff.File, error) {
	sourceClean := filepath.Clean(source)
	var cmd *exec.Cmd
	cmd = exec.Command("git", "-C", sourceClean, "diff", "-U0", ".")
	if staged {
		cmd = exec.Command("git", "-C", sourceClean, "diff", "-U0",
			"--staged", ".")
	}
	log.Debug().Msgf("executing: %s", cmd.String())

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go listenForStdErr(stderr)
	// HACK: to avoid https://github.com/zricethezav/gitleaks/issues/722
	time.Sleep(50 * time.Millisecond)

	return gitdiff.Parse(stdout)
}

// listenForStdErr listens for stderr output from git and prints it to stdout
// then exits with exit code 1
func listenForStdErr(stderr io.Reader) {
	if stderr == nil {
		return
	}

	scanner := bufio.NewScanner(stderr)
	errEncountered := false
	for scanner.Scan() {
		// if git throws one of the following errors:
		//
		//  exhaustive rename detection was skipped due to too many files.
		//  you may want to set your diff.renameLimit variable to at least
		//  (some large number) and retry the command.
		//
		//	inexact rename detection was skipped due to too many files.
		//  you may want to set your diff.renameLimit variable to at least
		//  (some large number) and retry the command.
		//
		// we skip exiting the program as git log -p/git diff will continue
		// to send data to stdout and finish executing. This next bit of
		// code prevents gitleaks from stopping mid scan if this error is
		// encountered
		if strings.Contains(scanner.Text(),
			"exhaustive rename detection was skipped") ||
			strings.Contains(scanner.Text(),
				"inexact rename detection was skipped") ||
			strings.Contains(scanner.Text(),
				"you may want to set your diff.renameLimit") {

			log.Warn().Msg(scanner.Text())
		} else {
			log.Error().Msg(scanner.Text())
			errEncountered = true
		}
	}
	if errEncountered {
		os.Exit(1)
	}
}
