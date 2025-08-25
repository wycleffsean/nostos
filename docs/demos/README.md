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
