package codegen

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"baziclang/internal/ast"
)

func GenerateGo(p *ast.Program) (string, error) {
	needsSession := programUsesSession(p)
	needsDB := programUsesDB(p) || needsSession
	needsBcrypt := programUsesBcrypt(p)
	needsJwt := programUsesJwt(p)
	needsHmac := programUsesHmac(p) || needsJwt
	hasHttpResponse := false
	for _, d := range p.Decls {
		if s, ok := d.(*ast.StructDecl); ok && s.Name == "HttpResponse" {
			hasHttpResponse = true
			break
		}
	}
	handlers := collectHttpHandlers(p)
	needsHttpServeApp := programUsesHttpServeApp(p) || len(handlers) > 0
	needsHeaderString := hasHttpResponse || len(handlers) > 0
	var b strings.Builder
	b.WriteString("package main\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"bufio\"\n")
	b.WriteString("\t\"bytes\"\n")
	b.WriteString("\t\"crypto/rand\"\n")
	b.WriteString("\t\"crypto/sha256\"\n")
	if needsHmac {
		b.WriteString("\t\"crypto/hmac\"\n")
	}
	b.WriteString("\t\"crypto/tls\"\n")
	b.WriteString("\t\"crypto/x509\"\n")
	b.WriteString("\t\"encoding/base64\"\n")
	b.WriteString("\t\"encoding/hex\"\n")
	b.WriteString("\t\"encoding/json\"\n")
	b.WriteString("\t\"fmt\"\n")
	b.WriteString("\t\"io\"\n")
	b.WriteString("\t\"math\"\n")
	b.WriteString("\t\"net\"\n")
	b.WriteString("\t\"net/http\"\n")
	b.WriteString("\t\"net/url\"\n")
	b.WriteString("\t\"os\"\n")
	b.WriteString("\t\"os/exec\"\n")
	b.WriteString("\t\"path/filepath\"\n")
	b.WriteString("\t\"strconv\"\n")
	b.WriteString("\t\"strings\"\n")
	b.WriteString("\t\"time\"\n")
	b.WriteString("\t\"unicode/utf8\"\n")
	b.WriteString("\t\"runtime\"\n")
	if needsBcrypt {
		b.WriteString("\t\"golang.org/x/crypto/bcrypt\"\n")
	}
	if needsSession {
		b.WriteString("\t\"sync\"\n")
	}
	target := strings.ToLower(strings.TrimSpace(os.Getenv("BAZIC_TARGET")))
	if target == "wasm" {
		b.WriteString("\t\"syscall/js\"\n")
	}
	if needsDB {
		b.WriteString("\t\"database/sql\"\n")
	}
	if needsDB && target != "wasm" {
		b.WriteString("\t_ \"github.com/go-sql-driver/mysql\"\n")
		b.WriteString("\t_ \"github.com/lib/pq\"\n")
		b.WriteString("\t_ \"modernc.org/sqlite\"\n")
	}
	b.WriteString(")\n\n")
	if needsSession {
		b.WriteString("type __bazic_session_entry struct { UserID string; ExpiresAt time.Time }\n")
		b.WriteString("var __bazic_session_mu sync.Mutex\n")
		b.WriteString("var __bazic_session_store = map[string]__bazic_session_entry{}\n\n")
	}
	b.WriteString("func print(v any) { fmt.Print(v) }\n")
	b.WriteString("func println(v any) { fmt.Println(v) }\n")
	b.WriteString("func str(v any) string { return fmt.Sprint(v) }\n")
	b.WriteString("func bazic_len(s string) int64 { return int64(utf8.RuneCountInString(s)) }\n")
	b.WriteString("func contains(s string, sub string) bool { return strings.Contains(s, sub) }\n")
	b.WriteString("func starts_with(s string, prefix string) bool { return strings.HasPrefix(s, prefix) }\n")
	b.WriteString("func ends_with(s string, suffix string) bool { return strings.HasSuffix(s, suffix) }\n")
	b.WriteString("func to_upper(s string) string { return strings.ToUpper(s) }\n")
	b.WriteString("func to_lower(s string) string { return strings.ToLower(s) }\n")
	b.WriteString("func trim_space(s string) string { return strings.TrimSpace(s) }\n")
	b.WriteString("func replace(s string, old string, new string) string { return strings.ReplaceAll(s, old, new) }\n")
	b.WriteString("func repeat(s string, count int64) string {\n")
	b.WriteString("\tif count <= 0 { return \"\" }\n")
	b.WriteString("\tmax := int64(^uint(0) >> 1)\n")
	b.WriteString("\tif count > max { count = max }\n")
	b.WriteString("\treturn strings.Repeat(s, int(count))\n")
	b.WriteString("}\n")
	b.WriteString("func parse_int(s string) Result[int64, Error] {\n")
	b.WriteString("\tv, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[int64, Error]{Is_ok: true, Value: v, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func parse_float(s string) Result[float64, Error] {\n")
	b.WriteString("\tv, err := strconv.ParseFloat(strings.TrimSpace(s), 64)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[float64, Error]{Is_ok: false, Value: 0.0, Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[float64, Error]{Is_ok: true, Value: v, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n\n")
	b.WriteString("func __std_read_file(path string) Result[string, Error] {\n")
	b.WriteString("\tdata, err := os.ReadFile(path)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: string(data), Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_write_file(path string, data string) Result[bool, Error] {\n")
	b.WriteString("\tif err := os.WriteFile(path, []byte(data), 0644); err != nil {\n")
	b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_read_line() Result[string, Error] {\n")
	b.WriteString("\treader := bufio.NewReader(os.Stdin)\n")
	b.WriteString("\tline, err := reader.ReadString('\\n')\n")
	b.WriteString("\tif err != nil && err != io.EOF {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tline = strings.TrimRight(line, \"\\r\\n\")\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: line, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_read_all() Result[string, Error] {\n")
	b.WriteString("\tdata, err := io.ReadAll(os.Stdin)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: string(data), Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_exists(path string) bool {\n")
	b.WriteString("\t_, err := os.Stat(path)\n")
	b.WriteString("\treturn err == nil\n")
	b.WriteString("}\n")
	b.WriteString("func __std_mkdir_all(path string) Result[bool, Error] {\n")
	b.WriteString("\tif err := os.MkdirAll(path, 0755); err != nil {\n")
	b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_remove(path string) Result[bool, Error] {\n")
	b.WriteString("\tif err := os.RemoveAll(path); err != nil {\n")
	b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_list_dir(path string) Result[string, Error] {\n")
	b.WriteString("\tentries, err := os.ReadDir(path)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tparts := make([]string, 0, len(entries))\n")
	b.WriteString("\tfor _, e := range entries {\n")
	b.WriteString("\t\tparts = append(parts, e.Name())\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: strings.Join(parts, \"\\n\"), Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_unix_millis() int64 { return time.Now().UnixMilli() }\n")
	b.WriteString("func __std_sleep_ms(ms int64) { time.Sleep(time.Duration(ms) * time.Millisecond) }\n")
	b.WriteString("func __std_now_rfc3339() string { return time.Now().UTC().Format(time.RFC3339) }\n")
	b.WriteString("func __std_time_add_days(rfc3339 string, days int64) Result[string, Error] {\n")
	b.WriteString("\tt, err := time.Parse(time.RFC3339, strings.TrimSpace(rfc3339))\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tout := t.Add(time.Duration(days) * 24 * time.Hour).UTC().Format(time.RFC3339)\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: out, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_json_escape(s string) string {\n")
	b.WriteString("\tenc, _ := json.Marshal(s)\n")
	b.WriteString("\tif len(enc) >= 2 {\n")
	b.WriteString("\t\treturn string(enc[1:len(enc)-1])\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn \"\"\n")
	b.WriteString("}\n")
	b.WriteString("func __std_json_validate(s string) bool {\n")
	b.WriteString("\treturn json.Valid([]byte(s))\n")
	b.WriteString("}\n")
	b.WriteString("func __std_json_pretty(s string) Result[string, Error] {\n")
	b.WriteString("\tvar v any\n")
	b.WriteString("\tif err := json.Unmarshal([]byte(s), &v); err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tout, err := json.MarshalIndent(v, \"\", \"  \")\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: string(out), Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_json_minify(s string) Result[string, Error] {\n")
	b.WriteString("\tif !json.Valid([]byte(s)) {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"invalid json\"}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tvar buf bytes.Buffer\n")
	b.WriteString("\tif err := json.Compact(&buf, []byte(s)); err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: buf.String(), Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_json_get_value(s string, key string) (any, error) {\n")
	b.WriteString("\tdec := json.NewDecoder(strings.NewReader(s))\n")
	b.WriteString("\tdec.UseNumber()\n")
	b.WriteString("\ttok, err := dec.Token()\n")
	b.WriteString("\tif err != nil { return nil, err }\n")
	b.WriteString("\tif d, ok := tok.(json.Delim); !ok || d != '{' { return nil, fmt.Errorf(\"invalid json\") }\n")
	b.WriteString("\tfor dec.More() {\n")
	b.WriteString("\t\tk, err := dec.Token()\n")
	b.WriteString("\t\tif err != nil { return nil, err }\n")
	b.WriteString("\t\tks, ok := k.(string)\n")
	b.WriteString("\t\tif !ok { return nil, fmt.Errorf(\"invalid json\") }\n")
	b.WriteString("\t\tvar v any\n")
	b.WriteString("\t\tif err := dec.Decode(&v); err != nil { return nil, err }\n")
	b.WriteString("\t\tif ks == key { return v, nil }\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn nil, fmt.Errorf(\"key not found\")\n")
	b.WriteString("}\n")
	b.WriteString("func __std_json_get_raw(s string, key string) Result[string, Error] {\n")
	b.WriteString("\tv, err := __std_json_get_value(s, key)\n")
	b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
	b.WriteString("\tout, err := json.Marshal(v)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: string(out), Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_json_get_string(s string, key string) Result[string, Error] {\n")
	b.WriteString("\tv, err := __std_json_get_value(s, key)\n")
	b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
	b.WriteString("\tstr, ok := v.(string)\n")
	b.WriteString("\tif !ok {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"not a string\"}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: str, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_json_get_bool(s string, key string) Result[bool, Error] {\n")
	b.WriteString("\tv, err := __std_json_get_value(s, key)\n")
	b.WriteString("\tif err != nil { return Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}} }\n")
	b.WriteString("\tb, ok := v.(bool)\n")
	b.WriteString("\tif !ok {\n")
	b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: \"not a bool\"}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: b, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_json_get_int(s string, key string) Result[int64, Error] {\n")
	b.WriteString("\tv, err := __std_json_get_value(s, key)\n")
	b.WriteString("\tif err != nil { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}} }\n")
	b.WriteString("\tswitch n := v.(type) {\n")
	b.WriteString("\tcase float64:\n")
	b.WriteString("\t\tif math.Trunc(n) != n { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: \"not an int\"}} }\n")
	b.WriteString("\t\treturn Result[int64, Error]{Is_ok: true, Value: int64(n), Err: Error{Message: \"\"}}\n")
	b.WriteString("\tcase json.Number:\n")
	b.WriteString("\t\tparsed, err := n.Int64()\n")
	b.WriteString("\t\tif err != nil { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: \"not an int\"}} }\n")
	b.WriteString("\t\treturn Result[int64, Error]{Is_ok: true, Value: parsed, Err: Error{Message: \"\"}}\n")
	b.WriteString("\tcase int64:\n")
	b.WriteString("\t\treturn Result[int64, Error]{Is_ok: true, Value: n, Err: Error{Message: \"\"}}\n")
	b.WriteString("\tcase int:\n")
	b.WriteString("\t\treturn Result[int64, Error]{Is_ok: true, Value: int64(n), Err: Error{Message: \"\"}}\n")
	b.WriteString("\tcase string:\n")
	b.WriteString("\t\tparsed, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)\n")
	b.WriteString("\t\tif err != nil { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: \"not an int\"}} }\n")
	b.WriteString("\t\treturn Result[int64, Error]{Is_ok: true, Value: parsed, Err: Error{Message: \"\"}}\n")
	b.WriteString("\tdefault:\n")
	b.WriteString("\t\treturn Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: \"not an int\"}}\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_json_get_float(s string, key string) Result[float64, Error] {\n")
	b.WriteString("\tv, err := __std_json_get_value(s, key)\n")
	b.WriteString("\tif err != nil { return Result[float64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}} }\n")
	b.WriteString("\tswitch n := v.(type) {\n")
	b.WriteString("\tcase float64:\n")
	b.WriteString("\t\treturn Result[float64, Error]{Is_ok: true, Value: n, Err: Error{Message: \"\"}}\n")
	b.WriteString("\tcase json.Number:\n")
	b.WriteString("\t\tparsed, err := n.Float64()\n")
	b.WriteString("\t\tif err != nil { return Result[float64, Error]{Is_ok: false, Value: 0, Err: Error{Message: \"not a float\"}} }\n")
	b.WriteString("\t\treturn Result[float64, Error]{Is_ok: true, Value: parsed, Err: Error{Message: \"\"}}\n")
	b.WriteString("\tcase int64:\n")
	b.WriteString("\t\treturn Result[float64, Error]{Is_ok: true, Value: float64(n), Err: Error{Message: \"\"}}\n")
	b.WriteString("\tcase int:\n")
	b.WriteString("\t\treturn Result[float64, Error]{Is_ok: true, Value: float64(n), Err: Error{Message: \"\"}}\n")
	b.WriteString("\tcase string:\n")
	b.WriteString("\t\tparsed, err := strconv.ParseFloat(strings.TrimSpace(n), 64)\n")
	b.WriteString("\t\tif err != nil { return Result[float64, Error]{Is_ok: false, Value: 0, Err: Error{Message: \"not a float\"}} }\n")
	b.WriteString("\t\treturn Result[float64, Error]{Is_ok: true, Value: parsed, Err: Error{Message: \"\"}}\n")
	b.WriteString("\tdefault:\n")
	b.WriteString("\t\treturn Result[float64, Error]{Is_ok: false, Value: 0, Err: Error{Message: \"not a float\"}}\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_http_get(url string) Result[string, Error] {\n")
	b.WriteString("\tresp, err := http.Get(url)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tdefer resp.Body.Close()\n")
	b.WriteString("\tbody, err := io.ReadAll(resp.Body)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: string(body), Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_http_post(url string, body string) Result[string, Error] {\n")
	b.WriteString("\tresp, err := http.Post(url, \"text/plain\", strings.NewReader(body))\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tdefer resp.Body.Close()\n")
	b.WriteString("\tdata, err := io.ReadAll(resp.Body)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: string(data), Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_http_get_opts(url string, connectTimeoutMs int64, timeoutMs int64, headers string, userAgent string, tlsInsecure bool, caBundle string) Result[string, Error] {\n")
	b.WriteString("\tclient, err := buildHTTPClient(connectTimeoutMs, timeoutMs, tlsInsecure, caBundle)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treq, err := http.NewRequest(\"GET\", url, nil)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tapplyHeaders(req, headers, userAgent, \"\")\n")
	b.WriteString("\tresp, err := client.Do(req)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tdefer resp.Body.Close()\n")
	b.WriteString("\tbody, err := io.ReadAll(resp.Body)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: string(body), Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_http_post_opts(url string, body string, connectTimeoutMs int64, timeoutMs int64, headers string, userAgent string, contentType string, tlsInsecure bool, caBundle string) Result[string, Error] {\n")
	b.WriteString("\tclient, err := buildHTTPClient(connectTimeoutMs, timeoutMs, tlsInsecure, caBundle)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treq, err := http.NewRequest(\"POST\", url, strings.NewReader(body))\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tapplyHeaders(req, headers, userAgent, contentType)\n")
	b.WriteString("\tresp, err := client.Do(req)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tdefer resp.Body.Close()\n")
	b.WriteString("\tdata, err := io.ReadAll(resp.Body)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: string(data), Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_http_request(method string, url string, body string, connectTimeoutMs int64, timeoutMs int64, headers string, userAgent string, contentType string, tlsInsecure bool, caBundle string) Result[string, Error] {\n")
	b.WriteString("\tif method == \"\" { method = \"GET\" }\n")
	b.WriteString("\tclient, err := buildHTTPClient(connectTimeoutMs, timeoutMs, tlsInsecure, caBundle)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tvar reader io.Reader\n")
	b.WriteString("\tif body != \"\" { reader = strings.NewReader(body) }\n")
	b.WriteString("\treq, err := http.NewRequest(method, url, reader)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tapplyHeaders(req, headers, userAgent, contentType)\n")
	b.WriteString("\tresp, err := client.Do(req)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tdefer resp.Body.Close()\n")
	b.WriteString("\tdata, err := io.ReadAll(resp.Body)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: string(data), Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	if hasHttpResponse {
		b.WriteString("func __std_http_get_opts_resp(url string, connectTimeoutMs int64, timeoutMs int64, headers string, userAgent string, tlsInsecure bool, caBundle string) Result[HttpResponse, Error] {\n")
		b.WriteString("\tclient, err := buildHTTPClient(connectTimeoutMs, timeoutMs, tlsInsecure, caBundle)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[HttpResponse, Error]{Is_ok: false, Value: HttpResponse{}, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treq, err := http.NewRequest(\"GET\", url, nil)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[HttpResponse, Error]{Is_ok: false, Value: HttpResponse{}, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tapplyHeaders(req, headers, userAgent, \"\")\n")
		b.WriteString("\tresp, err := client.Do(req)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[HttpResponse, Error]{Is_ok: false, Value: HttpResponse{}, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tdefer resp.Body.Close()\n")
		b.WriteString("\tbody, err := io.ReadAll(resp.Body)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[HttpResponse, Error]{Is_ok: false, Value: HttpResponse{}, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn Result[HttpResponse, Error]{Is_ok: true, Value: HttpResponse{Status: int64(resp.StatusCode), Headers: headerString(resp.Header), Body: string(body)}, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_http_post_opts_resp(url string, body string, connectTimeoutMs int64, timeoutMs int64, headers string, userAgent string, contentType string, tlsInsecure bool, caBundle string) Result[HttpResponse, Error] {\n")
		b.WriteString("\tclient, err := buildHTTPClient(connectTimeoutMs, timeoutMs, tlsInsecure, caBundle)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[HttpResponse, Error]{Is_ok: false, Value: HttpResponse{}, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treq, err := http.NewRequest(\"POST\", url, strings.NewReader(body))\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[HttpResponse, Error]{Is_ok: false, Value: HttpResponse{}, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tapplyHeaders(req, headers, userAgent, contentType)\n")
		b.WriteString("\tresp, err := client.Do(req)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[HttpResponse, Error]{Is_ok: false, Value: HttpResponse{}, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tdefer resp.Body.Close()\n")
		b.WriteString("\tdata, err := io.ReadAll(resp.Body)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[HttpResponse, Error]{Is_ok: false, Value: HttpResponse{}, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn Result[HttpResponse, Error]{Is_ok: true, Value: HttpResponse{Status: int64(resp.StatusCode), Headers: headerString(resp.Header), Body: string(data)}, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_http_request_resp(method string, url string, body string, connectTimeoutMs int64, timeoutMs int64, headers string, userAgent string, contentType string, tlsInsecure bool, caBundle string) Result[HttpResponse, Error] {\n")
		b.WriteString("\tif method == \"\" { method = \"GET\" }\n")
		b.WriteString("\tclient, err := buildHTTPClient(connectTimeoutMs, timeoutMs, tlsInsecure, caBundle)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[HttpResponse, Error]{Is_ok: false, Value: HttpResponse{}, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tvar reader io.Reader\n")
		b.WriteString("\tif body != \"\" { reader = strings.NewReader(body) }\n")
		b.WriteString("\treq, err := http.NewRequest(method, url, reader)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[HttpResponse, Error]{Is_ok: false, Value: HttpResponse{}, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tapplyHeaders(req, headers, userAgent, contentType)\n")
		b.WriteString("\tresp, err := client.Do(req)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[HttpResponse, Error]{Is_ok: false, Value: HttpResponse{}, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tdefer resp.Body.Close()\n")
		b.WriteString("\tdata, err := io.ReadAll(resp.Body)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[HttpResponse, Error]{Is_ok: false, Value: HttpResponse{}, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn Result[HttpResponse, Error]{Is_ok: true, Value: HttpResponse{Status: int64(resp.StatusCode), Headers: headerString(resp.Header), Body: string(data)}, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
	}
	b.WriteString("func __std_kv_get(kv string, key string) string {\n")
	b.WriteString("\tif key == \"\" || kv == \"\" { return \"\" }\n")
	b.WriteString("\tstart := 0\n")
	b.WriteString("\tfor start <= len(kv) {\n")
	b.WriteString("\t\tend := strings.IndexByte(kv[start:], '\\n')\n")
	b.WriteString("\t\tif end < 0 { end = len(kv) - start }\n")
	b.WriteString("\t\tline := kv[start : start+end]\n")
	b.WriteString("\t\tif line != \"\" {\n")
	b.WriteString("\t\t\tif idx := strings.IndexByte(line, '='); idx >= 0 {\n")
	b.WriteString("\t\t\t\tif line[:idx] == key { return line[idx+1:] }\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tstart += end + 1\n")
	b.WriteString("\t\tif start > len(kv) { break }\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn \"\"\n")
	b.WriteString("}\n")
	b.WriteString("func __std_header_get(headers string, key string) string {\n")
	b.WriteString("\tif key == \"\" || headers == \"\" { return \"\" }\n")
	b.WriteString("\tneedle := strings.TrimSpace(key)\n")
	b.WriteString("\tif needle == \"\" { return \"\" }\n")
	b.WriteString("\tstart := 0\n")
	b.WriteString("\tfor start <= len(headers) {\n")
	b.WriteString("\t\tend := strings.IndexByte(headers[start:], '\\n')\n")
	b.WriteString("\t\tif end < 0 { end = len(headers) - start }\n")
	b.WriteString("\t\tline := headers[start : start+end]\n")
	b.WriteString("\t\tif line != \"\" {\n")
	b.WriteString("\t\t\tif idx := strings.IndexByte(line, ':'); idx >= 0 {\n")
	b.WriteString("\t\t\t\tk := strings.TrimSpace(line[:idx])\n")
	b.WriteString("\t\t\t\tif strings.EqualFold(k, needle) { return strings.TrimSpace(line[idx+1:]) }\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tstart += end + 1\n")
	b.WriteString("\t\tif start > len(headers) { break }\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn \"\"\n")
	b.WriteString("}\n")
	b.WriteString("func __std_query_get(query string, key string) string {\n")
	b.WriteString("\tif key == \"\" { return \"\" }\n")
	b.WriteString("\tvals, err := url.ParseQuery(query)\n")
	b.WriteString("\tif err != nil { return \"\" }\n")
	b.WriteString("\treturn vals.Get(key)\n")
	b.WriteString("}\n")
	b.WriteString("func __std_http_serve_text(addr string, body string) Result[bool, Error] {\n")
	b.WriteString("\thandler := func(w http.ResponseWriter, r *http.Request) {\n")
	b.WriteString("\t\tw.Header().Set(\"Content-Type\", \"text/plain; charset=utf-8\")\n")
	b.WriteString("\t\t_, _ = w.Write([]byte(body))\n")
	b.WriteString("\t}\n")
	b.WriteString("\thttp.HandleFunc(\"/\", handler)\n")
	b.WriteString("\tif err := http.ListenAndServe(addr, nil); err != nil {\n")
	b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	if needsHttpServeApp {
		writeHttpServeApp(&b, handlers)
	}
	b.WriteString("func __std_sha256_hex(s string) string {\n")
	b.WriteString("\tsum := sha256.Sum256([]byte(s))\n")
	b.WriteString("\treturn hex.EncodeToString(sum[:])\n")
	b.WriteString("}\n")
	if needsHmac {
		b.WriteString("func __std_hmac_sha256_hex(message string, secret string) string {\n")
		b.WriteString("\th := hmac.New(sha256.New, []byte(secret))\n")
		b.WriteString("\t_, _ = h.Write([]byte(message))\n")
		b.WriteString("\treturn hex.EncodeToString(h.Sum(nil))\n")
		b.WriteString("}\n")
	}
	if needsJwt {
		b.WriteString("func __std_jwt_sign_hs256(headerJSON string, payloadJSON string, secret string) Result[string, Error] {\n")
		b.WriteString("\tenc := base64.RawURLEncoding\n")
		b.WriteString("\theader := enc.EncodeToString([]byte(headerJSON))\n")
		b.WriteString("\tpayload := enc.EncodeToString([]byte(payloadJSON))\n")
		b.WriteString("\tsigning := header + \".\" + payload\n")
		b.WriteString("\th := hmac.New(sha256.New, []byte(secret))\n")
		b.WriteString("\t_, _ = h.Write([]byte(signing))\n")
		b.WriteString("\tsig := enc.EncodeToString(h.Sum(nil))\n")
		b.WriteString("\ttoken := signing + \".\" + sig\n")
		b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: token, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_jwt_verify_hs256(token string, secret string) Result[bool, Error] {\n")
		b.WriteString("\tparts := strings.Split(token, \".\")\n")
		b.WriteString("\tif len(parts) != 3 {\n")
		b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: \"invalid token\"}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tsigning := parts[0] + \".\" + parts[1]\n")
		b.WriteString("\tsig := parts[2]\n")
		b.WriteString("\th := hmac.New(sha256.New, []byte(secret))\n")
		b.WriteString("\t_, _ = h.Write([]byte(signing))\n")
		b.WriteString("\texpected := base64.RawURLEncoding.EncodeToString(h.Sum(nil))\n")
		b.WriteString("\tif !hmac.Equal([]byte(sig), []byte(expected)) {\n")
		b.WriteString("\t\treturn Result[bool, Error]{Is_ok: true, Value: false, Err: Error{Message: \"\"}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
	}
	b.WriteString("func __std_random_hex(n int64) Result[string, Error] {\n")
	b.WriteString("\tif n <= 0 {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: true, Value: \"\", Err: Error{Message: \"\"}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tbuf := make([]byte, int(n))\n")
	b.WriteString("\tif _, err := rand.Read(buf); err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: hex.EncodeToString(buf), Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n\n")
	if needsBcrypt {
		b.WriteString("func __std_bcrypt_hash(password string, cost int64) Result[string, Error] {\n")
		b.WriteString("\tif cost == 0 { cost = int64(bcrypt.DefaultCost) }\n")
		b.WriteString("\tif cost < int64(bcrypt.MinCost) { cost = int64(bcrypt.MinCost) }\n")
		b.WriteString("\tif cost > int64(bcrypt.MaxCost) { cost = int64(bcrypt.MaxCost) }\n")
		b.WriteString("\thash, err := bcrypt.GenerateFromPassword([]byte(password), int(cost))\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: string(hash), Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_bcrypt_verify(password string, hash string) Result[bool, Error] {\n")
		b.WriteString("\terr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))\n")
		b.WriteString("\tif err == nil {\n")
		b.WriteString("\t\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tif err == bcrypt.ErrMismatchedHashAndPassword {\n")
		b.WriteString("\t\treturn Result[bool, Error]{Is_ok: true, Value: false, Err: Error{Message: \"\"}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}}\n")
		b.WriteString("}\n\n")
	}
	if needsSession {
		b.WriteString("func __std_session_init(path string) Result[bool, Error] {\n")
		b.WriteString("\tif path == \"\" {\n")
		b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: \"empty path\"}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tif path == \"memory\" || path == \"memory:\" {\n")
		b.WriteString("\t\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tdb, err := sql.Open(\"sqlite\", path)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\t_, err = db.Exec(\"create table if not exists sessions (token_hash text primary key, user_id text not null, expires_at text)\")\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_session_put(path string, tokenHash string, userID string, expiresAt string) Result[bool, Error] {\n")
		b.WriteString("\tif path == \"\" { return Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: \"empty path\"}} }\n")
		b.WriteString("\tif tokenHash == \"\" { return Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: \"empty token\"}} }\n")
		b.WriteString("\tif path == \"memory\" || path == \"memory:\" {\n")
		b.WriteString("\t\tvar exp time.Time\n")
		b.WriteString("\t\tif strings.TrimSpace(expiresAt) != \"\" {\n")
		b.WriteString("\t\t\tt, err := time.Parse(time.RFC3339, strings.TrimSpace(expiresAt))\n")
		b.WriteString("\t\t\tif err != nil { return Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\t\t\texp = t\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\t__bazic_session_mu.Lock()\n")
		b.WriteString("\t\t__bazic_session_store[tokenHash] = __bazic_session_entry{UserID: userID, ExpiresAt: exp}\n")
		b.WriteString("\t\t__bazic_session_mu.Unlock()\n")
		b.WriteString("\t\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tdb, err := sql.Open(\"sqlite\", path)\n")
		b.WriteString("\tif err != nil { return Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\t_, err = db.Exec(\"insert into sessions (token_hash, user_id, expires_at) values (?, ?, ?) on conflict(token_hash) do update set user_id=excluded.user_id, expires_at=excluded.expires_at\", tokenHash, userID, expiresAt)\n")
		b.WriteString("\tif err != nil { return Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_session_get_user(path string, tokenHash string) Result[string, Error] {\n")
		b.WriteString("\tif path == \"\" { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"empty path\"}} }\n")
		b.WriteString("\tif tokenHash == \"\" { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"empty token\"}} }\n")
		b.WriteString("\tif path == \"memory\" || path == \"memory:\" {\n")
		b.WriteString("\t\t__bazic_session_mu.Lock()\n")
		b.WriteString("\t\tentry, ok := __bazic_session_store[tokenHash]\n")
		b.WriteString("\t\tif !ok {\n")
		b.WriteString("\t\t\t__bazic_session_mu.Unlock()\n")
		b.WriteString("\t\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"not found\"}}\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tif !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {\n")
		b.WriteString("\t\t\tdelete(__bazic_session_store, tokenHash)\n")
		b.WriteString("\t\t\t__bazic_session_mu.Unlock()\n")
		b.WriteString("\t\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"expired\"}}\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tuserID := entry.UserID\n")
		b.WriteString("\t\t__bazic_session_mu.Unlock()\n")
		b.WriteString("\t\treturn Result[string, Error]{Is_ok: true, Value: userID, Err: Error{Message: \"\"}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tdb, err := sql.Open(\"sqlite\", path)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\tvar userID string\n")
		b.WriteString("\tvar expiresAt string\n")
		b.WriteString("\trow := db.QueryRow(\"select user_id, expires_at from sessions where token_hash = ?\", tokenHash)\n")
		b.WriteString("\tif err := row.Scan(&userID, &expiresAt); err != nil {\n")
		b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"not found\"}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tif strings.TrimSpace(expiresAt) != \"\" {\n")
		b.WriteString("\t\texp, err := time.Parse(time.RFC3339, strings.TrimSpace(expiresAt))\n")
		b.WriteString("\t\tif err == nil && time.Now().After(exp) {\n")
		b.WriteString("\t\t\t_, _ = db.Exec(\"delete from sessions where token_hash = ?\", tokenHash)\n")
		b.WriteString("\t\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"expired\"}}\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: userID, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_session_delete(path string, tokenHash string) Result[bool, Error] {\n")
		b.WriteString("\tif path == \"\" { return Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: \"empty path\"}} }\n")
		b.WriteString("\tif tokenHash == \"\" { return Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: \"empty token\"}} }\n")
		b.WriteString("\tif path == \"memory\" || path == \"memory:\" {\n")
		b.WriteString("\t\t__bazic_session_mu.Lock()\n")
		b.WriteString("\t\tdelete(__bazic_session_store, tokenHash)\n")
		b.WriteString("\t\t__bazic_session_mu.Unlock()\n")
		b.WriteString("\t\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tdb, err := sql.Open(\"sqlite\", path)\n")
		b.WriteString("\tif err != nil { return Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\t_, err = db.Exec(\"delete from sessions where token_hash = ?\", tokenHash)\n")
		b.WriteString("\tif err != nil { return Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n\n")
	}
	b.WriteString("func __std_args() string {\n")
	b.WriteString("\tif len(os.Args) <= 1 { return \"\" }\n")
	b.WriteString("\treturn strings.Join(os.Args[1:], \"\\n\")\n")
	b.WriteString("}\n")
	b.WriteString("func __std_getenv(key string) Result[string, Error] {\n")
	b.WriteString("\tif key == \"\" {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"empty key\"}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tval, ok := os.LookupEnv(key)\n")
	b.WriteString("\tif !ok {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"not set\"}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: val, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n\n")
	b.WriteString("func __std_cwd() Result[string, Error] {\n")
	b.WriteString("\twd, err := os.Getwd()\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: wd, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_chdir(path string) Result[bool, Error] {\n")
	b.WriteString("\tif err := os.Chdir(path); err != nil {\n")
	b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_env_list() Result[string, Error] {\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: strings.Join(os.Environ(), \"\\n\"), Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_temp_dir() Result[string, Error] {\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: os.TempDir(), Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n\n")
	b.WriteString("func __std_exe_path() Result[string, Error] {\n")
	b.WriteString("\tpath, err := os.Executable()\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: path, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func __std_home_dir() Result[string, Error] {\n")
	b.WriteString("\tpath, err := os.UserHomeDir()\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: path, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n\n")
	if target == "wasm" {
		b.WriteString("func __std_web_get_json(key string) Result[string, Error] {\n")
		b.WriteString("\tbridge := js.Global().Get(\"BAZIC_WEB\")\n")
		b.WriteString("\tif bridge.IsUndefined() || bridge.IsNull() {\n")
		b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"BAZIC_WEB not found\"}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tres := bridge.Call(\"get\", key)\n")
		b.WriteString("\tif res.IsUndefined() || res.IsNull() {\n")
		b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"missing key\"}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: res.String(), Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_web_set_json(key string, jsonText string) Result[bool, Error] {\n")
		b.WriteString("\tbridge := js.Global().Get(\"BAZIC_WEB\")\n")
		b.WriteString("\tif bridge.IsUndefined() || bridge.IsNull() {\n")
		b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: \"BAZIC_WEB not found\"}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tbridge.Call(\"set\", key, jsonText)\n")
		b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n\n")
	} else {
		b.WriteString("func __std_web_get_json(key string) Result[string, Error] {\n")
		b.WriteString("\t_ = key\n")
		b.WriteString("\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"web interop only supported in wasm\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_web_set_json(key string, jsonText string) Result[bool, Error] {\n")
		b.WriteString("\t_, _ = key, jsonText\n")
		b.WriteString("\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: \"web interop only supported in wasm\"}}\n")
		b.WriteString("}\n\n")
	}
	b.WriteString("func __std_base64_encode(s string) string {\n")
	b.WriteString("\treturn base64.StdEncoding.EncodeToString([]byte(s))\n")
	b.WriteString("}\n")
	b.WriteString("func __std_base64_decode(s string) Result[string, Error] {\n")
	b.WriteString("\tout, err := base64.StdEncoding.DecodeString(s)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: string(out), Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n\n")
	b.WriteString("func __std_path_basename(path string) string { return filepath.Base(path) }\n")
	b.WriteString("func __std_path_dirname(path string) string { return filepath.Dir(path) }\n")
	b.WriteString("func __std_path_join(a string, b string) string { return filepath.Join(a, b) }\n\n")
	if needsDB {
		b.WriteString("func __std_db_exec(path string, sqlText string) Result[bool, Error] {\n")
		b.WriteString("\tdb, err := sql.Open(\"sqlite\", path)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\tif _, err := db.Exec(sqlText); err != nil {\n")
		b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_query(path string, sqlText string) Result[string, Error] {\n")
		b.WriteString("\tdb, err := sql.Open(\"sqlite\", path)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\trows, err := db.Query(sqlText)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tdefer rows.Close()\n")
		b.WriteString("\tcols, err := rows.Columns()\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tvar out strings.Builder\n")
		b.WriteString("\tout.WriteString(strings.Join(cols, \"\\t\"))\n")
		b.WriteString("\tout.WriteString(\"\\n\")\n")
		b.WriteString("\tvals := make([]any, len(cols))\n")
		b.WriteString("\tptrs := make([]any, len(cols))\n")
		b.WriteString("\tfor i := range vals { ptrs[i] = &vals[i] }\n")
		b.WriteString("\tfor rows.Next() {\n")
		b.WriteString("\t\tif err := rows.Scan(ptrs...); err != nil {\n")
		b.WriteString("\t\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tfor i, v := range vals {\n")
		b.WriteString("\t\t\tif i > 0 { out.WriteString(\"\\t\") }\n")
		b.WriteString("\t\t\tswitch x := v.(type) {\n")
		b.WriteString("\t\t\tcase nil:\n")
		b.WriteString("\t\t\t\tout.WriteString(\"null\")\n")
		b.WriteString("\t\t\tcase []byte:\n")
		b.WriteString("\t\t\t\tout.WriteString(string(x))\n")
		b.WriteString("\t\t\tdefault:\n")
		b.WriteString("\t\t\t\tout.WriteString(fmt.Sprint(x))\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tout.WriteString(\"\\n\")\n")
		b.WriteString("\t}\n")
		b.WriteString("\tif err := rows.Err(); err != nil {\n")
		b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: out.String(), Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func normalizeDBDriver(driver string) string {\n")
		b.WriteString("\tswitch driver {\n")
		b.WriteString("\tcase \"sqlite3\":\n")
		b.WriteString("\t\treturn \"sqlite\"\n")
		b.WriteString("\tcase \"postgresql\":\n")
		b.WriteString("\t\treturn \"postgres\"\n")
		b.WriteString("\tdefault:\n")
		b.WriteString("\t\treturn driver\n")
		b.WriteString("\t}\n")
		b.WriteString("}\n")
		b.WriteString("func parseSQLParams(params string) ([]any, error) {\n")
		b.WriteString("\tif strings.TrimSpace(params) == \"\" { return nil, nil }\n")
		b.WriteString("\tlines := strings.Split(params, \"\\n\")\n")
		b.WriteString("\targs := make([]any, 0, len(lines))\n")
		b.WriteString("\tfor _, line := range lines {\n")
		b.WriteString("\t\tif line == \"\" { continue }\n")
		b.WriteString("\t\tparts := strings.SplitN(line, \":\", 2)\n")
		b.WriteString("\t\tif len(parts) != 2 { return nil, fmt.Errorf(\"invalid param\") }\n")
		b.WriteString("\t\tswitch parts[0] {\n")
		b.WriteString("\t\tcase \"str\":\n")
		b.WriteString("\t\t\targs = append(args, parts[1])\n")
		b.WriteString("\t\tcase \"int\":\n")
		b.WriteString("\t\t\tv, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)\n")
		b.WriteString("\t\t\tif err != nil { return nil, err }\n")
		b.WriteString("\t\t\targs = append(args, v)\n")
		b.WriteString("\t\tcase \"float\":\n")
		b.WriteString("\t\t\tv, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)\n")
		b.WriteString("\t\t\tif err != nil { return nil, err }\n")
		b.WriteString("\t\t\targs = append(args, v)\n")
		b.WriteString("\t\tcase \"bool\":\n")
		b.WriteString("\t\t\tval := strings.ToLower(strings.TrimSpace(parts[1]))\n")
		b.WriteString("\t\t\targs = append(args, val == \"true\" || val == \"1\")\n")
		b.WriteString("\t\tcase \"null\":\n")
		b.WriteString("\t\t\targs = append(args, nil)\n")
		b.WriteString("\t\tdefault:\n")
		b.WriteString("\t\t\treturn nil, fmt.Errorf(\"invalid param type\")\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn args, nil\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_exec_with(driver string, dsn string, sqlText string) Result[bool, Error] {\n")
		b.WriteString("\tif driver == \"\" { return Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: \"db_exec_with: empty driver\"}} }\n")
		b.WriteString("\tdriver = normalizeDBDriver(driver)\n")
		b.WriteString("\tdb, err := sql.Open(driver, dsn)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\tif _, err := db.Exec(sqlText); err != nil {\n")
		b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_exec_params(path string, sqlText string, params string) Result[bool, Error] {\n")
		b.WriteString("\tdb, err := sql.Open(\"sqlite\", path)\n")
		b.WriteString("\tif err != nil { return Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\targs, err := parseSQLParams(params)\n")
		b.WriteString("\tif err != nil { return Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tif _, err := db.Exec(sqlText, args...); err != nil {\n")
		b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_exec_params_with(driver string, dsn string, sqlText string, params string) Result[bool, Error] {\n")
		b.WriteString("\tif driver == \"\" { return Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: \"db_exec_params_with: empty driver\"}} }\n")
		b.WriteString("\tdriver = normalizeDBDriver(driver)\n")
		b.WriteString("\tdb, err := sql.Open(driver, dsn)\n")
		b.WriteString("\tif err != nil { return Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\targs, err := parseSQLParams(params)\n")
		b.WriteString("\tif err != nil { return Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tif _, err := db.Exec(sqlText, args...); err != nil {\n")
		b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_query_with(driver string, dsn string, sqlText string) Result[string, Error] {\n")
		b.WriteString("\tif driver == \"\" { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"db_query_with: empty driver\"}} }\n")
		b.WriteString("\tdriver = normalizeDBDriver(driver)\n")
		b.WriteString("\tdb, err := sql.Open(driver, dsn)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\trows, err := db.Query(sqlText)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tdefer rows.Close()\n")
		b.WriteString("\tcols, err := rows.Columns()\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tvar out strings.Builder\n")
		b.WriteString("\tout.WriteString(strings.Join(cols, \"\\t\"))\n")
		b.WriteString("\tout.WriteString(\"\\n\")\n")
		b.WriteString("\tvals := make([]any, len(cols))\n")
		b.WriteString("\tptrs := make([]any, len(cols))\n")
		b.WriteString("\tfor i := range vals { ptrs[i] = &vals[i] }\n")
		b.WriteString("\tfor rows.Next() {\n")
		b.WriteString("\t\tif err := rows.Scan(ptrs...); err != nil {\n")
		b.WriteString("\t\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tfor i, v := range vals {\n")
		b.WriteString("\t\t\tif i > 0 { out.WriteString(\"\\t\") }\n")
		b.WriteString("\t\t\tswitch x := v.(type) {\n")
		b.WriteString("\t\t\tcase nil:\n")
		b.WriteString("\t\t\t\tout.WriteString(\"null\")\n")
		b.WriteString("\t\t\tcase []byte:\n")
		b.WriteString("\t\t\t\tout.WriteString(string(x))\n")
		b.WriteString("\t\t\tdefault:\n")
		b.WriteString("\t\t\t\tout.WriteString(fmt.Sprint(x))\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tout.WriteString(\"\\n\")\n")
		b.WriteString("\t}\n")
		b.WriteString("\tif err := rows.Err(); err != nil {\n")
		b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: out.String(), Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_query_params(path string, sqlText string, params string) Result[string, Error] {\n")
		b.WriteString("\tdb, err := sql.Open(\"sqlite\", path)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\targs, err := parseSQLParams(params)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\trows, err := db.Query(sqlText, args...)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer rows.Close()\n")
		b.WriteString("\tcols, err := rows.Columns()\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tvar out strings.Builder\n")
		b.WriteString("\tout.WriteString(strings.Join(cols, \"\\t\"))\n")
		b.WriteString("\tout.WriteString(\"\\n\")\n")
		b.WriteString("\tvals := make([]any, len(cols))\n")
		b.WriteString("\tptrs := make([]any, len(cols))\n")
		b.WriteString("\tfor i := range vals { ptrs[i] = &vals[i] }\n")
		b.WriteString("\tfor rows.Next() {\n")
		b.WriteString("\t\tif err := rows.Scan(ptrs...); err != nil {\n")
		b.WriteString("\t\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tfor i, v := range vals {\n")
		b.WriteString("\t\t\tif i > 0 { out.WriteString(\"\\t\") }\n")
		b.WriteString("\t\t\tswitch x := v.(type) {\n")
		b.WriteString("\t\t\tcase nil:\n")
		b.WriteString("\t\t\t\tout.WriteString(\"null\")\n")
		b.WriteString("\t\t\tcase []byte:\n")
		b.WriteString("\t\t\t\tout.WriteString(string(x))\n")
		b.WriteString("\t\t\tdefault:\n")
		b.WriteString("\t\t\t\tout.WriteString(fmt.Sprint(x))\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tout.WriteString(\"\\n\")\n")
		b.WriteString("\t}\n")
		b.WriteString("\tif err := rows.Err(); err != nil {\n")
		b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: out.String(), Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_query_params_with(driver string, dsn string, sqlText string, params string) Result[string, Error] {\n")
		b.WriteString("\tif driver == \"\" { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"db_query_params_with: empty driver\"}} }\n")
		b.WriteString("\tdriver = normalizeDBDriver(driver)\n")
		b.WriteString("\tdb, err := sql.Open(driver, dsn)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\targs, err := parseSQLParams(params)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\trows, err := db.Query(sqlText, args...)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer rows.Close()\n")
		b.WriteString("\tcols, err := rows.Columns()\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tvar out strings.Builder\n")
		b.WriteString("\tout.WriteString(strings.Join(cols, \"\\t\"))\n")
		b.WriteString("\tout.WriteString(\"\\n\")\n")
		b.WriteString("\tvals := make([]any, len(cols))\n")
		b.WriteString("\tptrs := make([]any, len(cols))\n")
		b.WriteString("\tfor i := range vals { ptrs[i] = &vals[i] }\n")
		b.WriteString("\tfor rows.Next() {\n")
		b.WriteString("\t\tif err := rows.Scan(ptrs...); err != nil {\n")
		b.WriteString("\t\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tfor i, v := range vals {\n")
		b.WriteString("\t\t\tif i > 0 { out.WriteString(\"\\t\") }\n")
		b.WriteString("\t\t\tswitch x := v.(type) {\n")
		b.WriteString("\t\t\tcase nil:\n")
		b.WriteString("\t\t\t\tout.WriteString(\"null\")\n")
		b.WriteString("\t\t\tcase []byte:\n")
		b.WriteString("\t\t\t\tout.WriteString(string(x))\n")
		b.WriteString("\t\t\tdefault:\n")
		b.WriteString("\t\t\t\tout.WriteString(fmt.Sprint(x))\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tout.WriteString(\"\\n\")\n")
		b.WriteString("\t}\n")
		b.WriteString("\tif err := rows.Err(); err != nil {\n")
		b.WriteString("\t\treturn Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: out.String(), Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_query_json_params(path string, sqlText string, params string) Result[string, Error] {\n")
		b.WriteString("\tdb, err := sql.Open(\"sqlite\", path)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\targs, err := parseSQLParams(params)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\trows, err := db.Query(sqlText, args...)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer rows.Close()\n")
		b.WriteString("\tout, err := rowsToJSON(rows)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: out, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_query_json_params_with(driver string, dsn string, sqlText string, params string) Result[string, Error] {\n")
		b.WriteString("\tif driver == \"\" { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"db_query_json_params_with: empty driver\"}} }\n")
		b.WriteString("\tdriver = normalizeDBDriver(driver)\n")
		b.WriteString("\tdb, err := sql.Open(driver, dsn)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\targs, err := parseSQLParams(params)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\trows, err := db.Query(sqlText, args...)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer rows.Close()\n")
		b.WriteString("\tout, err := rowsToJSON(rows)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: out, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_query_one_json_params(path string, sqlText string, params string) Result[string, Error] {\n")
		b.WriteString("\tdb, err := sql.Open(\"sqlite\", path)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\targs, err := parseSQLParams(params)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\trows, err := db.Query(sqlText, args...)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer rows.Close()\n")
		b.WriteString("\tcols, err := rows.Columns()\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tvals := make([]any, len(cols))\n")
		b.WriteString("\tptrs := make([]any, len(cols))\n")
		b.WriteString("\tfor i := range vals { ptrs[i] = &vals[i] }\n")
		b.WriteString("\tif !rows.Next() { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"not found\"}} }\n")
		b.WriteString("\tif err := rows.Scan(ptrs...); err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdata, err := json.Marshal(rowToMap(cols, vals))\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: string(data), Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_query_one_json_params_with(driver string, dsn string, sqlText string, params string) Result[string, Error] {\n")
		b.WriteString("\tif driver == \"\" { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"db_query_one_json_params_with: empty driver\"}} }\n")
		b.WriteString("\tdriver = normalizeDBDriver(driver)\n")
		b.WriteString("\tdb, err := sql.Open(driver, dsn)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\targs, err := parseSQLParams(params)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\trows, err := db.Query(sqlText, args...)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer rows.Close()\n")
		b.WriteString("\tcols, err := rows.Columns()\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tvals := make([]any, len(cols))\n")
		b.WriteString("\tptrs := make([]any, len(cols))\n")
		b.WriteString("\tfor i := range vals { ptrs[i] = &vals[i] }\n")
		b.WriteString("\tif !rows.Next() { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"not found\"}} }\n")
		b.WriteString("\tif err := rows.Scan(ptrs...); err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdata, err := json.Marshal(rowToMap(cols, vals))\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: string(data), Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_exec_returning_id_params(path string, sqlText string, params string) Result[int64, Error] {\n")
		b.WriteString("\tdb, err := sql.Open(\"sqlite\", path)\n")
		b.WriteString("\tif err != nil { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\targs, err := parseSQLParams(params)\n")
		b.WriteString("\tif err != nil { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tres, err := db.Exec(sqlText, args...)\n")
		b.WriteString("\tif err != nil { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tlast, err := res.LastInsertId()\n")
		b.WriteString("\tif err != nil { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\treturn Result[int64, Error]{Is_ok: true, Value: last, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_exec_returning_id_params_with(driver string, dsn string, sqlText string, params string) Result[int64, Error] {\n")
		b.WriteString("\tif driver == \"\" { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: \"db_exec_returning_id_params_with: empty driver\"}} }\n")
		b.WriteString("\tdriver = normalizeDBDriver(driver)\n")
		b.WriteString("\tdb, err := sql.Open(driver, dsn)\n")
		b.WriteString("\tif err != nil { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\targs, err := parseSQLParams(params)\n")
		b.WriteString("\tif err != nil { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tres, err := db.Exec(sqlText, args...)\n")
		b.WriteString("\tif err != nil { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tlast, err := res.LastInsertId()\n")
		b.WriteString("\tif err != nil { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\treturn Result[int64, Error]{Is_ok: true, Value: last, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func rowToMap(cols []string, vals []any) map[string]any {\n")
		b.WriteString("\tm := make(map[string]any, len(cols))\n")
		b.WriteString("\tfor i, col := range cols {\n")
		b.WriteString("\t\tv := vals[i]\n")
		b.WriteString("\t\tswitch x := v.(type) {\n")
		b.WriteString("\t\tcase nil:\n")
		b.WriteString("\t\t\tm[col] = nil\n")
		b.WriteString("\t\tcase []byte:\n")
		b.WriteString("\t\t\tm[col] = string(x)\n")
		b.WriteString("\t\tcase time.Time:\n")
		b.WriteString("\t\t\tm[col] = x.UTC().Format(time.RFC3339)\n")
		b.WriteString("\t\tdefault:\n")
		b.WriteString("\t\t\tm[col] = x\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn m\n")
		b.WriteString("}\n")
	b.WriteString("func rowsToJSON(rows *sql.Rows) (string, error) {\n")
	b.WriteString("\tcols, err := rows.Columns()\n")
	b.WriteString("\tif err != nil { return \"\", err }\n")
	b.WriteString("\tvals := make([]any, len(cols))\n")
	b.WriteString("\tptrs := make([]any, len(cols))\n")
	b.WriteString("\tfor i := range vals { ptrs[i] = &vals[i] }\n")
	b.WriteString("\tvar buf bytes.Buffer\n")
	b.WriteString("\tbuf.WriteByte('[')\n")
	b.WriteString("\tfirst := true\n")
	b.WriteString("\tfor rows.Next() {\n")
	b.WriteString("\t\tif err := rows.Scan(ptrs...); err != nil { return \"\", err }\n")
	b.WriteString("\t\tdata, err := json.Marshal(rowToMap(cols, vals))\n")
	b.WriteString("\t\tif err != nil { return \"\", err }\n")
	b.WriteString("\t\tif !first { buf.WriteByte(',') }\n")
	b.WriteString("\t\tfirst = false\n")
	b.WriteString("\t\t_, _ = buf.Write(data)\n")
	b.WriteString("\t}\n")
	b.WriteString("\tif err := rows.Err(); err != nil { return \"\", err }\n")
	b.WriteString("\tbuf.WriteByte(']')\n")
	b.WriteString("\treturn buf.String(), nil\n")
	b.WriteString("}\n")
		b.WriteString("func __std_db_query_json(path string, sqlText string) Result[string, Error] {\n")
		b.WriteString("\tdb, err := sql.Open(\"sqlite\", path)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\trows, err := db.Query(sqlText)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer rows.Close()\n")
		b.WriteString("\tout, err := rowsToJSON(rows)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: out, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_query_json_with(driver string, dsn string, sqlText string) Result[string, Error] {\n")
		b.WriteString("\tif driver == \"\" { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"db_query_json_with: empty driver\"}} }\n")
		b.WriteString("\tdriver = normalizeDBDriver(driver)\n")
		b.WriteString("\tdb, err := sql.Open(driver, dsn)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\trows, err := db.Query(sqlText)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer rows.Close()\n")
		b.WriteString("\tout, err := rowsToJSON(rows)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: out, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_query_one_json(path string, sqlText string) Result[string, Error] {\n")
		b.WriteString("\tdb, err := sql.Open(\"sqlite\", path)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\trows, err := db.Query(sqlText)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer rows.Close()\n")
		b.WriteString("\tcols, err := rows.Columns()\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tvals := make([]any, len(cols))\n")
		b.WriteString("\tptrs := make([]any, len(cols))\n")
		b.WriteString("\tfor i := range vals { ptrs[i] = &vals[i] }\n")
		b.WriteString("\tif !rows.Next() { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"not found\"}} }\n")
		b.WriteString("\tif err := rows.Scan(ptrs...); err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdata, err := json.Marshal(rowToMap(cols, vals))\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: string(data), Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_query_one_json_with(driver string, dsn string, sqlText string) Result[string, Error] {\n")
		b.WriteString("\tif driver == \"\" { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"db_query_one_json_with: empty driver\"}} }\n")
		b.WriteString("\tdriver = normalizeDBDriver(driver)\n")
		b.WriteString("\tdb, err := sql.Open(driver, dsn)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\trows, err := db.Query(sqlText)\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer rows.Close()\n")
		b.WriteString("\tcols, err := rows.Columns()\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tvals := make([]any, len(cols))\n")
		b.WriteString("\tptrs := make([]any, len(cols))\n")
		b.WriteString("\tfor i := range vals { ptrs[i] = &vals[i] }\n")
		b.WriteString("\tif !rows.Next() { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: \"not found\"}} }\n")
		b.WriteString("\tif err := rows.Scan(ptrs...); err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdata, err := json.Marshal(rowToMap(cols, vals))\n")
		b.WriteString("\tif err != nil { return Result[string, Error]{Is_ok: false, Value: \"\", Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\treturn Result[string, Error]{Is_ok: true, Value: string(data), Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_exec_returning_id(path string, sqlText string) Result[int64, Error] {\n")
		b.WriteString("\tdb, err := sql.Open(\"sqlite\", path)\n")
		b.WriteString("\tif err != nil { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\tres, err := db.Exec(sqlText)\n")
		b.WriteString("\tif err != nil { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tlast, err := res.LastInsertId()\n")
		b.WriteString("\tif err != nil { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\treturn Result[int64, Error]{Is_ok: true, Value: last, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
		b.WriteString("func __std_db_exec_returning_id_with(driver string, dsn string, sqlText string) Result[int64, Error] {\n")
		b.WriteString("\tif driver == \"\" { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: \"db_exec_returning_id_with: empty driver\"}} }\n")
		b.WriteString("\tdriver = normalizeDBDriver(driver)\n")
		b.WriteString("\tdb, err := sql.Open(driver, dsn)\n")
		b.WriteString("\tif err != nil { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tdefer db.Close()\n")
		b.WriteString("\tres, err := db.Exec(sqlText)\n")
		b.WriteString("\tif err != nil { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\tlast, err := res.LastInsertId()\n")
		b.WriteString("\tif err != nil { return Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}} }\n")
		b.WriteString("\treturn Result[int64, Error]{Is_ok: true, Value: last, Err: Error{Message: \"\"}}\n")
		b.WriteString("}\n")
	}
	b.WriteString("func __std_open_url(url string) Result[bool, Error] {\n")
	b.WriteString("\tvar cmd *exec.Cmd\n")
	b.WriteString("\tif runtime.GOOS == \"windows\" {\n")
	b.WriteString("\t\tcmd = exec.Command(\"cmd\", \"/c\", \"start\", url)\n")
	b.WriteString("\t} else if runtime.GOOS == \"darwin\" {\n")
	b.WriteString("\t\tcmd = exec.Command(\"open\", url)\n")
	b.WriteString("\t} else {\n")
	b.WriteString("\t\tcmd = exec.Command(\"xdg-open\", url)\n")
	b.WriteString("\t}\n")
	b.WriteString("\tif err := cmd.Start(); err != nil {\n")
	b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n\n")
	b.WriteString("func applyHeaders(req *http.Request, headers string, userAgent string, contentType string) {\n")
	b.WriteString("\tif userAgent != \"\" {\n")
	b.WriteString("\t\treq.Header.Set(\"User-Agent\", userAgent)\n")
	b.WriteString("\t} else if req.Header.Get(\"User-Agent\") == \"\" {\n")
	b.WriteString("\t\treq.Header.Set(\"User-Agent\", \"Bazic/1.0\")\n")
	b.WriteString("\t}\n")
	b.WriteString("\tif contentType != \"\" {\n")
	b.WriteString("\t\treq.Header.Set(\"Content-Type\", contentType)\n")
	b.WriteString("\t}\n")
	b.WriteString("\tlines := strings.Split(headers, \"\\n\")\n")
	b.WriteString("\tfor _, line := range lines {\n")
	b.WriteString("\t\tline = strings.TrimSpace(line)\n")
	b.WriteString("\t\tif line == \"\" { continue }\n")
	b.WriteString("\t\tparts := strings.SplitN(line, \":\", 2)\n")
	b.WriteString("\t\tif len(parts) != 2 { continue }\n")
	b.WriteString("\t\tkey := strings.TrimSpace(parts[0])\n")
	b.WriteString("\t\tval := strings.TrimSpace(parts[1])\n")
	b.WriteString("\t\tif key == \"\" { continue }\n")
	b.WriteString("\t\treq.Header.Set(key, val)\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")
	b.WriteString("func buildHTTPClient(connectTimeoutMs int64, timeoutMs int64, tlsInsecure bool, caBundle string) (*http.Client, error) {\n")
	b.WriteString("\tclient := &http.Client{}\n")
	b.WriteString("\tif timeoutMs > 0 {\n")
	b.WriteString("\t\tclient.Timeout = time.Duration(timeoutMs) * time.Millisecond\n")
	b.WriteString("\t}\n")
	b.WriteString("\tvar tlsConfig *tls.Config\n")
	b.WriteString("\tif tlsInsecure {\n")
	b.WriteString("\t\ttlsConfig = &tls.Config{InsecureSkipVerify: true}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tif caBundle != \"\" {\n")
	b.WriteString("\t\tpool := x509.NewCertPool()\n")
	b.WriteString("\t\tif !pool.AppendCertsFromPEM([]byte(caBundle)) {\n")
	b.WriteString("\t\t\treturn nil, fmt.Errorf(\"invalid CA bundle\")\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tif tlsConfig == nil { tlsConfig = &tls.Config{} }\n")
	b.WriteString("\t\ttlsConfig.RootCAs = pool\n")
	b.WriteString("\t}\n")
	b.WriteString("\tif connectTimeoutMs > 0 || tlsConfig != nil {\n")
	b.WriteString("\t\tdialer := &net.Dialer{}\n")
	b.WriteString("\t\tif connectTimeoutMs > 0 { dialer.Timeout = time.Duration(connectTimeoutMs) * time.Millisecond }\n")
	b.WriteString("\t\ttransport := &http.Transport{DialContext: dialer.DialContext}\n")
	b.WriteString("\t\tif tlsConfig != nil { transport.TLSClientConfig = tlsConfig }\n")
	b.WriteString("\t\tclient.Transport = transport\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn client, nil\n")
	b.WriteString("}\n\n")
	if needsHeaderString {
		b.WriteString("func headerString(h http.Header) string {\n")
		b.WriteString("\tvar b strings.Builder\n")
		b.WriteString("\tfor k, vals := range h {\n")
		b.WriteString("\t\tfor _, v := range vals {\n")
		b.WriteString("\t\t\tb.WriteString(k)\n")
		b.WriteString("\t\t\tb.WriteString(\": \")\n")
		b.WriteString("\t\t\tb.WriteString(v)\n")
		b.WriteString("\t\t\tb.WriteString(\"\\n\")\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn b.String()\n")
		b.WriteString("}\n\n")
	}
	if len(handlers) > 0 {
		b.WriteString("func cookieString(cookies []*http.Cookie) string {\n")
		b.WriteString("\tif len(cookies) == 0 { return \"\" }\n")
		b.WriteString("\tparts := make([]string, 0, len(cookies))\n")
		b.WriteString("\tfor _, c := range cookies {\n")
		b.WriteString("\t\tparts = append(parts, c.Name+\"=\"+c.Value)\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn strings.Join(parts, \"\\n\")\n")
		b.WriteString("}\n\n")
	}

	for _, d := range p.Decls {
		switch decl := d.(type) {
		case *ast.StructDecl:
			s, err := genStruct(decl)
			if err != nil {
				return "", err
			}
			b.WriteString(s + "\n")
		case *ast.InterfaceDecl:
			s, err := genInterface(decl)
			if err != nil {
				return "", err
			}
			b.WriteString(s + "\n")
		case *ast.EnumDecl:
			s, err := genEnum(decl)
			if err != nil {
				return "", err
			}
			b.WriteString(s + "\n")
		}
	}
	for _, d := range p.Decls {
		if g, ok := d.(*ast.GlobalLetDecl); ok {
			line, err := genGlobal(g)
			if err != nil {
				return "", err
			}
			b.WriteString(line + "\n")
		}
	}
	if len(p.Decls) > 0 {
		b.WriteString("\n")
	}
	for _, d := range p.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			s, err := genFunc(fn)
			if err != nil {
				return "", err
			}
			b.WriteString(s)
			b.WriteString("\n")
		}
	}
	return b.String(), nil
}

func programUsesDB(p *ast.Program) bool {
	for _, d := range p.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			if blockUsesDB(fn.Body) {
				return true
			}
		}
		if g, ok := d.(*ast.GlobalLetDecl); ok {
			if exprUsesDB(g.Init) {
				return true
			}
		}
	}
	return false
}

func blockUsesDB(b *ast.BlockStmt) bool {
	if b == nil {
		return false
	}
	for _, st := range b.Stmts {
		if stmtUsesDB(st) {
			return true
		}
	}
	return false
}

func stmtUsesDB(s ast.Stmt) bool {
	switch st := s.(type) {
	case *ast.LetStmt:
		return exprUsesDB(st.Init)
	case *ast.AssignStmt:
		return exprUsesDB(st.Value)
	case *ast.IfStmt:
		return exprUsesDB(st.Cond) || blockUsesDB(st.Then) || blockUsesDB(st.Else)
	case *ast.WhileStmt:
		return exprUsesDB(st.Cond) || blockUsesDB(st.Body)
	case *ast.MatchStmt:
		if exprUsesDB(st.Subject) {
			return true
		}
		for _, arm := range st.Arms {
			if exprUsesDB(arm.Guard) || blockUsesDB(arm.Body) {
				return true
			}
		}
		return false
	case *ast.ReturnStmt:
		return exprUsesDB(st.Value)
	case *ast.ExprStmt:
		return exprUsesDB(st.Expr)
	default:
		return false
	}
}

func exprUsesDB(e ast.Expr) bool {
	if e == nil {
		return false
	}
	switch ex := e.(type) {
	case *ast.CallExpr:
		if isDBCall(ex.Callee) || isDBCall(ex.Method) {
			return true
		}
		if exprUsesDB(ex.Receiver) {
			return true
		}
		for _, a := range ex.Args {
			if exprUsesDB(a) {
				return true
			}
		}
		return false
	case *ast.BinaryExpr:
		return exprUsesDB(ex.Left) || exprUsesDB(ex.Right)
	case *ast.UnaryExpr:
		return exprUsesDB(ex.Right)
	case *ast.MatchExpr:
		if exprUsesDB(ex.Subject) {
			return true
		}
		for _, arm := range ex.Arms {
			if exprUsesDB(arm.Guard) || exprUsesDB(arm.Value) {
				return true
			}
		}
		return false
	case *ast.StructLitExpr:
		for _, f := range ex.Fields {
			if exprUsesDB(f.Value) {
				return true
			}
		}
		return false
	case *ast.FieldAccessExpr:
		return exprUsesDB(ex.Object)
	default:
		return false
	}
}

func isDBCall(name string) bool {
	switch name {
	case "db_exec", "db_query", "db_exec_with", "db_query_with",
		"db_query_json", "db_query_json_with", "db_query_one_json", "db_query_one_json_with",
		"db_exec_returning_id", "db_exec_returning_id_with",
		"db_exec_params", "db_exec_params_with", "db_query_params", "db_query_params_with",
		"db_query_json_params", "db_query_json_params_with", "db_query_one_json_params", "db_query_one_json_params_with",
		"db_exec_returning_id_params", "db_exec_returning_id_params_with",
		"__std_db_exec", "__std_db_query", "__std_db_exec_with", "__std_db_query_with",
		"__std_db_query_json", "__std_db_query_json_with", "__std_db_query_one_json", "__std_db_query_one_json_with",
		"__std_db_exec_returning_id", "__std_db_exec_returning_id_with",
		"__std_db_exec_params", "__std_db_exec_params_with", "__std_db_query_params", "__std_db_query_params_with",
		"__std_db_query_json_params", "__std_db_query_json_params_with", "__std_db_query_one_json_params", "__std_db_query_one_json_params_with",
		"__std_db_exec_returning_id_params", "__std_db_exec_returning_id_params_with",
		"session_init", "session_put", "session_get_user", "session_delete",
		"__std_session_init", "__std_session_put", "__std_session_get_user", "__std_session_delete":
		return true
	default:
		return false
	}
}

func programUsesSession(p *ast.Program) bool {
	for _, d := range p.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			if blockUsesSession(fn.Body) {
				return true
			}
		}
		if g, ok := d.(*ast.GlobalLetDecl); ok {
			if exprUsesSession(g.Init) {
				return true
			}
		}
	}
	return false
}

func programUsesBcrypt(p *ast.Program) bool {
	for _, d := range p.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			if blockUsesBcrypt(fn.Body) {
				return true
			}
		}
		if g, ok := d.(*ast.GlobalLetDecl); ok {
			if exprUsesBcrypt(g.Init) {
				return true
			}
		}
	}
	return false
}

func blockUsesBcrypt(b *ast.BlockStmt) bool {
	if b == nil {
		return false
	}
	for _, st := range b.Stmts {
		if stmtUsesBcrypt(st) {
			return true
		}
	}
	return false
}

func stmtUsesBcrypt(s ast.Stmt) bool {
	switch st := s.(type) {
	case *ast.LetStmt:
		return exprUsesBcrypt(st.Init)
	case *ast.AssignStmt:
		return exprUsesBcrypt(st.Value)
	case *ast.IfStmt:
		return exprUsesBcrypt(st.Cond) || blockUsesBcrypt(st.Then) || blockUsesBcrypt(st.Else)
	case *ast.WhileStmt:
		return exprUsesBcrypt(st.Cond) || blockUsesBcrypt(st.Body)
	case *ast.MatchStmt:
		if exprUsesBcrypt(st.Subject) {
			return true
		}
		for _, arm := range st.Arms {
			if exprUsesBcrypt(arm.Guard) || blockUsesBcrypt(arm.Body) {
				return true
			}
		}
		return false
	case *ast.ReturnStmt:
		return exprUsesBcrypt(st.Value)
	case *ast.ExprStmt:
		return exprUsesBcrypt(st.Expr)
	default:
		return false
	}
}

func exprUsesBcrypt(e ast.Expr) bool {
	if e == nil {
		return false
	}
	switch ex := e.(type) {
	case *ast.CallExpr:
		if isBcryptCall(ex.Callee) || isBcryptCall(ex.Method) {
			return true
		}
		if exprUsesBcrypt(ex.Receiver) {
			return true
		}
		for _, a := range ex.Args {
			if exprUsesBcrypt(a) {
				return true
			}
		}
		return false
	case *ast.BinaryExpr:
		return exprUsesBcrypt(ex.Left) || exprUsesBcrypt(ex.Right)
	case *ast.UnaryExpr:
		return exprUsesBcrypt(ex.Right)
	case *ast.MatchExpr:
		if exprUsesBcrypt(ex.Subject) {
			return true
		}
		for _, arm := range ex.Arms {
			if exprUsesBcrypt(arm.Guard) || exprUsesBcrypt(arm.Value) {
				return true
			}
		}
		return false
	case *ast.StructLitExpr:
		for _, f := range ex.Fields {
			if exprUsesBcrypt(f.Value) {
				return true
			}
		}
		return false
	case *ast.FieldAccessExpr:
		return exprUsesBcrypt(ex.Object)
	default:
		return false
	}
}

func isBcryptCall(name string) bool {
	switch name {
	case "crypto_bcrypt_hash", "crypto_bcrypt_verify",
		"auth_hash_password", "auth_verify_password",
		"__std_bcrypt_hash", "__std_bcrypt_verify":
		return true
	default:
		return false
	}
}

func programUsesHmac(p *ast.Program) bool {
	for _, d := range p.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			if blockUsesHmac(fn.Body) {
				return true
			}
		}
		if g, ok := d.(*ast.GlobalLetDecl); ok {
			if exprUsesHmac(g.Init) {
				return true
			}
		}
	}
	return false
}

