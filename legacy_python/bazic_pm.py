#!/usr/bin/env python3
"""
bazic_pm.py - simple package manager prototype for BAZIC-Lang

Usage examples:
  python bazic_pm.py init mypkg
  python bazic_pm.py pack .            # create mypkg-1.0.0.bazpkg
  python bazic_pm.py publish ./dist/*.bazpkg --registry ./local_registry
  python bazic_pm.py install maths@1.0.0 --registry ./local_registry
  python bazic_pm.py serve-registry ./local_registry 8080
"""

import argparse
import json
import os
import tarfile
import tempfile
import shutil
from pathlib import Path
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer

# ---------- Defaults ----------
DEFAULT_REGISTRY = "./local_registry"
MODULES_DIRNAME = "bazic_modules"
LOCKFILE = ".bazic_lock.json"

# ---------- Helpers ----------
def load_manifest(path):
    with open(path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    # validation (minimal)
    if 'name' not in data or 'version' not in data:
        raise ValueError("bazic.json must have 'name' and 'version'")
    data.setdefault('main', 'index.baz')
    data.setdefault('dependencies', {})
    return data

def pack_package(src_dir, out_dir=None):
    """
    Create <name>-<version>.bazpkg from src_dir containing bazic.json.
    """
    src = Path(src_dir)
    manifest_path = src / "bazic.json"
    if not manifest_path.exists():
        raise FileNotFoundError("No bazic.json found in project root")
    manifest = load_manifest(manifest_path)
    name = manifest['name']
    version = manifest['version']
    pkg_name = f"{name}-{version}.bazpkg"
    out_dir = Path(out_dir or ".")
    out_path = out_dir / pkg_name
    # Create tar.gz
    with tarfile.open(out_path, "w:gz") as tar:
        for root, dirs, files in os.walk(src_dir):
            for fn in files:
                # include all project files except node_modules-like directory and .pyc etc
                if fn.endswith('.pyc'): continue
                full = Path(root) / fn
                arcname = str(full.relative_to(src_dir))
                tar.add(full, arcname=arcname)
    print(f"Packed {pkg_name} -> {out_path}")
    return out_path

def publish_package(pkg_path, registry_dir=DEFAULT_REGISTRY):
    pkg_path = Path(pkg_path)
    if not pkg_path.exists():
        raise FileNotFoundError(pkg_path)
    registry_dir = Path(registry_dir)
    registry_dir.mkdir(parents=True, exist_ok=True)
    dest = registry_dir / pkg_path.name
    shutil.copy2(pkg_path, dest)
    print(f"Published {pkg_path.name} to registry {registry_dir}")
    return dest

def list_registry(registry_dir=DEFAULT_REGISTRY):
    registry_dir = Path(registry_dir)
    if not registry_dir.exists():
        print("Registry empty (no dir)")
        return []
    pkgs = sorted([p.name for p in registry_dir.glob("*.bazpkg")])
    for p in pkgs:
        print(p)
    return pkgs

# Extract a package to destination dir, preserve internal structure
def extract_pkg(pkg_path, dest_dir):
    pkg_path = Path(pkg_path)
    dest_dir = Path(dest_dir)
    dest_dir.mkdir(parents=True, exist_ok=True)
    with tarfile.open(pkg_path, "r:gz") as tar:
        tar.extractall(path=dest_dir)
    return dest_dir

# Find a package file in registry by name@version (exact)
def find_pkg_in_registry(name, version, registry_dir=DEFAULT_REGISTRY):
    registry_dir = Path(registry_dir)
    fname = f"{name}-{version}.bazpkg"
    candidate = registry_dir / fname
    if candidate.exists():
        return candidate
    raise FileNotFoundError(f"Package {name}@{version} not found in registry {registry_dir}")

# Install package into project (project_dir)
def install_pkg(name, version, project_dir=".", registry_dir=DEFAULT_REGISTRY):
    project_dir = Path(project_dir)
    registry_dir = Path(registry_dir)
    modules_dir = project_dir / MODULES_DIRNAME
    modules_dir.mkdir(parents=True, exist_ok=True)
    pkg_file = find_pkg_in_registry(name, version, registry_dir)
    dest_dir = modules_dir / f"{name}@{version}"
    if dest_dir.exists():
        print(f"{name}@{version} already installed at {dest_dir}")
        return dest_dir
    # extract to temporary then move
    with tempfile.TemporaryDirectory() as td:
        extract_pkg(pkg_file, td)
        shutil.copytree(td, dest_dir)
    # update lockfile
    lockfile = project_dir / LOCKFILE
    lock = {}
    if lockfile.exists():
        with open(lockfile, 'r', encoding='utf-8') as f:
            lock = json.load(f)
    lock[name] = version
    with open(lockfile, 'w', encoding='utf-8') as f:
        json.dump(lock, f, indent=2)
    print(f"Installed {name}@{version} -> {dest_dir}")
    return dest_dir

def uninstall_pkg(name, project_dir="."):
    project_dir = Path(project_dir)
    modules_dir = project_dir / MODULES_DIRNAME
    lockfile = project_dir / LOCKFILE
    lock = {}
    if lockfile.exists():
        with open(lockfile, 'r', encoding='utf-8') as f:
            lock = json.load(f)
    if name not in lock:
        print(f"{name} not installed (no entry in lockfile)")
        return False
    version = lock[name]
    pkg_dir = modules_dir / f"{name}@{version}"
    if pkg_dir.exists():
        shutil.rmtree(pkg_dir)
    del lock[name]
    with open(lockfile, 'w', encoding='utf-8') as f:
        json.dump(lock, f, indent=2)
    print(f"Uninstalled {name}@{version}")
    return True

# Resolve dependencies recursively from a package dir or manifest
def resolve_dependencies_from_manifest(manifest, registry_dir=DEFAULT_REGISTRY, seen=None):
    seen = seen or {}
    deps = manifest.get('dependencies', {})
    results = {}
    for name, ver in deps.items():
        if name in seen:
            # version conflict naive handling: require same version exact
            if seen[name] != ver:
                raise RuntimeError(f"Version conflict for {name}: {seen[name]} vs {ver}")
            continue
        seen[name] = ver
        # ensure package exists in registry
        pkg = find_pkg_in_registry(name, ver, registry_dir)
        # read its manifest from the tarball without extracting
        with tarfile.open(pkg, "r:gz") as tar:
            try:
                manifest_member = tar.getmember("bazic.json")
                f = tar.extractfile(manifest_member)
                submanifest = json.load(f)
            except KeyError:
                raise RuntimeError(f"Package {name}@{ver} missing bazic.json")
        results[name] = ver
        subdeps = resolve_dependencies_from_manifest(submanifest, registry_dir, seen)
        results.update(subdeps)
    return results

# Serve registry as static HTTP directory
def serve_registry(path, port=8000):
    path = os.path.abspath(path)
    os.makedirs(path, exist_ok=True)
    os.chdir(path)
    handler = SimpleHTTPRequestHandler
    httpd = ThreadingHTTPServer(("0.0.0.0", port), handler)
    print(f"Serving registry {path} on http://0.0.0.0:{port}")
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("Stopping registry")
        httpd.server_close()

# ---------- CLI ----------
def main():
    ap = argparse.ArgumentParser(prog="bazic_pm")
    sub = ap.add_subparsers(dest='cmd')

    p_init = sub.add_parser('init', help='create sample bazic.json in current dir')
    p_init.add_argument('name', nargs='?', default=None)

    p_pack = sub.add_parser('pack', help='pack project into .bazpkg')
    p_pack.add_argument('project', nargs='?', default='.')

    p_publish = sub.add_parser('publish', help='publish .bazpkg to local registry dir')
    p_publish.add_argument('pkg', help='path to .bazpkg')
    p_publish.add_argument('--registry', default=DEFAULT_REGISTRY)

    p_list = sub.add_parser('list', help='list registry packages')
    p_list.add_argument('--registry', default=DEFAULT_REGISTRY)

    p_install = sub.add_parser('install', help='install package from registry (name@version)')
    p_install.add_argument('spec', help='name@version')
    p_install.add_argument('--project', default='.')
    p_install.add_argument('--registry', default=DEFAULT_REGISTRY)

    p_uninstall = sub.add_parser('uninstall', help='uninstall a package by name from project')
    p_uninstall.add_argument('name')
    p_uninstall.add_argument('--project', default='.')

    p_serve = sub.add_parser('serve-registry', help='serve a registry directory over HTTP')
    p_serve.add_argument('dir', nargs='?', default=DEFAULT_REGISTRY)
    p_serve.add_argument('port', nargs='?', default=8000, type=int)

    p_resolve = sub.add_parser('resolve', help='resolve dependencies of a local project using registry')
    p_resolve.add_argument('project', nargs='?', default='.')
    p_resolve.add_argument('--registry', default=DEFAULT_REGISTRY)

    args = ap.parse_args()
    if args.cmd == 'init':
        name = args.name or 'mypkg'
        manifest = {
            "name": name,
            "version": "1.0.0",
            "main": "index.baz",
            "dependencies": {}
        }
        with open("bazic.json", "w", encoding='utf-8') as f:
            json.dump(manifest, f, indent=2)
        # create a trivial index.baz
        with open("index.baz", "w", encoding='utf-8') as f:
            f.write('// sample package\nprint("Hello from ' + name + '")\n')
        print("Created bazic.json and index.baz")
        return

    if args.cmd == 'pack':
        pack_package(args.project, out_dir='.')
        return

    if args.cmd == 'publish':
        publish_package(args.pkg, registry_dir=args.registry)
        return

    if args.cmd == 'list':
        list_registry(registry_dir=args.registry)
        return

    if args.cmd == 'install':
        spec = args.spec
        if '@' not in spec:
            print("Please specify exact version: name@version")
            return
        name, ver = spec.split('@', 1)
        # resolve remote deps and install them first
        pkg_file = find_pkg_in_registry(name, ver, args.registry)
        # read manifest from tarball
        with tarfile.open(pkg_file, "r:gz") as tar:
            try:
                m = json.load(tar.extractfile('bazic.json'))
            except KeyError:
                raise RuntimeError("Package missing bazic.json")
        deps = resolve_dependencies_from_manifest(m, args.registry)
        # install dependencies first
        for dep_name, dep_ver in deps.items():
            install_pkg(dep_name, dep_ver, project_dir=args.project, registry_dir=args.registry)
        # install requested package
        install_pkg(name, ver, project_dir=args.project, registry_dir=args.registry)
        return

    if args.cmd == 'uninstall':
        uninstall_pkg(args.name, project_dir=args.project)
        return

    if args.cmd == 'serve-registry':
        serve_registry(args.dir, args.port)
        return

    if args.cmd == 'resolve':
        manifest_path = Path(args.project) / "bazic.json"
        if not manifest_path.exists():
            print("No bazic.json in project")
            return
        m = load_manifest(manifest_path)
        deps = resolve_dependencies_from_manifest(m, registry_dir=args.registry)
        print("Resolved dependency tree (flat):")
        for n,v in deps.items():
            print(f"  {n}@{v}")
        return

    ap.print_help()

if __name__ == '__main__':
    main()
