package pkg

// Cache layer for downloaded NVIDIA toolkit packages.
//
// Each `airgap bundle create` writes <packageDir>/packages.lock.json after a
// successful download. The next run reads the lock file and skips the
// apt-get / dnf / zypper dance entirely when:
//   - the lock schema and toolkit version match the compiled-in constants
//   - every file referenced by the lock is still on disk and has the same
//     sha256
//   - the dependency chain described by each cached package is still
//     resolvable within the cached set (deps present, versions satisfy the
//     declared comparator).
//
// On any validation failure the lock file is removed so Download() can start
// fresh; the user does not need to manually clean the package directory.

import (
	osi "csghub-airgap/internal/infra/os"
	"csghub-airgap/internal/log"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	lockSchemaVersion = 1
	lockFileName      = "packages.lock.json"
)

// NoCache, when true, makes Download() skip the cache validation step and
// always re-pull from upstream. It is set from the --no-pkg-cache CLI flag.
var NoCache = false

type lockFile struct {
	SchemaVersion  int           `json:"schemaVersion"`
	ToolkitVersion string        `json:"toolkitVersion"`
	CapturedAt     string        `json:"capturedAt"`
	Host           lockHost      `json:"host"`
	SourceRepo     string        `json:"sourceRepo,omitempty"`
	Packages       []lockPackage `json:"packages"`
}

type lockHost struct {
	Distro string `json:"distro"`
	Arch   string `json:"arch"`
}

type lockPackage struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Sha256   string `json:"sha256"`
}

// depRef is a single dependency constraint, e.g. `libc6 (>= 2.31)` or `foo`.
// Op and Version are empty when the dep is unconstrained.
type depRef struct {
	Name    string
	Op      string // "=", ">=", "<=", ">>", "<<", ""
	Version string
}

// validateCache returns (true, "") if the cached packages are intact and
// the dependency chain resolves. Otherwise it returns (false, reason) where
// reason is a short string suitable for the user-visible log line. An empty
// lock file is reported as invalid with a clear reason so the caller can
// decide whether to redownload or just skip silently.
func validateCache(packageDir string) (bool, string) {
	lockPath := filepath.Join(packageDir, lockFileName)
	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "no lock file"
		}
		return false, fmt.Sprintf("cannot read lock file: %v", err)
	}

	var lock lockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		return false, fmt.Sprintf("lock file malformed: %v", err)
	}

	if lock.SchemaVersion != lockSchemaVersion {
		return false, fmt.Sprintf("schemaVersion=%d, want %d", lock.SchemaVersion, lockSchemaVersion)
	}
	if lock.ToolkitVersion != NvidiaContainerToolkitVersion {
		return false, fmt.Sprintf("toolkitVersion=%q, want %q", lock.ToolkitVersion, NvidiaContainerToolkitVersion)
	}

	hostDistro, err := osi.GetOSID()
	if err != nil {
		return false, fmt.Sprintf("cannot detect host distro: %v", err)
	}
	if lock.Host.Distro != hostDistro {
		return false, fmt.Sprintf("distro=%q, host=%q", lock.Host.Distro, hostDistro)
	}
	if lock.Host.Arch != runtime.GOARCH {
		return false, fmt.Sprintf("arch=%q, host=%q", lock.Host.Arch, runtime.GOARCH)
	}

	if len(lock.Packages) == 0 {
		return false, "lock lists zero packages"
	}

	// File-level integrity.
	for _, lp := range lock.Packages {
		full := filepath.Join(packageDir, lp.Filename)
		fi, err := os.Stat(full)
		if err != nil {
			return false, fmt.Sprintf("file missing: %s", lp.Filename)
		}
		if fi.Size() != lp.Size {
			return false, fmt.Sprintf("size mismatch: %s (have=%d want=%d)", lp.Filename, fi.Size(), lp.Size)
		}
		sum, err := fileSha256(full)
		if err != nil {
			return false, fmt.Sprintf("cannot hash %s: %v", lp.Filename, err)
		}
		if sum != lp.Sha256 {
			return false, fmt.Sprintf("sha256 mismatch: %s (have=%s want=%s)", lp.Filename, short(sum), short(lp.Sha256))
		}
	}

	// Dependency chain: rebuild the index from disk and verify every declared
	// dep resolves to another cached package at a compatible version.
	index, err := buildDepIndex(packageDir)
	if err != nil {
		return false, fmt.Sprintf("cannot build dep index: %v", err)
	}

	for filename, entry := range index {
		for gi, group := range entry.depends {
			satisfied := false
			var lastWhy string
			for _, dep := range group {
				ok, why := resolveDep(dep, index)
				if ok {
					satisfied = true
					break
				}
				lastWhy = why
			}
			if !satisfied {
				return false, fmt.Sprintf("dep unsatisfied in %s (group %d: %s: %s)",
					filename, gi, formatGroup(group), lastWhy)
			}
		}
	}

	return true, fmt.Sprintf("toolkit=%s distro=%s arch=%s files=%d",
		lock.ToolkitVersion, lock.Host.Distro, lock.Host.Arch, len(lock.Packages))
}