func blockUsesHmac(b *ast.BlockStmt) bool {
	if b == nil {
		return false
	}
	for _, st := range b.Stmts {
		if stmtUsesHmac(st) {
			return true
		}
	}
	return false
}

func stmtUsesHmac(s ast.Stmt) bool {
	switch st := s.(type) {
	case *ast.LetStmt:
		return exprUsesHmac(st.Init)
	case *ast.AssignStmt:
		return exprUsesHmac(st.Value)
	case *ast.IfStmt:
		return exprUsesHmac(st.Cond) || blockUsesHmac(st.Then) || blockUsesHmac(st.Else)
	case *ast.WhileStmt:
		return exprUsesHmac(st.Cond) || blockUsesHmac(st.Body)
	case *ast.MatchStmt:
		if exprUsesHmac(st.Subject) {
			return true
		}
		for _, arm := range st.Arms {
			if exprUsesHmac(arm.Guard) || blockUsesHmac(arm.Body) {
				return true
			}
		}
		return false
	case *ast.ReturnStmt:
		return exprUsesHmac(st.Value)
	case *ast.ExprStmt:
		return exprUsesHmac(st.Expr)
	default:
		return false
	}
}

func exprUsesHmac(e ast.Expr) bool {
	if e == nil {
		return false
	}
	switch ex := e.(type) {
	case *ast.CallExpr:
		if isHmacCall(ex.Callee) || isHmacCall(ex.Method) {
			return true
		}
		if exprUsesHmac(ex.Receiver) {
			return true
		}
		for _, a := range ex.Args {
			if exprUsesHmac(a) {
				return true
			}
		}
		return false
	case *ast.BinaryExpr:
		return exprUsesHmac(ex.Left) || exprUsesHmac(ex.Right)
	case *ast.UnaryExpr:
		return exprUsesHmac(ex.Right)
	case *ast.MatchExpr:
		if exprUsesHmac(ex.Subject) {
			return true
		}
		for _, arm := range ex.Arms {
			if exprUsesHmac(arm.Guard) || exprUsesHmac(arm.Value) {
				return true
			}
		}
		return false
	case *ast.StructLitExpr:
		for _, f := range ex.Fields {
			if exprUsesHmac(f.Value) {
				return true
			}
		}
		return false
	case *ast.FieldAccessExpr:
		return exprUsesHmac(ex.Object)
	default:
		return false
	}
}

