package intrinsics

import (
	"strings"
	"testing"
)

func TestGoStdRuntimeHelperCodeIncludesCoreHelpers(t *testing.T) {
	code := GoStdRuntimeHelperCode()
	for _, want := range []string{
		"func __std_read_file(path string) Result[string, Error]",
		"func __std_time_add_days(rfc3339 string, days int64) Result[string, Error]",
		"func __std_getenv(key string) Result[string, Error]",
		"func __std_path_join(a string, b string) string",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected helper code to include %q", want)
		}
	}
}

func TestGoStdJSONWebHelperCodeIncludesTargetSpecificWebBindings(t *testing.T) {
	wasm := GoStdJSONWebHelperCode("wasm")
	if !strings.Contains(wasm, "bridge := js.Global().Get(\"BAZIC_WEB\")") {
		t.Fatalf("expected wasm helper code to use js bridge")
	}
	host := GoStdJSONWebHelperCode("native")
	if !strings.Contains(host, "web interop only supported in wasm") {
		t.Fatalf("expected non-wasm helper code to emit stub message")
	}
	if !strings.Contains(host, "func __std_json_get_float(s string, key string) Result[float64, Error]") {
		t.Fatalf("expected json helpers to be included in helper code")
	}
}

func TestGoStdHTTPHelperCodeIncludesTypedResponseAndHeaderHelpers(t *testing.T) {
	code := GoStdHTTPHelperCode("HttpResponse", true, true)
	for _, want := range []string{
		"func __std_http_get_opts(url string, connectTimeoutMs int64, timeoutMs int64, headers string, userAgent string, tlsInsecure bool, caBundle string) Result[string, Error]",
		"func __std_http_get_opts_resp(url string, connectTimeoutMs int64, timeoutMs int64, headers string, userAgent string, tlsInsecure bool, caBundle string) Result[HttpResponse, Error]",
		"func headerString(h http.Header) string",
		"func buildHTTPClient(connectTimeoutMs int64, timeoutMs int64, tlsInsecure bool, caBundle string) (*http.Client, error)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected http helper code to include %q", want)
		}
	}
	plain := GoStdHTTPHelperCode("", false, false)
	if strings.Contains(plain, "__std_http_get_opts_resp") {
		t.Fatalf("did not expect typed response helpers without HttpResponse support")
	}
}

func TestGoStdCryptoSystemHelperCodeRespectsFeatureFlags(t *testing.T) {
	full := GoStdCryptoSystemHelperCode(true, true, true)
	for _, want := range []string{
		"func __std_open_url(url string) Result[bool, Error]",
		"func __std_sha256_hex(s string) string",
		"func __std_hmac_sha256_hex(message string, secret string) string",
		"func __std_jwt_sign_hs256(headerJSON string, payloadJSON string, secret string) Result[string, Error]",
		"func __std_bcrypt_hash(password string, cost int64) Result[string, Error]",
	} {
		if !strings.Contains(full, want) {
			t.Fatalf("expected crypto/system helper code to include %q", want)
		}
	}
	minimal := GoStdCryptoSystemHelperCode(false, false, false)
	for _, unwanted := range []string{"__std_hmac_sha256_hex", "__std_jwt_sign_hs256", "__std_bcrypt_hash"} {
		if strings.Contains(minimal, unwanted) {
			t.Fatalf("did not expect %q without feature flag", unwanted)
		}
	}
}

func TestGoStdSessionHelperCodeIncludesCoreSessionOps(t *testing.T) {
	code := GoStdSessionHelperCode()
	for _, want := range []string{
		"func __std_session_init(path string) Result[bool, Error]",
		"func __std_session_put(path string, tokenHash string, userID string, expiresAt string) Result[bool, Error]",
		"func __std_session_get_user(path string, tokenHash string) Result[string, Error]",
		"func __std_session_delete(path string, tokenHash string) Result[bool, Error]",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected session helper code to include %q", want)
		}
	}
}

func TestGoStdDBHelperCodeIncludesCoreDBOps(t *testing.T) {
	code := GoStdDBHelperCode()
	for _, want := range []string{
		"func __std_db_exec(path string, sqlText string) Result[bool, Error]",
		"func normalizeDBDriver(driver string) string",
		"func parseSQLParams(params string) ([]any, error)",
		"func rowsToJSON(rows *sql.Rows) (string, error)",
		"func __std_db_exec_returning_id_with(driver string, dsn string, sqlText string) Result[int64, Error]",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected db helper code to include %q", want)
		}
	}
}

func TestGoHTTPServeAppHelperCodeIncludesRoutingAndEmptyCase(t *testing.T) {
	empty := GoHTTPServeAppHelperCode(nil, "ServerRequest", "ServerResponse")
	if !strings.Contains(empty, "no http handlers found") {
		t.Fatalf("expected empty app helper code to reject missing handlers")
	}
	handlers := []HTTPHandlerSpec{
		{
			Method: "GET",
			Segments: []HTTPRouteSegmentSpec{
				{Literal: "users"},
				{Param: "id", IsParam: true},
			},
			FuncName: "GET_users_p_id",
		},
	}
	code := GoHTTPServeAppHelperCode(handlers, "ServerRequest", "ServerResponse")
	for _, want := range []string{
		"func cookieString(cookies []*http.Cookie) string",
		"func __std_http_write_response(w http.ResponseWriter, resp ServerResponse)",
		"if method == \"GET\" && len(segs) == 2",
		"params = append(params, \"id=\"+segs[1])",
		"resp := GET_users_p_id(req)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected app helper code to include %q", want)
		}
	}
}
