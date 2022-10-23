package executor

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"tart/helper"
	"tart/network"
	"time"

	"github.com/fatih/color"
)

type Build struct {
	job        network.RequestJobResp
	workingDir string
}

type BuildOpt struct {
	Job        network.RequestJobResp
	WorkingDir string
}

func NewBuild(opt BuildOpt) (b *Build, err error) {
	if opt.WorkingDir == "" {
		err = errors.New("working directory can not be empty")
		return
	}

	b = &Build{
		job:        opt.Job,
		workingDir: opt.WorkingDir,
	}
	return
}

func (b *Build) repoURL() string {
	return b.job.GitInfo.RepoURL
}

func (b *Build) PrepareScript(w io.Writer) (err error) {
	_, err = io.WriteString(w, "set -euo pipefail\n")
	if err != nil {
		return
	}

	_, err = io.WriteString(w, fmt.Sprintf("git clone -b %s --single-branch --depth %d %s %s\n",
		b.job.GitInfo.Ref,
		b.job.GitInfo.Depth,
		b.job.GitInfo.RepoURL,
		b.workingDir,
	))
	if err != nil {
		err = fmt.Errorf("wrting to writer: %w", err)
		return
	}

	return
}

func (b *Build) BuildScript(w io.Writer) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("wrting to writer: %w", err)
		}
	}()

	_, err = io.WriteString(w, "set -euo pipefail\n")
	if err != nil {
		return
	}

	_, err = fmt.Fprintf(w, "cd ~/%s\n", b.workingDir)
	if err != nil {
		return
	}

	for _, env := range b.job.Variables {
		_, err = fmt.Fprintf(w, "export %s=%s\n", env.Key, helper.ShellEscape(env.Value))
		if err != nil {
			return
		}
	}

	// run user script
	for idx, step := range b.job.Steps {
		if step.When != "on_success" {
			err = fmt.Errorf("tart only support on_success step, got %q on step %q", step.When, step.Name)
			return
		}

		_, err = io.WriteString(w, "set +x\n")
		if err != nil {
			return
		}

		stepName := step.Name
		if stepName == "" {
			stepName = strconv.Itoa(idx)
		}
		_, err = fmt.Fprintf(w, "echo %s\n", color.BlueString("running step %s...", stepName))
		if err != nil {
			return
		}

		_, err = io.WriteString(w, "set -x\n")
		if err != nil {
			return
		}

		for _, script := range step.Script {
			_, err = io.WriteString(w, script)
			if err != nil {
				return
			}
			_, err = io.WriteString(w, "\n")
			if err != nil {
				return
			}
		}
	}

	return
}

func (b *Build) Timeout() time.Duration {
	var timeout int
	for _, step := range b.job.Steps {
		if step.When != "on_success" {
			continue
		}
		timeout += step.Timeout
	}

	return time.Duration(timeout) * time.Second
}
