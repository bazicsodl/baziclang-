#ifdef _WIN32
#define _CRT_SECURE_NO_WARNINGS
#endif

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <stdbool.h>
#include <ctype.h>
#include <errno.h>
#include <time.h>

#ifdef _WIN32
#include <winsock2.h>
#include <ws2tcpip.h>
#include <windows.h>
#include <winhttp.h>
#include <bcrypt.h>
#include <direct.h>
#include <io.h>
#else
#include <dirent.h>
#include <fcntl.h>
#include <sys/socket.h>
#include <netdb.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <sys/stat.h>
#include <sys/wait.h>
#include <unistd.h>
#include <curl/curl.h>
#endif
#ifdef BAZIC_SQLITE
#include <sqlite3.h>
#endif
#ifdef __APPLE__
#include <mach-o/dyld.h>
#endif

typedef struct {
	char *message;
} Error;

typedef struct {
	bool is_ok;
	char *value;
	Error err;
} Result_string_Error;

typedef struct {
	bool is_ok;
	bool value;
	Error err;
} Result_bool_Error;

typedef struct {
	bool is_ok;
	int64_t value;
	Error err;
} Result_int_Error;

typedef struct {
	bool is_ok;
	double value;
	Error err;
} Result_float_Error;

typedef struct {
	int64_t status;
	char *headers;
	char *body;
} HttpResponse;

typedef struct {
	bool is_ok;
	HttpResponse value;
	Error err;
} Result_HttpResponse_Error;

typedef struct {
	char *method;
	char *path;
	char *query;
	char *headers;
	char *body;
	char *remote_addr;
	char *cookies;
	char *params;
} ServerRequest;

typedef struct {
	int64_t status;
	char *headers;
	char *body;
} ServerResponse;

typedef struct {
	char *method;
	char *path;
	ServerResponse (*handler)(ServerRequest);
} BazicRoute;

extern BazicRoute __bazic_routes[];
extern int64_t __bazic_routes_len;

static int bazic_argc = 0;
static char **bazic_argv = NULL;

Result_string_Error __std_http_get_opts(char *url, int64_t connect_timeout_ms, int64_t timeout_ms, char *hdrs, char *user_agent, bool tls_insecure, char *ca_bundle_pem);
Result_string_Error __std_http_post_opts(char *url, char *body, int64_t connect_timeout_ms, int64_t timeout_ms, char *hdrs, char *user_agent, char *content_type, bool tls_insecure, char *ca_bundle_pem);
Result_string_Error __std_http_request(char *method, char *url, char *body, int64_t connect_timeout_ms, int64_t timeout_ms, char *hdrs, char *user_agent, char *content_type, bool tls_insecure, char *ca_bundle_pem);
Result_HttpResponse_Error __std_http_get_opts_resp(char *url, int64_t connect_timeout_ms, int64_t timeout_ms, char *hdrs, char *user_agent, bool tls_insecure, char *ca_bundle_pem);
Result_HttpResponse_Error __std_http_post_opts_resp(char *url, char *body, int64_t connect_timeout_ms, int64_t timeout_ms, char *hdrs, char *user_agent, char *content_type, bool tls_insecure, char *ca_bundle_pem);
Result_HttpResponse_Error __std_http_request_resp(char *method, char *url, char *body, int64_t connect_timeout_ms, int64_t timeout_ms, char *hdrs, char *user_agent, char *content_type, bool tls_insecure, char *ca_bundle_pem);
Result_bool_Error __std_db_exec(char *path, char *sql);
Result_string_Error __std_db_query(char *path, char *sql);
Result_bool_Error __std_db_exec_with(char *driver, char *dsn, char *sql);
Result_string_Error __std_db_query_with(char *driver, char *dsn, char *sql);
Result_string_Error __std_db_query_json(char *path, char *sql);
Result_string_Error __std_db_query_json_with(char *driver, char *dsn, char *sql);
Result_string_Error __std_db_query_one_json(char *path, char *sql);
Result_string_Error __std_db_query_one_json_with(char *driver, char *dsn, char *sql);
Result_int_Error __std_db_exec_returning_id(char *path, char *sql);
Result_int_Error __std_db_exec_returning_id_with(char *driver, char *dsn, char *sql);
Result_bool_Error __std_db_exec_params(char *path, char *sql, char *params);
Result_bool_Error __std_db_exec_params_with(char *driver, char *dsn, char *sql, char *params);
Result_string_Error __std_db_query_params(char *path, char *sql, char *params);
Result_string_Error __std_db_query_params_with(char *driver, char *dsn, char *sql, char *params);
Result_string_Error __std_db_query_json_params(char *path, char *sql, char *params);
Result_string_Error __std_db_query_json_params_with(char *driver, char *dsn, char *sql, char *params);
Result_string_Error __std_db_query_one_json_params(char *path, char *sql, char *params);
Result_string_Error __std_db_query_one_json_params_with(char *driver, char *dsn, char *sql, char *params);
Result_int_Error __std_db_exec_returning_id_params(char *path, char *sql, char *params);
Result_int_Error __std_db_exec_returning_id_params_with(char *driver, char *dsn, char *sql, char *params);
bool __std_json_validate(char *s);
Result_string_Error __std_json_minify(char *s);
Result_string_Error __std_json_get_raw(char *s, char *key);
Result_string_Error __std_json_get_string(char *s, char *key);
Result_bool_Error __std_json_get_bool(char *s, char *key);
Result_int_Error __std_json_get_int(char *s, char *key);
Result_float_Error __std_json_get_float(char *s, char *key);
Result_string_Error __std_read_all(void);
void __bazic_set_args(int argc, char **argv);
char *__std_args(void);
Result_string_Error __std_getenv(char *key);
Result_string_Error __std_cwd(void);
Result_bool_Error __std_chdir(char *path);
Result_string_Error __std_env_list(void);
Result_string_Error __std_temp_dir(void);
Result_string_Error __std_exe_path(void);
Result_string_Error __std_home_dir(void);
Result_string_Error __std_web_get_json(char *key);
Result_bool_Error __std_web_set_json(char *key, char *jsonText);
char *__std_base64_encode(char *s);
Result_string_Error __std_base64_decode(char *s);
char *__std_path_basename(char *path);
char *__std_path_dirname(char *path);
char *__std_path_join(char *a, char *b);
Result_string_Error __std_time_add_days(char *rfc3339, int64_t days);
char *__std_hmac_sha256_hex(char *message, char *secret);
Result_string_Error __std_jwt_sign_hs256(char *header_json, char *payload_json, char *secret);
Result_bool_Error __std_jwt_verify_hs256(char *token, char *secret);
Result_string_Error __std_bcrypt_hash(char *password, int64_t cost);
Result_bool_Error __std_bcrypt_verify(char *password, char *hash);
Result_bool_Error __std_http_serve_app(char *addr);
char *__std_kv_get(char *kv, char *key);
char *__std_header_get(char *headers, char *key);
char *__std_query_get(char *query, char *key);

static char *bazic_strdup(const char *s) {
	size_t n = s ? strlen(s) : 0;
	char *out = (char *)malloc(n + 1);
	if (!out) {
		return NULL;
	}
	if (n > 0) {
		memcpy(out, s, n);
	}
	out[n] = '\0';
	return out;
}

static char *bazic_strndup(const char *s, size_t n) {
	if (!s) {
		return bazic_strdup("");
	}
	char *out = (char *)malloc(n + 1);
	if (!out) {
		return NULL;
	}
	if (n > 0) {
		memcpy(out, s, n);
	}
	out[n] = '\0';
	return out;
}

static char *bazic_base64url_encode_bytes(const uint8_t *data, size_t len);

static int bazic_stricmp(const char *a, const char *b) {
	if (a == b) {
		return 0;
	}
	if (!a || !b) {
		return a ? 1 : -1;
	}
	while (*a && *b) {
		int ca = tolower((unsigned char)*a);
		int cb = tolower((unsigned char)*b);
		if (ca != cb) {
			return ca - cb;
		}
		a++;
		b++;
	}
	return (unsigned char)*a - (unsigned char)*b;
}

static int bazic_strnicmp(const char *a, const char *b, size_t n) {
	for (size_t i = 0; i < n; i++) {
		unsigned char ca = (unsigned char)a[i];
		unsigned char cb = (unsigned char)b[i];
		ca = (unsigned char)tolower(ca);
		cb = (unsigned char)tolower(cb);
		if (ca != cb) {
			return (int)ca - (int)cb;
		}
		if (a[i] == '\0' || b[i] == '\0') {
			break;
		}
	}
	return 0;
}

#ifdef _WIN32
typedef SOCKET bazic_socket_t;
#else
typedef int bazic_socket_t;
#endif

int64_t bazic_len(char *s) {
	if (!s) {
		return 0;
	}
	const unsigned char *p = (const unsigned char *)s;
	int64_t count = 0;
	while (*p) {
		if ((*p & 0xC0) != 0x80) {
			count++;
		}
		p++;
	}
	return count;
}

static Error bazic_error(const char *msg) {
	Error e;
	e.message = bazic_strdup(msg ? msg : "");
	return e;
}

static Result_string_Error ok_string(const char *s) {
	Result_string_Error r;
	r.is_ok = true;
	r.value = bazic_strdup(s ? s : "");
	r.err = bazic_error("");
	return r;
}

static Result_string_Error ok_string_owned(char *s) {
	Result_string_Error r;
	r.is_ok = true;
	r.value = s ? s : bazic_strdup("");
	r.err = bazic_error("");
	return r;
}

static bool json_validate(const char *s, size_t *i);
static void append_str(char **buf, size_t *cap, size_t *len, const char *s);

static Result_string_Error err_string(const char *msg) {
	Result_string_Error r;
	r.is_ok = false;
	r.value = bazic_strdup("");
	r.err = bazic_error(msg ? msg : "error");
	return r;
}

static Result_bool_Error ok_bool(bool v) {
	Result_bool_Error r;
	r.is_ok = true;
	r.value = v;
	r.err = bazic_error("");
	return r;
}

static Result_bool_Error err_bool(const char *msg) {
	Result_bool_Error r;
	r.is_ok = false;
	r.value = false;
	r.err = bazic_error(msg ? msg : "error");
	return r;
}

static Result_int_Error ok_int(int64_t v) {
	Result_int_Error r;
	r.is_ok = true;
	r.value = v;
	r.err = bazic_error("");
	return r;
}

static Result_int_Error err_int(const char *msg) {
	Result_int_Error r;
	r.is_ok = false;
	r.value = 0;
	r.err = bazic_error(msg ? msg : "error");
	return r;
}

static Result_float_Error ok_float(double v) {
	Result_float_Error r;
	r.is_ok = true;
	r.value = v;
	r.err = bazic_error("");
	return r;
}

static Result_float_Error err_float(const char *msg) {
	Result_float_Error r;
	r.is_ok = false;
	r.value = 0.0;
	r.err = bazic_error(msg ? msg : "error");
	return r;
}

static Result_HttpResponse_Error ok_http_response(int64_t status, char *headers, char *body) {
	Result_HttpResponse_Error r;
	r.is_ok = true;
	r.value.status = status;
	r.value.headers = headers ? headers : bazic_strdup("");
	r.value.body = body ? body : bazic_strdup("");
	r.err = bazic_error("");
	return r;
}

static Result_HttpResponse_Error err_http_response(const char *msg) {
	Result_HttpResponse_Error r;
	r.is_ok = false;
	r.value.status = 0;
	r.value.headers = bazic_strdup("");
	r.value.body = bazic_strdup("");
	r.err = bazic_error(msg ? msg : "error");
	return r;
}

#ifdef _WIN32
static char *win_last_error(void) {
	DWORD code = GetLastError();
	if (code == 0) {
		return bazic_strdup("unknown error");
	}
	LPSTR msg = NULL;
	DWORD len = FormatMessageA(
		FORMAT_MESSAGE_ALLOCATE_BUFFER | FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS,
		NULL, code, MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT), (LPSTR)&msg, 0, NULL);
	if (len == 0 || msg == NULL) {
		return bazic_strdup("unknown error");
	}
	char *out = bazic_strdup(msg);
	LocalFree(msg);
	return out;
}

static wchar_t *utf8_to_wide(const char *s) {
	if (!s) {
		return NULL;
	}
	int needed = MultiByteToWideChar(CP_UTF8, 0, s, -1, NULL, 0);
	if (needed <= 0) {
		return NULL;
	}
	wchar_t *w = (wchar_t *)malloc((size_t)needed * sizeof(wchar_t));
	if (!w) {
		return NULL;
	}
	if (MultiByteToWideChar(CP_UTF8, 0, s, -1, w, needed) == 0) {
		free(w);
		return NULL;
	}
	return w;
}

static char *wide_to_utf8(const wchar_t *w) {
	if (!w) {
		return bazic_strdup("");
	}
	int needed = WideCharToMultiByte(CP_UTF8, 0, w, -1, NULL, 0, NULL, NULL);
	if (needed <= 0) {
		return bazic_strdup("");
	}
	char *out = (char *)malloc((size_t)needed);
	if (!out) {
		return bazic_strdup("");
	}
	if (WideCharToMultiByte(CP_UTF8, 0, w, -1, out, needed, NULL, NULL) == 0) {
		free(out);
		return bazic_strdup("");
	}
	return out;
}

static char *build_header_block(const char *hdrs, const char *accept, const char *content_type) {
	size_t cap = 256;
	size_t len = 0;
	char *out = (char *)malloc(cap);
	if (!out) {
		return NULL;
	}
	out[0] = '\0';
	if (hdrs && hdrs[0] != '\0') {
		char *copy = bazic_strdup(hdrs);
		char *line = strtok(copy, "\n");
		while (line) {
			while (*line == ' ' || *line == '\t') { line++; }
			if (*line != '\0') {
				append_str(&out, &cap, &len, line);
				append_str(&out, &cap, &len, "\r\n");
			}
			line = strtok(NULL, "\n");
		}
		free(copy);
	}
	if (accept && accept[0] != '\0') {
		append_str(&out, &cap, &len, "Accept: ");
		append_str(&out, &cap, &len, accept);
		append_str(&out, &cap, &len, "\r\n");
	}
	if (content_type && content_type[0] != '\0') {
		append_str(&out, &cap, &len, "Content-Type: ");
		append_str(&out, &cap, &len, content_type);
		append_str(&out, &cap, &len, "\r\n");
	}
	return out;
}
#endif

#ifndef _WIN32
static size_t curl_write_cb(char *ptr, size_t size, size_t nmemb, void *userdata) {
	size_t total = size * nmemb;
	if (total == 0) {
		return 0;
	}
	struct {
		char *buf;
		size_t len;
		size_t cap;
	} *ctx = userdata;
	if (ctx->len + total + 1 > ctx->cap) {
		size_t next = ctx->cap == 0 ? 4096 : ctx->cap * 2;
		while (ctx->len + total + 1 > next) {
			next *= 2;
		}
		char *tmp = (char *)realloc(ctx->buf, next);
		if (!tmp) {
			return 0;
		}
		ctx->buf = tmp;
		ctx->cap = next;
	}
	memcpy(ctx->buf + ctx->len, ptr, total);
	ctx->len += total;
	ctx->buf[ctx->len] = '\0';
	return total;
}

static size_t curl_header_cb(char *ptr, size_t size, size_t nmemb, void *userdata) {
	size_t total = size * nmemb;
	if (total == 0) {
		return 0;
	}
	struct {
		char *buf;
		size_t len;
		size_t cap;
	} *ctx = userdata;
	if (total >= 5 && strncmp(ptr, "HTTP/", 5) == 0) {
		return total;
	}
	size_t trim = total;
	while (trim > 0 && (ptr[trim - 1] == '\n' || ptr[trim - 1] == '\r')) {
		trim--;
	}
	if (trim == 0) {
		return total;
	}
	if (ctx->len + trim + 2 > ctx->cap) {
		size_t next = ctx->cap == 0 ? 256 : ctx->cap * 2;
		while (ctx->len + trim + 2 > next) {
			next *= 2;
		}
		char *tmp = (char *)realloc(ctx->buf, next);
		if (!tmp) {
			return 0;
		}
		ctx->buf = tmp;
		ctx->cap = next;
	}
	memcpy(ctx->buf + ctx->len, ptr, trim);
	ctx->len += trim;
	ctx->buf[ctx->len++] = '\n';
	ctx->buf[ctx->len] = '\0';
	return total;
}

static char *write_temp_ca(const char *pem) {
	if (!pem || pem[0] == '\0') {
		return NULL;
	}
	char tmpl[] = "/tmp/bazic_ca_XXXXXX";
	int fd = mkstemp(tmpl);
	if (fd < 0) {
		return NULL;
	}
	size_t len = strlen(pem);
	size_t written = 0;
	while (written < len) {
		ssize_t n = write(fd, pem + written, len - written);
		if (n <= 0) {
			close(fd);
			unlink(tmpl);
			return NULL;
		}
		written += (size_t)n;
	}
	close(fd);
	return bazic_strdup(tmpl);
}
#endif

Result_string_Error __std_read_file(char *path) {
	if (!path) {
		return err_string("read_file: null path");
	}
	FILE *f = fopen(path, "rb");
	if (!f) {
		return err_string(strerror(errno));
	}
	if (fseek(f, 0, SEEK_END) != 0) {
		fclose(f);
		return err_string("read_file: seek failed");
	}
	long size = ftell(f);
	if (size < 0) {
		fclose(f);
		return err_string("read_file: size failed");
	}
	if (fseek(f, 0, SEEK_SET) != 0) {
		fclose(f);
		return err_string("read_file: seek failed");
	}
	char *buf = (char *)malloc((size_t)size + 1);
	if (!buf) {
		fclose(f);
		return err_string("read_file: out of memory");
	}
	size_t n = fread(buf, 1, (size_t)size, f);
	fclose(f);
	buf[n] = '\0';
	return ok_string_owned(buf);
}

Result_bool_Error __std_write_file(char *path, char *data) {
	if (!path) {
		return err_bool("write_file: null path");
	}
	FILE *f = fopen(path, "wb");
	if (!f) {
		return err_bool(strerror(errno));
	}
	size_t len = data ? strlen(data) : 0;
	if (len > 0) {
		size_t n = fwrite(data, 1, len, f);
		if (n != len) {
			fclose(f);
			return err_bool("write_file: short write");
		}
	}
	fclose(f);
	return ok_bool(true);
}

Result_string_Error __std_read_line(void) {
	size_t cap = 128;
	size_t len = 0;
	char *buf = (char *)malloc(cap);
	if (!buf) {
		return err_string("read_line: out of memory");
	}
	int ch;
	while ((ch = fgetc(stdin)) != EOF) {
		if (ch == '\n') {
			break;
		}
		if (len + 1 >= cap) {
			cap *= 2;
			char *next = (char *)realloc(buf, cap);
			if (!next) {
				free(buf);
				return err_string("read_line: out of memory");
			}
			buf = next;
		}
		buf[len++] = (char)ch;
	}
	buf[len] = '\0';
	return ok_string_owned(buf);
}

Result_string_Error __std_read_all(void) {
	size_t cap = 4096;
	size_t len = 0;
	char *buf = (char *)malloc(cap);
	if (!buf) {
		return err_string("read_all: out of memory");
	}
	for (;;) {
		size_t n = fread(buf + len, 1, cap - len, stdin);
		len += n;
		if (n == 0) {
			if (feof(stdin)) {
				break;
			}
			free(buf);
			return err_string("read_all: failed");
		}
		if (len + 1 >= cap) {
			size_t next = cap * 2;
			char *tmp = (char *)realloc(buf, next);
			if (!tmp) {
				free(buf);
				return err_string("read_all: out of memory");
			}
			buf = tmp;
			cap = next;
		}
	}
	buf[len] = '\0';
	return ok_string_owned(buf);
}

bool __std_exists(char *path) {
	if (!path) {
		return false;
	}
#ifdef _WIN32
	return _access(path, 0) == 0;
#else
	return access(path, F_OK) == 0;
#endif
}

static int bazic_mkdir_one(const char *path) {
#ifdef _WIN32
	return _mkdir(path);
#else
	return mkdir(path, 0755);
#endif
}

Result_bool_Error __std_mkdir_all(char *path) {
	if (!path) {
		return err_bool("mkdir_all: null path");
	}
	size_t len = strlen(path);
	if (len == 0) {
		return ok_bool(true);
	}
	char *tmp = bazic_strdup(path);
	if (!tmp) {
		return err_bool("mkdir_all: out of memory");
	}
	size_t start = 0;
#ifdef _WIN32
	if (len >= 3 && tmp[1] == ':' && (tmp[2] == '\\' || tmp[2] == '/')) {
		start = 3;
	}
#endif
	for (size_t i = start; i < len; i++) {
		char c = tmp[i];
		if (c == '/' || c == '\\') {
			tmp[i] = '\0';
			if (strlen(tmp) > 0) {
				if (bazic_mkdir_one(tmp) != 0 && errno != EEXIST) {
					free(tmp);
					return err_bool(strerror(errno));
				}
			}
			tmp[i] = c;
		}
	}
	if (bazic_mkdir_one(tmp) != 0 && errno != EEXIST) {
		free(tmp);
		return err_bool(strerror(errno));
	}
	free(tmp);
	return ok_bool(true);
}

Result_bool_Error __std_remove(char *path) {
	if (!path) {
		return err_bool("remove: null path");
	}
	if (remove(path) == 0) {
		return ok_bool(true);
	}
#ifdef _WIN32
	if (_rmdir(path) == 0) {
		return ok_bool(true);
	}
#else
	if (rmdir(path) == 0) {
		return ok_bool(true);
	}
#endif
	return err_bool(strerror(errno));
}

static void append_str(char **buf, size_t *cap, size_t *len, const char *s) {
	if (!s) {
		return;
	}
	size_t n = strlen(s);
	if (*cap == 0) {
		*cap = 256;
		*buf = (char *)malloc(*cap);
		if (!*buf) {
			*cap = 0;
			return;
		}
		(*buf)[0] = '\0';
	}
	if (*len + n + 1 > *cap) {
		size_t next = (*cap) * 2;
		while (*len + n + 1 > next) {
			next *= 2;
		}
		char *tmp = (char *)realloc(*buf, next);
		if (!tmp) {
			return;
		}
		*buf = tmp;
		*cap = next;
	}
	memcpy(*buf + *len, s, n);
	*len += n;
	(*buf)[*len] = '\0';
}

