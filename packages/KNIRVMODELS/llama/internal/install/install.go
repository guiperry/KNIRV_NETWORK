// Package install discovers or provisions the llama.cpp server and a default model.
package install

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/guiperry/knirv/llama/internal/config"
)

const (
	DefaultServerURL = "https://releases.knirv.com/knirv/llama/linux-amd64/llama-server.gz"
	// PinnedLlamaTag is the upstream llama.cpp release every install and update targets.
	// It is a fixed build tag, not a query for the newest release: no install or update
	// asks GitHub what "latest" is, so a new upstream release cannot change what this
	// host builds or runs. Move it deliberately, with a reviewed change here.
	PinnedLlamaTag  = "b12388"
	upstreamRepoURL = "https://github.com/ggml-org/llama.cpp.git"
)

type Options struct {
	DataDir, ServerPath, ServerURL, ModelPath, ModelURL, ModelName string
	NoInstall                                                      bool
	// CheckUpdate compares the resolved llama-server against the latest upstream
	// release and replaces it when older. A failed check or update keeps the
	// working server, so an offline host still starts.
	CheckUpdate bool
}

type Result struct{ ServerPath, ModelPath, ModelName string }

type Installer struct {
	Run func(context.Context, string, ...string) error
	// Output runs a command and returns its combined stdout and stderr.
	Output func(context.Context, string, ...string) (string, error)
	Get    func(string) (*http.Response, error)
	// Conditional issues a GET with request headers (If-None-Match and
	// If-Modified-Since), used to ask whether the release artifact changed.
	Conditional func(string, http.Header) (*http.Response, error)
	LookPath    func(string) (string, error)
	// Prog, when non-nil, receives provisioning progress (download bars and
	// stage lines). Leave nil for silent provisioning, as the tests do.
	Prog *progress
}

func (i *Installer) startProgress(title string, statuses ...string) {
	if i.Prog != nil {
		i.Prog.start(title, 0, statuses...)
	}
}

func (i *Installer) stopProgress(summary string) {
	if i.Prog != nil {
		i.Prog.stop(summary)
	}
}

func (i *Installer) logProgress(format string, args ...any) {
	if i.Prog != nil {
		i.Prog.logf(format, args...)
		return
	}
	fmt.Printf(format, args...)
	if !strings.HasSuffix(format, "\n") {
		fmt.Println()
	}
}

func New() *Installer {
	return &Installer{
		Run: func(ctx context.Context, name string, args ...string) error {
			// Provisioning is intentionally visible to the parent process. In
			// particular, CMake's diagnostic output is the only useful detail
			// when a platform cannot build llama.cpp.
			cmd := exec.CommandContext(ctx, name, args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
		Output: func(ctx context.Context, name string, args ...string) (string, error) {
			out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
			return string(out), err
		},
		Get: http.Get,
		Conditional: func(url string, header http.Header) (*http.Response, error) {
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				return nil, err
			}
			for k, v := range header {
				req.Header[k] = v
			}
			return http.DefaultClient.Do(req)
		},
		LookPath: exec.LookPath,
	}
}

// The two `llama-server --version` forms seen in practice:
//
//	version: 7780 (7dee9ff59)                          git checkout build
//	version: 1.0.0-dev (build 0, commit unknown)       tarball build, no git metadata
//
// The first line of the git form starts with the build number. The tarball form
// starts with the semver "1.0.0-dev", so the build number must come from the
// "(build N" field, and 0 there means the build is unknown.
var (
	// The number must end at whitespace, a parenthesis or the line end, so the
	// semver "1.0.0-dev" never matches as build 1. A bare "version: 7780" also parses.
	buildNumber = regexp.MustCompile(`(?m)^version:\s*(\d+)(?:\s|\(|$)`)
	buildTagged = regexp.MustCompile(`(?m)^version:.*\(build (\d+)`)
)

// errUnknownBuild marks a llama-server that reports a version but no build
// number, such as a tarball build (build 0). It cannot be compared with upstream.
var errUnknownBuild = errors.New("llama-server reports no build number")

// parseBuild extracts the build number from `llama-server --version` output.
func parseBuild(out string) (int, error) {
	if m := buildNumber.FindStringSubmatch(out); m != nil {
		return strconv.Atoi(m[1])
	}
	if m := buildTagged.FindStringSubmatch(out); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, errUnknownBuild
		}
		return n, nil
	}
	return 0, fmt.Errorf("no build number in --version output")
}

// Install stamp. A tarball build reports no build number, so the stamp written
// beside each llama-server we install or build is the only record of what it is.
// For source builds it holds the release tag; for prebuilt artifacts it holds the
// validators (ETag, Last-Modified) and the sha256 of the installed bytes.
const stampFile = "llama-server.tag"

const (
	sourceGitBuild = "git-build"
	sourcePrebuilt = "prebuilt"
)

