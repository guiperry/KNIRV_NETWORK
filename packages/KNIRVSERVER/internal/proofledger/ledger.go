package proofledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"knirv-server/internal/dveevidence"
	"knirv-server/internal/knirvproof"
)

const (
	ReceivePackRequest       = "application/x-git-receive-pack-request"
	ReceivePackAdvertisement = "application/x-git-receive-pack-advertisement"
	ReceivePackResult        = "application/x-git-receive-pack-result"
	ReceivePackService       = "git-receive-pack"
)

type Ledger struct {
	Root      string
	Ingest    *dveevidence.IngestService
	Authorize knirvproof.Authorizer
}

func New(root string, ingest *dveevidence.IngestService, authorize knirvproof.Authorizer) (*Ledger, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("proof ledger root is required")
	}
	if ingest == nil {
		return nil, fmt.Errorf("proof ledger ingest service is required")
	}
	if authorize == nil {
		return nil, fmt.Errorf("proof ledger authorizer is required")
	}
	return &Ledger{Root: root, Ingest: ingest, Authorize: authorize}, nil
}

func (l *Ledger) RepoPath(projectID string) (string, error) {
	if err := validateProjectID(projectID); err != nil {
		return "", err
	}
	return filepath.Join(l.Root, projectID+".git"), nil
}

func (l *Ledger) Create(projectID string) (string, error) {
	path, err := l.RepoPath(projectID)
	if err != nil {
		return "", err
	}
	if info, statErr := os.Stat(path); statErr == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("proof ledger path is not a directory")
		}
		out, verifyErr := gitOutput(context.Background(), path, "rev-parse", "--is-bare-repository")
		if verifyErr != nil || strings.TrimSpace(string(out)) != "true" {
			return "", fmt.Errorf("proof ledger path exists but is not a bare repository: %s", path)
		}
		return path, nil
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", statErr
	}
	if err := os.MkdirAll(l.Root, 0o700); err != nil {
		return "", err
	}
	cmd := exec.Command("git", "init", "--bare", "--template", "", path)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("create proof ledger: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return path, nil
}

func RegisterRoutes(r gin.IRouter, ledger *Ledger) {
	r.POST("/git/:project_id.git/create", ledger.create)
	r.GET("/git/:project_id.git/info/refs", ledger.infoRefs)
	r.POST("/git/:project_id.git/git-receive-pack", ledger.receivePack)
}

func (l *Ledger) create(c *gin.Context) {
	projectID := c.Param("project_id")
	if !l.authorize(c, projectID, knirvproof.ActionProofSubmit) {
		return
	}
	path, err := l.Create(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"project_id": projectID, "path": path, "ref_namespace": "refs/knirv/dve/"})
}

func (l *Ledger) authorize(c *gin.Context, projectID, action string) bool {
	token, err := credentialToken(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return false
	}
	principal, err := l.Authorize.Authorize(c.Request.Context(), token, projectID, action)
	if err != nil {
		status := http.StatusForbidden
		if errors.Is(err, knirvproof.ErrUnauthorized) {
			status = http.StatusUnauthorized
		} else if errors.Is(err, knirvproof.ErrAuthUnavailable) {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return false
	}
	if principal == nil || principal.ID == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "authorization service omitted principal"})
		return false
	}
	c.Set("knirv_principal_id", principal.ID)
	return true
}

func credentialToken(r *http.Request) (string, error) {
	username, password, ok := r.BasicAuth()
	if !ok || strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return "", knirvproof.ErrUnauthorized
	}
	return password, nil
}

