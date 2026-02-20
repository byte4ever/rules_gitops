# Copyright 2017 The Bazel Authors. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""An implementation of container_push using rules_oci.

This wraps oci_push tooling in a Bazel rule for publishing images,
replacing the former rules_docker-based implementation.
"""

load("@bazel_skylib//lib:dicts.bzl", "dicts")

K8sPushInfo = provider(
    "Information required to inject image into a manifest",
    fields = [
        "image_label",  # bazel target label of the image
        "legacy_image_name",  # optional short name
        "registry",
        "repository",
        "digestfile",
    ],
)

def _runfile_path(ctx, f):
    """Return the runfiles-relative path of a file."""
    if f.owner and f.owner.workspace_name:
        return f.owner.workspace_name + "/" + f.short_path
    return ctx.workspace_name + "/" + f.short_path

def _get_runfile_path(ctx, f):
    return "${RUNFILES}/%s" % _runfile_path(ctx, f)

def _impl(ctx):
    """Core implementation of k8s_container_push for rules_oci."""

    if K8sPushInfo in ctx.attr.image:
        # The image was already pushed, just rename if needed.
        # Ignore registry and repository parameters.
        kpi = ctx.attr.image[K8sPushInfo]
        if ctx.attr.image[DefaultInfo].files_to_run.executable:
            ctx.actions.expand_template(
                template = ctx.file._tag_tpl,
                substitutions = {
                    "%{args}": "",
                    "%{container_pusher}": _get_runfile_path(ctx, ctx.attr.image[DefaultInfo].files_to_run.executable),
                },
                output = ctx.outputs.executable,
                is_executable = True,
            )
        else:
            ctx.actions.write(
                content = "#!/bin/bash\n",
                output = ctx.outputs.executable,
                is_executable = True,
            )

        runfiles = ctx.runfiles(files = []).merge(ctx.attr.image[DefaultInfo].default_runfiles)

        ctx.actions.run_shell(
            tools = [kpi.digestfile],
            outputs = [ctx.outputs.digest],
            command = "cp -f \"$1\" \"$2\"",
            arguments = [kpi.digestfile.path, ctx.outputs.digest.path],
            mnemonic = "CopyFile",
            progress_message = "Copying files",
            use_default_shell_env = True,
            execution_requirements = {"no-remote": "1", "no-cache": "1"},
        )

        return [
            DefaultInfo(
                executable = ctx.outputs.executable,
                runfiles = runfiles,
            ),
            K8sPushInfo(
                image_label = kpi.image_label,
                legacy_image_name = ctx.attr.legacy_image_name,
                registry = kpi.registry,
                repository = kpi.repository,
                digestfile = kpi.digestfile,
            ),
        ]

    # -- Primary path: push an oci_image target --

    pusher_args = []
    pusher_input = []

    # Parse and get destination registry to be pushed to
    registry = ctx.expand_make_variables("registry", ctx.attr.registry, {})
    repository = ctx.expand_make_variables("repository", ctx.attr.repository, {})
    if not repository:
        repository = ctx.attr.image.label.package.lstrip("/") + "/" + ctx.attr.image.label.name
    prefix = ctx.attr.repository_prefix
    if prefix and prefix != repository.partition("/")[0]:
        repository = "%s/%s" % (prefix, repository)
    tag = ctx.expand_make_variables("tag", ctx.attr.tag, {})

    # If a tag file is provided, override tag with tag value
    if ctx.file.tag_file:
        tag = "$(cat {})".format(_get_runfile_path(ctx, ctx.file.tag_file))
        pusher_input.append(ctx.file.tag_file)

    stamp = "{" in tag or "{" in registry or "{" in repository
    stamp_inputs = [ctx.file._info_file] if stamp else []
    for f in stamp_inputs:
        pusher_args += ["-stamp-info-file", "%s" % _get_runfile_path(ctx, f)]
    pusher_input += stamp_inputs

    # Collect OCI image layout files from the oci_image target.
    # rules_oci oci_image produces an OCI image layout directory as a TreeArtifact.
    # We pass the image directory to the pusher.
    image_files = ctx.attr.image[DefaultInfo].files.to_list()
    pusher_input += image_files

    # TODO(rules_oci): The oci_push rule from rules_oci works differently than
    # the rules_docker pusher. rules_oci provides `oci_push` as a rule that
    # generates a launcher script. For our use case, we need to:
    # 1. Compute the digest from the OCI layout
    # 2. Push via crane/oci_push tooling
    # For now, we generate the digest by reading the OCI index and produce the
    # push script via the tag template.

    # Generate digest file from the OCI image layout.
    # The OCI image layout contains an index.json with the manifest digest.
    # We extract it at build time.
    ctx.actions.run_shell(
        inputs = image_files,
        outputs = [ctx.outputs.digest],
        command = """
            set -euo pipefail
            # Find the OCI layout directory (TreeArtifact) among inputs
            for f in "$@"; do
                if [ -f "$f/index.json" ]; then
                    # Extract the digest from the OCI index
                    # index.json has manifests[0].digest
                    grep -o '"sha256:[a-f0-9]*"' "$f/index.json" | head -1 | tr -d '"' > "%s"
                    exit 0
                fi
            done
            # Fallback: if input is a single digest file, copy it
            cp -f "$1" "%s"
        """ % (ctx.outputs.digest.path, ctx.outputs.digest.path),
        arguments = [f.path for f in image_files],
        mnemonic = "OCIDigest",
        progress_message = "Extracting OCI image digest for %s" % ctx.label,
    )

    if ctx.attr.image_digest_tag:
        tag = "$(cat {} | cut -d ':' -f 2 | cut -c 1-7)".format(_get_runfile_path(ctx, ctx.outputs.digest))
        pusher_input.append(ctx.outputs.digest)

    pusher_args.append("--dst={registry}/{repository}:{tag}".format(
        registry = registry,
        repository = repository,
        tag = tag,
    ))

    # TODO(rules_oci): Wire up crane-based push. The _pusher attr should point
    # to a crane binary or an oci_push wrapper that accepts --dst and OCI layout
    # directory arguments. For now we reference a placeholder.
    for f in image_files:
        pusher_args.append("--image-dir=%s" % _get_runfile_path(ctx, f))

    if ctx.attr.skip_unchanged_digest:
        pusher_args.append("-skip-unchanged-digest")

    ctx.actions.expand_template(
        template = ctx.file._tag_tpl,
        substitutions = {
            "%{args}": " ".join(pusher_args),
            "%{container_pusher}": _get_runfile_path(ctx, ctx.executable._pusher),
        },
        output = ctx.outputs.executable,
        is_executable = True,
    )

    pusher_runfiles = [ctx.executable._pusher] + pusher_input
    runfiles = ctx.runfiles(files = pusher_runfiles)
    runfiles = runfiles.merge(ctx.attr._pusher[DefaultInfo].default_runfiles)

    return [
        DefaultInfo(
            executable = ctx.outputs.executable,
            runfiles = runfiles,
        ),
        K8sPushInfo(
            image_label = ctx.attr.image.label,
            legacy_image_name = ctx.attr.legacy_image_name,
            registry = registry,
            repository = repository,
            digestfile = ctx.outputs.digest,
        ),
    ]

# Pushes a container image to a registry.
k8s_container_push = rule(
    doc = """
