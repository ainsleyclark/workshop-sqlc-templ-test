// Command key saves the workshop's Anthropic API key for Claude Code, in
// .claude/settings.local.json under env, where any claude started in the
// fork picks it up. It is make key, written in Go so it runs on Windows too.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const settingsPath = ".claude/settings.local.json"

func main() {
	fmt.Fprint(os.Stdout, "Paste the workshop key from Slack, or press Enter to use your own Claude login: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		fmt.Fprintln(os.Stdout, "No key saved; Claude Code will use your own login.")
		return
	}
	key := clean(line)
	if key == "" {
		fmt.Fprintln(os.Stdout, "No key saved; Claude Code will use your own login.")
		return
	}

	existing, err := os.ReadFile(settingsPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	out, err := merge(existing, key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", settingsPath, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o750); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(settingsPath, out, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "Saved to %s. Start Claude Code here with: claude, then type /design-site\n", settingsPath)
}

// clean trims the pasted key of whitespace and the quotes a copy from Slack
// or a shell sometimes carries.
func clean(s string) string {
	return strings.Trim(strings.TrimSpace(s), `'"`)
}

// merge sets env.ANTHROPIC_API_KEY in the settings JSON, keeping every other
// setting as it was. A nil or empty existing means no file yet.
func merge(existing []byte, key string) ([]byte, error) {
	settings := map[string]any{}
	if len(strings.TrimSpace(string(existing))) > 0 {
		if err := json.Unmarshal(existing, &settings); err != nil {
			return nil, err
		}
	}
	env, _ := settings["env"].(map[string]any)
	if env == nil {
		env = map[string]any{}
	}
	env["ANTHROPIC_API_KEY"] = key
	settings["env"] = env

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}
