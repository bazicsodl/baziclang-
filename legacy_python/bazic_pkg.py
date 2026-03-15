
# bazic_pkg.py - simple local pack/install scaffold
from pathlib import Path, TarPath
import json, zipfile, shutil
BASE = Path.cwd()

def pack_module(src_dir: str, out_file: str):
    src = Path(src_dir)
    if not src.exists(): raise FileNotFoundError(src_dir)
    with zipfile.ZipFile(out_file, "w", zipfile.ZIP_DEFLATED) as z:
        for p in src.rglob("*"):
            if p.is_file():
                z.write(p, arcname=str(p.relative_to(src)))
    print("Packed", out_file)

def install_module(zip_file: str, modules_dir="bazic_modules"):
    zf = Path(zip_file)
    modules = Path(modules_dir); modules.mkdir(exist_ok=True)
    # naive install: unzip into modules/<name>@<version> using name from manifest if present
    import zipfile, json
    with zipfile.ZipFile(zf, "r") as z:
        names = z.namelist()
        manifest_name = None
        if "bazic.json" in names: manifest_name = "bazic.json"
        if manifest_name:
            meta = json.loads(z.read(manifest_name).decode("utf-8"))
            name = meta.get("name","unnamed"); ver = meta.get("version","0.0.0")
            dest = modules / f"{name}@{ver}"; dest.mkdir(parents=True, exist_ok=True)
            z.extractall(path=dest)
            print("Installed to", dest)
        else:
            print("No bazic.json found in package; skipping install")
