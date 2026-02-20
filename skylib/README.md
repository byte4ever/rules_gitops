# Starlark Rules Reference

## Public API

The public API is exported from `@rules_gitops//gitops:defs.bzl`:

```starlark
load("@rules_gitops//gitops:defs.bzl", "k8s_deploy", "k8s_test_setup", "external_image")
```

### k8s_deploy

Main deployment macro. Creates kustomize, kubectl, show, and optionally gitops targets for a set of Kubernetes manifests.

**Generated targets:**

| Target | Rule | Description |
|--------|------|-------------|
| `name` | `kustomize` | Rendered manifest YAML |
| `name.show` | `show` | Print rendered manifests |
| `name.apply` | `kubectl` | kubectl apply |
| `name.delete` | `kubectl` | kubectl delete (only when `gitops=False`) |
| `name.gitops` | `gitops` | Write manifests for gitops (only when `gitops=True`) |

When `gitops=False`, both `.apply` and `.delete` targets are created. When `gitops=True`, `.apply` and `.gitops` targets are created instead.

**Attributes:**

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `name` | `string` | required | Rule name; becomes part of the target manifest file name in gitops output. |
| `cluster` | `string` | `"dev"` | Target cluster name. |
| `user` | `string` | `"{BUILD_USER}"` | Kubectl user for apply/delete. |
| `namespace` | `string` | `None` | Kubernetes namespace. Required when `gitops=True`. Defaults to `"{BUILD_USER}"` when `gitops=False`. |
| `configmaps_srcs` | `label_list` | `None` | Configmap source files. |
| `secrets_srcs` | `label_list` | `None` | Secret source files. |
| `configmaps_renaming` | `string` | `None` | Renaming policy for configmaps. Can be `None` or `"hash"`. |
| `manifests` | `label_list` | `None` | List of manifest files. Defaults to all `*.yaml` and `*.yaml.tpl` in the package. |
| `name_prefix` | `string` | `None` | Kustomize name prefix. |
| `name_suffix` | `string` | `None` | Kustomize name suffix. |
| `prefix_suffix_app_labels` | `bool` | `False` | When `True`, apply kustomize configuration to modify app labels in Deployments. |
| `patches` | `label_list` | `None` | Kustomize patches. |
| `image_name_patches` | `string_dict` | `{}` | Dict of image name replacements. |
| `image_tag_patches` | `string_dict` | `{}` | Dict of image tag replacements. |
| `substitutions` | `string_dict` | `{}` | Template parameter substitutions. `CLUSTER` and `NAMESPACE` are added automatically. |
| `configurations` | `label_list` | `[]` | Additional kustomize config files. |
| `common_labels` | `string_dict` | `{}` | Labels applied to all objects. |
| `common_annotations` | `string_dict` | `{}` | Annotations applied to all objects. |
| `deps` | `label_list` | `[]` | Template dependencies. |
| `deps_aliases` | `string_dict` | `{}` | Aliases for template deps. |
| `images` | `list` | `[]` | Container images to push. Can be a list of labels or a dict of `{name: label}`. |
| `image_digest_tag` | `bool` | `False` | Tag images with digest. |
| `image_registry` | `string` | `"docker.io"` | Registry to push to. |
| `image_repository` | `string` | `None` | Repository path for push. |
| `image_repository_prefix` | `string` | `None` | Prefix added to repository name. |
| `objects` | `label_list` | `[]` | Dependent kustomize objects. |
| `gitops` | `bool` | `True` | Enable gitops mode. Set to `False` for individual namespace work. |
| `gitops_path` | `string` | `"cloud"` | Path prefix for gitops output. |
| `deployment_branch` | `string` | `None` | Git branch for deployment. |
| `release_branch_prefix` | `string` | `"main"` | Release branch prefix. |
| `start_tag` | `string` | `"{{"` | Template start delimiter. |
| `end_tag` | `string` | `"}}"` | Template end delimiter. |
| `tags` | `string_list` | `[]` | Bazel tags for all generated targets. |
| `visibility` | `list` | `None` | Bazel visibility for targets. |

### k8s_test_setup

Creates an integration test setup executable that pushes images, applies manifests, waits for pods, and sets up port forwarding.

**Attributes:**

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `kubeconfig` | `label` | `@k8s_test//:kubeconfig` | Path to the kubeconfig file. |
| `kubectl` | `label` | `@k8s_test//:kubectl` | Path to the kubectl binary. |
| `objects` | `label_list` | `[]` | Kustomize targets to apply. |
| `portforward_services` | `string_list` | `[]` | Services to port-forward. |
| `setup_timeout` | `string` | `"10m"` | Timeout for setup operations. |
| `wait_for_apps` | `string_list` | `[]` | App labels to wait for readiness. |
| `cluster` | `label` | `@k8s_test//:cluster` | Cluster name file. |

