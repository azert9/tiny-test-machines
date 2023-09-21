package main

import (
	"errors"
	"fmt"
	"github.com/azert9/tiny-test-machines/pkg/protocol"
	"io"
	"os"
	"os/exec"
	"time"
)

type VM struct {
}

func (v *VM) Sleep(req *protocol.SleepRequest, resp *protocol.SleepResponse) error {
	time.Sleep(req.Duration)
	return nil
}

func (v *VM) Exec(req *protocol.ExecRequest, resp *protocol.ExecResponse) error {

	resp.ExitCode = -2

	if len(req.Args) == 0 {
		return fmt.Errorf("not enough arguments")
	}

	cmd := exec.Command(req.Args[0], req.Args[1:]...)

	// we do not have /dev/null, so we use the currently opened file
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	waited := false
	if err := cmd.Start(); err != nil {
		return err
	}
	defer func() {
		if !waited {
			_ = cmd.Wait()
		}
	}()

	stdout, err := io.ReadAll(stdoutPipe)
	if err != nil {
		return err
	}

	resp.Stdout = stdout

	waited = true
	processErr := cmd.Wait()
	if processErr != nil {
		var exitErr *exec.ExitError
		if errors.As(processErr, &exitErr) {
			resp.ExitCode = exitErr.ExitCode()
		} else {
			return exitErr
		}
	} else {
		resp.ExitCode = 0
	}

	return nil
}
