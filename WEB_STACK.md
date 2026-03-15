# Bazic Web Stack (Server Routing)

Bazic provides a convention-based HTTP server with native routing via `http_serve_app`.

## Quickstart
```powershell
.\bin\bazic.exe run .\examples\apps\webstack\main.bz
```

## Server API
- `http_serve_app(addr: string): Result[bool, Error]`
- `ServerRequest { method, path, query, headers, body, remote_addr, cookies, params }`
- `ServerResponse { status, headers, body }`
- Helpers: `http_response`, `http_text`, `http_json`, `http_header_get`, `http_cookie_get`, `http_params_get`, `http_query_get`

## Routing Convention
Handlers are discovered by name and must match the signature:
```
fn <method>_<path>(req: ServerRequest): ServerResponse { ... }
```

### Method tokens
- `get`, `post`, `put`, `patch`, `delete`, `options`

### Path tokens
- `_root` maps to `/`
- `_segment` maps to `/segment`
- `_p_<name>` maps to `/:name` (path param)

### Examples
- `get_root` => `GET /`
- `get_health` => `GET /health`
- `get_posts_p_id` => `GET /posts/:id`
- `post_users_p_userId_posts` => `POST /users/:userId/posts`

## Params, Cookies, Headers
- `req.params` is newline-separated `key=value` pairs.
- `req.cookies` is newline-separated `key=value` pairs.
- `req.headers` is newline-separated `Key: Value` pairs.

Use helpers:
- `http_params_get(req.params, "id")`
- `http_cookie_get(req.cookies, "session")`
- `http_header_get(req.headers, "Content-Type")`
- `http_query_get(req.query, "page")`

## Notes
- `http_serve_app` is implemented in the Go and LLVM backends.

## Runtime Limits
`http_serve_app` applies sane defaults and allows overrides via env vars:
- `BAZIC_HTTP_MAX_BODY` (default `1048576` bytes)
- `BAZIC_HTTP_READ_TIMEOUT_MS` (default `10000`)
- `BAZIC_HTTP_READ_HEADER_TIMEOUT_MS` (default `5000`)
- `BAZIC_HTTP_WRITE_TIMEOUT_MS` (default `15000`)
- `BAZIC_HTTP_IDLE_TIMEOUT_MS` (default `60000`)

LLVM backend honors `BAZIC_HTTP_MAX_BODY`, `BAZIC_HTTP_READ_TIMEOUT_MS`, and `BAZIC_HTTP_WRITE_TIMEOUT_MS`.