Result_string_Error __std_list_dir(char *path) {
	if (!path) {
		return err_string("list_dir: null path");
	}
	size_t cap = 256;
	size_t len = 0;
	char *out = (char *)malloc(cap);
	if (!out) {
		return err_string("list_dir: out of memory");
	}
	out[0] = '\0';
#ifdef _WIN32
	char pattern[MAX_PATH];
	snprintf(pattern, MAX_PATH, "%s\\*", path);
	WIN32_FIND_DATAA data;
	HANDLE h = FindFirstFileA(pattern, &data);
	if (h == INVALID_HANDLE_VALUE) {
		free(out);
		return err_string(strerror(errno));
	}
	do {
		const char *name = data.cFileName;
		if (strcmp(name, ".") == 0 || strcmp(name, "..") == 0) {
			continue;
		}
		if (len > 0) {
			append_str(&out, &cap, &len, "\n");
		}
		append_str(&out, &cap, &len, name);
	} while (FindNextFileA(h, &data));
	FindClose(h);
#else
	DIR *dir = opendir(path);
	if (!dir) {
		free(out);
		return err_string(strerror(errno));
	}
	struct dirent *ent;
	while ((ent = readdir(dir)) != NULL) {
		const char *name = ent->d_name;
		if (strcmp(name, ".") == 0 || strcmp(name, "..") == 0) {
			continue;
		}
		if (len > 0) {
			append_str(&out, &cap, &len, "\n");
		}
		append_str(&out, &cap, &len, name);
	}
	closedir(dir);
#endif
	return ok_string_owned(out);
}

int64_t __std_unix_millis(void) {
#ifdef _WIN32
	FILETIME ft;
	ULARGE_INTEGER ui;
	GetSystemTimeAsFileTime(&ft);
	ui.LowPart = ft.dwLowDateTime;
	ui.HighPart = ft.dwHighDateTime;
	uint64_t t = ui.QuadPart;
	uint64_t unix100ns = t - 116444736000000000ULL;
	return (int64_t)(unix100ns / 10000ULL);
#else
	struct timespec ts;
	clock_gettime(CLOCK_REALTIME, &ts);
	return (int64_t)ts.tv_sec * 1000LL + (int64_t)(ts.tv_nsec / 1000000LL);
#endif
}

void __std_sleep_ms(int64_t ms) {
	if (ms <= 0) {
		return;
	}
#ifdef _WIN32
	Sleep((DWORD)ms);
#else
	usleep((useconds_t)(ms * 1000));
#endif
}

char *__std_now_rfc3339(void) {
	time_t now = time(NULL);
	struct tm tmv;
#ifdef _WIN32
	gmtime_s(&tmv, &now);
#else
	gmtime_r(&now, &tmv);
#endif
	char buf[32];
	snprintf(buf, sizeof(buf), "%04d-%02d-%02dT%02d:%02d:%02dZ",
		tmv.tm_year + 1900, tmv.tm_mon + 1, tmv.tm_mday,
		tmv.tm_hour, tmv.tm_min, tmv.tm_sec);
	return bazic_strdup(buf);
}

static time_t bazic_timegm(struct tm *tm) {
#ifdef _WIN32
	return _mkgmtime(tm);
#else
	return timegm(tm);
#endif
}

static void bazic_gmtime_r(time_t t, struct tm *out) {
#ifdef _WIN32
	gmtime_s(out, &t);
#else
	gmtime_r(&t, out);
#endif
}

Result_string_Error __std_time_add_days(char *rfc3339, int64_t days) {
	int year, month, day, hour, minute, second;
	if (sscanf(rfc3339, "%4d-%2d-%2dT%2d:%2d:%2dZ", &year, &month, &day, &hour, &minute, &second) != 6) {
		return err_string("time_add_days: invalid format");
	}
	struct tm tmv;
	memset(&tmv, 0, sizeof(tmv));
	tmv.tm_year = year - 1900;
	tmv.tm_mon = month - 1;
	tmv.tm_mday = day;
	tmv.tm_hour = hour;
	tmv.tm_min = minute;
	tmv.tm_sec = second;
	tmv.tm_isdst = 0;

	time_t ts = bazic_timegm(&tmv);
	if (ts == (time_t)-1) {
		return err_string("time_add_days: conversion failed");
	}
	if (days != 0) {
		const int64_t delta = days * 86400LL;
		ts += delta;
	}
	struct tm out_tm;
	bazic_gmtime_r(ts, &out_tm);

	char buf[32];
	if (strftime(buf, sizeof(buf), "%Y-%m-%dT%H:%M:%SZ", &out_tm) == 0) {
		return err_string("time_add_days: format failed");
	}
	return ok_string(buf);
}

char *__std_json_escape(char *s) {
	if (!s) {
		return bazic_strdup("");
	}
	size_t n = strlen(s);
	size_t cap = n * 6 + 1;
	char *out = (char *)malloc(cap);
	if (!out) {
		return bazic_strdup("");
	}
	size_t j = 0;
	for (size_t i = 0; i < n; i++) {
		unsigned char c = (unsigned char)s[i];
		switch (c) {
		case '\\':
			out[j++] = '\\'; out[j++] = '\\';
			break;
		case '\"':
			out[j++] = '\\'; out[j++] = '\"';
			break;
		case '\b':
			out[j++] = '\\'; out[j++] = 'b';
			break;
		case '\f':
			out[j++] = '\\'; out[j++] = 'f';
			break;
		case '\n':
			out[j++] = '\\'; out[j++] = 'n';
			break;
		case '\r':
			out[j++] = '\\'; out[j++] = 'r';
			break;
		case '\t':
			out[j++] = '\\'; out[j++] = 't';
			break;
		default:
			if (c < 0x20) {
				snprintf(out + j, 7, "\\u%04X", (unsigned)c);
				j += 6;
			} else {
				out[j++] = (char)c;
			}
		}
	}
	out[j] = '\0';
	return out;
}

Result_string_Error __std_json_pretty(char *s) {
	if (!s) {
		return err_string("json_pretty: null input");
	}
	size_t vi = 0;
	if (!json_validate(s, &vi)) {
		return err_string("json_pretty: invalid json");
	}
	while (s[vi] == ' ' || s[vi] == '\t' || s[vi] == '\n' || s[vi] == '\r') {
		vi++;
	}
	if (s[vi] != '\0') {
		return err_string("json_pretty: invalid json");
	}
	size_t n = strlen(s);
	size_t cap = n * 2 + 64;
	char *out = (char *)malloc(cap);
	if (!out) {
		return err_string("json_pretty: out of memory");
	}
	size_t len = 0;
	int indent = 0;
	bool in_string = false;
	bool escape = false;
	bool need_indent = false;

	for (size_t i = 0; i < n; i++) {
		char c = s[i];
		if (escape) {
			escape = false;
			if (len + 1 >= cap) {
				cap *= 2;
				char *tmp = (char *)realloc(out, cap);
				if (!tmp) {
					free(out);
					return err_string("json_pretty: out of memory");
				}
				out = tmp;
			}
			out[len++] = c;
			continue;
		}
		if (c == '\\' && in_string) {
			escape = true;
			if (len + 1 >= cap) {
				cap *= 2;
				char *tmp = (char *)realloc(out, cap);
				if (!tmp) {
					free(out);
					return err_string("json_pretty: out of memory");
				}
				out = tmp;
			}
			out[len++] = c;
			continue;
		}
		if (c == '"' ) {
			in_string = !in_string;
			if (len + 1 >= cap) {
				cap *= 2;
				char *tmp = (char *)realloc(out, cap);
				if (!tmp) {
					free(out);
					return err_string("json_pretty: out of memory");
				}
				out = tmp;
			}
			out[len++] = c;
			continue;
		}
		if (in_string) {
			if (len + 1 >= cap) {
				cap *= 2;
				char *tmp = (char *)realloc(out, cap);
				if (!tmp) {
					free(out);
					return err_string("json_pretty: out of memory");
				}
				out = tmp;
			}
			out[len++] = c;
			continue;
		}

		if (c == ' ' || c == '\t' || c == '\n' || c == '\r') {
			continue;
		}
		if (need_indent) {
			for (int k = 0; k < indent; k++) {
				if (len + 1 >= cap) {
					cap *= 2;
					char *tmp = (char *)realloc(out, cap);
					if (!tmp) {
						free(out);
						return err_string("json_pretty: out of memory");
					}
					out = tmp;
				}
				out[len++] = ' ';
			}
			need_indent = false;
		}

		switch (c) {
		case '{':
		case '[':
			if (len + 1 >= cap) {
				cap *= 2;
				char *tmp = (char *)realloc(out, cap);
				if (!tmp) {
					free(out);
					return err_string("json_pretty: out of memory");
				}
				out = tmp;
			}
			out[len++] = c;
			if (len + 1 >= cap) {
				cap *= 2;
				char *tmp = (char *)realloc(out, cap);
				if (!tmp) {
					free(out);
					return err_string("json_pretty: out of memory");
				}
				out = tmp;
			}
			out[len++] = '\n';
			indent += 2;
			need_indent = true;
			break;
		case '}':
		case ']':
			if (len + 1 >= cap) {
				cap *= 2;
				char *tmp = (char *)realloc(out, cap);
				if (!tmp) {
					free(out);
					return err_string("json_pretty: out of memory");
				}
				out = tmp;
			}
			out[len++] = '\n';
			if (indent >= 2) {
				indent -= 2;
			}
			for (int k = 0; k < indent; k++) {
				if (len + 1 >= cap) {
					cap *= 2;
					char *tmp = (char *)realloc(out, cap);
					if (!tmp) {
						free(out);
						return err_string("json_pretty: out of memory");
					}
					out = tmp;
				}
				out[len++] = ' ';
			}
			out[len++] = c;
			break;
		case ',':
			if (len + 1 >= cap) {
				cap *= 2;
				char *tmp = (char *)realloc(out, cap);
				if (!tmp) {
					free(out);
					return err_string("json_pretty: out of memory");
				}
				out = tmp;
			}
			out[len++] = c;
			if (len + 1 >= cap) {
				cap *= 2;
				char *tmp = (char *)realloc(out, cap);
				if (!tmp) {
					free(out);
					return err_string("json_pretty: out of memory");
				}
				out = tmp;
			}
			out[len++] = '\n';
			need_indent = true;
			break;
		case ':':
			if (len + 2 >= cap) {
				cap *= 2;
				char *tmp = (char *)realloc(out, cap);
				if (!tmp) {
					free(out);
					return err_string("json_pretty: out of memory");
				}
				out = tmp;
			}
			out[len++] = c;
			out[len++] = ' ';
			break;
		default:
			if (len + 1 >= cap) {
				cap *= 2;
				char *tmp = (char *)realloc(out, cap);
				if (!tmp) {
					free(out);
					return err_string("json_pretty: out of memory");
				}
				out = tmp;
			}
			out[len++] = c;
			break;
		}
	}
	out[len] = '\0';
	return ok_string_owned(out);
}

bool __std_json_validate(char *s) {
	if (!s) {
		return false;
	}
	size_t vi = 0;
	if (!json_validate(s, &vi)) {
		return false;
	}
	while (s[vi] == ' ' || s[vi] == '\t' || s[vi] == '\n' || s[vi] == '\r') {
		vi++;
	}
	return s[vi] == '\0';
}

Result_string_Error __std_json_minify(char *s) {
	if (!s) {
		return err_string("json_minify: null input");
	}
	if (!__std_json_validate(s)) {
		return err_string("json_minify: invalid json");
	}
	size_t n = strlen(s);
	size_t cap = n + 1;
	char *out = (char *)malloc(cap);
	if (!out) {
		return err_string("json_minify: out of memory");
	}
	size_t len = 0;
	bool in_string = false;
	bool escape = false;
	for (size_t i = 0; i < n; i++) {
		char c = s[i];
		if (escape) {
			escape = false;
			out[len++] = c;
			continue;
		}
		if (c == '\\' && in_string) {
			escape = true;
			out[len++] = c;
			continue;
		}
		if (c == '"') {
			in_string = !in_string;
			out[len++] = c;
			continue;
		}
		if (!in_string) {
			if (c == ' ' || c == '\t' || c == '\n' || c == '\r') {
				continue;
			}
		}
		out[len++] = c;
	}
	out[len] = '\0';
	return ok_string_owned(out);
}

static bool json_find_value_range(const char *s, const char *key, size_t *val_start, size_t *val_end);
static bool json_decode_string_at(const char *s, size_t *i, char **out);
static void json_trim_ws(const char *s, size_t *start, size_t *end);
static bool json_parse_int64_simple(const char *s, size_t len, int64_t *out);
static bool json_parse_float_simple(const char *s, size_t len, double *out);

typedef struct BazicSessionEntry {
	char *token;
	char *user;
	time_t expires;
	struct BazicSessionEntry *next;
} BazicSessionEntry;

static BazicSessionEntry *bazic_session_head = NULL;

Result_string_Error __std_json_get_raw(char *s, char *key) {
	if (!s || !key) {
		return err_string("json_get_raw: null input");
	}
	size_t vs = 0;
	size_t ve = 0;
	if (!json_find_value_range(s, key, &vs, &ve)) {
		return err_string("key not found");
	}
	if (ve <= vs) {
		return err_string("json_get_raw: invalid value");
	}
	size_t n = ve - vs;
	char *raw = (char *)malloc(n + 1);
	if (!raw) {
		return err_string("json_get_raw: out of memory");
	}
	memcpy(raw, s + vs, n);
	raw[n] = '\0';
	Result_string_Error min = __std_json_minify(raw);
	free(raw);
	if (!min.is_ok) {
		return min;
	}
	return min;
}

Result_string_Error __std_json_get_string(char *s, char *key) {
	if (!s || !key) {
		return err_string("json_get_string: null input");
	}
	Result_string_Error raw = __std_json_get_raw(s, key);
	if (!raw.is_ok) {
		return err_string(raw.err.message);
	}
	size_t vs = 0;
	size_t ve = strlen(raw.value);
	json_trim_ws(raw.value, &vs, &ve);
	if (raw.value[vs] != '"') {
		return err_string("not a string");
	}
	size_t i = vs;
	char *out = NULL;
	if (!json_decode_string_at(raw.value, &i, &out)) {
		return err_string("json_get_string: invalid string");
	}
	if (!out) {
		return err_string("json_get_string: out of memory");
	}
	return ok_string_owned(out);
}

Result_bool_Error __std_json_get_bool(char *s, char *key) {
	if (!s || !key) {
		return err_bool("json_get_bool: null input");
	}
	Result_string_Error raw = __std_json_get_raw(s, key);
	if (!raw.is_ok) {
		return err_bool(raw.err.message);
	}
	size_t vs = 0;
	size_t ve = strlen(raw.value);
	json_trim_ws(raw.value, &vs, &ve);
	size_t n = ve - vs;
	if (n == 4 && strncmp(raw.value + vs, "true", 4) == 0) {
		return ok_bool(true);
	}
	if (n == 5 && strncmp(raw.value + vs, "false", 5) == 0) {
		return ok_bool(false);
	}
	return err_bool("not a bool");
}

Result_int_Error __std_json_get_int(char *s, char *key) {
	if (!s || !key) {
		return err_int("json_get_int: null input");
	}
	Result_string_Error raw = __std_json_get_raw(s, key);
	if (!raw.is_ok) {
		return err_int(raw.err.message);
	}
	size_t vs = 0;
	size_t ve = strlen(raw.value);
	json_trim_ws(raw.value, &vs, &ve);
	for (size_t i = vs; i < ve; i++) {
		char c = raw.value[i];
		if (c == '.' || c == 'e' || c == 'E') {
			return err_int("not an int");
		}
	}
	int64_t v = 0;
	if (!json_parse_int64_simple(raw.value + vs, ve - vs, &v)) {
		return err_int("not an int");
	}
	return ok_int(v);
}

Result_float_Error __std_json_get_float(char *s, char *key) {
	if (!s || !key) {
		return err_float("json_get_float: null input");
	}
	Result_string_Error raw = __std_json_get_raw(s, key);
	if (!raw.is_ok) {
		return err_float(raw.err.message);
	}
	size_t vs = 0;
	size_t ve = strlen(raw.value);
	json_trim_ws(raw.value, &vs, &ve);
	double v = 0.0;
	if (!json_parse_float_simple(raw.value + vs, ve - vs, &v)) {
		return err_float("not a float");
	}
	return ok_float(v);
}

static void json_skip_ws(const char *s, size_t *i) {
	while (s[*i] == ' ' || s[*i] == '\t' || s[*i] == '\n' || s[*i] == '\r') {
		(*i)++;
	}
}

static void json_trim_ws(const char *s, size_t *start, size_t *end) {
	while (*start < *end && (s[*start] == ' ' || s[*start] == '\t' || s[*start] == '\n' || s[*start] == '\r')) {
		(*start)++;
	}
	while (*end > *start && (s[*end - 1] == ' ' || s[*end - 1] == '\t' || s[*end - 1] == '\n' || s[*end - 1] == '\r')) {
		(*end)--;
	}
}

static bool json_parse_int64_simple(const char *s, size_t len, int64_t *out) {
	if (!s || len == 0) {
		return false;
	}
	int sign = 1;
	size_t i = 0;
	if (s[i] == '-') {
		sign = -1;
		i++;
	}
	if (i >= len) {
		return false;
	}
	int64_t v = 0;
	for (; i < len; i++) {
		char c = s[i];
		if (c < '0' || c > '9') {
			return false;
		}
		int64_t digit = (int64_t)(c - '0');
		if (v > (INT64_MAX - digit) / 10) {
			return false;
		}
		v = v * 10 + digit;
	}
	*out = (sign < 0) ? -v : v;
	return true;
}

static double json_pow10_int(int exp) {
	double result = 1.0;
	double base = 10.0;
	int e = exp < 0 ? -exp : exp;
	while (e > 0) {
		if (e & 1) {
			result *= base;
		}
		base *= base;
		e >>= 1;
	}
	if (exp < 0) {
		return 1.0 / result;
	}
	return result;
}

static bool json_parse_float_simple(const char *s, size_t len, double *out) {
	if (!s || len == 0) {
		return false;
	}
	size_t i = 0;
	int sign = 1;
	if (s[i] == '-') {
		sign = -1;
		i++;
	}
	if (i >= len) {
		return false;
	}
	double v = 0.0;
	int digits = 0;
	while (i < len && s[i] >= '0' && s[i] <= '9') {
		v = v * 10.0 + (double)(s[i] - '0');
		i++;
		digits++;
	}
	int fracDigits = 0;
	if (i < len && s[i] == '.') {
		i++;
		while (i < len && s[i] >= '0' && s[i] <= '9') {
			v = v * 10.0 + (double)(s[i] - '0');
			i++;
			fracDigits++;
		}
	}
	if (digits == 0 && fracDigits == 0) {
		return false;
	}
	int exp = 0;
	if (i < len && (s[i] == 'e' || s[i] == 'E')) {
		i++;
		int expSign = 1;
		if (i < len && (s[i] == '+' || s[i] == '-')) {
			if (s[i] == '-') expSign = -1;
			i++;
		}
		if (i >= len || s[i] < '0' || s[i] > '9') {
			return false;
		}
		while (i < len && s[i] >= '0' && s[i] <= '9') {
			exp = exp * 10 + (int)(s[i] - '0');
			i++;
		}
		exp *= expSign;
	}
	if (i != len) {
		return false;
	}
	exp -= fracDigits;
	v = v * json_pow10_int(exp);
	*out = (sign < 0) ? -v : v;
	return true;
}

static bool json_parse_string(const char *s, size_t *i) {
	if (s[*i] != '"') {
		return false;
	}
	(*i)++;
	for (;;) {
		char c = s[*i];
		if (c == '\0') {
			return false;
		}
		if (c == '"') {
			(*i)++;
			return true;
		}
		if ((unsigned char)c < 0x20) {
			return false;
		}
		if (c == '\\') {
			(*i)++;
			char e = s[*i];
			if (e == '\0') {
				return false;
			}
			switch (e) {
			case '"':
			case '\\':
			case '/':
			case 'b':
			case 'f':
			case 'n':
			case 'r':
			case 't':
				(*i)++;
				break;
			case 'u':
				(*i)++;
				for (int k = 0; k < 4; k++) {
					char h = s[*i];
					if (!((h >= '0' && h <= '9') || (h >= 'a' && h <= 'f') || (h >= 'A' && h <= 'F'))) {
						return false;
					}
					(*i)++;
				}
				break;
			default:
				return false;
			}
			continue;
		}
		(*i)++;
	}
}

static bool json_parse_number(const char *s, size_t *i) {
	size_t start = *i;
	if (s[*i] == '-') {
		(*i)++;
	}
	if (s[*i] == '0') {
		(*i)++;
	} else if (s[*i] >= '1' && s[*i] <= '9') {
		while (s[*i] >= '0' && s[*i] <= '9') {
			(*i)++;
		}
	} else {
		return false;
	}
	if (s[*i] == '.') {
		(*i)++;
		if (!(s[*i] >= '0' && s[*i] <= '9')) {
			return false;
		}
		while (s[*i] >= '0' && s[*i] <= '9') {
			(*i)++;
		}
	}
	if (s[*i] == 'e' || s[*i] == 'E') {
		(*i)++;
		if (s[*i] == '+' || s[*i] == '-') {
			(*i)++;
		}
		if (!(s[*i] >= '0' && s[*i] <= '9')) {
			return false;
		}
		while (s[*i] >= '0' && s[*i] <= '9') {
			(*i)++;
		}
	}
	return *i > start;
}

static bool json_parse_value(const char *s, size_t *i);

static bool json_parse_array(const char *s, size_t *i) {
	if (s[*i] != '[') {
		return false;
	}
	(*i)++;
	json_skip_ws(s, i);
	if (s[*i] == ']') {
		(*i)++;
		return true;
	}
	for (;;) {
		if (!json_parse_value(s, i)) {
			return false;
		}
		json_skip_ws(s, i);
		if (s[*i] == ']') {
			(*i)++;
			return true;
		}
		if (s[*i] != ',') {
			return false;
		}
		(*i)++;
		json_skip_ws(s, i);
	}
}

static bool json_parse_object(const char *s, size_t *i) {
	if (s[*i] != '{') {
		return false;
	}
	(*i)++;
	json_skip_ws(s, i);
	if (s[*i] == '}') {
		(*i)++;
		return true;
	}
	for (;;) {
		if (!json_parse_string(s, i)) {
			return false;
		}
		json_skip_ws(s, i);
		if (s[*i] != ':') {
			return false;
		}
		(*i)++;
		json_skip_ws(s, i);
		if (!json_parse_value(s, i)) {
			return false;
		}
		json_skip_ws(s, i);
		if (s[*i] == '}') {
			(*i)++;
			return true;
		}
		if (s[*i] != ',') {
			return false;
		}
		(*i)++;
		json_skip_ws(s, i);
	}
}

