#!/usr/bin/env bash
# Nostos GitHub Pages scaffolding (MkDocs + Material)
# Usage: bash scripts/setup-pages.sh
# Run from the root of your Nostos repo. This will create docs/, mkdocs.yml,
# a GitHub Actions workflow for publishing to gh-pages, and placeholder pages
# including slots for offline Asciinema/VHS demos.

set -euo pipefail

mkdir -p scripts .github/workflows docs/{commands,language,demos,assets/{asciinema,demos},styles}

################################################################################
# mkdocs.yml
################################################################################
cat > mkdocs.yml << 'YAML'
site_name: Nostos
site_description: >-
  A functional package manager for Kubernetes — Nix‑like language, Terraform‑style plan/apply, kubectl‑aware LSP.
site_url: https://<your-gh-username>.github.io/<your-repo>
repo_url: https://github.com/<your-gh-username>/<your-repo>
edit_uri: edit/main/docs/

theme:
  name: material
  language: en
  features:
    - navigation.instant
    - navigation.sections
    - navigation.expand
    - content.code.copy
    - toc.integrate
    - header.autohide
    - search.suggest
    - search.highlight

plugins:
  - search

markdown_extensions:
  - admonition
  - def_list
  - footnotes
  - attr_list
  - toc:
      permalink: true
  - pymdownx.superfences
  - pymdownx.details
  - pymdownx.highlight
  - pymdownx.inlinehilite
  - pymdownx.emoji

extra_javascript:
  - assets/asciinema/asciinema-player.js
extra_css:
  - assets/asciinema/asciinema-player.css
  - styles/asciinema.css

nav:
  - Home: index.md
  - Getting Started: getting-started.md
  - Odyssey Files: odyssey.md
  - CLI:
      - Diff: commands/diff.md
      - Plan: commands/plan.md
      - Apply: commands/apply.md
  - Language & LSP: lsp.md
  - Demos: demos/README.md
YAML

################################################################################
# GitHub Actions workflow to build & publish to gh-pages
################################################################################
cat > .github/workflows/gh-pages.yml << 'YAML'
name: Publish Docs

on:
  push:
    branches: [ main ]
  workflow_dispatch:

permissions:
  contents: write

jobs:
  build-deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Python
        uses: actions/setup-python@v5
        with:
          python-version: '3.x'

      - name: Install MkDocs
        run: |
          pip install --upgrade pip
          pip install mkdocs-material

      - name: Build
        run: mkdocs build --strict

      - name: Deploy to gh-pages
        uses: peaceiris/actions-gh-pages@v3
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./site
          publish_branch: gh-pages
YAML

################################################################################
# Styles for Asciinema player wrapper
################################################################################
cat > docs/styles/asciinema.css << 'CSS'
.asciinema-wrapper { max-width: 1080px; margin: 1rem auto; }
.asciinema-wrapper .player { width: 100%; }
CSS

################################################################################
# Home page
################################################################################
cat > docs/index.md << 'MD'
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
MD

################################################################################
# Getting Started
################################################################################
cat > docs/getting-started.md << 'MD'
# Getting Started

## Install

> TODO: replace with your actual install instructions.

```bash
# Example (adjust accordingly):
go install github.com/<your-gh-username>/nostos/cmd/nostos@latest
```

## Create a project

```bash
mkdir my-app && cd my-app
printf '%s\n' "let\n  redisService: import(./redis-service.no)\n  redisDeployment: import(./redis-deployment.no)\nin\nmy-cluster:\n  default:\n  - redisService\n  - redisDeployment" > odyssey.no
```

## Dry‑run and apply

```bash
# Show what's different between workspace and cluster
nostos diff --workspace-dir .

# Review a Terraform‑style plan
nostos plan --workspace-dir .

# Apply the changes
nostos apply --workspace-dir .
```

See [CLI](commands/diff.md) for details and flags.
MD

################################################################################
# Odyssey file format
################################################################################
cat > docs/odyssey.md << 'MD'
# Odyssey Files

An `odyssey.no` file is the **entrypoint** for a Nostos project. The top‑level keys
map to **kubectl contexts/clusters**. Each cluster key contains namespaces; each
namespace lists the **resources** to materialize.

```no
let
  svc: import(./svc.no)
  deploy: import(./deploy.no)
in
my-cluster:
  default:
  - svc
  - deploy
```

- **`let … in`** binds local names to imported modules/resources (Nix‑style).
- **Cluster key** (`my-cluster`) is inferred as the kubectl context/cluster.
- **Namespace** (`default`) groups resources for that namespace.
- **Resource list** are expressions that evaluate to Kubernetes specs.

Nostos evaluates the file into a **typed graph** using built‑in Kubernetes type
information plus CRDs discovered from the cluster. The resulting graph is diffed
against live state to produce a safe plan.
MD

################################################################################
# LSP page
################################################################################
cat > docs/lsp.md << 'MD'
# Language & LSP

Nostos ships an LSP server to supercharge your editor with:

- Completions, hovers, go‑to definition, diagnostics
- **Type information** for Nostos language values
- **Kubernetes spec awareness** (field types, enums, required props)

## Starting the server

```bash
nostos lsp --stdio
```

> Point your editor's LSP client to `nostos lsp --stdio`. Most editors (VS Code,
> Neovim, Kakoune via kak-lsp, etc.) support stdio LSP servers.

## Demo (placeholder)