func isHmacCall(name string) bool {
	switch name {
	case "crypto_hmac_sha256_hex", "__std_hmac_sha256_hex":
		return true
	default:
		return false
	}
}

func programUsesJwt(p *ast.Program) bool {
	for _, d := range p.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			if blockUsesJwt(fn.Body) {
				return true
			}
		}
		if g, ok := d.(*ast.GlobalLetDecl); ok {
			if exprUsesJwt(g.Init) {
				return true
			}
		}
	}
	return false
}

func blockUsesJwt(b *ast.BlockStmt) bool {
	if b == nil {
		return false
	}
	for _, st := range b.Stmts {
		if stmtUsesJwt(st) {
			return true
		}
	}
	return false
}

func stmtUsesJwt(s ast.Stmt) bool {
	switch st := s.(type) {
	case *ast.LetStmt:
		return exprUsesJwt(st.Init)
	case *ast.AssignStmt:
		return exprUsesJwt(st.Value)
	case *ast.IfStmt:
		return exprUsesJwt(st.Cond) || blockUsesJwt(st.Then) || blockUsesJwt(st.Else)
	case *ast.WhileStmt:
		return exprUsesJwt(st.Cond) || blockUsesJwt(st.Body)
	case *ast.MatchStmt:
		if exprUsesJwt(st.Subject) {
			return true
		}
		for _, arm := range st.Arms {
			if exprUsesJwt(arm.Guard) || blockUsesJwt(arm.Body) {
				return true
			}
		}
		return false
	case *ast.ReturnStmt:
		return exprUsesJwt(st.Value)
	case *ast.ExprStmt:
		return exprUsesJwt(st.Expr)
	default:
		return false
	}
}

