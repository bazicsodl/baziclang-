package api

import (
	"os"
	"path/filepath"
	"strings"
)

type Endpoint struct {
	Method string
	Path   string
	Func   string
}

func GenerateHandlers(modelsPath string, routes []Endpoint) (string, error) {
	modelsData, _ := os.ReadFile(modelsPath)
	modelNames := collectStructs(string(modelsData))
	var b strings.Builder
	b.WriteString("import \"std\";\n\n")
	b.WriteString("const DB_PATH = \"app.db\";\n\n")
	b.WriteString("fn api_json_ok(body: string): ServerResponse { return http_json(200, body); }\n")
	b.WriteString("fn api_json_err(status: int, msg: string): ServerResponse { return http_json(status, \"{\\\"error\\\":\\\"\" + msg + \"\\\"}\"); }\n\n")
	b.WriteString("fn api_require_session(req: ServerRequest): Result[string,Error] {\n")
	b.WriteString("    return auth_session_user(DB_PATH, req, \"bazic_session\");\n")
	b.WriteString("}\n\n")
	for _, r := range routes {
		b.WriteString("fn ")
		b.WriteString(r.Func)
		b.WriteString("(req: ServerRequest): ServerResponse {\n")
		model, mode := inferModelForRoute(r.Path, modelNames)
		isPublic := strings.HasPrefix(r.Func, "public_")
		if r.Method != "GET" && !isPublic {
			b.WriteString("    let auth = api_require_session(req);\n")
			b.WriteString("    if !auth.is_ok { return api_json_err(401, \"unauthorized\"); }\n")
		}
		if model != "" && (r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH") {
			b.WriteString("    let parsed = ")
			b.WriteString(strings.ToLower(model))
			b.WriteString("_from_json(req.body);\n")
			b.WriteString("    if !parsed.is_ok { return api_json_err(400, parsed.err.message); }\n")
			if r.Method == "POST" {
				b.WriteString("    let res = ")
				b.WriteString(strings.ToLower(model))
				b.WriteString("_create(DB_PATH, parsed.value);\n")
				b.WriteString("    if !res.is_ok { return api_json_err(500, res.err.message); }\n")
				b.WriteString("    return api_json_ok(\"{\\\"id\\\":\" + str(res.value) + \"}\");\n")
			} else {
				b.WriteString("    let upd = ")
				b.WriteString(strings.ToLower(model))
				b.WriteString("_update(DB_PATH, parsed.value);\n")
				b.WriteString("    if !upd.is_ok { return api_json_err(500, upd.err.message); }\n")
				if model != "" {
					b.WriteString("    let out = ")
					b.WriteString(strings.ToLower(model))
					b.WriteString("_from_json(req.body);\n")
					b.WriteString("    if !out.is_ok { return api_json_err(500, \"serialize\"); }\n")
					b.WriteString("    return api_json_ok(req.body);\n")
				} else {
					b.WriteString("    return api_json_ok(\"{\\\"ok\\\":true}\");\n")
				}
			}
			b.WriteString("}\n\n")
			continue
		}
		if model != "" && r.Method == "GET" {
			if mode == "single" {
				b.WriteString("    let id = http_params_get(req.params, \"id\");\n")
				b.WriteString("    let pid = parse_int(id);\n")
				b.WriteString("    if !pid.is_ok { return api_json_err(400, \"invalid id\"); }\n")
				b.WriteString("    let res = ")
				b.WriteString(strings.ToLower(model))
				b.WriteString("_find_by_id(DB_PATH, pid.value);\n")
				b.WriteString("    if !res.is_ok { return api_json_err(404, res.err.message); }\n")
				b.WriteString("    let row = db_query_one_json(DB_PATH, \"select \" + ")
				b.WriteString(strings.ToLower(model))
				b.WriteString("_columns() + \" from ")
				b.WriteString(strings.ToLower(model))
				b.WriteString("s where id = \" + str(pid.value));\n")
				b.WriteString("    if !row.is_ok { return api_json_err(404, row.err.message); }\n")
				b.WriteString("    return api_json_ok(row.value);\n")
				b.WriteString("}\n\n")
				continue
			}
			b.WriteString("    let limit = parse_int(http_query_get(req.query, \"limit\"));\n")
			b.WriteString("    let offset = parse_int(http_query_get(req.query, \"offset\"));\n")
			b.WriteString("    let order = http_query_get(req.query, \"order\");\n")
			b.WriteString("    let dir = http_query_get(req.query, \"dir\");\n")
			b.WriteString("    let lim = 0; let off = 0;\n")
			b.WriteString("    if limit.is_ok { lim = limit.value; }\n")
			b.WriteString("    if offset.is_ok { off = offset.value; }\n")
			b.WriteString("    let asc = true; if to_lower(dir) == \"desc\" { asc = false; }\n")
			b.WriteString("    let res = ")
			b.WriteString(strings.ToLower(model))
			b.WriteString("_list_paged_json(DB_PATH, lim, off, order, asc);\n")
			b.WriteString("    if !res.is_ok { return api_json_err(500, res.err.message); }\n")
			b.WriteString("    return api_json_ok(res.value);\n")
			b.WriteString("}\n\n")
			continue
		}
		if model != "" && r.Method == "DELETE" {
			b.WriteString("    let id = http_params_get(req.params, \"id\");\n")
			b.WriteString("    let pid = parse_int(id);\n")
			b.WriteString("    if !pid.is_ok { return api_json_err(400, \"invalid id\"); }\n")
			b.WriteString("    let res = ")
			b.WriteString(strings.ToLower(model))
			b.WriteString("_delete(DB_PATH, pid.value);\n")
			b.WriteString("    if !res.is_ok { return api_json_err(500, res.err.message); }\n")
			b.WriteString("    return api_json_ok(\"{\\\"ok\\\":true}\");\n")
			b.WriteString("}\n\n")
			continue
		}
		b.WriteString("    return api_json_err(501, \"not implemented\");\n")
		b.WriteString("}\n\n")
	}
	return b.String(), nil
}

func collectStructs(src string) []string {
	lines := strings.Split(src, "\n")
	out := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "struct ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				out = append(out, parts[1])
			}
		}
	}
	return out
}

func inferModelForRoute(path string, models []string) (string, string) {
	segments := []string{}
	for _, seg := range strings.Split(path, "/") {
		if seg == "" {
			continue
		}
		segments = append(segments, seg)
	}
	if len(segments) == 0 {
		return "", ""
	}
	keys := map[string]string{}
	for _, m := range models {
		lower := strings.ToLower(m)
		keys[lower] = m
		keys[lower+"s"] = m
	}
	last := segments[len(segments)-1]
	if model, ok := keys[last]; ok {
		if strings.Contains(path, "{") {
			return model, "single"
		}
		return model, "list"
	}
	return "", ""
}

func WriteHandlers(outPath string, code string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(outPath, []byte(code), 0644)
}