type installStamp struct {
	Source       string `json:"source"`
	Tag          string `json:"tag,omitempty"`
	ETag         string `json:"etag,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
	SHA256       string `json:"sha256,omitempty"`
	// CheckedAt is the last time the artifact bytes were compared by sha256 (a
	// download, or the install itself). A conditional GET that answers 304 does
	// not move it. It bounds the full re-download for an artifact whose server
	// sends no usable validators.
	CheckedAt   string `json:"checked_at,omitempty"`
	InstalledAt string `json:"installed_at"`
}

// prebuiltRecheckInterval bounds how often a prebuilt artifact is downloaded and
// compared by sha256 when the server cannot answer a conditional GET. Within the
// interval such an artifact is kept without any network request.
const prebuiltRecheckInterval = 24 * time.Hour

// checkDue reports whether the artifact bytes behind a prebuilt stamp are due for
// a full comparison. A missing or unparseable CheckedAt, or one in the future
// (clock moved back), is due.
func checkDue(st installStamp, now time.Time) bool {
	t, err := time.Parse(time.RFC3339, st.CheckedAt)
	if err != nil || t.After(now) {
		return true
	}
	return now.Sub(t) >= prebuiltRecheckInterval
}

// artifactMeta describes the release artifact a prebuilt binary was staged from.
type artifactMeta struct{ ETag, LastModified, SHA256 string }

func stampPath(server string) string { return filepath.Join(filepath.Dir(server), stampFile) }

// readStamp returns (nil, nil) when no stamp exists. A corrupt or unrecognised
// stamp is an error, and callers treat it as absent.
func readStamp(server string) (*installStamp, error) {
	b, err := os.ReadFile(stampPath(server))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var s installStamp
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("corrupt install stamp: %w", err)
	}
	if s.Source != sourceGitBuild && s.Source != sourcePrebuilt {
		return nil, fmt.Errorf("install stamp has unknown source %q", s.Source)
	}
	return &s, nil
}

// writeStamp replaces the stamp atomically: write a temp file in the same
// directory, sync it, then rename it over the stamp.
func writeStamp(server string, s installStamp) error {
	s.InstalledAt = time.Now().UTC().Format(time.RFC3339)
	return writeJSONAtomic(stampPath(server), s)
}

// writeJSONAtomic replaces path with the JSON encoding of v: temp file in the same
// directory, fsync, rename over path, then fsync of the directory.
func writeJSONAtomic(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(append(b, '\n'))
	if serr := tmp.Sync(); werr == nil {
		werr = serr
	}
	if cerr := tmp.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Rename(tmp.Name(), path)
	}
	if werr != nil {
		_ = os.Remove(tmp.Name())
		return werr
	}
	// Make the rename itself durable. Best effort: a directory that cannot be
	// synced still has the file, and a lost stamp only means a later recheck.
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

// recordInstall writes the stamp for a successful install, build or download. A
// failed write does not fail the install: the binary still works, and the next
// start sees no stamp and keeps it.
func (i *Installer) recordInstall(server string, s installStamp) {
	if err := writeStamp(server, s); err != nil {
		i.logProgress("[KNIRVLLAMA] Could not record install stamp for %s: %v", server, err)
	}
}

// parseTagBuild reads a release tag ("b12388") as its build number.
func parseTagBuild(tag string) (int, bool) {
	if !strings.HasPrefix(tag, "b") {
		return 0, false
	}
	n, err := strconv.Atoi(tag[1:])
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// installedBuild returns the llama.cpp build number of a llama-server binary.
func (i *Installer) installedBuild(ctx context.Context, server string) (int, error) {
	out, err := i.Output(ctx, server, "--version")
	n, perr := parseBuild(out)
	if perr == nil {
		return n, nil
	}
	if errors.Is(perr, errUnknownBuild) {
		return 0, perr
	}
	if err != nil {
		return 0, fmt.Errorf("%s --version: %w", server, err)
	}
	return 0, fmt.Errorf("%s: %w", server, perr)
}

// pinnedTag is the release every install and update targets. It is PinnedLlamaTag in
// production; only tests change it, to exercise update logic against other builds.
var pinnedTag = PinnedLlamaTag

// pinnedBuild returns the pinned upstream release tag and its build number. It makes
// no network request.
func (i *Installer) pinnedBuild() (tag string, build int, err error) {
	build, ok := parseTagBuild(pinnedTag)
	if !ok {
		return "", 0, fmt.Errorf("pinned llama.cpp tag %q is not a build tag", pinnedTag)
	}
	return pinnedTag, build, nil
}

// updateServer replaces server with the latest upstream build when it is older.
// It returns the path to use, which is server itself whenever nothing changed or
// the update failed; the working binary is never removed before a replacement
// has been verified.
func (i *Installer) updateServer(ctx context.Context, o Options, server string) string {
	installed, ierr := i.installedBuild(ctx, server)
	if errors.Is(ierr, errUnknownBuild) {
		return i.refreshUnknownServer(ctx, o, server)
	}
	tag, latest, err := i.pinnedBuild()
	if err != nil {
		i.logProgress("[KNIRVLLAMA] Could not check the latest llama.cpp release (%v); keeping %s", err, server)
		return server
	}
	if ierr != nil {
		i.logProgress("[KNIRVLLAMA] Could not read the installed llama.cpp version (%v); updating to %s", ierr, tag)
	} else if installed >= latest {
		i.logProgress("[KNIRVLLAMA] llama.cpp b%d is up to date (latest %s)", installed, tag)
		return server
	} else {
		i.logProgress("[KNIRVLLAMA] llama.cpp b%d is older than %s; updating", installed, tag)
	}

	// The candidate is staged beside the live binary under another name, so a stale
	// or broken artifact can be discarded without touching the server that works.
	staged, meta, hold, err := i.fetchCandidate(ctx, o, tag)
	var refused *refusedArtifact
	switch {
	case errors.As(err, &refused):
		i.stopProgress("")
		i.recordRefused(o.DataDir, tag, refused.meta, refused.reason)
		i.logProgress("[KNIRVLLAMA] Prebuilt llama-server for %s %s; building it from source", tag, refused.reason)
	case err != nil:
		i.stopProgress("")
		fmt.Printf("[KNIRVLLAMA] Prebuilt llama-server unavailable (%v); building %s from source\n", err, tag)
	case hold:
		// The refusal stands and no comparison is due: neither the artifact nor the
		// source build is retried until the recheck interval has passed.
		return server
	case staged == "":
		i.logProgress("[KNIRVLLAMA] Prebuilt llama-server for %s is still refused (same bytes); building it from source", tag)
	default:
		got, verr := i.installedBuild(ctx, staged)
		if verr == nil && got >= latest {
			dest := filepath.Join(o.DataDir, "bin", "llama-server")
			if err := os.Rename(staged, dest); err == nil {
				_ = os.Remove(rejectedPath(o.DataDir))
				i.recordInstall(dest, prebuiltStamp(fmt.Sprintf("b%d", got), meta))
				i.logProgress("[KNIRVLLAMA] Updated llama-server to b%d", got)
				return dest
			} else {
				// The candidate verified but could not be put in place. This is not a fault
				// of the bytes, so no refusal is recorded: the same bytes must still be
				// installed once the obstruction is gone. Dropping any marker means the next
				// start downloads the artifact again (one download per start, as before).
				_ = os.Remove(rejectedPath(o.DataDir))
				i.logProgress("[KNIRVLLAMA] Could not put the verified llama-server for %s in place (%v); it will be tried again on the next start", tag, err)
			}
		} else {
			reason := fmt.Sprintf("reports build %d, not %s", got, tag)
			if verr != nil {
				reason = "reports no build number"
			}
			i.recordRefused(o.DataDir, tag, meta, reason)
		}
		_ = os.Remove(staged)
		i.logProgress("[KNIRVLLAMA] Prebuilt llama-server is not %s yet; building it from source", tag)
	}
	return i.buildUpdate(ctx, o, server, tag)
}

// buildAttemptsFile is a sidecar beside the binary: tag -> RFC3339 time of the last
// source-build attempt for that tag. It is not part of the install stamp or the
// refusal marker, so both keep their formats. A missing or unreadable file means no
// attempt is recorded, which is what an install from before this file looks like.
const buildAttemptsFile = "llama-server.build-attempts"

func buildAttemptsPath(dataDir string) string {
	return filepath.Join(dataDir, "bin", buildAttemptsFile)
}

// readBuildAttempts returns the attempts that are still within the recheck interval.
// An entry that is older, or dated in the future (clock moved back), is dropped, so
// it cannot block a build forever.
func readBuildAttempts(dataDir string, now time.Time) map[string]string {
	out := map[string]string{}
	b, err := os.ReadFile(buildAttemptsPath(dataDir))
	if err != nil {
		return out
	}
	// Decode each entry on its own, so one malformed entry (not a string) cannot
	// discard the valid attempts for the other tags.
	var m map[string]json.RawMessage
	if json.Unmarshal(b, &m) != nil {
		return out
	}
	for tag, raw := range m {
		var at string
		if json.Unmarshal(raw, &at) != nil {
			continue
		}
		t, err := time.Parse(time.RFC3339, at)
		if err == nil && !t.After(now) && now.Sub(t) < prebuiltRecheckInterval {
			out[tag] = at
		}
	}
	return out
}

// The sidecar read-modify-write in claimBuildAttempt is guarded by a lock file beside
// it, created with O_EXCL so it works on every platform. The lock is held only for
// that read-modify-write, never across a build, so a holder that died mid-write
// leaves a lock that is safe to break once it is this old.
const (
	buildAttemptsLockWait  = 5 * time.Second
	buildAttemptsLockStale = 30 * time.Second
)

// lockBuildAttempts takes the sidecar lock and returns its release function.
func lockBuildAttempts(dataDir string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(buildAttemptsPath(dataDir)), 0o755); err != nil {
		return nil, err
	}
	path := buildAttemptsPath(dataDir) + ".lock"
	deadline := time.Now().Add(buildAttemptsLockWait)
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_ = f.Close()
			return func() { _ = os.Remove(path) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if fi, serr := os.Stat(path); serr == nil && time.Since(fi.ModTime()) > buildAttemptsLockStale {
			_ = os.Remove(path)
			continue
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for %s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// errBuildBounded is returned when a source build for a tag was already attempted
// within the recheck interval. The bound applies on every path that builds a tag.
type errBuildBounded struct {
	tag  string
	last time.Time
}

func (e *errBuildBounded) Error() string {
	return fmt.Sprintf("a source build of %s was attempted at %s; the next attempt is allowed after %s",
		e.tag, e.last.UTC().Format(time.RFC3339), e.last.Add(prebuiltRecheckInterval).UTC().Format(time.RFC3339))
}

// buildTagOnce builds tag from source, at most once per recheck interval. The attempt
// is recorded before the build runs, so a build that fails or is interrupted still
// counts. An attempt for another tag does not block this one.
func (i *Installer) buildTagOnce(ctx context.Context, dataDir, tag string) (string, error) {
	if err := i.claimBuildAttempt(dataDir, tag, time.Now()); err != nil {
		return "", err
	}
	return i.installServer(ctx, dataDir, tag)
}

// claimBuildAttempt checks the per-tag bound and records this attempt, under the
// sidecar lock, so two concurrent callers cannot both see "no attempt" and both
// build. If the lock cannot be taken, it fails open: the check runs unlocked and the
// build proceeds as it did before the lock existed.
func (i *Installer) claimBuildAttempt(dataDir, tag string, now time.Time) error {
	unlock, lerr := lockBuildAttempts(dataDir)
	if lerr != nil {
		i.logProgress("[KNIRVLLAMA] Could not lock the source build record (%v); checking it without the lock", lerr)
	} else {
		defer unlock()
	}
	attempts := readBuildAttempts(dataDir, now)
	if last, ok := attempts[tag]; ok {
		t, _ := time.Parse(time.RFC3339, last)
		return &errBuildBounded{tag: tag, last: t}
	}
	attempts[tag] = now.UTC().Format(time.RFC3339)
	if err := writeJSONAtomic(buildAttemptsPath(dataDir), attempts); err != nil {
		// A failed record only means the next start may try again, as before this bound.
		i.logProgress("[KNIRVLLAMA] Could not record the source build attempt for %s (%v); it may be attempted again on the next start", tag, err)
	}
	return nil
}

// buildUpdate is the source-build step of an update to tag. When the per-tag bound
// skips the build, the current server is kept and the log says why.
func (i *Installer) buildUpdate(ctx context.Context, o Options, server, tag string) string {
	built, err := i.buildTagOnce(ctx, o.DataDir, tag)
	var bounded *errBuildBounded
	if errors.As(err, &bounded) {
		i.logProgress("[KNIRVLLAMA] Not building %s again yet: %v; keeping %s", tag, err, server)
		return server
	}
	if err != nil {
		fmt.Printf("[KNIRVLLAMA] Update to %s failed (%v); keeping %s\n", tag, err, server)
		return server
	}
	i.logProgress("[KNIRVLLAMA] Built llama-server %s", tag)
	return built
}

// refreshUnknownServer decides, for a llama-server that reports no build number,
// from its install stamp. Such a binary cannot be compared by version. A stamp
// says what it is. A managed binary with no stamp is adopted once: it is
// compared by sha256 against the release artifact, so an install made before
// stamps existed still receives updates. Any other unstamped binary is kept.
func (i *Installer) refreshUnknownServer(ctx context.Context, o Options, server string) string {
	st, err := readStamp(server)
	if err != nil {
		i.logProgress("[KNIRVLLAMA] Ignoring unreadable install stamp for %s (%v)", server, err)
		st = nil
	}
	if st == nil {
		if filepath.Clean(server) != filepath.Join(o.DataDir, "bin", "llama-server") {
			i.logProgress("[KNIRVLLAMA] %s reports no llama.cpp build number and has no install stamp; keeping it", server)
			return server
		}
		sum, err := fileSHA256(server)
		if err != nil {
			i.logProgress("[KNIRVLLAMA] Could not hash %s to adopt it (%v); keeping it", server, err)
			return server
		}
		i.logProgress("[KNIRVLLAMA] Adopting the unstamped llama-server %s: comparing it with the release artifact once", server)
		st = &installStamp{Source: sourcePrebuilt, SHA256: sum}
	}
	if st.Source == sourceGitBuild {
		return i.refreshStampedBuild(ctx, o, server, *st)
	}
	return i.refreshPrebuilt(ctx, o, server, *st)
}

// refreshStampedBuild rebuilds a source build only when its stamped tag is older
// than the latest upstream release. A stamp with no resolvable tag counts as
// build 0, so it is rebuilt once at the latest tag and then stamped with it.
func (i *Installer) refreshStampedBuild(ctx context.Context, o Options, server string, st installStamp) string {
	stamped, ok := parseTagBuild(st.Tag)
	if !ok {
		stamped = 0
	}
	tag, latest, err := i.pinnedBuild()
	if err != nil {
		i.logProgress("[KNIRVLLAMA] Could not check the latest llama.cpp release (%v); keeping %s", err, server)
		return server
	}
	if stamped >= latest {
		i.logProgress("[KNIRVLLAMA] llama.cpp b%d is up to date (latest %s)", stamped, tag)
		return server
	}
	i.logProgress("[KNIRVLLAMA] llama.cpp (stamped %q) is older than %s; updating", st.Tag, tag)
	return i.buildUpdate(ctx, o, server, tag)
}

// refreshPrebuilt replaces a prebuilt artifact only when the artifact changed.
// A conditional GET asks first, and a 304 downloads nothing. When the server
// gives no usable validator, the artifact bytes are compared by sha256, at most
// once per prebuiltRecheckInterval. A stamp with no validators therefore costs a
// download only when it is due, never on every start.
func (i *Installer) refreshPrebuilt(ctx context.Context, o Options, server string, st installStamp) string {
	hasValidators := st.ETag != "" || st.LastModified != ""
	due := checkDue(st, time.Now())
	if !hasValidators && !due {
		i.logProgress("[KNIRVLLAMA] %s has no ETag or Last-Modified; %s, keeping it", server, nextCheckNote(st))
		return server
	}
	staged, meta, status, err := i.fetchIfChanged(ctx, o.DataDir, artifactURL(o), "llama-server.update", st, due)
	if err != nil {
		i.logProgress("[KNIRVLLAMA] Could not check the prebuilt llama-server (%v); keeping %s", err, server)
		return server
	}
	switch status {
	case fetchSkipped:
		i.logProgress("[KNIRVLLAMA] The release artifact gave no validator; %s, keeping %s", nextCheckNote(st), server)
		return server
	case fetchNotModified:
		// Keep CheckedAt: a conditional answer is not a byte comparison.
		updated := prebuiltStamp(st.Tag, meta)
		updated.CheckedAt = st.CheckedAt
		if updated.ETag != st.ETag || updated.LastModified != st.LastModified {
			i.recordInstall(server, updated)
		}
		i.logProgress("[KNIRVLLAMA] Prebuilt llama-server is unchanged; keeping %s", server)
		return server
	case fetchVerified:
		i.recordInstall(server, prebuiltStamp(st.Tag, meta))
		i.logProgress("[KNIRVLLAMA] Prebuilt llama-server is unchanged; keeping %s", server)
		return server
	}
	dest := filepath.Join(o.DataDir, "bin", "llama-server")
	if err := os.Rename(staged, dest); err != nil {
		_ = os.Remove(staged)
		i.logProgress("[KNIRVLLAMA] Could not install the changed prebuilt llama-server (%v); keeping %s", err, server)
		return server
	}
	tag := ""
	if got, err := i.installedBuild(ctx, dest); err == nil {
		tag = fmt.Sprintf("b%d", got)
	}
	i.recordInstall(dest, prebuiltStamp(tag, meta))
	i.logProgress("[KNIRVLLAMA] Updated llama-server from the changed release artifact")
	return dest
}

// nextCheckNote says when a stamp with no validators is next compared by sha256.
// An unreadable last-check time is reported as such: it is due at once, and the
// log must not claim a next comparison at the current time.
func nextCheckNote(st installStamp) string {
	t, err := time.Parse(time.RFC3339, st.CheckedAt)
	if err != nil {
		return "the last comparison time is missing or unreadable, so a comparison is due now"
	}
	return fmt.Sprintf("last compared at %s, next comparison after %s", st.CheckedAt, t.Add(prebuiltRecheckInterval).Format(time.RFC3339))
}

// prebuiltStamp records a prebuilt install. It counts as a full comparison of the
// artifact bytes, so CheckedAt is set to now.
func prebuiltStamp(tag string, meta artifactMeta) installStamp {
	return installStamp{
		Source: sourcePrebuilt, Tag: tag,
		ETag: meta.ETag, LastModified: meta.LastModified, SHA256: meta.SHA256,
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func artifactURL(o Options) string {
	if o.ServerURL != "" {
		return o.ServerURL
	}
	return DefaultServerURL
}

// fetchStatus is the outcome of fetchIfChanged.
type fetchStatus int

const (
	// fetchNotModified: a 304, or a 200 whose validator matches the stamp. The
	// bytes were not downloaded.
	fetchNotModified fetchStatus = iota
	// fetchVerified: the body was downloaded and hashes to the stamp's sha256.
	fetchVerified
	// fetchChanged: the body was downloaded, differs, and is staged at the path returned.
	fetchChanged
	// fetchSkipped: the response had no validators and a full comparison is not
	// due. Nothing was downloaded.
	fetchSkipped
)

// fetchIfChanged asks the release artifact whether it differs from prev, with a
// conditional GET. A 304, or a 200 whose validator matches prev, is unchanged
// without a download. Otherwise the body is downloaded only when due: a changed
// validator with the same bytes is not replaced, and a changed body is staged at
// <dataDir>/bin/<name>. When not due, a body whose validators are absent or
// changed is skipped and the next comparison falls on its due date.
func (i *Installer) fetchIfChanged(ctx context.Context, dataDir, url, name string, prev installStamp, due bool) (string, artifactMeta, fetchStatus, error) {
	header := http.Header{}
	if prev.ETag != "" {
		header.Set("If-None-Match", prev.ETag)
	}
	if prev.LastModified != "" {
		header.Set("If-Modified-Since", prev.LastModified)
	}
	resp, err := i.Conditional(url, header)
	if err != nil {
		return "", artifactMeta{}, 0, fmt.Errorf("check prebuilt llama-server: %w", err)
	}
	defer resp.Body.Close()
	prevMeta := artifactMeta{ETag: prev.ETag, LastModified: prev.LastModified, SHA256: prev.SHA256}
	if resp.StatusCode == http.StatusNotModified {
		return "", prevMeta, fetchNotModified, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", artifactMeta{}, 0, fmt.Errorf("check prebuilt llama-server: unexpected HTTP status %s", resp.Status)
	}
	remote := artifactMeta{ETag: resp.Header.Get("ETag"), LastModified: resp.Header.Get("Last-Modified")}
	if artifactUnchanged(remote, prevMeta) {
		// The server ignored the conditional headers but the artifact is the same.
		return "", mergeValidators(remote, prevMeta), fetchNotModified, nil
	}
	if !due {
		return "", artifactMeta{}, fetchSkipped, nil
	}
	i.startProgress("Provisioning KNIRVLLAMA", "downloading prebuilt llama-server...", "decompressing release archive...", "verifying executable...")
	path, staged, err := i.stageServer(ctx, dataDir, resp, url, name)
	if err != nil {
		return "", artifactMeta{}, 0, err
	}
	if prev.SHA256 != "" && staged.SHA256 == prev.SHA256 {
		_ = os.Remove(path)
		return "", mergeValidators(remote, prevMeta), fetchVerified, nil
	}
	return path, staged, fetchChanged, nil
}

// artifactUnchanged reports whether a response's validators show the artifact the
// stored validators describe. An ETag decides when either side has one. Last-Modified
// is weak: it decides only when neither side carries an ETag, so a server that
// adds an ETag while keeping its timestamp is still compared by sha256.
func artifactUnchanged(remote, prev artifactMeta) bool {
	if prev.ETag != "" {
		return remote.ETag == prev.ETag
	}
	return remote.ETag == "" && prev.LastModified != "" && remote.LastModified == prev.LastModified
}

// mergeValidators keeps the remote validators and fills any the response omitted
// from the previous stamp, so a headerless 200 cannot erase them.
func mergeValidators(remote, prev artifactMeta) artifactMeta {
	m := artifactMeta{ETag: remote.ETag, LastModified: remote.LastModified, SHA256: prev.SHA256}
	if m.ETag == "" {
		m.ETag = prev.ETag
	}
	if m.LastModified == "" {
		m.LastModified = prev.LastModified
	}
	return m
}

// fileSHA256 returns the hex sha256 of a file's bytes.
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (i *Installer) Ensure(ctx context.Context, o Options) (Result, error) {
	if o.DataDir == "" {
		return Result{}, fmt.Errorf("data directory is required")
	}
	if o.ModelName == "" {
		o.ModelName = config.DefaultModelName
	}
	if o.ModelURL == "" {
		o.ModelURL = config.DefaultModelURL
	}

	serverPaths := append([]string{o.ServerPath, filepath.Join(o.DataDir, "llama.cpp", "build", "bin", "llama-server")}, legacyServerPaths()...)
	server := firstFile(serverPaths...)
	if server == "" {
		if o.NoInstall {
			return Result{}, fmt.Errorf("llama-server was not found and installation is disabled")
		}
		serverURL := artifactURL(o)
		i.logProgress("[KNIRVLLAMA] Downloading prebuilt llama-server from %s\n", serverURL)
		var downloadErr error
		var meta artifactMeta
		server, meta, downloadErr = i.downloadServerMeta(ctx, o.DataDir, serverURL, "llama-server")
		if downloadErr == nil {
			i.recordInstall(server, prebuiltStamp("", meta))
		}
		if downloadErr != nil {
			// The native build streams large amounts of CMake output straight to
			// the terminal, so retire any in-place bar before it starts.
			i.stopProgress("")
			// A prebuilt CPU binary is portable across the supported Debian/Kali
			// image. Keep a native build as a fallback for incompatible libc/CPU
			// combinations and for deployments that require host-native tuning.
			fmt.Printf("[KNIRVLLAMA] Prebuilt llama-server unavailable (%v); falling back to a native build\n", downloadErr)
			var err error
			server, err = i.installServer(ctx, o.DataDir, pinnedTag)
			if err != nil {
				return Result{}, fmt.Errorf("provision llama-server: prebuilt download failed: %v; native build failed: %w", downloadErr, err)
			}
		}
	} else if o.CheckUpdate && !o.NoInstall {
		server = i.updateServer(ctx, o, server)
	}
	modelPaths := []string{o.ModelPath, filepath.Join(o.DataDir, "models", o.ModelName)}
	// Legacy installations only contain the bundled TinyLlama model. Do not let
	// one override an explicitly selected model name or URL.
	if o.ModelName == config.DefaultModelName && o.ModelURL == config.DefaultModelURL {
		modelPaths = append(modelPaths, legacyModelPaths()...)
	}
	model := firstFile(modelPaths...)
	if model == "" {
		if o.NoInstall {
			return Result{}, fmt.Errorf("model was not found and installation is disabled")
		}
		var err error
		model, err = i.downloadModel(o.DataDir, o.ModelName, o.ModelURL)
		if err != nil {
			return Result{}, err
		}
	}
	return Result{ServerPath: server, ModelPath: model, ModelName: strings.TrimSuffix(filepath.Base(model), filepath.Ext(model))}, nil
}

// installServer builds llama-server from source. A non-empty tag builds that
// upstream release (fetching it into an existing checkout); empty builds the
// default branch head on a fresh clone.
func (i *Installer) installServer(ctx context.Context, dataDir, tag string) (string, error) {
	dir := filepath.Join(dataDir, "llama.cpp")
	if err := i.ensureCommands(ctx, map[string]string{"git": "git"}); err != nil {
		return "", err
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Println("[KNIRVLLAMA] Cloning llama.cpp for native build")
		args := []string{"clone", "--depth", "1"}
		if tag != "" {
			args = append(args, "--branch", tag)
		}
		if err := i.Run(ctx, "git", append(args, upstreamRepoURL, dir)...); err != nil {
			return "", fmt.Errorf("clone llama.cpp: %w", err)
		}
	} else if tag != "" {
		fmt.Printf("[KNIRVLLAMA] Fetching llama.cpp %s\n", tag)
		if err := i.Run(ctx, "git", "-C", dir, "fetch", "--depth", "1", "origin", "tag", tag, "--no-tags"); err != nil {
			return "", fmt.Errorf("fetch llama.cpp %s: %w", tag, err)
		}
		if err := i.Run(ctx, "git", "-C", dir, "checkout", "--force", "FETCH_HEAD"); err != nil {
			return "", fmt.Errorf("checkout llama.cpp %s: %w", tag, err)
		}
	}
	if err := i.ensureCommands(ctx, map[string]string{"cmake": "cmake", "cc": "build-essential", "c++": "build-essential"}); err != nil {
		return "", err
	}
	build := filepath.Join(dir, "build")
	if v := os.Getenv("KNIRV_LLAMA_CUDA"); v != "" && v != "0" && v != "1" {
		i.logProgress("[KNIRVLLAMA] KNIRV_LLAMA_CUDA=%q is not 0 or 1; ignoring it and detecting the CUDA toolkit", v)
	}
	cuda := cudaRequested(os.Getenv, i.LookPath)
	fmt.Printf("[KNIRVLLAMA] Configuring native llama.cpp fallback build (CUDA %s)\n", map[bool]string{true: "on", false: "off"}[cuda])
	if err := i.Run(ctx, "cmake", cmakeConfigureArgs(dir, build, cuda)...); err != nil {
		return "", fmt.Errorf("configure llama.cpp: %w", err)
	}
	if err := i.Run(ctx, "cmake", "--build", build, "--target", "llama-server", "--config", "Release"); err != nil {
		return "", fmt.Errorf("build llama.cpp server: %w", err)
	}
	path := firstFile(filepath.Join(build, "bin", "llama-server"), filepath.Join(build, "bin", "Release", "llama-server"))
	if path == "" {
		return "", fmt.Errorf("llama.cpp build completed but llama-server was not produced")
	}
	// A default-branch build has no tag. Record the build number it reports, so the
	// stamp can be compared with upstream on later starts.
	stampTag := tag
	if stampTag == "" {
		if got, err := i.installedBuild(ctx, path); err == nil {
			stampTag = fmt.Sprintf("b%d", got)
		}
	}
	i.recordInstall(path, installStamp{Source: sourceGitBuild, Tag: stampTag})
	// A source build of this tag supersedes any refused prebuilt artifact for it.
	if rej := readRejected(dataDir); rej != nil && rej.Tag == stampTag {
		_ = os.Remove(rejectedPath(dataDir))
	}
	return path, nil
}

// cudaRequested decides whether the native fallback build enables CUDA.
// KNIRV_LLAMA_CUDA=1 or =0 overrides detection. Otherwise CUDA is enabled only when a
// toolkit is present: a CUDA_HOME directory, or nvcc on PATH. CPU is the default.
func cudaRequested(getenv func(string) string, lookPath func(string) (string, error)) bool {
	switch getenv("KNIRV_LLAMA_CUDA") {
	case "1":
		return true
	case "0":
		return false
	}
	if home := getenv("CUDA_HOME"); home != "" {
		if st, err := os.Stat(home); err == nil && st.IsDir() {
			return true
		}
	}
	_, err := lookPath("nvcc")
	return err == nil
}

// cmakeConfigureArgs is the configure step of the native fallback build.
//
// -DLLAMA_USE_PREBUILT_UI=OFF stops llama.cpp from fetching its prebuilt web UI,
// since the installer downloads nothing beyond the llama.cpp sources.
// -DLLAMA_BUILD_UI=OFF is explicit as well, so the no-npm, no-network guarantee does
// not depend on upstream's default.
//
// GGML_CUDA is always passed explicitly. The build directory is reused between
// updates and CMake keeps the previous cache value, so omitting it would leave a
// CUDA build configured after the toolkit has gone (or a CPU build after it arrived).
func cmakeConfigureArgs(src, build string, cuda bool) []string {
	gpu := "-DGGML_CUDA=OFF"
	if cuda {
		gpu = "-DGGML_CUDA=ON"
	}
	return []string{"-S", src, "-B", build, "-DLLAMA_BUILD_SERVER=ON", "-DLLAMA_USE_PREBUILT_UI=OFF", "-DLLAMA_BUILD_UI=OFF", gpu}
}

// ensureCommands installs only missing native-build dependencies. It is called
// solely after both a cached server and the release artifact are unavailable.
func (i *Installer) ensureCommands(ctx context.Context, packages map[string]string) error {
	missing := make(map[string]struct{})
	for command, pkg := range packages {
		if _, err := i.LookPath(command); err != nil {
			missing[pkg] = struct{}{}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	if _, err := i.LookPath("apt-get"); err != nil {
		return fmt.Errorf("missing native-build dependencies %v and apt-get is unavailable", missing)
	}
	args := make([]string, 0, len(missing)+3)
	for pkg := range missing {
		args = append(args, pkg)
	}
	sort.Strings(args)
	fmt.Printf("[KNIRVLLAMA] Installing missing native-build packages: %s\n", strings.Join(args, ", "))
	if err := i.Run(ctx, "apt-get", "update"); err != nil {
		return fmt.Errorf("apt-get update: %w", err)
	}
	installArgs := append([]string{"install", "--yes", "--no-install-recommends"}, args...)
	if err := i.Run(ctx, "apt-get", installArgs...); err != nil {
		return fmt.Errorf("apt-get install %s: %w", strings.Join(args, ", "), err)
	}
	return nil
}

func (i *Installer) downloadServer(ctx context.Context, dataDir, url, name string) (string, error) {
	path, _, err := i.downloadServerMeta(ctx, dataDir, url, name)
	return path, err
}

// downloadServerMeta fetches the artifact at url and stages it under name, also
// returning the artifact validators and the installed binary's sha256.
func (i *Installer) downloadServerMeta(ctx context.Context, dataDir, url, name string) (string, artifactMeta, error) {
	i.startProgress("Provisioning KNIRVLLAMA", "downloading prebuilt llama-server...", "decompressing release archive...", "verifying executable...")
	dir := filepath.Join(dataDir, "bin")
	if err := os.MkdirAll(dir, 0755); err != nil {
		i.stopProgress("")
		return "", artifactMeta{}, fmt.Errorf("create server directory: %w", err)
	}
	resp, err := i.Get(url)
	if err != nil {
		i.stopProgress("")
		return "", artifactMeta{}, fmt.Errorf("download prebuilt llama-server: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		i.stopProgress("")
		return "", artifactMeta{}, fmt.Errorf("download prebuilt llama-server: unexpected HTTP status %s", resp.Status)
	}
	return i.stageServer(ctx, dataDir, resp, url, name)
}

// stageServer writes the response body to <dataDir>/bin/<name> (gunzipping .gz
// URLs) and verifies that it runs. Progress must already be started by the caller.
func (i *Installer) stageServer(ctx context.Context, dataDir string, resp *http.Response, url, name string) (string, artifactMeta, error) {
	meta := artifactMeta{ETag: resp.Header.Get("ETag"), LastModified: resp.Header.Get("Last-Modified")}
	dir := filepath.Join(dataDir, "bin")
	cr := &countingReader{r: resp.Body, prog: i.Prog}
	reader := io.Reader(cr)
	if strings.HasSuffix(strings.ToLower(url), ".gz") {
		gz, err := gzip.NewReader(cr)
		if err != nil {
			i.stopProgress("")
			return "", artifactMeta{}, fmt.Errorf("read prebuilt llama-server archive: %w", err)
		}
		defer gz.Close()
		reader = gz
	}
	path := filepath.Join(dir, name)
	tmp := path + ".part"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		i.stopProgress("")
		return "", artifactMeta{}, err
	}
	h := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(f, h), reader)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		i.stopProgress("")
		return "", artifactMeta{}, fmt.Errorf("write prebuilt llama-server: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		i.stopProgress("")
		return "", artifactMeta{}, closeErr
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		i.stopProgress("")
		return "", artifactMeta{}, fmt.Errorf("finalize prebuilt llama-server: %w", err)
	}
	meta.SHA256 = hex.EncodeToString(h.Sum(nil))
	// Verify the artifact before caching it permanently. This catches a release
	// built against an incompatible glibc or CPU feature set and lets Ensure
	// continue into the native-build fallback on the same first run.
	i.stopProgress(fmt.Sprintf("[KNIRVLLAMA] Installed prebuilt llama-server (%s)", fmtBytes(cr.total)))
	if err := i.Run(ctx, path, "--version"); err != nil {
		_ = os.Remove(path)
		return "", artifactMeta{}, &refusedArtifact{
			meta: meta, reason: "does not run",
			err: fmt.Errorf("verify prebuilt llama-server: %w", err),
		}
	}
	return path, meta, nil
}

// refusedArtifact is returned when a downloaded artifact cannot be used. Its
// validators and sha256 let the caller record it, so the next start does not
// download the same bytes again.
type refusedArtifact struct {
	meta   artifactMeta
	reason string
	err    error
}

func (e *refusedArtifact) Error() string { return e.err.Error() }
func (e *refusedArtifact) Unwrap() error { return e.err }

// rejectedFile records the last release artifact that the numeric path downloaded
// and refused, beside the binary. It is not an install stamp: nothing was installed.
const rejectedFile = "llama-server.rejected"

// rejectedArtifact is the refused-artifact marker. CheckedAt is the last full
// sha256 comparison of the artifact bytes.
type rejectedArtifact struct {
	Tag          string `json:"tag"`
	ETag         string `json:"etag,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
	SHA256       string `json:"sha256,omitempty"`
	Reason       string `json:"reason,omitempty"`
	CheckedAt    string `json:"checked_at"`
}

