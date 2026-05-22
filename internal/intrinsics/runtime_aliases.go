package intrinsics

import "strings"

var runtimeCallAliases = map[string]string{
	"auth_hash_password":               "crypto_bcrypt_hash",
	"auth_verify_password":             "crypto_bcrypt_verify",
	"crypto_bcrypt_hash":               "__std_bcrypt_hash",
	"crypto_bcrypt_verify":             "__std_bcrypt_verify",
	"crypto_hmac_sha256_hex":           "__std_hmac_sha256_hex",
	"crypto_random_hex":                "__std_random_hex",
	"crypto_sha256_hex":                "__std_sha256_hex",
	"db_exec":                          "__std_db_exec",
	"db_exec_params":                   "__std_db_exec_params",
	"db_exec_params_with":              "__std_db_exec_params_with",
	"db_exec_returning_id":             "__std_db_exec_returning_id",
	"db_exec_returning_id_params":      "__std_db_exec_returning_id_params",
	"db_exec_returning_id_params_with": "__std_db_exec_returning_id_params_with",
	"db_exec_returning_id_with":        "__std_db_exec_returning_id_with",
	"db_exec_with":                     "__std_db_exec_with",
	"db_query":                         "__std_db_query",
	"db_query_json":                    "__std_db_query_json",
	"db_query_json_params":             "__std_db_query_json_params",
	"db_query_json_params_with":        "__std_db_query_json_params_with",
	"db_query_json_with":               "__std_db_query_json_with",
	"db_query_one_json":                "__std_db_query_one_json",
	"db_query_one_json_params":         "__std_db_query_one_json_params",
	"db_query_one_json_params_with":    "__std_db_query_one_json_params_with",
	"db_query_one_json_with":           "__std_db_query_one_json_with",
	"db_query_params":                  "__std_db_query_params",
	"db_query_params_with":             "__std_db_query_params_with",
	"db_query_with":                    "__std_db_query_with",
	"http_serve_app":                   "__std_http_serve_app",
	"jwt_sign_hs256":                   "__std_jwt_sign_hs256",
	"jwt_verify_hs256":                 "__std_jwt_verify_hs256",
	"session_delete":                   "__std_session_delete",
	"session_get_user":                 "__std_session_get_user",
	"session_init":                     "__std_session_init",
	"session_put":                      "__std_session_put",
	"web_get_json":                     "__std_web_get_json",
	"web_set_json":                     "__std_web_set_json",
}

func CanonicalRuntimeCallName(name string) string {
	current := strings.TrimSpace(name)
	if current == "" {
		return ""
	}
	seen := map[string]struct{}{}
	for {
		next, ok := runtimeCallAliases[current]
		if !ok || next == "" {
			return current
		}
		if _, exists := seen[current]; exists {
			return current
		}
		seen[current] = struct{}{}
		current = next
	}
}

func IntrinsicTargetName(name string) (string, bool) {
	canonical := CanonicalRuntimeCallName(name)
	if strings.HasPrefix(canonical, "__std_") {
		return canonical, true
	}
	return "", false
}
