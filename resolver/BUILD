load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "resolver",
    srcs = [
        "doc.go",
        "resolver.go",
    ],
    importpath = "github.com/byte4ever/rules_gitops/resolver",
    visibility = ["//visibility:public"],
    deps = ["@com_github_goccy_go_yaml//:go-yaml"],
)

go_test(
    name = "resolver_test",
    srcs = [
        "export_test.go",
        "resolver_test.go",
    ],
    data = glob(["testdata/**"]),
    embed = [":resolver"],
    deps = [
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//require",
    ],
)
