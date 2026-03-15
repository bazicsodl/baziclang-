package codegenllvm

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"baziclang/internal/ast"
)

const (
	anyTagInt    = 1
	anyTagFloat  = 2
	anyTagBool   = 3
	anyTagString = 4
	anyTagOther  = 5
)

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
	if len(handlers) > 1 {
		sort.Slice(handlers, func(i, j int) bool {
			if handlers[i].Method == handlers[j].Method {
				return handlers[i].FuncName < handlers[j].FuncName
			}
			return handlers[i].Method < handlers[j].Method
		})
	}
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

func routePattern(h httpHandler) string {
	if len(h.Segments) == 0 {
		return "/"
	}
	parts := make([]string, 0, len(h.Segments))
	for _, seg := range h.Segments {
		if seg.IsParam {
			parts = append(parts, ":"+seg.Param)
		} else {
			parts = append(parts, seg.Literal)
		}
	}
	return "/" + strings.Join(parts, "/")
}

func GenerateLLVMIR(p *ast.Program) (string, error) {
	p = monomorphizeProgram(p)
	var b strings.Builder
	b.WriteString("; Bazic LLVM IR (early backend)\n")
	b.WriteString("source_filename = \"bazic_module\"\n\n")
	b.WriteString("declare i32 @printf(ptr, ...)\n")
	b.WriteString("declare i64 @strlen(ptr)\n")
	b.WriteString("declare i64 @bazic_len(ptr)\n")
	b.WriteString("declare i32 @strcmp(ptr, ptr)\n")
	b.WriteString("declare ptr @strstr(ptr, ptr)\n")
	b.WriteString("declare i32 @strncmp(ptr, ptr, i64)\n")
	b.WriteString("declare i32 @toupper(i32)\n")
	b.WriteString("declare i32 @tolower(i32)\n")
	b.WriteString("declare i32 @isspace(i32)\n")
	b.WriteString("declare i64 @strtol(ptr, ptr, i32)\n")
	b.WriteString("declare double @strtod(ptr, ptr)\n")
	b.WriteString("declare i32 @snprintf(ptr, i64, ptr, ...)\n")
	b.WriteString("declare ptr @malloc(i64)\n")
	b.WriteString("declare ptr @memcpy(ptr, ptr, i64)\n\n")

	enums := collectEnums(p)
	handlers := collectHttpHandlers(p)
	extraStrings := []string{}
	for _, h := range handlers {
		extraStrings = append(extraStrings, h.Method)
		extraStrings = append(extraStrings, routePattern(h))
	}
	strs := collectStringLiterals(p, extraStrings)
	structs := collectStructs(p)
	ifaces := collectInterfaces(p)
	globals := collectGlobals(p)

	for _, d := range p.Decls {
		switch decl := d.(type) {
		case *ast.StructDecl:
			b.WriteString(fmt.Sprintf("; struct %s\n", decl.Name))
		case *ast.InterfaceDecl:
			b.WriteString(fmt.Sprintf("; interface %s\n", decl.Name))
		case *ast.EnumDecl:
			b.WriteString(fmt.Sprintf("; enum %s\n", decl.Name))
		}
	}
	if len(p.Decls) > 0 {
		b.WriteString("\n")
	}

	if len(structs.order) > 0 {
		for _, name := range structs.order {
			info := structs.byName[name]
			decl, err := emitStructType(name, info, enums, structs, ifaces)
			if err != nil {
				return "", err
			}
			b.WriteString(decl)
		}
		b.WriteString("\n")
	}

	if len(ifaces.order) > 0 {
		for _, name := range ifaces.order {
			b.WriteString(fmt.Sprintf("%%%s = type { ptr, ptr }\n", name))
		}
		b.WriteString("\n")
	}

	b.WriteString("%Any = type { i64, ptr }\n\n")

	if len(strs.ordered) > 0 {
		for _, lit := range strs.ordered {
			name := strs.names[lit]
			b.WriteString(emitStringGlobal(name, lit))
		}
		b.WriteString("\n")
	}

	b.WriteString(emitRouteTable(handlers, strs))
	b.WriteString("\n")

	b.WriteString(emitStringRuntime())
	b.WriteString("\n")
	b.WriteString(emitBuiltinRuntime(structs, ifaces, strs))
	b.WriteString("\n")
	b.WriteString(emitAnyRuntime(strs))
	b.WriteString("\n")
	b.WriteString(emitStdDecls(structs))
	b.WriteString("\n")
	if len(globals.order) > 0 {
		decls, err := emitGlobalDecls(globals, enums, structs, ifaces)
		if err != nil {
			return "", err
		}
		b.WriteString(decls)
		b.WriteString("\n")
	}

	funcSigs := map[string]llvmFuncSig{}
	resultStrErr := ast.Type(resultStructName("string", "Error"))
	resultBoolErr := ast.Type(resultStructName("bool", "Error"))
	resultIntErr := ast.Type(resultStructName("int", "Error"))
	resultFloatErr := ast.Type(resultStructName("float", "Error"))
	for _, d := range p.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if len(fn.TypeParams) > 0 {
			return "", fmt.Errorf("llvm backend: unresolved generic function '%s'", fn.Name)
		}
		params := make([]ast.Type, 0, len(fn.Params))
		for _, p := range fn.Params {
			params = append(params, p.Type)
		}
		funcSigs[fn.Name] = llvmFuncSig{Params: params, Ret: fn.ReturnType}
	}
	if len(globals.order) > 0 {
		initIR, err := emitGlobalInit(globals, funcSigs, enums, structs, ifaces, strs)
		if err != nil {
			return "", err
		}
		b.WriteString(initIR)
		b.WriteString("\n")
	}
	funcSigs["__std_read_file"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_write_file"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultBoolErr}
	funcSigs["__std_read_line"] = llvmFuncSig{Params: []ast.Type{}, Ret: resultStrErr}
	funcSigs["__std_read_all"] = llvmFuncSig{Params: []ast.Type{}, Ret: resultStrErr}
	funcSigs["__std_exists"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeBool}
	funcSigs["__std_mkdir_all"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: resultBoolErr}
	funcSigs["__std_remove"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: resultBoolErr}
	funcSigs["__std_list_dir"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_unix_millis"] = llvmFuncSig{Params: []ast.Type{}, Ret: ast.TypeInt}
	funcSigs["__std_sleep_ms"] = llvmFuncSig{Params: []ast.Type{ast.TypeInt}, Ret: ast.TypeVoid}
	funcSigs["__std_now_rfc3339"] = llvmFuncSig{Params: []ast.Type{}, Ret: ast.TypeString}
	funcSigs["__std_json_escape"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeString}
	funcSigs["__std_json_pretty"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_json_validate"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeBool}
	funcSigs["__std_json_minify"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_json_get_raw"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_json_get_string"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_json_get_bool"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultBoolErr}
	funcSigs["__std_json_get_int"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultIntErr}
	funcSigs["__std_json_get_float"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultFloatErr}
	funcSigs["__std_http_get"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_http_post"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_http_serve_text"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultBoolErr}
	funcSigs["__std_http_serve_app"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: resultBoolErr}
	funcSigs["__std_http_get_opts"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_http_post_opts"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_http_request"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_http_get_opts_resp"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString}, Ret: ast.Type(resultStructName("HttpResponse", "Error"))}
	funcSigs["__std_http_post_opts_resp"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString}, Ret: ast.Type(resultStructName("HttpResponse", "Error"))}
	funcSigs["__std_http_request_resp"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString}, Ret: ast.Type(resultStructName("HttpResponse", "Error"))}
	funcSigs["__std_db_exec"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultBoolErr}
	funcSigs["__std_db_query"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_db_exec_with"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: resultBoolErr}
	funcSigs["__std_db_query_with"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_db_query_json"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_db_query_json_with"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_db_query_one_json"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_db_query_one_json_with"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_db_exec_returning_id"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultIntErr}
	funcSigs["__std_db_exec_returning_id_with"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: resultIntErr}
	funcSigs["__std_db_exec_params"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: resultBoolErr}
	funcSigs["__std_db_exec_params_with"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString}, Ret: resultBoolErr}
	funcSigs["__std_db_query_params"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_db_query_params_with"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_db_query_json_params"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_db_query_json_params_with"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_db_query_one_json_params"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_db_query_one_json_params_with"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_db_exec_returning_id_params"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: resultIntErr}
	funcSigs["__std_db_exec_returning_id_params_with"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString}, Ret: resultIntErr}
	funcSigs["__std_sha256_hex"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeString}
	funcSigs["__std_hmac_sha256_hex"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.TypeString}
	funcSigs["__std_jwt_sign_hs256"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_jwt_verify_hs256"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultBoolErr}
	funcSigs["__std_random_hex"] = llvmFuncSig{Params: []ast.Type{ast.TypeInt}, Ret: resultStrErr}
	funcSigs["__std_bcrypt_hash"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeInt}, Ret: resultStrErr}
	funcSigs["__std_bcrypt_verify"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultBoolErr}
	funcSigs["__std_session_init"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: resultBoolErr}
	funcSigs["__std_session_put"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString}, Ret: resultBoolErr}
	funcSigs["__std_session_get_user"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_session_delete"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultBoolErr}
	funcSigs["__std_time_add_days"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeInt}, Ret: resultStrErr}
	funcSigs["__std_kv_get"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.TypeString}
	funcSigs["__std_header_get"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.TypeString}
	funcSigs["__std_query_get"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.TypeString}
	funcSigs["__std_args"] = llvmFuncSig{Params: []ast.Type{}, Ret: ast.TypeString}
	funcSigs["__std_getenv"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_cwd"] = llvmFuncSig{Params: []ast.Type{}, Ret: resultStrErr}
	funcSigs["__std_chdir"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: resultBoolErr}
	funcSigs["__std_env_list"] = llvmFuncSig{Params: []ast.Type{}, Ret: resultStrErr}
	funcSigs["__std_temp_dir"] = llvmFuncSig{Params: []ast.Type{}, Ret: resultStrErr}
	funcSigs["__std_exe_path"] = llvmFuncSig{Params: []ast.Type{}, Ret: resultStrErr}
	funcSigs["__std_home_dir"] = llvmFuncSig{Params: []ast.Type{}, Ret: resultStrErr}
	funcSigs["__std_web_get_json"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_web_set_json"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: resultBoolErr}
	funcSigs["__std_base64_encode"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeString}
	funcSigs["__std_base64_decode"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: resultStrErr}
	funcSigs["__std_path_basename"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeString}
	funcSigs["__std_path_dirname"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeString}
	funcSigs["__std_path_join"] = llvmFuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.TypeString}
	funcSigs["__std_open_url"] = llvmFuncSig{Params: []ast.Type{ast.TypeString}, Ret: resultBoolErr}

	for _, d := range p.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if len(fn.TypeParams) > 0 {
			return "", fmt.Errorf("llvm backend: unresolved generic function '%s'", fn.Name)
		}
		if fn.Name == "main" {
			mainIR, err := emitMain(fn, funcSigs, globals.slots, enums, structs, ifaces, strs, len(globals.order) > 0)
			if err != nil {
				return "", err
			}
			b.WriteString(mainIR)
			b.WriteString("\n")
			continue
		}
		fnIR, err := emitFunction(fn, funcSigs, globals.slots, enums, structs, ifaces, strs)
		if err != nil {
			return "", err
		}
		b.WriteString(fnIR)
		b.WriteString("\n")
	}

	return b.String(), nil
}

type globalInfo struct {
	Name string
	Type ast.Type
	Init ast.Expr
}

type globalSlot struct {
	ptr string
	typ ast.Type
}

type globalSet struct {
	order []globalInfo
	slots map[string]globalSlot
}

func collectGlobals(p *ast.Program) globalSet {
	out := globalSet{order: []globalInfo{}, slots: map[string]globalSlot{}}
	for _, d := range p.Decls {
		g, ok := d.(*ast.GlobalLetDecl)
		if !ok {
			continue
		}
		info := globalInfo{Name: g.Name, Type: g.Type, Init: g.Init}
		out.order = append(out.order, info)
		out.slots[g.Name] = globalSlot{ptr: "@" + g.Name, typ: g.Type}
	}
	return out
}

type enumInfo struct {
	variantIndex map[string]int
	variantType  map[string]string
	enumTypes    map[string]bool
}

func collectEnums(p *ast.Program) enumInfo {
	info := enumInfo{
		variantIndex: map[string]int{},
		variantType:  map[string]string{},
		enumTypes:    map[string]bool{},
	}
	for _, d := range p.Decls {
		ed, ok := d.(*ast.EnumDecl)
		if !ok {
			continue
		}
		info.enumTypes[ed.Name] = true
		for i, v := range ed.Variants {
			info.variantIndex[v] = i
			info.variantType[v] = ed.Name
		}
	}
	return info
}

type structFieldInfo struct {
	Name string
	Type ast.Type
}

type structInfo struct {
	Fields     []structFieldInfo
	FieldIndex map[string]int
}

type structPool struct {
	byName map[string]structInfo
	order  []string
}

type interfacePool struct {
	names map[string]bool
	order []string
}

func collectStructs(p *ast.Program) structPool {
	pool := structPool{
		byName: map[string]structInfo{},
		order:  []string{},
	}
	for _, d := range p.Decls {
		decl, ok := d.(*ast.StructDecl)
		if !ok || len(decl.TypeParams) > 0 {
			continue
		}
		if _, exists := pool.byName[decl.Name]; exists {
			continue
		}
		fields := make([]structFieldInfo, 0, len(decl.Fields))
		index := map[string]int{}
		for i, f := range decl.Fields {
			fields = append(fields, structFieldInfo{Name: f.Name, Type: f.Type})
			index[f.Name] = i
		}
		pool.byName[decl.Name] = structInfo{Fields: fields, FieldIndex: index}
		pool.order = append(pool.order, decl.Name)
	}
	return pool
}

func collectInterfaces(p *ast.Program) interfacePool {
	pool := interfacePool{
		names: map[string]bool{},
		order: []string{},
	}
	for _, d := range p.Decls {
		decl, ok := d.(*ast.InterfaceDecl)
		if !ok {
			continue
		}
		if pool.names[decl.Name] {
			continue
		}
		pool.names[decl.Name] = true
		pool.order = append(pool.order, decl.Name)
	}
	return pool
}

func emitStructType(name string, info structInfo, enums enumInfo, structs structPool, ifaces interfacePool) (string, error) {
	parts := make([]string, 0, len(info.Fields))
	for _, f := range info.Fields {
		llvmType, ok := mapLLVMType(f.Type, enums, structs, ifaces)
		if !ok {
			return "", fmt.Errorf("llvm backend: unsupported field type '%s.%s' (%s)", name, f.Name, f.Type)
		}
		parts = append(parts, llvmType)
	}
	return fmt.Sprintf("%%%s = type { %s }\n", name, strings.Join(parts, ", ")), nil
}

type stringPool struct {
	names   map[string]string
	ordered []string
}

func collectStringLiterals(p *ast.Program, extra []string) stringPool {
	pool := stringPool{
		names:   map[string]string{},
		ordered: []string{},
	}
	add := func(s string) {
		if _, ok := pool.names[s]; ok {
			return
		}
		name := fmt.Sprintf(".str%d", len(pool.ordered))
		pool.names[s] = name
		pool.ordered = append(pool.ordered, s)
	}
	add("%ld")
	add("%ld\n")
	add("%g")
	add("%g\n")
	add("%s")
	add("%s\n")
	add("true")
	add("false")
	add("")
	add("invalid int")
	add("invalid float")
	add("std unavailable")
	add("<any>")
	for _, s := range extra {
		add(s)
	}
	for _, d := range p.Decls {
		switch decl := d.(type) {
		case *ast.FuncDecl:
			collectStringsFromBlock(decl.Body, add)
		case *ast.GlobalLetDecl:
			collectStringsFromExpr(decl.Init, add)
		}
	}
	return pool
}

func emitAnyRuntime(strs stringPool) string {
	var b strings.Builder
	trueName, trueLen := stringGlobalRef(strs, "true")
	falseName, falseLen := stringGlobalRef(strs, "false")
	anyName, anyLen := stringGlobalRef(strs, "<any>")

	b.WriteString("define ptr @bazic_any_to_str(%Any %v) {\n")
	b.WriteString("entry:\n")
	b.WriteString("  %tag = extractvalue %Any %v, 0\n")
	b.WriteString("  %payload = extractvalue %Any %v, 1\n")
	b.WriteString("  switch i64 %tag, label %any_fallback [\n")
	b.WriteString("    i64 1, label %any_int\n")
	b.WriteString("    i64 2, label %any_float\n")
	b.WriteString("    i64 3, label %any_bool\n")
	b.WriteString("    i64 4, label %any_string\n")
	b.WriteString("  ]\n")
	b.WriteString("any_int:\n")
	b.WriteString("  %ival = ptrtoint ptr %payload to i64\n")
	b.WriteString("  %istr = call ptr @bazic_int_to_str(i64 %ival)\n")
	b.WriteString("  ret ptr %istr\n")
	b.WriteString("any_float:\n")
	b.WriteString("  %fbits = ptrtoint ptr %payload to i64\n")
	b.WriteString("  %fval = bitcast i64 %fbits to double\n")
	b.WriteString("  %fstr = call ptr @bazic_float_to_str(double %fval)\n")
	b.WriteString("  ret ptr %fstr\n")
	b.WriteString("any_bool:\n")
	b.WriteString("  %bval = ptrtoint ptr %payload to i64\n")
	b.WriteString("  %btrue = icmp ne i64 %bval, 0\n")
	b.WriteString(fmt.Sprintf("  %%btrue_ptr = %s\n", stringGEP(trueName, trueLen)))
	b.WriteString(fmt.Sprintf("  %%bfalse_ptr = %s\n", stringGEP(falseName, falseLen)))
	b.WriteString("  %bstr = select i1 %btrue, ptr %btrue_ptr, ptr %bfalse_ptr\n")
	b.WriteString("  ret ptr %bstr\n")
	b.WriteString("any_string:\n")
	b.WriteString("  ret ptr %payload\n")
	b.WriteString("any_fallback:\n")
	b.WriteString(fmt.Sprintf("  %%any_ptr = %s\n", stringGEP(anyName, anyLen)))
	b.WriteString("  ret ptr %any_ptr\n")
	b.WriteString("}\n\n")

	b.WriteString("define i8 @bazic_any_eq(%Any %a, %Any %b) {\n")
	b.WriteString("entry:\n")
	b.WriteString("  %tagA = extractvalue %Any %a, 0\n")
	b.WriteString("  %tagB = extractvalue %Any %b, 0\n")
	b.WriteString("  %tagsame = icmp eq i64 %tagA, %tagB\n")
	b.WriteString("  br i1 %tagsame, label %any_eq_switch, label %any_eq_false\n")
	b.WriteString("any_eq_false:\n")
	b.WriteString("  ret i8 0\n")
	b.WriteString("any_eq_switch:\n")
	b.WriteString("  %payloadA = extractvalue %Any %a, 1\n")
	b.WriteString("  %payloadB = extractvalue %Any %b, 1\n")
	b.WriteString("  switch i64 %tagA, label %any_eq_ptr [\n")
	b.WriteString("    i64 1, label %any_eq_int\n")
	b.WriteString("    i64 2, label %any_eq_float\n")
	b.WriteString("    i64 3, label %any_eq_bool\n")
	b.WriteString("    i64 4, label %any_eq_string\n")
	b.WriteString("  ]\n")
	b.WriteString("any_eq_int:\n")
	b.WriteString("  %aint = ptrtoint ptr %payloadA to i64\n")
	b.WriteString("  %bint = ptrtoint ptr %payloadB to i64\n")
	b.WriteString("  %eqi = icmp eq i64 %aint, %bint\n")
	b.WriteString("  %eqi8 = zext i1 %eqi to i8\n")
	b.WriteString("  ret i8 %eqi8\n")
	b.WriteString("any_eq_float:\n")
	b.WriteString("  %abits = ptrtoint ptr %payloadA to i64\n")
	b.WriteString("  %bbits = ptrtoint ptr %payloadB to i64\n")
	b.WriteString("  %af = bitcast i64 %abits to double\n")
	b.WriteString("  %bf = bitcast i64 %bbits to double\n")
	b.WriteString("  %eqf = fcmp oeq double %af, %bf\n")
	b.WriteString("  %eqf8 = zext i1 %eqf to i8\n")
	b.WriteString("  ret i8 %eqf8\n")
	b.WriteString("any_eq_bool:\n")
	b.WriteString("  %ab = ptrtoint ptr %payloadA to i64\n")
	b.WriteString("  %bb = ptrtoint ptr %payloadB to i64\n")
	b.WriteString("  %eqb = icmp eq i64 %ab, %bb\n")
	b.WriteString("  %eqb8 = zext i1 %eqb to i8\n")
	b.WriteString("  ret i8 %eqb8\n")
	b.WriteString("any_eq_string:\n")
	b.WriteString("  %cmp = call i32 @bazic_str_cmp(ptr %payloadA, ptr %payloadB)\n")
	b.WriteString("  %eqs = icmp eq i32 %cmp, 0\n")
	b.WriteString("  %eqs8 = zext i1 %eqs to i8\n")
	b.WriteString("  ret i8 %eqs8\n")
	b.WriteString("any_eq_ptr:\n")
	b.WriteString("  %eqp = icmp eq ptr %payloadA, %payloadB\n")
	b.WriteString("  %eqp8 = zext i1 %eqp to i8\n")
	b.WriteString("  ret i8 %eqp8\n")
	b.WriteString("}\n")
	return b.String()
}

func emitGlobalDecls(globals globalSet, enums enumInfo, structs structPool, ifaces interfacePool) (string, error) {
	var b strings.Builder
	for _, g := range globals.order {
		llvmType, ok := mapLLVMType(g.Type, enums, structs, ifaces)
		if !ok {
			return "", fmt.Errorf("llvm backend: unsupported global type '%s' (%s)", g.Name, g.Type)
		}
		b.WriteString(fmt.Sprintf("@%s = global %s %s\n", g.Name, llvmType, defaultLLVMValue(g.Type, enums, structs, ifaces)))
	}
	return b.String(), nil
}

func emitGlobalInit(globals globalSet, funcs map[string]llvmFuncSig, enums enumInfo, structs structPool, ifaces interfacePool, strs stringPool) (string, error) {
	var b strings.Builder
	b.WriteString("define void @__bazic_init_globals() {\n")
	b.WriteString("entry:\n")
	ctx := newFuncCtx(enums, structs, ifaces, strs, false, globals.slots)
	for _, g := range globals.order {
		slot, ok := globals.slots[g.Name]
		if !ok {
			continue
		}
		code, value, t, ok := emitExpr(ctx, g.Init, funcs)
		if !ok {
			return "", fmt.Errorf("llvm backend: unsupported global init for '%s' (%T)", g.Name, g.Init)
		}
		if slot.typ == ast.TypeAny && t != ast.TypeAny {
			boxCode, boxed, ok := boxToAny(ctx, value, t)
			if !ok {
				return "", fmt.Errorf("llvm backend: unsupported global any init for '%s'", g.Name)
			}
			code += boxCode
			value = boxed
			t = ast.TypeAny
		}
		if t != slot.typ {
			return "", fmt.Errorf("llvm backend: global init type mismatch for '%s' (got %s, expected %s)", g.Name, t, slot.typ)
		}
		b.WriteString(code)
		llvmType, _ := mapLLVMType(slot.typ, enums, structs, ifaces)
		b.WriteString(fmt.Sprintf("  store %s %s, ptr %s\n", llvmType, value, slot.ptr))
	}
	b.WriteString("  ret void\n")
	b.WriteString("}\n")
	return b.String(), nil
}
func collectStringsFromBlock(b *ast.BlockStmt, add func(string)) {
	if b == nil {
		return
	}
	for _, st := range b.Stmts {
		switch s := st.(type) {
		case *ast.LetStmt:
			collectStringsFromExpr(s.Init, add)
		case *ast.AssignStmt:
			collectStringsFromExpr(s.Value, add)
		case *ast.IfStmt:
			collectStringsFromExpr(s.Cond, add)
			collectStringsFromBlock(s.Then, add)
			collectStringsFromBlock(s.Else, add)
		case *ast.WhileStmt:
			collectStringsFromExpr(s.Cond, add)
			collectStringsFromBlock(s.Body, add)
		case *ast.MatchStmt:
			collectStringsFromExpr(s.Subject, add)
			for _, arm := range s.Arms {
				collectStringsFromBlock(arm.Body, add)
			}
		case *ast.ReturnStmt:
			collectStringsFromExpr(s.Value, add)
		case *ast.ExprStmt:
			collectStringsFromExpr(s.Expr, add)
		}
	}
}

func collectStringsFromExpr(e ast.Expr, add func(string)) {
	switch ex := e.(type) {
	case *ast.StringExpr:
		add(ex.Value)
	case *ast.UnaryExpr:
		collectStringsFromExpr(ex.Right, add)
	case *ast.BinaryExpr:
		collectStringsFromExpr(ex.Left, add)
		collectStringsFromExpr(ex.Right, add)
	case *ast.CallExpr:
		for _, a := range ex.Args {
			collectStringsFromExpr(a, add)
		}
		if ex.Receiver != nil {
			collectStringsFromExpr(ex.Receiver, add)
		}
	case *ast.FieldAccessExpr:
		collectStringsFromExpr(ex.Object, add)
	case *ast.StructLitExpr:
		for _, f := range ex.Fields {
			collectStringsFromExpr(f.Value, add)
		}
	case *ast.MatchExpr:
		collectStringsFromExpr(ex.Subject, add)
		for _, arm := range ex.Arms {
			collectStringsFromExpr(arm.Value, add)
		}
	}
}

func emitStringGlobal(name string, value string) string {
	escaped := escapeLLVMString(value)
	length := len([]byte(value)) + 1
	return fmt.Sprintf("@%s = private unnamed_addr constant [%d x i8] c\"%s\\00\"\n", name, length, escaped)
}

func escapeLLVMString(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch ch {
		case '\\':
			b.WriteString("\\5C")
		case '"':
			b.WriteString("\\22")
		case '\n':
			b.WriteString("\\0A")
		case '\r':
			b.WriteString("\\0D")
		case '\t':
			b.WriteString("\\09")
		default:
			if ch < 32 || ch >= 127 {
				b.WriteString(fmt.Sprintf("\\%02X", ch))
			} else {
				b.WriteByte(ch)
			}
		}
	}
	return b.String()
}

