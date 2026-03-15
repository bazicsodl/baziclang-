# Bazic Stdlib (MVP)

This standard library MVP ships as a local Bazic package (`std`) and is imported with:

```bazic
import "std";
```

## Modules and APIs

### io
- `io_read_line(): Result[string, Error]`
- `io_read_all(): Result[string, Error]`

### fs
- `fs_read_file(path: string): Result[string, Error]`
- `fs_write_file(path: string, data: string): Result[bool, Error]`
- `fs_exists(path: string): bool`
- `fs_mkdir_all(path: string): Result[bool, Error]`
- `fs_remove(path: string): Result[bool, Error]`
- `fs_list_dir(path: string): Result[string, Error]`

### time
- `time_unix_millis(): int`
- `time_sleep_ms(ms: int): void`
- `time_now_rfc3339(): string`
- `time_add_days(rfc3339: string, days: int): Result[string, Error]`

### json
- `json_escape(s: string): string`
- `json_pretty(s: string): Result[string, Error]`
- `json_validate(s: string): bool`
- `json_minify(s: string): Result[string, Error]`
- `json_get_raw(s: string, key: string): Result[string, Error]`
- `json_get_string(s: string, key: string): Result[string, Error]`
- `json_get_bool(s: string, key: string): Result[bool, Error]`
- `json_get_int(s: string, key: string): Result[int, Error]`
- `json_get_float(s: string, key: string): Result[float, Error]`

### http
- `http_get(url: string): Result[string, Error]`
- `http_get_raw(url: string): Result[HttpResponse, Error]`
- `http_post(url: string, body: string): Result[string, Error]`
- `http_post_raw(url: string, body: string): Result[HttpResponse, Error]`
- `http_serve_text(addr: string, body: string): Result[bool, Error]`
- `http_serve_app(addr: string): Result[bool, Error]`
- `HttpOptions { connect_timeout_ms, timeout_ms, headers, user_agent, content_type, tls_insecure_skip_verify, tls_ca_bundle_pem }`
- `http_get_opts(url: string, opts: HttpOptions): Result[string, Error]`
- `http_post_opts(url: string, body: string, opts: HttpOptions): Result[string, Error]`
- `HttpRequest { method, url, body, opts }`
- `http_request(req: HttpRequest): Result[string, Error]`
- `HttpResponse { status, headers, body }`
- `http_get_resp(url: string): Result[HttpResponse, Error]`
- `http_get_opts_resp(url: string, opts: HttpOptions): Result[HttpResponse, Error]`
- `http_post_resp(url: string, body: string): Result[HttpResponse, Error]`
- `http_post_opts_resp(url: string, body: string, opts: HttpOptions): Result[HttpResponse, Error]`
- `http_request_resp(req: HttpRequest): Result[HttpResponse, Error]`
- `http_get_status(url: string): Result[int, Error]`
- `http_request_status(req: HttpRequest): Result[int, Error]`
- `http_status_ok(status: int): bool`
- `http_get_ok(url: string): Result[bool, Error]`
- `http_serve_json(addr: string, body: string): Result[bool, Error]`
- `ServerRequest { method, path, query, headers, body, remote_addr, cookies, params }`
- `ServerResponse { status, headers, body }`
- `http_response(status: int, body: string, content_type: string): ServerResponse`
- `http_text(status: int, body: string): ServerResponse`
- `http_json(status: int, json_body: string): ServerResponse`
- `http_header_get(headers: string, key: string): string`
- `http_cookie_get(cookies: string, key: string): string`
- `http_params_get(params: string, key: string): string`
- `http_query_get(query: string, key: string): string`

### crypto
- `crypto_sha256_hex(s: string): string`
- `crypto_hmac_sha256_hex(message: string, secret: string): string`
- `crypto_random_hex(n: int): Result[string, Error]`
- `crypto_bcrypt_hash(password: string, cost: int): Result[string, Error]`
- `crypto_bcrypt_verify(password: string, hash: string): Result[bool, Error]`

