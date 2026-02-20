# client

Package `client` provides a Go test helper for orchestrating the
`it_sidecar` process from `TestMain`. It manages the sidecar subprocess
lifecycle, waits for readiness, and exposes forwarded service ports to
test code.

## Usage

Wire up `K8STestSetup` in your test file's `TestMain`:

```go
package myapp_test

import (
    "testing"

    "github.com/byte4ever/rules_gitops/testing/it_sidecar/client"
)

var setup = &client.K8STestSetup{
    WaitForPods:         []string{"myapp"},
    PortForwardServices: map[string]int{"myapp-svc": 8080},
}

func TestMain(m *testing.M) {
    setup.TestMain(m)
}

func TestEndpoint(t *testing.T) {
    port := setup.GetServiceLocalPort("myapp-svc")
    // use port to connect to the forwarded service
}
```

Run the test with the `-setup` flag pointing to the `it_sidecar` binary:

```sh
go test -setup /path/to/it_sidecar -v ./...
```

## API

```go
type Callback func() error

type K8STestSetup struct {
    WaitForPods         []string
    PortForwardServices map[string]int
    ReadyCallback       Callback
}

func (s *K8STestSetup) TestMain(m *testing.M)
func (s *K8STestSetup) GetServiceLocalPort(serviceName string) int
```

## How It Works

`TestMain` starts the `it_sidecar` binary as a subprocess, connecting via
stdin/stdout pipes (not HTTP).

1. The sidecar binary path comes from the `-setup` test flag.
2. `K8STestSetup` passes `-waitforapp` and `-portforward` flags to the
   subprocess based on `WaitForPods` and `PortForwardServices`.
3. It reads stdout line by line, parsing `FORWARD <service>:<remote>:<local>`
   lines to record the local port assigned to each service.
4. When the sidecar prints `READY`, the setup phase is complete.
5. If `ReadyCallback` is set, it is invoked before tests run.
6. `m.Run()` executes the test suite.
7. On completion, closing stdin signals the sidecar to tear down.

`GetServiceLocalPort` returns the locally-assigned port for a given service
name, as reported by the sidecar's `FORWARD` output.
