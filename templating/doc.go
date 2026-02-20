// Package templating provides a fast template engine that substitutes
// variables from stamp info files and explicit key-value pairs. It uses
// valyala/fasttemplate with configurable delimiters (default "{{" and "}}").
//
// The Engine type holds configuration (start/end tags, stamp info files) and
// expands templates via the Expand method, which reads a template file,
// applies variable substitution and import expansion, and writes the result.
package templating
