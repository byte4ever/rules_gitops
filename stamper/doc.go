// Package stamper reads Bazel workspace status files and substitutes
// single-brace {VAR} placeholders in format strings. LoadStamps parses one or
// more status files into a variable map; Stamp combines loading and
// substitution in a single call.
package stamper
