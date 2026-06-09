package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/dvepod/internal/tee"
)

type BusyBox struct {
	ctx    *tee.Context
	cmdMap map[string]func(args []string, cwd string, env map[string]string) (string, error)
}

func NewBusyBox(ctx *tee.Context) *BusyBox {
	bb := &BusyBox{
		ctx:    ctx,
		cmdMap: make(map[string]func(args []string, cwd string, env map[string]string) (string, error)),
	}

	bb.cmdMap["ls"] = bb.lsCmd
	bb.cmdMap["cat"] = bb.catCmd
	bb.cmdMap["echo"] = bb.echoCmd
	bb.cmdMap["pwd"] = bb.pwdCmd
	bb.cmdMap["env"] = bb.envCmd
	bb.cmdMap["whoami"] = bb.whoamiCmd
	bb.cmdMap["date"] = bb.dateCmd
	bb.cmdMap["uname"] = bb.unameCmd
	bb.cmdMap["ps"] = bb.psCmd
	bb.cmdMap["df"] = bb.dfCmd
	bb.cmdMap["free"] = bb.freeCmd
	bb.cmdMap["mkdir"] = bb.mkdirCmd
	bb.cmdMap["rm"] = bb.rmCmd
	bb.cmdMap["cp"] = bb.cpCmd
	bb.cmdMap["mv"] = bb.mvCmd
	bb.cmdMap["chmod"] = bb.chmodCmd
	bb.cmdMap["wc"] = bb.wcCmd
	bb.cmdMap["grep"] = bb.grepCmd
	bb.cmdMap["find"] = bb.findCmd
	bb.cmdMap["curl"] = bb.curlCmd

	return bb
}

func (bb *BusyBox) Run(cmd string, args []string, cwd string, env map[string]string) (string, error) {
	fn, ok := bb.cmdMap[cmd]
	if !ok {
		return "", fmt.Errorf("dvepod: %s: command not found", cmd)
	}
	return fn(args, cwd, env)
}

func (bb *BusyBox) lsCmd(args []string, cwd string, env map[string]string) (string, error) {
	long := false
	var target string

	for _, a := range args {
		if a == "-l" || a == "-la" || a == "-al" {
			long = true
		} else if !strings.HasPrefix(a, "-") {
			target = a
		}
	}

	if target == "" {
		target = cwd
	} else if !strings.HasPrefix(target, "/") {
		target = cwd + "/" + target
	}
	target = filepath.Clean(target)

	entries, err := os.ReadDir(target)
	if err != nil {
		return "", fmt.Errorf("ls: %s: %v", target, err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var out strings.Builder
	if long {
		fmt.Fprintf(&out, "total %d\n", len(entries))
		for _, e := range entries {
			info, err := e.Info()
			perm := "-rw-r--r--"
			size := "-"
			mtime := ""
			if err == nil {
				perm = info.Mode().String()
				if info.IsDir() {
					perm = "drwxr-xr-x"
				}
				size = fmt.Sprintf("%6d", info.Size())
				mtime = info.ModTime().Format("Jan 02 15:04")
			} else if e.IsDir() {
				perm = "drwxr-xr-x"
			}

			name := e.Name()
			if e.IsDir() {
				name = name + "/"
			}
			fmt.Fprintf(&out, "%s  1 dvepod  %s  %s  %s\n", perm, size, mtime, name)
		}
	} else {
		var names []string
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() {
				name = name + "/"
			}
			names = append(names, name)
		}
		out.WriteString(strings.Join(names, "  "))
		out.WriteString("\n")
	}

	return out.String(), nil
}

func (bb *BusyBox) catCmd(args []string, cwd string, env map[string]string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("cat: missing operand")
	}

	target := args[0]
	if !strings.HasPrefix(target, "/") {
		target = cwd + "/" + target
	}
	target = filepath.Clean(target)

	data, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("cat: %s: %v", args[0], err)
	}
	return string(data), nil
}

func (bb *BusyBox) echoCmd(args []string, cwd string, env map[string]string) (string, error) {
	return strings.Join(args, " ") + "\n", nil
}

func (bb *BusyBox) pwdCmd(args []string, cwd string, env map[string]string) (string, error) {
	return cwd + "\n", nil
}

func (bb *BusyBox) envCmd(args []string, cwd string, env map[string]string) (string, error) {
	var keys []string
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var out strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&out, "%s=%s\n", k, env[k])
	}
	return out.String(), nil
}

func (bb *BusyBox) whoamiCmd(args []string, cwd string, env map[string]string) (string, error) {
	user := env["USER"]
	if user == "" {
		user = "dvepod"
	}
	return user + "\n", nil
}

func (bb *BusyBox) dateCmd(args []string, cwd string, env map[string]string) (string, error) {
	return time.Now().Format("Mon Jan 2 15:04:05 MST 2006") + "\n", nil
}

func (bb *BusyBox) unameCmd(args []string, cwd string, env map[string]string) (string, error) {
	if len(args) > 0 && args[0] == "-a" {
		return fmt.Sprintf("DVEPod dvepod-wasm 1.0.0-wasi WASI #1 SMP %s wasm32\n", time.Now().Format("Mon Jan 2 2006")), nil
	}
	return "DVEPod\n", nil
}

func (bb *BusyBox) psCmd(args []string, cwd string, env map[string]string) (string, error) {
	var out strings.Builder
	out.WriteString("  PID  CMD\n")
	out.WriteString("    1  dvepod-wasm [wasi runtime]\n")
	out.WriteString("    2  knirvagent [solo mode]\n")
	return out.String(), nil
}