// writeLock scans packageDir for .deb/.rpm files, computes sha256 + size,
// and writes packages.lock.json. Called after a successful download.
func writeLock(packageDir, distroID, sourceRepo string) error {
	pkgs, err := scanPackageDir(packageDir)
	if err != nil {
		return fmt.Errorf("scan packages: %w", err)
	}

	lock := lockFile{
		SchemaVersion:  lockSchemaVersion,
		ToolkitVersion: NvidiaContainerToolkitVersion,
		CapturedAt:     nowISO(),
		Host: lockHost{
			Distro: distroID,
			Arch:   runtime.GOARCH,
		},
		SourceRepo: sourceRepo,
		Packages:   pkgs,
	}

	data, err := json.MarshalIndent(&lock, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(packageDir, lockFileName), data, 0644)
}

// scanPackageDir lists .deb and .rpm files in dir and returns their lock
// entries. Files that fail to stat or hash are skipped with a warning so a
// single corrupt file does not abort the whole write.
func scanPackageDir(dir string) ([]lockPackage, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []lockPackage
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		low := strings.ToLower(name)
		if !strings.HasSuffix(low, ".deb") && !strings.HasSuffix(low, ".rpm") {
			continue
		}
		full := filepath.Join(dir, name)
		fi, err := os.Stat(full)
		if err != nil {
			log.Warnf("scan: stat %s failed: %v", name, err)
			continue
		}
		sum, err := fileSha256(full)
		if err != nil {
			log.Warnf("scan: hash %s failed: %v", name, err)
			continue
		}
		out = append(out, lockPackage{
			Filename: name,
			Size:     fi.Size(),
			Sha256:   sum,
		})
	}
	return out, nil
}

// depEntry is one cached package's metadata, indexed by filename for
// debugging and by package name for resolver lookup. `arch` is the
// architecture qualifier from the deb filename (e.g. "amd64") and is empty
// for rpm packages.
type depEntry struct {
	filename string
	name     string
	version  string
	arch     string
	kind     string // "deb" or "rpm"
	depends  [][]depRef
}

// buildDepIndex scans packageDir and extracts per-package metadata by
// shelling out to dpkg-deb or rpm. Both tools are expected to be present on
// any host that just successfully ran the corresponding Download branch.
func buildDepIndex(packageDir string) (map[string]*depEntry, error) {
	entries, err := os.ReadDir(packageDir)
	if err != nil {
		return nil, err
	}
	index := make(map[string]*depEntry)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		low := strings.ToLower(name)
		var (
			de    *depEntry
			derr  error
			entry depEntry
		)
		entry.filename = name
		switch {
		case strings.HasSuffix(low, ".deb"):
			entry.kind = "deb"
			entry.name, entry.version, entry.arch, entry.depends, derr = extractDebMeta(filepath.Join(packageDir, name))
		case strings.HasSuffix(low, ".rpm"):
			entry.kind = "rpm"
			entry.name, entry.version, entry.depends, derr = extractRpmMeta(filepath.Join(packageDir, name))
		default:
			continue
		}
		if derr != nil {
			// Treat a metadata read failure as an invalid entry — surface it
			// to validateCache so the operator sees a clear cause.
			return nil, fmt.Errorf("extract meta for %s: %w", name, derr)
		}
		de = &entry
		index[name] = de
	}
	return index, nil
}