### external_image

Declares an externally-hosted container image (not built by Bazel) for injection into manifests via `K8sPushInfo`.

**Attributes:**

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `image` | `string` | required | The image location, e.g. `gcr.io/foo/bar:baz`. |
| `image_name` | `string` | `""` | Image name alias. Deprecated: use full target label instead. |
| `digest` | `string` | required | The image digest, e.g. `sha256:deadbeef`. |

## Image Pushing

### k8s_container_push

Pushes a container image to a registry using rules_oci tooling.

**Attributes:**

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `image` | `label` | required | The label of the `oci_image` to push. |
| `image_digest_tag` | `bool` | `False` | Tag the image with the container digest. |
| `legacy_image_name` | `string` | `""` | Image name used in deployments for backward compatibility. Prefer full Bazel target labels. |
| `registry` | `string` | `"docker.io"` | The registry to push to. |
| `repository` | `string` | `""` | The image name. Defaults to the image's Bazel target path if not set. |
| `repository_prefix` | `string` | `""` | An optional prefix added to the image name. |
| `skip_unchanged_digest` | `bool` | `False` | Only push images if the digest has changed. |
| `stamp` | `bool` | `False` | Unused. |
| `tag` | `string` | `"latest"` | The tag of the image. |
| `tag_file` | `label` | `None` | Label of a file containing the tag value. Overrides `tag`. |

**Outputs:**

| Name | Description |
|------|-------------|
| `%{name}.digest` | File containing the image digest. |

### K8sPushInfo Provider

Carries image metadata from push rules to kustomize/deploy rules.

| Field | Description |
|-------|-------------|
| `image_label` | Bazel target label of the image. |
| `legacy_image_name` | Optional short name alias for backward compatibility. |
| `registry` | Target registry. |
| `repository` | Target repository. |
| `digestfile` | File containing the image digest. |

## Kustomize

### kustomize

Builds Kubernetes manifests using kustomize with template expansion and image resolution.

Defined in `//skylib/kustomize:kustomize.bzl`.

**Attributes:**

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `manifests` | `label_list` | `[]` | Kubernetes manifest files. |
| `namespace` | `string` | `""` | Kubernetes namespace. |
| `configmaps_srcs` | `label_list` | `[]` | Configmap source files. |
| `secrets_srcs` | `label_list` | `[]` | Secret source files. |
| `images` | `label_list` | `[]` | Images used in manifests (must provide `K8sPushInfo`). |
| `objects` | `label_list` | `[]` | Dependent kustomize objects (must provide `KustomizeInfo`). |
| `substitutions` | `string_dict` | `{}` | Template variable substitutions. |
| `deps` | `label_list` | `[]` | Additional template dependencies. |
| `deps_aliases` | `string_dict` | `{}` | Aliases for template dependencies. |
| `patches` | `label_list` | `[]` | Kustomize patches to apply. |
| `image_name_patches` | `string_dict` | `{}` | Set new names for selected images. |
| `image_tag_patches` | `string_dict` | `{}` | Set new tags for selected images. |
| `configurations` | `label_list` | `[]` | Additional kustomize configuration files. |
| `common_labels` | `string_dict` | `{}` | Labels applied to all resources. |
| `common_annotations` | `string_dict` | `{}` | Annotations applied to all resources. |
| `name_prefix` | `string` | `""` | Kustomize namePrefix to add to all resources. |
| `name_suffix` | `string` | `""` | Kustomize nameSuffix to add to all resources. |
| `disable_name_suffix_hash` | `bool` | `True` | Disable hash suffix on configmap/secret names. |
| `start_tag` | `string` | `"{{"` | Start delimiter for template expansion. |
| `end_tag` | `string` | `"}}"` | End delimiter for template expansion. |

**Outputs:**

| Name | Description |
|------|-------------|
| `%{name}.yaml` | Rendered Kubernetes manifests. |

### KustomizeInfo Provider

Carries image push metadata through the kustomize rule graph.

| Field | Description |
|-------|-------------|
| `image_pushes` | Depset of targets providing `K8sPushInfo`, representing image push shell script fragments. |

## Supporting Rules

### show

Generates an executable that renders and prints the final Kubernetes manifests through the template engine.

Defined in `//skylib:k8s.bzl`.

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `src` | `label` | required | Input kustomize target. |
| `namespace` | `string` | required | Kubernetes namespace. |

### kubectl

Generates a kubectl apply/delete executable for rendered manifests.

