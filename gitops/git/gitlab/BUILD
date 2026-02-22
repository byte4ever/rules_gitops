load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "gitlab",
    srcs = [
        "doc.go",
        "gitlab.go",
    ],
    importpath = "github.com/byte4ever/rules_gitops/gitops/git/gitlab",
    visibility = ["//visibility:public"],
    deps = ["@com_gitlab_gitlab_org_api_client_go//:client-go"],
)

go_test(
    name = "gitlab_test",
    srcs = ["gitlab_test.go"],
    deps = [
        ":gitlab",
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//require",
    ],
)
