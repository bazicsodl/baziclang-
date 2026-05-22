package intrinsics

type CallFamily string

const (
	CallFamilyBcrypt       CallFamily = "bcrypt"
	CallFamilyDB           CallFamily = "db"
	CallFamilyHMAC         CallFamily = "hmac"
	CallFamilyHTTPServeApp CallFamily = "http_serve_app"
	CallFamilyJWT          CallFamily = "jwt"
	CallFamilySession      CallFamily = "session"
)

var callFamilyNames = map[CallFamily]map[string]struct{}{
	CallFamilyBcrypt: toNameSet(
		"__std_bcrypt_hash", "__std_bcrypt_verify",
	),
	CallFamilyDB: toNameSet(
		"__std_db_exec", "__std_db_query", "__std_db_exec_with", "__std_db_query_with",
		"__std_db_query_json", "__std_db_query_json_with", "__std_db_query_one_json", "__std_db_query_one_json_with",
		"__std_db_exec_returning_id", "__std_db_exec_returning_id_with",
		"__std_db_exec_params", "__std_db_exec_params_with", "__std_db_query_params", "__std_db_query_params_with",
		"__std_db_query_json_params", "__std_db_query_json_params_with", "__std_db_query_one_json_params", "__std_db_query_one_json_params_with",
		"__std_db_exec_returning_id_params", "__std_db_exec_returning_id_params_with",
	),
	CallFamilyHMAC: toNameSet(
		"__std_hmac_sha256_hex",
	),
	CallFamilyHTTPServeApp: toNameSet(
		"__std_http_serve_app",
	),
	CallFamilyJWT: toNameSet(
		"__std_jwt_sign_hs256", "__std_jwt_verify_hs256",
	),
	CallFamilySession: toNameSet(
		"__std_session_init", "__std_session_put", "__std_session_get_user", "__std_session_delete",
	),
}

func IsBcryptCall(name string) bool       { return isCallFamily(CallFamilyBcrypt, name) }
func IsDBCall(name string) bool           { return isCallFamily(CallFamilyDB, name) }
func IsHMACCall(name string) bool         { return isCallFamily(CallFamilyHMAC, name) }
func IsHTTPServeAppCall(name string) bool { return isCallFamily(CallFamilyHTTPServeApp, name) }
func IsJWTCall(name string) bool          { return isCallFamily(CallFamilyJWT, name) }
func IsSessionCall(name string) bool      { return isCallFamily(CallFamilySession, name) }

func isCallFamily(family CallFamily, name string) bool {
	names := callFamilyNames[family]
	_, ok := names[CanonicalRuntimeCallName(name)]
	return ok
}

func toNameSet(names ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(names))
	for _, name := range names {
		out[name] = struct{}{}
	}
	return out
}
