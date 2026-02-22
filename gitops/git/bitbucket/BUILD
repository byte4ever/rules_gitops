load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "bitbucket",
    srcs = [
        "bitbucket.go",
        "doc.go",
    ],
    importpath = "github.com/byte4ever/rules_gitops/gitops/git/bitbucket",
    visibility = ["//visibility:public"],
    deps = ["@com_github_goccy_go_json//:go-json"],
)

go_test(
    name = "bitbucket_test",
    srcs = ["bitbucket_test.go"],
    deps = [
        ":bitbucket",
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//require",
    ],
)
