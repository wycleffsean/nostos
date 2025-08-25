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
