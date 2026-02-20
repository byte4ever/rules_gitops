package bazel

import "strings"

// TargetToExecutable converts a Bazel target label like
// //pkg:name to the corresponding bazel-bin executable
// path. Non-label inputs are returned unchanged.
func TargetToExecutable(target string) string {
	if !strings.HasPrefix(target, "//") {
		return target
	}

	tg := "bazel-bin/" + target[2:]
	tg = strings.Replace(tg, ":", "/", 1)

	return tg
}