func stringGlobalRef(strs stringPool, lit string) (string, int) {
	name := strs.names[lit]
	return name, len([]byte(lit)) + 1
}

func stringGEP(name string, length int) string {
	return fmt.Sprintf("getelementptr inbounds [%d x i8], ptr @%s, i64 0, i64 0", length, name)
}

func stringGEPConst(name string, length int) string {
	return fmt.Sprintf("getelementptr inbounds ([%d x i8], ptr @%s, i64 0, i64 0)", length, name)
}

func emitRouteTable(handlers []httpHandler, strs stringPool) string {
	var b strings.Builder
	b.WriteString("%bazic_route = type { ptr, ptr, ptr }\n")
	if len(handlers) == 0 {
		b.WriteString("@__bazic_routes = global [0 x %bazic_route] []\n")
		b.WriteString("@__bazic_routes_len = global i64 0\n")
		return b.String()
	}
	b.WriteString(fmt.Sprintf("@__bazic_routes = global [%d x %%bazic_route] [\n", len(handlers)))
	for i, h := range handlers {
		methodName, methodLen := stringGlobalRef(strs, h.Method)
		path := routePattern(h)
		pathName, pathLen := stringGlobalRef(strs, path)
		methodPtr := stringGEPConst(methodName, methodLen)
		pathPtr := stringGEPConst(pathName, pathLen)
		b.WriteString(fmt.Sprintf("  %%bazic_route { ptr %s, ptr %s, ptr @%s }", methodPtr, pathPtr, h.FuncName))
		if i+1 < len(handlers) {
			b.WriteString(",\n")
		} else {
			b.WriteString("\n")
		}
	}
	b.WriteString("]\n")
	b.WriteString(fmt.Sprintf("@__bazic_routes_len = global i64 %d\n", len(handlers)))
	return b.String()
}

func emitStringRuntime() string {
	var b strings.Builder
	b.WriteString("define ptr @bazic_str_concat(ptr %a, ptr %b) {\n")
	b.WriteString("entry:\n")
	b.WriteString("  %lenA = call i64 @strlen(ptr %a)\n")
	b.WriteString("  %lenB = call i64 @strlen(ptr %b)\n")
	b.WriteString("  %aempty = icmp eq i64 %lenA, 0\n")
	b.WriteString("  br i1 %aempty, label %ret_b, label %check_b\n")
	b.WriteString("ret_b:\n")
	b.WriteString("  ret ptr %b\n")
	b.WriteString("check_b:\n")
	b.WriteString("  %bempty = icmp eq i64 %lenB, 0\n")
	b.WriteString("  br i1 %bempty, label %ret_a, label %cont\n")
	b.WriteString("ret_a:\n")
	b.WriteString("  ret ptr %a\n")
	b.WriteString("cont:\n")
	b.WriteString("  %sum = add i64 %lenA, %lenB\n")
	b.WriteString("  %total = add i64 %sum, 1\n")
	b.WriteString("  %buf = call ptr @malloc(i64 %total)\n")
	b.WriteString("  call ptr @memcpy(ptr %buf, ptr %a, i64 %lenA)\n")
	b.WriteString("  %dstB = getelementptr i8, ptr %buf, i64 %lenA\n")
	b.WriteString("  call ptr @memcpy(ptr %dstB, ptr %b, i64 %lenB)\n")
	b.WriteString("  %end = getelementptr i8, ptr %buf, i64 %sum\n")
	b.WriteString("  store i8 0, ptr %end\n")
	b.WriteString("  ret ptr %buf\n")
	b.WriteString("}\n\n")
	b.WriteString("define i32 @bazic_str_cmp(ptr %a, ptr %b) {\n")
	b.WriteString("entry:\n")
	b.WriteString("  %c = call i32 @strcmp(ptr %a, ptr %b)\n")
	b.WriteString("  ret i32 %c\n")
	b.WriteString("}\n")
	return b.String()
}

