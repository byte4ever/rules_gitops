// Package sidecar provides Kubernetes integration test lifecycle helpers that
// watch pods, set up port forwarding, tail logs via stern, and handle graceful
// cleanup. It is used by the it_sidecar CLI binary to manage the test
// environment lifecycle.
package sidecar