static bool json_parse_value(const char *s, size_t *i) {
	json_skip_ws(s, i);
	char c = s[*i];
	if (c == '"') {
		return json_parse_string(s, i);
	}
	if (c == '{') {
		return json_parse_object(s, i);
	}
	if (c == '[') {
		return json_parse_array(s, i);
	}
	if (c == '-' || (c >= '0' && c <= '9')) {
		return json_parse_number(s, i);
	}
	if (strncmp(&s[*i], "true", 4) == 0) {
		*i += 4;
		return true;
	}
	if (strncmp(&s[*i], "false", 5) == 0) {
		*i += 5;
		return true;
	}
	if (strncmp(&s[*i], "null", 4) == 0) {
		*i += 4;
		return true;
	}
	return false;
}

static bool json_decode_string_at(const char *s, size_t *i, char **out) {
	if (s[*i] != '"') {
		return false;
	}
	(*i)++;
	size_t cap = 16;
	size_t len = 0;
	char *buf = (char *)malloc(cap);
	if (!buf) {
		return false;
	}
	for (;;) {
		char c = s[*i];
		if (c == '\0') {
			free(buf);
			return false;
		}
		if (c == '"') {
			(*i)++;
			buf[len] = '\0';
			*out = buf;
			return true;
		}
		if ((unsigned char)c < 0x20) {
			free(buf);
			return false;
		}
		if (c == '\\') {
			(*i)++;
			char e = s[*i];
			if (e == '\0') {
				free(buf);
				return false;
			}
			char outc = 0;
			switch (e) {
			case '"': outc = '"'; break;
			case '\\': outc = '\\'; break;
			case '/': outc = '/'; break;
			case 'b': outc = '\b'; break;
			case 'f': outc = '\f'; break;
			case 'n': outc = '\n'; break;
			case 'r': outc = '\r'; break;
			case 't': outc = '\t'; break;
			case 'u': {
				(*i)++;
				int code = 0;
				for (int k = 0; k < 4; k++) {
					char h = s[*i];
					int v = 0;
					if (h >= '0' && h <= '9') v = h - '0';
					else if (h >= 'a' && h <= 'f') v = 10 + (h - 'a');
					else if (h >= 'A' && h <= 'F') v = 10 + (h - 'A');
					else { free(buf); return false; }
					code = (code << 4) | v;
					(*i)++;
				}
				if (code >= 0xD800 && code <= 0xDBFF) {
					size_t save = *i;
					if (s[*i] == '\\' && s[*i + 1] == 'u') {
						*i += 2;
						int code2 = 0;
						for (int k = 0; k < 4; k++) {
							char h = s[*i];
							int v = 0;
							if (h >= '0' && h <= '9') v = h - '0';
							else if (h >= 'a' && h <= 'f') v = 10 + (h - 'a');
							else if (h >= 'A' && h <= 'F') v = 10 + (h - 'A');
							else { free(buf); return false; }
							code2 = (code2 << 4) | v;
							(*i)++;
						}
						if (code2 >= 0xDC00 && code2 <= 0xDFFF) {
							code = 0x10000 + (((code - 0xD800) << 10) | (code2 - 0xDC00));
						} else {
							*i = save;
						}
					} else {
						*i = save;
					}
				}
				char tmp[4];
				int tlen = 0;
				if (code <= 0x7F) {
					tmp[tlen++] = (char)code;
				} else if (code <= 0x7FF) {
					tmp[tlen++] = (char)(0xC0 | ((code >> 6) & 0x1F));
					tmp[tlen++] = (char)(0x80 | (code & 0x3F));
				} else if (code <= 0xFFFF) {
					tmp[tlen++] = (char)(0xE0 | ((code >> 12) & 0x0F));
					tmp[tlen++] = (char)(0x80 | ((code >> 6) & 0x3F));
					tmp[tlen++] = (char)(0x80 | (code & 0x3F));
				} else {
					tmp[tlen++] = (char)(0xF0 | ((code >> 18) & 0x07));
					tmp[tlen++] = (char)(0x80 | ((code >> 12) & 0x3F));
					tmp[tlen++] = (char)(0x80 | ((code >> 6) & 0x3F));
					tmp[tlen++] = (char)(0x80 | (code & 0x3F));
				}
				while (len + (size_t)tlen + 1 > cap) {
					cap *= 2;
					char *nb = (char *)realloc(buf, cap);
					if (!nb) { free(buf); return false; }
					buf = nb;
				}
				for (int k = 0; k < tlen; k++) {
					buf[len++] = tmp[k];
				}
				continue;
			}
			default:
				free(buf);
				return false;
			}
			(*i)++;
			if (len + 2 > cap) {
				cap *= 2;
				char *nb = (char *)realloc(buf, cap);
				if (!nb) { free(buf); return false; }
				buf = nb;
			}
			buf[len++] = outc;
			continue;
		}
		if (len + 2 > cap) {
			cap *= 2;
			char *nb = (char *)realloc(buf, cap);
			if (!nb) { free(buf); return false; }
			buf = nb;
		}
		buf[len++] = c;
		(*i)++;
	}
}

static bool json_find_value_range(const char *s, const char *key, size_t *val_start, size_t *val_end) {
	if (!s || !key) {
		return false;
	}
	size_t i = 0;
	json_skip_ws(s, &i);
	if (s[i] != '{') {
		return false;
	}
	i++;
	json_skip_ws(s, &i);
	if (s[i] == '}') {
		return false;
	}
	for (;;) {
		if (s[i] != '"') {
			return false;
		}
		char *decoded = NULL;
		if (!json_decode_string_at(s, &i, &decoded)) {
			free(decoded);
			return false;
		}
		bool match = (decoded && strcmp(decoded, key) == 0);
		free(decoded);
		json_skip_ws(s, &i);
		if (s[i] != ':') {
			return false;
		}
		i++;
		json_skip_ws(s, &i);
		size_t vstart = i;
		if (!json_parse_value(s, &i)) {
			return false;
		}
		size_t vend = i;
		if (match) {
			*val_start = vstart;
			*val_end = vend;
			return true;
		}
		json_skip_ws(s, &i);
		if (s[i] == ',') {
			i++;
			json_skip_ws(s, &i);
			continue;
		}
		if (s[i] == '}') {
			return false;
		}
		return false;
	}
}

static bool json_validate(const char *s, size_t *i) {
	if (!s) {
		return false;
	}
	return json_parse_value(s, i);
}

Result_string_Error __std_http_get(char *url) {
	return __std_http_get_opts(url, 5000, 15000, "", "Bazic/1.0", false, "");
}

Result_string_Error __std_http_get_opts(char *url, int64_t connect_timeout_ms, int64_t timeout_ms, char *hdrs, char *user_agent, bool tls_insecure, char *ca_bundle_pem) {
#ifdef _WIN32
	if (!url) {
		return err_string("http_get: null url");
	}
	if (ca_bundle_pem && ca_bundle_pem[0] != '\0') {
		return err_string("http_get: custom CA bundle not supported on Windows");
	}
	wchar_t *wurl = utf8_to_wide(url);
	if (!wurl) {
		return err_string("http_get: url conversion failed");
	}
	URL_COMPONENTS parts;
	memset(&parts, 0, sizeof(parts));
	parts.dwStructSize = sizeof(parts);
	parts.dwSchemeLength = (DWORD)-1;
	parts.dwHostNameLength = (DWORD)-1;
	parts.dwUrlPathLength = (DWORD)-1;
	parts.dwExtraInfoLength = (DWORD)-1;
	if (!WinHttpCrackUrl(wurl, 0, 0, &parts)) {
		free(wurl);
		return err_string("http_get: invalid url");
	}
	const char *ua = (user_agent && user_agent[0] != '\0') ? user_agent : "Bazic/1.0";
	wchar_t *wua = utf8_to_wide(ua);
	if (!wua) {
		free(wurl);
		return err_string("http_get: user agent conversion failed");
	}
	HINTERNET hSession = WinHttpOpen(wua, WINHTTP_ACCESS_TYPE_DEFAULT_PROXY, NULL, NULL, 0);
	if (!hSession) {
		free(wurl);
		free(wua);
		char *msg = win_last_error();
		Result_string_Error r = err_string(msg);
		free(msg);
		return r;
	}
	HINTERNET hConnect = WinHttpConnect(hSession, parts.lpszHostName, parts.nPort, 0);
	if (!hConnect) {
		char *msg = win_last_error();
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		Result_string_Error r = err_string(msg);
		free(msg);
		return r;
	}
	DWORD flags = 0;
	if (parts.nScheme == INTERNET_SCHEME_HTTPS) {
		flags |= WINHTTP_FLAG_SECURE;
	}
	wchar_t path[2048];
	path[0] = L'\0';
	if (parts.lpszUrlPath && parts.dwUrlPathLength > 0) {
		wcsncat(path, parts.lpszUrlPath, parts.dwUrlPathLength);
	}
	if (parts.lpszExtraInfo && parts.dwExtraInfoLength > 0) {
		wcsncat(path, parts.lpszExtraInfo, parts.dwExtraInfoLength);
	}
	if (wcslen(path) == 0) {
		wcscpy(path, L"/");
	}
	HINTERNET hRequest = WinHttpOpenRequest(hConnect, L"GET", path, NULL, WINHTTP_NO_REFERER, WINHTTP_DEFAULT_ACCEPT_TYPES, flags);
	if (!hRequest) {
		char *msg = win_last_error();
		WinHttpCloseHandle(hConnect);
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		Result_string_Error r = err_string(msg);
		free(msg);
		return r;
	}
	int ct = (int)(connect_timeout_ms > 0 ? connect_timeout_ms : 5000);
	int tt = (int)(timeout_ms > 0 ? timeout_ms : 15000);
	WinHttpSetTimeouts(hRequest, ct, ct, tt, tt);
	if (tls_insecure) {
		DWORD flags = SECURITY_FLAG_IGNORE_CERT_CN_INVALID |
			SECURITY_FLAG_IGNORE_CERT_DATE_INVALID |
			SECURITY_FLAG_IGNORE_UNKNOWN_CA |
			SECURITY_FLAG_IGNORE_CERT_WRONG_USAGE;
		WinHttpSetOption(hRequest, WINHTTP_OPTION_SECURITY_FLAGS, &flags, sizeof(flags));
	}
	char *hdrBlock = build_header_block(hdrs, "*/*", "");
	if (hdrBlock) {
		wchar_t *whdr = utf8_to_wide(hdrBlock);
		if (whdr) {
			WinHttpAddRequestHeaders(hRequest, whdr, (DWORD)-1L, WINHTTP_ADDREQ_FLAG_ADD);
			free(whdr);
		}
		free(hdrBlock);
	}
	BOOL ok = WinHttpSendRequest(hRequest, WINHTTP_NO_ADDITIONAL_HEADERS, 0, WINHTTP_NO_REQUEST_DATA, 0, 0, 0);
	if (!ok || !WinHttpReceiveResponse(hRequest, NULL)) {
		char *msg = win_last_error();
		WinHttpCloseHandle(hRequest);
		WinHttpCloseHandle(hConnect);
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		Result_string_Error r = err_string(msg);
		free(msg);
		return r;
	}
	size_t cap = 4096;
	size_t len = 0;
	char *buf = (char *)malloc(cap);
	if (!buf) {
		WinHttpCloseHandle(hRequest);
		WinHttpCloseHandle(hConnect);
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		return err_string("http_get: out of memory");
	}
	buf[0] = '\0';
	for (;;) {
		DWORD available = 0;
		if (!WinHttpQueryDataAvailable(hRequest, &available)) {
			char *msg = win_last_error();
			free(buf);
			WinHttpCloseHandle(hRequest);
			WinHttpCloseHandle(hConnect);
			WinHttpCloseHandle(hSession);
			free(wurl);
			free(wua);
			Result_string_Error r = err_string(msg);
			free(msg);
			return r;
		}
		if (available == 0) {
			break;
		}
		if (len + available + 1 > cap) {
			size_t next = cap * 2;
			while (len + available + 1 > next) {
				next *= 2;
			}
			char *tmp = (char *)realloc(buf, next);
			if (!tmp) {
				free(buf);
				WinHttpCloseHandle(hRequest);
				WinHttpCloseHandle(hConnect);
				WinHttpCloseHandle(hSession);
				free(wurl);
				free(wua);
				return err_string("http_get: out of memory");
			}
			buf = tmp;
			cap = next;
		}
		DWORD read = 0;
		if (!WinHttpReadData(hRequest, buf + len, available, &read)) {
			char *msg = win_last_error();
			free(buf);
			WinHttpCloseHandle(hRequest);
			WinHttpCloseHandle(hConnect);
			WinHttpCloseHandle(hSession);
			free(wurl);
			free(wua);
			Result_string_Error r = err_string(msg);
			free(msg);
			return r;
		}
		len += read;
		buf[len] = '\0';
	}
	WinHttpCloseHandle(hRequest);
	WinHttpCloseHandle(hConnect);
	WinHttpCloseHandle(hSession);
	free(wurl);
	free(wua);
	return ok_string_owned(buf);
#else
	if (!url) {
		return err_string("http_get: null url");
	}
	static bool curl_inited = false;
	if (!curl_inited) {
		curl_global_init(CURL_GLOBAL_DEFAULT);
		curl_inited = true;
	}
	CURL *curl = curl_easy_init();
	if (!curl) {
		return err_string("http_get: curl init failed");
	}
	struct {
		char *buf;
		size_t len;
		size_t cap;
	} ctx = {0};
	curl_easy_setopt(curl, CURLOPT_URL, url);
	curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
	curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT_MS, (long)(connect_timeout_ms > 0 ? connect_timeout_ms : 5000));
	curl_easy_setopt(curl, CURLOPT_TIMEOUT_MS, (long)(timeout_ms > 0 ? timeout_ms : 15000));
	curl_easy_setopt(curl, CURLOPT_USERAGENT, (user_agent && user_agent[0] != '\0') ? user_agent : "Bazic/1.0");
	if (tls_insecure) {
		curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0L);
		curl_easy_setopt(curl, CURLOPT_SSL_VERIFYHOST, 0L);
	}
	char *ca_path = NULL;
	if (ca_bundle_pem && ca_bundle_pem[0] != '\0') {
		ca_path = write_temp_ca(ca_bundle_pem);
		if (!ca_path) {
			curl_easy_cleanup(curl);
			return err_string("http_get: failed to write CA bundle");
		}
		curl_easy_setopt(curl, CURLOPT_CAINFO, ca_path);
	}
	struct curl_slist *headers = NULL;
	if (hdrs && hdrs[0] != '\0') {
		char *copy = bazic_strdup(hdrs);
		char *line = strtok(copy, "\n");
		while (line) {
			while (*line == ' ' || *line == '\t') { line++; }
			if (*line != '\0') {
				headers = curl_slist_append(headers, line);
			}
			line = strtok(NULL, "\n");
		}
		free(copy);
	}
	headers = curl_slist_append(headers, "Accept: */*");
	curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
	curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, curl_write_cb);
	curl_easy_setopt(curl, CURLOPT_WRITEDATA, &ctx);
	CURLcode res = curl_easy_perform(curl);
	if (res != CURLE_OK) {
		const char *msg = curl_easy_strerror(res);
		curl_slist_free_all(headers);
		curl_easy_cleanup(curl);
		if (ca_path) {
			unlink(ca_path);
			free(ca_path);
		}
		if (ctx.buf) {
			free(ctx.buf);
		}
		return err_string(msg);
	}
	curl_slist_free_all(headers);
	curl_easy_cleanup(curl);
	if (ca_path) {
		unlink(ca_path);
		free(ca_path);
	}
	if (!ctx.buf) {
		return ok_string("");
	}
	return ok_string_owned(ctx.buf);
#endif
}

Result_string_Error __std_http_post(char *url, char *body) {
	return __std_http_post_opts(url, body, 5000, 15000, "", "Bazic/1.0", "text/plain; charset=utf-8", false, "");
}

Result_string_Error __std_http_post_opts(char *url, char *body, int64_t connect_timeout_ms, int64_t timeout_ms, char *hdrs, char *user_agent, char *content_type, bool tls_insecure, char *ca_bundle_pem) {
#ifdef _WIN32
	if (!url) {
		return err_string("http_post: null url");
	}
	if (!body) {
		body = "";
	}
	if (ca_bundle_pem && ca_bundle_pem[0] != '\0') {
		return err_string("http_post: custom CA bundle not supported on Windows");
	}
	wchar_t *wurl = utf8_to_wide(url);
	if (!wurl) {
		return err_string("http_post: url conversion failed");
	}
	URL_COMPONENTS parts;
	memset(&parts, 0, sizeof(parts));
	parts.dwStructSize = sizeof(parts);
	parts.dwSchemeLength = (DWORD)-1;
	parts.dwHostNameLength = (DWORD)-1;
	parts.dwUrlPathLength = (DWORD)-1;
	parts.dwExtraInfoLength = (DWORD)-1;
	if (!WinHttpCrackUrl(wurl, 0, 0, &parts)) {
		free(wurl);
		return err_string("http_post: invalid url");
	}
	const char *ua = (user_agent && user_agent[0] != '\0') ? user_agent : "Bazic/1.0";
	wchar_t *wua = utf8_to_wide(ua);
	if (!wua) {
		free(wurl);
		return err_string("http_post: user agent conversion failed");
	}
	HINTERNET hSession = WinHttpOpen(wua, WINHTTP_ACCESS_TYPE_DEFAULT_PROXY, NULL, NULL, 0);
	if (!hSession) {
		free(wurl);
		free(wua);
		char *msg = win_last_error();
		Result_string_Error r = err_string(msg);
		free(msg);
		return r;
	}
	HINTERNET hConnect = WinHttpConnect(hSession, parts.lpszHostName, parts.nPort, 0);
	if (!hConnect) {
		char *msg = win_last_error();
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		Result_string_Error r = err_string(msg);
		free(msg);
		return r;
	}
	DWORD flags = 0;
	if (parts.nScheme == INTERNET_SCHEME_HTTPS) {
		flags |= WINHTTP_FLAG_SECURE;
	}
	wchar_t path[2048];
	path[0] = L'\0';
	if (parts.lpszUrlPath && parts.dwUrlPathLength > 0) {
		wcsncat(path, parts.lpszUrlPath, parts.dwUrlPathLength);
	}
	if (parts.lpszExtraInfo && parts.dwExtraInfoLength > 0) {
		wcsncat(path, parts.lpszExtraInfo, parts.dwExtraInfoLength);
	}
	if (wcslen(path) == 0) {
		wcscpy(path, L"/");
	}
	HINTERNET hRequest = WinHttpOpenRequest(hConnect, L"POST", path, NULL, WINHTTP_NO_REFERER, WINHTTP_DEFAULT_ACCEPT_TYPES, flags);
	if (!hRequest) {
		char *msg = win_last_error();
		WinHttpCloseHandle(hConnect);
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		Result_string_Error r = err_string(msg);
		free(msg);
		return r;
	}
	int ct = (int)(connect_timeout_ms > 0 ? connect_timeout_ms : 5000);
	int tt = (int)(timeout_ms > 0 ? timeout_ms : 15000);
	WinHttpSetTimeouts(hRequest, ct, ct, tt, tt);
	if (tls_insecure) {
		DWORD flags = SECURITY_FLAG_IGNORE_CERT_CN_INVALID |
			SECURITY_FLAG_IGNORE_CERT_DATE_INVALID |
			SECURITY_FLAG_IGNORE_UNKNOWN_CA |
			SECURITY_FLAG_IGNORE_CERT_WRONG_USAGE;
		WinHttpSetOption(hRequest, WINHTTP_OPTION_SECURITY_FLAGS, &flags, sizeof(flags));
	}
	const char *ctHeader = (content_type && content_type[0] != '\0') ? content_type : "text/plain; charset=utf-8";
	char *hdrBlock = build_header_block(hdrs, "*/*", ctHeader);
	wchar_t *whdr = NULL;
	if (hdrBlock) {
		whdr = utf8_to_wide(hdrBlock);
	}
	DWORD bodyLen = (DWORD)strlen(body);
	BOOL ok = WinHttpSendRequest(hRequest, whdr ? whdr : WINHTTP_NO_ADDITIONAL_HEADERS, whdr ? (DWORD)-1L : 0, (LPVOID)body, bodyLen, bodyLen, 0);
	if (whdr) {
		free(whdr);
	}
	if (hdrBlock) {
		free(hdrBlock);
	}
	if (!ok || !WinHttpReceiveResponse(hRequest, NULL)) {
		char *msg = win_last_error();
		WinHttpCloseHandle(hRequest);
		WinHttpCloseHandle(hConnect);
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		Result_string_Error r = err_string(msg);
		free(msg);
		return r;
	}
	size_t cap = 4096;
	size_t len = 0;
	char *buf = (char *)malloc(cap);
	if (!buf) {
		WinHttpCloseHandle(hRequest);
		WinHttpCloseHandle(hConnect);
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		return err_string("http_post: out of memory");
	}
	buf[0] = '\0';
	for (;;) {
		DWORD available = 0;
		if (!WinHttpQueryDataAvailable(hRequest, &available)) {
			char *msg = win_last_error();
			free(buf);
			WinHttpCloseHandle(hRequest);
			WinHttpCloseHandle(hConnect);
			WinHttpCloseHandle(hSession);
			free(wurl);
			free(wua);
			Result_string_Error r = err_string(msg);
			free(msg);
			return r;
		}
		if (available == 0) {
			break;
		}
		if (len + available + 1 > cap) {
			size_t next = cap * 2;
			while (len + available + 1 > next) {
				next *= 2;
			}
			char *tmp = (char *)realloc(buf, next);
			if (!tmp) {
				free(buf);
				WinHttpCloseHandle(hRequest);
				WinHttpCloseHandle(hConnect);
				WinHttpCloseHandle(hSession);
				free(wurl);
				free(wua);
				return err_string("http_post: out of memory");
			}
			buf = tmp;
			cap = next;
		}
		DWORD read = 0;
		if (!WinHttpReadData(hRequest, buf + len, available, &read)) {
			char *msg = win_last_error();
			free(buf);
			WinHttpCloseHandle(hRequest);
			WinHttpCloseHandle(hConnect);
			WinHttpCloseHandle(hSession);
			free(wurl);
			free(wua);
			Result_string_Error r = err_string(msg);
			free(msg);
			return r;
		}
		len += read;
		buf[len] = '\0';
	}
	WinHttpCloseHandle(hRequest);
	WinHttpCloseHandle(hConnect);
	WinHttpCloseHandle(hSession);
	free(wurl);
	free(wua);
	return ok_string_owned(buf);
#else
	if (!url) {
		return err_string("http_post: null url");
	}
	if (!body) {
		body = "";
	}
	static bool curl_inited = false;
	if (!curl_inited) {
		curl_global_init(CURL_GLOBAL_DEFAULT);
		curl_inited = true;
	}
	CURL *curl = curl_easy_init();
	if (!curl) {
		return err_string("http_post: curl init failed");
	}
	struct {
		char *buf;
		size_t len;
		size_t cap;
	} ctx = {0};
	curl_easy_setopt(curl, CURLOPT_URL, url);
	curl_easy_setopt(curl, CURLOPT_POST, 1L);
	curl_easy_setopt(curl, CURLOPT_POSTFIELDS, body);
	curl_easy_setopt(curl, CURLOPT_POSTFIELDSIZE, (long)strlen(body));
	curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
	curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT_MS, (long)(connect_timeout_ms > 0 ? connect_timeout_ms : 5000));
	curl_easy_setopt(curl, CURLOPT_TIMEOUT_MS, (long)(timeout_ms > 0 ? timeout_ms : 15000));
	curl_easy_setopt(curl, CURLOPT_USERAGENT, (user_agent && user_agent[0] != '\0') ? user_agent : "Bazic/1.0");
	if (tls_insecure) {
		curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0L);
		curl_easy_setopt(curl, CURLOPT_SSL_VERIFYHOST, 0L);
	}
	char *ca_path = NULL;
	if (ca_bundle_pem && ca_bundle_pem[0] != '\0') {
		ca_path = write_temp_ca(ca_bundle_pem);
		if (!ca_path) {
			curl_easy_cleanup(curl);
			return err_string("http_post: failed to write CA bundle");
		}
		curl_easy_setopt(curl, CURLOPT_CAINFO, ca_path);
	}
	struct curl_slist *headers = NULL;
	if (hdrs && hdrs[0] != '\0') {
		char *copy = bazic_strdup(hdrs);
		char *line = strtok(copy, "\n");
		while (line) {
			while (*line == ' ' || *line == '\t') { line++; }
			if (*line != '\0') {
				headers = curl_slist_append(headers, line);
			}
			line = strtok(NULL, "\n");
		}
		free(copy);
	}
	if (content_type && content_type[0] != '\0') {
		char ctLine[256];
		snprintf(ctLine, sizeof(ctLine), "Content-Type: %s", content_type);
		headers = curl_slist_append(headers, ctLine);
	} else {
		headers = curl_slist_append(headers, "Content-Type: text/plain; charset=utf-8");
	}
	headers = curl_slist_append(headers, "Accept: */*");
	curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
	curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, curl_write_cb);
	curl_easy_setopt(curl, CURLOPT_WRITEDATA, &ctx);
	CURLcode res = curl_easy_perform(curl);
	if (res != CURLE_OK) {
		const char *msg = curl_easy_strerror(res);
		curl_slist_free_all(headers);
		curl_easy_cleanup(curl);
		if (ca_path) {
			unlink(ca_path);
			free(ca_path);
		}
		if (ctx.buf) {
			free(ctx.buf);
		}
		return err_string(msg);
	}
	curl_slist_free_all(headers);
	curl_easy_cleanup(curl);
	if (ca_path) {
		unlink(ca_path);
		free(ca_path);
	}
	if (!ctx.buf) {
		return ok_string("");
	}
	return ok_string_owned(ctx.buf);