func emitBuiltinRuntime(structs structPool, ifaces interfacePool, strs stringPool) string {
	var b strings.Builder
	_ = ifaces
	b.WriteString("define i8 @bazic_contains(ptr %s, ptr %sub) {\n")
	b.WriteString("entry:\n")
	b.WriteString("  %found = call ptr @strstr(ptr %s, ptr %sub)\n")
	b.WriteString("  %ok = icmp ne ptr %found, null\n")
	b.WriteString("  %ok8 = zext i1 %ok to i8\n")
	b.WriteString("  ret i8 %ok8\n")
	b.WriteString("}\n\n")

	b.WriteString("define i8 @bazic_starts_with(ptr %s, ptr %prefix) {\n")
	b.WriteString("entry:\n")
	b.WriteString("  %len = call i64 @strlen(ptr %prefix)\n")
	b.WriteString("  %cmp = call i32 @strncmp(ptr %s, ptr %prefix, i64 %len)\n")
	b.WriteString("  %ok = icmp eq i32 %cmp, 0\n")
	b.WriteString("  %ok8 = zext i1 %ok to i8\n")
	b.WriteString("  ret i8 %ok8\n")
	b.WriteString("}\n\n")

	b.WriteString("define i8 @bazic_ends_with(ptr %s, ptr %suffix) {\n")
	b.WriteString("entry:\n")
	b.WriteString("  %lenS = call i64 @strlen(ptr %s)\n")
	b.WriteString("  %lenT = call i64 @strlen(ptr %suffix)\n")
	b.WriteString("  %short = icmp ult i64 %lenS, %lenT\n")
	b.WriteString("  br i1 %short, label %retfalse, label %cont\n")
	b.WriteString("retfalse:\n")
	b.WriteString("  ret i8 0\n")
	b.WriteString("cont:\n")
	b.WriteString("  %start = sub i64 %lenS, %lenT\n")
	b.WriteString("  %ptr = getelementptr i8, ptr %s, i64 %start\n")
	b.WriteString("  %cmp = call i32 @strncmp(ptr %ptr, ptr %suffix, i64 %lenT)\n")
	b.WriteString("  %ok = icmp eq i32 %cmp, 0\n")
	b.WriteString("  %ok8 = zext i1 %ok to i8\n")
	b.WriteString("  ret i8 %ok8\n")
	b.WriteString("}\n\n")

	b.WriteString(emitCaseTransform("bazic_to_upper", "toupper"))
	b.WriteString("\n")
	b.WriteString(emitCaseTransform("bazic_to_lower", "tolower"))
	b.WriteString("\n")
	b.WriteString(emitTrimSpace(strs))
	b.WriteString("\n")
	b.WriteString(emitRepeat(strs))
	b.WriteString("\n")
	b.WriteString(emitReplace())
	b.WriteString("\n")
	b.WriteString(emitIntToStr(strs))
	b.WriteString("\n")
	b.WriteString(emitFloatToStr(strs))
	b.WriteString("\n")
	if _, ok := structs.byName[resultStructName("int", "Error")]; ok {
		b.WriteString(emitParseInt(structs, strs))
		b.WriteString("\n")
	}
	if _, ok := structs.byName[resultStructName("float", "Error")]; ok {
		b.WriteString(emitParseFloat(structs, strs))
		b.WriteString("\n")
	}
	return b.String()
}

func emitStdDecls(structs structPool) string {
	var b strings.Builder
	resultStrErr := resultStructName("string", "Error")
	resultBoolErr := resultStructName("bool", "Error")
	resultIntErr := resultStructName("int", "Error")
	resultFloatErr := resultStructName("float", "Error")
	if _, ok := structs.byName[resultStrErr]; ok {
		b.WriteString("declare void @__std_read_file(ptr sret(%" + resultStrErr + "), ptr)\n")
		b.WriteString("declare void @__std_read_line(ptr sret(%" + resultStrErr + "))\n")
		b.WriteString("declare void @__std_read_all(ptr sret(%" + resultStrErr + "))\n")
		b.WriteString("declare void @__std_list_dir(ptr sret(%" + resultStrErr + "), ptr)\n")
		b.WriteString("declare void @__std_json_pretty(ptr sret(%" + resultStrErr + "), ptr)\n")
		b.WriteString("declare i8 @__std_json_validate(ptr)\n")
		b.WriteString("declare void @__std_json_minify(ptr sret(%" + resultStrErr + "), ptr)\n")
		b.WriteString("declare void @__std_json_get_raw(ptr sret(%" + resultStrErr + "), ptr, ptr)\n")
		b.WriteString("declare void @__std_json_get_string(ptr sret(%" + resultStrErr + "), ptr, ptr)\n")
		if _, ok := structs.byName[resultIntErr]; ok {
			b.WriteString("declare void @__std_json_get_int(ptr sret(%" + resultIntErr + "), ptr, ptr)\n")
		}
		if _, ok := structs.byName[resultFloatErr]; ok {
			b.WriteString("declare void @__std_json_get_float(ptr sret(%" + resultFloatErr + "), ptr, ptr)\n")
		}
		b.WriteString("declare void @__std_http_get(ptr sret(%" + resultStrErr + "), ptr)\n")
		b.WriteString("declare void @__std_http_post(ptr sret(%" + resultStrErr + "), ptr, ptr)\n")
		b.WriteString("declare void @__std_http_get_opts(ptr sret(%" + resultStrErr + "), ptr, i64, i64, ptr, ptr, i8, ptr)\n")
		b.WriteString("declare void @__std_http_post_opts(ptr sret(%" + resultStrErr + "), ptr, ptr, i64, i64, ptr, ptr, ptr, i8, ptr)\n")
		b.WriteString("declare void @__std_http_request(ptr sret(%" + resultStrErr + "), ptr, ptr, ptr, i64, i64, ptr, ptr, ptr, i8, ptr)\n")
		if _, ok := structs.byName["HttpResponse"]; ok {
			resultRespErr := resultStructName("HttpResponse", "Error")
			b.WriteString("declare void @__std_http_get_opts_resp(ptr sret(%" + resultRespErr + "), ptr, i64, i64, ptr, ptr, i8, ptr)\n")
			b.WriteString("declare void @__std_http_post_opts_resp(ptr sret(%" + resultRespErr + "), ptr, ptr, i64, i64, ptr, ptr, ptr, i8, ptr)\n")
			b.WriteString("declare void @__std_http_request_resp(ptr sret(%" + resultRespErr + "), ptr, ptr, ptr, i64, i64, ptr, ptr, ptr, i8, ptr)\n")
		}
		b.WriteString("declare void @__std_db_exec(ptr sret(%" + resultBoolErr + "), ptr, ptr)\n")
		b.WriteString("declare void @__std_db_query(ptr sret(%" + resultStrErr + "), ptr, ptr)\n")
		b.WriteString("declare void @__std_db_exec_with(ptr sret(%" + resultBoolErr + "), ptr, ptr, ptr)\n")
		b.WriteString("declare void @__std_db_query_with(ptr sret(%" + resultStrErr + "), ptr, ptr, ptr)\n")
		b.WriteString("declare void @__std_db_query_json(ptr sret(%" + resultStrErr + "), ptr, ptr)\n")
		b.WriteString("declare void @__std_db_query_json_with(ptr sret(%" + resultStrErr + "), ptr, ptr, ptr)\n")
		b.WriteString("declare void @__std_db_query_one_json(ptr sret(%" + resultStrErr + "), ptr, ptr)\n")
		b.WriteString("declare void @__std_db_query_one_json_with(ptr sret(%" + resultStrErr + "), ptr, ptr, ptr)\n")
		b.WriteString("declare void @__std_db_exec_params(ptr sret(%" + resultBoolErr + "), ptr, ptr, ptr)\n")
		b.WriteString("declare void @__std_db_exec_params_with(ptr sret(%" + resultBoolErr + "), ptr, ptr, ptr, ptr)\n")
		b.WriteString("declare void @__std_db_query_params(ptr sret(%" + resultStrErr + "), ptr, ptr, ptr)\n")
		b.WriteString("declare void @__std_db_query_params_with(ptr sret(%" + resultStrErr + "), ptr, ptr, ptr, ptr)\n")
		b.WriteString("declare void @__std_db_query_json_params(ptr sret(%" + resultStrErr + "), ptr, ptr, ptr)\n")
		b.WriteString("declare void @__std_db_query_json_params_with(ptr sret(%" + resultStrErr + "), ptr, ptr, ptr, ptr)\n")
		b.WriteString("declare void @__std_db_query_one_json_params(ptr sret(%" + resultStrErr + "), ptr, ptr, ptr)\n")
		b.WriteString("declare void @__std_db_query_one_json_params_with(ptr sret(%" + resultStrErr + "), ptr, ptr, ptr, ptr)\n")
		if _, ok := structs.byName[resultIntErr]; ok {
			b.WriteString("declare void @__std_db_exec_returning_id(ptr sret(%" + resultIntErr + "), ptr, ptr)\n")
			b.WriteString("declare void @__std_db_exec_returning_id_with(ptr sret(%" + resultIntErr + "), ptr, ptr, ptr)\n")
			b.WriteString("declare void @__std_db_exec_returning_id_params(ptr sret(%" + resultIntErr + "), ptr, ptr, ptr)\n")
			b.WriteString("declare void @__std_db_exec_returning_id_params_with(ptr sret(%" + resultIntErr + "), ptr, ptr, ptr, ptr)\n")
		}
		b.WriteString("declare void @__std_random_hex(ptr sret(%" + resultStrErr + "), i64)\n")
		b.WriteString("declare void @__std_bcrypt_hash(ptr sret(%" + resultStrErr + "), ptr, i64)\n")
		b.WriteString("declare void @__std_session_get_user(ptr sret(%" + resultStrErr + "), ptr, ptr)\n")
		b.WriteString("declare void @__std_time_add_days(ptr sret(%" + resultStrErr + "), ptr, i64)\n")
		b.WriteString("declare ptr @__std_args()\n")
		b.WriteString("declare void @__std_getenv(ptr sret(%" + resultStrErr + "), ptr)\n")
		b.WriteString("declare void @__std_cwd(ptr sret(%" + resultStrErr + "))\n")
		b.WriteString("declare void @__std_chdir(ptr sret(%" + resultBoolErr + "), ptr)\n")
		b.WriteString("declare void @__std_env_list(ptr sret(%" + resultStrErr + "))\n")
		b.WriteString("declare void @__std_temp_dir(ptr sret(%" + resultStrErr + "))\n")
		b.WriteString("declare void @__std_exe_path(ptr sret(%" + resultStrErr + "))\n")
		b.WriteString("declare void @__std_home_dir(ptr sret(%" + resultStrErr + "))\n")
		b.WriteString("declare void @__std_web_get_json(ptr sret(%" + resultStrErr + "), ptr)\n")
		b.WriteString("declare void @__std_web_set_json(ptr sret(%" + resultBoolErr + "), ptr, ptr)\n")
		b.WriteString("declare ptr @__std_base64_encode(ptr)\n")
		b.WriteString("declare void @__std_base64_decode(ptr sret(%" + resultStrErr + "), ptr)\n")
		b.WriteString("declare ptr @__std_kv_get(ptr, ptr)\n")
		b.WriteString("declare ptr @__std_header_get(ptr, ptr)\n")
		b.WriteString("declare ptr @__std_query_get(ptr, ptr)\n")
		b.WriteString("declare ptr @__std_path_basename(ptr)\n")
		b.WriteString("declare ptr @__std_path_dirname(ptr)\n")
		b.WriteString("declare ptr @__std_path_join(ptr, ptr)\n")
	}
	if _, ok := structs.byName[resultBoolErr]; ok {
		b.WriteString("declare void @__std_write_file(ptr sret(%" + resultBoolErr + "), ptr, ptr)\n")
		b.WriteString("declare void @__std_mkdir_all(ptr sret(%" + resultBoolErr + "), ptr)\n")
		b.WriteString("declare void @__std_remove(ptr sret(%" + resultBoolErr + "), ptr)\n")
		b.WriteString("declare void @__std_http_serve_text(ptr sret(%" + resultBoolErr + "), ptr, ptr)\n")
		b.WriteString("declare void @__std_http_serve_app(ptr sret(%" + resultBoolErr + "), ptr)\n")
		b.WriteString("declare void @__std_bcrypt_verify(ptr sret(%" + resultBoolErr + "), ptr, ptr)\n")
		b.WriteString("declare void @__std_json_get_bool(ptr sret(%" + resultBoolErr + "), ptr, ptr)\n")
		b.WriteString("declare void @__std_session_init(ptr sret(%" + resultBoolErr + "), ptr)\n")
		b.WriteString("declare void @__std_session_put(ptr sret(%" + resultBoolErr + "), ptr, ptr, ptr, ptr)\n")
		b.WriteString("declare void @__std_session_delete(ptr sret(%" + resultBoolErr + "), ptr, ptr)\n")
		b.WriteString("declare void @__std_open_url(ptr sret(%" + resultBoolErr + "), ptr)\n")
	}
	b.WriteString("declare i8 @__std_exists(ptr)\n")
	b.WriteString("declare i64 @__std_unix_millis()\n")
	b.WriteString("declare void @__std_sleep_ms(i64)\n")
	b.WriteString("declare ptr @__std_now_rfc3339()\n")
	b.WriteString("declare ptr @__std_json_escape(ptr)\n")
	b.WriteString("declare ptr @__std_sha256_hex(ptr)\n")
	b.WriteString("declare ptr @__std_hmac_sha256_hex(ptr, ptr)\n")
	b.WriteString("declare void @__std_jwt_sign_hs256(ptr sret(%" + resultStrErr + "), ptr, ptr, ptr)\n")
	b.WriteString("declare void @__std_jwt_verify_hs256(ptr sret(%" + resultBoolErr + "), ptr, ptr)\n")
	b.WriteString("declare void @__bazic_set_args(i32, ptr)\n")
	return b.String()
}

func resultStructName(okType string, errType string) string {
	return fmt.Sprintf("Result__%s__%s", sanitizeName(okType), sanitizeName(errType))
}

func emitCaseTransform(name string, fn string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("define ptr @%s(ptr %%s) {\n", name))
	b.WriteString("entry:\n")
	b.WriteString("  %len = call i64 @strlen(ptr %s)\n")
	b.WriteString("  %total = add i64 %len, 1\n")
	b.WriteString("  %buf = call ptr @malloc(i64 %total)\n")
	b.WriteString("  br label %loop\n")
	b.WriteString("loop:\n")
	b.WriteString("  %i = phi i64 [ 0, %entry ], [ %next, %body ]\n")
	b.WriteString("  %done = icmp eq i64 %i, %len\n")
	b.WriteString("  br i1 %done, label %end, label %body\n")
	b.WriteString("body:\n")
	b.WriteString("  %srcPtr = getelementptr i8, ptr %s, i64 %i\n")
	b.WriteString("  %ch = load i8, ptr %srcPtr\n")
	b.WriteString("  %ch32 = zext i8 %ch to i32\n")
	b.WriteString(fmt.Sprintf("  %%conv = call i32 @%s(i32 %%ch32)\n", fn))
	b.WriteString("  %out = trunc i32 %conv to i8\n")
	b.WriteString("  %dstPtr = getelementptr i8, ptr %buf, i64 %i\n")
	b.WriteString("  store i8 %out, ptr %dstPtr\n")
	b.WriteString("  %next = add i64 %i, 1\n")
	b.WriteString("  br label %loop\n")
	b.WriteString("end:\n")
	b.WriteString("  %endPtr = getelementptr i8, ptr %buf, i64 %len\n")
	b.WriteString("  store i8 0, ptr %endPtr\n")
	b.WriteString("  ret ptr %buf\n")
	b.WriteString("}\n")
	return b.String()
}

