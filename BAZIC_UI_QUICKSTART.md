# Bazic UI Quickstart

This is the shortest path to a working Bazic UI app.

## 1) Create a UI app
```powershell
.\bin\bazic.exe ui init --dir .\my-ui
```

## 2) Run the dev server
```powershell
.\bin\bazic.exe ui dev --dir .\my-ui --port 8080
```

## 3) Edit the app
- `app.bz` owns routing, state, and the event loop.
- `pages/home.bz` and `pages/about.bz` are plain Bazic functions.

## 4) Add a new page
```powershell
.\bin\bazic.exe ui page settings --dir .\my-ui
```

The new page is wired into `app.bz` automatically.

## 5) Build
```powershell
.\bin\bazic.exe ui build --dir .\my-ui
```

## UI loop rules
- Events come from `ui_event_type`, `ui_event_target`, `ui_event_value`.
- Call `ui_event_clear()` after handling each event.
- Re-render after state changes.
