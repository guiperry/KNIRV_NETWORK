package incident

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"text/template"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/runner"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

type Incident struct {
	ID             string
	AgentName      string
	AgentType      string
	Status         string
	ExitCode       int
	Timestamp      time.Time
	Environment    string
	ErrorSignature string
	TriggerLine    string
	NRNBounty      float64
	LogSnapshot    []string
	Tags           []string
}

func New(proc *runner.AgentProcess, snapshot []string, triggerLine, sigName string, exitCode int) *Incident {
	id := fmt.Sprintf("INC-%s-%s", time.Now().Format("20060102"), uuid.NewString()[:8])
	last50 := snapshot
	if len(last50) > 50 {
		last50 = last50[len(last50)-50:]
	}
	return &Incident{
		ID:             id,
		AgentName:      proc.AgentName,
		AgentType:      string(proc.Type),
		Status:         "DRAFT",
		ExitCode:       exitCode,
		Timestamp:      time.Now().UTC(),
		Environment:    string(proc.Type),
		ErrorSignature: sigName,
		TriggerLine:    triggerLine,
		NRNBounty:      25.0,
		LogSnapshot:    last50,
		Tags:           []string{"ergo", "error-node", "knirvshell-capture"},
	}
}

const incidentTemplate = `---
id: "{{.ID}}"
agent: "[[{{.AgentName}}]]"
status: "{{.Status}}"
exit_code: {{.ExitCode}}
timestamp: {{.Timestamp.Format "2006-01-02T15:04:05Z07:00"}}
environment: "{{.Environment}}"
error_signature: "{{.ErrorSignature}}"
nrn_bounty_suggested: {{.NRNBounty}}
tags: [{{range $i, $t := .Tags}}{{if $i}}, {{end}}{{$t}}{{end}}]
---

## 🚨 Incident Summary

**Agent:** {{.AgentName}}  
**Signature:** ` + "`{{.ErrorSignature}}`" + `  
**Trigger:** ` + "`{{.TriggerLine}}`" + `  
**Captured:** {{.Timestamp.Format "2006-01-02 15:04:05 UTC"}}

## 🧠 Brain State

> _LLM prompt/response pairs — populate from agent memory if available._

## 🛠 Tool Context

> _Active tool JSON arguments at time of failure — populate from agent context._

## 📜 Raw Forensic Trace

> [!ABSTRACT] Standard Error (stderr — last {{len .LogSnapshot}} lines)
> ` + "```" + `bash
{{range .LogSnapshot}}{{.}}
{{end}}` + "```"

func (inc *Incident) WriteToVault() (string, error) {
	vaultDir := vaultPath()
	filename := inc.ID + ".md"
	path := filepath.Join(vaultDir, "Incidents", filename)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}

	tmpl, err := template.New("incident").Parse(incidentTemplate)
	if err != nil {
		return "", err
	}

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	return path, tmpl.Execute(f, inc)
}

func OpenInObsidian(path string) {
	vaultName := viper.GetString("obsidian.vault_name")
	if vaultName == "" {
		vaultName = "KNIRVVault"
	}

	vaultDir := viper.GetString("obsidian.vault_path")
	if vaultDir == "" {
		home, _ := os.UserHomeDir()
		vaultDir = filepath.Join(home, "Documents", "KNIRVVault")
	}

	relPath, _ := filepath.Rel(vaultDir, path)

	uri := &url.URL{
		Scheme: "obsidian",
		Host:   "open",
	}
	q := uri.Query()
	q.Set("vault", url.PathEscape(vaultName))
	q.Set("file", url.PathEscape(relPath))
	uri.RawQuery = q.Encode()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", uri.String())
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", uri.String())
	default:
		cmd = exec.Command("xdg-open", uri.String())
	}
	cmd.Start()
}

func vaultPath() string {
	if v := viper.GetString("obsidian.vault_path"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Documents", "KNIRVVault")
}
