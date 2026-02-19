// Package main provides the it_sidecar CLI that
// watches pods, sets up port forwarding, tails logs,
// and handles graceful shutdown for integration tests.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	sidecar "github.com/byte4ever/rules_gitops/testing/it_sidecar"
	"github.com/byte4ever/rules_gitops/testing/it_sidecar/stern"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// config holds all CLI parameters for the sidecar.
type config struct {
	namespace       string
	timeout         time.Duration
	deleteNamespace bool
	kubeconfig      string
	portForwards    map[string][]uint16
	waitForApps     []string
}

// portForwardFlag implements flag.Value for collecting
// port-forward arguments in the form service:port.
type portForwardFlag struct {
	services map[string][]uint16
}

// arrayFlag implements flag.Value for collecting
// repeated string flags.
type arrayFlag []string

func (f *portForwardFlag) String() string {
	return fmt.Sprintf("%v", f.services)
}

func (f *portForwardFlag) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)

	const expectedParts = 2
	if len(parts) != expectedParts {
		return fmt.Errorf(
			"incorrect portforward %q:"+
				" must be service:port",
			value,
		)
	}

	port, err := strconv.ParseUint(
		parts[1], 10, 16,
	)
	if err != nil {
		return fmt.Errorf(
			"incorrect port in portforward %q: %w",
			value, err,
		)
	}

	f.services[parts[0]] = append(
		f.services[parts[0]], uint16(port),
	)

	return nil
}

func (f *arrayFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *arrayFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func parseConfig() (*config, error) {
	const errCtx = "parse config"

	var (
		pfFlag = portForwardFlag{
			services: make(map[string][]uint16),
		}
		waitForApps arrayFlag
		namespace   string
		timeout     time.Duration
		deleteNS    bool
		kubeconfig  string
	)

	flag.StringVar(
		&namespace,
		"namespace",
		os.Getenv("NAMESPACE"),
		"kubernetes namespace",
	)
	flag.DurationVar(
		&timeout,
		"timeout",
		30*time.Second,
		"execution timeout",
	)
	flag.BoolVar(
		&deleteNS,
		"delete_namespace", false,
		"delete namespace during cleanup",
	)
	flag.StringVar(
		&kubeconfig,
		"kubeconfig",
		os.Getenv("KUBECONFIG"),
		"path to kubernetes config file",
	)
	flag.Var(
		&pfFlag,
		"portforward",
		"port forward in form service:port",
	)
	flag.Var(
		&waitForApps,
		"waitforapp",
		"wait for pods with label app=<value>",
	)

	flag.Parse()

	if namespace == "" {
		return nil, fmt.Errorf(
			"%s: namespace is required", errCtx,
		)
	}

	return &config{
		namespace:       namespace,
		timeout:         timeout,
		deleteNamespace: deleteNS,
		kubeconfig:      kubeconfig,
		portForwards:    pfFlag.services,
		waitForApps:     waitForApps,
	}, nil
}

func run() error {
	const errCtx = "it_sidecar"

	cfg, err := parseConfig()
	if err != nil {
		return fmt.Errorf("%s: %w", errCtx, err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(), cfg.timeout,
	)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	defer func() {
		signal.Stop(sigCh)
		cancel()
	}()

	// Cancel context on signal.
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	// Cancel context when stdin closes.
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			_, _, readErr := reader.ReadRune()
			if readErr != nil && readErr == io.EOF {
				cancel()

				return
			}
		}
	}()

	kubeconfig := cfg.kubeconfig
	if kubeconfig == "" {
		if _, ok := os.LookupEnv(
			"KUBERNETES_SERVICE_HOST",
		); !ok {
			kubeconfig = filepath.Join(
				homedir.HomeDir(),
				".kube", "config",
			)
		}
	}

	restConfig, err := clientcmd.BuildConfigFromFlags(
		"", kubeconfig,
	)
	if err != nil {
		return fmt.Errorf(
			"%s: building kubeconfig: %w",
			errCtx, err,
		)
	}

	clientset := kubernetes.NewForConfigOrDie(restConfig)

	defer sidecar.Cleanup(
		clientset, cfg.namespace, cfg.deleteNamespace,
	)

	go func() {
		if sternErr := stern.Run(
			ctx, cfg.namespace, clientset,
		); sternErr != nil {
			slog.Error(
				"stern failed", "error", sternErr,
			)
		}
	}()

	if len(cfg.waitForApps) > 0 {
		if err = sidecar.WaitForPods(
			ctx, clientset,
			cfg.namespace, cfg.waitForApps,
		); err != nil {
			return fmt.Errorf("%s: %w", errCtx, err)
		}
	}

	if len(cfg.portForwards) > 0 {
		pfConf := &sidecar.PortForwardConf{
			Services: cfg.portForwards,
		}

		if err = sidecar.WaitForEndpoints(
			ctx, clientset,
			restConfig, cfg.namespace, pfConf,
		); err != nil {
			return fmt.Errorf("%s: %w", errCtx, err)
		}
	}

	//nolint:forbidigo // protocol output
	fmt.Println("READY")
	<-ctx.Done()

	return nil
}

func main() {
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
