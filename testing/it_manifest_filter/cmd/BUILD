load("@rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "cmd_lib",
    srcs = ["main.go"],
    importpath = "github.com/byte4ever/rules_gitops/testing/it_manifest_filter/cmd",
    visibility = ["//visibility:private"],
    deps = ["//testing/it_manifest_filter"],
)

go_binary(
    name = "it_manifest_filter",
    embed = [":cmd_lib"],
    visibility = ["//visibility:public"],
)
