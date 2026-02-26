// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package runtime

import (
	"backend_server/internal/ebpf"
)

// OhMyPiAgentSyscalls defines the syscalls required by the oh-my-pi agent runtime.
// These include: python execution, git operations, curl/HTTP, LSP servers, browser.
var OhMyPiAgentSyscalls = []uint32{
	0,   // read
	1,   // write
	2,   // open
	3,   // close
	4,   // stat
	5,   // fstat
	6,   // lstat
	8,   // lseek
	9,   // mmap
	10,  // mprotect
	11,  // munmap
	12,  // brk
	16,  // ioctl
	17,  // pread64
	18,  // pwrite64
	19,  // readv
	20,  // writev
	21,  // access
	22,  // pipe
	23,  // select
	24,  // sched_yield
	32,  // dup
	33,  // dup2
	35,  // nanosleep
	39,  // getpid
	41,  // socket
	42,  // connect
	43,  // accept
	44,  // sendto
	45,  // recvfrom
	49,  // bind
	50,  // listen
	54,  // setsockopt
	55,  // getsockopt
	56,  // clone (for subprocess execution: git, python)
	57,  // fork
	58,  // vfork
	59,  // execve (for tool invocations)
	60,  // exit
	61,  // wait4
	62,  // kill
	78,  // getdents
	79,  // getcwd
	80,  // chdir
	82,  // rename
	83,  // mkdir
	84,  // rmdir
	85,  // creat
	86,  // link
	87,  // unlink
	88,  // symlink
	90,  // chmod
	92,  // chown
	110, // getppid
	157, // prctl
	201, // time
	202, // futex
	217, // getdents64
	231, // exit_group
	257, // openat
	258, // mkdirat
	259, // mknodat
	262, // newfstatat
	263, // unlinkat
	266, // symlinkat
	280, // utimensat
	281, // epoll_pwait
	291, // epoll_create1
	292, // dup3
	293, // pipe2
	302, // prlimit64
	317, // seccomp
	318, // getrandom
	332, // statx
}

// OhMyPiAgentPaths defines the filesystem paths the agent is permitted to access
var OhMyPiAgentPaths = []string{
	"/tmp",
	"/proc/self",
	"/workspace",
	"/usr",
	"/lib",
	"/lib64",
	"/bin",
	"/sbin",
	"/etc/ssl",
	"/etc/resolv.conf",
	"/etc/hosts",
	"/dev/null",
	"/dev/urandom",
	"/dev/random",
}

// OhMyPiAgentNetworks defines allowed network CIDRs for the agent
var OhMyPiAgentNetworks = []string{
	"127.0.0.0/8",   // loopback
	"10.0.0.0/8",    // DVE internal network
	"172.16.0.0/12", // Docker bridge networks
}

// NewOhMyPiAgentPolicyConfig creates a security policy config for the oh-my-pi agent.
// This allows the agent its necessary tool access while enforcing DVE isolation.
func NewOhMyPiAgentPolicyConfig() *ebpf.AgentPolicyConfig {
	return &ebpf.AgentPolicyConfig{
		AllowNetwork:    true,
		AllowFilesystem: true,
		RequireTEE:      false, // TEE optional for agents
		MaxMemoryMB:     8192,
		MaxCPUPercent:   80,
		AllowedSyscalls: OhMyPiAgentSyscalls,
		AllowedPaths:    OhMyPiAgentPaths,
		AllowedNetworks: OhMyPiAgentNetworks,
	}
}
