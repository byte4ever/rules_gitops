# stern

Package `stern` provides pod log tailing for Kubernetes integration tests.
It watches pods in a namespace, follows container logs in real time, and
handles container lifecycle transitions. Ported from the
[stern](https://github.com/stern/stern) project.

## Key Types

### ContainerState

```go
type ContainerState string

const (
    Running    ContainerState = "running"
    Waiting    ContainerState = "waiting"
    Terminated ContainerState = "terminated"
)
```

Represents the lifecycle state of a container. Used to filter which
containers to tail based on their current state.

### Target

```go
type Target struct {
    Namespace string
    Pod       string
    Container string
}
```

Identifies a specific container within a pod. `GetID()` returns a unique
`namespace-pod-container` string used as a map key for tracking active tails.

### Tail

```go
type Tail struct {
    Namespace     string
    PodName       string
    ContainerName string
}
```

Follows the log stream of a single container. `Start` opens a streaming
log request with `Follow: true` and `Timestamps: true`, then prints each
line to stderr prefixed with `[pod/container]`. `Close` terminates the
stream.

## Functions

```go
func Run(ctx context.Context, namespace string, clientset *kubernetes.Clientset) error
func Watch(ctx context.Context, i v1.PodInterface, podFilter *regexp.Regexp, containerFilter *regexp.Regexp, containerState ContainerState, labelSelector labels.Selector) (chan *Target, chan *Target, error)
```

`Run` is the high-level entry point. It watches all pods in the namespace
(matching `.*` for both pod and container names), creates a `Tail` for each
running container that appears, and removes tails when containers are
deleted. It blocks until the context is cancelled.

`Watch` is the lower-level watcher that returns added/removed target
channels. It accepts regex filters for pod and container names, a
container state filter, and a label selector.

## License

This package is derived from the stern project and is licensed under the
Apache License, Version 2.0. See the license headers in each source file
for details.
