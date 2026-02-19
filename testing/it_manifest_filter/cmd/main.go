// Package main provides the it_manifest_filter CLI
// that reads multi-document YAML Kubernetes manifests
// and transforms them for integration testing.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	filter "github.com/byte4ever/rules_gitops/testing/it_manifest_filter"
)

func run() error {
	const errCtx = "it_manifest_filter"

	var (
		inFile  string
		outFile string
	)

	flag.StringVar(
		&inFile, "infile", "",
		"input YAML file path",
	)

	flag.StringVar(
		&outFile, "outfile", "",
		"output YAML file path",
	)

	flag.Parse()

	inReader := os.Stdin

	if inFile != "" {
		//nolint:gosec // path from CLI flag
		fi, err := os.Open(inFile)
		if err != nil {
			return fmt.Errorf(
				"%s: opening input: %w",
				errCtx, err,
			)
		}

		//nolint:errcheck // best-effort close
		defer fi.Close()

		inReader = fi
	}

	outWriter := os.Stdout

	if outFile != "" {
		//nolint:gosec // path from CLI flag
		fo, err := os.Create(outFile)
		if err != nil {
			return fmt.Errorf(
				"%s: creating output: %w",
				errCtx, err,
			)
		}

		//nolint:errcheck // best-effort close
		defer fo.Close()

		outWriter = fo
	}

	if err := filter.ReplacePDWithEmptyDirs(
		inReader, outWriter,
	); err != nil {
		return fmt.Errorf("%s: %w", errCtx, err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
