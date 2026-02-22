load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "prer",
    srcs = [
        "doc.go",
        "prer.go",
    ],
    importpath = "github.com/byte4ever/rules_gitops/gitops/prer",
    visibility = ["//visibility:public"],
    deps = [
        "//gitops/bazel",
        "//gitops/commitmsg",
        "//gitops/digester",
        "//gitops/exec",
        "//gitops/git",
        "@com_github_goccy_go_json//:go-json",
        "@com_github_valyala_fasttemplate//:fasttemplate",
    ],
)

go_test(
    name = "prer_test",
    srcs = [
        "export_test.go",
        "prer_test.go",
    ],
    embed = [":prer"],
    deps = [
        "@com_github_goccy_go_json//:go-json",
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//require",
    ],
)
