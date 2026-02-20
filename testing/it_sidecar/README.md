# it_sidecar

Package `sidecar` provides Kubernetes integration test lifecycle helpers.
It watches pods until they are ready, sets up port forwarding to services,
tails container logs via the `stern` sub-package, and handles graceful
cleanup.

## Lifecycle

The sidecar process follows this sequence:

1. Connect to the cluster using kubeconfig or in-cluster config.
2. Start stern log tailing for all pods in the namespace.
3. Watch pods via a shared informer until all `--waitforapp` apps have at
   least one ready pod (matched by `app` or `app.kubernetes.io/name` label).
4. Watch endpoints via a shared informer until all `--portforward` services
   have ready endpoints, then establish port forwarding for each.
5. Print `FORWARD <service>:<remotePort>:<localPort>` lines for each
   forwarded port.
6. Print `READY` to stdout.
7. Block until the context is cancelled (timeout, signal, or stdin EOF).
8. Run cleanup (optionally delete the namespace).

The client package communicates with this process via stdin/stdout pipes,
not HTTP. Closing stdin signals the sidecar to shut down.

## API

```go
type PortForwardConf struct {
    Services map[string][]uint16
}

func Contains(v []string, item string) bool
func ListReadyApps(list []interface{}, waitForApps []string) (readyPods, notReady []string)
func ListReadyServices(list []interface{}, pfConfig *PortForwardConf) (ready, notReady []string)
func WaitForPods(ctx context.Context, clientset *kubernetes.Clientset, namespace string, waitForApps []string) error
func WaitForEndpoints(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, namespace string, pfConfig *PortForwardConf) error
func PortForward(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, namespace string, serviceName string, ports []uint16) error
func Cleanup(clientset *kubernetes.Clientset, namespace string, deleteNamespace bool)
```

## CLI

The binary at `testing/it_sidecar/cmd/` runs the full lifecycle described
above.

```
it_sidecar --namespace <ns> [flags]
```

| Flag | Default | Env | Description |
|------|---------|-----|-------------|
| `--namespace` | | `NAMESPACE` | Kubernetes namespace (required) |
| `--timeout` | `30s` | | Execution timeout |
| `--delete_namespace` | `false` | | Delete namespace during cleanup |
| `--kubeconfig` | `~/.kube/config` | `KUBECONFIG` | Path to kubeconfig file |
| `--portforward` | | | Port forward as `service:port` (repeatable) |
| `--waitforapp` | | | Wait for pods with `app=<value>` label (repeatable) |

When running in-cluster (`KUBERNETES_SERVICE_HOST` is set), the kubeconfig
flag is ignored and in-cluster config is used instead.
