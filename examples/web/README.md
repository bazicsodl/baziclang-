# Bazic Web Demo

This demo runs Bazic in the browser via WASM + `wasm_exec.js`.

## Build
```powershell
.\tools\web\build.ps1
```

```bash
./tools/web/build.sh
```

## Run
- Serve `examples/web` with any static server and open `index.html`.
- Click **Run Bazic** to execute the WASM binary and view output.
- `theme.css` defines the UI tokens used by `style.css`.

## Notes
- WASM builds use the Go backend.
- If you see a missing `wasm_exec.js`, rerun the build script.
- JS -> Bazic interop is done via argv/env and a JSON bridge:
  - `app.js` sets `go.argv`, `go.env`, and `globalThis.BAZIC_WEB.get/set`.
  - `app.bz` reads `os_args()`, `os_getenv()`, and `web_get_json()`.
- Bazic UI: Bazic builds a JSON UI tree and calls `ui_render()`. JS renders it into the DOM via `BAZIC_UI_RENDER`.
- React adapter demo: open `react.html` to render the same UI tree with React (via `react_adapter.js`).
- React + TypeScript demo: `examples/web/react_ts` (build adapter with npm, open `react_ts/index.html`).
- Vue adapter demo: open `vue.html` (via `vue_adapter.js`).
- Svelte adapter demo: open `svelte.html` (via `svelte_adapter.js`).
- Reference app: `examples/web/apps/dashboard` (open `apps/dashboard/index.html`).
- Reference app: `examples/web/apps/form` (open `apps/form/index.html`).
- Reference app: `examples/web/apps/landing` (open `apps/landing/index.html`).
- Reference app: `examples/web/apps/admin` (open `apps/admin/index.html`).
- Event/state loop: clicks are sent through `event_type/event_target`, and Bazic persists click count via `web_set_json("clicks", ...)`.
- State helpers: `ui_state_get`/`ui_state_set` provide a simple storage layer for UI state.
- Renderer includes a minimal DOM patcher to avoid full re-renders on each update.
- Pages live in `pages/` and are routed in `app.bz` via `render_page(...)`.
- Component demo: `pages/components.bz`.
