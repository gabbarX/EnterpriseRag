package types

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// YAML data structures for config/builtin_agents.yaml
// ---------------------------------------------------------------------------

// BuiltinAgentI18n holds the display name and description for a built-in agent.
type BuiltinAgentI18n struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// BuiltinAgentEntry is one entry in the builtin_agents list in YAML.
type BuiltinAgentEntry struct {
	ID        string                      `yaml:"id"`
	Avatar    string                      `yaml:"avatar"`
	IsBuiltin bool                        `yaml:"is_builtin"`
	I18n      map[string]BuiltinAgentI18n `yaml:"i18n"`
	Config    CustomAgentConfig           `yaml:"config"`
}

// builtinAgentsFile is the top-level YAML structure.
type builtinAgentsFile struct {
	BuiltinAgents []BuiltinAgentEntry `yaml:"builtin_agents"`
}

// ---------------------------------------------------------------------------
// Global registry (populated from YAML at startup)
// ---------------------------------------------------------------------------

var (
	builtinAgentEntries     map[string]*BuiltinAgentEntry // keyed by agent ID
	builtinAgentEntriesMu   sync.RWMutex
	builtinAgentEntriesOnce sync.Once
)

// LoadBuiltinAgentsConfig loads built-in agent definitions from the given
// config directory (e.g. "./config"). The file must be named "builtin_agents.yaml".
// This should be called once at startup, after config.LoadConfig determines
// the config directory.
//
// If the file does not exist, the function is a no-op and the hard-coded
// defaults in BuiltinAgentRegistry remain effective.
func LoadBuiltinAgentsConfig(configDir string) error {
	var loadErr error
	builtinAgentEntriesOnce.Do(func() {
		filePath := filepath.Join(configDir, "builtin_agents.yaml")
		data, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				// File not found – perfectly fine, keep using hard-coded defaults.
				return
			}
			loadErr = fmt.Errorf("read builtin_agents.yaml: %w", err)
			return
		}

		var file builtinAgentsFile
		if err := yaml.Unmarshal(data, &file); err != nil {
			loadErr = fmt.Errorf("parse builtin_agents.yaml: %w", err)
			return
		}

		builtinAgentEntriesMu.Lock()
		defer builtinAgentEntriesMu.Unlock()

		builtinAgentEntries = make(map[string]*BuiltinAgentEntry, len(file.BuiltinAgents))
		for i := range file.BuiltinAgents {
			entry := &file.BuiltinAgents[i]
			builtinAgentEntries[entry.ID] = entry
		}

		// Rebuild the BuiltinAgentRegistry so that IsBuiltinAgentID / GetBuiltinAgent
		// continue to work transparently.
		rebuildRegistryFromConfig()
	})
	return loadErr
}

