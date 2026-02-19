# Copyright 2020 Adobe. All rights reserved.
# This file is licensed to you under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License. You may obtain a copy
# of the License at http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software distributed under
# the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
# OF ANY KIND, either express or implied. See the License for the specific language
# governing permissions and limitations under the License.

"""Module extension for downloading the kustomize binary."""

_KUSTOMIZE_VERSION = "5.4.3"

_BINARIES = {
    "darwin_amd64": (
        "kustomize_v{v}_darwin_amd64.tar.gz",
        "6a708ef727594bbb5f2b8f9f8049375a6028d57fa8897c1f9e78effde4e403a2",
    ),
    "darwin_arm64": (
        "kustomize_v{v}_darwin_arm64.tar.gz",
        "3e159813a5feae46726fb22736b8764f2dbac83ba982c91ccd0244762456272c",
    ),
    "linux_amd64": (
        "kustomize_v{v}_linux_amd64.tar.gz",
        "3669470b454d865c8184d6bce78df05e977c9aea31c30df3c669317d43bcc7a7",
    ),
    "linux_arm64": (
        "kustomize_v{v}_linux_arm64.tar.gz",
        "1b515578b0af12c15d9856720066ce2fe66756d63785b2cbccaf2885beb2381c",
    ),
}

_DOWNLOAD_URL = "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv{v}/"

def _detect_platform(ctx):
    """Detect the host platform and return a key into _BINARIES."""
    if ctx.os.name == "linux":
        arch = ctx.execute(["uname", "-m"]).stdout.strip()
        if arch == "aarch64":
            return "linux_arm64"
        return "linux_amd64"
    elif ctx.os.name == "mac os x":
        arch = ctx.execute(["uname", "-m"]).stdout.strip()
        if arch == "arm64":
            return "darwin_arm64"
        return "darwin_amd64"
    else:
        fail("Unsupported platform: " + ctx.os.name)

def _kustomize_repo_impl(ctx):
    """Repository rule that downloads the kustomize binary."""
    platform = _detect_platform(ctx)
    filename, sha256 = _BINARIES[platform]
    filename = filename.format(v = _KUSTOMIZE_VERSION)
    url = _DOWNLOAD_URL.format(v = _KUSTOMIZE_VERSION) + filename

    ctx.file("BUILD.bazel", """\
package(default_visibility = ["//visibility:public"])

sh_binary(
    name = "kustomize",
    srcs = ["bin/kustomize"],
)
""")

    ctx.download_and_extract(
        url = url,
        output = "bin/",
        sha256 = sha256,
    )

_kustomize_repo = repository_rule(
    implementation = _kustomize_repo_impl,
)

def _gitops_impl(ctx):
    _kustomize_repo(name = "kustomize_bin")

gitops = module_extension(
    implementation = _gitops_impl,
)
