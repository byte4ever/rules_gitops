# Documentation Design

## Context

rules_gitops is a modern port of Adobe's rules_gitops for Bazel 8+ with bzlmod. The project has good inline godoc comments on exported Go symbols but zero user-facing documentation: no root README, no package READMEs, no doc.go files, incomplete Starlark docstrings.

## Approach: Distributed Documentation

Documentation lives next to the code it describes. Each layer has its own docs:
- Root README for project overview and quickstart
- doc.go + README.md per Go package
- Starlark docstrings inline, with skylib/README.md as a reference
- docs/architecture.md for system-level understanding

## Deliverables

### 1. Root README.md

Simplified structure:
- Overview (what it does, origin as port of Adobe's rules_gitops, Bazel 8+/bzlmod/rules_oci)
- Quick Start (MODULE.bazel setup, minimal k8s_deploy example)
- Architecture (3-tool manifest pipeline diagram)
- Documentation links (skylib, examples, go packages, architecture doc)
- CLI Tools table (inline)
- Development (build/test/lint commands)
- License

### 2. docs/architecture.md

- 3-tool manifest pipeline: kustomize build -> templating ({{VAR}}) -> resolver (image substitution)
- Package dependency map
- How Starlark rules connect to Go tooling
- GitProvider strategy pattern
- Worker pool in prer

### 3. skylib/README.md — Starlark Rules Reference

For each rule/macro/provider:
- Name, description, what it produces
- Attributes table (name, type, default, description)
- Example usage snippet
- Provider field descriptions

Covers: k8s_deploy, k8s_test_setup, external_image, k8s_container_push, show, kubeconfig, k8s_test_namespace, stamp, stamp_value, more_stable_status, expand_template, merge_files, workspace_binary, kustomize, kubectl, gitops, imagePushStatements, KustomizeInfo, K8sPushInfo.

### 4. skylib/kustomize/README.md

- Module extension for downloading kustomize binary
- How the extension works with bzlmod
- KustomizeInfo provider

### 5. examples/README.md

- Prerequisites (Bazel 8+, Docker, kind for e2e)
- Helloworld example walkthrough (three deployment variants: dev, canary, release)
- How to build and run

### 6. Go doc.go Files (16 packages)

One per package. Contains the canonical `// Package ...` comment (migrated from current first-file location where needed) plus `package <name>` declaration.

Packages: bazel, commitmsg, digester, exec, git, github, gitlab, bitbucket, prer, resolver, stamper, templating, filter (it_manifest_filter), sidecar (it_sidecar), client (sidecar/client), stern.

### 7. Go README.md Files (16 packages)

One per package. Detail scaled to complexity:

**Small utilities** (bazel, commitmsg, digester, exec): Brief description, API overview, usage example.

**Core libraries** (git, resolver, stamper, templating): Moderate README with API overview, examples, configuration.

**Complex packages** (prer, sidecar, client): Detailed README with configuration tables, workflow description.

**Provider packages** (github, gitlab, bitbucket): Setup guide, required tokens/env vars, configuration.

### 8. Starlark In-Code Docstrings

Add `doc=` to all rules/macros/providers/attributes missing them:
- gitops/defs.bzl: module-level docstring
- skylib/k8s.bzl: show, kubeconfig, k8s_test_namespace, k8s_test_setup rules
- skylib/external_image.bzl: rule-level doc=
- skylib/stamp.bzl: stamp_value, more_stable_status rules
- skylib/kustomize/kustomize.bzl: KustomizeInfo provider fields, kustomize/kubectl/gitops rules
- skylib/run_in_workspace.bzl: workspace_binary macro docstring
- All _impl functions: brief docstrings

### 9. Missing Go Doc

- stern package: add `// Package stern ...` comment

## Scope

~40 files created or modified. No code logic changes — documentation only.

## Audience

Both Bazel users (people adding GitOps rules to their project) and contributors (developers extending the project).
