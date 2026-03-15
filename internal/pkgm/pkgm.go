package pkgm

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Manifest struct {
	Name string            `json:"name"`
	Deps map[string]string `json:"deps"`
}

type Lockfile struct {
	Name               string               `json:"name"`
	Version            int                  `json:"version"`
	SigningPublicKey   string               `json:"signing_public_key,omitempty"`
	Deps               map[string]LockedDep `json:"deps"`
	GeneratedAtRFC3339 string               `json:"generated_at,omitempty"`
}

type LockedDep struct {
	Provenance Provenance `json:"provenance"`
	Integrity  Integrity  `json:"integrity"`
	Signature  string     `json:"signature,omitempty"`

	SourcePath string `json:"source_path,omitempty"`
	SourceKind string `json:"source_kind,omitempty"`
	Checksum   string `json:"checksum,omitempty"`
}

type Provenance struct {
	SourcePath string `json:"source_path"`
	SourceKind string `json:"source_kind"`
}

type Integrity struct {
	Algorithm string `json:"algorithm"`
	Checksum  string `json:"checksum"`
}

const ManifestFile = "bazic.mod.json"
const LockfileFile = "bazic.lock.json"

func FindProjectRoot(start string) (string, error) {
	dir := start
	for {
		candidate := filepath.Join(dir, ManifestFile)
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			return "", fmt.Errorf("could not find %s", ManifestFile)
		}
		dir = next
	}
}

func LoadManifest(root string) (*Manifest, error) {
	path := filepath.Join(root, ManifestFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := &Manifest{}
	if err := json.Unmarshal(data, m); err != nil {
		return nil, err
	}
	if m.Deps == nil {
		m.Deps = map[string]string{}
	}
	return m, nil
}

func SaveManifest(root string, m *Manifest) error {
	if m.Deps == nil {
		m.Deps = map[string]string{}
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(root, ManifestFile), data, 0644)
}

func Init(root, name string) error {
	path := filepath.Join(root, ManifestFile)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", ManifestFile)
	}
	if err := SaveManifest(root, &Manifest{Name: name, Deps: map[string]string{}}); err != nil {
		return err
	}
	return SaveLockfile(root, &Lockfile{Name: name, Version: 2, Deps: map[string]LockedDep{}})
}

func AddDep(root, alias, depPath string) error {
	if err := validateAlias(alias); err != nil {
		return err
	}
	m, err := LoadManifest(root)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(depPath) {
		depPath = filepath.Clean(filepath.Join(root, depPath))
	}
	m.Deps[alias] = depPath
	return SaveManifest(root, m)
}

func DetectStdlibPath() (string, bool) {
	if v := strings.TrimSpace(os.Getenv("BAZIC_STDLIB")); v != "" {
		if info, err := os.Stat(v); err == nil && info.IsDir() {
			return v, true
		}
	}
	if v := strings.TrimSpace(os.Getenv("BAZIC_HOME")); v != "" {
		candidate := filepath.Join(v, "std")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, true
		}
	}
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	exeDir := filepath.Dir(exe)
	candidates := []string{
		filepath.Join(exeDir, "std"),
		filepath.Join(exeDir, "..", "std"),
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c, true
		}
	}
	return "", false
}

func Sync(root string) error {
	m, err := LoadManifest(root)
	if err != nil {
		return err
	}
	pkgRoot := filepath.Join(root, ".bazic", "pkg")
	if err := os.MkdirAll(pkgRoot, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(pkgRoot)
	if err != nil {
		return err
	}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		if _, ok := m.Deps[ent.Name()]; !ok {
			if err := os.RemoveAll(filepath.Join(pkgRoot, ent.Name())); err != nil {
				return err
			}
		}
	}
	pub, priv, err := ensureSigningKey(root)
	if err != nil {
		return err
	}
	lock := &Lockfile{
		Name:               m.Name,
		Version:            2,
		SigningPublicKey:   base64.StdEncoding.EncodeToString(pub),
		GeneratedAtRFC3339: time.Now().UTC().Format(time.RFC3339),
		Deps:               map[string]LockedDep{},
	}
	for alias, src := range m.Deps {
		if err := validateAlias(alias); err != nil {
			return err
		}
		src, err = filepath.Abs(src)
		if err != nil {
			return err
		}
		dst := filepath.Join(pkgRoot, alias)
		if err := os.RemoveAll(dst); err != nil {
			return err
		}
		if err := copyDir(src, dst); err != nil {
			return fmt.Errorf("sync %s: %w", alias, err)
		}
		sum, err := checksumDir(dst)
		if err != nil {
			return fmt.Errorf("checksum %s: %w", alias, err)
		}
		dep := LockedDep{
			Provenance: Provenance{
				SourcePath: filepath.Clean(src),
				SourceKind: "local_path",
			},
			Integrity: Integrity{
				Algorithm: "sha256",
				Checksum:  sum,
			},
		}
		sig, err := signLockedDep(alias, dep, priv)
		if err != nil {
			return fmt.Errorf("sign %s: %w", alias, err)
		}
		dep.Signature = sig
		lock.Deps[alias] = dep
	}
	if err := SaveLockfile(root, lock); err != nil {
		return err
	}
	return nil
}