### auth
- `auth_hash_password(password: string): Result[string, Error]`
- `auth_verify_password(password: string, hash: string): Result[bool, Error]`
- `auth_session_token(): Result[string, Error]`
- `auth_cookie_header(name: string, value: string, max_age_days: int, secure: bool): string`
- `auth_cookie_clear_header(name: string, secure: bool): string`
- `auth_session_hash(token: string): string`
- `auth_session_create(db_path: string, user_id: string, days: int): Result[string, Error]`
- `auth_session_user(db_path: string, req: ServerRequest, cookie_name: string): Result[string, Error]`
- `auth_session_destroy(db_path: string, req: ServerRequest, cookie_name: string): Result[bool, Error]`

### jwt
- `jwt_b64url(s: string): string`
- `jwt_b64url_decode(s: string): Result[string, Error]`
- `jwt_sign_hs256(header_json: string, payload_json: string, secret: string): Result[string, Error]`
- `jwt_verify_hs256(token: string, secret: string): Result[bool, Error]`
- `csrf_token(): Result[string, Error]`

### session
- `session_init(path: string): Result[bool, Error]`
- `session_put(path: string, token_hash: string, user_id: string, expires_at_rfc3339: string): Result[bool, Error]`
- `session_get_user(path: string, token_hash: string): Result[string, Error]`
- `session_delete(path: string, token_hash: string): Result[bool, Error]`

### base64
- `base64_encode(s: string): string`
- `base64_decode(s: string): Result[string, Error]`

### db
- `db_exec(path: string, sql: string): Result[bool, Error]`
- `db_query(path: string, sql: string): Result[string, Error]`
- `db_exec_with(driver: string, dsn: string, sql: string): Result[bool, Error]`
- `db_query_with(driver: string, dsn: string, sql: string): Result[string, Error]`
- `db_query_json(path: string, sql: string): Result[string, Error]`
- `db_query_json_with(driver: string, dsn: string, sql: string): Result[string, Error]`
- `db_query_one_json(path: string, sql: string): Result[string, Error]`
- `db_query_one_json_with(driver: string, dsn: string, sql: string): Result[string, Error]`
- `db_exec_returning_id(path: string, sql: string): Result[int, Error]`
- `db_exec_returning_id_with(driver: string, dsn: string, sql: string): Result[int, Error]`
- `db_exec_params(path: string, sql: string, params: string): Result[bool, Error]`
- `db_exec_params_with(driver: string, dsn: string, sql: string, params: string): Result[bool, Error]`
- `db_query_params(path: string, sql: string, params: string): Result[string, Error]`
- `db_query_params_with(driver: string, dsn: string, sql: string, params: string): Result[string, Error]`
- `db_query_json_params(path: string, sql: string, params: string): Result[string, Error]`
- `db_query_json_params_with(driver: string, dsn: string, sql: string, params: string): Result[string, Error]`
- `db_query_one_json_params(path: string, sql: string, params: string): Result[string, Error]`
- `db_query_one_json_params_with(driver: string, dsn: string, sql: string, params: string): Result[string, Error]`
- `db_exec_returning_id_params(path: string, sql: string, params: string): Result[int, Error]`
- `db_exec_returning_id_params_with(driver: string, dsn: string, sql: string, params: string): Result[int, Error]`
- Migration tooling: `bazic migrate <create|apply|rollback|status>`
- Model schema tooling: `bazic model <init|generate|migrate>` (see `MODEL_SCHEMA.md`)

Driver compatibility:
- Go target: `sqlite`, `postgres`, `mysql` (aliases: `sqlite3`, `postgresql`)
- Native/LLVM target: `sqlite` / `sqlite3` when built with `BAZIC_SQLITE=1`

