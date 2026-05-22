package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	outC := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	fn()
	_ = w.Close()
	return <-outC
}

func TestDoctorCmdPrintsAlphaReleasePosture(t *testing.T) {
	out := captureStdout(t, func() {
		if rc := doctorCmd("bazic", "go", nil); rc != 0 {
			t.Fatalf("expected doctorCmd success, got %d", rc)
		}
	})
	for _, want := range []string{
		"release track: alpha",
		"go backend: stable release path",
		"llvm backend: experimental",
		"stdlib core: io, fs, time, json, http, crypto, base64, collections, os, path",
		"stdlib experimental: db, auth, jwt, session, desktop, web, ui, sql, validate",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected doctor output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestJoinModules(t *testing.T) {
	if got := joinModules([]string{"io", "fs", "time"}); got != "io, fs, time" {
		t.Fatalf("unexpected joined modules: %q", got)
	}
}
