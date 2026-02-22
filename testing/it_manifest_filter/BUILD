load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "it_manifest_filter",
    srcs = [
        "doc.go",
        "filter.go",
    ],
    importpath = "github.com/byte4ever/rules_gitops/testing/it_manifest_filter",
    visibility = ["//visibility:public"],
    deps = ["@com_github_goccy_go_yaml//:go-yaml"],
)

go_test(
    name = "it_manifest_filter_test",
    srcs = [
        "export_test.go",
        "filter_test.go",
    ],
    data = glob(["testdata/**"]),
    embed = [":it_manifest_filter"],
    deps = [
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//require",
    ],
)
