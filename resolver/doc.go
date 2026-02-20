// Package resolver walks multi-document YAML looking for container image
// references and substitutes them with fully-qualified registry URLs from an
// image map. It handles both single-document and multi-document YAML streams
// separated by "---" markers.
package resolver
