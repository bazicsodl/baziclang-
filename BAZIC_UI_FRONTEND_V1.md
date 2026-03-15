# Bazic UI Frontend V1

This document defines the minimal frontend contract for Bazic UI apps.

## Goals
- Straightforward: explicit state, explicit events, predictable render.
- Fast: no framework runtime required; WASM + small JS bridge.
- Portable: works in static hosting and server-side assets.

## V1 App Contract
Every Bazic UI app follows this contract:
- `app.bz` is the entrypoint.
- `pages/` contains Bazic functions for pages.
- `index.html`, `theme.css`, `style.css`, `app.js` are static assets.

Required behaviors:
- `ui_render(tree)` renders a JSON UI tree.
- `ui_event_type`, `ui_event_target`, `ui_event_value` read the last event.
- `ui_event_clear()` is called after handling an event.
- `ui_route_get(default)` reads the current route.
- `ui_route_set(route)` updates the route.

V1 event loop rules:
1. Read the current state.
2. Render the UI.
3. Poll events and update state.
4. Clear events after handling.
5. Re-render after a state change.

## V1 Routing
- Hash routes only (`#home`, `#about`).
- UI runtime sends a `route` event when the hash changes.

## V1 State
- `ui_state_get` / `ui_state_set` for persistent UI state.
- `ui_state_int_get` / `ui_state_int_set` for int values.
- `ui_store_get` / `ui_store_set` for shared component state.

## V1 Files
```
my-ui/
  app.bz
  pages/
    home.bz
    about.bz
  index.html
  app.js
  theme.css
  style.css
```

## V1 Commands
```powershell
.\bin\bazic.exe ui init --dir .\my-ui
.\bin\bazic.exe ui page settings --dir .\my-ui
.\bin\bazic.exe ui build --dir .\my-ui
.\bin\bazic.exe ui dev --dir .\my-ui --port 8080
```

## V1 Non-Goals
- Complex framework diffing.
- Hidden implicit state.
- Build-time codegen beyond WASM build.