#endif
}

Result_string_Error __std_http_request(char *method, char *url, char *body, int64_t connect_timeout_ms, int64_t timeout_ms, char *hdrs, char *user_agent, char *content_type, bool tls_insecure, char *ca_bundle_pem) {
	if (!method || method[0] == '\0') {
		method = "GET";
	}
	if (!url) {
		return err_string("http_request: null url");
	}
#ifdef _WIN32
	if (ca_bundle_pem && ca_bundle_pem[0] != '\0') {
		return err_string("http_request: custom CA bundle not supported on Windows");
	}
	wchar_t *wurl = utf8_to_wide(url);
	if (!wurl) {
		return err_string("http_request: url conversion failed");
	}
	URL_COMPONENTS parts;
	memset(&parts, 0, sizeof(parts));
	parts.dwStructSize = sizeof(parts);
	parts.dwSchemeLength = (DWORD)-1;
	parts.dwHostNameLength = (DWORD)-1;
	parts.dwUrlPathLength = (DWORD)-1;
	parts.dwExtraInfoLength = (DWORD)-1;
	if (!WinHttpCrackUrl(wurl, 0, 0, &parts)) {
		free(wurl);
		return err_string("http_request: invalid url");
	}
	const char *ua = (user_agent && user_agent[0] != '\0') ? user_agent : "Bazic/1.0";
	wchar_t *wua = utf8_to_wide(ua);
	if (!wua) {
		free(wurl);
		return err_string("http_request: user agent conversion failed");
	}
	HINTERNET hSession = WinHttpOpen(wua, WINHTTP_ACCESS_TYPE_DEFAULT_PROXY, NULL, NULL, 0);
	if (!hSession) {
		free(wurl);
		free(wua);
		char *msg = win_last_error();
		Result_string_Error r = err_string(msg);
		free(msg);
		return r;
	}
	HINTERNET hConnect = WinHttpConnect(hSession, parts.lpszHostName, parts.nPort, 0);
	if (!hConnect) {
		char *msg = win_last_error();
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		Result_string_Error r = err_string(msg);
		free(msg);
		return r;
	}
	DWORD flags = 0;
	if (parts.nScheme == INTERNET_SCHEME_HTTPS) {
		flags |= WINHTTP_FLAG_SECURE;
	}
	wchar_t path[2048];
	path[0] = L'\0';
	if (parts.lpszUrlPath && parts.dwUrlPathLength > 0) {
		wcsncat(path, parts.lpszUrlPath, parts.dwUrlPathLength);
	}
	if (parts.lpszExtraInfo && parts.dwExtraInfoLength > 0) {
		wcsncat(path, parts.lpszExtraInfo, parts.dwExtraInfoLength);
	}
	if (wcslen(path) == 0) {
		wcscpy(path, L"/");
	}
	wchar_t *wmethod = utf8_to_wide(method);
	if (!wmethod) {
		WinHttpCloseHandle(hConnect);
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		return err_string("http_request: method conversion failed");
	}
	HINTERNET hRequest = WinHttpOpenRequest(hConnect, wmethod, path, NULL, WINHTTP_NO_REFERER, WINHTTP_DEFAULT_ACCEPT_TYPES, flags);
	free(wmethod);
	if (!hRequest) {
		char *msg = win_last_error();
		WinHttpCloseHandle(hConnect);
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		Result_string_Error r = err_string(msg);
		free(msg);
		return r;
	}
	int ct = (int)(connect_timeout_ms > 0 ? connect_timeout_ms : 5000);
	int tt = (int)(timeout_ms > 0 ? timeout_ms : 15000);
	WinHttpSetTimeouts(hRequest, ct, ct, tt, tt);
	if (tls_insecure) {
		DWORD flags = SECURITY_FLAG_IGNORE_CERT_CN_INVALID |
			SECURITY_FLAG_IGNORE_CERT_DATE_INVALID |
			SECURITY_FLAG_IGNORE_UNKNOWN_CA |
			SECURITY_FLAG_IGNORE_CERT_WRONG_USAGE;
		WinHttpSetOption(hRequest, WINHTTP_OPTION_SECURITY_FLAGS, &flags, sizeof(flags));
	}
	const char *ctHeader = (content_type && content_type[0] != '\0') ? content_type : "text/plain; charset=utf-8";
	char *hdrBlock = build_header_block(hdrs, "*/*", ctHeader);
	wchar_t *whdr = NULL;
	if (hdrBlock) {
		whdr = utf8_to_wide(hdrBlock);
	}
	DWORD bodyLen = (DWORD)(body ? strlen(body) : 0);
	BOOL ok = WinHttpSendRequest(hRequest, whdr ? whdr : WINHTTP_NO_ADDITIONAL_HEADERS, whdr ? (DWORD)-1L : 0, body ? (LPVOID)body : WINHTTP_NO_REQUEST_DATA, bodyLen, bodyLen, 0);
	if (whdr) {
		free(whdr);
	}
	if (hdrBlock) {
		free(hdrBlock);
	}
	if (!ok || !WinHttpReceiveResponse(hRequest, NULL)) {
		char *msg = win_last_error();
		WinHttpCloseHandle(hRequest);
		WinHttpCloseHandle(hConnect);
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		Result_string_Error r = err_string(msg);
		free(msg);
		return r;
	}
	size_t cap = 4096;
	size_t len = 0;
	char *buf = (char *)malloc(cap);
	if (!buf) {
		WinHttpCloseHandle(hRequest);
		WinHttpCloseHandle(hConnect);
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		return err_string("http_request: out of memory");
	}
	buf[0] = '\0';
	for (;;) {
		DWORD available = 0;
		if (!WinHttpQueryDataAvailable(hRequest, &available)) {
			char *msg = win_last_error();
			free(buf);
			WinHttpCloseHandle(hRequest);
			WinHttpCloseHandle(hConnect);
			WinHttpCloseHandle(hSession);
			free(wurl);
			free(wua);
			Result_string_Error r = err_string(msg);
			free(msg);
			return r;
		}
		if (available == 0) {
			break;
		}
		if (len + available + 1 > cap) {
			size_t next = cap * 2;
			while (len + available + 1 > next) {
				next *= 2;
			}
			char *tmp = (char *)realloc(buf, next);
			if (!tmp) {
				free(buf);
				WinHttpCloseHandle(hRequest);
				WinHttpCloseHandle(hConnect);
				WinHttpCloseHandle(hSession);
				free(wurl);
				free(wua);
				return err_string("http_request: out of memory");
			}
			buf = tmp;
			cap = next;
		}
		DWORD read = 0;
		if (!WinHttpReadData(hRequest, buf + len, available, &read)) {
			char *msg = win_last_error();
			free(buf);
			WinHttpCloseHandle(hRequest);
			WinHttpCloseHandle(hConnect);
			WinHttpCloseHandle(hSession);
			free(wurl);
			free(wua);
			Result_string_Error r = err_string(msg);
			free(msg);
			return r;
		}
		len += read;
		buf[len] = '\0';
	}
	WinHttpCloseHandle(hRequest);
	WinHttpCloseHandle(hConnect);
	WinHttpCloseHandle(hSession);
	free(wurl);
	free(wua);
	return ok_string_owned(buf);
#else
	static bool curl_inited = false;
	if (!curl_inited) {
		curl_global_init(CURL_GLOBAL_DEFAULT);
		curl_inited = true;
	}
	CURL *curl = curl_easy_init();
	if (!curl) {
		return err_string("http_request: curl init failed");
	}
	struct {
		char *buf;
		size_t len;
		size_t cap;
	} ctx = {0};
	curl_easy_setopt(curl, CURLOPT_URL, url);
	curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, method);
	if (body && body[0] != '\0') {
		curl_easy_setopt(curl, CURLOPT_POSTFIELDS, body);
		curl_easy_setopt(curl, CURLOPT_POSTFIELDSIZE, (long)strlen(body));
	}
	curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
	curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT_MS, (long)(connect_timeout_ms > 0 ? connect_timeout_ms : 5000));
	curl_easy_setopt(curl, CURLOPT_TIMEOUT_MS, (long)(timeout_ms > 0 ? timeout_ms : 15000));
	curl_easy_setopt(curl, CURLOPT_USERAGENT, (user_agent && user_agent[0] != '\0') ? user_agent : "Bazic/1.0");
	if (tls_insecure) {
		curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0L);
		curl_easy_setopt(curl, CURLOPT_SSL_VERIFYHOST, 0L);
	}
	char *ca_path = NULL;
	if (ca_bundle_pem && ca_bundle_pem[0] != '\0') {
		ca_path = write_temp_ca(ca_bundle_pem);
		if (!ca_path) {
			curl_easy_cleanup(curl);
			return err_string("http_request: failed to write CA bundle");
		}
		curl_easy_setopt(curl, CURLOPT_CAINFO, ca_path);
	}
	struct curl_slist *headers = NULL;
	if (hdrs && hdrs[0] != '\0') {
		char *copy = bazic_strdup(hdrs);
		char *line = strtok(copy, "\n");
		while (line) {
			while (*line == ' ' || *line == '\t') { line++; }
			if (*line != '\0') {
				headers = curl_slist_append(headers, line);
			}
			line = strtok(NULL, "\n");
		}
		free(copy);
	}
	if (content_type && content_type[0] != '\0') {
		char ctLine[256];
		snprintf(ctLine, sizeof(ctLine), "Content-Type: %s", content_type);
		headers = curl_slist_append(headers, ctLine);
	}
	headers = curl_slist_append(headers, "Accept: */*");
	curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
	curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, curl_write_cb);
	curl_easy_setopt(curl, CURLOPT_WRITEDATA, &ctx);
	CURLcode res = curl_easy_perform(curl);
	if (res != CURLE_OK) {
		const char *msg = curl_easy_strerror(res);
		curl_slist_free_all(headers);
		curl_easy_cleanup(curl);
		if (ca_path) {
			unlink(ca_path);
			free(ca_path);
		}
		if (ctx.buf) {
			free(ctx.buf);
		}
		return err_string(msg);
	}
	curl_slist_free_all(headers);
	curl_easy_cleanup(curl);
	if (ca_path) {
		unlink(ca_path);
		free(ca_path);
	}
	if (!ctx.buf) {
		return ok_string("");
	}
	return ok_string_owned(ctx.buf);
#endif
}

Result_HttpResponse_Error __std_http_get_opts_resp(char *url, int64_t connect_timeout_ms, int64_t timeout_ms, char *hdrs, char *user_agent, bool tls_insecure, char *ca_bundle_pem) {
	return __std_http_request_resp("GET", url, NULL, connect_timeout_ms, timeout_ms, hdrs, user_agent, "", tls_insecure, ca_bundle_pem);
}

Result_HttpResponse_Error __std_http_post_opts_resp(char *url, char *body, int64_t connect_timeout_ms, int64_t timeout_ms, char *hdrs, char *user_agent, char *content_type, bool tls_insecure, char *ca_bundle_pem) {
	return __std_http_request_resp("POST", url, body, connect_timeout_ms, timeout_ms, hdrs, user_agent, content_type, tls_insecure, ca_bundle_pem);
}

Result_HttpResponse_Error __std_http_request_resp(char *method, char *url, char *body, int64_t connect_timeout_ms, int64_t timeout_ms, char *hdrs, char *user_agent, char *content_type, bool tls_insecure, char *ca_bundle_pem) {
	if (!method || method[0] == '\0') {
		method = "GET";
	}
	if (!url) {
		return err_http_response("http_request: null url");
	}
#ifdef _WIN32
	if (ca_bundle_pem && ca_bundle_pem[0] != '\0') {
		return err_http_response("http_request: custom CA bundle not supported on Windows");
	}
	wchar_t *wurl = utf8_to_wide(url);
	if (!wurl) {
		return err_http_response("http_request: url conversion failed");
	}
	URL_COMPONENTS parts;
	memset(&parts, 0, sizeof(parts));
	parts.dwStructSize = sizeof(parts);
	parts.dwSchemeLength = (DWORD)-1;
	parts.dwHostNameLength = (DWORD)-1;
	parts.dwUrlPathLength = (DWORD)-1;
	parts.dwExtraInfoLength = (DWORD)-1;
	if (!WinHttpCrackUrl(wurl, 0, 0, &parts)) {
		free(wurl);
		return err_http_response("http_request: invalid url");
	}
	const char *ua = (user_agent && user_agent[0] != '\0') ? user_agent : "Bazic/1.0";
	wchar_t *wua = utf8_to_wide(ua);
	if (!wua) {
		free(wurl);
		return err_http_response("http_request: user agent conversion failed");
	}
	HINTERNET hSession = WinHttpOpen(wua, WINHTTP_ACCESS_TYPE_DEFAULT_PROXY, NULL, NULL, 0);
	if (!hSession) {
		free(wurl);
		free(wua);
		char *msg = win_last_error();
		Result_HttpResponse_Error r = err_http_response(msg);
		free(msg);
		return r;
	}
	HINTERNET hConnect = WinHttpConnect(hSession, parts.lpszHostName, parts.nPort, 0);
	if (!hConnect) {
		char *msg = win_last_error();
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		Result_HttpResponse_Error r = err_http_response(msg);
		free(msg);
		return r;
	}
	DWORD flags = 0;
	if (parts.nScheme == INTERNET_SCHEME_HTTPS) {
		flags |= WINHTTP_FLAG_SECURE;
	}
	wchar_t path[2048];
	path[0] = L'\0';
	if (parts.lpszUrlPath && parts.dwUrlPathLength > 0) {
		wcsncat(path, parts.lpszUrlPath, parts.dwUrlPathLength);
	}
	if (parts.lpszExtraInfo && parts.dwExtraInfoLength > 0) {
		wcsncat(path, parts.lpszExtraInfo, parts.dwExtraInfoLength);
	}
	if (wcslen(path) == 0) {
		wcscpy(path, L"/");
	}
	wchar_t *wmethod = utf8_to_wide(method);
	if (!wmethod) {
		WinHttpCloseHandle(hConnect);
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		return err_http_response("http_request: method conversion failed");
	}
	HINTERNET hRequest = WinHttpOpenRequest(hConnect, wmethod, path, NULL, WINHTTP_NO_REFERER, WINHTTP_DEFAULT_ACCEPT_TYPES, flags);
	free(wmethod);
	if (!hRequest) {
		char *msg = win_last_error();
		WinHttpCloseHandle(hConnect);
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		Result_HttpResponse_Error r = err_http_response(msg);
		free(msg);
		return r;
	}
	int ct = (int)(connect_timeout_ms > 0 ? connect_timeout_ms : 5000);
	int tt = (int)(timeout_ms > 0 ? timeout_ms : 15000);
	WinHttpSetTimeouts(hRequest, ct, ct, tt, tt);
	if (tls_insecure) {
		DWORD secFlags = SECURITY_FLAG_IGNORE_CERT_CN_INVALID |
			SECURITY_FLAG_IGNORE_CERT_DATE_INVALID |
			SECURITY_FLAG_IGNORE_UNKNOWN_CA |
			SECURITY_FLAG_IGNORE_CERT_WRONG_USAGE;
		WinHttpSetOption(hRequest, WINHTTP_OPTION_SECURITY_FLAGS, &secFlags, sizeof(secFlags));
	}
	const char *ctHeader = (content_type && content_type[0] != '\0') ? content_type : "text/plain; charset=utf-8";
	char *hdrBlock = build_header_block(hdrs, "*/*", ctHeader);
	wchar_t *whdr = NULL;
	if (hdrBlock) {
		whdr = utf8_to_wide(hdrBlock);
	}
	DWORD bodyLen = (DWORD)(body ? strlen(body) : 0);
	BOOL ok = WinHttpSendRequest(hRequest, whdr ? whdr : WINHTTP_NO_ADDITIONAL_HEADERS, whdr ? (DWORD)-1L : 0, body ? (LPVOID)body : WINHTTP_NO_REQUEST_DATA, bodyLen, bodyLen, 0);
	if (whdr) {
		free(whdr);
	}
	if (hdrBlock) {
		free(hdrBlock);
	}
	if (!ok || !WinHttpReceiveResponse(hRequest, NULL)) {
		char *msg = win_last_error();
		WinHttpCloseHandle(hRequest);
		WinHttpCloseHandle(hConnect);
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		Result_HttpResponse_Error r = err_http_response(msg);
		free(msg);
		return r;
	}
	DWORD status = 0;
	DWORD statusSize = sizeof(status);
	WinHttpQueryHeaders(hRequest, WINHTTP_QUERY_STATUS_CODE | WINHTTP_QUERY_FLAG_NUMBER, WINHTTP_HEADER_NAME_BY_INDEX, &status, &statusSize, WINHTTP_NO_HEADER_INDEX);
	DWORD rawSize = 0;
	WinHttpQueryHeaders(hRequest, WINHTTP_QUERY_RAW_HEADERS_CRLF, WINHTTP_HEADER_NAME_BY_INDEX, NULL, &rawSize, WINHTTP_NO_HEADER_INDEX);
	wchar_t *rawW = NULL;
	char *rawHeaders = NULL;
	if (rawSize > 0) {
		rawW = (wchar_t *)malloc(rawSize);
		if (rawW && WinHttpQueryHeaders(hRequest, WINHTTP_QUERY_RAW_HEADERS_CRLF, WINHTTP_HEADER_NAME_BY_INDEX, rawW, &rawSize, WINHTTP_NO_HEADER_INDEX)) {
			rawHeaders = wide_to_utf8(rawW);
		}
		if (rawW) {
			free(rawW);
		}
	}
	size_t cap = 4096;
	size_t len = 0;
	char *buf = (char *)malloc(cap);
	if (!buf) {
		WinHttpCloseHandle(hRequest);
		WinHttpCloseHandle(hConnect);
		WinHttpCloseHandle(hSession);
		free(wurl);
		free(wua);
		if (rawHeaders) {
			free(rawHeaders);
		}
		return err_http_response("http_request: out of memory");
	}
	buf[0] = '\0';
	for (;;) {
		DWORD available = 0;
		if (!WinHttpQueryDataAvailable(hRequest, &available)) {
			char *msg = win_last_error();
			free(buf);
			WinHttpCloseHandle(hRequest);
			WinHttpCloseHandle(hConnect);
			WinHttpCloseHandle(hSession);
			free(wurl);
			free(wua);
			if (rawHeaders) {
				free(rawHeaders);
			}
			Result_HttpResponse_Error r = err_http_response(msg);
			free(msg);
			return r;
		}
		if (available == 0) {
			break;
		}
		if (len + available + 1 > cap) {
			size_t next = cap * 2;
			while (len + available + 1 > next) {
				next *= 2;
			}
			char *tmp = (char *)realloc(buf, next);
			if (!tmp) {
				free(buf);
				WinHttpCloseHandle(hRequest);
				WinHttpCloseHandle(hConnect);
				WinHttpCloseHandle(hSession);
				free(wurl);
				free(wua);
				if (rawHeaders) {
					free(rawHeaders);
				}
				return err_http_response("http_request: out of memory");
			}
			buf = tmp;
			cap = next;
		}
		DWORD read = 0;
		if (!WinHttpReadData(hRequest, buf + len, available, &read)) {
			char *msg = win_last_error();
			free(buf);
			WinHttpCloseHandle(hRequest);
			WinHttpCloseHandle(hConnect);
			WinHttpCloseHandle(hSession);
			free(wurl);
			free(wua);
			if (rawHeaders) {
				free(rawHeaders);
			}
			Result_HttpResponse_Error r = err_http_response(msg);
			free(msg);
			return r;
		}
		len += read;
		buf[len] = '\0';
	}
	WinHttpCloseHandle(hRequest);
	WinHttpCloseHandle(hConnect);
	WinHttpCloseHandle(hSession);
	free(wurl);
	free(wua);
	return ok_http_response((int64_t)status, rawHeaders ? rawHeaders : bazic_strdup(""), buf);
#else
	static bool curl_inited = false;
	if (!curl_inited) {
		curl_global_init(CURL_GLOBAL_DEFAULT);
		curl_inited = true;
	}
	CURL *curl = curl_easy_init();
	if (!curl) {
		return err_http_response("http_request: curl init failed");
	}
	struct {
		char *buf;
		size_t len;
		size_t cap;
	} bodyCtx = {0};
	struct {
		char *buf;
		size_t len;
		size_t cap;
	} hdrCtx = {0};
	curl_easy_setopt(curl, CURLOPT_URL, url);
	curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, method);
	if (body && body[0] != '\0') {
		curl_easy_setopt(curl, CURLOPT_POSTFIELDS, body);
		curl_easy_setopt(curl, CURLOPT_POSTFIELDSIZE, (long)strlen(body));
	}
	curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
	curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT_MS, (long)(connect_timeout_ms > 0 ? connect_timeout_ms : 5000));
	curl_easy_setopt(curl, CURLOPT_TIMEOUT_MS, (long)(timeout_ms > 0 ? timeout_ms : 15000));
	curl_easy_setopt(curl, CURLOPT_USERAGENT, (user_agent && user_agent[0] != '\0') ? user_agent : "Bazic/1.0");
	if (tls_insecure) {
		curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0L);
		curl_easy_setopt(curl, CURLOPT_SSL_VERIFYHOST, 0L);
	}
	char *ca_path = NULL;
	if (ca_bundle_pem && ca_bundle_pem[0] != '\0') {
		ca_path = write_temp_ca(ca_bundle_pem);
		if (!ca_path) {
			curl_easy_cleanup(curl);
			return err_http_response("http_request: failed to write CA bundle");
		}
		curl_easy_setopt(curl, CURLOPT_CAINFO, ca_path);
	}
	struct curl_slist *headers = NULL;
	if (hdrs && hdrs[0] != '\0') {
		char *copy = bazic_strdup(hdrs);
		char *line = strtok(copy, "\n");
		while (line) {
			while (*line == ' ' || *line == '\t') { line++; }
			if (*line != '\0') {
				headers = curl_slist_append(headers, line);
			}
			line = strtok(NULL, "\n");
		}
		free(copy);
	}
	if (content_type && content_type[0] != '\0') {
		char ctLine[256];
		snprintf(ctLine, sizeof(ctLine), "Content-Type: %s", content_type);
		headers = curl_slist_append(headers, ctLine);
	}
	headers = curl_slist_append(headers, "Accept: */*");
	curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
	curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, curl_write_cb);
	curl_easy_setopt(curl, CURLOPT_WRITEDATA, &bodyCtx);
	curl_easy_setopt(curl, CURLOPT_HEADERFUNCTION, curl_header_cb);
	curl_easy_setopt(curl, CURLOPT_HEADERDATA, &hdrCtx);
	CURLcode res = curl_easy_perform(curl);
	if (res != CURLE_OK) {
		const char *msg = curl_easy_strerror(res);
		curl_slist_free_all(headers);
		curl_easy_cleanup(curl);
		if (ca_path) {
			unlink(ca_path);
			free(ca_path);
		}
		if (bodyCtx.buf) {
			free(bodyCtx.buf);
		}
		if (hdrCtx.buf) {
			free(hdrCtx.buf);
		}
		return err_http_response(msg);
	}
	long status = 0;
	curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &status);
	curl_slist_free_all(headers);
	curl_easy_cleanup(curl);
	if (ca_path) {
		unlink(ca_path);
		free(ca_path);
	}
	char *headersOut = hdrCtx.buf ? hdrCtx.buf : bazic_strdup("");
	char *bodyOut = bodyCtx.buf ? bodyCtx.buf : bazic_strdup("");
	return ok_http_response((int64_t)status, headersOut, bodyOut);
