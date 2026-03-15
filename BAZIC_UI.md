# Bazic UI (WASM)

Bazic UI is a minimal JSON UI DSL rendered in the browser via a small JS bridge.

## Why Bazic UI Is Simple
- No Node/webpack required for the default Bazic UI template.
- One entry file (`app.bz`) with explicit routing and a small event loop.
- Pages are just Bazic functions in `pages/`.
- Pure Bazic code + static assets (HTML/CSS/JS).
- `bazic ui dev` rebuilds and reloads automatically.

## Quickstart
```powershell
.\bin\bazic.exe ui init --dir .\my-ui
.\bin\bazic.exe ui build --dir .\my-ui
.\bin\bazic.exe ui dev --dir .\my-ui --port 8080
```
For a shorter walkthrough, see `BAZIC_UI_QUICKSTART.md`.
For the frontend contract and standard flow, see `BAZIC_UI_FRONTEND_V1.md`.
For a straight-line workflow, see `BAZIC_UI_TOOLKIT.md`.

## CLI Helpers
```powershell
.\bin\bazic.exe ui init --dir .\my-ui
.\bin\bazic.exe ui page settings --dir .\my-ui
.\bin\bazic.exe ui component card --dir .\my-ui
.\bin\bazic.exe ui build --dir .\my-ui
.\bin\bazic.exe ui dev --dir .\my-ui --port 8080
.\bin\bazic.exe ui layout --dir .\my-ui
.\bin\bazic.exe ui routes --dir .\my-ui
```

## Project Layout
- `app.bz` is the entry point. It owns state and routes between pages.
- `pages/` contains page functions like `page_home` and `page_about`.
- `app.js` renders the JSON tree and bridges events to Bazic.
- `index.html`, `theme.css`, and `style.css` are standard static assets.

## UI Flow
1. Bazic builds a JSON UI tree with `ui_*` helpers.
2. Bazic calls `ui_render(tree_json)`.
3. JS receives the tree and renders or patches the DOM.

## Layouts
Use `ui_layout(title, nav, body)` to wrap a shared header and layout shell.
You can also generate a `layout.bz` file:
```powershell
.\bin\bazic.exe ui layout --dir .\my-ui
```
New UI apps auto-create `layout.bz` and default pages use `ui_layout_shell`.

## State And Events
- Persistent UI state uses `ui_state_get` and `ui_state_set`.
- Events are delivered via `ui_event_type`, `ui_event_target`, `ui_event_value`.
- Call `ui_event_clear()` after handling an event.

## Routing (Hash)
- `ui_route_get` reads the current route (from `location.hash`).
- `ui_route_set` updates the route and syncs the hash.
- JS emits a `route` event on hash changes.

## Components
Core primitives:
- `ui_badge`, `ui_card`, `ui_list`, `ui_list_item`
- `ui_form_row`, `ui_label`, `ui_select`, `ui_option`
- `ui_checkbox`, `ui_switch`, `ui_range`
See `BAZIC_UI_COOKBOOK.md` for more patterns.

## Store + Focus
- `ui_store_get` / `ui_store_set` provide shared state between components.
- `ui_focus_set` sets focus on a DOM element by `id`.

## Accessibility
- `ui_aria_label`, `ui_props_aria`, `ui_props_role`
- Focus ring is enabled by default in the web demo CSS.

## Nested Pages
Create nested pages by using `/` in the page name:
```powershell
.\bin\bazic.exe ui page settings/profile --dir .\my-ui
```
This creates `pages/settings/profile.bz` and auto-wires routing.

## Routes
List discovered routes from `pages/`:
```powershell
.\bin\bazic.exe ui routes --dir .\my-ui
```

## Templates
Default template:
```powershell
.\bin\bazic.exe ui init --dir .\my-ui
```

React template (CDN):
```powershell
.\bin\bazic.exe ui init --dir .\my-ui --template react
```

React + TypeScript template:
```powershell
.\bin\bazic.exe ui init --dir .\my-ui --template react-ts
```
Then run:
```powershell
npm install
npm run build:ui
```

## Theme Tokens
`theme.css` defines CSS variables used by `style.css`. Customize tokens like:
- `--ui-bg`, `--ui-panel`, `--ui-ink`, `--ui-accent`

## React Interop
The demo includes a React adapter.
- `examples/web/react_adapter.js`
- `examples/web/react.html`
React + TypeScript reference:
- `examples/web/react_ts` (build `src/react_adapter.ts` with esbuild)

## Interop Spec
See `BAZIC_UI_INTEROP.md` for adapter contract details (React/Vue/Svelte).

## Notes
- WASM builds require the Go backend.
- For full control, edit `app.js` to wire custom events or storage.

## Cookbook
See `BAZIC_UI_COOKBOOK.md` for ready-to-copy UI patterns.
