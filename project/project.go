package project

import (
	"sync"

	gwl "github.com/andrewrothman/gowatch/log"
)

func ExecuteBuildSteps(projectDirectory, outputName string, appArguments string, shouldTest bool, shouldLint bool) *ExecuteHandle {

	handle := &ExecuteHandle{
		sync.Mutex{},
		projectDirectory,
		make(chan StepResult, 1),
		false,
		nil,
		false,
		nil,
	}

	if !build(projectDirectory, outputName) {
		handle.Kill(ErrorBuildFailed)
	} else if shouldLint && !lint(projectDirectory) {
		handle.Kill(ErrorLintFailed)
	} else if shouldTest && !test(projectDirectory) {
		handle.Kill(ErrorTestFailed)
	} else {
		handle.start(run(projectDirectory, outputName, appArguments))
	}

	gwl.LogDebug("[DEBUG] build steps completed")

	return handle
}