#endif
}

Result_bool_Error __std_db_exec(char *path, char *sql) {
	if (!path || !sql) {
		return err_bool("db_exec: null input");
	}
#ifdef BAZIC_SQLITE
	sqlite3 *db = NULL;
	if (sqlite3_open(path, &db) != SQLITE_OK) {
		if (db) sqlite3_close(db);
		return err_bool("db_exec: open failed");
	}
	char *errmsg = NULL;
	if (sqlite3_exec(db, sql, NULL, NULL, &errmsg) != SQLITE_OK) {
		if (errmsg) sqlite3_free(errmsg);
		sqlite3_close(db);
		return err_bool("db_exec: exec failed");
	}
	sqlite3_close(db);
	return ok_bool(true);
#else
	(void)path;
	(void)sql;
	return err_bool("db_exec not implemented in native runtime (enable BAZIC_SQLITE)");
#endif
}

Result_string_Error __std_db_query(char *path, char *sql) {
	if (!path || !sql) {
		return err_string("db_query: null input");
	}
#ifdef BAZIC_SQLITE
	sqlite3 *db = NULL;
	if (sqlite3_open(path, &db) != SQLITE_OK) {
		if (db) sqlite3_close(db);
		return err_string("db_query: open failed");
	}
	sqlite3_stmt *stmt = NULL;
	if (sqlite3_prepare_v2(db, sql, -1, &stmt, NULL) != SQLITE_OK) {
		sqlite3_close(db);
		return err_string("db_query: prepare failed");
	}
	int cols = sqlite3_column_count(stmt);
	size_t cap = 256;
	size_t len = 0;
	char *out = (char *)malloc(cap);
	if (!out) {
		sqlite3_finalize(stmt);
		sqlite3_close(db);
		return err_string("db_query: out of memory");
	}
	out[0] = '\0';
	for (int i = 0; i < cols; i++) {
		const char *name = sqlite3_column_name(stmt, i);
		if (i > 0) append_str(&out, &cap, &len, "\t");
		append_str(&out, &cap, &len, name ? name : "");
	}
	append_str(&out, &cap, &len, "\n");
	while (sqlite3_step(stmt) == SQLITE_ROW) {
		for (int i = 0; i < cols; i++) {
			if (i > 0) append_str(&out, &cap, &len, "\t");
			const unsigned char *txt = sqlite3_column_text(stmt, i);
			if (txt) {
				append_str(&out, &cap, &len, (const char *)txt);
			} else {
				append_str(&out, &cap, &len, "null");
			}
		}
		append_str(&out, &cap, &len, "\n");
	}
	sqlite3_finalize(stmt);
	sqlite3_close(db);
	if (len > 0 && out[len - 1] == '\n') {
		out[len - 1] = '\0';
	}
	return ok_string_owned(out);
#else
	(void)path;
	(void)sql;
	return err_string("db_query not implemented in native runtime (enable BAZIC_SQLITE)");
#endif
}

Result_bool_Error __std_db_exec_with(char *driver, char *dsn, char *sql) {
	if (!driver || driver[0] == '\0') {
		return err_bool("db_exec_with: empty driver");
	}
	if (!sql) {
		return err_bool("db_exec_with: null sql");
	}
	if (!dsn) {
		dsn = "";
	}
	if (bazic_stricmp(driver, "sqlite") == 0 || bazic_stricmp(driver, "sqlite3") == 0) {
		return __std_db_exec(dsn, sql);
	}
	return err_bool("db_exec_with: unsupported driver (native supports sqlite/sqlite3)");
}

Result_string_Error __std_db_query_with(char *driver, char *dsn, char *sql) {
	if (!driver || driver[0] == '\0') {
		return err_string("db_query_with: empty driver");
	}
	if (!sql) {
		return err_string("db_query_with: null sql");
	}
	if (!dsn) {
		dsn = "";
	}
	if (bazic_stricmp(driver, "sqlite") == 0 || bazic_stricmp(driver, "sqlite3") == 0) {
		return __std_db_query(dsn, sql);
	}
	return err_string("db_query_with: unsupported driver (native supports sqlite/sqlite3)");
}

Result_string_Error __std_db_query_json(char *path, char *sql) {
	if (!path || !sql) {
		return err_string("db_query_json: null input");
	}
#ifdef BAZIC_SQLITE
	sqlite3 *db = NULL;
	if (sqlite3_open(path, &db) != SQLITE_OK) {
		if (db) sqlite3_close(db);
		return err_string("db_query_json: open failed");
	}
	sqlite3_stmt *stmt = NULL;
	if (sqlite3_prepare_v2(db, sql, -1, &stmt, NULL) != SQLITE_OK) {
		sqlite3_close(db);
		return err_string("db_query_json: prepare failed");
	}
	int cols = sqlite3_column_count(stmt);
	size_t cap = 256;
	size_t len = 0;
	char *out = (char *)malloc(cap);
	if (!out) {
		sqlite3_finalize(stmt);
		sqlite3_close(db);
		return err_string("db_query_json: out of memory");
	}
	out[0] = '\0';
	append_str(&out, &cap, &len, "[");
	bool firstRow = true;
	while (sqlite3_step(stmt) == SQLITE_ROW) {
		if (!firstRow) {
			append_str(&out, &cap, &len, ",");
		}
		firstRow = false;
		append_str(&out, &cap, &len, "{");
		for (int i = 0; i < cols; i++) {
			if (i > 0) append_str(&out, &cap, &len, ",");
			const char *name = sqlite3_column_name(stmt, i);
			char *escapedName = __std_json_escape((char *)(name ? name : ""));
			append_str(&out, &cap, &len, "\"");
			append_str(&out, &cap, &len, escapedName ? escapedName : "");
			append_str(&out, &cap, &len, "\":");
			if (escapedName) free(escapedName);
			int t = sqlite3_column_type(stmt, i);
			if (t == SQLITE_NULL) {
				append_str(&out, &cap, &len, "null");
			} else if (t == SQLITE_INTEGER) {
				char buf[64];
				snprintf(buf, sizeof(buf), "%lld", (long long)sqlite3_column_int64(stmt, i));
				append_str(&out, &cap, &len, buf);
			} else if (t == SQLITE_FLOAT) {
				char buf[64];
				snprintf(buf, sizeof(buf), "%g", sqlite3_column_double(stmt, i));
				append_str(&out, &cap, &len, buf);
			} else {
				const unsigned char *txt = sqlite3_column_text(stmt, i);
				char *escaped = __std_json_escape((char *)(txt ? (const char *)txt : ""));
				append_str(&out, &cap, &len, "\"");
				append_str(&out, &cap, &len, escaped ? escaped : "");
				append_str(&out, &cap, &len, "\"");
				if (escaped) free(escaped);
			}
		}
		append_str(&out, &cap, &len, "}");
	}
	append_str(&out, &cap, &len, "]");
	sqlite3_finalize(stmt);
	sqlite3_close(db);
	return ok_string_owned(out);
#else
	(void)path;
	(void)sql;
	return err_string("db_query_json not implemented in native runtime (enable BAZIC_SQLITE)");
#endif
}

Result_string_Error __std_db_query_json_with(char *driver, char *dsn, char *sql) {
	if (!driver || driver[0] == '\0') {
		return err_string("db_query_json_with: empty driver");
	}
	if (!sql) {
		return err_string("db_query_json_with: null sql");
	}
	if (!dsn) {
		dsn = "";
	}
	if (bazic_stricmp(driver, "sqlite") == 0 || bazic_stricmp(driver, "sqlite3") == 0) {
		return __std_db_query_json(dsn, sql);
	}
	return err_string("db_query_json_with: unsupported driver (native supports sqlite/sqlite3)");
}

Result_string_Error __std_db_query_one_json(char *path, char *sql) {
	if (!path || !sql) {
		return err_string("db_query_one_json: null input");
	}
#ifdef BAZIC_SQLITE
	sqlite3 *db = NULL;
	if (sqlite3_open(path, &db) != SQLITE_OK) {
		if (db) sqlite3_close(db);
		return err_string("db_query_one_json: open failed");
	}
	sqlite3_stmt *stmt = NULL;
	if (sqlite3_prepare_v2(db, sql, -1, &stmt, NULL) != SQLITE_OK) {
		sqlite3_close(db);
		return err_string("db_query_one_json: prepare failed");
	}
	int cols = sqlite3_column_count(stmt);
	int rc = sqlite3_step(stmt);
	if (rc != SQLITE_ROW) {
		sqlite3_finalize(stmt);
		sqlite3_close(db);
		return err_string("db_query_one_json: not found");
	}
	size_t cap = 256;
	size_t len = 0;
	char *out = (char *)malloc(cap);
	if (!out) {
		sqlite3_finalize(stmt);
		sqlite3_close(db);
		return err_string("db_query_one_json: out of memory");
	}
	out[0] = '\0';
	append_str(&out, &cap, &len, "{");
	for (int i = 0; i < cols; i++) {
		if (i > 0) append_str(&out, &cap, &len, ",");
		const char *name = sqlite3_column_name(stmt, i);
		char *escapedName = __std_json_escape((char *)(name ? name : ""));
		append_str(&out, &cap, &len, "\"");
		append_str(&out, &cap, &len, escapedName ? escapedName : "");
		append_str(&out, &cap, &len, "\":");
		if (escapedName) free(escapedName);
		int t = sqlite3_column_type(stmt, i);
		if (t == SQLITE_NULL) {
			append_str(&out, &cap, &len, "null");
		} else if (t == SQLITE_INTEGER) {
			char buf[64];
			snprintf(buf, sizeof(buf), "%lld", (long long)sqlite3_column_int64(stmt, i));
			append_str(&out, &cap, &len, buf);
		} else if (t == SQLITE_FLOAT) {
			char buf[64];
			snprintf(buf, sizeof(buf), "%g", sqlite3_column_double(stmt, i));
			append_str(&out, &cap, &len, buf);
		} else {
			const unsigned char *txt = sqlite3_column_text(stmt, i);
			char *escaped = __std_json_escape((char *)(txt ? (const char *)txt : ""));
			append_str(&out, &cap, &len, "\"");
			append_str(&out, &cap, &len, escaped ? escaped : "");
			append_str(&out, &cap, &len, "\"");
			if (escaped) free(escaped);
		}
	}
	append_str(&out, &cap, &len, "}");
	sqlite3_finalize(stmt);
	sqlite3_close(db);
	return ok_string_owned(out);
#else
	(void)path;
	(void)sql;
	return err_string("db_query_one_json not implemented in native runtime (enable BAZIC_SQLITE)");
#endif
}

Result_string_Error __std_db_query_one_json_with(char *driver, char *dsn, char *sql) {
	if (!driver || driver[0] == '\0') {
		return err_string("db_query_one_json_with: empty driver");
	}
	if (!sql) {
		return err_string("db_query_one_json_with: null sql");
	}
	if (!dsn) {
		dsn = "";
	}
	if (bazic_stricmp(driver, "sqlite") == 0 || bazic_stricmp(driver, "sqlite3") == 0) {
		return __std_db_query_one_json(dsn, sql);
	}
	return err_string("db_query_one_json_with: unsupported driver (native supports sqlite/sqlite3)");
}

Result_bool_Error __std_db_exec_params(char *path, char *sql, char *params) {
	(void)path;
	(void)sql;
	(void)params;
	return err_bool("db_exec_params not supported in llvm backend");
}

Result_bool_Error __std_db_exec_params_with(char *driver, char *dsn, char *sql, char *params) {
	(void)driver;
	(void)dsn;
	(void)sql;
	(void)params;
	return err_bool("db_exec_params_with not supported in llvm backend");
}

Result_string_Error __std_db_query_params(char *path, char *sql, char *params) {
	(void)path;
	(void)sql;
	(void)params;
	return err_string("db_query_params not supported in llvm backend");
}

Result_string_Error __std_db_query_params_with(char *driver, char *dsn, char *sql, char *params) {
	(void)driver;
	(void)dsn;
	(void)sql;
	(void)params;
	return err_string("db_query_params_with not supported in llvm backend");
}

Result_string_Error __std_db_query_json_params(char *path, char *sql, char *params) {
	(void)path;
	(void)sql;
	(void)params;
	return err_string("db_query_json_params not supported in llvm backend");
}

Result_string_Error __std_db_query_json_params_with(char *driver, char *dsn, char *sql, char *params) {
	(void)driver;
	(void)dsn;
	(void)sql;
	(void)params;
	return err_string("db_query_json_params_with not supported in llvm backend");
}

Result_string_Error __std_db_query_one_json_params(char *path, char *sql, char *params) {
	(void)path;
	(void)sql;
	(void)params;
	return err_string("db_query_one_json_params not supported in llvm backend");
}

Result_string_Error __std_db_query_one_json_params_with(char *driver, char *dsn, char *sql, char *params) {
	(void)driver;
	(void)dsn;
	(void)sql;
	(void)params;
	return err_string("db_query_one_json_params_with not supported in llvm backend");
}

Result_int_Error __std_db_exec_returning_id_params(char *path, char *sql, char *params) {
	(void)path;
	(void)sql;
	(void)params;
	return err_int("db_exec_returning_id_params not supported in llvm backend");
}

Result_int_Error __std_db_exec_returning_id_params_with(char *driver, char *dsn, char *sql, char *params) {
	(void)driver;
	(void)dsn;
	(void)sql;
	(void)params;
	return err_int("db_exec_returning_id_params_with not supported in llvm backend");
}

Result_int_Error __std_db_exec_returning_id(char *path, char *sql) {
	if (!path || !sql) {
		return err_int("db_exec_returning_id: null input");
	}
#ifdef BAZIC_SQLITE
	sqlite3 *db = NULL;
	if (sqlite3_open(path, &db) != SQLITE_OK) {
		if (db) sqlite3_close(db);
		return err_int("db_exec_returning_id: open failed");
	}
	char *errmsg = NULL;
	if (sqlite3_exec(db, sql, NULL, NULL, &errmsg) != SQLITE_OK) {
		if (errmsg) sqlite3_free(errmsg);
		sqlite3_close(db);
		return err_int("db_exec_returning_id: exec failed");
	}
	int64_t id = sqlite3_last_insert_rowid(db);
	sqlite3_close(db);
	return ok_int(id);
#else
	(void)path;
	(void)sql;
	return err_int("db_exec_returning_id not implemented in native runtime (enable BAZIC_SQLITE)");
#endif
}

Result_int_Error __std_db_exec_returning_id_with(char *driver, char *dsn, char *sql) {
	if (!driver || driver[0] == '\0') {
		return err_int("db_exec_returning_id_with: empty driver");
	}
	if (!sql) {
		return err_int("db_exec_returning_id_with: null sql");
	}
	if (!dsn) {
		dsn = "";
	}
	if (bazic_stricmp(driver, "sqlite") == 0 || bazic_stricmp(driver, "sqlite3") == 0) {
		return __std_db_exec_returning_id(dsn, sql);
	}
	return err_int("db_exec_returning_id_with: unsupported driver (native supports sqlite/sqlite3)");
}

static int64_t bazic_env_int64(const char *key, int64_t def) {
	const char *val = getenv(key);
	if (!val || val[0] == '\0') {
		return def;
	}
	char *end = NULL;
	long long v = strtoll(val, &end, 10);
	if (!end || end == val) {
		return def;
	}
	return (int64_t)v;
}

static int bazic_headers_has(const char *headers, const char *key) {
	if (!headers || !key || key[0] == '\0') {
		return 0;
	}
	size_t keylen = strlen(key);
	const char *p = headers;
	while (*p) {
		const char *line_end = strchr(p, '\n');
		if (!line_end) {
			line_end = p + strlen(p);
		}
		const char *colon = memchr(p, ':', (size_t)(line_end - p));
		if (colon) {
			size_t klen = (size_t)(colon - p);
			while (klen > 0 && p[klen - 1] == ' ') {
				klen--;
			}
			if (klen == keylen && bazic_strnicmp(p, key, klen) == 0) {
				return 1;
			}
		}
		if (*line_end == '\0') {
			break;
		}
		p = line_end + 1;
	}
	return 0;
}

static const char *bazic_status_text(int64_t status) {
	switch (status) {
	case 200: return "OK";
	case 201: return "Created";
	case 202: return "Accepted";
	case 204: return "No Content";
	case 400: return "Bad Request";
	case 401: return "Unauthorized";
	case 403: return "Forbidden";
	case 404: return "Not Found";
	case 405: return "Method Not Allowed";
	case 409: return "Conflict";
	case 413: return "Payload Too Large";
	case 422: return "Unprocessable Entity";
	case 500: return "Internal Server Error";
	default: return "OK";
	}
}

static void bazic_upper_inplace(char *s) {
	if (!s) return;
	for (; *s; s++) {
		*s = (char)toupper((unsigned char)*s);
	}
}

static int bazic_send_all_socket(bazic_socket_t sock, const char *buf, int len) {
	int sent = 0;
	while (sent < len) {
		int n = send(sock, buf + sent, len - sent, 0);
		if (n <= 0) {
			return 0;
		}
		sent += n;
	}
	return 1;
}

static void bazic_append_str(char **buf, size_t *len, size_t *cap, const char *s, size_t n) {
	if (!s || n == 0) {
		return;
	}
	if (*cap < *len + n + 1) {
		size_t next = (*cap == 0) ? 128 : *cap * 2;
		while (next < *len + n + 1) {
			next *= 2;
		}
		char *nbuf = (char *)realloc(*buf, next);
		if (!nbuf) {
			return;
		}
		*buf = nbuf;
		*cap = next;
	}
	memcpy(*buf + *len, s, n);
	*len += n;
	(*buf)[*len] = '\0';
}

static void bazic_params_add(char **buf, size_t *len, size_t *cap, const char *key, size_t klen, const char *val, size_t vlen) {
	bazic_append_str(buf, len, cap, key, klen);
	bazic_append_str(buf, len, cap, "=", 1);
	bazic_append_str(buf, len, cap, val, vlen);
	bazic_append_str(buf, len, cap, "\n", 1);
}