func rejectedPath(dataDir string) string { return filepath.Join(dataDir, "bin", rejectedFile) }

// readRejected returns the marker, or nil when there is none or it is unreadable.
func readRejected(dataDir string) *rejectedArtifact {
	b, err := os.ReadFile(rejectedPath(dataDir))
	if err != nil {
		return nil
	}
	var r rejectedArtifact
	if json.Unmarshal(b, &r) != nil || r.Tag == "" {
		return nil
	}
	return &r
}

// recordRefused stores that the artifact for tag was downloaded and refused. A
// failed write only means the next start downloads it again, so it is logged.
func (i *Installer) recordRefused(dataDir, tag string, meta artifactMeta, reason string) {
	i.recordRefusedAt(dataDir, tag, meta, reason, time.Now())
}

// recordRefusedAt is recordRefused with an explicit checked_at. A hold within the
// recheck interval keeps the previous checked_at, so a flaky host cannot push the
// next full comparison further away.
func (i *Installer) recordRefusedAt(dataDir, tag string, meta artifactMeta, reason string, checked time.Time) {
	r := rejectedArtifact{Tag: tag, ETag: meta.ETag, LastModified: meta.LastModified, SHA256: meta.SHA256, Reason: reason}
	r.CheckedAt = checked.UTC().Format(time.RFC3339)
	if err := writeJSONAtomic(rejectedPath(dataDir), r); err != nil {
		i.logProgress("[KNIRVLLAMA] Could not record the refused llama-server artifact (%v); it will be downloaded again on the next start", err)
	}
}