Pushes a container image.

This rule pushes a container image to a registry using rules_oci tooling.

Args:
  name: name of the rule
  image: the label of the oci_image to push.
  legacy_image_name: alias for the image name in addition to default full bazel target name. Please use only for compatibility with older deployment
  registry: the registry to which we are pushing.
  repository: the name of the image. If not present, default to the image's bazel target path
  repository_prefix: an optional prefix added to the name of the image
  tag: (optional) the tag of the image, default to 'latest'.
""",
    attrs = dicts.add({
        "image": attr.label(
            mandatory = True,
            doc = "The label of the oci_image to push.",
        ),
        "image_digest_tag": attr.bool(
            default = False,
            mandatory = False,
            doc = "Tag the image with the container digest, default to False",
        ),
        "legacy_image_name": attr.string(doc = "image name used in deployments, for compatibility with k8s_deploy. Do not use, refer images by full bazel target name instead"),
        "registry": attr.string(
            doc = "The registry to which we are pushing.",
            default = "docker.io",
        ),
        "repository": attr.string(
            doc = "the name of the image. If not present, default to the image's bazel target path",
        ),
        "repository_prefix": attr.string(
            doc = "an optional prefix added to the name of the image",
        ),
        "skip_unchanged_digest": attr.bool(
            default = False,
            doc = "Only push images if the digest has changed, default to False",
        ),
        "stamp": attr.bool(
            default = False,
            mandatory = False,
            doc = "(unused)",
        ),
        "tag": attr.string(
            default = "latest",
            doc = "(optional) The tag of the image, default to 'latest'.",
        ),
        "tag_file": attr.label(
            allow_single_file = True,
            doc = "(optional) The label of the file with tag value. Overrides 'tag'.",
        ),
        "_info_file": attr.label(
            default = Label("//skylib:more_stable_status.txt"),
            allow_single_file = True,
        ),
        "_pusher": attr.label(
            # TODO(rules_oci): Point to crane binary or oci_push wrapper.
            # For rules_oci, this should be something like
            # "@oci_crane_toolchains//:current_toolchain" or a custom push script.
            default = Label("//skylib:oci_push_launcher"),
            cfg = "exec",
            executable = True,
            allow_files = True,
        ),
        "_tag_tpl": attr.label(
            default = Label("//skylib:push-tag.sh.tpl"),
            allow_single_file = True,
        ),
    }),
    executable = True,
    implementation = _impl,
    outputs = {
        "digest": "%{name}.digest",
    },
)
