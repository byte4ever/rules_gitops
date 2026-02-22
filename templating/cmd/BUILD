load("@rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "cmd_lib",
    srcs = ["main.go"],
    importpath = "github.com/byte4ever/rules_gitops/templating/cmd",
    visibility = ["//visibility:private"],
    deps = ["//templating"],
)

go_binary(
    name = "templating",
    embed = [":cmd_lib"],
    visibility = ["//visibility:public"],
)
