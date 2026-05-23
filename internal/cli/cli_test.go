package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"baziclang/internal/releasecontract"
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
		"release track: " + releasecontract.ReleaseTrackAlpha,
		"go backend: " + releasecontract.GoBackendReleaseStatus,
		"llvm backend: " + releasecontract.LLVMBackendReleaseStatus,
		"stdlib core: " + releasecontract.JoinModules(releasecontract.AlphaStableStdModules()),
		"stdlib experimental: " + releasecontract.JoinModules(releasecontract.AlphaExperimentalStdModules()),
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected doctor output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestDoctorCmdJSONUsesSharedContractSurface(t *testing.T) {
	out := captureStdout(t, func() {
		if rc := doctorCmd("bazic", "go", []string{"--json"}); rc != 0 {
			t.Fatalf("expected doctorCmd json success, got %d", rc)
		}
	})
	var report doctorReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("unmarshal doctor json: %v\n%s", err, out)
	}
	if report.ReleaseTrack != releasecontract.ReleaseTrackAlpha {
		t.Fatalf("unexpected release track: %#v", report)
	}
	if got := strings.Join(report.StdlibCore, ", "); got != releasecontract.JoinModules(releasecontract.AlphaStableStdModules()) {
		t.Fatalf("unexpected stdlib core: %q", got)
	}
	if got := strings.Join(report.StdlibExperimental, ", "); got != releasecontract.JoinModules(releasecontract.AlphaExperimentalStdModules()) {
		t.Fatalf("unexpected stdlib experimental: %q", got)
	}
	if report.GoBackend != releasecontract.GoBackendReleaseStatus || report.LLVMBackend != releasecontract.LLVMBackendReleaseStatus {
		t.Fatalf("unexpected backend status report: %#v", report)
	}
}