// extractDebMeta reads Package, Version, Architecture, Depends, Pre-Depends
// from a .deb control file via `dpkg-deb -f`. The returned Depends list is
// a slice of OR-groups, each group being a set of alternative packages that
// any one of which satisfies the dependency.
func extractDebMeta(path string) (name, version, arch string, groups [][]depRef, err error) {
	out, err := captureCmd("dpkg-deb", "-f", path, "Package", "Version", "Architecture", "Depends", "Pre-Depends")
	if err != nil {
		return "", "", "", nil, fmt.Errorf("dpkg-deb -f: %w", err)
	}
	fields := splitFields(out)
	name = fields["Package"]
	version = fields["Version"]
	arch = fields["Architecture"]
	if d := fields["Depends"]; d != "" {
		groups = append(groups, parseDebDeps(d)...)
	}
	if d := fields["Pre-Depends"]; d != "" {
		groups = append(groups, parseDebDeps(d)...)
	}
	if name == "" || version == "" {
		return "", "", "", nil, fmt.Errorf("dpkg-deb output missing Package/Version for %s", path)
	}
	return name, version, arch, groups, nil
}

// extractRpmMeta reads Name, Version, Release, Requires from a .rpm via
// `rpm -qp --qf ...`.
func extractRpmMeta(path string) (name, version string, groups [][]depRef, err error) {
	out, err := captureCmd("rpm", "-qp", "--qf",
		"%{NAME}\n%{VERSION}\n%{RELEASE}\n%{REQUIRES}\n", path)
	if err != nil {
		return "", "", nil, fmt.Errorf("rpm -qp: %w", err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 3 {
		return "", "", nil, fmt.Errorf("rpm output too short for %s", path)
	}
	name = lines[0]
	version = lines[1] + "-" + lines[2] // rpm version = VERSION-RELEASE
	// Remaining lines after line 2 are Requires entries; empty trailing
	// lines are normal for packages with no deps.
	for _, req := range lines[3:] {
		req = strings.TrimSpace(req)
		if req == "" {
			continue
		}
		// rpm Requires format: "name", "name >= version", or "(name if ..."".
		// Conditional deps (the (...) form) are treated as unversioned by
		// name match — finer-grained parsing would require rpm's full
		// dependency grammar; for the purpose of "is there a cached
		// package satisfying this", name-only resolution is sufficient.
		groups = append(groups, parseRpmRequires(req))
	}
	if name == "" {
		return "", "", nil, fmt.Errorf("rpm output missing Name for %s", path)
	}
	return name, version, groups, nil
}

// parseDebDeps parses a Depends/Pre-Depends value. The grammar is:
//
//	dep_list := dep_group ("," dep_group)*
//	dep_group := single (" | " single)*
//	single    := name | name "(" op version ")"
//
// Returns a slice of OR-groups; each group is a set of alternatives any one
// of which satisfies the constraint.
func parseDebDeps(s string) [][]depRef {
	var groups [][]depRef
	for _, raw := range strings.Split(s, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		var orGroup []depRef
		for _, alt := range strings.Split(raw, "|") {
			alt = strings.TrimSpace(alt)
			if alt == "" {
				continue
			}
			orGroup = append(orGroup, parseDebSingle(alt))
		}
		if len(orGroup) > 0 {
			groups = append(groups, orGroup)
		}
	}
	return groups
}

func parseDebSingle(s string) depRef {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, " ("); i >= 0 {
		name := strings.TrimSpace(s[:i])
		rest := strings.TrimSpace(s[i+2:])
		rest = strings.TrimSuffix(rest, ")")
		parts := strings.Fields(rest)
		if len(parts) == 1 {
			// Bare version with no operator: dpkg convention is ">=".
			return depRef{Name: name, Op: ">=", Version: parts[0]}
		}
		if len(parts) >= 2 {
			return depRef{Name: name, Op: parts[0], Version: parts[1]}
		}
		return depRef{Name: name}
	}
	return depRef{Name: s}
}

// parseRpmRequires parses a single rpm Requires line. Conditional requires
// (the "(name if ...)" form) are reduced to the package name; richer
// semantics are not required for the cache validation use case.
func parseRpmRequires(s string) []depRef {
	s = strings.TrimSpace(s)
	// Strip conditional "(... if ...)" wrapper
	if strings.HasPrefix(s, "(") {
		if sp := strings.Index(s, " "); sp > 0 {
			s = s[1:sp]
		} else if strings.HasSuffix(s, ")") {
			s = s[1 : len(s)-1]
		}
	}
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return nil
	}
	dep := depRef{Name: parts[0]}
	if len(parts) >= 3 {
		dep.Op = parts[1]
		dep.Version = parts[2]
	}
	return []depRef{dep}
}

