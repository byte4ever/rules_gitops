// Package client provides a test-infrastructure helper that Go integration
// tests use to orchestrate the it_sidecar process from TestMain. K8STestSetup
// manages the sidecar subprocess lifecycle, waits for readiness, and exposes
// forwarded service ports to test code.
package client
