// Package digester calculates and verifies SHA256 file digests. It stores
// digests in companion .digest files alongside the original, enabling
// skip-if-unchanged optimizations for image pushes.
package digester
