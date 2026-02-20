package sidecar

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// PortForwardConf holds port-forward configuration
// mapping service names to target ports.
type PortForwardConf struct {
	Services map[string][]uint16
}

// Contains returns true if the slice contains the item.
func Contains(v []string, item string) bool {
	for _, s := range v {
		if s == item {
			return true
		}
	}

	return false
}

// ListReadyApps converts a list returned from the pod
// informer store to lists of ready pod names and
// not-ready app names. An app is identified by the "app"
// or "app.kubernetes.io/name" label.
func ListReadyApps(
	list []interface{},
	waitForApps []string,
) (readyPods, notReady []string) {
	var readyApps []string

	for _, it := range list {
		pod, ok := it.(*v1.Pod)
		if !ok {
			panic(
				errors.New(
					"expected pod in informer",
				),
			)
		}

		for _, cond := range pod.Status.Conditions {
			if cond.Type != v1.PodReady {
				continue
			}

			if cond.Status != v1.ConditionTrue {
				continue
			}

			readyPods = append(readyPods, pod.Name)

			if app := pod.GetLabels()["app"]; app != "" {
				readyApps = append(readyApps, app)
			}

			appK8s := pod.GetLabels()["app.kubernetes.io/name"]
			if appK8s != "" {
				readyApps = append(readyApps, appK8s)
			}
		}
	}

	for _, app := range waitForApps {
		if !Contains(readyApps, app) {
			notReady = append(notReady, app)
		}
	}

	return readyPods, notReady
}

// ListReadyServices converts an endpoints informer
// store list to lists of ready and not-ready service
// names.
func ListReadyServices(
	list []interface{},
	pfConfig *PortForwardConf,
) (ready, notReady []string) {
	for _, it := range list {
		ep, ok := it.(*v1.Endpoints)
		if !ok {
			panic(
				errors.New(
					"expected Endpoints in informer",
				),
			)
		}

		for _, subset := range ep.Subsets {
			if len(subset.Addresses) > 0 {
				ready = append(ready, ep.Name)

				break
			}
		}
	}

	for service := range pfConfig.Services {
		if !Contains(ready, service) {
			notReady = append(notReady, service)
		}
	}

	return ready, notReady
}

// WaitForPods uses a shared informer to wait until all
// specified apps have at least one ready pod.
func WaitForPods(
	ctx context.Context,
	clientset *kubernetes.Clientset,
	namespace string,
	waitForApps []string,
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	events := make(chan interface{})
	fn := func(obj interface{}) {
		events <- obj
	}

	handler := &cache.ResourceEventHandlerFuncs{
		AddFunc:    fn,
		DeleteFunc: fn,
		UpdateFunc: func(_, newObj interface{}) {
			fn(newObj)
		},
	}

	factory := informers.NewFilteredSharedInformerFactory(
		clientset, 30*time.Second, namespace, nil,
	)
	podsInformer := factory.Core().
		V1().Pods().Informer()

	//nolint:errcheck,gosec // informer registration
	podsInformer.AddEventHandler(handler)

	go factory.Start(ctx.Done())

	for {
		select {
		case <-events:
			v := podsInformer.GetStore().List()
			readyPods, notReady := ListReadyApps(
				v, waitForApps,
			)

			slog.Info("ready pods", "pods", readyPods)

			if len(notReady) != 0 {
				slog.Info(
					"waiting for apps",
					"apps", notReady,
				)
			} else {
				slog.Info("all apps are ready")

				return nil
			}
		case <-ctx.Done():
			return errors.New(
				"timed out waiting for apps",
			)
		}
	}
}

// WaitForEndpoints uses a shared informer to wait until
// all port-forward services have ready endpoints, then
// sets up port forwarding for each.
func WaitForEndpoints(
	ctx context.Context,
	clientset *kubernetes.Clientset,
	restConfig *rest.Config,
	namespace string,
	pfConfig *PortForwardConf,
) error {
	events := make(chan interface{})
	fn := func(obj interface{}) {
		events <- obj
	}

	handler := &cache.ResourceEventHandlerFuncs{
		AddFunc:    fn,
		DeleteFunc: fn,
		UpdateFunc: func(_, newObj interface{}) {
			fn(newObj)
		},
	}

	factory := informers.NewFilteredSharedInformerFactory(
		clientset, 30*time.Second, namespace, nil,
	)
	endpointsInformer := factory.Core().
		V1().Endpoints().Informer()

	//nolint:errcheck,gosec // informer registration
	endpointsInformer.AddEventHandler(handler)

	go factory.Start(ctx.Done())

	allReady := make(map[string]bool)

	for {
		select {
		case <-events:
			v := endpointsInformer.GetStore().List()
			ready, notReady := ListReadyServices(
				v, pfConfig,
			)

			slog.Info(
				"ready services",
				"services", ready,
			)

			for _, svc := range ready {
				if allReady[svc] {
					continue
				}

				allReady[svc] = true
				slog.Info("SERVICE_READY", "svc", svc)

				ports := pfConfig.Services[svc]
				if len(ports) == 0 {
					continue
				}

				if err := PortForward(
					ctx, clientset,
					restConfig, namespace,
					svc, ports,
				); err != nil {
					return err
				}
			}

			if len(notReady) != 0 {
				slog.Info(
					"waiting for endpoints",
					"endpoints", notReady,
				)
			} else {
				slog.Info("all services are ready")

				return nil
			}
		case <-ctx.Done():
			return errors.New(
				"timed out waiting for services",
			)
		}
	}
}