Defined in `//skylib/kustomize:kustomize.bzl`.

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `srcs` | `label_list` | `[]` | Kustomize targets (must provide `KustomizeInfo`). |
| `cluster` | `string` | required | Target cluster name. |
| `namespace` | `string` | required | Kubernetes namespace. |
| `command` | `string` | `"apply"` | kubectl command (`apply` or `delete`). |
| `user` | `string` | `"{BUILD_USER}"` | kubectl user. |
| `push` | `bool` | `True` | Whether to push images before applying. |

### gitops

Generates a gitops output executable that writes rendered manifests to a directory tree.

Defined in `//skylib/kustomize:kustomize.bzl`.

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `srcs` | `label_list` | `[]` | Kustomize targets (must provide `KustomizeInfo`). |
| `cluster` | `string` | required | Target cluster name. |
| `namespace` | `string` | required | Kubernetes namespace. |
| `deployment_branch` | `string` | `""` | Git branch for deployment. |
| `gitops_path` | `string` | `""` | Path prefix for gitops output. |
| `release_branch_prefix` | `string` | `""` | Release branch prefix. |
| `strip_prefixes` | `string_list` | `[]` | Prefixes to strip from output filenames. |

### expand_template

Expands a template file by substituting variables and stamp values.

Defined in `//skylib:templates.bzl`.

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `template` | `label` | required | The template file to expand. |
| `out` | `output` | required | The name of the output file. |
| `substitutions` | `string_dict` | required | Key-value pairs available as template variables. |
| `deps` | `label_list` | `[]` | Additional files accessible as `imports[label]` in the template. |
| `deps_aliases` | `string_dict` | `{}` | Name-to-label mapping for import aliases. |
| `start_tag` | `string` | `"{{"` | Start delimiter for template expansion. |
| `end_tag` | `string` | `"}}"` | End delimiter for template expansion. |
| `executable` | `bool` | `True` | Mark the output as executable. |

### merge_files

Merges a set of files into a single tarball. Files ending with the template extension (default `.tpl`) are expanded before merging.

Defined in `//skylib:templates.bzl`.

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `srcs` | `label_list` | `[]` | Files to merge. `.tpl` files are expanded as templates. |
| `directory` | `string` | `"/"` | Base directory for files in the tarball. |
| `path_format` | `string` | `"{path}"` | Format string for output paths. Supports `{path}`, `{dirname}`, `{basename}`, `{extension}`. |
| `strip_prefixes` | `string_list` | `[]` | Prefixes to strip from source paths. |
| `strip_suffixes` | `string_list` | `["-staging", "-test"]` | Suffixes to strip from source paths. |
| `substitutions` | `string_dict` | `{}` | Template variable substitutions. |
| `template_extension` | `string` | `".tpl"` | File extension treated as templates. |

**Outputs:**

| Name | Description |
|------|-------------|
| `%{name}.tar` | Tarball containing all merged files. |

### stamp_value

Stamps a string by substituting `{VAR}` placeholders from workspace status files.

Defined in `//skylib:stamp.bzl`.

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `str` | `string` | `"{BUILD_USER}"` | Format string with `{VAR}` placeholders. |

**Outputs:**

| Name | Description |
|------|-------------|
| `%{name}.txt` | The stamped string. |

### more_stable_status

Extracts selected variables from `stable-status.txt` into a reduced status file. This produces a more cacheable subset of the full workspace status.

Defined in `//skylib:stamp.bzl`.

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `vars` | `string_list` | required | Variables to extract from `stable-status.txt`. |

**Outputs:**

| Name | Description |
|------|-------------|
| `%{name}.txt` | Filtered status file. |

### workspace_binary

Wraps a binary tool so it executes with the working directory set to the workspace root.

Defined in `//skylib:run_in_workspace.bzl`. This is a macro, not a rule.

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `name` | `string` | required | Target name. |
| `cmd` | `label` | required | Label of the binary to wrap. |
| `args` | `list` | `None` | Optional arguments to pass to the binary. |
| `visibility` | `list` | `None` | Bazel visibility. |
| `data` | `label_list` | `None` | Additional data dependencies. |
| `root_file` | `label` | `//:MODULE.bazel` | Label used to locate the workspace root in runfiles. |

## Module Extension

### gitops

The `gitops` module extension downloads the kustomize binary for manifest building.

```starlark
gitops = use_extension("@rules_gitops//skylib/kustomize:extensions.bzl", "gitops")
use_repo(gitops, "kustomize_bin")
```

- **Kustomize version:** 5.4.3
- **Supported platforms:** `darwin_amd64`, `darwin_arm64`, `linux_amd64`, `linux_arm64`

The extension auto-detects the host platform and downloads the matching kustomize binary from the official GitHub releases. The binary is exposed as `@kustomize_bin//:kustomize`.
