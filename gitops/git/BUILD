load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "git",
    srcs = [
        "doc.go",
        "provider.go",
        "repo.go",
    ],
    importpath = "github.com/byte4ever/rules_gitops/gitops/git",
    visibility = ["//visibility:public"],
    deps = ["//gitops/exec"],
)

go_test(
    name = "git_test",
    srcs = [
        "export_test.go",
        "provider_test.go",
        "repo_test.go",
    ],
    embed = [":git"],
    deps = [
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//require",
    ],
)