func ResolveImport(root, importerDir, importPath string) (string, error) {
	if filepath.IsAbs(importPath) {
		return "", fmt.Errorf("absolute imports are disallowed: '%s'", importPath)
	}
	if len(importPath) > 0 && importPath[0] == '.' {
		return resolveRelativeImport(root, importerDir, importPath)
	}
	return resolveAliasImport(root, importPath)
}

func Verify(root string) error {
	m, err := LoadManifest(root)
	if err != nil {
		return err
	}
	lock, err := LoadLockfile(root)
	if err != nil {
		return fmt.Errorf("missing or invalid %s: %w", LockfileFile, err)
	}
	if lock.Version < 2 {
		return fmt.Errorf("lockfile version %d is unsupported; run 'bazc pkg sync' to upgrade", lock.Version)
	}
	if lock.SigningPublicKey == "" {
		return fmt.Errorf("lockfile missing signing public key; run 'bazc pkg sync'")
	}
	pubKey, err := base64.StdEncoding.DecodeString(lock.SigningPublicKey)
	if err != nil {
		return fmt.Errorf("invalid signing public key in lockfile: %w", err)
	}
	if lock.Name != "" && m.Name != "" && lock.Name != m.Name {
		return fmt.Errorf("lockfile project name mismatch: manifest '%s' vs lock '%s'", m.Name, lock.Name)
	}
	for alias := range m.Deps {
		if _, ok := lock.Deps[alias]; !ok {
			return fmt.Errorf("lockfile missing dependency '%s'; run 'bazc pkg sync'", alias)
		}
	}
	for alias := range lock.Deps {
		if _, ok := m.Deps[alias]; !ok {
			return fmt.Errorf("lockfile has stale dependency '%s'; run 'bazc pkg sync'", alias)
		}
	}
	for alias, locked := range lock.Deps {
		if err := validateAlias(alias); err != nil {
			return err
		}
		locked = upgradeLockedDep(locked)
		pkgDir := filepath.Join(root, ".bazic", "pkg", alias)
		sum, err := checksumDir(pkgDir)
		if err != nil {
			return fmt.Errorf("verify '%s': %w", alias, err)
		}
		if sum != locked.Integrity.Checksum {
			return fmt.Errorf("checksum mismatch for package '%s': expected %s got %s; run 'bazc pkg sync'", alias, locked.Integrity.Checksum, sum)
		}
		src := m.Deps[alias]
		if !filepath.IsAbs(src) {
			src = filepath.Clean(filepath.Join(root, src))
		}
		src, err = filepath.Abs(src)
		if err != nil {
			return err
		}
		if locked.Provenance.SourcePath != "" && filepath.Clean(locked.Provenance.SourcePath) != filepath.Clean(src) {
			return fmt.Errorf("source path mismatch for package '%s': manifest '%s' vs lock '%s'; run 'bazc pkg sync'", alias, src, locked.Provenance.SourcePath)
		}
		if locked.Provenance.SourceKind == "local_path" {
			srcSum, err := checksumDir(src)
			if err != nil {
				return fmt.Errorf("verify '%s' source: %w", alias, err)
			}
			if srcSum != locked.Integrity.Checksum {
				return fmt.Errorf("source drift for package '%s': lock checksum %s but source checksum %s; run 'bazc pkg sync'", alias, locked.Integrity.Checksum, srcSum)
			}
		}
		if err := verifyLockedDepSignature(alias, locked, pubKey); err != nil {
			return fmt.Errorf("signature verification failed for package '%s': %w", alias, err)
		}
	}
	return nil
}

