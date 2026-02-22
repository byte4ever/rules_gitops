load("@rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "cmd_lib",
    srcs = ["main.go"],
    importpath = "github.com/byte4ever/rules_gitops/testing/it_sidecar/cmd",
    visibility = ["//visibility:private"],
    deps = [
        "//testing/it_sidecar",
        "//testing/it_sidecar/stern",
        "@io_k8s_client_go//kubernetes",
        "@io_k8s_client_go//plugin/pkg/client/auth/gcp",
        "@io_k8s_client_go//tools/clientcmd",
        "@io_k8s_client_go//util/homedir",
    ],
)

go_binary(
    name = "it_sidecar",
    embed = [":cmd_lib"],
    visibility = ["//visibility:public"],
)