func exprUsesJwt(e ast.Expr) bool {
	if e == nil {
		return false
	}
	switch ex := e.(type) {
	case *ast.CallExpr:
		if isJwtCall(ex.Callee) || isJwtCall(ex.Method) {
			return true
		}
		if exprUsesJwt(ex.Receiver) {
			return true
		}
		for _, a := range ex.Args {
			if exprUsesJwt(a) {
				return true
			}
		}
		return false
	case *ast.BinaryExpr:
		return exprUsesJwt(ex.Left) || exprUsesJwt(ex.Right)
	case *ast.UnaryExpr:
		return exprUsesJwt(ex.Right)
	case *ast.MatchExpr:
		if exprUsesJwt(ex.Subject) {
			return true
		}
		for _, arm := range ex.Arms {
			if exprUsesJwt(arm.Guard) || exprUsesJwt(arm.Value) {
				return true
			}
		}
		return false
	case *ast.StructLitExpr:
		for _, f := range ex.Fields {
			if exprUsesJwt(f.Value) {
				return true
			}
		}
		return false
	case *ast.FieldAccessExpr:
		return exprUsesJwt(ex.Object)
	default:
		return false
	}
}

func isJwtCall(name string) bool {
	switch name {
	case "jwt_sign_hs256", "jwt_verify_hs256", "__std_jwt_sign_hs256", "__std_jwt_verify_hs256":
		return true
	default:
		return false
	}
}

