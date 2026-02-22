load("@rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "cmd_lib",
    srcs = ["main.go"],
    importpath = "github.com/byte4ever/rules_gitops/gitops/prer/cmd",
    visibility = ["//visibility:private"],
    deps = [
        "//gitops/git",
        "//gitops/git/bitbucket",
        "//gitops/git/github",
        "//gitops/git/gitlab",
        "//gitops/prer",
    ],
)

go_binary(
    name = "cmd",
    embed = [":cmd_lib"],
    visibility = ["//visibility:public"],
)
