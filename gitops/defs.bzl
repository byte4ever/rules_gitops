"""GitOps rules public interface."""

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
