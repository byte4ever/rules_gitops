load("@rules_gitops//gitops:defs.bzl", "k8s_deploy")
load("@rules_go//go:def.bzl", "go_binary", "go_library")
load("@rules_oci//oci:defs.bzl", "oci_image")
load("@rules_pkg//pkg:tar.bzl", "pkg_tar")

go_library(
    name = "helloworld_lib",
    srcs = ["helloworld.go"],
    importpath = "github.com/byte4ever/rules_gitops/examples/helloworld",
    visibility = ["//visibility:private"],
)

go_binary(
    name = "helloworld",
    embed = [":helloworld_lib"],
    goarch = "amd64",
    goos = "linux",
    visibility = ["//visibility:public"],
)

pkg_tar(
    name = "helloworld_layer",
    srcs = [":helloworld"],
)

oci_image(
    name = "image",
    base = "@distroless_static",
    entrypoint = ["/helloworld"],
    tars = [":helloworld_layer"],
    visibility = ["//visibility:public"],
)

CLUSTER = "kind-kind"
USER = "kind-kind"
REGISTRY = "localhost:15000"

k8s_deploy(
    name = "mynamespace",
    cluster = CLUSTER,
    image_registry = REGISTRY,
    images = {
        "helloworld-image": ":image",
    },
    manifests = [
        "deployment.yaml",
        "service.yaml",
    ],
    namespace = "{BUILD_USER}",
    user = USER,
)

NAMESPACE = "hwteam"

k8s_deploy(
    name = "canary",
    cluster = CLUSTER,
    deployment_branch = "helloworld-canary",
    image_digest_tag = True,
    image_registry = REGISTRY,
    image_repository_prefix = "k8s",
    images = {
        "helloworld-image": ":image",
    },
    manifests = [
        "deployment.yaml",
        "service.yaml",
    ],
    name_suffix = "-canary",
    namespace = NAMESPACE,
    prefix_suffix_app_labels = True,
    user = USER,
)

k8s_deploy(
    name = "release",
    cluster = CLUSTER,
    deployment_branch = "helloworld-prod",
    image_digest_tag = True,
    image_registry = REGISTRY,
    image_repository_prefix = "k8s",
    images = {
        "helloworld-image": ":image",
    },
    manifests = [
        "deployment.yaml",
        "service.yaml",
    ],
    namespace = NAMESPACE,
    tags = ["release"],
    user = USER,
)
