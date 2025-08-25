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
