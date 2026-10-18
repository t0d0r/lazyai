#!/bin/sh
# Runs inside a make test-linux container as an ordinary user. /t holds the
# Linux test binaries and bin/lazyai built on the host. Fake agents are
# shell scripts named after real ones, so nothing is installed or called
# over the network.
set -eu

fail() { echo "FAIL: $*" >&2; exit 1; }
. /etc/os-release
echo "== $PRETTY_NAME, $(tmux -V), $(ps --version)"

for t in /t/*.test; do
	echo "== $(basename "$t")"
	(cd /tmp && "$t" -test.count=1) || fail "$(basename "$t")"
done

mkdir -p "$HOME/bin" /tmp/work
export PATH="$HOME/bin:$PATH"
# claude echoes what it is sent; codex keeps a CPU busy.
printf '#!/bin/sh\nwhile read -r l; do echo "got: $l"; done\n' >"$HOME/bin/claude"
printf '#!/bin/sh\nwhile :; do :; done\n' >"$HOME/bin/codex"
chmod +x "$HOME/bin/claude" "$HOME/bin/codex"
lazyai=/t/lazyai

echo "== launch"
$lazyai launch -d /tmp/work claude
$lazyai launch -d /tmp/work codex
sleep 1

echo "== list"
$lazyai list
json=$($lazyai list --json)
claude=$(echo "$json" | jq -e '.[] | select(.kind == "claude")') || fail "claude not listed"
codex=$(echo "$json" | jq -e '.[] | select(.kind == "codex")') || fail "codex not listed"
[ "$(echo "$claude" | jq -r .tmux_session)" = lazyai-claude-work ] || fail "claude tmux session: $claude"
[ "$(echo "$claude" | jq -r .cwd)" = /tmp/work ] || fail "claude cwd: $claude"
echo "$claude" | jq -r .tty | grep -q '^pts/' || fail "claude tty: $claude"
[ "$(echo "$claude" | jq -r .status)" = idle ] || fail "claude should be idle: $claude"
[ "$(echo "$codex" | jq -r .status)" = busy ] || fail "codex should be busy: $codex"
echo "$codex" | jq -e '.cpu > 50' >/dev/null || fail "codex cpu: $codex"

echo "== show, send"
$lazyai show claude-work >/dev/null || fail "show by session name"
$lazyai send claude-work 'hello $HOME' || fail send
sleep 0.5
tmux capture-pane -p -t =lazyai-claude-work: | grep -qF 'got: hello $HOME' || fail "send did not arrive"

echo "== tui"
tmux new-session -d -s ui -x 160 -y 40 "$lazyai"
sleep 3
screen=$(tmux capture-pane -p -t =ui:)
echo "$screen" | head -12
echo "$screen" | grep -q Agents || fail "dashboard not drawn"
echo "$screen" | grep -q codex || fail "dashboard lacks codex"
tmux send-keys -t =ui: q
sleep 0.5
! tmux has-session -t =ui 2>/dev/null || fail "q did not quit the dashboard"

echo "== stop"
pid=$(echo "$codex" | jq -r .pid)
$lazyai stop --yes "$pid" || fail stop
! kill -0 "$pid" 2>/dev/null || fail "codex still running"
$lazyai kill --yes claude-work || fail kill

echo "== ok: $PRETTY_NAME"
