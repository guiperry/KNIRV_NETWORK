package inner

import "os/exec"

func DefaultSpawnHooks(cmd *exec.Cmd, dveID, sessionID string) {
	cmd.Env = append(cmd.Env,
		"KNIRV_DVE_ID="+dveID,
		"KNIRV_SESSION_ID="+sessionID,
		"KNIRV_SUPERVISOR=knirvagent",
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
	)
}