static int bazic_match_route(const char *path, const char *pattern, char **params_out) {
	if (!path || !pattern) {
		return 0;
	}
	if (strcmp(pattern, "/") == 0) {
		if (strcmp(path, "/") == 0) {
			if (params_out) {
				*params_out = bazic_strdup("");
			}
			return 1;
		}
		return 0;
	}
	const char *p = pattern;
	const char *s = path;
	if (*p != '/' || *s != '/') {
		return 0;
	}
	p++;
	s++;
	char *params = NULL;
	size_t params_len = 0;
	size_t params_cap = 0;
	while (1) {
		if (*p == '\0' && *s == '\0') {
			if (params_out) {
				*params_out = params ? params : bazic_strdup("");
			} else if (params) {
				free(params);
			}
			return 1;
		}
		if (*p == '\0' || *s == '\0') {
			if (params) free(params);
			return 0;
		}
		const char *pnext = strchr(p, '/');
		const char *snext = strchr(s, '/');
		if (!pnext) pnext = p + strlen(p);
		if (!snext) snext = s + strlen(s);
		size_t plen = (size_t)(pnext - p);
		size_t slen = (size_t)(snext - s);
		if (plen == 0 || slen == 0) {
			if (params) free(params);
			return 0;
		}
		if (p[0] == ':') {
			if (plen == 1) {
				if (params) free(params);
				return 0;
			}
			bazic_params_add(&params, &params_len, &params_cap, p + 1, plen - 1, s, slen);
		} else {
			if (plen != slen || strncmp(p, s, plen) != 0) {
				if (params) free(params);
				return 0;
			}
		}
		if (*pnext == '\0' && *snext == '\0') {
			break;
		}
		if ((*pnext == '\0') != (*snext == '\0')) {
			if (params) free(params);
			return 0;
		}
		p = (*pnext == '\0') ? pnext : pnext + 1;
		s = (*snext == '\0') ? snext : snext + 1;
	}
	if (params_out) {
		*params_out = params ? params : bazic_strdup("");
	} else if (params) {
		free(params);
	}
	return 1;
}

static void bazic_send_simple_response(bazic_socket_t sock, int64_t status, const char *body) {
	if (!body) body = "";
	char header[512];
	size_t body_len = strlen(body);
	const char *status_text = bazic_status_text(status);
	snprintf(header, sizeof(header),
		"HTTP/1.1 %lld %s\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: %u\r\nConnection: close\r\n\r\n",
		(long long)status, status_text, (unsigned)body_len);
	bazic_send_all_socket(sock, header, (int)strlen(header));
	if (body_len > 0) {
		bazic_send_all_socket(sock, body, (int)body_len);
	}
}

static void bazic_normalize_path(char *path) {
	if (!path || path[0] == '\0') {
		return;
	}
	size_t len = strlen(path);
	if (len == 0) return;
	if (path[0] != '/') {
		return;
	}
	while (len > 1 && path[len - 1] == '/') {
		path[len - 1] = '\0';
		len--;
	}
}

Result_bool_Error __std_http_serve_app(char *addr) {
	if (!addr) {
		return err_bool("http_serve_app: null addr");
	}
	if (__bazic_routes_len <= 0) {
		return err_bool("http_serve_app: no http handlers found");
	}
	int64_t max_body = bazic_env_int64("BAZIC_HTTP_MAX_BODY", 1048576);
	int64_t read_timeout_ms = bazic_env_int64("BAZIC_HTTP_READ_TIMEOUT_MS", 10000);
	int64_t write_timeout_ms = bazic_env_int64("BAZIC_HTTP_WRITE_TIMEOUT_MS", 15000);

#ifdef _WIN32
	char host[256];
	char port[16];
	host[0] = '\0';
	port[0] = '\0';
	if (addr[0] == ':') {
		strcpy(host, "0.0.0.0");
		strncpy(port, addr + 1, sizeof(port) - 1);
		port[sizeof(port) - 1] = '\0';
	} else {
		const char *colon = strrchr(addr, ':');
		if (!colon) {
			return err_bool("http_serve_app: addr must be host:port or :port");
		}
		size_t hlen = (size_t)(colon - addr);
		if (hlen >= sizeof(host)) {
			return err_bool("http_serve_app: host too long");
		}
		memcpy(host, addr, hlen);
		host[hlen] = '\0';
		strncpy(port, colon + 1, sizeof(port) - 1);
		port[sizeof(port) - 1] = '\0';
		if (host[0] == '\0') {
			strcpy(host, "0.0.0.0");
		}
	}
	WSADATA wsa;
	if (WSAStartup(MAKEWORD(2, 2), &wsa) != 0) {
		return err_bool("http_serve_app: WSAStartup failed");
	}
	struct addrinfo hints;
	memset(&hints, 0, sizeof(hints));
	hints.ai_family = AF_INET;
	hints.ai_socktype = SOCK_STREAM;
	hints.ai_protocol = IPPROTO_TCP;
	hints.ai_flags = AI_PASSIVE;
	struct addrinfo *res = NULL;
	if (getaddrinfo(host, port, &hints, &res) != 0) {
		WSACleanup();
		return err_bool("http_serve_app: getaddrinfo failed");
	}
	SOCKET server = socket(res->ai_family, res->ai_socktype, res->ai_protocol);
	if (server == INVALID_SOCKET) {
		freeaddrinfo(res);
		WSACleanup();
		return err_bool("http_serve_app: socket failed");
	}
	int opt = 1;
	setsockopt(server, SOL_SOCKET, SO_REUSEADDR, (const char *)&opt, sizeof(opt));
	if (bind(server, res->ai_addr, (int)res->ai_addrlen) == SOCKET_ERROR) {
		closesocket(server);
		freeaddrinfo(res);
		WSACleanup();
		return err_bool("http_serve_app: bind failed");
	}
	freeaddrinfo(res);
	if (listen(server, 64) == SOCKET_ERROR) {
		closesocket(server);
		WSACleanup();
		return err_bool("http_serve_app: listen failed");
	}
	for (;;) {
		struct sockaddr_storage peer;
		int peerlen = sizeof(peer);
		SOCKET client = accept(server, (struct sockaddr *)&peer, &peerlen);
		if (client == INVALID_SOCKET) {
			closesocket(server);
			WSACleanup();
			return err_bool("http_serve_app: accept failed");
		}
		DWORD rcv_timeout = (DWORD)read_timeout_ms;
		DWORD snd_timeout = (DWORD)write_timeout_ms;
		setsockopt(client, SOL_SOCKET, SO_RCVTIMEO, (const char *)&rcv_timeout, sizeof(rcv_timeout));
		setsockopt(client, SOL_SOCKET, SO_SNDTIMEO, (const char *)&snd_timeout, sizeof(snd_timeout));

		char *buf = NULL;
		size_t len = 0;
		size_t cap = 0;
		const size_t max_header = 65536;
		int header_found = 0;
		while (!header_found) {
			char tmp[4096];
			int n = recv(client, tmp, sizeof(tmp), 0);
			if (n <= 0) {
				break;
			}
			if (cap < len + (size_t)n + 1) {
				size_t next = cap == 0 ? 8192 : cap * 2;
				while (next < len + (size_t)n + 1) next *= 2;
				char *nbuf = (char *)realloc(buf, next);
				if (!nbuf) {
					free(buf);
					buf = NULL;
					break;
				}
				buf = nbuf;
				cap = next;
			}
			memcpy(buf + len, tmp, (size_t)n);
			len += (size_t)n;
			buf[len] = '\0';
			if (len >= 4 && strstr(buf, "\r\n\r\n")) {
				header_found = 1;
				break;
			}
			if (len > max_header) {
				break;
			}
		}
		if (!buf || !header_found) {
			if (buf) free(buf);
			bazic_send_simple_response(client, 400, "bad request");
			closesocket(client);
			continue;
		}
		char *header_end = strstr(buf, "\r\n\r\n");
		size_t header_len = (size_t)(header_end - buf);
		char *line_end = strstr(buf, "\r\n");
		if (!line_end) {
			free(buf);
			bazic_send_simple_response(client, 400, "bad request");
			closesocket(client);
			continue;
		}
		size_t line_len = (size_t)(line_end - buf);
		char *sp1 = memchr(buf, ' ', line_len);
		char *sp2 = sp1 ? memchr(sp1 + 1, ' ', (size_t)(buf + line_len - (sp1 + 1))) : NULL;
		if (!sp1 || !sp2) {
			free(buf);
			bazic_send_simple_response(client, 400, "bad request");
			closesocket(client);
			continue;
		}
		char *method = bazic_strndup(buf, (size_t)(sp1 - buf));
		char *target = bazic_strndup(sp1 + 1, (size_t)(sp2 - sp1 - 1));
		if (!method || !target) {
			free(buf);
			free(method);
			free(target);
			bazic_send_simple_response(client, 500, "server error");
			closesocket(client);
			continue;
		}
		bazic_upper_inplace(method);
		char *query = NULL;
		char *qmark = strchr(target, '?');
		if (qmark) {
			query = bazic_strdup(qmark + 1);
			*qmark = '\0';
		} else {
			query = bazic_strdup("");
		}
		if (!query) query = bazic_strdup("");
		char *path = target;
		if (path[0] == '\0') {
			free(path);
			path = bazic_strdup("/");
		}
		if (path) {
			bazic_normalize_path(path);
		}

		char *headers = NULL;
		size_t headers_len = 0;
		size_t headers_cap = 0;
		char *cookies = NULL;
		size_t cookies_len = 0;
		size_t cookies_cap = 0;
		int64_t content_length = 0;
		char *hptr = line_end + 2;
		char *hend = buf + header_len;
		while (hptr < hend) {
			char *eol = strstr(hptr, "\r\n");
			if (!eol || eol > hend) {
				break;
			}
			if (eol == hptr) {
				break;
			}
			char *colon = memchr(hptr, ':', (size_t)(eol - hptr));
			if (colon) {
				size_t klen = (size_t)(colon - hptr);
				char *val = colon + 1;
				while (val < eol && (*val == ' ' || *val == '\t')) val++;
				size_t vlen = (size_t)(eol - val);
				bazic_append_str(&headers, &headers_len, &headers_cap, hptr, klen);
				bazic_append_str(&headers, &headers_len, &headers_cap, ": ", 2);
				bazic_append_str(&headers, &headers_len, &headers_cap, val, vlen);
				bazic_append_str(&headers, &headers_len, &headers_cap, "\n", 1);
				if (klen == strlen("Content-Length") && bazic_strnicmp(hptr, "Content-Length", klen) == 0) {
					char *endp = NULL;
					long long v = strtoll(val, &endp, 10);
					if (endp && endp != val && v > 0) {
						content_length = (int64_t)v;
					}
				}
				if (klen == strlen("Cookie") && bazic_strnicmp(hptr, "Cookie", klen) == 0) {
					const char *c = val;
					const char *cend = eol;
					while (c < cend) {
						while (c < cend && (*c == ' ' || *c == '\t' || *c == ';')) c++;
						const char *eq = memchr(c, '=', (size_t)(cend - c));
						if (!eq) break;
						const char *vstart = eq + 1;
						const char *vend = vstart;
						while (vend < cend && *vend != ';') vend++;
						bazic_append_str(&cookies, &cookies_len, &cookies_cap, c, (size_t)(eq - c));
						bazic_append_str(&cookies, &cookies_len, &cookies_cap, "=", 1);
						bazic_append_str(&cookies, &cookies_len, &cookies_cap, vstart, (size_t)(vend - vstart));
						bazic_append_str(&cookies, &cookies_len, &cookies_cap, "\n", 1);
						c = vend;
					}
				}
			}
			hptr = eol + 2;
		}
		if (!headers) headers = bazic_strdup("");
		if (!cookies) cookies = bazic_strdup("");
		if (content_length < 0) content_length = 0;
		if (max_body > 0 && content_length > max_body) {
			free(buf);
			free(method);
			free(query);
			free(headers);
			free(cookies);
			free(path);
			bazic_send_simple_response(client, 413, "payload too large");
			closesocket(client);
			continue;
		}
		size_t body_in_buf = len - (header_len + 4);
		char *body = NULL;
		if (content_length > 0) {
			body = (char *)malloc((size_t)content_length + 1);
			if (!body) {
				free(buf);
				free(method);
				free(query);
				free(headers);
				free(cookies);
				free(path);
				bazic_send_simple_response(client, 500, "server error");
				closesocket(client);
				continue;
			}
			size_t copied = 0;
			if (body_in_buf > 0) {
				size_t ncopy = body_in_buf < (size_t)content_length ? body_in_buf : (size_t)content_length;
				memcpy(body, header_end + 4, ncopy);
				copied = ncopy;
			}
			while (copied < (size_t)content_length) {
				int n = recv(client, body + copied, (int)((size_t)content_length - copied), 0);
				if (n <= 0) break;
				copied += (size_t)n;
			}
			body[copied] = '\0';
		} else {
			body = bazic_strdup("");
		}
		free(buf);

		char remote[128];
		remote[0] = '\0';
		if (peer.ss_family == AF_INET) {
			struct sockaddr_in *sa = (struct sockaddr_in *)&peer;
			char ip[64];
			inet_ntop(AF_INET, &sa->sin_addr, ip, sizeof(ip));
			snprintf(remote, sizeof(remote), "%s:%d", ip, ntohs(sa->sin_port));
		}
		char *remote_addr = bazic_strdup(remote[0] ? remote : "");

		ServerResponse resp;
		int handled = 0;
		char *params_out = NULL;
		for (int64_t i = 0; i < __bazic_routes_len; i++) {
			BazicRoute *route = &__bazic_routes[i];
			if (route->method && strcmp(route->method, method) == 0) {
				if (bazic_match_route(path, route->path ? route->path : "", &params_out)) {
					ServerRequest req;
					req.method = method;
					req.path = path;
					req.query = query;
					req.headers = headers;
					req.body = body;
					req.remote_addr = remote_addr ? remote_addr : bazic_strdup("");
					req.cookies = cookies;
					req.params = params_out ? params_out : bazic_strdup("");
					resp = route->handler(req);
					handled = 1;
					break;
				}
			}
		}
		if (!handled) {
			resp.status = 404;
			resp.headers = bazic_strdup("Content-Type: text/plain; charset=utf-8");
			resp.body = bazic_strdup("not found");
		}
		if (resp.status <= 0) resp.status = 200;
		if (!resp.headers) resp.headers = bazic_strdup("");
		if (!resp.body) resp.body = bazic_strdup("");

		int has_len = bazic_headers_has(resp.headers, "Content-Length");
		int has_conn = bazic_headers_has(resp.headers, "Connection");
		char status_line[128];
		snprintf(status_line, sizeof(status_line), "HTTP/1.1 %lld %s\r\n", (long long)resp.status, bazic_status_text(resp.status));
		bazic_send_all_socket(client, status_line, (int)strlen(status_line));
		const char *hptr2 = resp.headers;
		while (*hptr2) {
			const char *line_end2 = strchr(hptr2, '\n');
			if (!line_end2) line_end2 = hptr2 + strlen(hptr2);
			size_t l2 = (size_t)(line_end2 - hptr2);
			if (l2 > 0) {
				char *line = bazic_strndup(hptr2, l2);
				if (line && line[l2 - 1] == '\r') line[l2 - 1] = '\0';
				if (line && line[0] != '\0') {
					bazic_send_all_socket(client, line, (int)strlen(line));
					bazic_send_all_socket(client, "\r\n", 2);
				}
				if (line) free(line);
			}
			if (*line_end2 == '\0') break;
			hptr2 = line_end2 + 1;
		}
		if (!has_len) {
			char lenbuf[64];
			snprintf(lenbuf, sizeof(lenbuf), "Content-Length: %u\r\n", (unsigned)strlen(resp.body));
			bazic_send_all_socket(client, lenbuf, (int)strlen(lenbuf));
		}
		if (!has_conn) {
			bazic_send_all_socket(client, "Connection: close\r\n", 19);
		}
		bazic_send_all_socket(client, "\r\n", 2);
		if (resp.body && resp.body[0] != '\0') {
			bazic_send_all_socket(client, resp.body, (int)strlen(resp.body));
		}
		if (params_out) {
			free(params_out);
		}
		closesocket(client);
		free(method);
		free(query);
		free(headers);
		free(cookies);
		free(body);
		free(path);
		free(remote_addr);
	}
	closesocket(server);
	WSACleanup();
	return ok_bool(true);
#else
	char host[256];
	char port[16];
	host[0] = '\0';
	port[0] = '\0';
	if (addr[0] == ':') {
		strcpy(host, "0.0.0.0");
		strncpy(port, addr + 1, sizeof(port) - 1);
		port[sizeof(port) - 1] = '\0';
	} else {
		const char *colon = strrchr(addr, ':');
		if (!colon) {
			return err_bool("http_serve_app: addr must be host:port or :port");
		}
		size_t hlen = (size_t)(colon - addr);
		if (hlen >= sizeof(host)) {
			return err_bool("http_serve_app: host too long");
		}
		memcpy(host, addr, hlen);
		host[hlen] = '\0';
		strncpy(port, colon + 1, sizeof(port) - 1);
		port[sizeof(port) - 1] = '\0';
		if (host[0] == '\0') {
			strcpy(host, "0.0.0.0");
		}
	}
	struct addrinfo hints;
	memset(&hints, 0, sizeof(hints));
	hints.ai_family = AF_INET;
	hints.ai_socktype = SOCK_STREAM;
	hints.ai_flags = AI_PASSIVE;
	struct addrinfo *res = NULL;
	if (getaddrinfo(host, port, &hints, &res) != 0) {
		return err_bool("http_serve_app: getaddrinfo failed");
	}
	int server = socket(res->ai_family, res->ai_socktype, res->ai_protocol);
	if (server < 0) {
		freeaddrinfo(res);
		return err_bool("http_serve_app: socket failed");
	}
	int opt = 1;
	setsockopt(server, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));
	if (bind(server, res->ai_addr, res->ai_addrlen) != 0) {
		close(server);
		freeaddrinfo(res);
		return err_bool("http_serve_app: bind failed");
	}
	freeaddrinfo(res);
	if (listen(server, 64) != 0) {
		close(server);
		return err_bool("http_serve_app: listen failed");
	}
	for (;;) {
		struct sockaddr_storage peer;
		socklen_t peerlen = sizeof(peer);
		int client = accept(server, (struct sockaddr *)&peer, &peerlen);
		if (client < 0) {
			close(server);
			return err_bool("http_serve_app: accept failed");
		}
		if (read_timeout_ms > 0) {
			struct timeval tv;
			tv.tv_sec = (int)(read_timeout_ms / 1000);
			tv.tv_usec = (int)((read_timeout_ms % 1000) * 1000);
			setsockopt(client, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));
		}
		if (write_timeout_ms > 0) {
			struct timeval tv;
			tv.tv_sec = (int)(write_timeout_ms / 1000);
			tv.tv_usec = (int)((write_timeout_ms % 1000) * 1000);
			setsockopt(client, SOL_SOCKET, SO_SNDTIMEO, &tv, sizeof(tv));
		}

		char *buf = NULL;
		size_t len = 0;
		size_t cap = 0;
		const size_t max_header = 65536;
		int header_found = 0;
		while (!header_found) {
			char tmp[4096];
			int n = recv(client, tmp, sizeof(tmp), 0);
			if (n <= 0) {
				break;
			}
			if (cap < len + (size_t)n + 1) {
				size_t next = cap == 0 ? 8192 : cap * 2;
				while (next < len + (size_t)n + 1) next *= 2;
				char *nbuf = (char *)realloc(buf, next);
				if (!nbuf) {
					free(buf);
					buf = NULL;
					break;
				}
				buf = nbuf;
				cap = next;
			}
			memcpy(buf + len, tmp, (size_t)n);
			len += (size_t)n;
			buf[len] = '\0';
			if (len >= 4 && strstr(buf, "\r\n\r\n")) {
				header_found = 1;
				break;
			}
			if (len > max_header) {
				break;
			}
		}
		if (!buf || !header_found) {
			if (buf) free(buf);
			bazic_send_simple_response(client, 400, "bad request");
			close(client);
			continue;
		}
		char *header_end = strstr(buf, "\r\n\r\n");
		size_t header_len = (size_t)(header_end - buf);
		char *line_end = strstr(buf, "\r\n");
		if (!line_end) {
			free(buf);
			bazic_send_simple_response(client, 400, "bad request");
			close(client);
			continue;
		}
		size_t line_len = (size_t)(line_end - buf);
		char *sp1 = memchr(buf, ' ', line_len);
		char *sp2 = sp1 ? memchr(sp1 + 1, ' ', (size_t)(buf + line_len - (sp1 + 1))) : NULL;
		if (!sp1 || !sp2) {
			free(buf);
			bazic_send_simple_response(client, 400, "bad request");
			close(client);
			continue;
		}
		char *method = bazic_strndup(buf, (size_t)(sp1 - buf));
		char *target = bazic_strndup(sp1 + 1, (size_t)(sp2 - sp1 - 1));
		if (!method || !target) {
			free(buf);
			free(method);
			free(target);
			bazic_send_simple_response(client, 500, "server error");
			close(client);
			continue;
		}
		bazic_upper_inplace(method);
		char *query = NULL;
		char *qmark = strchr(target, '?');
		if (qmark) {
			query = bazic_strdup(qmark + 1);
			*qmark = '\0';
		} else {
			query = bazic_strdup("");
		}
		if (!query) query = bazic_strdup("");
		char *path = target;
		if (path[0] == '\0') {
			free(path);
			path = bazic_strdup("/");
		}
		if (path) {
			bazic_normalize_path(path);
		}

		char *headers = NULL;
		size_t headers_len = 0;
		size_t headers_cap = 0;
		char *cookies = NULL;
		size_t cookies_len = 0;
		size_t cookies_cap = 0;
		int64_t content_length = 0;
		char *hptr = line_end + 2;
		char *hend = buf + header_len;
		while (hptr < hend) {
			char *eol = strstr(hptr, "\r\n");
			if (!eol || eol > hend) {
				break;
			}
			if (eol == hptr) {
				break;
			}
			char *colon = memchr(hptr, ':', (size_t)(eol - hptr));
			if (colon) {
				size_t klen = (size_t)(colon - hptr);
				char *val = colon + 1;
				while (val < eol && (*val == ' ' || *val == '\t')) val++;
				size_t vlen = (size_t)(eol - val);
				bazic_append_str(&headers, &headers_len, &headers_cap, hptr, klen);
				bazic_append_str(&headers, &headers_len, &headers_cap, ": ", 2);
				bazic_append_str(&headers, &headers_len, &headers_cap, val, vlen);
				bazic_append_str(&headers, &headers_len, &headers_cap, "\n", 1);
				if (klen == strlen("Content-Length") && bazic_strnicmp(hptr, "Content-Length", klen) == 0) {
					char *endp = NULL;
					long long v = strtoll(val, &endp, 10);
					if (endp && endp != val && v > 0) {
						content_length = (int64_t)v;
					}
				}
				if (klen == strlen("Cookie") && bazic_strnicmp(hptr, "Cookie", klen) == 0) {
					const char *c = val;
					const char *cend = eol;
					while (c < cend) {
						while (c < cend && (*c == ' ' || *c == '\t' || *c == ';')) c++;
						const char *eq = memchr(c, '=', (size_t)(cend - c));
						if (!eq) break;
						const char *vstart = eq + 1;
						const char *vend = vstart;
						while (vend < cend && *vend != ';') vend++;
						bazic_append_str(&cookies, &cookies_len, &cookies_cap, c, (size_t)(eq - c));
						bazic_append_str(&cookies, &cookies_len, &cookies_cap, "=", 1);
						bazic_append_str(&cookies, &cookies_len, &cookies_cap, vstart, (size_t)(vend - vstart));
						bazic_append_str(&cookies, &cookies_len, &cookies_cap, "\n", 1);
						c = vend;
					}
				}
			}
			hptr = eol + 2;
		}
		if (!headers) headers = bazic_strdup("");
		if (!cookies) cookies = bazic_strdup("");
		if (content_length < 0) content_length = 0;
		if (max_body > 0 && content_length > max_body) {
			free(buf);
			free(method);
			free(query);
			free(headers);
			free(cookies);
			free(path);
			bazic_send_simple_response(client, 413, "payload too large");
			close(client);
			continue;
		}
		size_t body_in_buf = len - (header_len + 4);
		char *body = NULL;
		if (content_length > 0) {
			body = (char *)malloc((size_t)content_length + 1);
			if (!body) {
				free(buf);
				free(method);
				free(query);
				free(headers);
				free(cookies);
				free(path);
				bazic_send_simple_response(client, 500, "server error");
				close(client);
				continue;
			}
			size_t copied = 0;
			if (body_in_buf > 0) {
				size_t ncopy = body_in_buf < (size_t)content_length ? body_in_buf : (size_t)content_length;
				memcpy(body, header_end + 4, ncopy);
				copied = ncopy;
			}
			while (copied < (size_t)content_length) {
				int n = recv(client, body + copied, (int)((size_t)content_length - copied), 0);
				if (n <= 0) break;
				copied += (size_t)n;
			}
			body[copied] = '\0';
		} else {
			body = bazic_strdup("");
		}
		free(buf);

		char remote[128];
		remote[0] = '\0';
		if (peer.ss_family == AF_INET) {
			struct sockaddr_in *sa = (struct sockaddr_in *)&peer;
			char ip[64];
			inet_ntop(AF_INET, &sa->sin_addr, ip, sizeof(ip));
			snprintf(remote, sizeof(remote), "%s:%d", ip, ntohs(sa->sin_port));
		}
		char *remote_addr = bazic_strdup(remote[0] ? remote : "");

		ServerResponse resp;
		int handled = 0;
		char *params_out = NULL;
		for (int64_t i = 0; i < __bazic_routes_len; i++) {
			BazicRoute *route = &__bazic_routes[i];
			if (route->method && strcmp(route->method, method) == 0) {
				if (bazic_match_route(path, route->path ? route->path : "", &params_out)) {
					ServerRequest req;
					req.method = method;
					req.path = path;
					req.query = query;
					req.headers = headers;
					req.body = body;
					req.remote_addr = remote_addr ? remote_addr : bazic_strdup("");
					req.cookies = cookies;
					req.params = params_out ? params_out : bazic_strdup("");
					resp = route->handler(req);
					handled = 1;
					break;
				}
			}
		}
		if (!handled) {
			resp.status = 404;
			resp.headers = bazic_strdup("Content-Type: text/plain; charset=utf-8");
			resp.body = bazic_strdup("not found");
		}
		if (resp.status <= 0) resp.status = 200;
		if (!resp.headers) resp.headers = bazic_strdup("");
		if (!resp.body) resp.body = bazic_strdup("");

		int has_len = bazic_headers_has(resp.headers, "Content-Length");
		int has_conn = bazic_headers_has(resp.headers, "Connection");
		char status_line[128];
		snprintf(status_line, sizeof(status_line), "HTTP/1.1 %lld %s\r\n", (long long)resp.status, bazic_status_text(resp.status));
		bazic_send_all_socket(client, status_line, (int)strlen(status_line));
		const char *hptr2 = resp.headers;
		while (*hptr2) {
			const char *line_end2 = strchr(hptr2, '\n');
			if (!line_end2) line_end2 = hptr2 + strlen(hptr2);
			size_t l2 = (size_t)(line_end2 - hptr2);
			if (l2 > 0) {
				char *line = bazic_strndup(hptr2, l2);
				if (line && line[l2 - 1] == '\r') line[l2 - 1] = '\0';
				if (line && line[0] != '\0') {
					bazic_send_all_socket(client, line, (int)strlen(line));
					bazic_send_all_socket(client, "\r\n", 2);
				}
				if (line) free(line);
			}
			if (*line_end2 == '\0') break;
			hptr2 = line_end2 + 1;
		}
		if (!has_len) {
			char lenbuf[64];
			snprintf(lenbuf, sizeof(lenbuf), "Content-Length: %u\r\n", (unsigned)strlen(resp.body));
			bazic_send_all_socket(client, lenbuf, (int)strlen(lenbuf));
		}
		if (!has_conn) {
			bazic_send_all_socket(client, "Connection: close\r\n", 19);
		}
		bazic_send_all_socket(client, "\r\n", 2);
		if (resp.body && resp.body[0] != '\0') {
			bazic_send_all_socket(client, resp.body, (int)strlen(resp.body));
		}
		if (params_out) {
			free(params_out);
		}
		close(client);
		free(method);
		free(query);
		free(headers);
		free(cookies);
		free(body);
		free(path);
		free(remote_addr);
	}
	close(server);
	return ok_bool(true);
