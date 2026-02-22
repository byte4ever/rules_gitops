load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "it_sidecar",
    srcs = [
        "doc.go",
        "sidecar.go",
    ],
    importpath = "github.com/byte4ever/rules_gitops/testing/it_sidecar",
    visibility = ["//visibility:public"],
    deps = [
        "@io_k8s_api//core/v1:core",
        "@io_k8s_apimachinery//pkg/apis/meta/v1:meta",
        "@io_k8s_client_go//informers",
        "@io_k8s_client_go//kubernetes",
        "@io_k8s_client_go//rest",
        "@io_k8s_client_go//tools/cache",
        "@io_k8s_client_go//tools/portforward",
        "@io_k8s_client_go//transport/spdy",
    ],
)