// fetchCandidate decides, for the numeric path, whether the release artifact for
// tag is worth taking. It returns:
//   - a staged path, when the artifact bytes differ from the refused ones;
//   - hold, when the refusal stands and nothing is due, so neither the artifact
//     nor a source build is attempted this start;
//   - neither, when a full comparison found the same bytes (the caller builds from
//     source once; the next comparison is a recheck interval away).
//
// A marker for another tag, or none, means a plain download. A marker for this tag
// is compared in full (no conditional headers) once checked_at is one recheck
// interval old, so a server that keeps its ETag across a re-upload cannot hide a
// fixed build. Within the interval the stored validators are sent, and a 304 or a
// matching validator holds without a body download.
func (i *Installer) fetchCandidate(ctx context.Context, o Options, tag string) (string, artifactMeta, bool, error) {
	url := artifactURL(o)
	rej := readRejected(o.DataDir)
	if rej == nil || rej.Tag != tag {
		path, meta, err := i.downloadServerMeta(ctx, o.DataDir, url, "llama-server.update")
		return path, meta, false, err
	}
	prev := artifactMeta{ETag: rej.ETag, LastModified: rej.LastModified, SHA256: rej.SHA256}
	checked := installStamp{CheckedAt: rej.CheckedAt}
	if checkDue(checked, time.Now()) {
		path, meta, hold, err := i.compareRefused(ctx, o, tag, url, rej)
		var refused *refusedArtifact
		if err != nil && !errors.As(err, &refused) {
			// The due comparison itself failed: this attempt is the recheck, so it is stamped.
			return i.holdOnHostError(o.DataDir, tag, prev, rej, err, true)
		}
		return path, meta, hold, err
	}
	hasValidators := prev.ETag != "" || prev.LastModified != ""
	if !hasValidators {
		i.logProgress("[KNIRVLLAMA] Prebuilt llama-server for %s was refused (%s); %s", tag, rej.Reason, nextCheckNote(checked))
		return "", artifactMeta{}, true, nil
	}
	header := http.Header{}
	if prev.ETag != "" {
		header.Set("If-None-Match", prev.ETag)
	}
	if prev.LastModified != "" {
		header.Set("If-Modified-Since", prev.LastModified)
	}
	resp, err := i.Conditional(url, header)
	if err != nil {
		return i.holdOnHostError(o.DataDir, tag, prev, rej, fmt.Errorf("check prebuilt llama-server: %w", err), false)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		i.logProgress("[KNIRVLLAMA] Prebuilt llama-server for %s was refused (%s) and is unchanged; %s", tag, rej.Reason, nextCheckNote(checked))
		return "", artifactMeta{}, true, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return i.holdOnHostError(o.DataDir, tag, prev, rej, fmt.Errorf("download prebuilt llama-server: unexpected HTTP status %s", resp.Status), false)
	}
	remote := artifactMeta{ETag: resp.Header.Get("ETag"), LastModified: resp.Header.Get("Last-Modified")}
	if artifactUnchanged(remote, prev) {
		i.logProgress("[KNIRVLLAMA] Prebuilt llama-server for %s was refused (%s) and is unchanged; %s", tag, rej.Reason, nextCheckNote(checked))
		return "", artifactMeta{}, true, nil
	}
	// The validator changed: download and compare the bytes now.
	i.startProgress("Provisioning KNIRVLLAMA", "downloading prebuilt llama-server...", "decompressing release archive...", "verifying executable...")
	path, meta, err := i.stageServer(ctx, o.DataDir, resp, url, "llama-server.update")
	if err != nil {
		var refused *refusedArtifact
		if errors.As(err, &refused) {
			return "", artifactMeta{}, false, err
		}
		return i.holdOnHostError(o.DataDir, tag, mergeValidators(remote, prev), rej, err, false)
	}
	if prev.SHA256 != "" && meta.SHA256 == prev.SHA256 {
		// Same bytes, so the refusal still holds. Record the comparison and the
		// validators the server now sends.
		_ = os.Remove(path)
		i.recordRefused(o.DataDir, tag, mergeValidators(remote, prev), rej.Reason)
		return "", artifactMeta{}, true, nil
	}
	return path, meta, false, nil
}