#endif
}
Result_bool_Error __std_http_serve_text(char *addr, char *body) {
#ifdef _WIN32
	if (!addr) {
		return err_bool("http_serve_text: null addr");
	}
	if (!body) {
		body = "";
	}
	char host[256];
	char port[16];
	host[0] = '\0';
	port[0] = '\0';
	if (addr[0] == ':') {
		strcpy(host, "0.0.0.0");
		strncpy(port, addr + 1, sizeof(port) - 1);
		port[sizeof(port) - 1] = '\0';
	} else {
		const char *colon = strrchr(addr, ':');
		if (!colon) {
			return err_bool("http_serve_text: addr must be host:port or :port");
		}
		size_t hlen = (size_t)(colon - addr);
		if (hlen >= sizeof(host)) {
			return err_bool("http_serve_text: host too long");
		}
		memcpy(host, addr, hlen);
		host[hlen] = '\0';
		strncpy(port, colon + 1, sizeof(port) - 1);
		port[sizeof(port) - 1] = '\0';
		if (host[0] == '\0') {
			strcpy(host, "0.0.0.0");
		}
	}
	WSADATA wsa;
	if (WSAStartup(MAKEWORD(2, 2), &wsa) != 0) {
		return err_bool("http_serve_text: WSAStartup failed");
	}
	struct addrinfo hints;
	memset(&hints, 0, sizeof(hints));
	hints.ai_family = AF_INET;
	hints.ai_socktype = SOCK_STREAM;
	hints.ai_protocol = IPPROTO_TCP;
	hints.ai_flags = AI_PASSIVE;
	struct addrinfo *res = NULL;
	if (getaddrinfo(host, port, &hints, &res) != 0) {
		WSACleanup();
		return err_bool("http_serve_text: getaddrinfo failed");
	}
	SOCKET server = socket(res->ai_family, res->ai_socktype, res->ai_protocol);
	if (server == INVALID_SOCKET) {
		freeaddrinfo(res);
		WSACleanup();
		return err_bool("http_serve_text: socket failed");
	}
	int opt = 1;
	setsockopt(server, SOL_SOCKET, SO_REUSEADDR, (const char *)&opt, sizeof(opt));
	if (bind(server, res->ai_addr, (int)res->ai_addrlen) == SOCKET_ERROR) {
		closesocket(server);
		freeaddrinfo(res);
		WSACleanup();
		return err_bool("http_serve_text: bind failed");
	}
	freeaddrinfo(res);
	if (listen(server, 10) == SOCKET_ERROR) {
		closesocket(server);
		WSACleanup();
		return err_bool("http_serve_text: listen failed");
	}
	for (;;) {
		SOCKET client = accept(server, NULL, NULL);
		if (client == INVALID_SOCKET) {
			closesocket(server);
			WSACleanup();
			return err_bool("http_serve_text: accept failed");
		}
		char reqbuf[1024];
		recv(client, reqbuf, sizeof(reqbuf), 0);
		char header[256];
		size_t bodyLen = strlen(body);
		snprintf(header, sizeof(header),
			"HTTP/1.1 200 OK\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: %u\r\nConnection: close\r\n\r\n",
			(unsigned)bodyLen);
		send(client, header, (int)strlen(header), 0);
		if (bodyLen > 0) {
			send(client, body, (int)bodyLen, 0);
		}
		closesocket(client);
	}
	closesocket(server);
	WSACleanup();
	return ok_bool(true);
#else
	if (!addr) {
		return err_bool("http_serve_text: null addr");
	}
	if (!body) {
		body = "";
	}
	char host[256];
	char port[16];
	host[0] = '\0';
	port[0] = '\0';
	if (addr[0] == ':') {
		strcpy(host, "0.0.0.0");
		strncpy(port, addr + 1, sizeof(port) - 1);
		port[sizeof(port) - 1] = '\0';
	} else {
		const char *colon = strrchr(addr, ':');
		if (!colon) {
			return err_bool("http_serve_text: addr must be host:port or :port");
		}
		size_t hlen = (size_t)(colon - addr);
		if (hlen >= sizeof(host)) {
			return err_bool("http_serve_text: host too long");
		}
		memcpy(host, addr, hlen);
		host[hlen] = '\0';
		strncpy(port, colon + 1, sizeof(port) - 1);
		port[sizeof(port) - 1] = '\0';
		if (host[0] == '\0') {
			strcpy(host, "0.0.0.0");
		}
	}
	struct addrinfo hints;
	memset(&hints, 0, sizeof(hints));
	hints.ai_family = AF_INET;
	hints.ai_socktype = SOCK_STREAM;
	hints.ai_flags = AI_PASSIVE;
	struct addrinfo *res = NULL;
	if (getaddrinfo(host, port, &hints, &res) != 0) {
		return err_bool("http_serve_text: getaddrinfo failed");
	}
	int server = socket(res->ai_family, res->ai_socktype, res->ai_protocol);
	if (server < 0) {
		freeaddrinfo(res);
		return err_bool("http_serve_text: socket failed");
	}
	int opt = 1;
	setsockopt(server, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));
	if (bind(server, res->ai_addr, res->ai_addrlen) != 0) {
		close(server);
		freeaddrinfo(res);
		return err_bool("http_serve_text: bind failed");
	}
	freeaddrinfo(res);
	if (listen(server, 10) != 0) {
		close(server);
		return err_bool("http_serve_text: listen failed");
	}
	for (;;) {
		int client = accept(server, NULL, NULL);
		if (client < 0) {
			close(server);
			return err_bool("http_serve_text: accept failed");
		}
		char reqbuf[1024];
		recv(client, reqbuf, sizeof(reqbuf), 0);
		char header[256];
		size_t bodyLen = strlen(body);
		snprintf(header, sizeof(header),
			"HTTP/1.1 200 OK\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: %u\r\nConnection: close\r\n\r\n",
			(unsigned)bodyLen);
		send(client, header, strlen(header), 0);
		if (bodyLen > 0) {
			send(client, body, bodyLen, 0);
		}
		close(client);
	}
	close(server);
	return ok_bool(true);
#endif
}

static void sha256_transform(uint32_t state[8], const uint8_t data[64]);

static const uint32_t k256[64] = {
	0x428a2f98U, 0x71374491U, 0xb5c0fbcfU, 0xe9b5dba5U,
	0x3956c25bU, 0x59f111f1U, 0x923f82a4U, 0xab1c5ed5U,
	0xd807aa98U, 0x12835b01U, 0x243185beU, 0x550c7dc3U,
	0x72be5d74U, 0x80deb1feU, 0x9bdc06a7U, 0xc19bf174U,
	0xe49b69c1U, 0xefbe4786U, 0x0fc19dc6U, 0x240ca1ccU,
	0x2de92c6fU, 0x4a7484aaU, 0x5cb0a9dcU, 0x76f988daU,
	0x983e5152U, 0xa831c66dU, 0xb00327c8U, 0xbf597fc7U,
	0xc6e00bf3U, 0xd5a79147U, 0x06ca6351U, 0x14292967U,
	0x27b70a85U, 0x2e1b2138U, 0x4d2c6dfcU, 0x53380d13U,
	0x650a7354U, 0x766a0abbU, 0x81c2c92eU, 0x92722c85U,
	0xa2bfe8a1U, 0xa81a664bU, 0xc24b8b70U, 0xc76c51a3U,
	0xd192e819U, 0xd6990624U, 0xf40e3585U, 0x106aa070U,
	0x19a4c116U, 0x1e376c08U, 0x2748774cU, 0x34b0bcb5U,
	0x391c0cb3U, 0x4ed8aa4aU, 0x5b9cca4fU, 0x682e6ff3U,
	0x748f82eeU, 0x78a5636fU, 0x84c87814U, 0x8cc70208U,
	0x90befffaU, 0xa4506cebU, 0xbef9a3f7U, 0xc67178f2U
};

static uint32_t rotr(uint32_t x, uint32_t n) {
	return (x >> n) | (x << (32 - n));
}

static void sha256_transform(uint32_t state[8], const uint8_t data[64]) {
	uint32_t m[64];
	for (int i = 0; i < 16; i++) {
		m[i] = (uint32_t)data[i * 4] << 24 |
		       (uint32_t)data[i * 4 + 1] << 16 |
		       (uint32_t)data[i * 4 + 2] << 8 |
		       (uint32_t)data[i * 4 + 3];
	}
	for (int i = 16; i < 64; i++) {
		uint32_t s0 = rotr(m[i - 15], 7) ^ rotr(m[i - 15], 18) ^ (m[i - 15] >> 3);
		uint32_t s1 = rotr(m[i - 2], 17) ^ rotr(m[i - 2], 19) ^ (m[i - 2] >> 10);
		m[i] = m[i - 16] + s0 + m[i - 7] + s1;
	}
	uint32_t a = state[0];
	uint32_t b = state[1];
	uint32_t c = state[2];
	uint32_t d = state[3];
	uint32_t e = state[4];
	uint32_t f = state[5];
	uint32_t g = state[6];
	uint32_t h = state[7];
	for (int i = 0; i < 64; i++) {
		uint32_t S1 = rotr(e, 6) ^ rotr(e, 11) ^ rotr(e, 25);
		uint32_t ch = (e & f) ^ ((~e) & g);
		uint32_t temp1 = h + S1 + ch + k256[i] + m[i];
		uint32_t S0 = rotr(a, 2) ^ rotr(a, 13) ^ rotr(a, 22);
		uint32_t maj = (a & b) ^ (a & c) ^ (b & c);
		uint32_t temp2 = S0 + maj;
		h = g;
		g = f;
		f = e;
		e = d + temp1;
		d = c;
		c = b;
		b = a;
		a = temp1 + temp2;
	}
	state[0] += a;
	state[1] += b;
	state[2] += c;
	state[3] += d;
	state[4] += e;
	state[5] += f;
	state[6] += g;
	state[7] += h;
}

static void sha256(const uint8_t *data, size_t len, uint8_t out[32]) {
	uint32_t state[8] = {
		0x6a09e667U, 0xbb67ae85U, 0x3c6ef372U, 0xa54ff53aU,
		0x510e527fU, 0x9b05688cU, 0x1f83d9abU, 0x5be0cd19U
	};
	uint8_t block[64];
	size_t i = 0;
	while (len - i >= 64) {
		sha256_transform(state, data + i);
		i += 64;
	}
	size_t rem = len - i;
	memset(block, 0, 64);
	if (rem > 0) {
		memcpy(block, data + i, rem);
	}
	block[rem] = 0x80;
	if (rem >= 56) {
		sha256_transform(state, block);
		memset(block, 0, 64);
	}
	uint64_t bits = (uint64_t)len * 8ULL;
	block[63] = (uint8_t)(bits);
	block[62] = (uint8_t)(bits >> 8);
	block[61] = (uint8_t)(bits >> 16);
	block[60] = (uint8_t)(bits >> 24);
	block[59] = (uint8_t)(bits >> 32);
	block[58] = (uint8_t)(bits >> 40);
	block[57] = (uint8_t)(bits >> 48);
	block[56] = (uint8_t)(bits >> 56);
	sha256_transform(state, block);
	for (int j = 0; j < 8; j++) {
		out[j * 4] = (uint8_t)(state[j] >> 24);
		out[j * 4 + 1] = (uint8_t)(state[j] >> 16);
		out[j * 4 + 2] = (uint8_t)(state[j] >> 8);
		out[j * 4 + 3] = (uint8_t)(state[j]);
	}
}

static void hmac_sha256(const uint8_t *key, size_t key_len, const uint8_t *data, size_t data_len, uint8_t out[32]) {
	uint8_t key_block[64];
	memset(key_block, 0, sizeof(key_block));
	if (key_len > 64) {
		uint8_t hashed[32];
		sha256(key, key_len, hashed);
		memcpy(key_block, hashed, 32);
	} else if (key_len > 0) {
		memcpy(key_block, key, key_len);
	}
	uint8_t o_key_pad[64];
	uint8_t i_key_pad[64];
	for (int i = 0; i < 64; i++) {
		o_key_pad[i] = (uint8_t)(key_block[i] ^ 0x5c);
		i_key_pad[i] = (uint8_t)(key_block[i] ^ 0x36);
	}
	size_t inner_len = 64 + data_len;
	uint8_t *inner = (uint8_t *)malloc(inner_len);
	if (!inner) {
		memset(out, 0, 32);
		return;
	}
	memcpy(inner, i_key_pad, 64);
	if (data_len > 0) {
		memcpy(inner + 64, data, data_len);
	}
	uint8_t inner_hash[32];
	sha256(inner, inner_len, inner_hash);
	free(inner);
	uint8_t outer[96];
	memcpy(outer, o_key_pad, 64);
	memcpy(outer + 64, inner_hash, 32);
	sha256(outer, sizeof(outer), out);
}

char *__std_sha256_hex(char *s) {
	if (!s) {
		return bazic_strdup("");
	}
	uint8_t hash[32];
	sha256((const uint8_t *)s, strlen(s), hash);
static const char *hex = "0123456789abcdef";
	char *out = (char *)malloc(65);
	if (!out) {
		return bazic_strdup("");
	}
	for (int i = 0; i < 32; i++) {
		out[i * 2] = hex[(hash[i] >> 4) & 0xF];
		out[i * 2 + 1] = hex[hash[i] & 0xF];
	}
	out[64] = '\0';
	return out;
}

char *__std_hmac_sha256_hex(char *message, char *secret) {
	if (!message) {
		message = "";
	}
	if (!secret) {
		secret = "";
	}
	uint8_t digest[32];
	hmac_sha256((const uint8_t *)secret, strlen(secret), (const uint8_t *)message, strlen(message), digest);
	static const char *hex = "0123456789abcdef";
	char *out = (char *)malloc(65);
	if (!out) {
		return bazic_strdup("");
	}
	for (int i = 0; i < 32; i++) {
		out[i * 2] = hex[(digest[i] >> 4) & 0xF];
		out[i * 2 + 1] = hex[digest[i] & 0xF];
	}
	out[64] = '\0';
	return out;
}

static bool bazic_secure_random(uint8_t *buf, size_t bytes) {
	if (!buf || bytes == 0) {
		return true;
	}
#ifdef _WIN32
	return BCryptGenRandom(NULL, buf, (ULONG)bytes, BCRYPT_USE_SYSTEM_PREFERRED_RNG) == 0;
#elif defined(__APPLE__) || defined(__FreeBSD__) || defined(__NetBSD__) || defined(__OpenBSD__)
	arc4random_buf(buf, bytes);
	return true;
#else
	int fd = open("/dev/urandom", O_RDONLY);
	if (fd < 0) {
		return false;
	}
	size_t off = 0;
	while (off < bytes) {
		ssize_t n = read(fd, buf + off, bytes - off);
		if (n <= 0) {
			close(fd);
			return false;
		}
		off += (size_t)n;
	}
	close(fd);
	return true;
#endif
}

Result_string_Error __std_random_hex(int64_t n) {
	if (n <= 0) {
		return ok_string("");
	}
	size_t bytes = (size_t)n;
	uint8_t *buf = (uint8_t *)malloc(bytes);
	if (!buf) {
		return err_string("random_hex: out of memory");
	}
	if (!bazic_secure_random(buf, bytes)) {
		free(buf);
		return err_string("random_hex: secure rng unavailable");
	}
	char *out = (char *)malloc(bytes * 2 + 1);
	if (!out) {
		free(buf);
		return err_string("random_hex: out of memory");
	}
	static const char *hex = "0123456789abcdef";
	for (size_t i = 0; i < bytes; i++) {
		out[i * 2] = hex[(buf[i] >> 4) & 0xF];
		out[i * 2 + 1] = hex[buf[i] & 0xF];
	}
	out[bytes * 2] = '\0';
	free(buf);
	return ok_string_owned(out);
}

Result_string_Error __std_bcrypt_hash(char *password, int64_t cost) {
	(void)password;
	(void)cost;
	return err_string("bcrypt not supported in llvm backend");
}

Result_bool_Error __std_bcrypt_verify(char *password, char *hash) {
	(void)password;
	(void)hash;
	return err_bool("bcrypt not supported in llvm backend");
}

Result_string_Error __std_jwt_sign_hs256(char *header_json, char *payload_json, char *secret) {
	if (!header_json) {
		header_json = "";
	}
	if (!payload_json) {
		payload_json = "";
	}
	if (!secret) {
		secret = "";
	}
	char *header = bazic_base64url_encode_bytes((const uint8_t *)header_json, strlen(header_json));
	if (!header) {
		return err_string("jwt_sign_hs256: out of memory");
	}
	char *payload = bazic_base64url_encode_bytes((const uint8_t *)payload_json, strlen(payload_json));
	if (!payload) {
		free(header);
		return err_string("jwt_sign_hs256: out of memory");
	}
	size_t header_len = strlen(header);
	size_t payload_len = strlen(payload);
	size_t signing_len = header_len + 1 + payload_len;
	char *signing = (char *)malloc(signing_len + 1);
	if (!signing) {
		free(header);
		free(payload);
		return err_string("jwt_sign_hs256: out of memory");
	}
	memcpy(signing, header, header_len);
	signing[header_len] = '.';
	memcpy(signing + header_len + 1, payload, payload_len);
	signing[signing_len] = '\0';
	uint8_t digest[32];
	hmac_sha256((const uint8_t *)secret, strlen(secret), (const uint8_t *)signing, signing_len, digest);
	char *sig = bazic_base64url_encode_bytes(digest, 32);
	if (!sig) {
		free(header);
		free(payload);
		free(signing);
		return err_string("jwt_sign_hs256: out of memory");
	}
	size_t sig_len = strlen(sig);
	size_t token_len = signing_len + 1 + sig_len;
	char *token = (char *)malloc(token_len + 1);
	if (!token) {
		free(header);
		free(payload);
		free(signing);
		free(sig);
		return err_string("jwt_sign_hs256: out of memory");
	}
	memcpy(token, signing, signing_len);
	token[signing_len] = '.';
	memcpy(token + signing_len + 1, sig, sig_len);
	token[token_len] = '\0';
	free(header);
	free(payload);
	free(signing);
	free(sig);
	return ok_string_owned(token);
}

