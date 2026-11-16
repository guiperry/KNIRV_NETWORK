package launcher

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"knirv-server/internal/knirvbao"
)

// Tests for OpenBao-down behaviour of startOpenBao (N12.9, pendingImplementations/OpenBao.md
// §4.4 and PendingImplementations.md rule 7). Everything here uses a fake seal source and
// temp directories: no real openbao.env, key file, vault, or child process is touched.
// The seal-error text mirrors the SDK's "key file is locked" message (secrets/seal.go) and
// names the password variable only; the password value is never set.

// degradedSealSource stands in for the SDK FileProvider when the node-local OpenBao cannot
// be unsealed (key file missing, password unset, or sealed vault).
type degradedSealSource struct {
	key []byte
	err error
}

func (s degradedSealSource) SealKey(context.Context) ([]byte, error) { return s.key, s.err }

const degradedSealErrText = "key file is locked: ORACLE_KEY_PASSWORD is not set"

// degradedBaoApp returns an app whose OpenBao manager is a real knirvbao.Manager driven by
// src, with every app-data and config path redirected to temp dirs.
func degradedBaoApp(t *testing.T, enabled bool, networkMode string, src knirvbao.SealSource) (*ServerApp, *knirvbao.Manager) {
	t.Helper()
	t.Setenv("KNIRV_APP_DATA_DIR", t.TempDir())
	t.Setenv("KNIRV_CONFIG_DIR", t.TempDir())
	cfg := &Config{NetworkMode: networkMode}
	cfg.Secrets.OpenBao.Enabled = enabled
	mgrDir := t.TempDir()
	mgr := knirvbao.NewManager(knirvbao.Config{Enabled: true, DataDir: mgrDir, ConfigDir: mgrDir}, src)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = mgr.Stop(ctx)
	})
	return &ServerApp{config: cfg, bao: mgr}, mgr
}

// TestBaoDownAutoAndBaoSurfaceDegradedErrorWithoutExiting: with OpenBao unreachable in
// auto or bao mode, startOpenBao returns the error to its caller (which logs the WARN and
// continues in auto, or fails closed in bao) and records a Degraded status with the reason.
// It must not panic, exit, or swallow the error. Covers OpenBao.md §4.4 ("auto falls back
// to File with a WARN, bao fails closed") and rule 7.
func TestBaoDownAutoAndBaoSurfaceDegradedErrorWithoutExiting(t *testing.T) {
	for _, mode := range []string{"auto", "bao"} {
		for _, netMode := range []string{"testnet", "production"} {
			t.Run(mode+"/"+netMode, func(t *testing.T) {
				t.Setenv(secretsBackendEnv, mode)
				app, mgr := degradedBaoApp(t, true, netMode, degradedSealSource{err: errors.New(degradedSealErrText)})
				err := app.startOpenBao(context.Background())
				if err == nil {
					t.Fatal("startOpenBao returned nil with OpenBao unreachable; the caller cannot WARN or fail closed")
				}
				if !strings.Contains(err.Error(), "key file is locked") {
					t.Fatalf("error does not carry the seal failure: %v", err)
				}
				st := mgr.Health()
				if st.State != knirvbao.Degraded {
					t.Fatalf("status = %v, want Degraded", st.State)
				}
				if !strings.Contains(st.Reason, "key file is locked") {
					t.Fatalf("degraded reason lost: %q", st.Reason)
				}
			})
		}
	}
}

// TestBaoDownFileModeNeverStartsOpenBao: file mode never runs OpenBao, so an unreachable
// vault cannot fail startup and the manager is never started.
func TestBaoDownFileModeNeverStartsOpenBao(t *testing.T) {
	t.Setenv(secretsBackendEnv, "file")
	app, mgr := degradedBaoApp(t, true, "production", degradedSealSource{err: errors.New(degradedSealErrText)})
	if err := app.startOpenBao(context.Background()); err != nil {
		t.Fatalf("file mode returned error %v; OpenBao must not be started", err)
	}
	if st := mgr.Health(); st.State != knirvbao.Absent {
		t.Fatalf("file mode started OpenBao: status = %v", st.State)
	}
}

// TestBaoDownProdMatchesTestnet: -prod (NetworkMode production) takes exactly the same
// path as the testnet default for OpenBao-down: same error text, same manager status, no
// production-only fail-closed branch. Covers N12.9 ("no blanket fail-closed").
func TestBaoDownProdMatchesTestnet(t *testing.T) {
	type outcome struct {
		errText string
		status  knirvbao.Status
	}
	for _, mode := range []string{"auto", "bao"} {
		run := func(netMode string) outcome {
			t.Setenv(secretsBackendEnv, mode)
			app, mgr := degradedBaoApp(t, true, netMode, degradedSealSource{err: errors.New(degradedSealErrText)})
			var msg string
			if err := app.startOpenBao(context.Background()); err != nil {
				msg = err.Error()
			}
			return outcome{errText: msg, status: mgr.Health()}
		}
		testnet := run("testnet")
		prod := run("production")
		if !reflect.DeepEqual(testnet, prod) {
			t.Fatalf("mode=%s: production differs from testnet:\n testnet=%+v\n prod=%+v", mode, testnet, prod)
		}
	}
}

// TestBaoAbsentSealCurrentBehaviour is a KNOWN-ISSUE PIN, not a statement of intended behaviour.
// It pins what startOpenBao does when there is no key file at all (seal source returns no key
// and no error, i.e. OpenBao is Absent). In auto this is a quiet no-OpenBao start. In bao it
// returns nil, so the launcher's fail-closed branch does not fire, which differs from
// OpenBao.md §4.4 ("bao: fail closed if unreachable").
//
// Open owner decision (N12.9 gap 1, validation issue 2): fail closed on Absent in bao mode,
// or document bao-with-no-key as a no-OpenBao start. When the owner decides, update this test
// together with the production code; if the behaviour changes first, this test fails on purpose.
func TestBaoAbsentSealCurrentBehaviour(t *testing.T) {
	for _, mode := range []string{"auto", "bao"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv(secretsBackendEnv, mode)
			app, mgr := degradedBaoApp(t, true, "production", degradedSealSource{})
			if err := app.startOpenBao(context.Background()); err != nil {
				t.Fatalf("mode=%s: absent seal returned error %v (behaviour change: update the open item)", mode, err)
			}
			if st := mgr.Health(); st.State != knirvbao.Absent {
				t.Fatalf("mode=%s: status = %v, want Absent", mode, st.State)
			}
		})
	}
}