func (bb *BusyBox) dfCmd(args []string, cwd string, env map[string]string) (string, error) {
	var out strings.Builder
	out.WriteString("Filesystem      Size  Used  Avail Use% Mounted on\n")
	out.WriteString("wasi-fs         1.0G  4.2M  996M   1% /home/dvepod\n")
	out.WriteString("tmpfs            64M     0   64M   0% /tmp\n")
	return out.String(), nil
}

func (bb *BusyBox) freeCmd(args []string, cwd string, env map[string]string) (string, error) {
	var out strings.Builder
	out.WriteString("              total  used  free  shared  buff/cache  available\n")
	out.WriteString("Mem:          128M   12M  112M     2M        4M       116M\n")
	out.WriteString("Swap:           0     0     0\n")
	return out.String(), nil
}

func (bb *BusyBox) mkdirCmd(args []string, cwd string, env map[string]string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("mkdir: missing operand")
	}

	target := args[0]
	if !strings.HasPrefix(target, "/") {
		target = cwd + "/" + target
	}
	target = filepath.Clean(target)

	err := os.MkdirAll(target, 0755)
	if err != nil {
		return "", fmt.Errorf("mkdir: %v", err)
	}
	return "", nil
}

func (bb *BusyBox) rmCmd(args []string, cwd string, env map[string]string) (string, error) {
	recursive := false
	var targets []string

	for _, a := range args {
		if a == "-r" || a == "-rf" || a == "-fr" {
			recursive = true
		} else if !strings.HasPrefix(a, "-") {
			targets = append(targets, a)
		}
	}

	if len(targets) == 0 {
		return "", fmt.Errorf("rm: missing operand")
	}

	for _, t := range targets {
		target := t
		if !strings.HasPrefix(target, "/") {
			target = cwd + "/" + target
		}
		target = filepath.Clean(target)

		var err error
		if recursive {
			err = os.RemoveAll(target)
		} else {
			err = os.Remove(target)
		}
		if err != nil {
			return "", fmt.Errorf("rm: %s: %v", t, err)
		}
	}
	return "", nil
}

func (bb *BusyBox) cpCmd(args []string, cwd string, env map[string]string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("cp: missing file operand")
	}

	src := args[0]
	dst := args[1]

	if !strings.HasPrefix(src, "/") {
		src = cwd + "/" + src
	}
	if !strings.HasPrefix(dst, "/") {
		dst = cwd + "/" + dst
	}
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	data, err := os.ReadFile(src)
	if err != nil {
		return "", fmt.Errorf("cp: %v", err)
	}

	err = os.WriteFile(dst, data, 0644)
	if err != nil {
		return "", fmt.Errorf("cp: %v", err)
	}
	return "", nil
}

func (bb *BusyBox) mvCmd(args []string, cwd string, env map[string]string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("mv: missing file operand")
	}

	src := args[0]
	dst := args[1]

	if !strings.HasPrefix(src, "/") {
		src = cwd + "/" + src
	}
	if !strings.HasPrefix(dst, "/") {
		dst = cwd + "/" + dst
	}
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	err := os.Rename(src, dst)
	if err != nil {
		return "", fmt.Errorf("mv: %v", err)
	}
	return "", nil
}

func (bb *BusyBox) chmodCmd(args []string, cwd string, env map[string]string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("chmod: missing operand")
	}

	mode, err := strconv.ParseUint(args[0], 8, 32)
	if err != nil {
		return "", fmt.Errorf("chmod: invalid mode: %s", args[0])
	}

	target := args[1]
	if !strings.HasPrefix(target, "/") {
		target = cwd + "/" + target
	}
	target = filepath.Clean(target)

	err = os.Chmod(target, os.FileMode(mode))
	if err != nil {
		return "", fmt.Errorf("chmod: %v", err)
	}
	return "", nil
}

func (bb *BusyBox) wcCmd(args []string, cwd string, env map[string]string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("wc: missing operand")
	}

	target := args[0]
	if !strings.HasPrefix(target, "/") {
		target = cwd + "/" + target
	}
	target = filepath.Clean(target)

	data, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("wc: %s: %v", args[0], err)
	}

	lines := strings.Count(string(data), "\n")
	words := len(strings.Fields(string(data)))
	chars := len(data)

	return fmt.Sprintf("%6d %6d %6d %s\n", lines, words, chars, args[0]), nil
}

func (bb *BusyBox) grepCmd(args []string, cwd string, env map[string]string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("grep: usage: grep <pattern> <file>")
	}

	pattern := args[0]
	target := args[1]

	if !strings.HasPrefix(target, "/") {
		target = cwd + "/" + target
	}
	target = filepath.Clean(target)

	data, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("grep: %s: %v", args[1], err)
	}

	var out strings.Builder
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, pattern) {
			out.WriteString(line + "\n")
		}
	}
	return out.String(), nil
}

func (bb *BusyBox) findCmd(args []string, cwd string, env map[string]string) (string, error) {
	target := cwd
	name := ""

	if len(args) >= 1 {
		p := args[0]
		if !strings.HasPrefix(p, "-") {
			if !strings.HasPrefix(p, "/") {
				p = cwd + "/" + p
			}
			target = filepath.Clean(p)
			if len(args) >= 3 && args[1] == "-name" {
				name = args[2]
			}
		} else if args[0] == "-name" && len(args) >= 2 {
			name = args[1]
		}
	}

	var out strings.Builder
	err := filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if name != "" && !strings.Contains(info.Name(), name) {
			return nil
		}
		rel, _ := filepath.Rel(target, path)
		if rel == "." {
			return nil
		}
		out.WriteString("./" + rel + "\n")
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("find: %v", err)
	}
	return out.String(), nil
}

func (bb *BusyBox) curlCmd(args []string, cwd string, env map[string]string) (string, error) {
	return "", fmt.Errorf("curl: network access not available in solo mode — dock to KNIRVSERVER for network access")
}