Result_bool_Error __std_jwt_verify_hs256(char *token, char *secret) {
	if (!token || token[0] == '\0') {
		return err_bool("invalid token");
	}
	if (!secret) {
		secret = "";
	}
	char *first = strchr(token, '.');
	if (!first) {
		return err_bool("invalid token");
	}
	char *second = strchr(first + 1, '.');
	if (!second) {
		return err_bool("invalid token");
	}
	if (strchr(second + 1, '.') != NULL) {
		return err_bool("invalid token");
	}
	size_t signing_len = (size_t)(second - token);
	char *signing = (char *)malloc(signing_len + 1);
	if (!signing) {
		return err_bool("jwt_verify_hs256: out of memory");
	}
	memcpy(signing, token, signing_len);
	signing[signing_len] = '\0';
	char *sig = second + 1;
	uint8_t digest[32];
	hmac_sha256((const uint8_t *)secret, strlen(secret), (const uint8_t *)signing, signing_len, digest);
	char *expected = bazic_base64url_encode_bytes(digest, 32);
	free(signing);
	if (!expected) {
		return err_bool("jwt_verify_hs256: out of memory");
	}
	size_t sig_len = strlen(sig);
	size_t expected_len = strlen(expected);
	bool match = false;
	if (sig_len == expected_len) {
		unsigned char diff = 0;
		for (size_t i = 0; i < sig_len; i++) {
			diff |= (unsigned char)(sig[i] ^ expected[i]);
		}
		match = (diff == 0);
	}
	free(expected);
	return ok_bool(match);
}

Result_bool_Error __std_session_init(char *path) {
	if (!path || path[0] == '\0') {
		return err_bool("session_init: empty path");
	}
	if (strcmp(path, "memory") == 0 || strcmp(path, "memory:") == 0) {
		return ok_bool(true);
	}
	return err_bool("session_init: only memory store supported in llvm backend");
}

Result_bool_Error __std_session_put(char *path, char *token_hash, char *user_id, char *expires_at) {
	if (!path || path[0] == '\0') {
		return err_bool("session_put: empty path");
	}
	if (!token_hash || token_hash[0] == '\0') {
		return err_bool("session_put: empty token");
	}
	if (strcmp(path, "memory") != 0 && strcmp(path, "memory:") != 0) {
		return err_bool("session_put: only memory store supported in llvm backend");
	}
	time_t exp = 0;
	if (expires_at && expires_at[0] != '\0') {
		int year, month, day, hour, minute, second;
		if (sscanf(expires_at, "%4d-%2d-%2dT%2d:%2d:%2dZ", &year, &month, &day, &hour, &minute, &second) != 6) {
			return err_bool("session_put: invalid expires_at");
		}
		struct tm tmv;
		memset(&tmv, 0, sizeof(tmv));
		tmv.tm_year = year - 1900;
		tmv.tm_mon = month - 1;
		tmv.tm_mday = day;
		tmv.tm_hour = hour;
		tmv.tm_min = minute;
		tmv.tm_sec = second;
		tmv.tm_isdst = 0;
		exp = bazic_timegm(&tmv);
	}
	BazicSessionEntry *prev = NULL;
	BazicSessionEntry *cur = bazic_session_head;
	while (cur) {
		if (strcmp(cur->token, token_hash) == 0) {
			break;
		}
		prev = cur;
		cur = cur->next;
	}
	if (!cur) {
		cur = (BazicSessionEntry *)malloc(sizeof(BazicSessionEntry));
		if (!cur) {
			return err_bool("session_put: out of memory");
		}
		memset(cur, 0, sizeof(BazicSessionEntry));
		cur->token = bazic_strdup(token_hash);
		cur->next = bazic_session_head;
		bazic_session_head = cur;
	}
	if (cur->user) {
		free(cur->user);
	}
	cur->user = bazic_strdup(user_id ? user_id : "");
	cur->expires = exp;
	return ok_bool(true);
}

Result_string_Error __std_session_get_user(char *path, char *token_hash) {
	if (!path || path[0] == '\0') {
		return err_string("session_get_user: empty path");
	}
	if (!token_hash || token_hash[0] == '\0') {
		return err_string("session_get_user: empty token");
	}
	if (strcmp(path, "memory") != 0 && strcmp(path, "memory:") != 0) {
		return err_string("session_get_user: only memory store supported in llvm backend");
	}
	BazicSessionEntry *prev = NULL;
	BazicSessionEntry *cur = bazic_session_head;
	while (cur) {
		if (strcmp(cur->token, token_hash) == 0) {
			break;
		}
		prev = cur;
		cur = cur->next;
	}
	if (!cur) {
		return err_string("session_get_user: not found");
	}
	if (cur->expires != 0) {
		time_t now = time(NULL);
		if (now > cur->expires) {
			if (prev) {
				prev->next = cur->next;
			} else {
				bazic_session_head = cur->next;
			}
			free(cur->token);
			free(cur->user);
			free(cur);
			return err_string("session_get_user: expired");
		}
	}
	return ok_string(cur->user ? cur->user : "");
}

Result_bool_Error __std_session_delete(char *path, char *token_hash) {
	if (!path || path[0] == '\0') {
		return err_bool("session_delete: empty path");
	}
	if (!token_hash || token_hash[0] == '\0') {
		return err_bool("session_delete: empty token");
	}
	if (strcmp(path, "memory") != 0 && strcmp(path, "memory:") != 0) {
		return err_bool("session_delete: only memory store supported in llvm backend");
	}
	BazicSessionEntry *prev = NULL;
	BazicSessionEntry *cur = bazic_session_head;
	while (cur) {
		if (strcmp(cur->token, token_hash) == 0) {
			break;
		}
		prev = cur;
		cur = cur->next;
	}
	if (!cur) {
		return ok_bool(true);
	}
	if (prev) {
		prev->next = cur->next;
	} else {
		bazic_session_head = cur->next;
	}
	free(cur->token);
	free(cur->user);
	free(cur);
	return ok_bool(true);
}

static char *bazic_kv_get(const char *kv, const char *key, char sep) {
	if (!kv || !key || key[0] == '\0') {
		return bazic_strdup("");
	}
	size_t keylen = strlen(key);
	const char *p = kv;
	while (*p) {
		const char *line_end = strchr(p, '\n');
		if (!line_end) {
			line_end = p + strlen(p);
		}
		const char *eq = memchr(p, sep, (size_t)(line_end - p));
		if (eq && (size_t)(eq - p) == keylen && strncmp(p, key, keylen) == 0) {
			const char *val = eq + 1;
			size_t vlen = (size_t)(line_end - val);
			return bazic_strndup(val, vlen);
		}
		if (*line_end == '\0') {
			break;
		}
		p = line_end + 1;
	}
	return bazic_strdup("");
}

char *__std_kv_get(char *kv, char *key) {
	return bazic_kv_get(kv, key, '=');
}

char *__std_header_get(char *headers, char *key) {
	if (!headers || !key || key[0] == '\0') {
		return bazic_strdup("");
	}
	size_t keylen = strlen(key);
	const char *p = headers;
	while (*p) {
		const char *line_end = strchr(p, '\n');
		if (!line_end) {
			line_end = p + strlen(p);
		}
		const char *colon = memchr(p, ':', (size_t)(line_end - p));
		if (colon) {
			size_t klen = (size_t)(colon - p);
			while (klen > 0 && p[klen - 1] == ' ') {
				klen--;
			}
			if (klen == keylen && bazic_strnicmp(p, key, klen) == 0) {
				const char *val = colon + 1;
				while (val < line_end && *val == ' ') {
					val++;
				}
				size_t vlen = (size_t)(line_end - val);
				return bazic_strndup(val, vlen);
			}
		}
		if (*line_end == '\0') {
			break;
		}
		p = line_end + 1;
	}
	return bazic_strdup("");
}

char *__std_query_get(char *query, char *key) {
	if (!query || !key || key[0] == '\0') {
		return bazic_strdup("");
	}
	size_t keylen = strlen(key);
	const char *p = query;
	while (*p) {
		const char *amp = strchr(p, '&');
		if (!amp) {
			amp = p + strlen(p);
		}
		const char *eq = memchr(p, '=', (size_t)(amp - p));
		if (eq && (size_t)(eq - p) == keylen && strncmp(p, key, keylen) == 0) {
			const char *val = eq + 1;
			size_t vlen = (size_t)(amp - val);
			return bazic_strndup(val, vlen);
		}
		if (*amp == '\0') {
			break;
		}
		p = amp + 1;
	}
	return bazic_strdup("");
}

void __bazic_set_args(int argc, char **argv) {
	bazic_argc = argc;
	bazic_argv = argv;
}

char *__std_args(void) {
	if (bazic_argc <= 1 || bazic_argv == NULL) {
		return bazic_strdup("");
	}
	size_t count = (size_t)(bazic_argc - 1);
	size_t total = 0;
	for (size_t i = 0; i < count; i++) {
		char *arg = bazic_argv[i + 1];
		if (arg) {
			total += strlen(arg);
		}
		if (i + 1 < count) {
			total += 1;
		}
	}
	char *out = (char *)malloc(total + 1);
	if (!out) {
		return bazic_strdup("");
	}
	size_t off = 0;
	for (size_t i = 0; i < count; i++) {
		char *arg = bazic_argv[i + 1];
		if (arg) {
			size_t n = strlen(arg);
			memcpy(out + off, arg, n);
			off += n;
		}
		if (i + 1 < count) {
			out[off++] = '\n';
		}
	}
	out[off] = '\0';
	return out;
}

Result_string_Error __std_getenv(char *key) {
	if (!key || key[0] == '\0') {
		return err_string("env: empty key");
	}
	const char *val = getenv(key);
	if (!val) {
		return err_string("env: not set");
	}
	return ok_string(val);
}

Result_string_Error __std_cwd(void) {
	char *buf = NULL;
#ifdef _WIN32
	buf = _getcwd(NULL, 0);
#else
	buf = getcwd(NULL, 0);
#endif
	if (!buf) {
		return err_string("cwd: failed");
	}
	Result_string_Error r = ok_string_owned(buf);
	return r;
}

Result_bool_Error __std_chdir(char *path) {
	if (!path || path[0] == '\0') {
		return err_bool("chdir: empty path");
	}
#ifdef _WIN32
	if (_chdir(path) != 0) {
		return err_bool("chdir: failed");
	}
#else
	if (chdir(path) != 0) {
		return err_bool("chdir: failed");
	}
#endif
	return ok_bool(true);
}

Result_string_Error __std_env_list(void) {
	char *out = NULL;
	size_t len = 0;
	size_t cap = 0;
#ifdef _WIN32
	LPCH block = GetEnvironmentStringsA();
	if (!block) {
		return err_string("env: failed");
	}
	for (LPCH p = block; *p != '\0'; p += strlen(p) + 1) {
		size_t n = strlen(p);
		append_str(&out, &cap, &len, p);
		if (n > 0) {
			append_str(&out, &cap, &len, "\n");
		}
	}
	FreeEnvironmentStringsA(block);
#else
	extern char **environ;
	if (!environ) {
		return ok_string("");
	}
	for (char **p = environ; *p != NULL; p++) {
		append_str(&out, &cap, &len, *p);
		append_str(&out, &cap, &len, "\n");
	}
#endif
	if (!out) {
		return ok_string("");
	}
	if (len > 0 && out[len - 1] == '\n') {
		out[len - 1] = '\0';
	}
	return ok_string_owned(out);
}

Result_string_Error __std_temp_dir(void) {
#ifdef _WIN32
	char buf[MAX_PATH];
	DWORD n = GetTempPathA(MAX_PATH, buf);
	if (n == 0 || n > MAX_PATH) {
		return err_string("temp: failed");
	}
	return ok_string(buf);
#else
	const char *val = getenv("TMPDIR");
	if (!val || val[0] == '\0') {
		val = "/tmp";
	}
	return ok_string(val);
#endif
}

Result_string_Error __std_exe_path(void) {
#ifdef _WIN32
	char buf[MAX_PATH];
	DWORD n = GetModuleFileNameA(NULL, buf, MAX_PATH);
	if (n == 0 || n >= MAX_PATH) {
		return err_string("exe_path: failed");
	}
	return ok_string(buf);
#elif __APPLE__
	uint32_t size = 0;
	_NSGetExecutablePath(NULL, &size);
	if (size == 0) {
		return err_string("exe_path: failed");
	}
	char *buf = (char *)malloc(size + 1);
	if (!buf) {
		return err_string("exe_path: out of memory");
	}
	if (_NSGetExecutablePath(buf, &size) != 0) {
		free(buf);
		return err_string("exe_path: failed");
	}
	buf[size] = '\0';
	return ok_string_owned(buf);
#else
	char buf[4096];
	ssize_t n = readlink("/proc/self/exe", buf, sizeof(buf) - 1);
	if (n <= 0) {
		if (bazic_argv && bazic_argv[0]) {
			return ok_string(bazic_argv[0]);
		}
		return err_string("exe_path: failed");
	}
	buf[n] = '\0';
	return ok_string(buf);
#endif
}

Result_string_Error __std_home_dir(void) {
#ifdef _WIN32
	const char *val = getenv("USERPROFILE");
#else
	const char *val = getenv("HOME");
#endif
	if (!val || val[0] == '\0') {
		return err_string("home: not set");
	}
	return ok_string(val);
}

Result_string_Error __std_web_get_json(char *key) {
	(void)key;
	return err_string("web_get_json: wasm only");
}

Result_bool_Error __std_web_set_json(char *key, char *jsonText) {
	(void)key;
	(void)jsonText;
	return err_bool("web_set_json: wasm only");
}

static const char bazic_b64_table[] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
static char *bazic_base64url_encode_bytes(const uint8_t *data, size_t len) {
	if (!data) {
		return bazic_strdup("");
	}
	if (len == 0) {
		return bazic_strdup("");
	}
	size_t outLen = ((len + 2) / 3) * 4;
	char *out = (char *)malloc(outLen + 1);
	if (!out) {
		return NULL;
	}
	size_t i = 0;
	size_t j = 0;
	while (i < len) {
		uint32_t octet_a = i < len ? data[i++] : 0;
		uint32_t octet_b = i < len ? data[i++] : 0;
		uint32_t octet_c = i < len ? data[i++] : 0;
		uint32_t triple = (octet_a << 16) | (octet_b << 8) | octet_c;
		out[j++] = bazic_b64_table[(triple >> 18) & 0x3F];
		out[j++] = bazic_b64_table[(triple >> 12) & 0x3F];
		out[j++] = bazic_b64_table[(triple >> 6) & 0x3F];
		out[j++] = bazic_b64_table[triple & 0x3F];
	}
	size_t mod = len % 3;
	if (mod == 1) {
		out[outLen - 1] = '=';
		out[outLen - 2] = '=';
	} else if (mod == 2) {
		out[outLen - 1] = '=';
	}
	while (outLen > 0 && out[outLen - 1] == '=') {
		outLen--;
	}
	for (size_t k = 0; k < outLen; k++) {
		if (out[k] == '+') out[k] = '-';
		else if (out[k] == '/') out[k] = '_';
	}
	out[outLen] = '\0';
	return out;
}

char *__std_base64_encode(char *s) {
	if (!s) {
		return bazic_strdup("");
	}
	size_t len = strlen(s);
	if (len == 0) {
		return bazic_strdup("");
	}
	size_t outLen = ((len + 2) / 3) * 4;
	char *out = (char *)malloc(outLen + 1);
	if (!out) {
		return bazic_strdup("");
	}
	size_t i = 0, o = 0;
	while (i < len) {
		size_t remaining = len - i;
		uint32_t a = (unsigned char)s[i++];
		uint32_t b = remaining > 1 ? (unsigned char)s[i++] : 0;
		uint32_t c = remaining > 2 ? (unsigned char)s[i++] : 0;
		uint32_t triple = (a << 16) | (b << 8) | c;
		out[o++] = bazic_b64_table[(triple >> 18) & 0x3F];
		out[o++] = bazic_b64_table[(triple >> 12) & 0x3F];
		out[o++] = (remaining > 1) ? bazic_b64_table[(triple >> 6) & 0x3F] : '=';
		out[o++] = (remaining > 2) ? bazic_b64_table[triple & 0x3F] : '=';
	}
	out[outLen] = '\0';
	return out;
}

static int b64_index(char c) {
	if (c >= 'A' && c <= 'Z') return c - 'A';
	if (c >= 'a' && c <= 'z') return c - 'a' + 26;
	if (c >= '0' && c <= '9') return c - '0' + 52;
	if (c == '+') return 62;
	if (c == '/') return 63;
	return -1;
}

Result_string_Error __std_base64_decode(char *s) {
	if (!s) {
		return ok_string("");
	}
	size_t len = strlen(s);
	if (len == 0) {
		return ok_string("");
	}
	if (len % 4 != 0) {
		return err_string("base64: invalid length");
	}
	size_t pad = 0;
	if (len >= 2) {
		if (s[len - 1] == '=') pad++;
		if (s[len - 2] == '=') pad++;
	}
	size_t outLen = (len / 4) * 3 - pad;
	unsigned char *out = (unsigned char *)malloc(outLen + 1);
	if (!out) {
		return err_string("base64: out of memory");
	}
	size_t i = 0, o = 0;
	while (i < len) {
		int v0 = b64_index(s[i++]);
		int v1 = b64_index(s[i++]);
		int v2 = s[i] == '=' ? -1 : b64_index(s[i]);
		i++;
		int v3 = s[i] == '=' ? -1 : b64_index(s[i]);
		i++;
		if (v0 < 0 || v1 < 0 || v2 < -1 || v3 < -1) {
			free(out);
			return err_string("base64: invalid character");
		}
		uint32_t triple = (uint32_t)(v0 << 18) | (uint32_t)(v1 << 12) | (uint32_t)((v2 < 0 ? 0 : v2) << 6) | (uint32_t)(v3 < 0 ? 0 : v3);
		if (o < outLen) out[o++] = (triple >> 16) & 0xFF;
		if (o < outLen && v2 >= 0) out[o++] = (triple >> 8) & 0xFF;
		if (o < outLen && v3 >= 0) out[o++] = triple & 0xFF;
	}
	out[outLen] = '\0';
	return ok_string_owned((char *)out);
}

static char bazic_path_sep(void) {
#ifdef _WIN32
	return '\\';
#else
	return '/';
#endif
}

char *__std_path_basename(char *path) {
	if (!path) {
		return bazic_strdup("");
	}
	size_t len = strlen(path);
	if (len == 0) {
		return bazic_strdup("");
	}
	char sep = bazic_path_sep();
	while (len > 0 && (path[len - 1] == sep || path[len - 1] == '/' || path[len - 1] == '\\')) {
		len--;
	}
	if (len == 0) {
		return bazic_strdup("");
	}
	size_t i = len;
	while (i > 0) {
		char c = path[i - 1];
		if (c == sep || c == '/' || c == '\\') {
			break;
		}
		i--;
	}
	return bazic_strdup(path + i);
}

char *__std_path_dirname(char *path) {
	if (!path) {
		return bazic_strdup(".");
	}
	size_t len = strlen(path);
	if (len == 0) {
		return bazic_strdup(".");
	}
	char sep = bazic_path_sep();
	while (len > 0 && (path[len - 1] == sep || path[len - 1] == '/' || path[len - 1] == '\\')) {
		len--;
	}
	if (len == 0) {
		return bazic_strdup(".");
	}
	size_t i = len;
	while (i > 0) {
		char c = path[i - 1];
		if (c == sep || c == '/' || c == '\\') {
			break;
		}
		i--;
	}
	if (i == 0) {
		return bazic_strdup(".");
	}
	size_t outLen = i - 1;
	while (outLen > 0) {
		char c = path[outLen];
		if (c != sep && c != '/' && c != '\\') {
			break;
		}
		outLen--;
	}
	if (outLen == 0) {
		outLen = 1;
	}
	char *out = (char *)malloc(outLen + 1);
	if (!out) {
		return bazic_strdup(".");
	}
	memcpy(out, path, outLen);
	out[outLen] = '\0';
	return out;
}

char *__std_path_join(char *a, char *b) {
	if (!a || a[0] == '\0') {
		return bazic_strdup(b ? b : "");
	}
	if (!b || b[0] == '\0') {
		return bazic_strdup(a);
	}
	char sep = bazic_path_sep();
	size_t alen = strlen(a);
	size_t blen = strlen(b);
	bool aHasSep = a[alen - 1] == '/' || a[alen - 1] == '\\';
	bool bHasSep = b[0] == '/' || b[0] == '\\';
	size_t extra = (aHasSep || bHasSep) ? 0 : 1;
	char *out = (char *)malloc(alen + blen + extra + 1);
	if (!out) {
		return bazic_strdup("");
	}
	memcpy(out, a, alen);
	size_t off = alen;
	if (extra == 1) {
		out[off++] = sep;
	}
	memcpy(out + off, b, blen);
	out[off + blen] = '\0';
	return out;
}

Result_bool_Error __std_open_url(char *url) {
	if (!url) {
		return err_bool("open_url: null url");
	}
#ifdef _WIN32
	char cmd[1024];
	snprintf(cmd, sizeof(cmd), "cmd /c start \"\" \"%s\"", url);
	int rc = system(cmd);
	if (rc != 0) {
		return err_bool("open_url: failed");
	}
	return ok_bool(true);
#else
#if defined(__APPLE__)
	const char *opener = "open";
#else
	const char *opener = "xdg-open";
#endif
	pid_t pid = fork();
	if (pid < 0) {
		return err_bool("open_url: fork failed");
	}
	if (pid == 0) {
		execlp(opener, opener, url, (char *)NULL);
		_exit(127);
	}
	int status = 0;
	if (waitpid(pid, &status, 0) < 0) {
		return err_bool("open_url: wait failed");
	}
	if (WIFEXITED(status) && WEXITSTATUS(status) == 0) {
		return ok_bool(true);
	}
	return err_bool("open_url: failed");
#endif
}
