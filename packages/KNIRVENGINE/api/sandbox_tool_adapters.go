package api

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

// toolPathArgs is shared by file-oriented consoles. TargetDir defaults to the
// session's explicit project mount; callers cannot make a tool inspect an
// arbitrary host path outside the sandbox configuration.
type toolPathArgs struct {
	TargetPath string `json:"targetPath"`
	TargetDir  string `json:"targetDir"`
	Script     string `json:"script"`
	Interface  string `json:"interface"`
}

func decodeToolPath(session *SandboxSession, raw json.RawMessage) toolPathArgs {
	args := toolPathArgs{}
	if session != nil {
		args.TargetDir = mountedTargetDir(session)
	}
	_ = json.Unmarshal(raw, &args)
	if args.TargetDir == "" && session != nil {
		args.TargetDir = mountedTargetDir(session)
	}
	if args.TargetPath == "" {
		args.TargetPath = args.TargetDir
	}
	return args
}

func init() {
	// Lane 1: file-oriented batch consoles.
	registerLane1Tool("jadx", toolScanAdapter{binary: "jadx", buildArgs: func(s *SandboxSession, raw json.RawMessage) ([]string, error) {
		a := decodeToolPath(s, raw)
		return []string{"--show-bad-code", "--no-replace-consts", a.TargetPath}, nil
	}})
	registerLane1Tool("ilspy", toolScanAdapter{binary: "ilspycmd", buildArgs: func(s *SandboxSession, raw json.RawMessage) ([]string, error) {
		a := decodeToolPath(s, raw)
		return []string{"--ilcode", a.TargetPath}, nil
	}})
	registerLane1Tool("jwt-tool", toolScanAdapter{binary: "jwt_tool.py", buildArgs: func(_ *SandboxSession, raw json.RawMessage) ([]string, error) {
		var req struct {
			Token string   `json:"token"`
			Args  []string `json:"args"`
		}
		_ = json.Unmarshal(raw, &req)
		if req.Token == "" {
			return nil, fmt.Errorf("token is required")
		}
		return append([]string{req.Token}, req.Args...), nil
	}})

	// Lane 2: joined long-running observers. Arguments intentionally accept an
	// operator-supplied trace/filter but retain safe, useful defaults.
	registerLane2Tool("bpftrace", lane2Adapter{binary: "bpftrace", needsJoin: true, buildArgs: func(_ *SandboxSession, raw json.RawMessage) ([]string, error) {
		a := decodeToolPath(nil, raw)
		script := a.Script
		if script == "" {
			script = "tracepoint:syscalls:sys_enter_* { printf(\"%s\\n\", probe); }"
		}
		return []string{"-e", script}, nil
	}})
	registerLane2Tool("tshark", lane2Adapter{binary: "tshark", needsJoin: true, buildArgs: func(_ *SandboxSession, raw json.RawMessage) ([]string, error) {
		a := decodeToolPath(nil, raw)
		iface := a.Interface
		if iface == "" {
			iface = "any"
		}
		return []string{"-l", "-i", iface}, nil
	}})
	registerLane2Tool("zeek", lane2Adapter{binary: "zeek", needsJoin: true, buildArgs: func(_ *SandboxSession, raw json.RawMessage) ([]string, error) {
		a := decodeToolPath(nil, raw)
		iface := a.Interface
		if iface == "" {
			iface = "any"
		}
		return []string{"-i", iface}, nil
	}})
	registerLane2Tool("afl-fuzz", lane2Adapter{binary: "afl-fuzz", needsJoin: true, buildArgs: func(s *SandboxSession, raw json.RawMessage) ([]string, error) {
		a := decodeToolPath(s, raw)
		return []string{"-i", filepath.Join(a.TargetDir, "in"), "-o", filepath.Join(a.TargetDir, "out"), "--", a.TargetPath}, nil
	}})

	// Lane 3: Frida CLI provides the process-attached interactive bridge.
	registerLane3Tool("frida", lane3Adapter{binary: "frida", needsJoin: true, buildArgs: func(_ *SandboxSession, pid int, raw json.RawMessage) ([]string, error) {
		var req struct {
			Script string `json:"script"`
		}
		_ = json.Unmarshal(raw, &req)
		args := []string{"-p", fmt.Sprintf("%d", pid), "-q"}
		if req.Script != "" {
			args = append(args, "-l", req.Script)
		}
		return args, nil
	}})

	// Lane 5: Cutter's rizin engine emits the native analysis that the Cutter
	// React console renders instead of embedding a second desktop GUI.
	registerLane5Tool("cutter", lane5Adapter{binary: "rizin", analysisCmd: func(_ *SandboxSession, binaryPath string, _ json.RawMessage) ([]string, error) {
		return []string{"-2", "-q", "-c", "aaa;aflj;pdgj", binaryPath}, nil
	}})
}
