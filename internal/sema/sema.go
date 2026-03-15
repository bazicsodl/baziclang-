package sema

import (
	"fmt"
	"sort"
	"strings"

	"baziclang/internal/ast"
)

type FuncSig struct {
	TypeParams      []string
	TypeParamBounds map[string]ast.Type
	Params          []ast.Type
	Ret             ast.Type
}

type StructSig struct {
	TypeParams      []string
	TypeParamBounds map[string]ast.Type
	Fields          map[string]ast.Type
}

type InterfaceMethodSig struct {
	Params []ast.Type
	Ret    ast.Type
}

type InterfaceSig struct {
	Methods map[string]InterfaceMethodSig
}

type Checker struct {
	functions  map[string]FuncSig
	structs    map[string]StructSig
	interfaces map[string]InterfaceSig
	enums      map[string]bool
	enumVars   map[string]map[string]bool
	globals    map[string]ast.Type
	globalsConst map[string]bool
	impls      []ast.ImplDecl
	scopes     []map[string]*varInfo
	currentFn  ast.Type
	fnTypes    map[string]bool
}

type varInfo struct {
	typ  ast.Type
	used bool
	isConst bool
}

func New() *Checker {
	c := &Checker{
		functions:  map[string]FuncSig{},
		structs:    map[string]StructSig{},
		interfaces: map[string]InterfaceSig{},
		enums:      map[string]bool{},
		enumVars:   map[string]map[string]bool{},
		globals:    map[string]ast.Type{},
		globalsConst: map[string]bool{},
		scopes:     []map[string]*varInfo{},
		fnTypes:    map[string]bool{},
	}
	c.functions["print"] = FuncSig{Params: []ast.Type{ast.TypeAny}, Ret: ast.TypeVoid}
	c.functions["println"] = FuncSig{Params: []ast.Type{ast.TypeAny}, Ret: ast.TypeVoid}
	c.functions["str"] = FuncSig{Params: []ast.Type{ast.TypeAny}, Ret: ast.TypeString}
	c.functions["len"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeInt}
	c.functions["contains"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.TypeBool}
	c.functions["starts_with"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.TypeBool}
	c.functions["ends_with"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.TypeBool}
	c.functions["to_upper"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeString}
	c.functions["to_lower"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeString}
	c.functions["trim_space"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeString}
	c.functions["replace"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.TypeString}
	c.functions["repeat"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeInt}, Ret: ast.TypeString}
	c.functions["parse_int"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.Type("Result[int,Error]")}
	c.functions["parse_float"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.Type("Result[float,Error]")}
	c.functions["__std_read_file"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_write_file"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_read_line"] = FuncSig{Params: []ast.Type{}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_read_all"] = FuncSig{Params: []ast.Type{}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_exists"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeBool}
	c.functions["__std_mkdir_all"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_remove"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_list_dir"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_unix_millis"] = FuncSig{Params: []ast.Type{}, Ret: ast.TypeInt}
	c.functions["__std_sleep_ms"] = FuncSig{Params: []ast.Type{ast.TypeInt}, Ret: ast.TypeVoid}
	c.functions["__std_now_rfc3339"] = FuncSig{Params: []ast.Type{}, Ret: ast.TypeString}
	c.functions["__std_json_escape"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeString}
	c.functions["__std_json_pretty"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_json_validate"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeBool}
	c.functions["__std_json_minify"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_json_get_raw"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_json_get_string"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_json_get_bool"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_json_get_int"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[int,Error]")}
	c.functions["__std_json_get_float"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[float,Error]")}
	c.functions["__std_http_get"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_http_post"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_http_serve_text"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_http_get_opts"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_http_post_opts"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_http_request"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_http_get_opts_resp"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString}, Ret: ast.Type("Result[HttpResponse,Error]")}
	c.functions["__std_http_post_opts_resp"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString}, Ret: ast.Type("Result[HttpResponse,Error]")}
	c.functions["__std_http_request_resp"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString}, Ret: ast.Type("Result[HttpResponse,Error]")}
	c.functions["__std_http_serve_app"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_sha256_hex"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeString}
	c.functions["__std_hmac_sha256_hex"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.TypeString}
	c.functions["__std_random_hex"] = FuncSig{Params: []ast.Type{ast.TypeInt}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_jwt_sign_hs256"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_jwt_verify_hs256"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_bcrypt_hash"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeInt}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_bcrypt_verify"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_session_init"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_session_put"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_session_get_user"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_session_delete"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_time_add_days"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeInt}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_kv_get"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.TypeString}
	c.functions["__std_header_get"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.TypeString}
	c.functions["__std_query_get"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.TypeString}
	c.functions["__std_open_url"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_args"] = FuncSig{Params: []ast.Type{}, Ret: ast.TypeString}
	c.functions["__std_getenv"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_cwd"] = FuncSig{Params: []ast.Type{}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_chdir"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_env_list"] = FuncSig{Params: []ast.Type{}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_temp_dir"] = FuncSig{Params: []ast.Type{}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_exe_path"] = FuncSig{Params: []ast.Type{}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_home_dir"] = FuncSig{Params: []ast.Type{}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_web_get_json"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_web_set_json"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_base64_encode"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeString}
	c.functions["__std_base64_decode"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_path_basename"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeString}
	c.functions["__std_path_dirname"] = FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeString}
	c.functions["__std_path_join"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.TypeString}
	c.functions["__std_db_exec"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_db_query"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_db_exec_with"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_db_query_with"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_db_query_json"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_db_query_json_with"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_db_query_one_json"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_db_query_one_json_with"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_db_exec_returning_id"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[int,Error]")}
	c.functions["__std_db_exec_returning_id_with"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[int,Error]")}
	c.functions["__std_db_exec_params"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_db_exec_params_with"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[bool,Error]")}
	c.functions["__std_db_query_params"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_db_query_params_with"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_db_query_json_params"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_db_query_json_params_with"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_db_query_one_json_params"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_db_query_one_json_params_with"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[string,Error]")}
	c.functions["__std_db_exec_returning_id_params"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[int,Error]")}
	c.functions["__std_db_exec_returning_id_params_with"] = FuncSig{Params: []ast.Type{ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString}, Ret: ast.Type("Result[int,Error]")}
	return c
}

func (c *Checker) Check(p *ast.Program) error {
	for _, d := range p.Decls {
		switch decl := d.(type) {
		case *ast.ImportDecl:
			continue
		case *ast.StructDecl:
			if _, exists := c.structs[decl.Name]; exists {
				return fmt.Errorf("type error: duplicate struct '%s'", decl.Name)
			}
			tpMap := map[string]bool{}
			for _, tp := range decl.TypeParams {
				if tpMap[tp] {
					return fmt.Errorf("type error: duplicate type parameter '%s' on struct '%s'", tp, decl.Name)
				}
				tpMap[tp] = true
			}
			if decl.TypeParamBounds == nil {
				decl.TypeParamBounds = map[string]ast.Type{}
			}
			for tp := range decl.TypeParamBounds {
				if !tpMap[tp] {
					return fmt.Errorf("type error: unknown type parameter '%s' in bounds for struct '%s'", tp, decl.Name)
				}
			}
			fields := map[string]ast.Type{}
			for _, f := range decl.Fields {
				if _, ok := fields[f.Name]; ok {
					return fmt.Errorf("type error: duplicate field '%s' in struct '%s'", f.Name, decl.Name)
				}
				if err := c.validateTypeRef(f.Type, tpMap, false); err != nil {
					return fmt.Errorf("in struct '%s': %w", decl.Name, err)
				}
				fields[f.Name] = f.Type
			}
			c.structs[decl.Name] = StructSig{TypeParams: decl.TypeParams, TypeParamBounds: decl.TypeParamBounds, Fields: fields}
		case *ast.InterfaceDecl:
			if _, exists := c.interfaces[decl.Name]; exists {
				return fmt.Errorf("type error: duplicate interface '%s'", decl.Name)
			}
			methods := map[string]InterfaceMethodSig{}
			for _, m := range decl.Methods {
				if _, exists := methods[m.Name]; exists {
					return fmt.Errorf("type error: duplicate method '%s' in interface '%s'", m.Name, decl.Name)
				}
				params := make([]ast.Type, 0, len(m.Params))
				for _, p := range m.Params {
					if err := c.validateTypeRef(p.Type, nil, false); err != nil {
						return fmt.Errorf("in interface '%s' method '%s': %w", decl.Name, m.Name, err)
					}
					params = append(params, p.Type)
				}
				if err := c.validateTypeRef(m.Return, nil, true); err != nil {
					return fmt.Errorf("in interface '%s' method '%s': %w", decl.Name, m.Name, err)
				}
				methods[m.Name] = InterfaceMethodSig{Params: params, Ret: m.Return}
			}
			c.interfaces[decl.Name] = InterfaceSig{Methods: methods}
		case *ast.EnumDecl:
			if c.enums[decl.Name] {
				return fmt.Errorf("type error: duplicate enum '%s'", decl.Name)
			}
			c.enums[decl.Name] = true
			c.enumVars[decl.Name] = map[string]bool{}
			for _, v := range decl.Variants {
				if c.enumVars[decl.Name][v] {
					return fmt.Errorf("type error: duplicate enum variant '%s' in enum '%s'", v, decl.Name)
				}
				c.enumVars[decl.Name][v] = true
				if _, exists := c.globals[v]; exists {
					return fmt.Errorf("type error: duplicate global symbol '%s'", v)
				}
				c.globals[v] = ast.Type(decl.Name)
			}
		case *ast.FuncDecl:
			if _, exists := c.functions[decl.Name]; exists {
				return fmt.Errorf("type error: duplicate function '%s'", decl.Name)
			}
			tpMap := map[string]bool{}
			for _, tp := range decl.TypeParams {
				if tpMap[tp] {
					return fmt.Errorf("type error: duplicate type parameter '%s' in function '%s'", tp, decl.Name)
				}
				tpMap[tp] = true
			}
			if decl.TypeParamBounds == nil {
				decl.TypeParamBounds = map[string]ast.Type{}
			}
			for tp := range decl.TypeParamBounds {
				if !tpMap[tp] {
					return fmt.Errorf("type error: unknown type parameter '%s' in bounds for function '%s'", tp, decl.Name)
				}
			}
			params := make([]ast.Type, 0, len(decl.Params))
			for _, param := range decl.Params {
				if err := c.validateTypeRef(param.Type, tpMap, false); err != nil {
					return fmt.Errorf("in function '%s': %w", decl.Name, err)
				}
				params = append(params, param.Type)
			}
			if err := c.validateTypeRef(decl.ReturnType, tpMap, true); err != nil {
				return fmt.Errorf("in function '%s': %w", decl.Name, err)
			}
			c.functions[decl.Name] = FuncSig{TypeParams: decl.TypeParams, TypeParamBounds: decl.TypeParamBounds, Params: params, Ret: decl.ReturnType}
		case *ast.ImplDecl:
			c.impls = append(c.impls, *decl)
		case *ast.GlobalLetDecl:
			continue
		}
	}

	for _, d := range p.Decls {
		if decl, ok := d.(*ast.GlobalLetDecl); ok {
			t, err := c.exprType(decl.Init)
			if err != nil {
				return err
			}
			if decl.Type == ast.TypeInvalid {
				decl.Type = t
			}
			if err := c.validateTypeRef(decl.Type, nil, false); err != nil {
				return err
			}
			if decl.Type != t && decl.Type != ast.TypeAny {
				return fmt.Errorf("type error: global '%s' expected %s but got %s", decl.Name, decl.Type, t)
			}
			if _, exists := c.globals[decl.Name]; exists {
				return fmt.Errorf("type error: duplicate global '%s'", decl.Name)
			}
			c.globals[decl.Name] = decl.Type
			if decl.IsConst {
				c.globalsConst[decl.Name] = true
			}
		}
	}

	if _, ok := c.functions["main"]; !ok {
		return fmt.Errorf("type error: missing required 'main' function")
	}
	if err := c.validateMainSignature(); err != nil {
		return err
	}
	if err := c.validateTypeParamBounds(); err != nil {
		return err
	}
	for _, d := range p.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			if err := c.checkFunc(fn); err != nil {
				return err
			}
		}
	}
	if err := c.checkImpls(); err != nil {
		return err
	}
	return nil
}

func (c *Checker) validateTypeParamBounds() error {
	for name, sig := range c.structs {
		for tp, bound := range sig.TypeParamBounds {
			if bound == "" {
				continue
			}
			if _, ok := c.interfaces[string(bound)]; !ok {
				return fmt.Errorf("type error: unknown interface '%s' bound for struct '%s' type param '%s'", bound, name, tp)
			}
		}
	}
	for name, sig := range c.functions {
		for tp, bound := range sig.TypeParamBounds {
			if bound == "" {
				continue
			}
			if _, ok := c.interfaces[string(bound)]; !ok {
				return fmt.Errorf("type error: unknown interface '%s' bound for function '%s' type param '%s'", bound, name, tp)
			}
		}
	}
	return nil
}

func (c *Checker) validateMainSignature() error {
	sig := c.functions["main"]
	if len(sig.TypeParams) != 0 {
		return fmt.Errorf("type error: 'main' cannot be generic")
	}
	if len(sig.Params) != 0 {
		return fmt.Errorf("type error: 'main' must not take parameters")
	}
	if sig.Ret != ast.TypeVoid {
		return fmt.Errorf("type error: 'main' must return void")
	}
	return nil
}

func (c *Checker) checkFunc(fn *ast.FuncDecl) error {
	c.currentFn = fn.ReturnType
	c.fnTypes = map[string]bool{}
	for _, t := range fn.TypeParams {
		c.fnTypes[t] = true
	}
	c.pushScope()
	for _, param := range fn.Params {
		if err := c.declare(param.Name, param.Type, false); err != nil {
			return err
		}
	}
	for _, s := range fn.Body.Stmts {
		if err := c.checkStmt(s); err != nil {
			return fmt.Errorf("in function '%s': %w", fn.Name, err)
		}
	}
	if fn.ReturnType != ast.TypeVoid && !blockAlwaysReturns(fn.Body) {
		return fmt.Errorf("in function '%s': type error: missing return on some control paths", fn.Name)
	}
	if err := c.popScope(); err != nil {
		return fmt.Errorf("in function '%s': %w", fn.Name, err)
	}
	c.fnTypes = map[string]bool{}
	return nil
}

func (c *Checker) checkStmt(s ast.Stmt) error {
	switch st := s.(type) {
	case *ast.LetStmt:
		t, err := c.exprType(st.Init)
		if err != nil {
			return err
		}
		if st.Type == ast.TypeInvalid {
			st.Type = t
		}
		if err := c.validateTypeRef(st.Type, c.fnTypes, false); err != nil {
			return err
		}
		if st.Type != t && st.Type != ast.TypeAny {
			return fmt.Errorf("type error: variable '%s' expected %s but got %s", st.Name, st.Type, t)
		}
		return c.declare(st.Name, st.Type, st.IsConst)
	case *ast.AssignStmt:
		targetType, err := c.typeOfAssignTarget(st.Target)
		if err != nil {
			return err
		}
		if root, ok := rootIdent(st.Target); ok {
			if _, isConst, ok := c.resolveVar(root, false); ok && isConst {
				return fmt.Errorf("type error: cannot assign to const '%s'", root)
			}
		}
		rhs, err := c.exprType(st.Value)
		if err != nil {
			return err
		}
		if targetType != rhs && targetType != ast.TypeAny {
			return fmt.Errorf("type error: cannot assign %s to '%s' (%s)", rhs, formatAssignTargetName(st.Target), targetType)
		}
		return nil
	case *ast.IfStmt:
		cond, err := c.exprType(st.Cond)
		if err != nil {
			return err
		}
		if cond != ast.TypeBool {
			return fmt.Errorf("type error: if condition must be bool, got %s", cond)
		}
		if err := c.checkBlock(st.Then); err != nil {
			return err
		}
		if st.Else != nil {
			if err := c.checkBlock(st.Else); err != nil {
				return err
			}
		}
		return nil
	case *ast.WhileStmt:
		cond, err := c.exprType(st.Cond)
		if err != nil {
			return err
		}
		if cond != ast.TypeBool {
			return fmt.Errorf("type error: while condition must be bool, got %s", cond)
		}
		return c.checkBlock(st.Body)
	case *ast.MatchStmt:
		return c.checkMatchStmt(st)
	case *ast.ReturnStmt:
		if st.Value == nil {
			if c.currentFn != ast.TypeVoid {
				return fmt.Errorf("type error: return value required for function returning %s", c.currentFn)
			}
			return nil
		}
		t, err := c.exprType(st.Value)
		if err != nil {
			return err
		}
		if t != c.currentFn && c.currentFn != ast.TypeAny {
			return fmt.Errorf("type error: return type mismatch, expected %s got %s", c.currentFn, t)
		}
		return nil
	case *ast.ExprStmt:
		_, err := c.exprType(st.Expr)
		return err
	default:
		return fmt.Errorf("type error: unsupported statement")
	}
}

func (c *Checker) typeOfAssignTarget(target ast.Expr) (ast.Type, error) {
	switch t := target.(type) {
	case *ast.IdentExpr:
		typ, _, ok := c.resolveVar(t.Name, false)
		if !ok {
			return ast.TypeInvalid, fmt.Errorf("type error: unknown variable '%s'%s", t.Name, c.suggestVisibleName(t.Name))
		}
		return typ, nil
	case *ast.FieldAccessExpr:
		return c.exprType(t)
	default:
		return ast.TypeInvalid, fmt.Errorf("type error: invalid assignment target")
	}
}

func rootIdent(target ast.Expr) (string, bool) {
	switch t := target.(type) {
	case *ast.IdentExpr:
		return t.Name, true
	case *ast.FieldAccessExpr:
		return rootIdent(t.Object)
	default:
		return "", false
	}
}

func formatAssignTargetName(target ast.Expr) string {
	switch t := target.(type) {
	case *ast.IdentExpr:
		return t.Name
	case *ast.FieldAccessExpr:
		return formatAssignTargetName(t.Object) + "." + t.Field
	default:
		return "target"
	}
}

func (c *Checker) checkMatchStmt(st *ast.MatchStmt) error {
	enumName, variants, err := c.resolveMatchSubjectEnum(st.Subject)
	if err != nil {
		return err
	}
	unguarded := map[string]bool{}
	for _, arm := range st.Arms {
		if err := c.validateMatchVariant(enumName, variants, unguarded, arm.Variant, arm.Guard); err != nil {
			return err
		}
		if arm.Guard != nil {
			t, err := c.exprType(arm.Guard)
			if err != nil {
				return err
			}
			if t != ast.TypeBool {
				return fmt.Errorf("type error: match guard must be bool, got %s", t)
			}
		}
		if err := c.checkBlock(arm.Body); err != nil {
			return err
		}
	}
	if err := ensureMatchExhaustive(enumName, variants, unguarded); err != nil {
		return err
	}
	return nil
}

func (c *Checker) checkBlock(b *ast.BlockStmt) error {
	c.pushScope()
	for _, s := range b.Stmts {
		if err := c.checkStmt(s); err != nil {
			_ = c.popScope()
			return err
		}
	}
	return c.popScope()
}

func (c *Checker) exprType(e ast.Expr) (ast.Type, error) {
	switch ex := e.(type) {
	case *ast.IntExpr:
		return ast.TypeInt, nil
	case *ast.FloatExpr:
		return ast.TypeFloat, nil
	case *ast.BoolExpr:
		return ast.TypeBool, nil
	case *ast.StringExpr:
		return ast.TypeString, nil
	case *ast.NilExpr:
		return ast.TypeInvalid, fmt.Errorf("type error: 'nil' is not a value in Bazic; use Option[T], Result[T,E], or Error with explicit state")
	case *ast.IdentExpr:
		if t, ok := c.resolve(ex.Name, true); ok {
			return t, nil
		}
		return ast.TypeInvalid, fmt.Errorf("type error: unknown identifier '%s'%s", ex.Name, c.suggestVisibleName(ex.Name))
	case *ast.StructLitExpr:
		base, args, _ := splitGenericType(string(ex.TypeName))
		if base == "" {
			base = ex.TypeName
		}
		sig, ok := c.structs[base]
		if !ok {
			return ast.TypeInvalid, fmt.Errorf("type error: unknown struct '%s'", ex.TypeName)
		}
		mapping, err := bindTypeParams(sig.TypeParams, args)
		if err != nil {
			return ast.TypeInvalid, fmt.Errorf("type error: %w", err)
		}
		for tp, bound := range sig.TypeParamBounds {
			if bound == "" {
				continue
			}
			if actual, ok := mapping[tp]; ok {
				if err := c.requireImplements(actual, bound); err != nil {
					return ast.TypeInvalid, fmt.Errorf("type error: type argument '%s' does not satisfy bound %s: %w", actual, bound, err)
				}
			}
		}
		seen := map[string]bool{}
		for _, field := range ex.Fields {
			rawExpected, ok := sig.Fields[field.Name]
			if !ok {
				return ast.TypeInvalid, fmt.Errorf("type error: unknown field '%s' on struct '%s'%s", field.Name, ex.TypeName, suggestNameSuffix(field.Name, mapKeys(sig.Fields)))
			}
			expected := substType(rawExpected, mapping)
			vt, err := c.exprType(field.Value)
			if err != nil {
				return ast.TypeInvalid, err
			}
			if vt != expected && expected != ast.TypeAny {
				return ast.TypeInvalid, fmt.Errorf("type error: field '%s' on '%s' expected %s got %s", field.Name, ex.TypeName, expected, vt)
			}
			seen[field.Name] = true
		}
		for name := range sig.Fields {
			if !seen[name] {
				return ast.TypeInvalid, fmt.Errorf("type error: missing field '%s' in struct literal '%s'", name, ex.TypeName)
			}
		}
		return ast.Type(ex.TypeName), nil
	case *ast.FieldAccessExpr:
		objType, err := c.exprType(ex.Object)
		if err != nil {
			return ast.TypeInvalid, err
		}
		base, args, _ := splitGenericType(string(objType))
		if base == "" {
			base = string(objType)
		}
		sig, ok := c.structs[base]
		if !ok {
			return ast.TypeInvalid, fmt.Errorf("type error: field access requires struct type, got %s", objType)
		}
		mapping, err := bindTypeParams(sig.TypeParams, args)
		if err != nil {
			return ast.TypeInvalid, err
		}
		rawField, ok := sig.Fields[ex.Field]
		if !ok {
			return ast.TypeInvalid, fmt.Errorf("type error: struct '%s' has no field '%s'%s", objType, ex.Field, suggestNameSuffix(ex.Field, mapKeys(sig.Fields)))
		}
		return substType(rawField, mapping), nil
	case *ast.UnaryExpr:
		r, err := c.exprType(ex.Right)
		if err != nil {
			return ast.TypeInvalid, err
		}
		switch ex.Op {
		case "-":
			if r == ast.TypeInt || r == ast.TypeFloat {
				return r, nil
			}
			return ast.TypeInvalid, fmt.Errorf("type error: unary '-' requires numeric type")
		case "!":
			if r == ast.TypeBool {
				return ast.TypeBool, nil
			}
			return ast.TypeInvalid, fmt.Errorf("type error: unary '!' requires bool")
		}
		return ast.TypeInvalid, fmt.Errorf("type error: unsupported unary operator '%s'", ex.Op)
	case *ast.BinaryExpr:
		l, err := c.exprType(ex.Left)
		if err != nil {
			return ast.TypeInvalid, err
		}
		r, err := c.exprType(ex.Right)
		if err != nil {
			return ast.TypeInvalid, err
		}
		switch ex.Op {
		case "+", "-", "*", "/", "%":
			if l != r {
				return ast.TypeInvalid, fmt.Errorf("type error: operator '%s' requires matching operands", ex.Op)
			}
			if ex.Op == "+" && l == ast.TypeString {
				return ast.TypeString, nil
			}
			if l == ast.TypeInt || l == ast.TypeFloat {
				if ex.Op == "%" && l != ast.TypeInt {
					return ast.TypeInvalid, fmt.Errorf("type error: '%%' only supports int")
				}
				return l, nil
			}
			return ast.TypeInvalid, fmt.Errorf("type error: invalid operands for '%s'", ex.Op)
		case "==", "!=":
			if l != r {
				return ast.TypeInvalid, fmt.Errorf("type error: comparison requires same types")
			}
			return ast.TypeBool, nil
		case "<", "<=", ">", ">=":
			if l != r || (l != ast.TypeInt && l != ast.TypeFloat && l != ast.TypeString) {
				return ast.TypeInvalid, fmt.Errorf("type error: invalid operands for comparison")
			}
			return ast.TypeBool, nil
		case "&&", "||":
			if l == ast.TypeBool && r == ast.TypeBool {
				return ast.TypeBool, nil
			}
			return ast.TypeInvalid, fmt.Errorf("type error: logical operators require bool")
		}
		return ast.TypeInvalid, fmt.Errorf("type error: unsupported operator '%s'", ex.Op)
	case *ast.CallExpr:
		if ex.Receiver != nil {
			receiverType, err := c.exprType(ex.Receiver)
			if err != nil {
				return ast.TypeInvalid, err
			}
			base, _, ok := splitGenericType(string(receiverType))
			if !ok {
				base = string(receiverType)
			}
			resolvedName := fmt.Sprintf("%s_%s", base, ex.Method)
			sig, ok := c.functions[resolvedName]
			if !ok {
				return ast.TypeInvalid, fmt.Errorf("type error: unknown method '%s' on '%s'%s (expected function '%s')", ex.Method, receiverType, c.suggestMethodName(base, ex.Method), resolvedName)
			}
			ex.Callee = resolvedName
			ex.Args = append([]ast.Expr{ex.Receiver}, ex.Args...)
			ex.Receiver = nil
			ex.Method = ""
			return c.checkCallExpr(ex.Callee, ex.Args, sig)
		}
		sig, ok := c.functions[ex.Callee]
		if !ok {
			return ast.TypeInvalid, fmt.Errorf("type error: unknown function '%s'%s", ex.Callee, suggestNameSuffix(ex.Callee, mapKeys(c.functions)))
		}
		return c.checkCallExpr(ex.Callee, ex.Args, sig)
	case *ast.MatchExpr:
		enumName, variants, err := c.resolveMatchSubjectEnum(ex.Subject)
		if err != nil {
			return ast.TypeInvalid, err
		}
		unguarded := map[string]bool{}
		armType := ast.TypeInvalid
		for _, arm := range ex.Arms {
			if err := c.validateMatchVariant(enumName, variants, unguarded, arm.Variant, arm.Guard); err != nil {
				return ast.TypeInvalid, err
			}
			if arm.Guard != nil {
				t, err := c.exprType(arm.Guard)
				if err != nil {
					return ast.TypeInvalid, err
				}
				if t != ast.TypeBool {
					return ast.TypeInvalid, fmt.Errorf("type error: match guard must be bool, got %s", t)
				}
			}
			t, err := c.exprType(arm.Value)
			if err != nil {
				return ast.TypeInvalid, err
			}
			if armType == ast.TypeInvalid {
				armType = t
				continue
			}
			if armType != t {
				return ast.TypeInvalid, fmt.Errorf("type error: match expression arm type mismatch, expected %s got %s", armType, t)
			}
		}
		if err := ensureMatchExhaustive(enumName, variants, unguarded); err != nil {
			return ast.TypeInvalid, err
		}
		if armType == ast.TypeInvalid {
			return ast.TypeInvalid, fmt.Errorf("type error: match expression must have at least one arm")
		}
		ex.ResolvedType = armType
		return armType, nil
	default:
		return ast.TypeInvalid, fmt.Errorf("type error: unsupported expression")
	}
}

func (c *Checker) resolveMatchSubjectEnum(subject ast.Expr) (string, map[string]bool, error) {
	subjectType, err := c.exprType(subject)
	if err != nil {
		return "", nil, err
	}
	enumName := string(subjectType)
	variants, ok := c.enumVars[enumName]
	if !ok {
		return "", nil, fmt.Errorf("type error: match subject must be enum, got %s", subjectType)
	}
	return enumName, variants, nil
}

func (c *Checker) validateMatchVariant(enumName string, variants map[string]bool, unguarded map[string]bool, variant string, guard ast.Expr) error {
	if !variants[variant] {
		return fmt.Errorf("type error: unknown variant '%s' for enum '%s'", variant, enumName)
	}
	if guard == nil {
		if unguarded[variant] {
			return fmt.Errorf("type error: duplicate match arm for variant '%s'", variant)
		}
		unguarded[variant] = true
		return nil
	}
	if unguarded[variant] {
		return fmt.Errorf("type error: guarded match arm must appear before unguarded arm for variant '%s'", variant)
	}
	return nil
}

func ensureMatchExhaustive(enumName string, variants map[string]bool, seen map[string]bool) error {
	missing := make([]string, 0, len(variants))
	for v := range variants {
		if !seen[v] {
			missing = append(missing, v)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("type error: non-exhaustive match for enum '%s'; missing unguarded variants: %s", enumName, strings.Join(missing, ", "))
	}
	return nil
}

func (c *Checker) checkCallExpr(name string, args []ast.Expr, sig FuncSig) (ast.Type, error) {
	if len(args) != len(sig.Params) {
		return ast.TypeInvalid, fmt.Errorf("type error: function '%s' expects %d args, got %d", name, len(sig.Params), len(args))
	}
	mapping := map[string]ast.Type{}
	for i, arg := range args {
		at, err := c.exprType(arg)
		if err != nil {
			return ast.TypeInvalid, err
		}
		expected := sig.Params[i]
		if expected == ast.TypeAny {
			continue
		}
		if !unifyTypeParams(expected, at, mapping, sig.TypeParams) {
			return ast.TypeInvalid, fmt.Errorf("type error: arg %d to '%s' expected %s got %s", i+1, name, expected, at)
		}
	}
	ret := sig.Ret
	if len(mapping) > 0 {
		ret = substType(ret, mapping)
	}
	if err := c.validateTypeParamBoundsForMapping(sig.TypeParamBounds, mapping, name); err != nil {
		return ast.TypeInvalid, err
	}
	for _, tp := range sig.TypeParams {
		if typeContainsParam(ret, tp) {
			return ast.TypeInvalid, fmt.Errorf("type error: could not infer return type for generic function '%s'", name)
		}
	}
	return ret, nil
}

func unifyTypeParams(expected ast.Type, actual ast.Type, mapping map[string]ast.Type, params []string) bool {
	if expected == ast.TypeAny {
		return true
	}
	if contains(params, string(expected)) {
		if bound, ok := mapping[string(expected)]; ok {
			return bound == actual
		}
		mapping[string(expected)] = actual
		return true
	}
	baseE, argsE, okE := splitGenericType(string(expected))
	if okE {
		baseA, argsA, okA := splitGenericType(string(actual))
		if !okA || baseA != baseE || len(argsA) != len(argsE) {
			return false
		}
		for i := range argsE {
			if !unifyTypeParams(ast.Type(argsE[i]), ast.Type(argsA[i]), mapping, params) {
				return false
			}
		}
		return true
	}
	return expected == actual
}

func (c *Checker) validateTypeParamBoundsForMapping(bounds map[string]ast.Type, mapping map[string]ast.Type, owner string) error {
	if bounds == nil {
		return nil
	}
	for tp, bound := range bounds {
		if bound == "" {
			continue
		}
		actual, ok := mapping[tp]
		if !ok {
			return fmt.Errorf("type error: could not infer type argument '%s' for '%s'", tp, owner)
		}
		if err := c.requireImplements(actual, bound); err != nil {
			return fmt.Errorf("type error: type argument '%s' for '%s' does not satisfy bound %s: %w", tp, owner, bound, err)
		}
	}
	return nil
}

func (c *Checker) requireImplements(t ast.Type, iface ast.Type) error {
	base, _, ok := splitGenericType(string(t))
	if ok {
		t = ast.Type(base)
	}
	if _, ok := c.structs[string(t)]; !ok {
		return fmt.Errorf("'%s' is not a struct type", t)
	}
	if !c.hasImpl(t, string(iface)) {
		return fmt.Errorf("'%s' does not implement '%s'", t, iface)
	}
	return nil
}

func (c *Checker) hasImpl(structType ast.Type, iface string) bool {
	base, _, ok := splitGenericType(string(structType))
	if ok {
		structType = ast.Type(base)
	}
	for _, impl := range c.impls {
		implBase, _, ok := splitGenericType(string(impl.StructType))
		if ok {
			if implBase == string(structType) && impl.InterfaceName == iface {
				return true
			}
			continue
		}
		if impl.StructType == structType && impl.InterfaceName == iface {
			return true
		}
	}
	return false
}

func (c *Checker) checkImpls() error {
	for _, impl := range c.impls {
		base, _, ok := splitGenericType(string(impl.StructType))
		if !ok {
			base = string(impl.StructType)
		}
		if _, exists := c.structs[base]; !exists {
			return fmt.Errorf("type error: impl target struct '%s' not found", impl.StructType)
		}
		iface, exists := c.interfaces[impl.InterfaceName]
		if !exists {
			return fmt.Errorf("type error: interface '%s' not found for impl", impl.InterfaceName)
		}
		for mname, msig := range iface.Methods {
			fnName := fmt.Sprintf("%s_%s", base, mname)
			fn, ok := c.functions[fnName]
			if !ok {
				return fmt.Errorf("type error: impl %s:%s missing function '%s'", impl.StructType, impl.InterfaceName, fnName)
			}
			if len(msig.Params) > 0 && msig.Params[0] == impl.StructType {
				if len(fn.Params) != len(msig.Params) {
					return fmt.Errorf("type error: '%s' must have %d params", fnName, len(msig.Params))
				}
				for i := range msig.Params {
					if fn.Params[i] != msig.Params[i] {
						return fmt.Errorf("type error: '%s' param %d mismatch", fnName, i+1)
					}
				}
			} else {
				if len(fn.Params) != len(msig.Params)+1 {
					return fmt.Errorf("type error: '%s' must have receiver + %d params", fnName, len(msig.Params))
				}
				if fn.Params[0] != impl.StructType {
					return fmt.Errorf("type error: '%s' first param must be %s", fnName, impl.StructType)
				}
				for i := range msig.Params {
					if fn.Params[i+1] != msig.Params[i] {
						return fmt.Errorf("type error: '%s' param %d mismatch", fnName, i+2)
					}
				}
			}
			if fn.Ret != msig.Ret {
				return fmt.Errorf("type error: '%s' return type mismatch", fnName)
			}
		}
	}
	return nil
}

func bindTypeParams(typeParams []string, args []string) (map[string]ast.Type, error) {
	if len(typeParams) == 0 {
		if len(args) > 0 {
			return nil, fmt.Errorf("non-generic type used with type arguments")
		}
		return map[string]ast.Type{}, nil
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("generic type requires %d type arguments", len(typeParams))
	}
	if len(typeParams) != len(args) {
		return nil, fmt.Errorf("generic type expected %d type arguments, got %d", len(typeParams), len(args))
	}
	m := map[string]ast.Type{}
	for i, tp := range typeParams {
		m[tp] = ast.Type(args[i])
	}
	return m, nil
}

func substType(t ast.Type, mapping map[string]ast.Type) ast.Type {
	if v, ok := mapping[string(t)]; ok {
		return v
	}
	base, args, ok := splitGenericType(string(t))
	if !ok {
		return t
	}
	mapped := make([]string, 0, len(args))
	for _, a := range args {
		mapped = append(mapped, string(substType(ast.Type(a), mapping)))
	}
	return ast.Type(fmt.Sprintf("%s[%s]", base, strings.Join(mapped, ",")))
}

func (c *Checker) validateTypeRef(t ast.Type, typeParams map[string]bool, allowVoid bool) error {
	if t == ast.TypeInvalid {
		return fmt.Errorf("type error: invalid type")
	}
	if t == ast.TypeVoid && !allowVoid {
		return fmt.Errorf("type error: void cannot be used here")
	}
	name := string(t)
	if typeParams != nil && typeParams[name] {
		return nil
	}
	if isBuiltin(t) || c.structs[name].Fields != nil || c.enums[name] || c.interfaces[name].Methods != nil {
		return nil
	}
	if base, args, ok := splitGenericType(name); ok {
		sig, ok := c.structs[base]
		if !ok {
			return fmt.Errorf("type error: unknown generic base type '%s'", base)
		}
		if len(sig.TypeParams) != len(args) {
			return fmt.Errorf("type error: type '%s' expects %d args, got %d", base, len(sig.TypeParams), len(args))
		}
		for _, a := range args {
			if err := c.validateTypeRef(ast.Type(a), typeParams, false); err != nil {
				return err
			}
		}
		for i, tp := range sig.TypeParams {
			bound := sig.TypeParamBounds[tp]
			if bound == "" {
				continue
			}
			arg := ast.Type(args[i])
			if typeParams != nil && typeParams[string(arg)] {
				continue
			}
			if err := c.requireImplements(arg, bound); err != nil {
				return fmt.Errorf("type error: type argument '%s' does not satisfy bound %s: %w", arg, bound, err)
			}
		}
		return nil
	}
	return fmt.Errorf("type error: unknown type '%s'", t)
}

func splitGenericType(t string) (string, []string, bool) {
	open := strings.IndexRune(t, '[')
	close := strings.LastIndex(t, "]")
	if open <= 0 || close <= open {
		return "", nil, false
	}
	base := t[:open]
	inner := t[open+1 : close]
	parts := splitTopLevel(inner)
	if len(parts) == 0 {
		return "", nil, false
	}
	return base, parts, true
}

func splitTopLevel(s string) []string {
	depth := 0
	parts := []string{}
	start := 0
	for i, r := range s {
		switch r {
		case '[':
			depth++
		case ']':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	parts = append(parts, strings.TrimSpace(s[start:]))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func isBuiltin(t ast.Type) bool {
	switch t {
	case ast.TypeAny, ast.TypeBool, ast.TypeInt, ast.TypeFloat, ast.TypeString, ast.TypeVoid:
		return true
	default:
		return false
	}
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func typeContainsParam(t ast.Type, param string) bool {
	if string(t) == param {
		return true
	}
	base, args, ok := splitGenericType(string(t))
	if !ok || base == "" {
		return false
	}
	for _, a := range args {
		if typeContainsParam(ast.Type(a), param) {
			return true
		}
	}
	return false
}

func (c *Checker) suggestVisibleName(name string) string {
	candidates := map[string]bool{}
	for i := len(c.scopes) - 1; i >= 0; i-- {
		for n := range c.scopes[i] {
			if strings.HasPrefix(n, "_#") {
				continue
			}
			candidates[n] = true
		}
	}
	for n := range c.globals {
		candidates[n] = true
	}
	return suggestNameSuffix(name, mapKeys(candidates))
}

func (c *Checker) suggestMethodName(base, name string) string {
	prefix := base + "_"
	candidates := []string{}
	for fn := range c.functions {
		if strings.HasPrefix(fn, prefix) {
			candidates = append(candidates, strings.TrimPrefix(fn, prefix))
		}
	}
	return suggestNameSuffix(name, candidates)
}

func suggestNameSuffix(target string, candidates []string) string {
	best, ok := closestName(target, candidates)
	if !ok {
		return ""
	}
	return fmt.Sprintf(" (did you mean '%s'?)", best)
}

func closestName(target string, candidates []string) (string, bool) {
	best := ""
	bestScore := 1 << 30
	for _, c := range candidates {
		if c == "" {
			continue
		}
		score := levenshtein(strings.ToLower(target), strings.ToLower(c))
		if strings.HasPrefix(strings.ToLower(c), strings.ToLower(target)) || strings.HasPrefix(strings.ToLower(target), strings.ToLower(c)) {
			score--
		}
		if score < bestScore {
			bestScore = score
			best = c
		}
	}
	if best == "" {
		return "", false
	}
	if bestScore > 2 {
		return "", false
	}
	return best, true
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr := make([]int, len(b)+1)
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			curr[j] = minInt(del, minInt(ins, sub))
		}
		prev = curr
	}
	return prev[len(b)]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func mapKeys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func (c *Checker) declare(name string, t ast.Type, isConst bool) error {
	scope := c.scopes[len(c.scopes)-1]
	if name == "_" {
		// Allow explicit discard bindings repeatedly.
		scope[name+fmt.Sprintf("#%d", len(scope))] = &varInfo{typ: t, used: true, isConst: false}
		return nil
	}
	if _, exists := scope[name]; exists {
		return fmt.Errorf("type error: duplicate variable '%s'", name)
	}
	scope[name] = &varInfo{typ: t, used: false, isConst: isConst}
	return nil
}

func (c *Checker) resolve(name string, markUsed bool) (ast.Type, bool) {
	t, _, ok := c.resolveVar(name, markUsed)
	return t, ok
}

func (c *Checker) resolveVar(name string, markUsed bool) (ast.Type, bool, bool) {
	for i := len(c.scopes) - 1; i >= 0; i-- {
		if v, ok := c.scopes[i][name]; ok {
			if markUsed {
				v.used = true
			}
			return v.typ, v.isConst, true
		}
	}
	if t, ok := c.globals[name]; ok {
		return t, c.globalsConst[name], true
	}
	return ast.TypeInvalid, false, false
}

func (c *Checker) pushScope() { c.scopes = append(c.scopes, map[string]*varInfo{}) }

func (c *Checker) popScope() error {
	scope := c.scopes[len(c.scopes)-1]
	c.scopes = c.scopes[:len(c.scopes)-1]
	for name, info := range scope {
		if strings.HasPrefix(name, "_#") {
			continue
		}
		if !info.used {
			return fmt.Errorf("type error: unused variable '%s' (use '_' to ignore)", name)
		}
	}
	return nil
}

func blockAlwaysReturns(b *ast.BlockStmt) bool {
	if b == nil {
		return false
	}
	for _, s := range b.Stmts {
		if stmtAlwaysReturns(s) {
			return true
		}
	}
	return false
}

func stmtAlwaysReturns(s ast.Stmt) bool {
	switch st := s.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.IfStmt:
		if st.Else == nil {
			return false
		}
		return blockAlwaysReturns(st.Then) && blockAlwaysReturns(st.Else)
	case *ast.MatchStmt:
		if len(st.Arms) == 0 {
			return false
		}
		for _, arm := range st.Arms {
			if !blockAlwaysReturns(arm.Body) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