// resolveDep checks whether a single depRef is satisfied by either the
// cached package set or the host's installed package database.
//
// deb names can be qualified with `:any` to mean "any architecture"; for
// such names we strip the suffix and look up by name alone. rpm names are
// matched literally.
//
// The host fallback exists because the bundle only carries the NVIDIA
// toolkit and its apt/yum dependencies — base system libraries such as
// libc6, libssl, libsystemd, libgcc-s1 are not (and should not be) bundled.
// At install time they come from the target host's own base OS, so as long
// as the build host carries them at a compatible version, the dep is
// considered resolvable.
func resolveDep(dep depRef, index map[string]*depEntry) (bool, string) {
	name := strings.SplitN(dep.Name, ":", 2)[0]

	if dep.Op == "" {
		for _, e := range index {
			if strings.EqualFold(e.name, name) {
				return true, ""
			}
		}
		if kind := detectHostPkgMgr(index); kind != "" {
			if _, ok := hostPackageVersion(kind, name); ok {
				return true, ""
			}
		}
		return false, fmt.Sprintf("not cached and not on host: %s", dep.Name)
	}

	for _, e := range index {
		if !strings.EqualFold(e.name, name) {
			continue
		}
		if compareVersions(e.kind, dep.Op, e.version, dep.Version) {
			return true, ""
		}
	}
	if kind := detectHostPkgMgr(index); kind != "" {
		if v, ok := hostPackageVersion(kind, name); ok {
			if compareVersions(kind, dep.Op, v, dep.Version) {
				return true, ""
			}
			return false, fmt.Sprintf("%s %s %s not satisfied by host (%s)", dep.Name, dep.Op, dep.Version, v)
		}
	}
	return false, fmt.Sprintf("%s %s %s not satisfied by any cached or installed package", dep.Name, dep.Op, dep.Version)
}

// detectHostPkgMgr returns "deb" or "rpm" depending on which package
// format dominates the cached index. Used to choose the right host query
// tool when looking up installed packages as a fallback.
func detectHostPkgMgr(index map[string]*depEntry) string {
	deb, rpm := 0, 0
	for _, e := range index {
		switch e.kind {
		case "deb":
			deb++
		case "rpm":
			rpm++
		}
	}
	switch {
	case deb >= rpm:
		return "deb"
	case rpm > 0:
		return "rpm"
	default:
		return ""
	}
}

// hostPackageVersion queries the host's package manager for the installed
// version of name. Returns (version, true) when installed, (_, false)
// otherwise. Errors from the underlying command (e.g. command not found,
// package not installed) are treated as "not installed" rather than as
// hard failures.
func hostPackageVersion(kind, name string) (string, bool) {
	switch kind {
	case "deb":
		// dpkg-query -W -f='${Status}\t${Version}\n' <name>
		// Output when installed: "install ok installed\t<version>"
		out, err := captureCmd("dpkg-query", "-W", "-f=${Status}\t${Version}\n", name)
		if err != nil {
			return "", false
		}
		s := strings.TrimSpace(out)
		if !strings.HasPrefix(s, "install ok installed") {
			return "", false
		}
		parts := strings.SplitN(s, "\t", 2)
		if len(parts) != 2 {
			return "", false
		}
		return strings.TrimSpace(parts[1]), true
	case "rpm":
		// rpm -q --qf '%{VERSION}-%{RELEASE}\n' <name>
		// Empty output (or stderr "package <name> is not installed") means
		// not installed; captureCmd drops stderr so we only see the empty
		// stdout on miss.
		out, err := captureCmd("rpm", "-q", "--qf", "%{VERSION}-%{RELEASE}\n", name)
		if err != nil || strings.TrimSpace(out) == "" {
			return "", false
		}
		return strings.TrimSpace(out), true
	}
	return "", false
}

// compareVersions dispatches to the host's package manager for an
// authoritative comparison. dpkg-deb and rpm do not ship a dedicated
// "compare two version strings" verb; we delegate to dpkg for debs and use
// a lexical comparison for rpms (with the documented caveat that rpm's
// epoch/version/release grammar is richer than a string compare can
// express). For the in-bundle use case where all .rpm versions came down
// in the same download run, string compare is sufficient in practice.
func compareVersions(kind, op, have, want string) bool {
	switch kind {
	case "deb":
		cmd := exec.Command("dpkg", "--compare-versions", have, op, want)
		return cmd.Run() == nil
	case "rpm":
		return rpmStringCompare(op, have, want)
	default:
		return false
	}
}

