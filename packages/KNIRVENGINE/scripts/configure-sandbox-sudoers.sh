#!/usr/bin/env bash
#
# Install a deliberately narrow sudoers policy for two one-time setup needs:
# the Bubblewrap runtime dependency install, and reapplying the sandbox
# helpers' file capabilities. Routine sandbox tool attachment itself does not
# use sudo; it uses the capability-carrying bundled nsenter/bpftrace/
# frida-server helpers instead.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

if [ "$(uname -s)" != "Linux" ]; then
	echo "[configure-sandbox-sudoers] skipped: sandbox dependencies are Linux-only"
	exit 0
fi

operator="${SUDO_USER:-$(id -un)}"
if [ "$operator" = "root" ]; then
	echo "[configure-sandbox-sudoers] ERROR: run this setup as the desktop user, not root" >&2
	exit 1
fi

manager=""
for candidate in apt-get dnf microdnf yum pacman zypper apk; do
	if command -v "$candidate" >/dev/null 2>&1; then
		manager="$candidate"
		break
	fi
done
if [ -z "$manager" ]; then
	echo "[configure-sandbox-sudoers] ERROR: no supported package manager found" >&2
	exit 1
fi

manager_path="$(command -v "$manager")"
case "$manager" in
	apt-get)
		# x11vnc lives in Ubuntu's universe repo, which enableExtraReposForTool
		# (api/sandbox_deps.go) enables on demand via add-apt-repository before
		# retrying the install. Allow-list that call too, or the repo-enable
		# step silently no-ops under `sudo -n` and x11vnc install can fail on a
		# fresh Ubuntu host that doesn't already have universe enabled.
		addrepo_path="$(command -v add-apt-repository 2>/dev/null || echo /usr/bin/add-apt-repository)"
		commands="$manager_path update, $manager_path install -y bubblewrap, $manager_path install -y xvfb, $manager_path install -y x11vnc, $addrepo_path -y universe"
		;;
	dnf|microdnf|yum)
		# x11vnc on RHEL-like distros needs EPEL, which enableExtraReposForTool
		# enables on demand the same way. Allow-list it for the same reason as
		# add-apt-repository above.
		commands="$manager_path install -y bubblewrap, $manager_path install -y xorg-x11-server-Xvfb, $manager_path install -y x11vnc, $manager_path install -y epel-release"
		;;
	pacman)
		commands="$manager_path -S --noconfirm bubblewrap, $manager_path -S --noconfirm xorg-server-xvfb, $manager_path -S --noconfirm x11vnc"
		;;
	zypper)
		commands="$manager_path install -y bubblewrap, $manager_path install -y xorg-x11-server, $manager_path install -y x11vnc"
		;;
	# Alpine's bwrap package name remains bubblewrap; Xvfb and x11vnc match.
	apk)
		commands="$manager_path add --no-cache bubblewrap, $manager_path add --no-cache xvfb, $manager_path add --no-cache x11vnc"
		;;
esac

# setcap normally lives in /usr/sbin, which is often absent from a regular
# user's PATH even though the binary is there and runnable via sudo.
setcap_path="$(command -v setcap 2>/dev/null || true)"
for candidate in /usr/sbin/setcap /sbin/setcap; do
	[ -n "$setcap_path" ] && break
	[ -x "$candidate" ] && setcap_path="$candidate"
done
if [ -z "$setcap_path" ]; then
	echo "[configure-sandbox-sudoers] ERROR: setcap not found (install libcap2-bin / libcap)" >&2
	exit 1
fi

# bundle-sandbox-tools.sh re-copies nsenter/bpftrace/frida-server from PATH on
# every ordinary `make build`, which drops any capability a prior `make
# install-sandbox-privileges` granted. It reapplies the grant automatically
# via `sudo -n` when its marker file is present; allow-list exactly those
# setcap invocations (for both tools/ and its dist/tools/ mirror, since an
# unprivileged `cp -a` cannot carry the security.capability xattr across) so
# that reapply succeeds without a password prompt on every rebuild.
cap_commands=""
for dir in "$PROJECT_ROOT/tools" "$PROJECT_ROOT/dist/tools"; do
	# Commas separate command specifications in sudoers. Escape the commas in
	# setcap's capability argument so visudo parses each invocation as one
	# tightly-scoped command rather than treating capability fragments as new
	# sudoers entries.
	cap_commands="$cap_commands, $setcap_path cap_sys_admin\\,cap_sys_ptrace\\,cap_sys_chroot+ep $dir/nsenter"
	cap_commands="$cap_commands, $setcap_path cap_bpf\\,cap_perfmon\\,cap_sys_admin+ep $dir/bpftrace"
	cap_commands="$cap_commands, $setcap_path cap_sys_ptrace+ep $dir/frida-server"
done

temp_file="$(mktemp)"
trap 'rm -f "$temp_file"' EXIT
printf '%s ALL=(root) NOPASSWD: %s%s\n' "$operator" "$commands" "$cap_commands" >"$temp_file"

sudo visudo -cf "$temp_file"
sudo install -o root -g root -m 0440 "$temp_file" /etc/sudoers.d/knirvengine-sandbox-deps
echo "[configure-sandbox-sudoers] installed restricted dependency + capability-reapply policy for $operator ($manager)"