func emitTrimSpace(strs stringPool) string {
	var b strings.Builder
	emptyName, emptyLen := stringGlobalRef(strs, "")
	b.WriteString("define ptr @bazic_trim_space(ptr %s) {\n")
	b.WriteString("entry:\n")
	b.WriteString("  %len = call i64 @strlen(ptr %s)\n")
	b.WriteString("  %start = alloca i64\n")
	b.WriteString("  store i64 0, ptr %start\n")
	b.WriteString("  br label %loop_start\n")
	b.WriteString("loop_start:\n")
	b.WriteString("  %i = load i64, ptr %start\n")
	b.WriteString("  %done = icmp uge i64 %i, %len\n")
	b.WriteString("  br i1 %done, label %allspace, label %check_start\n")
	b.WriteString("check_start:\n")
	b.WriteString("  %ptr = getelementptr i8, ptr %s, i64 %i\n")
	b.WriteString("  %ch = load i8, ptr %ptr\n")
	b.WriteString("  %ch32 = zext i8 %ch to i32\n")
	b.WriteString("  %is = call i32 @isspace(i32 %ch32)\n")
	b.WriteString("  %iss = icmp ne i32 %is, 0\n")
	b.WriteString("  br i1 %iss, label %inc_start, label %start_done\n")
	b.WriteString("inc_start:\n")
	b.WriteString("  %ni = add i64 %i, 1\n")
	b.WriteString("  store i64 %ni, ptr %start\n")
	b.WriteString("  br label %loop_start\n")
	b.WriteString("start_done:\n")
	b.WriteString("  %startVal = load i64, ptr %start\n")
	b.WriteString("  %end = alloca i64\n")
	b.WriteString("  %last = sub i64 %len, 1\n")
	b.WriteString("  store i64 %last, ptr %end\n")
	b.WriteString("  br label %loop_end\n")
	b.WriteString("loop_end:\n")
	b.WriteString("  %j = load i64, ptr %end\n")
	b.WriteString("  %lt = icmp ult i64 %j, %startVal\n")
	b.WriteString("  br i1 %lt, label %allspace, label %check_end\n")
	b.WriteString("check_end:\n")
	b.WriteString("  %ptr2 = getelementptr i8, ptr %s, i64 %j\n")
	b.WriteString("  %ch2 = load i8, ptr %ptr2\n")
	b.WriteString("  %ch32b = zext i8 %ch2 to i32\n")
	b.WriteString("  %is2 = call i32 @isspace(i32 %ch32b)\n")
	b.WriteString("  %iss2 = icmp ne i32 %is2, 0\n")
	b.WriteString("  br i1 %iss2, label %dec_end, label %end_done\n")
	b.WriteString("dec_end:\n")
	b.WriteString("  %nj = sub i64 %j, 1\n")
	b.WriteString("  store i64 %nj, ptr %end\n")
	b.WriteString("  br label %loop_end\n")
	b.WriteString("end_done:\n")
	b.WriteString("  %endVal = load i64, ptr %end\n")
	b.WriteString("  %newLen = sub i64 %endVal, %startVal\n")
	b.WriteString("  %newLen2 = add i64 %newLen, 1\n")
	b.WriteString("  %total = add i64 %newLen2, 1\n")
	b.WriteString("  %buf = call ptr @malloc(i64 %total)\n")
	b.WriteString("  %src = getelementptr i8, ptr %s, i64 %startVal\n")
	b.WriteString("  call ptr @memcpy(ptr %buf, ptr %src, i64 %newLen2)\n")
	b.WriteString("  %endPtr = getelementptr i8, ptr %buf, i64 %newLen2\n")
	b.WriteString("  store i8 0, ptr %endPtr\n")
	b.WriteString("  ret ptr %buf\n")
	b.WriteString("allspace:\n")
	b.WriteString(fmt.Sprintf("  %%empty = %s\n", stringGEP(emptyName, emptyLen)))
	b.WriteString("  ret ptr %empty\n")
	b.WriteString("}\n")
	return b.String()
}

func emitRepeat(strs stringPool) string {
	var b strings.Builder
	emptyName, emptyLen := stringGlobalRef(strs, "")
	b.WriteString("define ptr @bazic_repeat(ptr %s, i64 %count) {\n")
	b.WriteString("entry:\n")
	b.WriteString("  %nonpos = icmp sle i64 %count, 0\n")
	b.WriteString("  br i1 %nonpos, label %repeat_empty, label %check_one\n")
	b.WriteString("repeat_empty:\n")
	b.WriteString(fmt.Sprintf("  %%empty_ptr = %s\n", stringGEP(emptyName, emptyLen)))
	b.WriteString("  ret ptr %empty_ptr\n")
	b.WriteString("check_one:\n")
	b.WriteString("  %one = icmp eq i64 %count, 1\n")
	b.WriteString("  br i1 %one, label %repeat_single, label %cont\n")
	b.WriteString("repeat_single:\n")
	b.WriteString("  ret ptr %s\n")
	b.WriteString("cont:\n")
	b.WriteString("  %len = call i64 @strlen(ptr %s)\n")
	b.WriteString("  %len0 = icmp eq i64 %len, 0\n")
	b.WriteString("  br i1 %len0, label %repeat_empty, label %cont_work\n")
	b.WriteString("cont_work:\n")
	b.WriteString("  %total = mul i64 %len, %count\n")
	b.WriteString("  %alloc = add i64 %total, 1\n")
	b.WriteString("  %buf = call ptr @malloc(i64 %alloc)\n")
	b.WriteString("  br label %loop\n")
	b.WriteString("loop:\n")
	b.WriteString("  %i = phi i64 [ 0, %cont_work ], [ %next, %body ]\n")
	b.WriteString("  %done = icmp eq i64 %i, %count\n")
	b.WriteString("  br i1 %done, label %end, label %body\n")
	b.WriteString("body:\n")
	b.WriteString("  %offset = mul i64 %i, %len\n")
	b.WriteString("  %dst = getelementptr i8, ptr %buf, i64 %offset\n")
	b.WriteString("  call ptr @memcpy(ptr %dst, ptr %s, i64 %len)\n")
	b.WriteString("  %next = add i64 %i, 1\n")
	b.WriteString("  br label %loop\n")
	b.WriteString("end:\n")
	b.WriteString("  %endPtr = getelementptr i8, ptr %buf, i64 %total\n")
	b.WriteString("  store i8 0, ptr %endPtr\n")
	b.WriteString("  ret ptr %buf\n")
	b.WriteString("}\n")
	return b.String()
}

func emitReplace() string {
	var b strings.Builder
	b.WriteString("define ptr @bazic_replace(ptr %s, ptr %old, ptr %new) {\n")
	b.WriteString("entry:\n")
	b.WriteString("  %oldLen = call i64 @strlen(ptr %old)\n")
	b.WriteString("  %zero = icmp eq i64 %oldLen, 0\n")
	b.WriteString("  br i1 %zero, label %retorig, label %count_block\n")
	b.WriteString("retorig:\n")
	b.WriteString("  ret ptr %s\n")
	b.WriteString("count_block:\n")
	b.WriteString("  %count = alloca i64\n")
	b.WriteString("  store i64 0, ptr %count\n")
	b.WriteString("  %cursor = alloca ptr\n")
	b.WriteString("  store ptr %s, ptr %cursor\n")
	b.WriteString("  br label %count_loop\n")
	b.WriteString("count_loop:\n")
	b.WriteString("  %cur = load ptr, ptr %cursor\n")
	b.WriteString("  %found = call ptr @strstr(ptr %cur, ptr %old)\n")
	b.WriteString("  %isnull = icmp eq ptr %found, null\n")
	b.WriteString("  br i1 %isnull, label %count_done, label %count_hit\n")
	b.WriteString("count_hit:\n")
	b.WriteString("  %c = load i64, ptr %count\n")
	b.WriteString("  %c1 = add i64 %c, 1\n")
	b.WriteString("  store i64 %c1, ptr %count\n")
	b.WriteString("  %next = getelementptr i8, ptr %found, i64 %oldLen\n")
	b.WriteString("  store ptr %next, ptr %cursor\n")
	b.WriteString("  br label %count_loop\n")
	b.WriteString("count_done:\n")
	b.WriteString("  %cfinal = load i64, ptr %count\n")
	b.WriteString("  %noccur = icmp eq i64 %cfinal, 0\n")
	b.WriteString("  br i1 %noccur, label %retorig, label %alloc\n")
	b.WriteString("alloc:\n")
	b.WriteString("  %lenS = call i64 @strlen(ptr %s)\n")
	b.WriteString("  %lenN = call i64 @strlen(ptr %new)\n")
	b.WriteString("  %diff = sub i64 %lenN, %oldLen\n")
	b.WriteString("  %extra = mul i64 %diff, %cfinal\n")
	b.WriteString("  %newLen = add i64 %lenS, %extra\n")
	b.WriteString("  %total = add i64 %newLen, 1\n")
	b.WriteString("  %buf = call ptr @malloc(i64 %total)\n")
	b.WriteString("  %src = alloca ptr\n")
	b.WriteString("  %dst = alloca ptr\n")
	b.WriteString("  store ptr %s, ptr %src\n")
	b.WriteString("  store ptr %buf, ptr %dst\n")
	b.WriteString("  br label %loop\n")
	b.WriteString("loop:\n")
	b.WriteString("  %srcv = load ptr, ptr %src\n")
	b.WriteString("  %found2 = call ptr @strstr(ptr %srcv, ptr %old)\n")
	b.WriteString("  %isnull2 = icmp eq ptr %found2, null\n")
	b.WriteString("  br i1 %isnull2, label %copy_tail, label %copy_seg\n")
	b.WriteString("copy_seg:\n")
	b.WriteString("  %srcInt = ptrtoint ptr %srcv to i64\n")
	b.WriteString("  %foundInt = ptrtoint ptr %found2 to i64\n")
	b.WriteString("  %segLen = sub i64 %foundInt, %srcInt\n")
	b.WriteString("  %dstv = load ptr, ptr %dst\n")
	b.WriteString("  call ptr @memcpy(ptr %dstv, ptr %srcv, i64 %segLen)\n")
	b.WriteString("  %dst2 = getelementptr i8, ptr %dstv, i64 %segLen\n")
	b.WriteString("  call ptr @memcpy(ptr %dst2, ptr %new, i64 %lenN)\n")
	b.WriteString("  %dst3 = getelementptr i8, ptr %dst2, i64 %lenN\n")
	b.WriteString("  store ptr %dst3, ptr %dst\n")
	b.WriteString("  %nextSrc = getelementptr i8, ptr %found2, i64 %oldLen\n")
	b.WriteString("  store ptr %nextSrc, ptr %src\n")
	b.WriteString("  br label %loop\n")
	b.WriteString("copy_tail:\n")
	b.WriteString("  %srcv2 = load ptr, ptr %src\n")
	b.WriteString("  %tailLen = call i64 @strlen(ptr %srcv2)\n")
	b.WriteString("  %dstv2 = load ptr, ptr %dst\n")
	b.WriteString("  call ptr @memcpy(ptr %dstv2, ptr %srcv2, i64 %tailLen)\n")
	b.WriteString("  %end = getelementptr i8, ptr %dstv2, i64 %tailLen\n")
	b.WriteString("  store i8 0, ptr %end\n")
	b.WriteString("  ret ptr %buf\n")
	b.WriteString("}\n")
	return b.String()
}

func emitIntToStr(strs stringPool) string {
	var b strings.Builder
	fmtName, fmtLen := stringGlobalRef(strs, "%ld")
	b.WriteString("define ptr @bazic_int_to_str(i64 %v) {\n")
	b.WriteString("entry:\n")
	b.WriteString(fmt.Sprintf("  %%fmt = %s\n", stringGEP(fmtName, fmtLen)))
	b.WriteString("  %len32 = call i32 (ptr, i64, ptr, ...) @snprintf(ptr null, i64 0, ptr %fmt, i64 %v)\n")
	b.WriteString("  %len = sext i32 %len32 to i64\n")
	b.WriteString("  %total = add i64 %len, 1\n")
	b.WriteString("  %buf = call ptr @malloc(i64 %total)\n")
	b.WriteString("  call i32 (ptr, i64, ptr, ...) @snprintf(ptr %buf, i64 %total, ptr %fmt, i64 %v)\n")
	b.WriteString("  ret ptr %buf\n")
	b.WriteString("}\n")
	return b.String()
}

func emitFloatToStr(strs stringPool) string {
	var b strings.Builder
	fmtName, fmtLen := stringGlobalRef(strs, "%g")
	b.WriteString("define ptr @bazic_float_to_str(double %v) {\n")
	b.WriteString("entry:\n")
	b.WriteString(fmt.Sprintf("  %%fmt = %s\n", stringGEP(fmtName, fmtLen)))
	b.WriteString("  %len32 = call i32 (ptr, i64, ptr, ...) @snprintf(ptr null, i64 0, ptr %fmt, double %v)\n")
	b.WriteString("  %len = sext i32 %len32 to i64\n")
	b.WriteString("  %total = add i64 %len, 1\n")
	b.WriteString("  %buf = call ptr @malloc(i64 %total)\n")
	b.WriteString("  call i32 (ptr, i64, ptr, ...) @snprintf(ptr %buf, i64 %total, ptr %fmt, double %v)\n")
	b.WriteString("  ret ptr %buf\n")
	b.WriteString("}\n")
	return b.String()
}

