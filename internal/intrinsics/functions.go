package intrinsics

import (
	"baziclang/internal/ast"
	baztypes "baziclang/internal/types"
)

type FunctionSpec struct {
	Name   string
	Params []ast.Type
	Ret    ast.Type
}

func FunctionSpecs(typeAliases map[string]ast.Type) []FunctionSpec {
	raw := []FunctionSpec{
		spec("print", ast.TypeVoid, ast.TypeAny),
		spec("println", ast.TypeVoid, ast.TypeAny),
		spec("str", ast.TypeString, ast.TypeAny),
		spec("len", ast.TypeInt, ast.TypeString),
		spec("contains", ast.TypeBool, ast.TypeString, ast.TypeString),
		spec("starts_with", ast.TypeBool, ast.TypeString, ast.TypeString),
		spec("ends_with", ast.TypeBool, ast.TypeString, ast.TypeString),
		spec("to_upper", ast.TypeString, ast.TypeString),
		spec("to_lower", ast.TypeString, ast.TypeString),
		spec("trim_space", ast.TypeString, ast.TypeString),
		spec("replace", ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString),
		spec("repeat", ast.TypeString, ast.TypeString, ast.TypeInt),
		spec("parse_int", ast.Type("Result[int,Error]"), ast.TypeString),
		spec("parse_float", ast.Type("Result[float,Error]"), ast.TypeString),
		spec("__std_read_file", ast.Type("Result[string,Error]"), ast.TypeString),
		spec("__std_write_file", ast.Type("Result[bool,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_read_line", ast.Type("Result[string,Error]")),
		spec("__std_read_all", ast.Type("Result[string,Error]")),
		spec("__std_exists", ast.TypeBool, ast.TypeString),
		spec("__std_mkdir_all", ast.Type("Result[bool,Error]"), ast.TypeString),
		spec("__std_remove", ast.Type("Result[bool,Error]"), ast.TypeString),
		spec("__std_list_dir", ast.Type("Result[string,Error]"), ast.TypeString),
		spec("__std_unix_millis", ast.TypeInt),
		spec("__std_sleep_ms", ast.TypeVoid, ast.TypeInt),
		spec("__std_now_rfc3339", ast.TypeString),
		spec("__std_json_escape", ast.TypeString, ast.TypeString),
		spec("__std_json_pretty", ast.Type("Result[string,Error]"), ast.TypeString),
		spec("__std_json_validate", ast.TypeBool, ast.TypeString),
		spec("__std_json_minify", ast.Type("Result[string,Error]"), ast.TypeString),
		spec("__std_json_get_raw", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_json_get_string", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_json_get_bool", ast.Type("Result[bool,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_json_get_int", ast.Type("Result[int,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_json_get_float", ast.Type("Result[float,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_http_get", ast.Type("Result[string,Error]"), ast.TypeString),
		spec("__std_http_post", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_http_serve_text", ast.Type("Result[bool,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_http_get_opts", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString),
		spec("__std_http_post_opts", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString),
		spec("__std_http_request", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString),
		spec("__std_http_get_opts_resp", ast.Type("Result[HttpResponse,Error]"), ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString),
		spec("__std_http_post_opts_resp", ast.Type("Result[HttpResponse,Error]"), ast.TypeString, ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString),
		spec("__std_http_request_resp", ast.Type("Result[HttpResponse,Error]"), ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeInt, ast.TypeInt, ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeBool, ast.TypeString),
		spec("__std_http_serve_app", ast.Type("Result[bool,Error]"), ast.TypeString),
		spec("__std_sha256_hex", ast.TypeString, ast.TypeString),
		spec("__std_hmac_sha256_hex", ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_random_hex", ast.Type("Result[string,Error]"), ast.TypeInt),
		spec("__std_jwt_sign_hs256", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_jwt_verify_hs256", ast.Type("Result[bool,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_bcrypt_hash", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeInt),
		spec("__std_bcrypt_verify", ast.Type("Result[bool,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_session_init", ast.Type("Result[bool,Error]"), ast.TypeString),
		spec("__std_session_put", ast.Type("Result[bool,Error]"), ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_session_get_user", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_session_delete", ast.Type("Result[bool,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_time_add_days", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeInt),
		spec("__std_kv_get", ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_header_get", ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_query_get", ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_open_url", ast.Type("Result[bool,Error]"), ast.TypeString),
		spec("__std_args", ast.TypeString),
		spec("__std_getenv", ast.Type("Result[string,Error]"), ast.TypeString),
		spec("__std_cwd", ast.Type("Result[string,Error]")),
		spec("__std_chdir", ast.Type("Result[bool,Error]"), ast.TypeString),
		spec("__std_env_list", ast.Type("Result[string,Error]")),
		spec("__std_temp_dir", ast.Type("Result[string,Error]")),
		spec("__std_exe_path", ast.Type("Result[string,Error]")),
		spec("__std_home_dir", ast.Type("Result[string,Error]")),
		spec("__std_web_get_json", ast.Type("Result[string,Error]"), ast.TypeString),
		spec("__std_web_set_json", ast.Type("Result[bool,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_base64_encode", ast.TypeString, ast.TypeString),
		spec("__std_base64_decode", ast.Type("Result[string,Error]"), ast.TypeString),
		spec("__std_path_basename", ast.TypeString, ast.TypeString),
		spec("__std_path_dirname", ast.TypeString, ast.TypeString),
		spec("__std_path_join", ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_db_exec", ast.Type("Result[bool,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_db_query", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_db_exec_with", ast.Type("Result[bool,Error]"), ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_db_query_with", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_db_query_json", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_db_query_json_with", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_db_query_one_json", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_db_query_one_json_with", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_db_exec_returning_id", ast.Type("Result[int,Error]"), ast.TypeString, ast.TypeString),
		spec("__std_db_exec_returning_id_with", ast.Type("Result[int,Error]"), ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_db_exec_params", ast.Type("Result[bool,Error]"), ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_db_exec_params_with", ast.Type("Result[bool,Error]"), ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_db_query_params", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_db_query_params_with", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_db_query_json_params", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_db_query_json_params_with", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_db_query_one_json_params", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_db_query_one_json_params_with", ast.Type("Result[string,Error]"), ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_db_exec_returning_id_params", ast.Type("Result[int,Error]"), ast.TypeString, ast.TypeString, ast.TypeString),
		spec("__std_db_exec_returning_id_params_with", ast.Type("Result[int,Error]"), ast.TypeString, ast.TypeString, ast.TypeString, ast.TypeString),
	}
	out := make([]FunctionSpec, 0, len(raw))
	for _, fn := range raw {
		spec := FunctionSpec{Name: fn.Name, Ret: substituteTypeAliases(fn.Ret, typeAliases)}
		if len(fn.Params) > 0 {
			spec.Params = make([]ast.Type, 0, len(fn.Params))
			for _, p := range fn.Params {
				spec.Params = append(spec.Params, substituteTypeAliases(p, typeAliases))
			}
		}
		out = append(out, spec)
	}
	return out
}

func spec(name string, ret ast.Type, params ...ast.Type) FunctionSpec {
	return FunctionSpec{Name: name, Params: params, Ret: ret}
}

func substituteTypeAliases(t ast.Type, aliases map[string]ast.Type) ast.Type {
	if len(aliases) == 0 || t == "" {
		return t
	}
	if replacement, ok := aliases[string(t)]; ok {
		return replacement
	}
	if !baztypes.IsGeneric(t) {
		return t
	}
	parsed := baztypes.MustParse(t)
	if len(parsed.Args) == 0 {
		if replacement, ok := aliases[parsed.Name]; ok {
			return replacement
		}
		return t
	}
	args := make([]baztypes.Type, 0, len(parsed.Args))
	changed := false
	for _, arg := range parsed.Args {
		next := substituteTypeAliases(baztypes.ToAST(arg), aliases)
		if next != baztypes.ToAST(arg) {
			changed = true
		}
		args = append(args, baztypes.MustParse(next))
	}
	if replacement, ok := aliases[parsed.Name]; ok {
		base := baztypes.MustParse(replacement)
		base.Args = args
		return baztypes.ToAST(base)
	}
	if !changed {
		return t
	}
	parsed.Args = args
	return baztypes.ToAST(parsed)
}
