// Package main provides the stamper CLI that reads Bazel
// workspace status files and substitutes {VAR} placeholders
// in a format string or file.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/byte4ever/rules_gitops/stamper"
)

type arrayFlags []string

func (af *arrayFlags) String() string {
	return ""
}

func (af *arrayFlags) Set(value string) error {
	*af = append(*af, value)
	return nil
}

func run() error {
	const errCtx = "stamper"

	var stampInfoFiles arrayFlags

	var (
		output     string
		format     string
		formatFile string
	)

	flag.Var(
		&stampInfoFiles,
		"stamp-info-file",
		"path to workspace status file (repeatable)",
	)

	flag.StringVar(
		&output, "output", "",
		"output file path (default: stdout)",
	)

	flag.StringVar(
		&formatFile, "format-file", "",
		"file containing stamp variable placeholders",
	)

	flag.StringVar(
		&format, "format", "",
		"format string containing stamp variables",
	)

	flag.Parse()

	if formatFile != "" && format != "" {
		return fmt.Errorf(
			"%s: only one of --format or"+
				" --format-file may be specified",
			errCtx,
		)
	}

	if formatFile != "" {
		content, err := os.ReadFile( //nolint:gosec // path from CLI flag
			formatFile,
		)
		if err != nil {
			return fmt.Errorf(
				"%s: reading format file: %w",
				errCtx, err,
			)
		}

		format = string(content)
	}

	result, err := stamper.Stamp(stampInfoFiles, format)
	if err != nil {
		return fmt.Errorf("%s: %w", errCtx, err)
	}

	if output != "" {
		err = os.WriteFile( //nolint:gosec // path from CLI flag
			output, []byte(result), 0o666,
		)
		if err != nil {
			return fmt.Errorf(
				"%s: writing output: %w",
				errCtx, err,
			)
		}

		return nil
	}

	_, err = os.Stdout.WriteString(result)
	if err != nil {
		return fmt.Errorf(
			"%s: writing to stdout: %w",
			errCtx, err,
		)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
