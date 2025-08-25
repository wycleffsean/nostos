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