func normalizeImportTarget(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if filepath.Ext(path) != ".bz" {
			path = path + ".bz"
			if _, err2 := os.Stat(path); err2 == nil {
				return filepath.Clean(path), nil
			}
		}
		return "", err
	}
	if info.IsDir() {
		mainFile := filepath.Join(path, "main.bz")
		if _, err := os.Stat(mainFile); err != nil {
			return "", fmt.Errorf("package directory '%s' missing main.bz", path)
		}
		return filepath.Clean(mainFile), nil
	}
	return filepath.Clean(path), nil
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		if _, err := io.Copy(out, in); err != nil {
			return err
		}
		return nil
	})
}

func LoadLockfile(root string) (*Lockfile, error) {
	path := filepath.Join(root, LockfileFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	l := &Lockfile{}
	if err := json.Unmarshal(data, l); err != nil {
		return nil, err
	}
	if l.Deps == nil {
		l.Deps = map[string]LockedDep{}
	}
	if l.Version == 0 {
		l.Version = 1
	}
	if l.Version == 1 {
		for k, v := range l.Deps {
			l.Deps[k] = upgradeLockedDep(v)
		}
	}
	return l, nil
}

func SaveLockfile(root string, l *Lockfile) error {
	if l.Deps == nil {
		l.Deps = map[string]LockedDep{}
	}
	if l.Version == 0 {
		l.Version = 2
	}
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(root, LockfileFile), data, 0644)
}

func resolveAliasImport(root, alias string) (string, error) {
	if err := validateAlias(alias); err != nil {
		return "", err
	}
	lock, err := LoadLockfile(root)
	if err != nil {
		return "", fmt.Errorf("missing or invalid %s: run 'bazc pkg sync' (%w)", LockfileFile, err)
	}
	locked, ok := lock.Deps[alias]
	if !ok {
		return "", fmt.Errorf("unknown package alias '%s' in %s", alias, LockfileFile)
	}
	locked = upgradeLockedDep(locked)
	pkgDir := filepath.Join(root, ".bazic", "pkg", alias)
	sum, err := checksumDir(pkgDir)
	if err != nil {
		return "", fmt.Errorf("could not verify package '%s': %w", alias, err)
	}
	if sum != locked.Integrity.Checksum {
		return "", fmt.Errorf("checksum mismatch for package '%s': expected %s got %s; run 'bazc pkg sync'", alias, locked.Integrity.Checksum, sum)
	}
	return normalizeImportTarget(pkgDir)
}

func upgradeLockedDep(d LockedDep) LockedDep {
	if d.Provenance.SourcePath == "" && d.SourcePath != "" {
		d.Provenance.SourcePath = d.SourcePath
	}
	if d.Provenance.SourceKind == "" && d.SourceKind != "" {
		d.Provenance.SourceKind = d.SourceKind
	}
	if d.Integrity.Checksum == "" && d.Checksum != "" {
		d.Integrity = Integrity{Algorithm: "sha256", Checksum: d.Checksum}
	}
	return d
}

func ensureSigningKey(root string) ([]byte, []byte, error) {
	keyDir := filepath.Join(root, ".bazic", "keys")
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return nil, nil, err
	}
	privPath := filepath.Join(keyDir, "signing_key")
	pubPath := filepath.Join(keyDir, "signing_key.pub")
	if data, err := os.ReadFile(privPath); err == nil {
		priv, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
		if err != nil {
			return nil, nil, err
		}
		pubData, err := os.ReadFile(pubPath)
		if err != nil {
			return nil, nil, err
		}
		pub, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(pubData)))
		if err != nil {
			return nil, nil, err
		}
		return pub, priv, nil
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(privPath, []byte(base64.StdEncoding.EncodeToString(priv)), 0600); err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(pubPath, []byte(base64.StdEncoding.EncodeToString(pub)), 0644); err != nil {
		return nil, nil, err
	}
	return pub, priv, nil
}

func signaturePayload(alias string, dep LockedDep) []byte {
	parts := []string{
		"alias=" + alias,
		"source_path=" + dep.Provenance.SourcePath,
		"source_kind=" + dep.Provenance.SourceKind,
		"algorithm=" + dep.Integrity.Algorithm,
		"checksum=" + dep.Integrity.Checksum,
	}
	return []byte(strings.Join(parts, "\n"))
}

func signLockedDep(alias string, dep LockedDep, privKey []byte) (string, error) {
	if len(privKey) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("invalid private key size")
	}
	sig := ed25519.Sign(ed25519.PrivateKey(privKey), signaturePayload(alias, dep))
	return base64.StdEncoding.EncodeToString(sig), nil
}

