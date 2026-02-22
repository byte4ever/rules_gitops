load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "exec",
    srcs = [
        "doc.go",
        "exec.go",
    ],
    importpath = "github.com/byte4ever/rules_gitops/gitops/exec",
    visibility = ["//visibility:public"],
)

go_test(
    name = "exec_test",
    srcs = ["exec_test.go"],
    deps = [
        ":exec",
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//require",
    ],
)
