load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "commitmsg",
    srcs = [
        "commitmsg.go",
        "doc.go",
    ],
    importpath = "github.com/byte4ever/rules_gitops/gitops/commitmsg",
    visibility = ["//visibility:public"],
)

go_test(
    name = "commitmsg_test",
    srcs = ["commitmsg_test.go"],
    deps = [
        ":commitmsg",
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//require",
    ],
)
