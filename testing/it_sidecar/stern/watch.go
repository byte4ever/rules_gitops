// Copyright 2016 Wercker Holding BV
//
// Licensed under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in
// compliance with the License. You may obtain a copy of
// the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in
// writing, software distributed under the License is
// distributed on an "AS IS" BASIS, WITHOUT WARRANTIES
// OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing
// permissions and limitations under the License.

package stern

import (
	"context"
	"fmt"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// Target is a target to watch.
type Target struct {
	Namespace string
	Pod       string
	Container string
}

// GetID returns a unique identifier for the target.
func (t *Target) GetID() string {
	return fmt.Sprintf(
		"%s-%s-%s",
		t.Namespace, t.Pod, t.Container,
	)
}

// Watch starts listening to Kubernetes events and emits
// modified containers/pods. The first channel receives
// added targets, the second receives removed targets.
func Watch(
	ctx context.Context,
	i v1.PodInterface,
	podFilter *regexp.Regexp,
	containerFilter *regexp.Regexp,
	containerState ContainerState,
	labelSelector labels.Selector,
) (chan *Target, chan *Target, error) {
	watcher, err := i.Watch(
		ctx,
		metav1.ListOptions{
			Watch:         true,
			LabelSelector: labelSelector.String(),
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"failed to set up watch: %w", err,
		)
	}

	added := make(chan *Target)
	removed := make(chan *Target)

	go func() {
		for {
			select {
			case event := <-watcher.ResultChan():
				if event.Object == nil {
					return
				}

				pod, ok := event.Object.(*corev1.Pod)
				if !ok {
					continue
				}

				if !podFilter.MatchString(pod.Name) {
					continue
				}

				handleEvent(
					event, pod,
					containerFilter,
					containerState,
					added, removed,
				)
			case <-ctx.Done():
				watcher.Stop()
				close(added)
				close(removed)

				return
			}
		}
	}()

	return added, removed, nil
}

func handleEvent(
	event watch.Event,
	pod *corev1.Pod,
	containerFilter *regexp.Regexp,
	containerState ContainerState,
	added, removed chan *Target,
) {
	switch event.Type {
	case watch.Added, watch.Modified:
		var statuses []corev1.ContainerStatus
		statuses = append(
			statuses,
			pod.Status.InitContainerStatuses...,
		)
		statuses = append(
			statuses,
			pod.Status.ContainerStatuses...,
		)

		for _, status := range statuses {
			if !containerFilter.MatchString(status.Name) {
				continue
			}

			if containerState.Match(status.State) {
				added <- &Target{
					Namespace: pod.Namespace,
					Pod:       pod.Name,
					Container: status.Name,
				}
			}
		}
	case watch.Deleted:
		var containers []corev1.Container
		containers = append(
			containers,
			pod.Spec.Containers...,
		)
		containers = append(
			containers,
			pod.Spec.InitContainers...,
		)

		for _, container := range containers {
			if !containerFilter.MatchString(container.Name) {
				continue
			}

			removed <- &Target{
				Namespace: pod.Namespace,
				Pod:       pod.Name,
				Container: container.Name,
			}
		}
	}
}
