// Package stern provides pod log tailing for Kubernetes integration tests.
// It watches pods matching a target pattern, follows container logs in real
// time, and handles container state transitions. Ported from the stern
// project.
package stern
