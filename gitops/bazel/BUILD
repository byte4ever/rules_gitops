load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "bazel",
    srcs = [
        "bazeltargets.go",
        "doc.go",
    ],
    importpath = "github.com/byte4ever/rules_gitops/gitops/bazel",
    visibility = ["//visibility:public"],
)

go_test(
    name = "bazel_test",
    srcs = ["bazeltargets_test.go"],
    deps = [
        ":bazel",
        "@com_github_stretchr_testify//assert",
    ],
)