### collections
- `StringBuilder`
- `sb_new(): StringBuilder`
- `sb_append(sb: StringBuilder, s: string): StringBuilder`
- `sb_append_line(sb: StringBuilder, s: string): StringBuilder`
- `sb_string(sb: StringBuilder): string`
- `sb_len(sb: StringBuilder): int`
- `sb_clear(sb: StringBuilder): StringBuilder`

### sql
- `sql_ident(s: string): string`
- `sql_str(v: string): string`
- `sql_bool(v: bool): string`
- `sql_eq(field: string, value: string): string`
- `sql_and(a: string, b: string): string`
- `sql_or(a: string, b: string): string`
- `sql_where(cond: string): string`
- `sql_in(field: string, csv: string): string`
- `sql_like(field: string, pattern: string): string`
- `sql_value_string(v: string): string`
- `sql_value_int(v: int): string`
- `sql_value_float(v: float): string`
- `sql_value_bool(v: bool): string`
- `sql_param_str(v: string): string`
- `sql_param_int(v: int): string`
- `sql_param_float(v: float): string`
- `sql_param_bool(v: bool): string`
- `sql_param_null(): string`
- `sql_params_add(params: string, value: string): string`

### validate
- `validate_required(s: string, field: string): Result[bool, Error]`
- `validate_min_len(s: string, min: int, field: string): Result[bool, Error]`
- `validate_max_len(s: string, max: int, field: string): Result[bool, Error]`
- `validate_email(s: string, field: string): Result[bool, Error]`
- `validate_int_range(v: int, min: int, max: int, field: string): Result[bool, Error]`
- `validate_float_range(v: float, min: float, max: float, field: string): Result[bool, Error]`
- `validate_enum(v: string, allowed_csv: string): Result[bool, Error]`

### desktop
- `desktop_open_url(url: string): Result[bool, Error]`

### os
- `os_args(): string` (newline-separated args, excluding program name)
- `os_getenv(key: string): Result[string, Error]`
- `os_cwd(): Result[string, Error]`
- `os_chdir(path: string): Result[bool, Error]`
- `os_env_list(): Result[string, Error]`
- `os_temp_dir(): Result[string, Error]`
- `os_exe_path(): Result[string, Error]`
- `os_home_dir(): Result[string, Error]`

### web
- `web_get_json(key: string): Result[string, Error]` (WASM only)
- `web_set_json(key: string, jsonText: string): Result[bool, Error]` (WASM only)

