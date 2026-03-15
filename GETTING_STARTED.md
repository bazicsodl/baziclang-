# Getting Started (Bazic v0.3)

This is the shortest path from zero to running code.

## 1) Install

Windows (PowerShell):
```powershell
.\tools\install.ps1
.\bin\bazic.exe doctor
```
Note: If you downloaded an unsigned build, Windows SmartScreen will warn. This is expected until the release is code-signed. Verify `SHA256SUMS.txt` before running the installer.

Linux/macOS:
```bash
./tools/install.sh
./bin/bazic doctor
```

If `bazic doctor` reports missing `clang`, install LLVM 15+ and ensure `clang` is on `PATH`.

## 2) Create a project

```powershell
.\bin\bazic.exe new hello
cd hello
```

This creates a `main.bz` and configures `.gitignore` and stdlib wiring.
It also creates a starter `main_test.bz`.

## 3) Run

```powershell
.\bin\bazic.exe run
```

## 4) Build a native binary

```powershell
.\bin\bazic.exe build
```

Output goes to `.\bin\hello.exe` by default.

## 5) Add stdlib usage

```bazic
import "std";

fn main(): void {
    const now = time_now_rfc3339();
    println(now);
}
```

## 6) Tests and lint

```powershell
.\bin\bazic.exe test .
.\bin\bazic.exe lint .
```

## 7) LLVM backend (optional)

```powershell
.\bin\bazic.exe build --backend llvm -o .\bin\hello-llvm.exe
.\bin\bazic.exe emit-llvm --check .\main.bz
```

## 8) Bazic UI (optional)

```powershell
.\bin\bazic.exe ui init --dir .\my-ui
.\bin\bazic.exe ui dev --dir .\my-ui --port 8080
```

See `BAZIC_UI_QUICKSTART.md` for the minimal UI workflow.

## Common fixes
- If `clang` is missing: install LLVM 15+ and add to `PATH`.
- If Windows headers are missing: install Visual Studio Build Tools (C++ workload).
- If stdlib isn't found: ensure `std/` sits next to `bazic.exe` or set `BAZIC_STDLIB`.
