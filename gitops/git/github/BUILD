load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "github",
    srcs = [
        "doc.go",
        "github.go",
    ],
    importpath = "github.com/byte4ever/rules_gitops/gitops/git/github",
    visibility = ["//visibility:public"],
    deps = ["@com_github_google_go_github_v68//github"],
)

go_test(
    name = "github_test",
    srcs = ["github_test.go"],
    deps = [
        ":github",
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//require",
    ],
)
