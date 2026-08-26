#!/usr/bin/env bash
#
# Install a deliberately narrow sudoers policy for the one-time Bubblewrap
# runtime dependency setup. Routine sandbox tool attachment does not use sudo;
# it uses the capability-carrying bundled nsenter helper instead.
set -euo pipefail

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
		commands="$manager_path update, $manager_path install -y bubblewrap, $manager_path install -y xvfb, $manager_path install -y x11vnc"
		;;
	dnf|microdnf|yum)
		commands="$manager_path install -y bubblewrap, $manager_path install -y xorg-x11-server-Xvfb, $manager_path install -y x11vnc"
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

temp_file="$(mktemp)"
trap 'rm -f "$temp_file"' EXIT
printf '%s ALL=(root) NOPASSWD: %s\n' "$operator" "$commands" >"$temp_file"

sudo visudo -cf "$temp_file"
sudo install -o root -g root -m 0440 "$temp_file" /etc/sudoers.d/knirvengine-sandbox-deps
echo "[configure-sandbox-sudoers] installed restricted dependency policy for $operator ($manager)"
