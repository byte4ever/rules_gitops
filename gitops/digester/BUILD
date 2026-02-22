load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "digester",
    srcs = [
        "digester.go",
        "doc.go",
    ],
    importpath = "github.com/byte4ever/rules_gitops/gitops/digester",
    visibility = ["//visibility:public"],
)

go_test(
    name = "digester_test",
    srcs = ["digester_test.go"],
    deps = [
        ":digester",
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//require",
    ],
)
