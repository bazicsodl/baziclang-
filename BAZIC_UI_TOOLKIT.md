# Bazic UI Toolkit

This is the straight-line workflow for building frontend apps with Bazic UI.

## 1) Create
```powershell
.\bin\bazic.exe ui init --dir .\my-ui
```

## 2) Run dev
```powershell
.\bin\bazic.exe ui dev --dir .\my-ui --port 8080
```

## 3) Add pages
```powershell
.\bin\bazic.exe ui page settings --dir .\my-ui
.\bin\bazic.exe ui page billing --dir .\my-ui
```

## 4) Add components
```powershell
.\bin\bazic.exe ui component card --dir .\my-ui
.\bin\bazic.exe ui component hero --dir .\my-ui
```

## 5) Build
```powershell
.\bin\bazic.exe ui build --dir .\my-ui
```

## Page skeleton
```bazic
import "std";

fn page_home(clicks: int, name: string): string {
    let line1 = "Welcome to Bazic UI.";
    let line2 = "Clicks: " + str(clicks);
    let body = ui_element("div", ui_props("", "stack"), ui_children_two(ui_p(line1), ui_p(line2)));
    return ui_section("Home", body);
}
```

## Event loop skeleton
```bazic
let clicks = ui_state_int_get("clicks", 0);
let page = ui_route_get("home");
let name = ui_state_get("name", "");
render_ui(page, clicks, name);

let i = 0;
while i < 24 {
    let ev = ui_event_type();
    let target = ui_event_target();
    let val = ui_event_value();
    if ev == "click" && target == "btn" {
        clicks = clicks + 1;
        ui_event_clear();
        let _ = ui_state_int_set("clicks", clicks);
    }
    if ev == "input" && target == "name" {
        ui_event_clear();
        name = val;
        let _ = ui_state_set("name", name);
    }
    if ev == "nav" {
        ui_event_clear();
        if val == "nav-home" { page = "home"; }
        if val == "nav-about" { page = "about"; }
        let _ = ui_state_set("page", page);
        let _ = ui_route_set(page);
    }
    if ev == "route" && val != "" {
        ui_event_clear();
        page = val;
        let _ = ui_state_set("page", page);
    }
    render_ui(page, clicks, name);
    i = i + 1;
}
```

## Layout helper
```bazic
fn render_ui(page: string, clicks: int, name: string): void {
    let nav = ui_element("nav", ui_props("nav", "nav"),
        ui_children_two(
            ui_nav_link("Home", "nav-home", page == "home"),
            ui_nav_link("About", "nav-about", page == "about")
        )
    );
    let hero = ui_element("div", ui_props("", "hero"),
        ui_children_three(
            ui_button("Click Me", "btn", "ghost"),
            ui_bind_input("name", "Type your name", "name"),
            ui_p("Clicks: " + str(clicks))
        )
    );
    let content = ui_element("div", ui_props("", "content"), ui_children_two(hero, render_page(page, clicks, name)));
    let root = ui_layout("Bazic UI", nav, content);
    let _ = ui_render(root);
}
```
