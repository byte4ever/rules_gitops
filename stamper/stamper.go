package stamper

import (
	"fmt"
	"os"
	"strings"

	"github.com/valyala/fasttemplate"
)

// LoadStamps reads workspace status files and merges them
// into a single map. Each line is "KEY VALUE" with the
// first space as delimiter. Lines without a space are
// silently skipped.
func LoadStamps(
	infoFiles []string,
) (map[string]interface{}, error) {
	const errCtx = "loading stamps"

	stamps := make(map[string]interface{})

	for _, sf := range infoFiles {
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

// Stamp loads workspace status variables from infoFiles
// and substitutes {VAR} placeholders in format. Unknown
// variables are preserved as-is.
func Stamp(
	infoFiles []string,
	format string,
) (string, error) {
	const errCtx = "stamping"

	stamps, err := LoadStamps(infoFiles)
	if err != nil {
		return "", fmt.Errorf(
			"%s: %w", errCtx, err,
		)
	}

	result := fasttemplate.ExecuteStringStd(
		format, "{", "}", stamps,
	)

	return result, nil
}