func programUsesHttpServeApp(p *ast.Program) bool {
	for _, d := range p.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			if blockUsesHttpServeApp(fn.Body) {
				return true
			}
		}
		if g, ok := d.(*ast.GlobalLetDecl); ok {
			if exprUsesHttpServeApp(g.Init) {
				return true
			}
		}
	}
	return false
}

func blockUsesHttpServeApp(b *ast.BlockStmt) bool {
	if b == nil {
		return false
	}
	for _, st := range b.Stmts {
		if stmtUsesHttpServeApp(st) {
			return true
		}
	}
	return false
}

func stmtUsesHttpServeApp(s ast.Stmt) bool {
	switch st := s.(type) {
	case *ast.LetStmt:
		return exprUsesHttpServeApp(st.Init)
	case *ast.AssignStmt:
		return exprUsesHttpServeApp(st.Value)
	case *ast.IfStmt:
		return exprUsesHttpServeApp(st.Cond) || blockUsesHttpServeApp(st.Then) || blockUsesHttpServeApp(st.Else)
	case *ast.WhileStmt:
		return exprUsesHttpServeApp(st.Cond) || blockUsesHttpServeApp(st.Body)
	case *ast.MatchStmt:
		if exprUsesHttpServeApp(st.Subject) {
			return true
		}
		for _, arm := range st.Arms {
			if exprUsesHttpServeApp(arm.Guard) || blockUsesHttpServeApp(arm.Body) {
				return true
			}
		}
		return false
	case *ast.ReturnStmt:
		return exprUsesHttpServeApp(st.Value)
	case *ast.ExprStmt:
		return exprUsesHttpServeApp(st.Expr)
	default:
		return false
	}
}

