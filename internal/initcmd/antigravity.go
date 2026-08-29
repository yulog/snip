package initcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/edouard-claude/snip/internal/config"
	"github.com/edouard-claude/snip/internal/hook"
)

func initAntigravity(snipBin, filterDir string) error {
	agyBase, err := config.AntigravityBaseDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	hookCommand := hook.QuoteBinFor(snipBin, runtime.GOOS) + " hook antigravity"
	hooksPath := filepath.Join(agyBase, "config", "hooks.json")
	if err := patchAntigravityHooks(hooksPath, hookCommand); err != nil {
		return fmt.Errorf("patch settings: %w", err)
	}

	fmt.Println("snip init complete:")
	fmt.Printf("  agent: antigravity\n")
	fmt.Printf("  hook: %s\n", hookCommand)
	fmt.Printf("  filters: %s\n", filterDir)
	fmt.Printf("  hooks: %s\n", hooksPath)
	return nil
}

func patchAntigravityHooks(path, hookCommand string) error {
	var settings map[string]any

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			settings = make(map[string]any)
		} else {
			return fmt.Errorf("read hooks: %w", err)
		}
	} else {
		// Backup (best-effort)
		backupPath := path + ".bak"
		_ = os.WriteFile(backupPath, data, 0644)

		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse hooks: %w", err)
		}
	}

	snipHookEntry := map[string]any{
		"type":    "command",
		"command": hookCommand,
	}

	snipMatcher := map[string]any{
		"matcher": "run_command",
		"hooks":   []any{snipHookEntry},
	}

	// Get or create snip-hooks section
	snipHooks, _ := settings["snip-hooks"].(map[string]any)
	if snipHooks == nil {
		snipHooks = make(map[string]any)
	}

	// Get existing PreToolUse array or create new one
	var preToolUse []any
	if existing, ok := snipHooks["PreToolUse"]; ok {
		if arr, ok := existing.([]any); ok {
			preToolUse = arr
		}
	}

	// Check if snip hook already exists (idempotent)
	found := false
	for i, entry := range preToolUse {
		// TODO: Windows .exe support
		if isSnipEntry(entry) {
			preToolUse[i] = snipMatcher // Update in place
			found = true
			break
		}
	}
	if !found {
		preToolUse = append(preToolUse, snipMatcher)
	}

	snipHooks["PreToolUse"] = preToolUse
	settings["snip-hooks"] = snipHooks

	// Ensure parent directory exists for fresh installations
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal hooks: %w", err)
	}

	return os.WriteFile(path, out, 0644)
}