### ui
- `ui_text(value: string): string`
- `ui_props(id: string, class_name: string): string`
- `ui_props_key(id: string, class_name: string, key: string): string`
- `ui_props_for(id: string, class_name: string, for_id: string): string`
- `ui_class_join(a: string, b: string): string`
- `ui_nav_link(label: string, id: string, is_active: bool): string`
- `ui_badge(text: string, tone: string): string`
- `ui_tab_button(label: string, id: string, is_active: bool): string`
- `ui_tabs(tabs_json: string): string`
- `ui_card(title: string, body_json: string): string`
- `ui_list(items_json: string): string`
- `ui_list_item(text: string): string`
- `ui_table(head_json: string, body_json: string): string`
- `ui_table_class(head_json: string, body_json: string, class_name: string): string`
- `ui_table_row(cells_json: string): string`
- `ui_table_cell(text: string): string`
- `ui_table_head_cell(text: string): string`
- `ui_table_head_cell_sort(label: string, id: string, dir: string): string`
- `ui_table_cell_span(text: string, span: int): string`
- `ui_menu(items_json: string): string`
- `ui_menu_item(label: string, id: string, is_active: bool): string`
- `ui_dropdown(label: string, id: string, is_open: bool, items_json: string): string`
- `ui_modal(title: string, body_json: string, is_open: bool): string`
- `ui_section(title: string, body_json: string): string`
- `ui_form_row(label: string, input_json: string): string`
- `ui_toast(message: string, tone: string): string`
- `ui_toast_stack(toasts_json: string): string`
- `ui_empty(text: string): string`
- `ui_page(page: string, name: string, body_json: string): string`
- `ui_element(tag: string, props_json: string, children_json: string): string`
- `ui_children_one/two/three/four(...)`
- `ui_div`, `ui_h1`, `ui_p`, `ui_label`, `ui_button`, `ui_input`, `ui_input_value`
- `ui_input_value_aria(placeholder, id, class_name, value, aria_label)`
- `ui_option(value: string, label: string, selected: bool): string`
- `ui_select(id: string, class_name: string, options_json: string): string`
- `ui_select_value(id: string, class_name: string, value: string, options_json: string): string`
- `ui_checkbox(id: string, class_name: string, checked: bool): string`
- `ui_switch(id: string, checked: bool): string`
- `ui_range(id: string, value: int, min: int, max: int): string`
- `ui_render(tree_json: string): Result[bool, Error]`
- `ui_event_poll(): string`
- `ui_event_type(): string`, `ui_event_target(): string`, `ui_event_value(): string`, `ui_event_action(): string`, `ui_event_clear(): void`
- `ui_state_get(key: string, fallback: string): string`
- `ui_state_set(key: string, value: string): Result[bool, Error]`
- `ui_state_int_get(key: string, fallback: int): int`
- `ui_state_int_set(key: string, value: int): Result[bool, Error]`
- `ui_route_get(fallback: string): string`
- `ui_route_set(route: string): Result[bool, Error]`
- `ui_store_get(key: string, fallback: string): string`
- `ui_store_set(key: string, value: string): Result[bool, Error]`
- `ui_store_int_get(key: string, fallback: int): int`
- `ui_store_int_set(key: string, value: int): Result[bool, Error]`
- `ui_focus_get(): string`
- `ui_focus_set(target_id: string): Result[bool, Error]`
- `ui_aria_label(text: string): string`
- `ui_props_aria(id: string, class_name: string, aria_label: string): string`
- `ui_props_role(id: string, class_name: string, role: string): string`
- `ui_app_root(children_json: string): string`
- `ui_component(name: string, props_json: string, children_json: string): string`
- `ui_loop_delay_ms(): int`
- `ui_layout(title: string, nav_json: string, body_json: string): string`
- `ui_render_from_state(title: string, subtitle: string, clicks: int, page: string): string`
- `ui_bind_input(key: string, placeholder: string, id: string): string`

### path
- `path_basename(path: string): string`
- `path_dirname(path: string): string`
- `path_join(a: string, b: string): string`

## Examples

### Read a file
```bazic
import "std";

fn main(): void {
    let res = fs_read_file("README.md");
    if res.is_ok {
        println(res.value);
    } else {
        println(res.err.message);
    }
}
```

### Paths
```bazic
import "std";

fn main(): void {
    let base = path_basename("C:\\Users\\Ipeh\\file.txt");
    let dir = path_dirname("/home/user/file.txt");
    let full = path_join(dir, "next.txt");
    println(base);
    println(dir);
    println(full);
}
```

### JSON pretty-print
```bazic
import "std";

fn main(): void {
    let res = json_pretty("{\"a\":1}");
    if res.is_ok {
        println(res.value);
    } else {
        println(res.err.message);
    }
}
```

### JSON validate + minify
```bazic
import "std";

fn main(): void {
    let ok = json_validate("{\"a\":1}");
    println(ok);
    let min = json_minify("{\"a\": 1}");
    if min.is_ok {
        println(min.value);
    }
}
```

### Simple HTTP server
```bazic
import "std";

fn main(): void {
    let res = http_serve_text(":8080", "hello from bazic");
    if !res.is_ok {
        println(res.err.message);
    }
}
```