func (l *Ledger) infoRefs(c *gin.Context) {
	projectID := c.Param("project_id")
	if !l.authorize(c, projectID, knirvproof.ActionProofRead) {
		return
	}
	if c.Query("service") != ReceivePackService {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service=git-receive-pack is required"})
		return
	}
	repo, err := l.Create(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Type", ReceivePackAdvertisement)
	c.Header("Cache-Control", "no-cache")
	c.Header("Pragma", "no-cache")
	c.Header("X-Content-Type-Options", "nosniff")
	// Git's smart-HTTP protocol requires the info/refs response to open with
	// a pkt-line service announcement + flush-pkt before the ref
	// advertisement itself, or every real git client rejects it with
	// "invalid server response" — `git receive-pack --advertise-refs` alone
	// does not emit this; the HTTP layer must prepend it.
	if _, err := c.Writer.Write(pktLine("# service=" + ReceivePackService + "\n")); err != nil {
		return
	}
	if _, err := c.Writer.Write([]byte("0000")); err != nil {
		return
	}
	if err := runGit(c.Request.Context(), repo, []string{"receive-pack", "--stateless-rpc", "--advertise-refs", repo}, nil, c.Writer); err != nil {
		return
	}
}

// pktLine encodes s as a git pkt-line: a 4-byte hex length prefix (counting
// itself) followed by s.
func pktLine(s string) []byte {
	return []byte(fmt.Sprintf("%04x%s", len(s)+4, s))
}

func (l *Ledger) receivePack(c *gin.Context) {
	projectID := c.Param("project_id")
	if !l.authorize(c, projectID, knirvproof.ActionProofSubmit) {
		return
	}
	repo, err := l.Create(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Type", ReceivePackResult)
	c.Header("Cache-Control", "no-cache")
	if err := runGit(c.Request.Context(), repo, []string{"receive-pack", "--stateless-rpc", repo}, c.Request.Body, c.Writer); err != nil {
		return
	}
	if err := l.ingestRefs(c.Request.Context(), projectID, repo); err != nil {
		// The receive-pack response has already been sent. The client cannot
		// safely retry the git transaction, so leave the ref in place and expose
		// the validation failure through the server log/status path.
		c.Error(fmt.Errorf("proof ledger ingest for %s: %w", projectID, err))
	}
}

func runGit(ctx context.Context, repo string, args []string, stdin io.Reader, stdout io.Writer) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_DIR="+repo)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w", args[0], err)
	}
	return nil
}

func (l *Ledger) ingestRefs(ctx context.Context, projectID, repo string) error {
	refs, err := gitOutput(ctx, repo, "for-each-ref", "--format=%(refname)", "refs/knirv/dve")
	if err != nil {
		return err
	}
	for _, ref := range strings.Fields(refs) {
		if !strings.HasPrefix(ref, "refs/knirv/dve/") {
			continue
		}
		files, err := gitOutput(ctx, repo, "ls-tree", "-r", "--name-only", ref)
		if err != nil {
			return err
		}
		bundlePath := findLedgerFile(files, "bundle.json")
		if bundlePath == "" {
			return fmt.Errorf("ref %s does not contain bundle.json", ref)
		}
		bundleJSON, err := gitOutput(ctx, repo, "show", ref+":"+bundlePath)
		if err != nil {
			return err
		}
		var bundle dveevidence.Bundle
		if err := json.Unmarshal([]byte(bundleJSON), &bundle); err != nil {
			return fmt.Errorf("decode %s bundle: %w", ref, err)
		}
		evidence := &dveevidence.Evidence{ArtifactHashes: make([]string, 0, len(bundle.Artifacts))}
		for _, artifact := range bundle.Artifacts {
			evidence.ArtifactHashes = append(evidence.ArtifactHashes, artifact.Hash)
		}
		eventsPath := findLedgerFile(files, "events.jsonl")
		if eventsPath != "" {
			eventsJSON, showErr := gitOutput(ctx, repo, "show", ref+":"+eventsPath)
			if showErr != nil {
				return showErr
			}
			for _, line := range strings.Split(strings.TrimSpace(eventsJSON), "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				var event dveevidence.Event
				if err := json.Unmarshal([]byte(line), &event); err != nil {
					return fmt.Errorf("decode %s event: %w", ref, err)
				}
				evidence.Events = append(evidence.Events, event)
			}
		}
		if _, err := l.Ingest.Ingest(&bundle, evidence); err != nil {
			return fmt.Errorf("ingest %s: %w", ref, err)
		}
	}
	return nil
}

func gitOutput(ctx context.Context, repo string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_DIR="+repo)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return string(output), nil
}

func findLedgerFile(list, name string) string {
	var matches []string
	for _, line := range strings.Split(list, "\n") {
		line = strings.TrimSpace(line)
		if line == name || strings.HasSuffix(line, "/"+name) {
			matches = append(matches, line)
		}
	}
	sort.Strings(matches)
	if len(matches) == 0 {
		return ""
	}
	return matches[0]
}

func validateProjectID(value string) error {
	if value == "" || value == "." || value == ".." || strings.ContainsAny(value, `/\\`) {
		return fmt.Errorf("invalid project id")
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return fmt.Errorf("invalid project id")
	}
	return nil
}
