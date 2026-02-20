// Package commitmsg generates and parses gitops target lists
// embedded in git commit messages.
package commitmsg

import (
	"log"
	"strings"
)

const (
	begin = "--- gitops targets begin ---"
	end   = "--- gitops targets end ---"
)

// ExtractTargets extracts the list of gitops targets from
// a commit message delimited by begin/end markers.
func ExtractTargets(msg string) []string {
	var targets []string

	betweenMarkers := false

	for _, line := range strings.Split(msg, "\n") {
		switch line {
		case begin:
			betweenMarkers = true
		case end:
			betweenMarkers = false
		default:
			if betweenMarkers {
				targets = append(targets, line)
			}
		}
	}

	if betweenMarkers {
		log.Print("unable to find end marker in commit message")

		return nil
	}

	return targets
}

// Generate produces a commit message section containing the
// given list of gitops targets between begin/end markers.
func Generate(targets []string) string {
	var sb strings.Builder

	sb.WriteByte('\n')
	sb.WriteString(begin)
	sb.WriteByte('\n')

	for _, t := range targets {
		sb.WriteString(t)
		sb.WriteByte('\n')
	}

	sb.WriteString(end)
	sb.WriteByte('\n')

	return sb.String()
}