func emitParseInt(structs structPool, strs stringPool) string {
	var b strings.Builder
	resultName := resultStructName("int", "Error")
	emptyName, emptyLen := stringGlobalRef(strs, "")
	errName, errLen := stringGlobalRef(strs, "invalid int")
	b.WriteString(fmt.Sprintf("define %%%s @bazic_parse_int(ptr %%s) {\n", resultName))
	b.WriteString("entry:\n")
	b.WriteString("  %endptr = alloca ptr\n")
	b.WriteString("  %val = call i64 @strtol(ptr %s, ptr %endptr, i32 10)\n")
	b.WriteString("  %end = load ptr, ptr %endptr\n")
	b.WriteString("  %same = icmp eq ptr %end, %s\n")
	b.WriteString("  br i1 %same, label %fail, label %checktail\n")
	b.WriteString("checktail:\n")
	b.WriteString("  br label %skip_ws\n")
	b.WriteString("skip_ws:\n")
	b.WriteString("  %cur = load ptr, ptr %endptr\n")
	b.WriteString("  %ch = load i8, ptr %cur\n")
	b.WriteString("  %ch32 = zext i8 %ch to i32\n")
	b.WriteString("  %isws = call i32 @isspace(i32 %ch32)\n")
	b.WriteString("  %iswsb = icmp ne i32 %isws, 0\n")
	b.WriteString("  br i1 %iswsb, label %skip_inc, label %check_end\n")
	b.WriteString("skip_inc:\n")
	b.WriteString("  %next = getelementptr i8, ptr %cur, i64 1\n")
	b.WriteString("  store ptr %next, ptr %endptr\n")
	b.WriteString("  br label %skip_ws\n")
	b.WriteString("check_end:\n")
	b.WriteString("  %end2 = load ptr, ptr %endptr\n")
	b.WriteString("  %ch2 = load i8, ptr %end2\n")
	b.WriteString("  %ch2_32 = zext i8 %ch2 to i32\n")
	b.WriteString("  %iszero = icmp eq i32 %ch2_32, 0\n")
	b.WriteString("  br i1 %iszero, label %ok, label %fail\n")
	b.WriteString("ok:\n")
	b.WriteString(fmt.Sprintf("  %%okmsg = %s\n", stringGEP(emptyName, emptyLen)))
	b.WriteString("  %okerr = insertvalue %Error undef, ptr %okmsg, 0\n")
	b.WriteString("  %r0 = insertvalue %" + resultName + " undef, i8 1, 0\n")
	b.WriteString("  %r1 = insertvalue %" + resultName + " %r0, i64 %val, 1\n")
	b.WriteString("  %r2 = insertvalue %" + resultName + " %r1, %Error %okerr, 2\n")
	b.WriteString("  ret %" + resultName + " %r2\n")
	b.WriteString("fail:\n")
	b.WriteString(fmt.Sprintf("  %%errmsg = %s\n", stringGEP(errName, errLen)))
	b.WriteString("  %errv = insertvalue %Error undef, ptr %errmsg, 0\n")
	b.WriteString("  %f0 = insertvalue %" + resultName + " undef, i8 0, 0\n")
	b.WriteString("  %f1 = insertvalue %" + resultName + " %f0, i64 0, 1\n")
	b.WriteString("  %f2 = insertvalue %" + resultName + " %f1, %Error %errv, 2\n")
	b.WriteString("  ret %" + resultName + " %f2\n")
	b.WriteString("}\n")
	return b.String()
}

