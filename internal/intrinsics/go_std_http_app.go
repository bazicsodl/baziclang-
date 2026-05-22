package intrinsics

import (
	"fmt"
	"strconv"
	"strings"
)

type HTTPRouteSegmentSpec struct {
	Literal string
	Param   string
	IsParam bool
}

type HTTPHandlerSpec struct {
	Method   string
	Segments []HTTPRouteSegmentSpec
	FuncName string
}

func GoHTTPServeAppHelperCode(handlers []HTTPHandlerSpec, serverRequestType string, serverResponseType string) string {
	var b strings.Builder
	b.WriteString("func cookieString(cookies []*http.Cookie) string {\n")
	b.WriteString("\tif len(cookies) == 0 { return \"\" }\n")
	b.WriteString("\tparts := make([]string, 0, len(cookies))\n")
	b.WriteString("\tfor _, c := range cookies {\n")
	b.WriteString("\t\tparts = append(parts, c.Name+\"=\"+c.Value)\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn strings.Join(parts, \"\\n\")\n")
	b.WriteString("}\n\n")
	b.WriteString("func __bazic_env_int64(key string, def int64) int64 {\n")
	b.WriteString("\tval := strings.TrimSpace(os.Getenv(key))\n")
	b.WriteString("\tif val == \"\" { return def }\n")
	b.WriteString("\tv, err := strconv.ParseInt(val, 10, 64)\n")
	b.WriteString("\tif err != nil { return def }\n")
	b.WriteString("\treturn v\n")
	b.WriteString("}\n")
	b.WriteString(fmt.Sprintf("func __std_http_write_response(w http.ResponseWriter, resp %s) {\n", serverResponseType))
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
		return b.String()
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
		b.WriteString(fmt.Sprintf("\t\t\t\treq := %s{Method: method, Path: path, Query: query, Headers: headers, Body: body, Remote_addr: remote, Cookies: cookies, Params: strings.Join(params, \"\\n\")}\n", serverRequestType))
		b.WriteString("\t\t\t\tresp := " + h.FuncName + "(req)\n")
		b.WriteString("\t\t\t\t__std_http_write_response(w, resp)\n")
		b.WriteString("\t\t\t\treturn\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t}\n")
	}
	b.WriteString(fmt.Sprintf("\t\t__std_http_write_response(w, %s{Status: 404, Headers: \"Content-Type: text/plain; charset=utf-8\", Body: \"not found\"})\n", serverResponseType))
	b.WriteString("\t}\n")
	b.WriteString("\tserver := &http.Server{Addr: addr, Handler: http.HandlerFunc(handler), ReadTimeout: readTimeout, ReadHeaderTimeout: readHeaderTimeout, WriteTimeout: writeTimeout, IdleTimeout: idleTimeout}\n")
	b.WriteString("\tif err := server.ListenAndServe(); err != nil {\n")
	b.WriteString("\t\treturn Result[bool, Error]{Is_ok: false, Value: false, Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[bool, Error]{Is_ok: true, Value: true, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	return b.String()
}
