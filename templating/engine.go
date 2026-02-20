package templating

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/valyala/fasttemplate"
)

// Engine expands templates using stamp info files and
// explicit variables.
type Engine struct {
	StartTag       string
	EndTag         string
	StampInfoFiles []string
}

// Expand reads a template, substitutes variables, and
// writes the result. If outPath is empty it writes to
// stdout. If executable is true the output file receives
// mode 0777 instead of 0666.
//
// Processing order mirrors the original algorithm:
//  1. Load stamp files into a stamp map.
//  2. For each variable NAME=VALUE, expand VALUE against
//     stamps using single-brace tags, then store as both
//     "NAME" and "variables.NAME" in context.
//  3. For each import NAME=filename, read the file, expand
//     it against context with the configured tags, then
//     expand again against stamps with single-brace tags,
//     and store as "imports.NAME" in context.
//  4. Expand the template against context.
func (en *Engine) Expand(
	tplPath string,
	outPath string,
	vars []string,
	imports []string,
	executable bool,
) error {
	const errCtx = "expanding template"

	stamps, err := en.loadStamps()
	if err != nil {
		return fmt.Errorf("%s: %w", errCtx, err)
	}

	// Stamps form the base context; variables and
	// imports override them.
	ctx := make(map[string]interface{})
	for key, val := range stamps {
		ctx[key] = val
	}

	if err := en.resolveVars(vars, stamps, ctx); err != nil {
		return fmt.Errorf("%s: %w", errCtx, err)
	}

	if err := en.resolveImports(imports, stamps, ctx); err != nil {
		return fmt.Errorf("%s: %w", errCtx, err)
	}

	tplContent, err := en.readTemplate(tplPath)
	if err != nil {
		return fmt.Errorf("%s: %w", errCtx, err)
	}

	startTag, endTag := en.tags()

	out, closer, err := en.openOutput(outPath, executable)
	if err != nil {
		return fmt.Errorf("%s: %w", errCtx, err)
	}

	if closer != nil {
		defer closer()
	}

	_, err = fasttemplate.ExecuteStd(
		string(tplContent), startTag, endTag, out, ctx,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", errCtx, err)
	}

	return nil
}

// tags returns the configured start/end tags, falling
// back to double-brace defaults.
func (en *Engine) tags() (string, string) {
	startTag := en.StartTag
	if startTag == "" {
		startTag = "{{"
	}

	endTag := en.EndTag
	if endTag == "" {
		endTag = "}}"
	}

	return startTag, endTag
}

// loadStamps reads all stamp info files and merges them
// into a single map. Each line is "KEY VALUE" with the
// first space as delimiter.
func (en *Engine) loadStamps() (
	map[string]interface{}, error,
) {
	const errCtx = "loading stamps"

	stamps := make(map[string]interface{})

	for _, sf := range en.StampInfoFiles {
		content, err := os.ReadFile(sf) //nolint:gosec // paths from CLI flags
		if err != nil {
			return nil, fmt.Errorf(
				"%s: %w", errCtx, err,
			)
		}

		for _, line := range strings.Split(
			string(content), "\n",
		) {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				stamps[parts[0]] = parts[1]
			}
		}
	}

	return stamps, nil
}

// resolveVars processes --variable flags. Each variable
// value is expanded against stamps using single-brace
// tags, then stored as both "NAME" and "variables.NAME".
func (en *Engine) resolveVars(
	vars []string,
	stamps map[string]interface{},
	ctx map[string]interface{},
) error {
	const errCtx = "resolving variables"

	for _, vr := range vars {
		parts := strings.SplitN(vr, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf(
				"%s: variable must be VAR=value, got %s",
				errCtx, vr,
			)
		}

		val := fasttemplate.ExecuteStringStd(
			parts[1], "{", "}", stamps,
		)

		ctx[parts[0]] = val
		ctx["variables."+parts[0]] = val
	}

	return nil
}

// resolveImports processes --imports flags. Each import
// file is read, expanded against ctx with the configured
// tags, then expanded against stamps with single-brace
// tags, and stored as "imports.NAME".
func (en *Engine) resolveImports(
	imports []string,
	stamps map[string]interface{},
	ctx map[string]interface{},
) error {
	const errCtx = "resolving imports"

	startTag, endTag := en.tags()

	for _, im := range imports {
		parts := strings.SplitN(im, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf(
				"%s: import must be NAME=filename, got %s",
				errCtx, im,
			)
		}

		content, err := os.ReadFile(parts[1]) //nolint:gosec // paths from CLI flags
		if err != nil {
			return fmt.Errorf(
				"%s: reading %s: %w",
				errCtx, parts[1], err,
			)
		}

		// First pass: expand against context with
		// configured tags.
		val := fasttemplate.ExecuteStringStd(
			string(content), startTag, endTag, ctx,
		)

		// Second pass: expand against stamps with
		// single-brace tags.
		ctx["imports."+parts[0]] = fasttemplate.ExecuteStringStd(
			val, "{", "}", stamps,
		)
	}

	return nil
}

// readTemplate reads the template from a file path. If
// tplPath is empty it reads from stdin.
func (en *Engine) readTemplate(
	tplPath string,
) ([]byte, error) {
	const errCtx = "reading template"

	if tplPath != "" {
		content, err := os.ReadFile(tplPath) //nolint:gosec // paths from CLI flags
		if err != nil {
			return nil, fmt.Errorf(
				"%s: %w", errCtx, err,
			)
		}

		return content, nil
	}

	content, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf(
			"%s: reading stdin: %w", errCtx, err,
		)
	}

	return content, nil
}

// openOutput returns a writer for the result. When
// outPath is empty it returns stdout. The returned
// closer function must be called to finalize the file
// (may be nil for stdout).
func (en *Engine) openOutput(
	outPath string,
	executable bool,
) (io.Writer, func(), error) {
	const errCtx = "opening output"

	if outPath == "" {
		return os.Stdout, nil, nil
	}

	var perm os.FileMode = 0o666
	if executable {
		perm = 0o777
	}

	fi, err := os.OpenFile( //nolint:gosec // paths from CLI flags
		outPath,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC,
		perm,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"%s: %w", errCtx, err,
		)
	}

	return fi, func() {
		_ = fi.Close() //nolint:errcheck // best-effort close
	}, nil
}