func verifyLockedDepSignature(alias string, dep LockedDep, pubKey []byte) error {
	if len(pubKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size")
	}
	if dep.Signature == "" {
		return fmt.Errorf("missing signature")
	}
	sig, err := base64.StdEncoding.DecodeString(dep.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding")
	}
	if !ed25519.Verify(ed25519.PublicKey(pubKey), signaturePayload(alias, dep), sig) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

type SBOM struct {
	Name      string      `json:"name"`
	Version   int         `json:"version"`
	Generated string      `json:"generated_at"`
	Deps      []SBOMEntry `json:"deps"`
}

type SBOMEntry struct {
	Alias      string `json:"alias"`
	SourcePath string `json:"source_path"`
	SourceKind string `json:"source_kind"`
	Checksum   string `json:"checksum"`
	Algorithm  string `json:"algorithm"`
}

func WriteSBOM(root, outPath string) error {
	m, err := LoadManifest(root)
	if err != nil {
		return err
	}
	lock, err := LoadLockfile(root)
	if err != nil {
		return err
	}
	deps := make([]SBOMEntry, 0, len(lock.Deps))
	for alias, dep := range lock.Deps {
		dep = upgradeLockedDep(dep)
		deps = append(deps, SBOMEntry{
			Alias:      alias,
			SourcePath: dep.Provenance.SourcePath,
			SourceKind: dep.Provenance.SourceKind,
			Checksum:   dep.Integrity.Checksum,
			Algorithm:  dep.Integrity.Algorithm,
		})
	}
	sort.Slice(deps, func(i, j int) bool { return deps[i].Alias < deps[j].Alias })
	sbom := SBOM{
		Name:      m.Name,
		Version:   lock.Version,
		Generated: time.Now().UTC().Format(time.RFC3339),
		Deps:      deps,
	}
	data, err := json.MarshalIndent(sbom, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(outPath, data, 0644)
}

func resolveRelativeImport(root, importerDir, importPath string) (string, error) {
	target, err := normalizeImportTarget(filepath.Join(importerDir, importPath))
	if err != nil {
		return "", err
	}
	allowedRoot, err := allowedScopeRoot(root, importerDir)
	if err != nil {
		return "", err
	}
	ok, err := isWithin(target, allowedRoot)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("relative import escapes allowed scope: '%s' not within '%s'", target, allowedRoot)
	}
	return target, nil
}

func allowedScopeRoot(root, importerDir string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	importerAbs, err := filepath.Abs(importerDir)
	if err != nil {
		return "", err
	}
	pkgRoot := filepath.Join(rootAbs, ".bazic", "pkg")
	inPkgRoot, err := isWithin(importerAbs, pkgRoot)
	if err != nil {
		return "", err
	}
	if !inPkgRoot {
		return rootAbs, nil
	}
	rel, err := filepath.Rel(pkgRoot, importerAbs)
	if err != nil {
		return "", err
	}
	first := rel
	if i := strings.IndexRune(rel, filepath.Separator); i >= 0 {
		first = rel[:i]
	}
	if first == "." || first == "" {
		return rootAbs, nil
	}
	return filepath.Join(pkgRoot, first), nil
}

func isWithin(path, base string) (bool, error) {
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return false, err
	}
	rel, err := filepath.Rel(baseAbs, pathAbs)
	if err != nil {
		return false, err
	}
	if rel == "." {
		return true, nil
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..", nil
}

func validateAlias(alias string) error {
	if alias == "" {
		return fmt.Errorf("package alias cannot be empty")
	}
	if alias == "." || alias == ".." {
		return fmt.Errorf("invalid package alias '%s'", alias)
	}
	if strings.Contains(alias, "/") || strings.Contains(alias, "\\") {
		return fmt.Errorf("package alias '%s' must not contain path separators", alias)
	}
	for _, r := range alias {
		isAlpha := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
		isNum := r >= '0' && r <= '9'
		if isAlpha || isNum || r == '_' || r == '-' {
			continue
		}
		return fmt.Errorf("package alias '%s' contains invalid character '%c'", alias, r)
	}
	return nil
}

func checksumDir(root string) (string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("checksum target '%s' is not a directory", root)
	}
	files := make([]string, 0, 16)
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(d.Name()) != ".bz" && filepath.Ext(d.Name()) != ".json" {
			return nil
		}
		files = append(files, path)
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(files)
	h := sha256.New()
	for _, f := range files {
		rel, err := filepath.Rel(root, f)
		if err != nil {
			return "", err
		}
		_, _ = h.Write([]byte(filepath.ToSlash(rel)))
		_, _ = h.Write([]byte{0})
		data, err := os.ReadFile(f)
		if err != nil {
			return "", err
		}
		_, _ = h.Write(data)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
