package compiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestStdParityGoVsLLVM_JSON(t *testing.T) {
	root, err := projectRoot()
	if err != nil {
		t.Fatalf("project root: %v", err)
	}
	src := `import "std";

fn main(): void {
    let js = "{\"a\":1,\"b\":true,\"c\":\"x\"}";
    let a = json_get_int(js, "a");
    let b = json_get_bool(js, "b");
    let c = json_get_string(js, "c");
    if a.is_ok { print(str(a.value)); } else { print("e"); }
    print("|");
    if b.is_ok && b.value { print("t"); } else { print("f"); }
    print("|");
    if c.is_ok { print(c.value); } else { print("e"); }
}
`
	p := filepath.Join(root, "_tmp_parity_json.bz")
	if err := writeFile(p, src); err != nil {
		t.Fatalf("write: %v", err)
	}
	defer removeFile(p)

	outGo, err := runBazic(p, "go")
	if err != nil {
		t.Fatalf("go run failed: %v\n%s", err, outGo)
	}
	outLlvm, err := runBazic(p, "llvm")
	if err != nil {
		t.Fatalf("llvm run failed: %v\n%s", err, outLlvm)
	}
	if strings.TrimSpace(outGo) != strings.TrimSpace(outLlvm) {
		t.Fatalf("parity mismatch: go='%s' llvm='%s'", outGo, outLlvm)
	}
}

func TestStdParityGoVsLLVM_Crypto(t *testing.T) {
	root, err := projectRoot()
	if err != nil {
		t.Fatalf("project root: %v", err)
	}
	src := `import "std";

fn main(): void {
    let h = crypto_sha256_hex("hello");
    print(h);
}
`
	p := filepath.Join(root, "_tmp_parity_crypto.bz")
	if err := writeFile(p, src); err != nil {
		t.Fatalf("write: %v", err)
	}
	defer removeFile(p)

	outGo, err := runBazic(p, "go")
	if err != nil {
		t.Fatalf("go run failed: %v\n%s", err, outGo)
	}
	outLlvm, err := runBazic(p, "llvm")
	if err != nil {
		t.Fatalf("llvm run failed: %v\n%s", err, outLlvm)
	}
	if strings.TrimSpace(outGo) != strings.TrimSpace(outLlvm) {
		t.Fatalf("parity mismatch: go='%s' llvm='%s'", outGo, outLlvm)
	}
}

func TestStdParityGoVsLLVM_Base64(t *testing.T) {
	root, err := projectRoot()
	if err != nil {
		t.Fatalf("project root: %v", err)
	}
	src := `import "std";

fn main(): void {
    let s = "hello";
    let enc = base64_encode(s);
    let dec = base64_decode(enc);
    if dec.is_ok { print(dec.value); } else { print("e"); }
}
`
	p := filepath.Join(root, "_tmp_parity_b64.bz")
	if err := writeFile(p, src); err != nil {
		t.Fatalf("write: %v", err)
	}
	defer removeFile(p)

	outGo, err := runBazic(p, "go")
	if err != nil {
		t.Fatalf("go run failed: %v\n%s", err, outGo)
	}
	outLlvm, err := runBazic(p, "llvm")
	if err != nil {
		t.Fatalf("llvm run failed: %v\n%s", err, outLlvm)
	}
	if strings.TrimSpace(outGo) != strings.TrimSpace(outLlvm) {
		t.Fatalf("parity mismatch: go='%s' llvm='%s'", outGo, outLlvm)
	}
}

func TestStdParityGoVsLLVM_JWT(t *testing.T) {
	root, err := projectRoot()
	if err != nil {
		t.Fatalf("project root: %v", err)
	}
	src := `import "std";

fn main(): void {
    let header = "{\"alg\":\"HS256\",\"typ\":\"JWT\"}";
    let payload = "{\"sub\":\"u1\"}";
    let tok = jwt_sign_hs256(header, payload, "s");
    if !tok.is_ok { print("e"); return; }
    let v = jwt_verify_hs256(tok.value, "s");
    if v.is_ok && v.value { print("ok"); } else { print("bad"); }
}
`
	p := filepath.Join(root, "_tmp_parity_jwt.bz")
	if err := writeFile(p, src); err != nil {
		t.Fatalf("write: %v", err)
	}
	defer removeFile(p)

	outGo, err := runBazic(p, "go")
	if err != nil {
		t.Fatalf("go run failed: %v\n%s", err, outGo)
	}
	outLlvm, err := runBazic(p, "llvm")
	if err != nil {
		t.Fatalf("llvm run failed: %v\n%s", err, outLlvm)
	}
	if strings.TrimSpace(outGo) != strings.TrimSpace(outLlvm) {
		t.Fatalf("parity mismatch: go='%s' llvm='%s'", outGo, outLlvm)
	}
}

