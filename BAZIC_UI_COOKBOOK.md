# Bazic UI Cookbook

Practical UI patterns you can copy into your Bazic UI apps.

## 1) Hero + CTA
```bazic
let hero = ui_element("div", ui_props("", "stack"),
    ui_children_three(
        ui_element("h2", ui_props("", "section-title"), ui_children_one(ui_text("Launch fast"))),
        ui_p("A minimal UI DSL for WASM apps."),
        ui_button("Get Started", "cta", "ghost")
    )
);
```

## 2) Auth Card
```bazic
let email = ui_input_value_aria("Email", "login-email", "", ui_state_get("login.email", ""), "Email");
let password = ui_input_value_aria("Password", "login-pass", "", ui_state_get("login.pass", ""), "Password");
let auth = ui_card("Sign In", ui_element("div", ui_props("", "stack"),
    ui_children_three(
        ui_form_row("Email", email),
        ui_form_row("Password", password),
        ui_button("Sign In", "login-submit", "ghost")
    )
));
```

## 3) Card Grid
```bazic
let cards = ui_element("div", ui_props("", "grid"),
    ui_children_three(
        ui_card("Latency", ui_p("120ms")),
        ui_card("Errors", ui_p("0.3%")),
        ui_card("Traffic", ui_p("12.4k"))
    )
);
```

## 4) Table
```bazic
let head = ui_table_row(ui_children_three(
    ui_table_head_cell("Service"),
    ui_table_head_cell("State"),
    ui_table_head_cell("Latency")
));
let row = ui_table_row(ui_children_three(
    ui_table_cell("API"),
    ui_table_cell("Healthy"),
    ui_table_cell("120ms")
));
let table = ui_table(head, ui_children_one(row));
```

## 5) Tabs
```bazic
let tab = ui_state_get("tab", "overview");
let tabs = ui_tabs(ui_children_three(
    ui_tab_button("Overview", "tab-overview", tab == "overview"),
    ui_tab_button("Stats", "tab-stats", tab == "stats"),
    ui_tab_button("Alerts", "tab-alerts", tab == "alerts")
));
```

## 6) Form Row + Select
```bazic
let opt1 = ui_option("starter", "Starter", true);
let opt2 = ui_option("team", "Team", false);
let select = ui_select("plan", "select", ui_children_two(opt1, opt2));
let row = ui_form_row("Plan", select);
```

## 7) Settings Block
```bazic
let dark = ui_switch("dark-mode", ui_state_get("dark-mode", "false") == "true");
let alerts = ui_switch("alerts", ui_state_get("alerts", "true") == "true");
let settings = ui_card("Settings", ui_element("div", ui_props("", "stack"),
    ui_children_two(
        ui_form_row("Dark mode", dark),
        ui_form_row("Alerts", alerts)
    )
));
```

## 8) Dropdown Menu
```bazic
let open = ui_state_get("menu.open", "false") == "true";
let pick = ui_state_get("menu.pick", "Starter");
let menu = ui_dropdown(pick, "menu-toggle", open, ui_children_three(
    ui_menu_item("Starter", "menu-starter", pick == "Starter"),
    ui_menu_item("Team", "menu-team", pick == "Team"),
    ui_menu_item("Enterprise", "menu-enterprise", pick == "Enterprise")
));
```

## 9) Switch + Range
```bazic
let dark = ui_switch("dark-mode", ui_state_get("dark-mode", "false") == "true");
let volume = ui_range("volume", ui_state_int_get("volume", 30), 0, 100);
```

## 10) Toast Stack
```bazic
let toast = ui_toast("Saved", "tone-accent");
let stack = ui_toast_stack(ui_children_one(toast));
```

## 11) Modal
```bazic
let body = ui_element("div", ui_props("", "stack"),
    ui_children_two(ui_p("Confirm changes?"), ui_button("Okay", "confirm", "ghost"))
);
let modal = ui_modal("Confirm", body, true);
```