// rebuildRegistryFromConfig replaces the BuiltinAgentRegistry entries with
// factory functions that read from the YAML-loaded config. Must be called
// while builtinAgentEntriesMu is held.
func rebuildRegistryFromConfig() {
	for id := range builtinAgentEntries {
		agentID := id // capture for closure
		BuiltinAgentRegistry[agentID] = func(tenantID uint64) *CustomAgent {
			return buildAgentFromEntry(agentID, tenantID)
		}
	}
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// GetBuiltinAgentWithContext returns a built-in agent with its display name and
// description taken from builtin_agents.yaml.
// Falls back to GetBuiltinAgent when no YAML config is loaded.
func GetBuiltinAgentWithContext(ctx context.Context, id string, tenantID uint64) *CustomAgent {
	builtinAgentEntriesMu.RLock()
	entry, ok := builtinAgentEntries[id]
	builtinAgentEntriesMu.RUnlock()

	if !ok || entry == nil {
		// No YAML entry — fall back to hard-coded factory.
		if factory, exists := BuiltinAgentRegistry[id]; exists {
			return factory(tenantID)
		}
		return nil
	}

	return buildAgentFromEntry(id, tenantID)
}

// ApplyBuiltinAgentLocalization overlays the name, description and avatar from
// builtin_agents.yaml onto a persisted built-in agent, so the YAML stays the
// single source of truth for built-in copy even after the rows were seeded.
// Tenant-specific Config and other DB fields are left untouched.
func ApplyBuiltinAgentLocalization(ctx context.Context, agent *CustomAgent) {
	if agent == nil {
		return
	}
	localized := GetBuiltinAgentWithContext(ctx, agent.ID, agent.TenantID)
	if localized == nil {
		return
	}
	if localized.Name != "" {
		agent.Name = localized.Name
	}
	if localized.Description != "" {
		agent.Description = localized.Description
	}
	agent.Avatar = localized.Avatar
}

var builtinAgentEntriesTestMu sync.Mutex

// OverrideBuiltinAgentEntriesForTest replaces the in-memory YAML registry.
// Tests in other packages use this to assert the display overlay; call the
// returned func to restore the previous map. Concurrent tests are serialized
// so they do not clobber each other's entries.
func OverrideBuiltinAgentEntriesForTest(entries map[string]*BuiltinAgentEntry) func() {
	builtinAgentEntriesTestMu.Lock()
	builtinAgentEntriesMu.Lock()
	prev := builtinAgentEntries
	builtinAgentEntries = entries
	builtinAgentEntriesMu.Unlock()
	return func() {
		builtinAgentEntriesMu.Lock()
		builtinAgentEntries = prev
		builtinAgentEntriesMu.Unlock()
		builtinAgentEntriesTestMu.Unlock()
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// buildAgentFromEntry constructs a *CustomAgent from a BuiltinAgentEntry.
func buildAgentFromEntry(id string, tenantID uint64) *CustomAgent {
	builtinAgentEntriesMu.RLock()
	entry, ok := builtinAgentEntries[id]
	builtinAgentEntriesMu.RUnlock()

	if !ok || entry == nil {
		return nil
	}

	display := resolveDisplay(entry.I18n)

	agent := &CustomAgent{
		ID:          entry.ID,
		Name:        display.Name,
		Description: display.Description,
		Avatar:      entry.Avatar,
		IsBuiltin:   entry.IsBuiltin,
		TenantID:    tenantID,
		Config:      entry.Config, // value copy
	}
	agent.EnsureDefaults()
	return agent
}

// resolveDisplay picks the display copy for a built-in agent, preferring the
// "default" entry and falling back to whatever entry the YAML provides.
func resolveDisplay(m map[string]BuiltinAgentI18n) BuiltinAgentI18n {
	if v, ok := m["default"]; ok {
		return v
	}
	for _, v := range m {
		return v
	}
	return BuiltinAgentI18n{}
}

// ResolveBuiltinAgentPromptRefs iterates over all builtin agent entries and
// resolves system_prompt_id / context_template_id references by calling the
// provided resolver function.  The resolver takes a template ID and returns
// the template content string (empty string if not found).
//
// This must be called after both LoadBuiltinAgentsConfig and prompt template
// loading have completed.
func ResolveBuiltinAgentPromptRefs(resolver func(id string) string) {
	builtinAgentEntriesMu.Lock()
	defer builtinAgentEntriesMu.Unlock()

	for _, entry := range builtinAgentEntries {
		if entry == nil {
			continue
		}
		// Resolve system_prompt_id → SystemPrompt
		if entry.Config.SystemPromptID != "" && entry.Config.SystemPrompt == "" {
			if content := resolver(entry.Config.SystemPromptID); content != "" {
				entry.Config.SystemPrompt = content
			} else {
				fmt.Printf("Warning: builtin agent %q references system_prompt_id %q but template not found\n",
					entry.ID, entry.Config.SystemPromptID)
			}
		}
		// Resolve context_template_id → ContextTemplate
		if entry.Config.ContextTemplateID != "" && entry.Config.ContextTemplate == "" {
			if content := resolver(entry.Config.ContextTemplateID); content != "" {
				entry.Config.ContextTemplate = content
			} else {
				fmt.Printf("Warning: builtin agent %q references context_template_id %q but template not found\n",
					entry.ID, entry.Config.ContextTemplateID)
			}
		}
	}
}