func TestStdParityGoVsLLVM_Session(t *testing.T) {
	root, err := projectRoot()
	if err != nil {
		t.Fatalf("project root: %v", err)
	}
	src := `import "std";

fn main(): void {
    let ok = session_init("memory");
    if !ok.is_ok { print("e"); return; }
    let put = session_put("memory", "t", "u", "");
    if !put.is_ok { print("e"); return; }
    let user = session_get_user("memory", "t");
    if user.is_ok { print(user.value); } else { print("e"); return; }
    let del = session_delete("memory", "t");
    if del.is_ok { print("|d"); } else { print("|e"); }
}
`
	p := filepath.Join(root, "_tmp_parity_session.bz")
	if err := writeFile(p, src); err != nil {
		t.Fatalf("write: %v", err)
	}
	defer removeFile(p)

	outGo, err := runBazic(p, "go")
	if err != nil {
		t.Fatalf("go run failed: %v\n%s", err, outGo)
	}
	outLlvm, err := runBazic(p, "llvm")
	if err != nil {
		t.Fatalf("llvm run failed: %v\n%s", err, outLlvm)
	}
	if strings.TrimSpace(outGo) != strings.TrimSpace(outLlvm) {
		t.Fatalf("parity mismatch: go='%s' llvm='%s'", outGo, outLlvm)
	}
}

func TestStdParityGoVsLLVM_HTTPHelpers(t *testing.T) {
	root, err := projectRoot()
	if err != nil {
		t.Fatalf("project root: %v", err)
	}
	src := `import "std";

fn main(): void {
    let headers = "X: a\nY: b";
    let cookies = "a=1\nb=2";
    let params = "id=42\nslug=post";
    let query = "q=hi&x=7";
    let h = http_header_get(headers, "x");
    let c = http_cookie_get(cookies, "b");
    let p = http_params_get(params, "slug");
    let q = http_query_get(query, "x");
    print(h + "|" + c + "|" + p + "|" + q);
}
`
	p := filepath.Join(root, "_tmp_parity_httphelpers.bz")
	if err := writeFile(p, src); err != nil {
		t.Fatalf("write: %v", err)
	}
	defer removeFile(p)

	outGo, err := runBazic(p, "go")
	if err != nil {
		t.Fatalf("go run failed: %v\n%s", err, outGo)
	}
	outLlvm, err := runBazic(p, "llvm")
	if err != nil {
		t.Fatalf("llvm run failed: %v\n%s", err, outLlvm)
	}
	if strings.TrimSpace(outGo) != strings.TrimSpace(outLlvm) {
		t.Fatalf("parity mismatch: go='%s' llvm='%s'", outGo, outLlvm)
	}
}

func TestStdParityGoVsLLVM_DB(t *testing.T) {
	if os.Getenv("BAZIC_SQLITE_TEST") != "1" {
		t.Skip("set BAZIC_SQLITE_TEST=1 to run sqlite parity test")
	}
	root, err := projectRoot()
	if err != nil {
		t.Fatalf("project root: %v", err)
	}
	src := `import "std";

fn main(): void {
    let path = "parity.db";
    let _ = db_exec(path, "drop table if exists users;");
    let ok = db_exec(path, "create table users(id int, name text);");
    if !ok.is_ok { print("e"); return; }
    let _ = db_exec(path, "insert into users values (1, 'Ada');");
    let res = db_query(path, "select id, name from users order by id;");
    if res.is_ok { print(res.value); } else { print("e"); }
}
`
	p := filepath.Join(root, "_tmp_parity_db.bz")
	if err := writeFile(p, src); err != nil {
		t.Fatalf("write: %v", err)
	}
	defer removeFile(p)
	dbFile := filepath.Join(root, "parity.db")
	_ = os.Remove(dbFile)
	defer os.Remove(dbFile)

	outGo, err := runBazic(p, "go")
	if err != nil {
		t.Fatalf("go run failed: %v\n%s", err, outGo)
	}
	outLlvm, err := runBazic(p, "llvm")
	if err != nil {
		t.Fatalf("llvm run failed: %v\n%s", err, outLlvm)
	}
	if strings.TrimSpace(outGo) != strings.TrimSpace(outLlvm) {
		t.Fatalf("parity mismatch: go='%s' llvm='%s'", outGo, outLlvm)
	}
}

func writeFile(path string, body string) error {
	return os.WriteFile(path, []byte(body), 0644)
}

func removeFile(path string) {
	_ = os.Remove(path)
}

func projectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root := wd
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			return root, nil
		}
		next := filepath.Dir(root)
		if next == root {
			break
		}
		root = next
	}
	return "", os.ErrNotExist
}

func runBazic(file string, backend string) (string, error) {
	root, err := projectRoot()
	if err != nil {
		return "", err
	}
	args := []string{"run", file, "--backend", backend}
	cmd := exec.Command("go", append([]string{"run", ".\\cmd\\bazc\\"}, args...)...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}
