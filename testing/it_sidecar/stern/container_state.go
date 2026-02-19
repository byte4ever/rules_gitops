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
	"errors"

	v1 "k8s.io/api/core/v1"
)

// ContainerState represents the state of a container.
type ContainerState string

const (
	// Running indicates a running container.
	Running ContainerState = "running"
	// Waiting indicates a waiting container.
	Waiting ContainerState = "waiting"
	// Terminated indicates a terminated container.
	Terminated ContainerState = "terminated"
)

// ErrInvalidContainerState is returned when the state
// is not one of running, waiting, or terminated.
var ErrInvalidContainerState = errors.New(
	"containerState should be one of" +
		" 'running', 'waiting', or 'terminated'",
)

// NewContainerState validates and returns a
// ContainerState from the given string.
func NewContainerState(
	stateConfig string,
) (ContainerState, error) {
	switch ContainerState(stateConfig) {
	case Running:
		return Running, nil
	case Waiting:
		return Waiting, nil
	case Terminated:
		return Terminated, nil
	default:
		return "", ErrInvalidContainerState
	}
}

// Match returns true if the ContainerState matches the
// given Kubernetes container state.
func (s ContainerState) Match(
	containerState v1.ContainerState,
) bool {
	return (s == Running &&
		containerState.Running != nil) ||
		(s == Waiting &&
			containerState.Waiting != nil) ||
		(s == Terminated &&
			containerState.Terminated != nil)
}