func emitParseFloat(structs structPool, strs stringPool) string {
	var b strings.Builder
	resultName := resultStructName("float", "Error")
	emptyName, emptyLen := stringGlobalRef(strs, "")
	errName, errLen := stringGlobalRef(strs, "invalid float")
	b.WriteString(fmt.Sprintf("define %%%s @bazic_parse_float(ptr %%s) {\n", resultName))
	b.WriteString("entry:\n")
	b.WriteString("  %endptr = alloca ptr\n")
	b.WriteString("  %val = call double @strtod(ptr %s, ptr %endptr)\n")
	b.WriteString("  %end = load ptr, ptr %endptr\n")
	b.WriteString("  %same = icmp eq ptr %end, %s\n")
	b.WriteString("  br i1 %same, label %fail, label %checktail\n")
	b.WriteString("checktail:\n")
	b.WriteString("  br label %skip_ws\n")
	b.WriteString("skip_ws:\n")
	b.WriteString("  %cur = load ptr, ptr %endptr\n")
	b.WriteString("  %ch = load i8, ptr %cur\n")
	b.WriteString("  %ch32 = zext i8 %ch to i32\n")
	b.WriteString("  %isws = call i32 @isspace(i32 %ch32)\n")
	b.WriteString("  %iswsb = icmp ne i32 %isws, 0\n")
	b.WriteString("  br i1 %iswsb, label %skip_inc, label %check_end\n")
	b.WriteString("skip_inc:\n")
	b.WriteString("  %next = getelementptr i8, ptr %cur, i64 1\n")
	b.WriteString("  store ptr %next, ptr %endptr\n")
	b.WriteString("  br label %skip_ws\n")
	b.WriteString("check_end:\n")
	b.WriteString("  %end2 = load ptr, ptr %endptr\n")
	b.WriteString("  %ch2 = load i8, ptr %end2\n")
	b.WriteString("  %ch2_32 = zext i8 %ch2 to i32\n")
	b.WriteString("  %iszero = icmp eq i32 %ch2_32, 0\n")
	b.WriteString("  br i1 %iszero, label %ok, label %fail\n")
	b.WriteString("ok:\n")
	b.WriteString(fmt.Sprintf("  %%okmsg = %s\n", stringGEP(emptyName, emptyLen)))
	b.WriteString("  %okerr = insertvalue %Error undef, ptr %okmsg, 0\n")
	b.WriteString("  %r0 = insertvalue %" + resultName + " undef, i8 1, 0\n")
	b.WriteString("  %r1 = insertvalue %" + resultName + " %r0, double %val, 1\n")
	b.WriteString("  %r2 = insertvalue %" + resultName + " %r1, %Error %okerr, 2\n")
	b.WriteString("  ret %" + resultName + " %r2\n")
	b.WriteString("fail:\n")
	b.WriteString(fmt.Sprintf("  %%errmsg = %s\n", stringGEP(errName, errLen)))
	b.WriteString("  %errv = insertvalue %Error undef, ptr %errmsg, 0\n")
	b.WriteString("  %f0 = insertvalue %" + resultName + " undef, i8 0, 0\n")
	b.WriteString("  %f1 = insertvalue %" + resultName + " %f0, double 0.0, 1\n")
	b.WriteString("  %f2 = insertvalue %" + resultName + " %f1, %Error %errv, 2\n")
	b.WriteString("  ret %" + resultName + " %f2\n")
	b.WriteString("}\n")
	return b.String()
}
func emitMain(fn *ast.FuncDecl, funcs map[string]llvmFuncSig, globals map[string]globalSlot, enums enumInfo, structs structPool, ifaces interfacePool, strs stringPool, hasGlobals bool) (string, error) {
	var b strings.Builder
	b.WriteString("define i32 @main(i32 %argc, ptr %argv) {\n")
	b.WriteString("entry:\n")
	b.WriteString("  call void @__bazic_set_args(i32 %argc, ptr %argv)\n")
	if hasGlobals {
		b.WriteString("  call void @__bazic_init_globals()\n")
	}
	ctx := newFuncCtx(enums, structs, ifaces, strs, true, globals)
	ctx.returnType = ast.TypeVoid
	if err := emitFunctionBody(&b, ctx, fn, funcs); err != nil {
		return "", err
	}
	if !ctx.terminated {
		b.WriteString("  ret i32 0\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}

func emitFunction(fn *ast.FuncDecl, funcs map[string]llvmFuncSig, globals map[string]globalSlot, enums enumInfo, structs structPool, ifaces interfacePool, strs stringPool) (string, error) {
	retType, ok := mapLLVMType(fn.ReturnType, enums, structs, ifaces)
	if !ok {
		return "", fmt.Errorf("llvm backend: unsupported return type '%s' for '%s'", fn.ReturnType, fn.Name)
	}
	paramParts := make([]string, 0, len(fn.Params))
	paramTypes := map[string]ast.Type{}
	for _, p := range fn.Params {
		pt, ok := mapLLVMType(p.Type, enums, structs, ifaces)
		if !ok {
			return "", fmt.Errorf("llvm backend: unsupported param type '%s' for '%s.%s'", p.Type, fn.Name, p.Name)
		}
		paramParts = append(paramParts, fmt.Sprintf("%s %%%s", pt, p.Name))
		paramTypes[p.Name] = p.Type
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("define %s @%s(%s) {\n", retType, fn.Name, strings.Join(paramParts, ", ")))
	b.WriteString("entry:\n")
	ctx := newFuncCtx(enums, structs, ifaces, strs, false, globals)
	ctx.returnType = fn.ReturnType
	for name, typ := range paramTypes {
		ptr := ctx.alloca(&b, typ)
		if ptr == "" {
			return "", fmt.Errorf("llvm backend: unsupported param type '%s' for '%s.%s'", typ, fn.Name, name)
		}
		pt, _ := mapLLVMType(typ, enums, structs, ifaces)
		b.WriteString(fmt.Sprintf("  store %s %%%s, ptr %s\n", pt, name, ptr))
		ctx.vars[name] = varSlot{ptr: ptr, typ: typ}
	}
	if err := emitFunctionBody(&b, ctx, fn, funcs); err != nil {
		return "", err
	}
	if !ctx.terminated {
		if fn.ReturnType == ast.TypeVoid {
			b.WriteString("  ret void\n")
		} else {
			b.WriteString(fmt.Sprintf("  ; missing return in function %s, defaulting\n", fn.Name))
			b.WriteString(fmt.Sprintf("  ret %s %s\n", retType, defaultLLVMValue(fn.ReturnType, enums, structs, ifaces)))
		}
	}
	b.WriteString("}\n")
	return b.String(), nil
}

type irCtx struct {
	tmp int
	lbl int
}

type llvmFuncSig struct {
	Params []ast.Type
	Ret    ast.Type
}

func newIRCtx() *irCtx { return &irCtx{} }

func (c *irCtx) nextTmp() string {
	c.tmp++
	return fmt.Sprintf("%%t%d", c.tmp)
}

func (c *irCtx) nextLabel(prefix string) string {
	c.lbl++
	return fmt.Sprintf("%s%d", prefix, c.lbl)
}

type varSlot struct {
	ptr string
	typ ast.Type
}

type funcCtx struct {
	ir         *irCtx
	vars       map[string]varSlot
	globals    map[string]globalSlot
	returnType ast.Type
	enums      enumInfo
	structs    structPool
	ifaces     interfacePool
	strs       stringPool
	terminated bool
	isMain     bool
}

func newFuncCtx(enums enumInfo, structs structPool, ifaces interfacePool, strs stringPool, isMain bool, globals map[string]globalSlot) *funcCtx {
	return &funcCtx{
		ir:      newIRCtx(),
		vars:    map[string]varSlot{},
		globals: globals,
		enums:   enums,
		structs: structs,
		ifaces:  ifaces,
		strs:    strs,
		isMain:  isMain,
	}
}

func (c *funcCtx) alloca(b *strings.Builder, t ast.Type) string {
	llvmType, ok := mapLLVMType(t, c.enums, c.structs, c.ifaces)
	if !ok {
		return ""
	}
	ptr := c.ir.nextTmp()
	b.WriteString(fmt.Sprintf("  %s = alloca %s\n", ptr, llvmType))
	return ptr
}

func emitFunctionBody(b *strings.Builder, ctx *funcCtx, fn *ast.FuncDecl, funcs map[string]llvmFuncSig) error {
	if fn.Body == nil {
		return nil
	}
	for _, st := range fn.Body.Stmts {
		if ctx.terminated {
			return nil
		}
		ok := emitStmt(b, ctx, st, funcs)
		if !ok {
			return fmt.Errorf("llvm backend: unsupported statement in function '%s': %T", fn.Name, st)
		}
	}
	return nil
}

func emitStmt(b *strings.Builder, ctx *funcCtx, s ast.Stmt, funcs map[string]llvmFuncSig) bool {
	switch st := s.(type) {
	case *ast.LetStmt:
		ptr := ctx.alloca(b, st.Type)
		if ptr == "" {
			return false
		}
		code, value, t, ok := emitExpr(ctx, st.Init, funcs)
		if !ok {
			return false
		}
		if st.Type == ast.TypeAny && t != ast.TypeAny {
			boxCode, boxed, ok := boxToAny(ctx, value, t)
			if !ok {
				return false
			}
			code += boxCode
			value = boxed
			t = ast.TypeAny
		}
		if t != st.Type {
			return false
		}
		b.WriteString(code)
		llvmType, _ := mapLLVMType(st.Type, ctx.enums, ctx.structs, ctx.ifaces)
		b.WriteString(fmt.Sprintf("  store %s %s, ptr %s\n", llvmType, value, ptr))
		ctx.vars[st.Name] = varSlot{ptr: ptr, typ: st.Type}
		return true
	case *ast.AssignStmt:
		ptrCode, ptr, targetType, ok := emitAssignTargetPtr(ctx, st.Target)
		if !ok {
			return false
		}
		code, value, t, ok := emitExpr(ctx, st.Value, funcs)
		if !ok {
			return false
		}
		if targetType == ast.TypeAny && t != ast.TypeAny {
			boxCode, boxed, ok := boxToAny(ctx, value, t)
			if !ok {
				return false
			}
			code += boxCode
			value = boxed
			t = ast.TypeAny
		}
		if t != targetType {
			return false
		}
		b.WriteString(ptrCode)
		b.WriteString(code)
		llvmType, _ := mapLLVMType(targetType, ctx.enums, ctx.structs, ctx.ifaces)
		b.WriteString(fmt.Sprintf("  store %s %s, ptr %s\n", llvmType, value, ptr))
		return true
	case *ast.ExprStmt:
		code, _, _, ok := emitExpr(ctx, st.Expr, funcs)
		if !ok {
			return false
		}
		b.WriteString(code)
		return true
	case *ast.ReturnStmt:
		if ctx.isMain {
			if st.Value != nil {
				return false
			}
			b.WriteString("  ret i32 0\n")
			ctx.terminated = true
			return true
		}
		if st.Value == nil {
			b.WriteString("  ret void\n")
			ctx.terminated = true
			return true
		}
		code, value, t, ok := emitExpr(ctx, st.Value, funcs)
		if !ok {
			return false
		}
		b.WriteString(code)
		if ctx.returnType == ast.TypeAny && t != ast.TypeAny {
			boxCode, boxed, ok := boxToAny(ctx, value, t)
			if !ok {
				return false
			}
			b.WriteString(boxCode)
			value = boxed
			t = ast.TypeAny
		}
		if t != ctx.returnType {
			return false
		}
		retType, ok := mapLLVMType(ctx.returnType, ctx.enums, ctx.structs, ctx.ifaces)
		if !ok {
			return false
		}
		b.WriteString(fmt.Sprintf("  ret %s %s\n", retType, value))
		ctx.terminated = true
		return true
	case *ast.IfStmt:
		code, condVal, condType, ok := emitExpr(ctx, st.Cond, funcs)
		if !ok || condType != ast.TypeBool {
			return false
		}
		condCode, cond := boolToI1(ctx, condVal)
		thenLabel := ctx.ir.nextLabel("then")
		elseLabel := ctx.ir.nextLabel("else")
		mergeLabel := ctx.ir.nextLabel("ifend")
		emitIsolated := func(block *ast.BlockStmt) bool {
			prev := ctx.terminated
			ctx.terminated = false
			term := emitBlock(b, ctx, block, funcs)
			ctx.terminated = prev
			return term
		}
		b.WriteString(code)
		b.WriteString(condCode)
		if st.Else == nil {
			b.WriteString(fmt.Sprintf("  br i1 %s, label %%%s, label %%%s\n", cond, thenLabel, mergeLabel))
			b.WriteString(fmt.Sprintf("%s:\n", thenLabel))
			thenTerm := emitIsolated(st.Then)
			if !thenTerm {
				b.WriteString(fmt.Sprintf("  br label %%%s\n", mergeLabel))
			}
			b.WriteString(fmt.Sprintf("%s:\n", mergeLabel))
			ctx.terminated = false
			return true
		}
		b.WriteString(fmt.Sprintf("  br i1 %s, label %%%s, label %%%s\n", cond, thenLabel, elseLabel))
		b.WriteString(fmt.Sprintf("%s:\n", thenLabel))
		thenTerm := emitIsolated(st.Then)
		if !thenTerm {
			b.WriteString(fmt.Sprintf("  br label %%%s\n", mergeLabel))
		}
		b.WriteString(fmt.Sprintf("%s:\n", elseLabel))
		elseTerm := emitIsolated(st.Else)
		if !elseTerm {
			b.WriteString(fmt.Sprintf("  br label %%%s\n", mergeLabel))
		}
		if !(thenTerm && elseTerm) {
			b.WriteString(fmt.Sprintf("%s:\n", mergeLabel))
		}
		ctx.terminated = thenTerm && elseTerm
		return true
	case *ast.WhileStmt:
		condLabel := ctx.ir.nextLabel("while_cond")
		bodyLabel := ctx.ir.nextLabel("while_body")
		endLabel := ctx.ir.nextLabel("while_end")
		b.WriteString(fmt.Sprintf("  br label %%%s\n", condLabel))
		b.WriteString(fmt.Sprintf("%s:\n", condLabel))
		code, condVal, condType, ok := emitExpr(ctx, st.Cond, funcs)
		if !ok || condType != ast.TypeBool {
			return false
		}
		condCode, cond := boolToI1(ctx, condVal)
		b.WriteString(code)
		b.WriteString(condCode)
		b.WriteString(fmt.Sprintf("  br i1 %s, label %%%s, label %%%s\n", cond, bodyLabel, endLabel))
		b.WriteString(fmt.Sprintf("%s:\n", bodyLabel))
		bodyTerm := emitBlock(b, ctx, st.Body, funcs)
		if !bodyTerm {
			b.WriteString(fmt.Sprintf("  br label %%%s\n", condLabel))
		}
		b.WriteString(fmt.Sprintf("%s:\n", endLabel))
		return true
	case *ast.MatchStmt:
		code, subjVal, subjType, ok := emitExpr(ctx, st.Subject, funcs)
		if !ok {
			return false
		}
		if _, ok := ctx.enums.enumTypes[string(subjType)]; !ok {
			return false
		}
		llvmType, _ := mapLLVMType(subjType, ctx.enums, ctx.structs, ctx.ifaces)
		endLabel := ctx.ir.nextLabel("match_end")
		grouped := groupMatchArmsLLVM(st.Arms)
		b.WriteString(code)
		b.WriteString(fmt.Sprintf("  switch %s %s, label %%%s [\n", llvmType, subjVal, endLabel))
		caseLabels := make([]string, 0, len(grouped))
		for _, g := range grouped {
			lbl := ctx.ir.nextLabel("match_arm")
			caseLabels = append(caseLabels, lbl)
			idx, ok := ctx.enums.variantIndex[g.Variant]
			if !ok {
				return false
			}
			b.WriteString(fmt.Sprintf("    %s %d, label %%%s\n", llvmType, idx, lbl))
		}
		b.WriteString("  ]\n")
		for i, g := range grouped {
			b.WriteString(fmt.Sprintf("%s:\n", caseLabels[i]))
			if !emitGuardedMatchStmt(b, ctx, g.Arms, funcs, endLabel) {
				return false
			}
		}
		b.WriteString(fmt.Sprintf("%s:\n", endLabel))
		return true
	default:
		return false
	}
}

func emitBlock(b *strings.Builder, ctx *funcCtx, blk *ast.BlockStmt, funcs map[string]llvmFuncSig) bool {
	if blk == nil {
		return false
	}
	prevVars := ctx.vars
	if prevVars != nil {
		ctx.vars = make(map[string]varSlot, len(prevVars))
		for k, v := range prevVars {
			ctx.vars[k] = v
		}
	} else {
		ctx.vars = map[string]varSlot{}
	}
	defer func() {
		ctx.vars = prevVars
	}()
	for _, st := range blk.Stmts {
		if ctx.terminated {
			return true
		}
		ok := emitStmt(b, ctx, st, funcs)
		if !ok {
			return true
		}
	}
	return ctx.terminated
}

type matchGroupStmt struct {
	Variant string
	Arms    []ast.MatchArm
}

type matchGroupExpr struct {
	Variant string
	Arms    []ast.MatchExprArm
}

func groupMatchArmsLLVM(arms []ast.MatchArm) []matchGroupStmt {
	order := []string{}
	by := map[string][]ast.MatchArm{}
	for _, arm := range arms {
		if _, ok := by[arm.Variant]; !ok {
			order = append(order, arm.Variant)
		}
		by[arm.Variant] = append(by[arm.Variant], arm)
	}
	out := make([]matchGroupStmt, 0, len(order))
	for _, v := range order {
		out = append(out, matchGroupStmt{Variant: v, Arms: by[v]})
	}
	return out
}

func groupMatchExprArmsLLVM(arms []ast.MatchExprArm) []matchGroupExpr {
	order := []string{}
	by := map[string][]ast.MatchExprArm{}
	for _, arm := range arms {
		if _, ok := by[arm.Variant]; !ok {
			order = append(order, arm.Variant)
		}
		by[arm.Variant] = append(by[arm.Variant], arm)
	}
	out := make([]matchGroupExpr, 0, len(order))
	for _, v := range order {
		out = append(out, matchGroupExpr{Variant: v, Arms: by[v]})
	}
	return out
}

func emitGuardedMatchStmt(b *strings.Builder, ctx *funcCtx, arms []ast.MatchArm, funcs map[string]llvmFuncSig, endLabel string) bool {
	unguarded := -1
	for i, arm := range arms {
		if arm.Guard == nil {
			unguarded = i
			break
		}
	}
	nextLabel := ""
	for i, arm := range arms {
		if arm.Guard == nil {
			continue
		}
		condCode, condVal, condType, ok := emitExpr(ctx, arm.Guard, funcs)
		if !ok || condType != ast.TypeBool {
			return false
		}
		condI1Code, condI1 := boolToI1(ctx, condVal)
		thenLabel := ctx.ir.nextLabel("match_guard_then")
		if i == len(arms)-1 && unguarded == -1 {
			nextLabel = endLabel
		} else {
			nextLabel = ctx.ir.nextLabel("match_guard_next")
		}
		b.WriteString(condCode)
		b.WriteString(condI1Code)
		b.WriteString(fmt.Sprintf("  br i1 %s, label %%%s, label %%%s\n", condI1, thenLabel, nextLabel))
		b.WriteString(fmt.Sprintf("%s:\n", thenLabel))
		term := emitBlock(b, ctx, arm.Body, funcs)
		if !term {
			b.WriteString(fmt.Sprintf("  br label %%%s\n", endLabel))
		}
		b.WriteString(fmt.Sprintf("%s:\n", nextLabel))
	}
	if unguarded >= 0 {
		term := emitBlock(b, ctx, arms[unguarded].Body, funcs)
		if !term {
			b.WriteString(fmt.Sprintf("  br label %%%s\n", endLabel))
		}
	}
	return true
}

func emitGuardedMatchExpr(b *strings.Builder, ctx *funcCtx, arms []ast.MatchExprArm, funcs map[string]llvmFuncSig, mergeLabel string, resolved ast.Type, caseLabel string) ([]string, bool) {
	phiVals := []string{}
	unguarded := -1
	for i, arm := range arms {
		if arm.Guard == nil {
			unguarded = i
			break
		}
	}
	hasGuard := false
	for _, arm := range arms {
		if arm.Guard != nil {
			hasGuard = true
			break
		}
	}
	if !hasGuard && unguarded >= 0 {
		valCode, val, valType, ok := emitExpr(ctx, arms[unguarded].Value, funcs)
		if !ok || valType != resolved {
			return nil, false
		}
		b.WriteString(valCode)
		b.WriteString(fmt.Sprintf("  br label %%%s\n", mergeLabel))
		phiVals = append(phiVals, fmt.Sprintf("[ %s, %%%s ]", val, caseLabel))
		return phiVals, true
	}
	unguardedLabel := ""
	unguardedLabelEmitted := false
	if unguarded >= 0 {
		unguardedLabel = ctx.ir.nextLabel("match_unguarded")
	}
	guarded := []int{}
	for i, arm := range arms {
		if arm.Guard != nil {
			guarded = append(guarded, i)
		}
	}
	nextLabel := ""
	for gi, idx := range guarded {
		arm := arms[idx]
		condCode, condVal, condType, ok := emitExpr(ctx, arm.Guard, funcs)
		if !ok || condType != ast.TypeBool {
			return nil, false
		}
		condI1Code, condI1 := boolToI1(ctx, condVal)
		thenLabel := ctx.ir.nextLabel("match_guard_then")
		if gi == len(guarded)-1 {
			if unguarded >= 0 {
				nextLabel = unguardedLabel
			} else {
				nextLabel = mergeLabel
			}
		} else {
			nextLabel = ctx.ir.nextLabel("match_guard_next")
		}
		b.WriteString(condCode)
		b.WriteString(condI1Code)
		b.WriteString(fmt.Sprintf("  br i1 %s, label %%%s, label %%%s\n", condI1, thenLabel, nextLabel))
		b.WriteString(fmt.Sprintf("%s:\n", thenLabel))
		valCode, val, valType, ok := emitExpr(ctx, arm.Value, funcs)
		if !ok || valType != resolved {
			return nil, false
		}
		b.WriteString(valCode)
		b.WriteString(fmt.Sprintf("  br label %%%s\n", mergeLabel))
		phiVals = append(phiVals, fmt.Sprintf("[ %s, %%%s ]", val, thenLabel))
		b.WriteString(fmt.Sprintf("%s:\n", nextLabel))
		if nextLabel == unguardedLabel {
			unguardedLabelEmitted = true
		}
	}
	if unguarded >= 0 {
		if unguardedLabel != "" && !unguardedLabelEmitted {
			b.WriteString(fmt.Sprintf("%s:\n", unguardedLabel))
		}
		valCode, val, valType, ok := emitExpr(ctx, arms[unguarded].Value, funcs)
		if !ok || valType != resolved {
			return nil, false
		}
		b.WriteString(valCode)
		b.WriteString(fmt.Sprintf("  br label %%%s\n", mergeLabel))
		phiVals = append(phiVals, fmt.Sprintf("[ %s, %%%s ]", val, unguardedLabel))
	}
	return phiVals, true
}

func emitAssignTargetPtr(ctx *funcCtx, target ast.Expr) (string, string, ast.Type, bool) {
	switch t := target.(type) {
	case *ast.IdentExpr:
		if slot, ok := ctx.vars[t.Name]; ok {
			return "", slot.ptr, slot.typ, true
		}
		if g, ok := ctx.globals[t.Name]; ok {
			return "", g.ptr, g.typ, true
		}
		return "", "", ast.TypeInvalid, false
	case *ast.FieldAccessExpr:
		fields := []string{}
		cur := target
		for {
			fa, ok := cur.(*ast.FieldAccessExpr)
			if !ok {
				break
			}
			fields = append(fields, fa.Field)
			cur = fa.Object
		}
		for i, j := 0, len(fields)-1; i < j; i, j = i+1, j-1 {
			fields[i], fields[j] = fields[j], fields[i]
		}
		baseIdent, ok := cur.(*ast.IdentExpr)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		var ptr string
		var typ ast.Type
		if slot, ok := ctx.vars[baseIdent.Name]; ok {
			ptr = slot.ptr
			typ = slot.typ
		} else if g, ok := ctx.globals[baseIdent.Name]; ok {
			ptr = g.ptr
			typ = g.typ
		} else {
			return "", "", ast.TypeInvalid, false
		}
		code := ""
		currentPtr := ptr
		currentType := typ
		for _, field := range fields {
			base, _ := splitGenericBase(string(currentType))
			if base == "" {
				base = string(currentType)
			}
			info, ok := ctx.structs.byName[base]
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			idx, ok := info.FieldIndex[field]
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			tmp := ctx.ir.nextTmp()
			code += fmt.Sprintf("  %s = getelementptr inbounds %%%s, ptr %s, i32 0, i32 %d\n", tmp, base, currentPtr, idx)
			currentPtr = tmp
			currentType = info.Fields[idx].Type
		}
		return code, currentPtr, currentType, true
	default:
		return "", "", ast.TypeInvalid, false
	}
}

func emitExpr(ctx *funcCtx, e ast.Expr, funcs map[string]llvmFuncSig) (string, string, ast.Type, bool) {
	switch ex := e.(type) {
	case *ast.IntExpr:
		return "", strconv.FormatInt(ex.Value, 10), ast.TypeInt, true
	case *ast.FloatExpr:
		return "", strconv.FormatFloat(ex.Value, 'f', -1, 64), ast.TypeFloat, true
	case *ast.BoolExpr:
		if ex.Value {
			return "", "1", ast.TypeBool, true
		}
		return "", "0", ast.TypeBool, true
	case *ast.StringExpr:
		code, ptr, ok := stringPtr(ctx, ex.Value)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		return code, ptr, ast.TypeString, true
	case *ast.StructLitExpr:
		base, ok := splitGenericBase(ex.TypeName)
		if !ok {
			base = ex.TypeName
		}
		info, ok := ctx.structs.byName[base]
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		structType := "%" + base
		code := ""
		curr := "undef"
		for _, f := range ex.Fields {
			idx, ok := info.FieldIndex[f.Name]
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			fcode, fval, ftype, ok := emitExpr(ctx, f.Value, funcs)
			if !ok || ftype != info.Fields[idx].Type {
				return "", "", ast.TypeInvalid, false
			}
			code += fcode
			next := ctx.ir.nextTmp()
			llvmFieldType, ok := mapLLVMType(info.Fields[idx].Type, ctx.enums, ctx.structs, ctx.ifaces)
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			code += fmt.Sprintf("  %s = insertvalue %s %s, %s %s, %d\n", next, structType, curr, llvmFieldType, fval, idx)
			curr = next
		}
		return code, curr, ast.Type(base), true
	case *ast.FieldAccessExpr:
		objCode, objVal, objType, ok := emitExpr(ctx, ex.Object, funcs)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		base, _ := splitGenericBase(string(objType))
		if base == "" {
			base = string(objType)
		}
		info, ok := ctx.structs.byName[base]
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		idx, ok := info.FieldIndex[ex.Field]
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		tmp := ctx.ir.nextTmp()
		code := objCode + fmt.Sprintf("  %s = extractvalue %%%s %s, %d\n", tmp, base, objVal, idx)
		return code, tmp, info.Fields[idx].Type, true
	case *ast.IdentExpr:
		if slot, ok := ctx.vars[ex.Name]; ok {
			llvmType, ok := mapLLVMType(slot.typ, ctx.enums, ctx.structs, ctx.ifaces)
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			tmp := ctx.ir.nextTmp()
			code := fmt.Sprintf("  %s = load %s, ptr %s\n", tmp, llvmType, slot.ptr)
			return code, tmp, slot.typ, true
		}
		if g, ok := ctx.globals[ex.Name]; ok {
			llvmType, ok := mapLLVMType(g.typ, ctx.enums, ctx.structs, ctx.ifaces)
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			tmp := ctx.ir.nextTmp()
			code := fmt.Sprintf("  %s = load %s, ptr %s\n", tmp, llvmType, g.ptr)
			return code, tmp, g.typ, true
		}
		if idx, ok := ctx.enums.variantIndex[ex.Name]; ok {
			enumName := ctx.enums.variantType[ex.Name]
			if enumName == "" {
				return "", "", ast.TypeInvalid, false
			}
			return "", strconv.Itoa(idx), ast.Type(enumName), true
		}
		return "", "", ast.TypeInvalid, false
	case *ast.UnaryExpr:
		code, value, t, ok := emitExpr(ctx, ex.Right, funcs)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		switch ex.Op {
		case "-":
			if t == ast.TypeInt {
				tmp := ctx.ir.nextTmp()
				return code + fmt.Sprintf("  %s = sub i64 0, %s\n", tmp, value), tmp, ast.TypeInt, true
			}
			if t == ast.TypeFloat {
				tmp := ctx.ir.nextTmp()
				return code + fmt.Sprintf("  %s = fsub double 0.0, %s\n", tmp, value), tmp, ast.TypeFloat, true
			}
		case "!":
			if t == ast.TypeBool {
				tmp := ctx.ir.nextTmp()
				code += fmt.Sprintf("  %s = icmp eq i8 %s, 0\n", tmp, value)
				zextCode, out := boolToI8(ctx, tmp)
				return code + zextCode, out, ast.TypeBool, true
			}
		}
		return "", "", ast.TypeInvalid, false
	case *ast.BinaryExpr:
		lc, lv, lt, ok := emitExpr(ctx, ex.Left, funcs)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		rc, rv, rt, ok := emitExpr(ctx, ex.Right, funcs)
		if !ok || lt != rt {
			return "", "", ast.TypeInvalid, false
		}
		code := lc + rc
		if lt == ast.TypeInt {
			if op, ok := mapIntArithOp(ex.Op); ok {
				tmp := ctx.ir.nextTmp()
				return code + fmt.Sprintf("  %s = %s i64 %s, %s\n", tmp, op, lv, rv), tmp, ast.TypeInt, true
			}
			if op, ok := mapIntCmpOp(ex.Op); ok {
				tmp := ctx.ir.nextTmp()
				code += fmt.Sprintf("  %s = icmp %s i64 %s, %s\n", tmp, op, lv, rv)
				zextCode, out := boolToI8(ctx, tmp)
				return code + zextCode, out, ast.TypeBool, true
			}
			return "", "", ast.TypeInvalid, false
		}
		if lt == ast.TypeFloat {
			if op, ok := mapFloatArithOp(ex.Op); ok {
				tmp := ctx.ir.nextTmp()
				return code + fmt.Sprintf("  %s = %s double %s, %s\n", tmp, op, lv, rv), tmp, ast.TypeFloat, true
			}
			if op, ok := mapFloatCmpOp(ex.Op); ok {
				tmp := ctx.ir.nextTmp()
				code += fmt.Sprintf("  %s = fcmp %s double %s, %s\n", tmp, op, lv, rv)
				zextCode, out := boolToI8(ctx, tmp)
				return code + zextCode, out, ast.TypeBool, true
			}
			return "", "", ast.TypeInvalid, false
		}
		if _, ok := ctx.enums.enumTypes[string(lt)]; ok {
			if ex.Op == "==" || ex.Op == "!=" {
				tmp := ctx.ir.nextTmp()
				cmp := "eq"
				if ex.Op == "!=" {
					cmp = "ne"
				}
				code += fmt.Sprintf("  %s = icmp %s i64 %s, %s\n", tmp, cmp, lv, rv)
				zextCode, out := boolToI8(ctx, tmp)
				return code + zextCode, out, ast.TypeBool, true
			}
		}
		if lt == ast.TypeBool {
			if ex.Op == "&&" || ex.Op == "||" {
				tmp := ctx.ir.nextTmp()
				irOp := "and"
				if ex.Op == "||" {
					irOp = "or"
				}
				return code + fmt.Sprintf("  %s = %s i8 %s, %s\n", tmp, irOp, lv, rv), tmp, ast.TypeBool, true
			}
			if ex.Op == "==" || ex.Op == "!=" {
				tmp := ctx.ir.nextTmp()
				cmp := "eq"
				if ex.Op == "!=" {
					cmp = "ne"
				}
				code += fmt.Sprintf("  %s = icmp %s i8 %s, %s\n", tmp, cmp, lv, rv)
				zextCode, out := boolToI8(ctx, tmp)
				return code + zextCode, out, ast.TypeBool, true
			}
			return "", "", ast.TypeInvalid, false
		}
		if lt == ast.TypeString {
			if ex.Op == "+" {
				tmp := ctx.ir.nextTmp()
				code += fmt.Sprintf("  %s = call ptr @bazic_str_concat(ptr %s, ptr %s)\n", tmp, lv, rv)
				return code, tmp, ast.TypeString, true
			}
			if ex.Op == "==" || ex.Op == "!=" || ex.Op == "<" || ex.Op == "<=" || ex.Op == ">" || ex.Op == ">=" {
				cmpTmp := ctx.ir.nextTmp()
				code += fmt.Sprintf("  %s = call i32 @bazic_str_cmp(ptr %s, ptr %s)\n", cmpTmp, lv, rv)
				tmp := ctx.ir.nextTmp()
				cmp := "eq"
				switch ex.Op {
				case "!=":
					cmp = "ne"
				case "<":
					cmp = "slt"
				case "<=":
					cmp = "sle"
				case ">":
					cmp = "sgt"
				case ">=":
					cmp = "sge"
				}
				code += fmt.Sprintf("  %s = icmp %s i32 %s, 0\n", tmp, cmp, cmpTmp)
				zextCode, out := boolToI8(ctx, tmp)
				return code + zextCode, out, ast.TypeBool, true
			}
		}
		if lt == ast.TypeAny && rt == ast.TypeAny && (ex.Op == "==" || ex.Op == "!=") {
			tmp := ctx.ir.nextTmp()
			code += fmt.Sprintf("  %s = call i8 @bazic_any_eq(%%Any %s, %%Any %s)\n", tmp, lv, rv)
			if ex.Op == "!=" {
				tmp2 := ctx.ir.nextTmp()
				code += fmt.Sprintf("  %s = xor i8 %s, 1\n", tmp2, tmp)
				return code, tmp2, ast.TypeBool, true
			}
			return code, tmp, ast.TypeBool, true
		}
		return "", "", ast.TypeInvalid, false
	case *ast.CallExpr:
		if isBuiltinVoidCall(ex.Callee) {
			code, ok := emitBuiltinCall(ctx, ex, funcs)
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			return code, "", ast.TypeVoid, true
		}
		if ex.Callee == "len" && len(ex.Args) == 1 {
			code, val, t, ok := emitExpr(ctx, ex.Args[0], funcs)
			if !ok || t != ast.TypeString {
				return "", "", ast.TypeInvalid, false
			}
			tmp := ctx.ir.nextTmp()
			code += fmt.Sprintf("  %s = call i64 @bazic_len(ptr %s)\n", tmp, val)
			return code, tmp, ast.TypeInt, true
		}
		if ex.Callee == "contains" && len(ex.Args) == 2 {
			c1, v1, t1, ok := emitExpr(ctx, ex.Args[0], funcs)
			if !ok || t1 != ast.TypeString {
				return "", "", ast.TypeInvalid, false
			}
			c2, v2, t2, ok := emitExpr(ctx, ex.Args[1], funcs)
			if !ok || t2 != ast.TypeString {
				return "", "", ast.TypeInvalid, false
			}
			tmp := ctx.ir.nextTmp()
			code := c1 + c2 + fmt.Sprintf("  %s = call i8 @bazic_contains(ptr %s, ptr %s)\n", tmp, v1, v2)
			return code, tmp, ast.TypeBool, true
		}
		if ex.Callee == "starts_with" && len(ex.Args) == 2 {
			c1, v1, t1, ok := emitExpr(ctx, ex.Args[0], funcs)
			if !ok || t1 != ast.TypeString {
				return "", "", ast.TypeInvalid, false
			}
			c2, v2, t2, ok := emitExpr(ctx, ex.Args[1], funcs)
			if !ok || t2 != ast.TypeString {
				return "", "", ast.TypeInvalid, false
			}
			tmp := ctx.ir.nextTmp()
			code := c1 + c2 + fmt.Sprintf("  %s = call i8 @bazic_starts_with(ptr %s, ptr %s)\n", tmp, v1, v2)
			return code, tmp, ast.TypeBool, true
		}
		if ex.Callee == "ends_with" && len(ex.Args) == 2 {
			c1, v1, t1, ok := emitExpr(ctx, ex.Args[0], funcs)
			if !ok || t1 != ast.TypeString {
				return "", "", ast.TypeInvalid, false
			}
			c2, v2, t2, ok := emitExpr(ctx, ex.Args[1], funcs)
			if !ok || t2 != ast.TypeString {
				return "", "", ast.TypeInvalid, false
			}
			tmp := ctx.ir.nextTmp()
			code := c1 + c2 + fmt.Sprintf("  %s = call i8 @bazic_ends_with(ptr %s, ptr %s)\n", tmp, v1, v2)
			return code, tmp, ast.TypeBool, true
		}
		if ex.Callee == "to_upper" && len(ex.Args) == 1 {
			c1, v1, t1, ok := emitExpr(ctx, ex.Args[0], funcs)
			if !ok || t1 != ast.TypeString {
				return "", "", ast.TypeInvalid, false
			}
			tmp := ctx.ir.nextTmp()
			code := c1 + fmt.Sprintf("  %s = call ptr @bazic_to_upper(ptr %s)\n", tmp, v1)
			return code, tmp, ast.TypeString, true
		}
		if ex.Callee == "to_lower" && len(ex.Args) == 1 {
			c1, v1, t1, ok := emitExpr(ctx, ex.Args[0], funcs)
			if !ok || t1 != ast.TypeString {
				return "", "", ast.TypeInvalid, false
			}
			tmp := ctx.ir.nextTmp()
			code := c1 + fmt.Sprintf("  %s = call ptr @bazic_to_lower(ptr %s)\n", tmp, v1)
			return code, tmp, ast.TypeString, true
		}
		if ex.Callee == "trim_space" && len(ex.Args) == 1 {
			c1, v1, t1, ok := emitExpr(ctx, ex.Args[0], funcs)
			if !ok || t1 != ast.TypeString {
				return "", "", ast.TypeInvalid, false
			}
			tmp := ctx.ir.nextTmp()
			code := c1 + fmt.Sprintf("  %s = call ptr @bazic_trim_space(ptr %s)\n", tmp, v1)
			return code, tmp, ast.TypeString, true
		}
		if ex.Callee == "replace" && len(ex.Args) == 3 {
			c1, v1, t1, ok := emitExpr(ctx, ex.Args[0], funcs)
			if !ok || t1 != ast.TypeString {
				return "", "", ast.TypeInvalid, false
			}
			c2, v2, t2, ok := emitExpr(ctx, ex.Args[1], funcs)
			if !ok || t2 != ast.TypeString {
				return "", "", ast.TypeInvalid, false
			}
			c3, v3, t3, ok := emitExpr(ctx, ex.Args[2], funcs)
			if !ok || t3 != ast.TypeString {
				return "", "", ast.TypeInvalid, false
			}
			tmp := ctx.ir.nextTmp()
			code := c1 + c2 + c3 + fmt.Sprintf("  %s = call ptr @bazic_replace(ptr %s, ptr %s, ptr %s)\n", tmp, v1, v2, v3)
			return code, tmp, ast.TypeString, true
		}
		if ex.Callee == "repeat" && len(ex.Args) == 2 {
			c1, v1, t1, ok := emitExpr(ctx, ex.Args[0], funcs)
			if !ok || t1 != ast.TypeString {
				return "", "", ast.TypeInvalid, false
			}
			c2, v2, t2, ok := emitExpr(ctx, ex.Args[1], funcs)
			if !ok || t2 != ast.TypeInt {
				return "", "", ast.TypeInvalid, false
			}
			tmp := ctx.ir.nextTmp()
			code := c1 + c2 + fmt.Sprintf("  %s = call ptr @bazic_repeat(ptr %s, i64 %s)\n", tmp, v1, v2)
			return code, tmp, ast.TypeString, true
		}
		if ex.Callee == "str" && len(ex.Args) == 1 {
			c1, v1, t1, ok := emitExpr(ctx, ex.Args[0], funcs)
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			if t1 == ast.TypeString {
				return c1, v1, ast.TypeString, true
			}
			if t1 == ast.TypeBool {
				trueCode, truePtr, ok := stringPtr(ctx, "true")
				if !ok {
					return "", "", ast.TypeInvalid, false
				}
				falseCode, falsePtr, ok := stringPtr(ctx, "false")
				if !ok {
					return "", "", ast.TypeInvalid, false
				}
				tmp := ctx.ir.nextTmp()
				condCode, cond := boolToI1(ctx, v1)
				code := c1 + trueCode + falseCode + condCode
				code += fmt.Sprintf("  %s = select i1 %s, ptr %s, ptr %s\n", tmp, cond, truePtr, falsePtr)
				return code, tmp, ast.TypeString, true
			}
			if t1 == ast.TypeInt || ctx.enums.enumTypes[string(t1)] {
				tmp := ctx.ir.nextTmp()
				code := c1 + fmt.Sprintf("  %s = call ptr @bazic_int_to_str(i64 %s)\n", tmp, v1)
				return code, tmp, ast.TypeString, true
			}
			if t1 == ast.TypeFloat {
				tmp := ctx.ir.nextTmp()
				code := c1 + fmt.Sprintf("  %s = call ptr @bazic_float_to_str(double %s)\n", tmp, v1)
				return code, tmp, ast.TypeString, true
			}
			if t1 == ast.TypeAny {
				tmp := ctx.ir.nextTmp()
				code := c1 + fmt.Sprintf("  %s = call ptr @bazic_any_to_str(%%Any %s)\n", tmp, v1)
				return code, tmp, ast.TypeString, true
			}
			return "", "", ast.TypeInvalid, false
		}
		if ex.Callee == "parse_int" && len(ex.Args) == 1 {
			c1, v1, t1, ok := emitExpr(ctx, ex.Args[0], funcs)
			if !ok || t1 != ast.TypeString {
				return "", "", ast.TypeInvalid, false
			}
			tmp := ctx.ir.nextTmp()
			typeName := resultStructName("int", "Error")
			code := c1 + fmt.Sprintf("  %s = call %%%s @bazic_parse_int(ptr %s)\n", tmp, typeName, v1)
			return code, tmp, ast.Type(typeName), true
		}
		if ex.Callee == "parse_float" && len(ex.Args) == 1 {
			c1, v1, t1, ok := emitExpr(ctx, ex.Args[0], funcs)
			if !ok || t1 != ast.TypeString {
				return "", "", ast.TypeInvalid, false
			}
			tmp := ctx.ir.nextTmp()
			typeName := resultStructName("float", "Error")
			code := c1 + fmt.Sprintf("  %s = call %%%s @bazic_parse_float(ptr %s)\n", tmp, typeName, v1)
			return code, tmp, ast.Type(typeName), true
		}
		sig, ok := funcs[ex.Callee]
		if !ok || len(sig.Params) != len(ex.Args) {
			return "", "", ast.TypeInvalid, false
		}
		var b strings.Builder
		argParts := make([]string, 0, len(ex.Args))
		for i, a := range ex.Args {
			code, value, t, ok := emitExpr(ctx, a, funcs)
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			if sig.Params[i] == ast.TypeAny && t != ast.TypeAny {
				boxCode, boxed, ok := boxToAny(ctx, value, t)
				if !ok {
					return "", "", ast.TypeInvalid, false
				}
				code += boxCode
				value = boxed
				t = ast.TypeAny
			}
			if t != sig.Params[i] {
				return "", "", ast.TypeInvalid, false
			}
			b.WriteString(code)
			llvmType, ok := mapLLVMType(t, ctx.enums, ctx.structs, ctx.ifaces)
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			argParts = append(argParts, fmt.Sprintf("%s %s", llvmType, value))
		}
		retType, ok := mapLLVMType(sig.Ret, ctx.enums, ctx.structs, ctx.ifaces)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		if sig.Ret == ast.TypeVoid {
			b.WriteString(fmt.Sprintf("  call %s @%s(%s)\n", retType, ex.Callee, strings.Join(argParts, ", ")))
			return b.String(), "", ast.TypeVoid, true
		}
		if strings.HasPrefix(ex.Callee, "__std_") {
			if _, ok := ctx.structs.byName[string(sig.Ret)]; ok {
				tmpPtr := ctx.ir.nextTmp()
				b.WriteString(fmt.Sprintf("  %s = alloca %s\n", tmpPtr, retType))
				args := append([]string{fmt.Sprintf("ptr sret(%s) %s", retType, tmpPtr)}, argParts...)
				b.WriteString(fmt.Sprintf("  call void @%s(%s)\n", ex.Callee, strings.Join(args, ", ")))
				tmp := ctx.ir.nextTmp()
				b.WriteString(fmt.Sprintf("  %s = load %s, ptr %s\n", tmp, retType, tmpPtr))
				return b.String(), tmp, sig.Ret, true
			}
		}
		tmp := ctx.ir.nextTmp()
		b.WriteString(fmt.Sprintf("  %s = call %s @%s(%s)\n", tmp, retType, ex.Callee, strings.Join(argParts, ", ")))
		return b.String(), tmp, sig.Ret, true
	case *ast.MatchExpr:
		if ex.ResolvedType == ast.TypeInvalid {
			return "", "", ast.TypeInvalid, false
		}
		code, subjVal, subjType, ok := emitExpr(ctx, ex.Subject, funcs)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		if _, ok := ctx.enums.enumTypes[string(subjType)]; !ok {
			return "", "", ast.TypeInvalid, false
		}
		llvmType, ok := mapLLVMType(ex.ResolvedType, ctx.enums, ctx.structs, ctx.ifaces)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		mergeLabel := ctx.ir.nextLabel("match_expr_end")
		defaultLabel := ctx.ir.nextLabel("match_expr_default")
		grouped := groupMatchExprArmsLLVM(ex.Arms)
		var b strings.Builder
		subjLLVMType, ok := mapLLVMType(subjType, ctx.enums, ctx.structs, ctx.ifaces)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		b.WriteString(code)
		b.WriteString(fmt.Sprintf("  switch %s %s, label %%%s [\n", subjLLVMType, subjVal, defaultLabel))
		caseLabels := make([]string, 0, len(grouped))
		for _, g := range grouped {
			lbl := ctx.ir.nextLabel("match_expr_arm")
			caseLabels = append(caseLabels, lbl)
			idx, ok := ctx.enums.variantIndex[g.Variant]
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			b.WriteString(fmt.Sprintf("    %s %d, label %%%s\n", subjLLVMType, idx, lbl))
		}
		b.WriteString("  ]\n")
		phiVals := make([]string, 0, len(ex.Arms))
		for i, g := range grouped {
			b.WriteString(fmt.Sprintf("%s:\n", caseLabels[i]))
			entries, ok := emitGuardedMatchExpr(&b, ctx, g.Arms, funcs, mergeLabel, ex.ResolvedType, caseLabels[i])
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			phiVals = append(phiVals, entries...)
		}
		b.WriteString(fmt.Sprintf("%s:\n", defaultLabel))
		defVal := defaultLLVMValue(ex.ResolvedType, ctx.enums, ctx.structs, ctx.ifaces)
		b.WriteString(fmt.Sprintf("  br label %%%s\n", mergeLabel))
		phiVals = append(phiVals, fmt.Sprintf("[ %s, %%%s ]", defVal, defaultLabel))
		b.WriteString(fmt.Sprintf("%s:\n", mergeLabel))
		tmp := ctx.ir.nextTmp()
		b.WriteString(fmt.Sprintf("  %s = phi %s %s\n", tmp, llvmType, strings.Join(phiVals, ", ")))
		return b.String(), tmp, ex.ResolvedType, true
	default:
		return "", "", ast.TypeInvalid, false
	}
}

func findReturnExpr(b *ast.BlockStmt) ast.Expr {
	if b == nil {
		return nil
	}
	for _, st := range b.Stmts {
		if ret, ok := st.(*ast.ReturnStmt); ok {
			return ret.Value
		}
	}
	return nil
}

func mapLLVMType(t ast.Type, enums enumInfo, structs structPool, ifaces interfacePool) (string, bool) {
	switch t {
	case ast.TypeVoid:
		return "void", true
	case ast.TypeInt:
		return "i64", true
	case ast.TypeFloat:
		return "double", true
	case ast.TypeBool:
		return "i8", true
	case ast.TypeString:
		return "ptr", true
	case ast.TypeAny:
		return "%Any", true
	default:
		if enums.enumTypes[string(t)] {
			return "i64", true
		}
		if isGenericType(string(t)) {
			return "", false
		}
		if _, ok := structs.byName[string(t)]; ok {
			return "%" + string(t), true
		}
		if ifaces.names[string(t)] {
			return "%" + string(t), true
		}
		return "", false
	}
}

func isGenericType(t string) bool {
	open := strings.IndexRune(t, '[')
	close := strings.LastIndex(t, "]")
	return open > 0 && close > open
}

func splitGenericBase(t string) (string, bool) {
	open := strings.IndexRune(t, '[')
	if open <= 0 {
		return "", false
	}
	return t[:open], true
}

func defaultLLVMValue(t ast.Type, enums enumInfo, structs structPool, ifaces interfacePool) string {
	switch t {
	case ast.TypeInt:
		return "0"
	case ast.TypeFloat:
		return "0.0"
	case ast.TypeBool:
		return "0"
	case ast.TypeString:
		return "null"
	case ast.TypeAny:
		return "zeroinitializer"
	default:
		if enums.enumTypes[string(t)] {
			return "0"
		}
		if _, ok := structs.byName[string(t)]; ok {
			return "zeroinitializer"
		}
		if ifaces.names[string(t)] {
			return "zeroinitializer"
		}
		return "0"
	}
}

func boolToI1(ctx *funcCtx, val string) (string, string) {
	tmp := ctx.ir.nextTmp()
	return fmt.Sprintf("  %s = icmp ne i8 %s, 0\n", tmp, val), tmp
}

func boolToI8(ctx *funcCtx, val string) (string, string) {
	tmp := ctx.ir.nextTmp()
	return fmt.Sprintf("  %s = zext i1 %s to i8\n", tmp, val), tmp
}

func isBuiltinVoidCall(name string) bool {
	switch name {
	case "print", "println":
		return true
	default:
		return false
	}
}

func stringPtr(ctx *funcCtx, value string) (string, string, bool) {
	name, ok := ctx.strs.names[value]
	if !ok {
		return "", "", false
	}
	length := len([]byte(value)) + 1
	tmp := ctx.ir.nextTmp()
	code := fmt.Sprintf("  %s = %s\n", tmp, stringGEP(name, length))
	return code, tmp, true
}

func boxToAny(ctx *funcCtx, value string, t ast.Type) (string, string, bool) {
	tag := anyTagOther
	var code strings.Builder
	payload := value

	switch {
	case t == ast.TypeInt || ctx.enums.enumTypes[string(t)]:
		tag = anyTagInt
		tmp := ctx.ir.nextTmp()
		code.WriteString(fmt.Sprintf("  %s = inttoptr i64 %s to ptr\n", tmp, value))
		payload = tmp
	case t == ast.TypeBool:
		tag = anyTagBool
		tmp := ctx.ir.nextTmp()
		code.WriteString(fmt.Sprintf("  %s = zext i8 %s to i64\n", tmp, value))
		tmp2 := ctx.ir.nextTmp()
		code.WriteString(fmt.Sprintf("  %s = inttoptr i64 %s to ptr\n", tmp2, tmp))
		payload = tmp2
	case t == ast.TypeFloat:
		tag = anyTagFloat
		tmp := ctx.ir.nextTmp()
		code.WriteString(fmt.Sprintf("  %s = bitcast double %s to i64\n", tmp, value))
		tmp2 := ctx.ir.nextTmp()
		code.WriteString(fmt.Sprintf("  %s = inttoptr i64 %s to ptr\n", tmp2, tmp))
		payload = tmp2
	case t == ast.TypeString:
		tag = anyTagString
	case ctx.structs.byName[string(t)].Fields != nil || ctx.ifaces.names[string(t)]:
		tag = anyTagOther
		llvmType, ok := mapLLVMType(t, ctx.enums, ctx.structs, ctx.ifaces)
		if !ok {
			return "", "", false
		}
		sizePtr := ctx.ir.nextTmp()
		sizeTmp := ctx.ir.nextTmp()
		code.WriteString(fmt.Sprintf("  %s = getelementptr %s, ptr null, i32 1\n", sizePtr, llvmType))
		code.WriteString(fmt.Sprintf("  %s = ptrtoint ptr %s to i64\n", sizeTmp, sizePtr))
		memTmp := ctx.ir.nextTmp()
		code.WriteString(fmt.Sprintf("  %s = call ptr @malloc(i64 %s)\n", memTmp, sizeTmp))
		castTmp := ctx.ir.nextTmp()
		code.WriteString(fmt.Sprintf("  %s = bitcast ptr %s to %s*\n", castTmp, memTmp, llvmType))
		code.WriteString(fmt.Sprintf("  store %s %s, ptr %s\n", llvmType, value, castTmp))
		payload = memTmp
	default:
		return "", "", false
	}

	tmp0 := ctx.ir.nextTmp()
	tmp1 := ctx.ir.nextTmp()
	code.WriteString(fmt.Sprintf("  %s = insertvalue %%Any undef, i64 %d, 0\n", tmp0, tag))
	code.WriteString(fmt.Sprintf("  %s = insertvalue %%Any %s, ptr %s, 1\n", tmp1, tmp0, payload))
	return code.String(), tmp1, true
}

func emitBuiltinCall(ctx *funcCtx, call *ast.CallExpr, funcs map[string]llvmFuncSig) (string, bool) {
	if len(call.Args) != 1 {
		return "", false
	}
	argCode, argVal, argType, ok := emitExpr(ctx, call.Args[0], funcs)
	if !ok {
		return "", false
	}
	isPrintln := call.Callee == "println"
	var fmtLit string
	switch {
	case argType == ast.TypeInt || ctx.enums.enumTypes[string(argType)]:
		if isPrintln {
			fmtLit = "%ld\n"
		} else {
			fmtLit = "%ld"
		}
		fmtCode, fmtPtr, ok := stringPtr(ctx, fmtLit)
		if !ok {
			return "", false
		}
		return argCode + fmtCode + fmt.Sprintf("  call i32 @printf(ptr %s, i64 %s)\n", fmtPtr, argVal), true
	case argType == ast.TypeFloat:
		if isPrintln {
			fmtLit = "%g\n"
		} else {
			fmtLit = "%g"
		}
		fmtCode, fmtPtr, ok := stringPtr(ctx, fmtLit)
		if !ok {
			return "", false
		}
		return argCode + fmtCode + fmt.Sprintf("  call i32 @printf(ptr %s, double %s)\n", fmtPtr, argVal), true
	case argType == ast.TypeBool:
		if isPrintln {
			fmtLit = "%s\n"
		} else {
			fmtLit = "%s"
		}
		fmtCode, fmtPtr, ok := stringPtr(ctx, fmtLit)
		if !ok {
			return "", false
		}
		trueCode, truePtr, ok := stringPtr(ctx, "true")
		if !ok {
			return "", false
		}
		falseCode, falsePtr, ok := stringPtr(ctx, "false")
		if !ok {
			return "", false
		}
		tmp := ctx.ir.nextTmp()
		condCode, cond := boolToI1(ctx, argVal)
		code := argCode + fmtCode + trueCode + falseCode + condCode
		code += fmt.Sprintf("  %s = select i1 %s, ptr %s, ptr %s\n", tmp, cond, truePtr, falsePtr)
		code += fmt.Sprintf("  call i32 @printf(ptr %s, ptr %s)\n", fmtPtr, tmp)
		return code, true
	case argType == ast.TypeString:
		if isPrintln {
			fmtLit = "%s\n"
		} else {
			fmtLit = "%s"
		}
		fmtCode, fmtPtr, ok := stringPtr(ctx, fmtLit)
		if !ok {
			return "", false
		}
		return argCode + fmtCode + fmt.Sprintf("  call i32 @printf(ptr %s, ptr %s)\n", fmtPtr, argVal), true
	case argType == ast.TypeAny:
		if isPrintln {
			fmtLit = "%s\n"
		} else {
			fmtLit = "%s"
		}
		fmtCode, fmtPtr, ok := stringPtr(ctx, fmtLit)
		if !ok {
			return "", false
		}
		tmp := ctx.ir.nextTmp()
		code := argCode + fmtCode
		code += fmt.Sprintf("  %s = call ptr @bazic_any_to_str(%%Any %s)\n", tmp, argVal)
		code += fmt.Sprintf("  call i32 @printf(ptr %s, ptr %s)\n", fmtPtr, tmp)
		return code, true
	default:
		return "", false
	}
}

func mapIntArithOp(op string) (string, bool) {
	switch op {
	case "+":
		return "add", true
	case "-":
		return "sub", true
	case "*":
		return "mul", true
	case "/":
		return "sdiv", true
	case "%":
		return "srem", true
	default:
		return "", false
	}
}

func mapFloatArithOp(op string) (string, bool) {
	switch op {
	case "+":
		return "fadd", true
	case "-":
		return "fsub", true
	case "*":
		return "fmul", true
	case "/":
		return "fdiv", true
	default:
		return "", false
	}
}

func mapIntCmpOp(op string) (string, bool) {
	switch op {
	case "==":
		return "eq", true
	case "!=":
		return "ne", true
	case "<":
		return "slt", true
	case "<=":
		return "sle", true
	case ">":
		return "sgt", true
	case ">=":
		return "sge", true
	default:
		return "", false
	}
}

func mapFloatCmpOp(op string) (string, bool) {
	switch op {
	case "==":
		return "oeq", true
	case "!=":
		return "one", true
	case "<":
		return "olt", true
	case "<=":
		return "ole", true
	case ">":
		return "ogt", true
	case ">=":
		return "oge", true
	default:
		return "", false
	}
}