// PortForward sets up port forwarding from localhost to
// the first available pod backing the named service.
func PortForward(
	ctx context.Context,
	clientset *kubernetes.Clientset,
	restConfig *rest.Config,
	namespace string,
	serviceName string,
	ports []uint16,
) error {
	const errCtx = "port forward"

	var wg sync.WaitGroup

	wg.Add(len(ports))

	for _, port := range ports {
		ep, err := clientset.CoreV1().
			Endpoints(namespace).
			Get(ctx, serviceName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf(
				"%s: listing endpoints for %s: %w",
				errCtx, serviceName, err,
			)
		}

		var podNamespace, podName string

		for _, subset := range ep.Subsets {
			if len(subset.Addresses) == 0 {
				continue
			}

			podNamespace = subset.Addresses[0].
				TargetRef.Namespace
			podName = subset.Addresses[0].
				TargetRef.Name

			break
		}

		if podNamespace == "" || podName == "" {
			return fmt.Errorf(
				"%s: no pods available for %s",
				errCtx, serviceName,
			)
		}

		slog.Info(
			"forwarding",
			"service", serviceName,
			"pod", podNamespace+"/"+podName,
		)

		url := clientset.CoreV1().
			RESTClient().Post().
			Resource("pods").
			Namespace(podNamespace).
			Name(podName).
			SubResource("portforward").
			URL()

		transport, upgrader, err := spdy.RoundTripperFor(restConfig)
		if err != nil {
			return fmt.Errorf(
				"%s: creating round tripper: %w",
				errCtx, err,
			)
		}

		dialer := spdy.NewDialer(
			upgrader,
			&http.Client{Transport: transport},
			"POST",
			url,
		)

		pfPorts := []string{
			fmt.Sprintf(":%d", port),
		}
		readyChan := make(chan struct{}, 1)

		pf, err := portforward.New(
			dialer, pfPorts, ctx.Done(),
			readyChan, os.Stderr, os.Stderr,
		)
		if err != nil {
			return fmt.Errorf(
				"%s: creating port forward: %w",
				errCtx, err,
			)
		}

		go func(targetPort uint16) {
			if fwdErr := pf.ForwardPorts(); fwdErr != nil {
				slog.Error(
					"port forward failed",
					"service", serviceName,
					"port", targetPort,
					"error", fwdErr,
				)
			}
		}(port)

		go func(targetPort uint16) {
			<-pf.Ready

			fwdPorts, err := pf.GetPorts()
			if err != nil {
				slog.Error(
					"getting forwarded ports",
					"service", serviceName,
					"port", targetPort,
					"error", err,
				)

				wg.Done()

				return
			}

			for _, fp := range fwdPorts {
				//nolint:forbidigo // protocol output
				fmt.Printf(
					"FORWARD %s:%d:%d\n",
					serviceName,
					fp.Remote, fp.Local,
				)
			}

			wg.Done()
		}(port)
	}

	wg.Wait()

	return nil
}

// Cleanup performs namespace cleanup. If deleteNamespace
// is true and namespace is non-empty, the namespace is
// deleted.
func Cleanup(
	clientset *kubernetes.Clientset,
	namespace string,
	deleteNamespace bool,
) {
	slog.Info("cleanup")

	if !deleteNamespace || namespace == "" {
		return
	}

	slog.Info("deleting namespace", "namespace", namespace)

	propagation := metav1.DeletePropagationBackground

	err := clientset.CoreV1().
		Namespaces().
		Delete(
			context.Background(),
			namespace,
			metav1.DeleteOptions{
				PropagationPolicy: &propagation,
			},
		)
	if err != nil {
		slog.Error(
			"unable to delete namespace",
			"namespace", namespace,
			"error", err,
		)
	}
}
