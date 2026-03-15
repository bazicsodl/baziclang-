# Bazic UI Guide (v1)

This guide shows the Bazic UI flow: init, state, routing, render.

## 1) Create the UI App
```powershell
.\bin\bazic.exe ui init --dir .\my-ui
.\bin\bazic.exe ui build --dir .\my-ui
.\bin\bazic.exe ui dev --dir .\my-ui --port 8080
```

## 2) Add a Page
```powershell
.\bin\bazic.exe ui page settings/profile --dir .\my-ui
```
This creates `pages/settings/profile.bz` and wires routing.

## 3) Routing
```bazic
let page = ui_route_get("home");
// ...
if ev == "route" { page = val; }
```

## 4) State + Store
```bazic
let name = ui_state_get("name", "");
let _ = ui_state_set("name", "Ada");
let note = ui_store_get("note", "");
let _ = ui_store_set("note", "hello");
```

## 5) Events
```bazic
let ev = ui_event_type();
let target = ui_event_target();
let val = ui_event_value();
if ev == "click" && target == "btn" {
    ui_event_clear();
}
```

## 6) Components
Use `ui_card`, `ui_badge`, `ui_list`, `ui_modal` for layout blocks.
Use `ui_form_row`, `ui_select`, `ui_switch` for forms.

## 7) Theme
Edit `theme.css` tokens like `--ui-bg`, `--ui-accent`.

## 8) Cookbook
See `BAZIC_UI_COOKBOOK.md` for UI patterns.

## 9) Frontend Contract
See `BAZIC_UI_FRONTEND_V1.md` for the full v1 contract.