func exprUsesHttpServeApp(e ast.Expr) bool {
	if e == nil {
		return false
	}
	switch ex := e.(type) {
	case *ast.CallExpr:
		if isHttpServeAppCall(ex.Callee) || isHttpServeAppCall(ex.Method) {
			return true
		}
		if exprUsesHttpServeApp(ex.Receiver) {
			return true
		}
		for _, a := range ex.Args {
			if exprUsesHttpServeApp(a) {
				return true
			}
		}
		return false
	case *ast.BinaryExpr:
		return exprUsesHttpServeApp(ex.Left) || exprUsesHttpServeApp(ex.Right)
	case *ast.UnaryExpr:
		return exprUsesHttpServeApp(ex.Right)
	case *ast.MatchExpr:
		if exprUsesHttpServeApp(ex.Subject) {
			return true
		}
		for _, arm := range ex.Arms {
			if exprUsesHttpServeApp(arm.Guard) || exprUsesHttpServeApp(arm.Value) {
				return true
			}
		}
		return false
	case *ast.StructLitExpr:
		for _, f := range ex.Fields {
			if exprUsesHttpServeApp(f.Value) {
				return true
			}
		}
		return false
	case *ast.FieldAccessExpr:
		return exprUsesHttpServeApp(ex.Object)
	default:
		return false
	}
}