Below is a slot for an **offline** Asciinema‑recorded session or an MP4 generated
with [Charm's VHS](https://github.com/charmbracelet/vhs). Drop your file(s) into
`docs/assets/demos/` and adjust the `src`.

<div class="asciinema-wrapper">
  <asciinema-player src="/assets/demos/lsp.cast" preload></asciinema-player>
</div>

<video src="/assets/demos/lsp.mp4" controls muted loop playsinline style="width:100%; max-width:1080px;"></video>
MD

################################################################################
# CLI pages
################################################################################
cat > docs/commands/diff.md << 'MD'
# `nostos diff`

Show differences between the evaluated workspace and the live cluster.

```bash
nostos diff --workspace-dir ./envs/prod
```

- Evaluates `odyssey.no` and imports to build the resource graph
- Compares to cluster state reachable via current kubecontext
- Prints a human‑readable diff

<div class="asciinema-wrapper">
  <asciinema-player src="/assets/demos/diff.cast" preload></asciinema-player>
</div>

<video src="/assets/demos/diff.mp4" controls muted loop playsinline style="width:100%; max-width:1080px;"></video>
MD

cat > docs/commands/plan.md << 'MD'
# `nostos plan`

Produce a Terraform‑style plan showing the actions that would be taken.

```bash
nostos plan --workspace-dir .
```

- Creates a graph diff (create/replace/update/delete)
- Orders operations to satisfy dependencies
- No changes are applied

<div class="asciinema-wrapper">
  <asciinema-player src="/assets/demos/plan.cast" preload></asciinema-player>
</div>

<video src="/assets/demos/plan.mp4" controls muted loop playsinline style="width:100%; max-width:1080px;"></video>
MD

cat > docs/commands/apply.md << 'MD'
# `nostos apply`

Apply the planned changes to the cluster.

```bash
nostos apply --workspace-dir .
```

- Executes the dependency‑ordered operations
- Reports progress and errors clearly
- Supports idempotent re‑runs

<div class="asciinema-wrapper">
  <asciinema-player src="/assets/demos/apply.cast" preload></asciinema-player>
</div>

<video src="/assets/demos/apply.mp4" controls muted loop playsinline style="width:100%; max-width:1080px;"></video>
MD

################################################################################
# Demos index
################################################################################
cat > docs/demos/README.md << 'MD'
# Demos

Record once, replay offline. Two good options:

- **Asciinema** → compact `.cast` files + web player (embed below)
- **Charm VHS** → deterministic scripts → `.gif`/`.mp4`

## Asciinema (offline)

1. Record:
   ```bash
   asciinema rec -q -c "nostos plan --workspace-dir ." ./docs/assets/demos/plan.cast
   ```
2. Commit the `.cast` file and vendor the player assets (see below).

**Embed:**

<div class="asciinema-wrapper">
  <asciinema-player src="/assets/demos/plan.cast" preload></asciinema-player>
</div>

## VHS (GIF/MP4)

1. Create a tape script, e.g. `demos/plan.tape` (a starter is generated in this repo).
2. Render:
   ```bash
   vhs demos/plan.tape # creates demos/plan.gif and demos/plan.mp4
   cp demos/plan.mp4 docs/assets/demos/
   ```

**Embed:**

<video src="/assets/demos/plan.mp4" controls muted loop playsinline style="width:100%; max-width:1080px;"></video>

---

### Vendoring Asciinema player for offline playback

Fetch once and commit:

```bash
curl -L -o docs/assets/asciinema/asciinema-player.css \
  https://cdn.jsdelivr.net/npm/asciinema-player@3.8.0/dist/bundle/asciinema-player.css
curl -L -o docs/assets/asciinema/asciinema-player.js \
  https://cdn.jsdelivr.net/npm/asciinema-player@3.8.0/dist/bundle/asciinema-player.js
```

> These are MIT‑licensed. After committing, playback works with **no external CDN**.
MD

################################################################################
# Minimal VHS tapes for reproducible demos
################################################################################
mkdir -p demos
cat > demos/plan.tape << 'TAPE'
Output demos/plan.mp4
Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 720
Set Padding 12

Hide
Type "clear" Enter
Show

Type "nostos plan --workspace-dir ." Enter
Sleep 5000
TAPE

cat > demos/diff.tape << 'TAPE'
Output demos/diff.mp4
Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 720
Set Padding 12

Hide
Type "clear" Enter
Show

Type "nostos diff --workspace-dir ." Enter
Sleep 5000
TAPE

cat > demos/lsp.tape << 'TAPE'
Output demos/lsp.mp4
Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 720
Set Padding 12

Hide
Type "clear" Enter
Show

Type "nostos lsp --stdio" Enter
Sleep 5000
TAPE

################################################################################
# Friendly reminder in repo root README (optional)
################################################################################
if [ ! -f README.md ]; then
  cat > README.md << 'MD'
# Nostos

Nostos is a functional package manager for Kubernetes — Nix‑like language, Terraform‑style plan/apply, and a kubectl‑aware LSP.

👉 Documentation: will be published to GitHub Pages once you push to `main`.

To serve docs locally:

```bash
pip install mkdocs-material
mkdocs serve
```
MD
fi

################################################################################
# Final notes
################################################################################
cat > scripts/setup-pages.sh << 'BASH'
#!/usr/bin/env bash
set -euo pipefail
# This file is the script you just ran; it's only saved for reference.
BASH
chmod +x scripts/setup-pages.sh

printf "\nDone.\n\nNext steps:\n  1) Edit mkdocs.yml and replace <your-gh-username>/<your-repo>.\n  2) (Optional) Fetch Asciinema player assets (see docs/demos/README.md).\n  3) Commit & push to main. The workflow publishes to gh-pages.\n  4) In GitHub → Settings → Pages, choose the gh-pages branch.\n\nLocal preview: pip install mkdocs-material && mkdocs serve\n\n"