// rpmStringCompare compares two rpm-style version strings ("epoch:version-release"
// or "version-release"). The semantics are approximated by lex order with the
// caveat that this is not a faithful implementation of rpm's comparator.
func rpmStringCompare(op, v1, v2 string) bool {
	c := compareRPMString(v1, v2)
	switch op {
	case "=", "==":
		return c == 0
	case ">":
		return c > 0
	case ">=":
		return c >= 0
	case "<":
		return c < 0
	case "<=":
		return c <= 0
	default:
		return false
	}
}

// compareRPMString splits "epoch:version-release" into numeric epoch +
// dotted version + release segments and compares segment-by-segment. It is a
// pragmatic approximation of rpm's vercmp; not bytewise equal to rpm -V but
// good enough to detect "this .rpm is the same version we downloaded last
// time".
func compareRPMString(a, b string) int {
	epochA, verA, relA := splitRPMVersion(a)
	epochB, verB, relB := splitRPMVersion(b)
	if c := compareIntStrings(epochA, epochB); c != 0 {
		return c
	}
	if c := compareDotted(verA, verB); c != 0 {
		return c
	}
	return compareDotted(relA, relB)
}

func splitRPMVersion(s string) (epoch, ver, rel string) {
	if i := strings.Index(s, ":"); i >= 0 {
		epoch = s[:i]
		s = s[i+1:]
	} else {
		epoch = "0"
	}
	if i := strings.LastIndex(s, "-"); i >= 0 {
		ver = s[:i]
		rel = s[i+1:]
	} else {
		ver = s
		rel = ""
	}
	return epoch, ver, rel
}

func compareIntStrings(a, b string) int {
	ai, aerr := strconv.Atoi(a)
	bi, berr := strconv.Atoi(b)
	if aerr == nil && berr == nil {
		switch {
		case ai < bi:
			return -1
		case ai > bi:
			return 1
		default:
			return 0
		}
	}
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func compareDotted(a, b string) int {
	pa := strings.Split(a, ".")
	pb := strings.Split(b, ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		if c := compareIntStrings(pa[i], pb[i]); c != 0 {
			return c
		}
	}
	switch {
	case len(pa) < len(pb):
		return -1
	case len(pa) > len(pb):
		return 1
	default:
		return 0
	}
}

// fileSha256 streams the file through sha256.New256 and returns hex.
func fileSha256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := h.Write(nil); err != nil {
		// h.Write never errors on a hash.Hash, but keep the contract
		return "", err
	}
	// Use io.Copy via a tiny buffer to avoid pulling in io for this one call.
	buf := make([]byte, 1<<20)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			h.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// captureCmd runs name with args and returns stdout. stderr is discarded so
// callers see only the parsed payload.
func captureCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// splitFields parses `Key: value\nKey2: value2\n...` output as produced by
// dpkg-deb -f when given a list of field names.
func splitFields(out string) map[string]string {
	m := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if i := strings.Index(line, ":"); i >= 0 {
			key := strings.TrimSpace(line[:i])
			val := strings.TrimSpace(line[i+1:])
			if _, dup := m[key]; !dup || val != "" {
				m[key] = val
			}
		}
	}
	return m
}

// formatGroup renders an OR-group for diagnostic messages.
func formatGroup(group []depRef) string {
	parts := make([]string, len(group))
	for i, d := range group {
		if d.Op == "" {
			parts[i] = d.Name
		} else {
			parts[i] = fmt.Sprintf("%s %s %s", d.Name, d.Op, d.Version)
		}
	}
	return strings.Join(parts, " | ")
}

// short returns the "sha256:abcdef123..." prefix used in log lines.
func short(s string) string {
	const prefix = "sha256:"
	if !strings.HasPrefix(s, prefix) {
		return s
	}
	const hexChars = 12
	tail := s[len(prefix):]
	if len(tail) > hexChars {
		tail = tail[:hexChars]
	}
	return prefix + tail
}

// nowISO returns the current UTC time formatted as RFC3339. Indirected
// through a variable so tests can swap it out.
var nowISO = func() string {
	return time.Now().UTC().Format(time.RFC3339)
}
