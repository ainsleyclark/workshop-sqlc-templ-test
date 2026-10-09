#!/bin/sh
# Sets up a fork for the workshop: builds the toolchain, checks for Claude Code
# and Python, and adds the upstream remote when given the workshop's owner/repo.
set -eu

upstream=${1:-}

# A clone of the workshop repository would only find out it can't push after an hour of work.
if [ -n "$upstream" ]; then
	case $(git remote get-url origin 2>/dev/null) in
	*"$upstream"*)
		echo "This is a clone of the workshop repository, not a fork of it."
		echo "Fork https://github.com/$upstream on GitHub, clone your fork, and run this there."
		exit 1
		;;
	esac
fi

# An older Go would quietly download the right one on the first build, which needs a network
# the room may not have. GOTOOLCHAIN=local reports the Go that's installed rather than switching.
if ! command -v go >/dev/null 2>&1; then
	echo "Go isn't installed. Install Go $(sed -n 's/^go //p' go.mod) from https://go.dev/dl/ and run this again."
	exit 1
fi
want=$(sed -n 's/^go //p' go.mod)
# go.mod's version is a minimum, so a newer Go is fine; only an older one downloads.
have=$(GOTOOLCHAIN=local go env GOVERSION)
have=${have%% *}
older=$(echo "$want ${have#go}" | awk '{
	split($1, w, "."); split($2, h, /[.a-z]/)
	for (i = 1; i <= 3; i++) if (h[i] + 0 != w[i] + 0) { print (h[i] + 0 < w[i] + 0); exit }
	print 0
}')
if [ "$older" = 1 ]; then
	echo "This workshop needs Go $want or newer, and you have ${have#go}."
	echo "Install Go $want from https://go.dev/dl/, open a new terminal, and run make setup again."
	exit 1
fi

log=$(mktemp)
trap 'rm -f "$log"' EXIT

# step runs a command with a spinner on a terminal, and one plain line otherwise.
step() {
	name=$1
	shift
	if [ ! -t 1 ]; then
		echo "  Building $name"
		"$@" >/dev/null
		return
	fi
	"$@" >"$log" 2>&1 &
	pid=$!
	i=0
	while kill -0 "$pid" 2>/dev/null; do
		case $((i % 4)) in
		0) c='|' ;; 1) c='/' ;; 2) c='-' ;; *) c='\' ;;
		esac
		printf '\r  %s Building %s' "$c" "$name"
		i=$((i + 1))
		sleep 0.1
	done
	if wait "$pid"; then
		printf '\r  \033[32m✓\033[0m %-40s\n' "$name"
	else
		printf '\r  \033[31m✗\033[0m %-40s\n' "$name"
		cat "$log"
		exit 1
	fi
}

printf '\n  Building the workshop toolchain. The first run takes a minute or so.\n\n'

# sqlc is built without cgo, so a Mac needs no C compiler (an Xcode prompt).
# templ is installed to ~/go/bin because templier and GoLand look for it there.
step sqlc env CGO_ENABLED=0 go tool sqlc version
step templ go install github.com/a-h/templ/cmd/templ
step templier go tool templier --version
# gopls is its own module: it only builds against the x/tools it was released with, and the
# app's dependencies need a newer one. Installed to ~/go/bin, where editors look for it.
step gopls sh -c 'cd tools/gopls && go install golang.org/x/tools/gopls'
step "the app" go build ./...

# Claude Code is only offered when someone is there to answer.
claude_docs=https://code.claude.com/docs/en/setup
if ! command -v claude >/dev/null 2>&1; then
	if [ -t 0 ] && [ -t 1 ]; then
		printf "\n  Claude Code isn't installed. You'll need it for the Theming block.\n"
		printf '  Install it now? [Y/n] '
		read -r answer || answer=n
		case $answer in
		[nN]*) echo "  Skipped. Install it before the Theming block: $claude_docs" ;;
		*)
			if curl -fsSL https://claude.ai/install.sh | bash; then
				command -v claude >/dev/null 2>&1 || [ ! -x "$HOME/.local/bin/claude" ] ||
					echo "  Claude Code is in ~/.local/bin. Add it to your PATH, or open a new terminal."
			else
				echo "  Couldn't install Claude Code just now. Try again later: $claude_docs"
			fi
			;;
		esac
	else
		echo "Claude Code isn't installed yet. You'll need it for the Theming block: $claude_docs"
	fi
fi

# Python usually needs sudo to install, so only name the command for this machine.
python_hint() {
	case $(uname -s) in
	Darwin)
		if command -v brew >/dev/null 2>&1; then echo "brew install python"; else echo "xcode-select --install"; fi
		;;
	Linux)
		distro=$(. /etc/os-release 2>/dev/null && echo "$ID ${ID_LIKE:-}")
		case $distro in
		*debian* | *ubuntu*) echo "sudo apt install python3" ;;
		*fedora* | *rhel*) echo "sudo dnf install python3" ;;
		*arch*) echo "sudo pacman -S python" ;;
		*) echo "https://www.python.org/downloads/" ;;
		esac
		;;
	MINGW* | MSYS* | CYGWIN*) echo "winget install Python.Python.3.12" ;;
	*) echo "https://www.python.org/downloads/" ;;
	esac
}
command -v python3 >/dev/null 2>&1 || command -v python >/dev/null 2>&1 ||
	echo "Python 3 isn't installed yet. The design-site skill's design search needs it: $(python_hint)"

[ -n "$upstream" ] || exit 0

git remote get-url upstream >/dev/null 2>&1 || git remote add upstream "https://github.com/$upstream.git"

# Silent on failure: make update fetches anyway, and a missing repo would prompt for a password.
GIT_TERMINAL_PROMPT=0 git fetch -q upstream 2>/dev/null || true

printf '\n  ==============================================\n'
printf '    Welcome to the workshop!\n'
printf '    Full-Stack Go with sqlc and templ\n'
printf '  ==============================================\n\n'
printf '  You are ready. Fill the database with go run . db seed, then make web serves the site on :8080.\n'
printf '  Have a look through ./architectures before you start.\n\n'