### SQLite (TSV output)
```bazic
import "std";

fn main(): void {
    let ok = db_exec("app.db", "create table if not exists users(id int, name string);");
    if !ok.is_ok { println(ok.err.message); return; }
    let _ = db_exec("app.db", "insert into users values (1, 'Ada');");
    let res = db_query("app.db", "select id, name from users;");
    if res.is_ok {
        println(res.value); // columns and rows as TSV
    } else {
        println(res.err.message);
    }
}
```

### Postgres example
```bazic
import "std";

fn main(): void {
    let dsn = "postgres://user:pass@localhost:5432/app?sslmode=disable";
    let ok = db_exec_with("postgres", dsn, "create table if not exists users(id int, name text);");
    if !ok.is_ok { println(ok.err.message); return; }
    let res = db_query_with("postgres", dsn, "select id, name from users;");
    if res.is_ok {
        println(res.value);
    } else {
        println(res.err.message);
    }
}
```

### MySQL example
```bazic
import "std";

fn main(): void {
    let dsn = "user:pass@tcp(127.0.0.1:3306)/app";
    let ok = db_exec_with("mysql", dsn, "create table if not exists users(id int, name varchar(255));");
    if !ok.is_ok { println(ok.err.message); return; }
    let res = db_query_with("mysql", dsn, "select id, name from users;");
    if res.is_ok {
        println(res.value);
    } else {
        println(res.err.message);
    }
}
```

### Base64
```bazic
import "std";

fn main(): void {
    let enc = base64_encode("hello");
    println(enc);
    let dec = base64_decode(enc);
    if dec.is_ok {
        println(dec.value);
    }
}
```

### HTTP with options
```bazic
import "std";

fn main(): void {
    let opts = HttpOptions{
        connect_timeout_ms: 2000,
        timeout_ms: 8000,
        headers: "Accept: application/json\nX-Env: dev",
        user_agent: "BazicClient/0.1",
        content_type: "application/json",
        tls_insecure_skip_verify: false,
        tls_ca_bundle_pem: "",
    };
    let res = http_post_opts("https://httpbin.org/post", "{\"ok\":true}", opts);
    if res.is_ok {
        println(res.value);
    } else {
        println(res.err.message);
    }
}
```

### Args + Env
```bazic
import "std";

fn main(): void {
    let args = os_args();
    println("Args:\n" + args);
    let home = os_getenv("HOME");
    if home.is_ok {
        println("HOME=" + home.value);
    }
    let cwd = os_cwd();
    if cwd.is_ok {
        println("CWD=" + cwd.value);
    }
    let exe = os_exe_path();
    if exe.is_ok {
        println("EXE=" + exe.value);
    }
}
```

### Web interop (WASM)
```bazic
import "std";

fn main(): void {
    let payload = web_get_json("payload");
    if payload.is_ok {
        println(payload.value);
    }
}
```

### UI layout (WASM)
```bazic
import "std";

fn main(): void {
    let nav = ui_element("nav", ui_props("nav", "nav"),
        ui_children_two(
            ui_nav_link("Home", "nav-home", true),
            ui_nav_link("About", "nav-about", false)
        )
    );
    let body = ui_element("div", ui_props("", "content"),
        ui_children_two(ui_p("Hello from Bazic UI"), ui_button("Click", "btn", "ghost"))
    );
    let root = ui_layout("Bazic UI", nav, body);
    let _ = ui_render(root);
}
```

### Custom HTTP method
```bazic
import "std";

fn main(): void {
    let req = HttpRequest{
        method: "PATCH",
        url: "https://httpbin.org/patch",
        body: "{\"ok\":true}",
        opts: http_options_default(),
    };
    let res = http_request(req);
    if res.is_ok {
        println(res.value);
    } else {
        println(res.err.message);
    }
}
```

### Response Details
```bazic
import "std";

fn main(): void {
    let res = http_get_resp("https://httpbin.org/get");
    if res.is_ok {
        println("Status: " + str(res.value.status));
        println("Headers:\n" + res.value.headers);
        println("Body:\n" + res.value.body);
    } else {
        println(res.err.message);
    }
}
```
