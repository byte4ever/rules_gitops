load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "stamper",
    srcs = [
        "doc.go",
        "stamper.go",
    ],
    importpath = "github.com/byte4ever/rules_gitops/stamper",
    visibility = ["//visibility:public"],
    deps = ["@com_github_valyala_fasttemplate//:fasttemplate"],
)

go_test(
    name = "stamper_test",
    srcs = ["stamper_test.go"],
    deps = [
        ":stamper",
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//require",
    ],
)
