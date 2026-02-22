load("@rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "cmd_lib",
    srcs = ["main.go"],
    importpath = "github.com/byte4ever/rules_gitops/stamper/cmd",
    visibility = ["//visibility:private"],
    deps = ["//stamper"],
)

go_binary(
    name = "stamper",
    embed = [":cmd_lib"],
    visibility = ["//visibility:public"],
)