func isHttpServeAppCall(name string) bool {
	switch name {
	case "http_serve_app", "__std_http_serve_app":
		return true
	default:
		return false
	}
}

func blockUsesSession(b *ast.BlockStmt) bool {
	if b == nil {
		return false
	}
	for _, st := range b.Stmts {
		if stmtUsesSession(st) {
			return true
		}
	}
	return false
}

func stmtUsesSession(s ast.Stmt) bool {
	switch st := s.(type) {
	case *ast.LetStmt:
		return exprUsesSession(st.Init)
	case *ast.AssignStmt:
		return exprUsesSession(st.Value)
	case *ast.IfStmt:
		return exprUsesSession(st.Cond) || blockUsesSession(st.Then) || blockUsesSession(st.Else)
	case *ast.WhileStmt:
		return exprUsesSession(st.Cond) || blockUsesSession(st.Body)
	case *ast.MatchStmt:
		if exprUsesSession(st.Subject) {
			return true
		}
		for _, arm := range st.Arms {
			if exprUsesSession(arm.Guard) || blockUsesSession(arm.Body) {
				return true
			}
		}
		return false
	case *ast.ReturnStmt:
		return exprUsesSession(st.Value)
	case *ast.ExprStmt:
		return exprUsesSession(st.Expr)
	default:
		return false
	}
}

func exprUsesSession(e ast.Expr) bool {
	if e == nil {
		return false
	}
	switch ex := e.(type) {
	case *ast.CallExpr:
		if isSessionCall(ex.Callee) || isSessionCall(ex.Method) {
			return true
		}
		if exprUsesSession(ex.Receiver) {
			return true
		}
		for _, a := range ex.Args {
			if exprUsesSession(a) {
				return true
			}
		}
		return false
	case *ast.BinaryExpr:
		return exprUsesSession(ex.Left) || exprUsesSession(ex.Right)
	case *ast.UnaryExpr:
		return exprUsesSession(ex.Right)
	case *ast.MatchExpr:
		if exprUsesSession(ex.Subject) {
			return true
		}
		for _, arm := range ex.Arms {
			if exprUsesSession(arm.Guard) || exprUsesSession(arm.Value) {
				return true
			}
		}
		return false
	case *ast.StructLitExpr:
		for _, f := range ex.Fields {
			if exprUsesSession(f.Value) {
				return true
			}
		}
		return false
	case *ast.FieldAccessExpr:
		return exprUsesSession(ex.Object)
	default:
		return false
	}
}

func isSessionCall(name string) bool {
	switch name {
	case "session_init", "session_put", "session_get_user", "session_delete",
		"__std_session_init", "__std_session_put", "__std_session_get_user", "__std_session_delete":
		return true
	default:
		return false
	}
}

type httpRouteSeg struct {
	Literal string
	Param   string
	IsParam bool
}

type httpHandler struct {
	Method   string
	Segments []httpRouteSeg
	FuncName string
}

func collectHttpHandlers(p *ast.Program) []httpHandler {
	handlers := []httpHandler{}
	for _, d := range p.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if h, ok := parseHttpHandler(fn); ok {
			handlers = append(handlers, h)
		}
	}
	sort.Slice(handlers, func(i, j int) bool {
		if handlers[i].Method == handlers[j].Method {
			return handlers[i].FuncName < handlers[j].FuncName
		}
		return handlers[i].Method < handlers[j].Method
	})
	return handlers
}

func parseHttpHandler(fn *ast.FuncDecl) (httpHandler, bool) {
	if len(fn.Params) != 1 {
		return httpHandler{}, false
	}
	if string(fn.Params[0].Type) != "ServerRequest" {
		return httpHandler{}, false
	}
	if string(fn.ReturnType) != "ServerResponse" {
		return httpHandler{}, false
	}
	parts := strings.Split(fn.Name, "_")
	if len(parts) < 2 {
		return httpHandler{}, false
	}
	method := strings.ToUpper(parts[0])
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
	default:
		return httpHandler{}, false
	}
	if len(parts) == 2 && parts[1] == "root" {
		return httpHandler{Method: method, Segments: nil, FuncName: fn.Name}, true
	}
	segments := []httpRouteSeg{}
	for i := 1; i < len(parts); {
		if parts[i] == "p" {
			if i+1 >= len(parts) || parts[i+1] == "" {
				return httpHandler{}, false
			}
			segments = append(segments, httpRouteSeg{Param: parts[i+1], IsParam: true})
			i += 2
			continue
		}
		if parts[i] == "" {
			return httpHandler{}, false
		}
		segments = append(segments, httpRouteSeg{Literal: parts[i]})
		i++
	}
	return httpHandler{Method: method, Segments: segments, FuncName: fn.Name}, true
}

func writeHttpServeApp(b *strings.Builder, handlers []httpHandler) {
	b.WriteString("func __bazic_env_int64(key string, def int64) int64 {\n")
	b.WriteString("\tval := strings.TrimSpace(os.Getenv(key))\n")
	b.WriteString("\tif val == \"\" { return def }\n")
	b.WriteString("\tv, err := strconv.ParseInt(val, 10, 64)\n")
	b.WriteString("\tif err != nil { return def }\n")
	b.WriteString("\treturn v\n")
	b.WriteString("}\n")
	b.WriteString("func __std_http_write_response(w http.ResponseWriter, resp ServerResponse) {\n")
	b.WriteString("\tif resp.Status <= 0 { resp.Status = 200 }\n")
	b.WriteString("\tfor _, line := range strings.Split(resp.Headers, \"\\n\") {\n")
	b.WriteString("\t\tif line == \"\" { continue }\n")
	b.WriteString("\t\tparts := strings.SplitN(line, \":\", 2)\n")
	b.WriteString("\t\tif len(parts) != 2 { continue }\n")
	b.WriteString("\t\tkey := strings.TrimSpace(parts[0])\n")
	b.WriteString("\t\tval := strings.TrimSpace(parts[1])\n")
	b.WriteString("\t\tif key != \"\" { w.Header().Add(key, val) }\n")
	b.WriteString("\t}\n")
	b.WriteString("\tw.WriteHeader(int(resp.Status))\n")
	b.WriteString("\t_, _ = io.WriteString(w, resp.Body)\n")
	b.WriteString("}\n")
	b.WriteString("func __std_http_serve_app(addr string) Result[bool, Error] {\n")
	if len(handlers) == 0 {
		b.WriteString("\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: \"no http handlers found\"}}\n")
		b.WriteString("}\n")
		return
	}
	b.WriteString("\tmaxBody := __bazic_env_int64(\"BAZIC_HTTP_MAX_BODY\", 1048576)\n")
	b.WriteString("\treadTimeout := time.Duration(__bazic_env_int64(\"BAZIC_HTTP_READ_TIMEOUT_MS\", 10000)) * time.Millisecond\n")
	b.WriteString("\treadHeaderTimeout := time.Duration(__bazic_env_int64(\"BAZIC_HTTP_READ_HEADER_TIMEOUT_MS\", 5000)) * time.Millisecond\n")
	b.WriteString("\twriteTimeout := time.Duration(__bazic_env_int64(\"BAZIC_HTTP_WRITE_TIMEOUT_MS\", 15000)) * time.Millisecond\n")
	b.WriteString("\tidleTimeout := time.Duration(__bazic_env_int64(\"BAZIC_HTTP_IDLE_TIMEOUT_MS\", 60000)) * time.Millisecond\n")
	b.WriteString("\thandler := func(w http.ResponseWriter, r *http.Request) {\n")
	b.WriteString("\t\tif maxBody > 0 { r.Body = http.MaxBytesReader(w, r.Body, maxBody) }\n")
	b.WriteString("\t\tpath := r.URL.Path\n")
	b.WriteString("\t\tif path != \"/\" && strings.HasSuffix(path, \"/\") { path = strings.TrimSuffix(path, \"/\") }\n")
	b.WriteString("\t\tsegs := []string{}\n")
	b.WriteString("\t\tif path != \"/\" { segs = strings.Split(strings.Trim(path, \"/\"), \"/\") }\n")
	b.WriteString("\t\tmethod := r.Method\n")
	b.WriteString("\t\tbodyBytes, _ := io.ReadAll(r.Body)\n")
	b.WriteString("\t\tbody := string(bodyBytes)\n")
	b.WriteString("\t\theaders := headerString(r.Header)\n")
	b.WriteString("\t\tcookies := cookieString(r.Cookies())\n")
	b.WriteString("\t\tquery := r.URL.RawQuery\n")
	b.WriteString("\t\tremote := r.RemoteAddr\n")
	for _, h := range handlers {
		b.WriteString("\t\tif method == \"" + h.Method + "\" && len(segs) == " + strconv.Itoa(len(h.Segments)) + " {\n")
		condParts := []string{}
		paramAssignments := []string{}
		for i, seg := range h.Segments {
			if seg.IsParam {
				paramAssignments = append(paramAssignments, "\t\t\tparams = append(params, \""+seg.Param+"=\"+segs["+strconv.Itoa(i)+"])\n")
			} else {
				condParts = append(condParts, "segs["+strconv.Itoa(i)+"] == \""+seg.Literal+"\"")
			}
		}
		if len(condParts) > 0 {
			b.WriteString("\t\t\tif " + strings.Join(condParts, " && ") + " {\n")
		} else {
			b.WriteString("\t\t\t{\n")
		}
		b.WriteString("\t\t\t\tparams := make([]string, 0)\n")
		for _, assign := range paramAssignments {
			b.WriteString(assign)
		}
		b.WriteString("\t\t\t\treq := ServerRequest{Method: method, Path: path, Query: query, Headers: headers, Body: body, Remote_addr: remote, Cookies: cookies, Params: strings.Join(params, \"\\n\")}\n")
		b.WriteString("\t\t\t\tresp := " + h.FuncName + "(req)\n")
		b.WriteString("\t\t\t\t__std_http_write_response(w, resp)\n")
		b.WriteString("\t\t\t\treturn\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t}\n")
	}
	b.WriteString("\t\t__std_http_write_response(w, ServerResponse{Status: 404, Headers: \"Content-Type: text/plain; charset=utf-8\", Body: \"not found\"})\n")
	b.WriteString("\t}\n")
	b.WriteString("\tserver := &http.Server{Addr: addr, Handler: http.HandlerFunc(handler), ReadTimeout: readTimeout, ReadHeaderTimeout: readHeaderTimeout, WriteTimeout: writeTimeout, IdleTimeout: idleTimeout}\n")
	b.WriteString("\tif err := server.ListenAndServe(); err != nil {\n")
	b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
}

