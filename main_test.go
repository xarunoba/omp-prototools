package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultTemplate(t *testing.T) {
	if defaultTemplate == "" {
		t.Error("Expected defaultTemplate to be non-empty")
	}

	if !contains(defaultTemplate, ".ToolIcon") {
		t.Error("Default template should contain ToolIcon variable")
	}

	if !contains(defaultTemplate, ".IsInstalled") {
		t.Error("Default template should contain IsInstalled variable")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestFormatOutput(t *testing.T) {
	config := ProtoConfig{
		Template: "{{.ToolIcon}} {{.ResolvedVersion}}",
		Tools: map[string]IconConfig{
			"node": {Icon: "\\ue718", Color: "green"},
			"go":   {Icon: "\\ue627", Color: "cyan"},
		},
	}

	tools := map[string]ToolStatus{
		"node": {ResolvedVersion: "24.0.0", IsInstalled: true},
		"go":   {ResolvedVersion: "1.26.0", IsInstalled: true},
	}

	outdated := map[string]OutdatedStatus{
		"node": {IsOutdated: true},
		"go":   {IsOutdated: false},
	}

	output := formatOutput(tools, outdated, config)

	if output == "" {
		t.Error("Expected non-empty output")
	}

	if !contains(output, "24.0.0") {
		t.Error("Expected node version in output")
	}

	if !contains(output, "1.26.0") {
		t.Error("Expected go version in output")
	}
}

func TestFormatOutputWithNewFields(t *testing.T) {
	config := ProtoConfig{
		Template: "{{.ToolIcon}} {{.ResolvedVersion}} {{.IsLatest}} {{.ConfigVersion}} {{.NewestVersion}} {{.LatestVersion}}",
		Tools: map[string]IconConfig{
			"node": {Icon: "\\ue718", Color: "green"},
			"go":   {Icon: "\\ue627", Color: "cyan"},
		},
	}

	tools := map[string]ToolStatus{
		"node": {ResolvedVersion: "24.0.0", IsInstalled: true},
		"go":   {ResolvedVersion: "1.26.0", IsInstalled: true},
	}

	outdated := map[string]OutdatedStatus{
		"node": {IsOutdated: true},
		"go":   {IsOutdated: false},
	}

	output := formatOutput(tools, outdated, config)

	if output == "" {
		t.Error("Expected non-empty output")
	}
}

func TestFormatOutputNoOutdatedData(t *testing.T) {
	config := ProtoConfig{
		Template: "{{.ToolIcon}} {{.ResolvedVersion}} {{.IsLatest}} {{.ConfigVersion}} {{.NewestVersion}} {{.LatestVersion}}",
		Tools: map[string]IconConfig{
			"node": {Icon: "\\ue718", Color: "green"},
			"go":   {Icon: "\\ue627", Color: "cyan"},
		},
	}

	tools := map[string]ToolStatus{
		"node": {ResolvedVersion: "24.0.0", IsInstalled: true},
		"go":   {ResolvedVersion: "1.26.0", IsInstalled: true},
	}

	outdated := map[string]OutdatedStatus{}

	output := formatOutput(tools, outdated, config)

	if output == "" {
		t.Error("Expected non-empty output")
	}
}

func TestFormatOutputTemplateWithEqFunction(t *testing.T) {
	config := ProtoConfig{
		Template: "{{if eq .IsLatest true}}Latest{{else}}Not Latest{{end}}",
		Tools: map[string]IconConfig{
			"node": {Icon: "\\ue718", Color: "green"},
			"go":   {Icon: "\\ue627", Color: "cyan"},
		},
	}

	tools := map[string]ToolStatus{
		"node": {ResolvedVersion: "24.0.0", IsInstalled: true},
		"go":   {ResolvedVersion: "1.26.0", IsInstalled: true},
	}

	outdated := map[string]OutdatedStatus{
		"node": {IsOutdated: true},
		"go":   {IsOutdated: false},
	}

	output := formatOutput(tools, outdated, config)

	if output == "" {
		t.Error("Expected non-empty output")
	}
}

func TestFormatOutputTemplateWithConfigVersion(t *testing.T) {
	config := ProtoConfig{
		Template: "Config Version: {{.ConfigVersion}}",
		Tools: map[string]IconConfig{
			"node": {Icon: "\\ue718", Color: "green"},
			"go":   {Icon: "\\ue627", Color: "cyan"},
		},
	}

	tools := map[string]ToolStatus{
		"node": {ResolvedVersion: "24.0.0", IsInstalled: true},
		"go":   {ResolvedVersion: "1.26.0", IsInstalled: true},
	}

	outdated := map[string]OutdatedStatus{
		"node": {IsOutdated: true},
		"go":   {IsOutdated: false},
	}

	output := formatOutput(tools, outdated, config)

	if output == "" {
		t.Error("Expected non-empty output")
	}
}

func TestFormatOutputLatestVersionOnly(t *testing.T) {
	config := ProtoConfig{
		Template: "{{.LatestVersion}}",
		Tools: map[string]IconConfig{
			"node": {Icon: "\\ue718", Color: "green"},
			"go":   {Icon: "\\ue627", Color: "cyan"},
		},
	}

	tools := map[string]ToolStatus{
		"node": {ResolvedVersion: "24.0.0", IsInstalled: true},
		"go":   {ResolvedVersion: "1.26.0", IsInstalled: true},
	}

	outdated := map[string]OutdatedStatus{
		"node": {
			IsOutdated:    true,
			LatestVersion: "25.0.0",
		},
		"go": {
			IsOutdated: false,
		},
	}

	output := formatOutput(tools, outdated, config)

	if output == "" {
		t.Error("Expected non-empty output")
	}
}

func TestFormatOutputOutdatedVersionOnly(t *testing.T) {
	config := ProtoConfig{
		Template: "{{.ConfigVersion}}",
		Tools: map[string]IconConfig{
			"node": {Icon: "\\ue718", Color: "green"},
			"go":   {Icon: "\\ue627", Color: "cyan"},
		},
	}

	tools := map[string]ToolStatus{
		"node": {ResolvedVersion: "24.0.0", IsInstalled: true, ConfigVersion: "24.0"},
		"go":   {ResolvedVersion: "1.26.0", IsInstalled: true},
	}

	outdated := map[string]OutdatedStatus{
		"node": {
			IsOutdated:    true,
			ConfigVersion: "24.0",
		},
		"go": {
			IsOutdated: false,
		},
	}

	output := formatOutput(tools, outdated, config)

	if output == "" {
		t.Error("Expected non-empty output")
	}
}

func TestFormatOutputIntegration(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name          string
		setupCache    func() string
		config        ProtoConfig
		forceRefresh  bool
		wantToolCount int
		wantVersion   string
		wantInstalled bool
		mockOutput    string
	}{
		{
			name: "valid status cache",
			setupCache: func() string {
				cacheFile := filepath.Join(t.TempDir(), "cache.jsonc")
				data := CachedData{
					Entries: map[string]DirectoryCacheData{
						"test-hash": {
							StatusData: map[string]ToolStatus{
								"node": {ResolvedVersion: "24.0.0", IsInstalled: true},
								"go":   {ResolvedVersion: "1.26.0", IsInstalled: true},
							},
							Timestamp: now,
						},
					},
				}
				jsonData, _ := json.Marshal(data)
				os.WriteFile(cacheFile, jsonData, 0644)
				return cacheFile
			},
			config:        ProtoConfig{Cache: CacheConfig{TTL: 300}},
			forceRefresh:  false,
			wantToolCount: 2,
			wantVersion:   "24.0.0",
			wantInstalled: true,
			mockOutput:    `{"node": {"is_installed": true, "resolved_version": "24.0.0"}, "go": {"is_installed": true, "resolved_version": "1.26.0"}}`,
		},
		{
			name: "cache miss - fetch from proto",
			setupCache: func() string {
				return filepath.Join(t.TempDir(), "cache.jsonc")
			},
			config: ProtoConfig{
				Cache:      CacheConfig{TTL: 300},
				ConfigMode: "all",
			},
			forceRefresh:  false,
			wantToolCount: 2,
			wantVersion:   "24.0.0",
			wantInstalled: true,
			mockOutput:    `{"node": {"is_installed": true, "resolved_version": "24.0.0"}, "go": {"is_installed": true, "resolved_version": "1.26.0"}}`,
		},
		{
			name: "force refresh - bypass cache",
			setupCache: func() string {
				cacheFile := filepath.Join(t.TempDir(), "cache.jsonc")
				data := CachedData{
					Entries: map[string]DirectoryCacheData{
						"test-hash": {
							StatusData: map[string]ToolStatus{
								"node": {ResolvedVersion: "old", IsInstalled: true},
							},
							Timestamp: now,
						},
					},
				}
				jsonData, _ := json.Marshal(data)
				os.WriteFile(cacheFile, jsonData, 0644)
				return cacheFile
			},
			config:        ProtoConfig{Cache: CacheConfig{TTL: 300}},
			forceRefresh:  true,
			wantToolCount: 1,
			wantVersion:   "24.0.0",
			wantInstalled: true,
			mockOutput:    `{"node": {"is_installed": true, "resolved_version": "24.0.0"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldGetCacheFile := getCacheFile
			oldGetDirectoryContext := getDirectoryContext
			oldRunProtoCommand := runProtoCommand
			oldForceRefresh := forceRefresh
			defer func() {
				getCacheFile = oldGetCacheFile
				getDirectoryContext = oldGetDirectoryContext
				runProtoCommand = oldRunProtoCommand
				forceRefresh = oldForceRefresh
			}()

			cacheFile := tt.setupCache()
			getCacheFile = func() string { return cacheFile }
			getDirectoryContext = func(configMode string) (string, error) { return "test-hash", nil }
			forceRefresh = tt.forceRefresh
			runProtoCommand = func(args []string) ([]byte, error) {
				return []byte(tt.mockOutput), nil
			}

			tools, err := getToolStatus(tt.config)
			if err != nil {
				t.Fatalf("getToolStatus() error = %v", err)
			}

			if len(tools) != tt.wantToolCount {
				t.Errorf("getToolStatus() got %d tools, want %d", len(tools), tt.wantToolCount)
			}

			if tt.wantVersion != "" {
				if node, ok := tools["node"]; ok {
					if node.ResolvedVersion != tt.wantVersion {
						t.Errorf("node version = %s, want %s", node.ResolvedVersion, tt.wantVersion)
					}
				}
			}

			if tt.wantInstalled {
				if node, ok := tools["node"]; ok {
					if !node.IsInstalled {
						t.Error("node should be installed")
					}
				}
			}

			updatedConfig := ProtoConfig{
				Template: "{{.ToolIcon}} {{.ResolvedVersion}}",
				Tools: map[string]IconConfig{
					"node": {Icon: "\\ue718", Color: "green"},
					"go":   {Icon: "\\ue627", Color: "cyan"},
				},
				Cache: CacheConfig{TTL: 300},
			}

			output := formatOutput(tools, map[string]OutdatedStatus{}, updatedConfig)

			if output == "" {
				t.Error("Expected non-empty output from formatOutput")
			}

			if tt.wantToolCount > 1 {
				if !contains(output, "1.26.0") {
					t.Error("Expected go version in output")
				}
			}
		})
	}
}

func TestGetProtoStatus_CompleteWorkflow(t *testing.T) {
	oldProtoInstalled := protoInstalled
	oldLoadConfig := loadConfig
	oldGetToolStatus := getToolStatus
	oldGetOutdatedStatus := getOutdatedStatus
	oldFormatOutput := formatOutput
	defer func() {
		protoInstalled = oldProtoInstalled
		loadConfig = oldLoadConfig
		getToolStatus = oldGetToolStatus
		getOutdatedStatus = oldGetOutdatedStatus
		formatOutput = oldFormatOutput
	}()

	protoInstalled = func() bool { return true }
	loadConfig = func() (ProtoConfig, error) {
		return ProtoConfig{
			Tools: map[string]IconConfig{
				"node": {Icon: "\\ue718", Color: "green"},
				"go":   {Icon: "\\ue627", Color: "cyan"},
			},
			Template: "{{.ToolIcon}} {{.ResolvedVersion}}",
			Cache:    CacheConfig{TTL: 300},
		}, nil
	}
	getToolStatus = func(config ProtoConfig) (map[string]ToolStatus, error) {
		return map[string]ToolStatus{
			"node": {ResolvedVersion: "24.0.0", IsInstalled: true},
			"go":   {ResolvedVersion: "1.26.0", IsInstalled: true},
		}, nil
	}
	getOutdatedStatus = func(config ProtoConfig) map[string]OutdatedStatus {
		return map[string]OutdatedStatus{
			"node": {IsOutdated: false},
			"go":   {IsOutdated: false},
		}
	}
	formatOutput = func(tools map[string]ToolStatus, outdatedTools map[string]OutdatedStatus, config ProtoConfig) string {
		var result string
		for _, toolName := range []string{"node", "go"} {
			if tool, ok := tools[toolName]; ok {
				if result != "" {
					result += " "
				}
				result += fmt.Sprintf("%s %s", toolName, tool.ResolvedVersion)
			}
		}
		return result
	}

	output := getProtoStatus()

	if output == "" {
		t.Error("Expected non-empty output")
	}

	if !contains(output, "node") {
		t.Error("Expected node in output")
	}

	if !contains(output, "24.0.0") {
		t.Error("Expected node version in output")
	}

	if !contains(output, "1.26.0") {
		t.Error("Expected go version in output")
	}
}
