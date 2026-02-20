"""Public API for rules_gitops.

Provides three rules for Kubernetes GitOps workflows:

- `k8s_deploy`: Main deployment macro that generates kustomize, kubectl, show,
  and gitops targets for a set of Kubernetes manifests.
- `k8s_test_setup`: Rule for setting up Kubernetes integration test environments
  with image pushing, manifest application, pod waiting, and port forwarding.
- `external_image`: Rule for declaring externally-hosted container images (not
  built by Bazel) that should be injected into manifests.
"""

load(
    "//skylib:external_image.bzl",
    _external_image = "external_image",
)
load(
    "//skylib:k8s.bzl",
    _k8s_deploy = "k8s_deploy",
    _k8s_test_setup = "k8s_test_setup",
)

k8s_deploy = _k8s_deploy
k8s_test_setup = _k8s_test_setup
external_image = _external_image
