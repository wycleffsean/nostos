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
