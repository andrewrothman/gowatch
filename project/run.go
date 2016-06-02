package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	gwl "github.com/andrewrothman/gowatch/log"
)

type ExecuteHandle struct {
	sync.Mutex
	projectDirectory string
	result           chan StepResult
	halted           bool
	cmd              *exec.Cmd
	running          bool
	errorCode        error
}

func (h *ExecuteHandle) Running() bool {
	h.Lock()
	defer h.Unlock()
	return h.running
}

func (h *ExecuteHandle) Error() StepResult {
	if h.running {
		return <-h.result
	} else {
		return h.errorCode
	}
}

// Kill kills the underlying application if its started
func (h *ExecuteHandle) Kill(reason StepResult) {

	if reason == nil {
		reason = ErrorAppKilled
	}

	gwl.LogDebug("hitting kill lock")
	h.Lock()
	gwl.LogDebug("done with kill lock")
	if h.running {
		cmd := h.cmd
		proc := cmd.Process

		gwl.LogDebug("Killing")

		if proc != nil {
			if err := proc.Kill(); err != nil && err.Error() != errorProcessAlreadyFinished.Error() {
				gwl.LogDebug("process didn't seem to exit gracefully", err)
				reason = err
			}
		}

		h.writeError(reason)
		h.errorCode = reason
		h.running = false
		h.halted = true
		close(h.result)
	} else if h.errorCode == nil {
		gwl.LogDebug("process never started %s", reason.Error())
		h.writeError(reason)
	}

	h.Unlock()
}

func (h *ExecuteHandle) writeError(reason StepResult) {
	if h.running {
		gwl.LogDebug("sending error")
		h.result <- reason
	} else {
		h.errorCode = reason
	}
}

func (h *ExecuteHandle) Halted() bool {
	h.Lock()
	defer h.Unlock()
	return h.halted
}

func (h *ExecuteHandle) start(cmd *exec.Cmd) {
	h.Lock()
	h.cmd = cmd
	err := cmd.Start()
	h.running = true
	h.Unlock()

	if err != nil {
		h.Kill(err)
	}

	waiter := make(chan bool)
	go func() {
		close(waiter)
		if err := cmd.Wait(); err != nil {
			gwl.LogDebug("app exited prematurely")
			h.Kill(err)
		}
	}()
	<-waiter
}

func run(projectDirectory, programName string, arguments string) *exec.Cmd {
	command := ""

	if programName != "" {
		command = programName
	} else {
		_, command = filepath.Split(projectDirectory)
	}

	cmd := exec.Command("./"+command, arguments)

	cmd.Dir = projectDirectory
	cmd.Env = os.Environ()

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd
}
