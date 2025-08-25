---
hide:
  - navigation
---

# Nostos

> A functional package manager for Kubernetes — **Nix‑like expressions**, **Terraform‑style graph diffs**, and a **kubectl‑aware LSP**.

[Get Started](getting-started.md){ .md-button .md-button--primary } [See Demos](demos/README.md){ .md-button }

---

## Why Nostos?

- **Expression‑oriented language** (YAML‑like surface, Nix‑like semantics) yields typed Kubernetes specs.
- **Graph diff & apply**: compute changes and apply them transactionally, Terraform‑style.
- **Batteries‑included LSP**: completions, hovers, go‑to, plus Kubernetes **type/spec intelligence**.

### Odyssey files (project entrypoint)

`odyssey.no` defines cluster → namespaces → resources. Example:

```no
let
  redisService: import(./redis-service.no)
  redisDeployment: import(./redis-deployment.no)
in
do-nyc1-k8s-1-33-1-do-0-nyc1-1750371119772:
  default:
  - redisService
  - redisDeployment
```

Nostos evaluates the file into a **graph of resources**, diffs it against the live cluster, and plans/applies the delta.

---

## What Nostos replaces

Nostos aims to consolidate your toolchain:

- **Helm** → declarative packaging without templates.
- **Terraform (for k8s)** → planning & applying infra‑safe diffs.
- **kubectl** → CRUD, but type‑safe and project‑scoped.

---

## Quick links

- [Getting Started](getting-started.md)
- [Odyssey File Format](odyssey.md)
- [CLI: diff, plan, apply](commands/diff.md)
- [Language & LSP](lsp.md)
