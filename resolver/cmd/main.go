// Package main provides the resolver CLI that reads
// multi-document YAML, substitutes container image
// references using an image map, and writes the result.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/byte4ever/rules_gitops/resolver"
)

type imagesFlags map[string]string

func (im *imagesFlags) String() string {
	return fmt.Sprintf("%v", *im)
}

func (im *imagesFlags) Set(value string) error {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return errors.New(
			"image flag must be imagename=imagevalue",
		)
	}

	(*im)[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])

	return nil
}

func run() error {
	const errCtx = "resolver"

	var (
		inFile  string
		outFile string
	)

	images := make(imagesFlags)

	flag.StringVar(
		&inFile, "infile", "",
		"input YAML file path",
	)

	flag.StringVar(
		&outFile, "outfile", "",
		"output YAML file path",
	)

	flag.Var(
		&images, "image",
		"imagename=imagevalue (repeatable)",
	)

	flag.Parse()

	inReader := os.Stdin

	if inFile != "" {
		fi, err := os.Open(inFile) //nolint:gosec // path from CLI flag
		if err != nil {
			return fmt.Errorf(
				"%s: opening input: %w",
				errCtx, err,
			)
		}

		defer fi.Close() //nolint:errcheck // best-effort close

		inReader = fi
	}

	outWriter := os.Stdout

	if outFile != "" {
		fo, err := os.Create(outFile) //nolint:gosec // path from CLI flag
		if err != nil {
			return fmt.Errorf(
				"%s: creating output: %w",
				errCtx, err,
			)
		}

		defer fo.Close() //nolint:errcheck // best-effort close

		outWriter = fo
	}

	if err := resolver.ResolveImages(
		inReader, outWriter, images,
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
