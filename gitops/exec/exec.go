// Package exec provides shell command execution helpers.
package exec

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

// Ex executes the named command in the given directory and
// returns combined stdout+stderr output. Pass empty dir to
// use the current working directory.
func Ex(
	dir string,
	name string,
	arg ...string,
) (string, error) {
	const errCtx = "executing command"

	slog.Info(
		"executing",
		"cmd", name,
		"args", strings.Join(arg, " "),
	)

	cmd := exec.CommandContext(context.Background(), name, arg...)
	if dir != "" {
		cmd.Dir = dir
	}

	by, err := cmd.CombinedOutput()

	slog.Info("output", "result", string(by))

	if err != nil {
		return string(by), fmt.Errorf(
			"%s: %s %s: %w",
			errCtx, name, strings.Join(arg, " "), err,
		)
	}

	return string(by), nil
}

// MustEx executes the command and panics on failure.
func MustEx(dir string, name string, arg ...string) {
	if _, err := Ex(dir, name, arg...); err != nil {
		panic(fmt.Sprintf("command failed: %v", err))
	}
}