// holdOnHostError handles an artifact-host failure while a refusal for tag is held.
// The marker is kept and no source build runs. When dueAttempt is set, the failed
// full comparison is the recheck, so checked_at is stamped now and the next one is a
// recheck interval away. Otherwise (a request made within the interval) checked_at is
// kept: a host that keeps failing cannot slide the full comparison forward, so it is
// attempted at least once per interval of wall time. validators from the failed
// response are used only when they were read.
func (i *Installer) holdOnHostError(dataDir, tag string, validators artifactMeta, rej *rejectedArtifact, err error, dueAttempt bool) (string, artifactMeta, bool, error) {
	i.logProgress("[KNIRVLLAMA] Could not check the refused llama-server for %s (%v); keeping the refusal until the next check (%s)", tag, err, rej.Reason)
	checked := time.Now()
	if !dueAttempt {
		if t, perr := time.Parse(time.RFC3339, rej.CheckedAt); perr == nil {
			checked = t
		}
	}
	i.recordRefusedAt(dataDir, tag, validators, rej.Reason, checked)
	return "", artifactMeta{}, true, nil
}

// compareRefused is the full comparison of a refused artifact, due once per recheck
// interval. It sends no conditional headers, so the body is downloaded and compared
// by sha256 whatever the validators say.
func (i *Installer) compareRefused(ctx context.Context, o Options, tag, url string, rej *rejectedArtifact) (string, artifactMeta, bool, error) {
	prev := artifactMeta{ETag: rej.ETag, LastModified: rej.LastModified, SHA256: rej.SHA256}
	resp, err := i.Get(url)
	if err != nil {
		return "", artifactMeta{}, false, fmt.Errorf("check prebuilt llama-server: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		// The artifact for this tag is not published. That is not a host outage and can
		// last indefinitely, so it is not held: it is "no usable artifact", and the caller
		// builds from source. The marker is then stamped, so the next attempt waits a
		// recheck interval.
		return "", artifactMeta{}, false, &refusedArtifact{
			meta: prev, reason: fmt.Sprintf("is not published (HTTP %d)", resp.StatusCode),
			err: fmt.Errorf("download prebuilt llama-server: unexpected HTTP status %s", resp.Status),
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", artifactMeta{}, false, fmt.Errorf("download prebuilt llama-server: unexpected HTTP status %s", resp.Status)
	}
	remote := artifactMeta{ETag: resp.Header.Get("ETag"), LastModified: resp.Header.Get("Last-Modified")}
	i.startProgress("Provisioning KNIRVLLAMA", "downloading prebuilt llama-server...", "decompressing release archive...", "verifying executable...")
	path, meta, err := i.stageServer(ctx, o.DataDir, resp, url, "llama-server.update")
	if err != nil {
		return "", artifactMeta{}, false, err
	}
	if prev.SHA256 != "" && meta.SHA256 == prev.SHA256 {
		_ = os.Remove(path)
		i.recordRefused(o.DataDir, tag, mergeValidators(remote, prev), rej.Reason)
		i.logProgress("[KNIRVLLAMA] Prebuilt llama-server for %s is still refused (%s): the artifact is unchanged after a full comparison", tag, rej.Reason)
		return "", artifactMeta{}, false, nil
	}
	return path, meta, false, nil
}

func (i *Installer) downloadModel(dataDir, name, url string) (string, error) {
	i.startProgress("Provisioning model weights", "downloading model weights...", "verifying download...")
	dir := filepath.Join(dataDir, "models")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create models directory: %w", err)
	}
	path := filepath.Join(dir, name)
	tmp := path + ".part"
	resp, err := i.Get(url)
	if err != nil {
		i.stopProgress("")
		return "", fmt.Errorf("download model: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		i.stopProgress("")
		return "", fmt.Errorf("download model: unexpected HTTP status %s", resp.Status)
	}
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		i.stopProgress("")
		return "", err
	}
	cr := &countingReader{r: resp.Body, prog: i.Prog}
	_, copyErr := io.Copy(f, cr)
	closeErr := f.Close()
	if copyErr != nil {
		i.stopProgress("")
		return "", fmt.Errorf("write model: %w", copyErr)
	}
	if closeErr != nil {
		i.stopProgress("")
		return "", closeErr
	}
	if err := os.Rename(tmp, path); err != nil {
		i.stopProgress("")
		return "", fmt.Errorf("finalize model download: %w", err)
	}
	i.stopProgress(fmt.Sprintf("[KNIRVLLAMA] Model ready (%s)", fmtBytes(cr.total)))
	return path, nil
}

func firstFile(paths ...string) string {
	for _, path := range paths {
		if path == "" {
			continue
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}
func legacyServerPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, "llama.cpp", "build", "bin", "llama-server"), filepath.Join(home, "models", "llama.cpp", "build", "bin", "llama-server")}
}
func legacyModelPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, "models", "tinyllama-1.1b-chat-v1.0.Q4_0.gguf"), filepath.Join(home, "tinyllama-1.1b-chat-v1.0.Q4_0.gguf")}
}