func genStruct(s *ast.StructDecl) (string, error) {
	var b strings.Builder
	b.WriteString("type ")
	b.WriteString(s.Name)
	if len(s.TypeParams) > 0 {
		b.WriteString("[")
		for i, tp := range s.TypeParams {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(tp)
			b.WriteString(" any")
		}
		b.WriteString("]")
	}
	b.WriteString(" struct {\n")
	for _, f := range s.Fields {
		b.WriteString("\t")
		b.WriteString(exportName(f.Name))
		b.WriteString(" ")
		b.WriteString(mapType(f.Type))
		b.WriteString("\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}

func genInterface(i *ast.InterfaceDecl) (string, error) {
	var b strings.Builder
	b.WriteString("type ")
	b.WriteString(i.Name)
	b.WriteString(" interface {\n")
	for _, m := range i.Methods {
		b.WriteString("\t")
		b.WriteString(exportName(m.Name))
		b.WriteString("(")
		for idx, p := range m.Params {
			if idx > 0 {
				b.WriteString(", ")
			}
			b.WriteString(p.Name)
			b.WriteString(" ")
			b.WriteString(mapType(p.Type))
		}
		b.WriteString(")")
		if m.Return != ast.TypeVoid {
			b.WriteString(" ")
			b.WriteString(mapType(m.Return))
		}
		b.WriteString("\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}

func genEnum(e *ast.EnumDecl) (string, error) {
	var b strings.Builder
	b.WriteString("type ")
	b.WriteString(e.Name)
	b.WriteString(" int\n\n")
	b.WriteString("const (\n")
	for i, v := range e.Variants {
		b.WriteString("\t")
		b.WriteString(v)
		b.WriteString(" ")
		b.WriteString(e.Name)
		b.WriteString(" = ")
		if i == 0 {
			b.WriteString("iota")
		} else {
			b.WriteString("iota")
		}
		b.WriteString("\n")
	}
	b.WriteString(")\n")
	return b.String(), nil
}

func genGlobal(g *ast.GlobalLetDecl) (string, error) {
	rhs, err := genExpr(g.Init)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("var %s %s = %s", g.Name, mapType(g.Type), rhs), nil
}

func genFunc(fn *ast.FuncDecl) (string, error) {
	var b strings.Builder
	b.WriteString("func ")
	b.WriteString(fn.Name)
	if len(fn.TypeParams) > 0 {
		b.WriteString("[")
		for i, tp := range fn.TypeParams {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(tp)
			b.WriteString(" any")
		}
		b.WriteString("]")
	}
	b.WriteString("(")
	for i, p := range fn.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(p.Name)
		b.WriteString(" ")
		b.WriteString(mapType(p.Type))
	}
	b.WriteString(")")
	if fn.ReturnType != ast.TypeVoid {
		b.WriteString(" ")
		b.WriteString(mapType(fn.ReturnType))
	}
	b.WriteString(" {\n")
	for _, st := range fn.Body.Stmts {
		line, err := genStmt(st)
		if err != nil {
			return "", err
		}
		b.WriteString(indent(line))
	}
	b.WriteString("}\n")
	return b.String(), nil
}

func genStmt(s ast.Stmt) (string, error) {
	switch st := s.(type) {
	case *ast.LetStmt:
		rhs, err := genExpr(st.Init)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("var %s %s = %s\n", st.Name, mapType(st.Type), rhs), nil
	case *ast.AssignStmt:
		rhs, err := genExpr(st.Value)
		if err != nil {
			return "", err
		}
		target, err := genAssignTarget(st.Target)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s = %s\n", target, rhs), nil
	case *ast.ExprStmt:
		e, err := genExpr(st.Expr)
		if err != nil {
			return "", err
		}
		return e + "\n", nil
	case *ast.ReturnStmt:
		if st.Value == nil {
			return "return\n", nil
		}
		e, err := genExpr(st.Value)
		if err != nil {
			return "", err
		}
		return "return " + e + "\n", nil
	case *ast.IfStmt:
		cond, err := genExpr(st.Cond)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		b.WriteString("if ")
		b.WriteString(cond)
		b.WriteString(" {\n")
		for _, x := range st.Then.Stmts {
			line, err := genStmt(x)
			if err != nil {
				return "", err
			}
			b.WriteString(indent(line))
		}
		b.WriteString("}")
		if st.Else != nil {
			b.WriteString(" else {\n")
			for _, x := range st.Else.Stmts {
				line, err := genStmt(x)
				if err != nil {
					return "", err
				}
				b.WriteString(indent(line))
			}
			b.WriteString("}")
		}
		b.WriteString("\n")
		return b.String(), nil
	case *ast.WhileStmt:
		cond, err := genExpr(st.Cond)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		b.WriteString("for ")
		b.WriteString(cond)
		b.WriteString(" {\n")
		for _, x := range st.Body.Stmts {
			line, err := genStmt(x)
			if err != nil {
				return "", err
			}
			b.WriteString(indent(line))
		}
		b.WriteString("}\n")
		return b.String(), nil
	case *ast.MatchStmt:
		subject, err := genExpr(st.Subject)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		b.WriteString("switch ")
		b.WriteString(subject)
		b.WriteString(" {\n")
		grouped := groupMatchArms(st.Arms)
		for _, gv := range grouped {
			b.WriteString("case ")
			b.WriteString(gv.Variant)
			b.WriteString(":\n")
			chain, err := genGuardChainStmt(gv.Arms)
			if err != nil {
				return "", err
			}
			b.WriteString(indent(chain))
		}
		b.WriteString("}\n")
		return b.String(), nil
	default:
		return "", fmt.Errorf("codegen: unsupported statement")
	}
}

func genExpr(e ast.Expr) (string, error) {
	switch ex := e.(type) {
	case *ast.IdentExpr:
		return ex.Name, nil
	case *ast.IntExpr:
		return fmt.Sprintf("int64(%s)", strconv.FormatInt(ex.Value, 10)), nil
	case *ast.FloatExpr:
		return strconv.FormatFloat(ex.Value, 'f', -1, 64), nil
	case *ast.BoolExpr:
		if ex.Value {
			return "true", nil
		}
		return "false", nil
	case *ast.StringExpr:
		return strconv.Quote(ex.Value), nil
	case *ast.NilExpr:
		return "", fmt.Errorf("codegen: nil literal is unsupported; semantic validation should reject it")
	case *ast.UnaryExpr:
		r, err := genExpr(ex.Right)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s%s)", ex.Op, r), nil
	case *ast.BinaryExpr:
		l, err := genExpr(ex.Left)
		if err != nil {
			return "", err
		}
		r, err := genExpr(ex.Right)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s %s %s)", l, ex.Op, r), nil
	case *ast.CallExpr:
		if ex.Callee == "" {
			if ex.Receiver != nil {
				return "", fmt.Errorf("codegen: unresolved method call '%s'; semantic resolution required before codegen", ex.Method)
			}
			return "", fmt.Errorf("codegen: unresolved call expression")
		}
		if ex.Callee == "len" {
			parts := make([]string, 0, len(ex.Args))
			for _, a := range ex.Args {
				s, err := genExpr(a)
				if err != nil {
					return "", err
				}
				parts = append(parts, s)
			}
			return fmt.Sprintf("bazic_len(%s)", strings.Join(parts, ", ")), nil
		}
		parts := make([]string, 0, len(ex.Args))
		for _, a := range ex.Args {
			s, err := genExpr(a)
			if err != nil {
				return "", err
			}
			parts = append(parts, s)
		}
		return fmt.Sprintf("%s(%s)", ex.Callee, strings.Join(parts, ", ")), nil
	case *ast.FieldAccessExpr:
		base, err := genExpr(ex.Object)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.%s", base, exportName(ex.Field)), nil
	case *ast.StructLitExpr:
		parts := make([]string, 0, len(ex.Fields))
		ordered := append([]ast.StructLitField{}, ex.Fields...)
		sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Name < ordered[j].Name })
		for _, f := range ordered {
			v, err := genExpr(f.Value)
			if err != nil {
				return "", err
			}
			parts = append(parts, fmt.Sprintf("%s: %s", exportName(f.Name), v))
		}
		return fmt.Sprintf("%s{%s}", mapType(ast.Type(ex.TypeName)), strings.Join(parts, ", ")), nil
	case *ast.MatchExpr:
		subject, err := genExpr(ex.Subject)
		if err != nil {
			return "", err
		}
		if ex.ResolvedType == ast.TypeInvalid {
			return "", fmt.Errorf("codegen: unresolved match expression type")
		}
		var b strings.Builder
		b.WriteString("func() ")
		b.WriteString(mapType(ex.ResolvedType))
		b.WriteString(" {\n")
		b.WriteString("switch ")
		b.WriteString(subject)
		b.WriteString(" {\n")
		grouped := groupMatchExprArms(ex.Arms)
		for _, gv := range grouped {
			b.WriteString("case ")
			b.WriteString(gv.Variant)
			b.WriteString(":\n")
			chain, err := genGuardChainExpr(gv.Arms)
			if err != nil {
				return "", err
			}
			b.WriteString(indent(chain))
		}
		b.WriteString("}\n")
		b.WriteString("panic(\"unreachable match\")\n")
		b.WriteString("}()")
		return b.String(), nil
	default:
		return "", fmt.Errorf("codegen: unsupported expression")
	}
}

func genAssignTarget(e ast.Expr) (string, error) {
	switch ex := e.(type) {
	case *ast.IdentExpr:
		return ex.Name, nil
	case *ast.FieldAccessExpr:
		return genExpr(ex)
	default:
		return "", fmt.Errorf("codegen: invalid assignment target")
	}
}

func mapType(t ast.Type) string {
	switch t {
	case ast.TypeInt:
		return "int64"
	case ast.TypeFloat:
		return "float64"
	case ast.TypeBool:
		return "bool"
	case ast.TypeString:
		return "string"
	case ast.TypeVoid:
		return ""
	case ast.TypeAny:
		return "any"
	default:
		if base, args, ok := splitGenericType(string(t)); ok {
			mapped := make([]string, 0, len(args))
			for _, a := range args {
				mapped = append(mapped, mapType(ast.Type(a)))
			}
			return fmt.Sprintf("%s[%s]", base, strings.Join(mapped, ", "))
		}
		return string(t)
	}
}

func splitGenericType(t string) (string, []string, bool) {
	open := strings.IndexRune(t, '[')
	close := strings.LastIndex(t, "]")
	if open <= 0 || close <= open {
		return "", nil, false
	}
	base := t[:open]
	inner := t[open+1 : close]
	depth := 0
	start := 0
	out := []string{}
	for i, r := range inner {
		switch r {
		case '[':
			depth++
		case ']':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(inner[start:i]))
				start = i + 1
			}
		}
	}
	out = append(out, strings.TrimSpace(inner[start:]))
	return base, out, true
}

func exportName(name string) string {
	if name == "" {
		return name
	}
	r := []rune(name)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] = r[0] - 32
	}
	return string(r)
}

func indent(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines[i] = "\t" + line
	}
	return strings.Join(lines, "\n")
}

type matchGroup[T any] struct {
	Variant string
	Arms    []T
}

func groupMatchArms(arms []ast.MatchArm) []matchGroup[ast.MatchArm] {
	order := []string{}
	by := map[string][]ast.MatchArm{}
	for _, arm := range arms {
		if _, ok := by[arm.Variant]; !ok {
			order = append(order, arm.Variant)
		}
		by[arm.Variant] = append(by[arm.Variant], arm)
	}
	out := make([]matchGroup[ast.MatchArm], 0, len(order))
	for _, v := range order {
		out = append(out, matchGroup[ast.MatchArm]{Variant: v, Arms: by[v]})
	}
	return out
}

func groupMatchExprArms(arms []ast.MatchExprArm) []matchGroup[ast.MatchExprArm] {
	order := []string{}
	by := map[string][]ast.MatchExprArm{}
	for _, arm := range arms {
		if _, ok := by[arm.Variant]; !ok {
			order = append(order, arm.Variant)
		}
		by[arm.Variant] = append(by[arm.Variant], arm)
	}
	out := make([]matchGroup[ast.MatchExprArm], 0, len(order))
	for _, v := range order {
		out = append(out, matchGroup[ast.MatchExprArm]{Variant: v, Arms: by[v]})
	}
	return out
}

func genGuardChainStmt(arms []ast.MatchArm) (string, error) {
	var b strings.Builder
	unguarded := -1
	for i, arm := range arms {
		if arm.Guard == nil {
			unguarded = i
			break
		}
	}
	for i, arm := range arms {
		if arm.Guard == nil {
			continue
		}
		cond, err := genExpr(arm.Guard)
		if err != nil {
			return "", err
		}
		if i == 0 {
			b.WriteString("if ")
		} else {
			b.WriteString("else if ")
		}
		b.WriteString(cond)
		b.WriteString(" {\n")
		for _, st := range arm.Body.Stmts {
			line, err := genStmt(st)
			if err != nil {
				return "", err
			}
			b.WriteString(indent(line))
		}
		b.WriteString("}")
		if i == len(arms)-1 && unguarded == -1 {
			b.WriteString("\n")
		} else {
			b.WriteString(" ")
		}
	}
	if unguarded >= 0 {
		if len(arms) > 0 && arms[0].Guard != nil {
			b.WriteString("else {\n")
		}
		for _, st := range arms[unguarded].Body.Stmts {
			line, err := genStmt(st)
			if err != nil {
				return "", err
			}
			b.WriteString(indent(line))
		}
		if len(arms) > 0 && arms[0].Guard != nil {
			b.WriteString("}\n")
		} else {
			b.WriteString("\n")
		}
	}
	return b.String(), nil
}

func genGuardChainExpr(arms []ast.MatchExprArm) (string, error) {
	var b strings.Builder
	unguarded := -1
	for i, arm := range arms {
		if arm.Guard == nil {
			unguarded = i
			break
		}
	}
	for i, arm := range arms {
		if arm.Guard == nil {
			continue
		}
		cond, err := genExpr(arm.Guard)
		if err != nil {
			return "", err
		}
		val, err := genExpr(arm.Value)
		if err != nil {
			return "", err
		}
		if i == 0 {
			b.WriteString("if ")
		} else {
			b.WriteString("else if ")
		}
		b.WriteString(cond)
		b.WriteString(" {\n")
		b.WriteString("\treturn ")
		b.WriteString(val)
		b.WriteString("\n")
		b.WriteString("} ")
	}
	if unguarded >= 0 {
		if len(arms) > 0 && arms[0].Guard != nil {
			b.WriteString("else {\n")
			b.WriteString("\treturn ")
		} else {
			b.WriteString("return ")
		}
		val, err := genExpr(arms[unguarded].Value)
		if err != nil {
			return "", err
		}
		b.WriteString(val)
		if len(arms) > 0 && arms[0].Guard != nil {
			b.WriteString("\n}\n")
		} else {
			b.WriteString("\n")
		}
	}
	return b.String(), nil
}
