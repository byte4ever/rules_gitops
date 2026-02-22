load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "templating",
    srcs = [
        "doc.go",
        "engine.go",
    ],
    importpath = "github.com/byte4ever/rules_gitops/templating",
    visibility = ["//visibility:public"],
    deps = ["@com_github_valyala_fasttemplate//:fasttemplate"],
)

go_test(
    name = "templating_test",
    srcs = ["engine_test.go"],
    deps = [
        ":templating",
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//require",
    ],
)
