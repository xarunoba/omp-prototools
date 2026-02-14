package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGetCacheFile(t *testing.T) {
	tests := []struct {
		name          string
		configPath    string
		wantCacheFile string
	}{
		{
			name:          "default config location",
			configPath:    "",
			wantCacheFile: "config.cache.jsonc",
		},
		{
			name:          "custom config with jsonc extension",
			configPath:    "/custom/path/my-config.jsonc",
			wantCacheFile: "my-config.cache.jsonc",
		},
		{
			name:          "custom config with json extension",
			configPath:    "/custom/path/my-config.json",
			wantCacheFile: "my-config.cache.json",
		},
		{
			name:          "custom config with no extension",
			configPath:    "/custom/path/config",
			wantCacheFile: "config.cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldConfigPath := configPath
			oldGetConfigFilePath := getConfigFilePath
			defer func() {
				configPath = oldConfigPath
				getConfigFilePath = oldGetConfigFilePath
			}()

			configPath = tt.configPath
			if tt.configPath == "" {
				tempDir := t.TempDir()
				configFile := filepath.Join(tempDir, "config.jsonc")
				os.WriteFile(configFile, []byte("{}"), 0644)
				getConfigFilePath = func() string { return configFile }
			} else {
				getConfigFilePath = func() string { return tt.configPath }
			}

			cacheFile := getCacheFile()
			if cacheFile == "" {
				t.Fatal("getCacheFile returned empty string")
			}

			if filepath.Base(cacheFile) != tt.wantCacheFile {
				t.Errorf("Expected %s, got %s", tt.wantCacheFile, filepath.Base(cacheFile))
			}

			if tt.configPath != "" {
				expectedDir := filepath.Dir(tt.configPath)
				actualDir := filepath.Dir(cacheFile)
				if actualDir != expectedDir {
					t.Errorf("Expected cache in %s, got %s", expectedDir, actualDir)
				}
			}
		})
	}
}

func TestReadCache(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() string
		wantErr bool
	}{
		{
			name: "existing cache",
			setup: func() string {
				cacheFile := filepath.Join(t.TempDir(), "cache.json")
				data := CachedData{
					Entries: map[string]DirectoryCacheData{
						"test-hash": {
							StatusData:   map[string]ToolStatus{"node": {IsInstalled: true}},
							OutdatedData: map[string]OutdatedStatus{"node": {IsOutdated: false}},
							Timestamp:    time.Now().Unix(),
						},
					},
				}
				jsonData, _ := json.Marshal(data)
				os.WriteFile(cacheFile, jsonData, 0644)
				return cacheFile
			},
			wantErr: false,
		},
		{
			name: "non-existent cache",
			setup: func() string {
				return filepath.Join(t.TempDir(), "nonexistent.json")
			},
			wantErr: true,
		},
		{
			name: "invalid json cache",
			setup: func() string {
				cacheFile := filepath.Join(t.TempDir(), "cache.json")
				os.WriteFile(cacheFile, []byte("{invalid json"), 0644)
				return cacheFile
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldGetCacheFile := getCacheFile
			defer func() { getCacheFile = oldGetCacheFile }()

			getCacheFile = func() string { return tt.setup() }

			data, err := readCache()
			if (err != nil) != tt.wantErr {
				t.Errorf("readCache() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && data.Entries == nil {
				t.Error("Expected Entries to be populated")
			}
		})
	}
}

func TestWriteCache(t *testing.T) {
	tests := []struct {
		name    string
		data    CachedData
		wantErr bool
	}{
		{
			name: "write cache successfully",
			data: CachedData{
				Entries: map[string]DirectoryCacheData{
					"test-hash": {
						StatusData:   map[string]ToolStatus{"node": {IsInstalled: true}},
						OutdatedData: map[string]OutdatedStatus{"node": {IsOutdated: false}},
						Timestamp:    time.Now().Unix(),
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			cacheFile := filepath.Join(tempDir, "cache.json")

			oldGetCacheFile := getCacheFile
			defer func() { getCacheFile = oldGetCacheFile }()
			getCacheFile = func() string { return cacheFile }

			if err := writeCache(tt.data); (err != nil) != tt.wantErr {
				t.Errorf("writeCache() error = %v, wantErr %v", err, tt.wantErr)
			}

			content, err := os.ReadFile(cacheFile)
			if err != nil {
				t.Fatalf("Failed to read cache file: %v", err)
			}

			if !strings.Contains(string(content), "  ") {
				t.Error("Cache file should be indented (JSONC-compatible format)")
			}

			if string(content)[0] != '{' || string(content)[1] != '\n' {
				t.Error("Cache file should start with formatted JSON object")
			}
		})
	}
}

func TestIsCacheEntryValid(t *testing.T) {
	now := time.Now().Unix()
	oneMinuteAgo := now - 60
	fiveMinutesAgo := now - 300
	tenMinutesAgo := now - 600

	tests := []struct {
		name       string
		cached     DirectoryCacheData
		ttlSeconds int
		wantValid  bool
	}{
		{
			name:       "valid cache within ttl",
			cached:     DirectoryCacheData{Timestamp: oneMinuteAgo},
			ttlSeconds: 300,
			wantValid:  true,
		},
		{
			name:       "cache exactly at ttl boundary",
			cached:     DirectoryCacheData{Timestamp: fiveMinutesAgo},
			ttlSeconds: 300,
			wantValid:  false,
		},
		{
			name:       "expired cache",
			cached:     DirectoryCacheData{Timestamp: tenMinutesAgo},
			ttlSeconds: 300,
			wantValid:  false,
		},
		{
			name:       "zero timestamp",
			cached:     DirectoryCacheData{Timestamp: 0},
			ttlSeconds: 300,
			wantValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCacheEntryValid(tt.cached, tt.ttlSeconds)
			if got != tt.wantValid {
				t.Errorf("isCacheEntryValid() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

func TestUpdateStatusCache(t *testing.T) {
	tests := []struct {
		name      string
		data      map[string]ToolStatus
		wantPanic bool
	}{
		{
			name: "update status data",
			data: map[string]ToolStatus{
				"go": {ResolvedVersion: "1.26.0", IsInstalled: true},
			},
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			cacheFile := filepath.Join(tempDir, "cache.json")

			oldGetCacheFile := getCacheFile
			defer func() { getCacheFile = oldGetCacheFile }()
			getCacheFile = func() string { return cacheFile }

			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic but none occurred")
					}
				}()
			}

			updateCache(tt.data, nil, "upwards")

			if !tt.wantPanic {
				readData, err := os.ReadFile(cacheFile)
				if err != nil {
					t.Fatalf("Failed to read cache file: %v", err)
				}

				var cached CachedData
				if err := json.Unmarshal(readData, &cached); err != nil {
					t.Fatalf("Failed to unmarshal cache data: %v", err)
				}

				if cached.Entries == nil {
					t.Error("Expected Entries to exist")
				}
			}
		})
	}
}

func TestUpdateOutdatedCache(t *testing.T) {
	tests := []struct {
		name      string
		data      map[string]OutdatedStatus
		wantPanic bool
	}{
		{
			name: "update outdated data",
			data: map[string]OutdatedStatus{
				"node": {IsOutdated: true},
			},
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			cacheFile := filepath.Join(tempDir, "cache.json")

			oldGetCacheFile := getCacheFile
			defer func() { getCacheFile = oldGetCacheFile }()
			getCacheFile = func() string { return cacheFile }

			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic but none occurred")
					}
				}()
			}

			updateCache(nil, tt.data, "upwards")

			if !tt.wantPanic {
				readData, err := os.ReadFile(cacheFile)
				if err != nil {
					t.Fatalf("Failed to read cache file: %v", err)
				}

				var cached CachedData
				if err := json.Unmarshal(readData, &cached); err != nil {
					t.Fatalf("Failed to unmarshal cache data: %v", err)
				}

				if cached.Entries == nil {
					t.Error("Expected Entries to exist")
				}
			}
		})
	}
}

func TestHexToANSI256(t *testing.T) {
	tests := []struct {
		name string
		hex  string
		want string
	}{
		{
			name: "red color",
			hex:  "#FF0000",
			want: "196",
		},
		{
			name: "green color",
			hex:  "#00FF00",
			want: "46",
		},
		{
			name: "blue color",
			hex:  "#0000FF",
			want: "21",
		},
		{
			name: "black (grayscale)",
			hex:  "#000000",
			want: "16",
		},
		{
			name: "white (grayscale)",
			hex:  "#FFFFFF",
			want: "231",
		},
		{
			name: "mid gray (grayscale)",
			hex:  "#808080",
			want: "244",
		},
		{
			name: "without hash prefix",
			hex:  "61AFEF",
			want: "74",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hexToANSI256(tt.hex)
			if got != tt.want {
				t.Errorf("hexToANSI256(%q) = %v, want %v", tt.hex, got, tt.want)
			}
		})
	}
}

func TestResolveColorName(t *testing.T) {
	tests := []struct {
		name  string
		color string
		want  string
	}{
		{
			name:  "black",
			color: "black",
			want:  ANSIBlack,
		},
		{
			name:  "red",
			color: "red",
			want:  ANSIRed,
		},
		{
			name:  "green",
			color: "green",
			want:  ANSIGreen,
		},
		{
			name:  "yellow",
			color: "yellow",
			want:  ANSIYellow,
		},
		{
			name:  "blue",
			color: "blue",
			want:  ANSIBlue,
		},
		{
			name:  "magenta",
			color: "magenta",
			want:  ANSIMagenta,
		},
		{
			name:  "cyan",
			color: "cyan",
			want:  ANSICyan,
		},
		{
			name:  "white",
			color: "white",
			want:  ANSIWhite,
		},
		{
			name:  "default",
			color: "default",
			want:  ANSIDefault,
		},
		{
			name:  "mixed case",
			color: "RED",
			want:  ANSIRed,
		},
		{
			name:  "unknown color",
			color: "unknown",
			want:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveColorName(tt.color)
			if got != tt.want {
				t.Errorf("resolveColorName(%q) = %v, want %v", tt.color, got, tt.want)
			}
		})
	}
}

func TestTemplateFgColor(t *testing.T) {
	tests := []struct {
		name  string
		color string
		want  string
	}{
		{
			name:  "hex color",
			color: "#FF0000",
			want:  "\x1b[38;5;196m",
		},
		{
			name:  "named color",
			color: "red",
			want:  "\x1b[31m",
		},
		{
			name:  "green",
			color: "green",
			want:  "\x1b[32m",
		},
		{
			name:  "unknown color",
			color: "unknown",
			want:  "\x1b[unknownm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := templateFgColor(tt.color)
			if got != tt.want {
				t.Errorf("templateFgColor(%q) = %q, want %q", tt.color, got, tt.want)
			}
		})
	}
}

func TestTemplateBgColor(t *testing.T) {
	tests := []struct {
		name  string
		color string
		want  string
	}{
		{
			name:  "hex color",
			color: "#FF0000",
			want:  "\x1b[48;5;196m",
		},
		{
			name:  "named color",
			color: "red",
			want:  "\x1b[431m",
		},
		{
			name:  "green",
			color: "green",
			want:  "\x1b[432m",
		},
		{
			name:  "unknown color",
			color: "unknown",
			want:  "\x1b[4unknownm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := templateBgColor(tt.color)
			if got != tt.want {
				t.Errorf("templateBgColor(%q) = %q, want %q", tt.color, got, tt.want)
			}
		})
	}
}

func TestDecodeUnicodeHex(t *testing.T) {
	tests := []struct {
		name string
		hex  string
		want string
	}{
		{
			name: "standard unicode",
			hex:  "\\u4e2d",
			want: "中",
		},
		{
			name: "icon e718 (node icon)",
			hex:  "\\ue718",
			want: "",
		},
		{
			name: "icon e76f (bun icon)",
			hex:  "\\ue76f",
			want: "",
		},
		{
			name: "without prefix",
			hex:  "4e2d",
			want: "中",
		},
		{
			name: "invalid hex",
			hex:  "\\uZZZZ",
			want: "",
		},
		{
			name: "empty string",
			hex:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeUnicodeHex(tt.hex)
			if got != tt.want {
				t.Errorf("decodeUnicodeHex(%q) = %q, want %q", tt.hex, got, tt.want)
			}
		})
	}
}

func TestDefaultCacheTTL(t *testing.T) {
	if defaultCacheTTL != 300 {
		t.Errorf("Expected defaultCacheTTL to be 300, got %d", defaultCacheTTL)
	}
}

func TestGetConfigMode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string defaults to upwards",
			input:    "",
			expected: "upwards",
		},
		{
			name:     "upwards stays as upwards",
			input:    "upwards",
			expected: "upwards",
		},
		{
			name:     "global stays as global",
			input:    "global",
			expected: "global",
		},
		{
			name:     "local stays as local",
			input:    "local",
			expected: "local",
		},
		{
			name:     "all stays as all",
			input:    "all",
			expected: "all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getConfigMode(tt.input)
			if got != tt.expected {
				t.Errorf("getConfigMode(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetConfigModeFlags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string returns nil",
			input:    "",
			expected: nil,
		},
		{
			name:     "upwards returns nil",
			input:    "upwards",
			expected: nil,
		},
		{
			name:     "global returns config-mode flag",
			input:    "global",
			expected: []string{"--config-mode", "global"},
		},
		{
			name:     "local returns config-mode flag",
			input:    "local",
			expected: []string{"--config-mode", "local"},
		},
		{
			name:     "all returns config-mode all",
			input:    "all",
			expected: []string{"--config-mode", "all"},
		},
		{
			name:     "upwards-global returns config-mode all",
			input:    "upwards-global",
			expected: []string{"--config-mode", "all"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getConfigModeFlags(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("getConfigModeFlags(%q) length = %d, want %d", tt.input, len(got), len(tt.expected))
			}
			if len(got) > 0 && (got[0] != tt.expected[0] || got[1] != tt.expected[1]) {
				t.Errorf("getConfigModeFlags(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetDefaultConfigContent(t *testing.T) {
	content := getDefaultConfigContent()

	tests := []struct {
		name  string
		check func() bool
	}{
		{
			name: "contains config_mode field",
			check: func() bool {
				return contains(content, `"config_mode":`)
			},
		},
		{
			name: "contains tools section",
			check: func() bool {
				return contains(content, `"tools":`)
			},
		},
		{
			name: "contains cache section",
			check: func() bool {
				return contains(content, `"cache":`)
			},
		},
		{
			name: "contains node tool",
			check: func() bool {
				return contains(content, `"node":`)
			},
		},
		{
			name: "contains go tool",
			check: func() bool {
				return contains(content, `"go":`)
			},
		},
		{
			name: "contains rust tool",
			check: func() bool {
				return contains(content, `"rust":`)
			},
		},
		{
			name: "contains comments",
			check: func() bool {
				return contains(content, "//")
			},
		},
		{
			name: "contains template",
			check: func() bool {
				return contains(content, `"template":`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.check() {
				t.Error("Check failed")
			}
		})
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.jsonc")

	err := createDefaultConfig(configFile)
	if err != nil {
		t.Fatalf("createDefaultConfig() error = %v", err)
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("Expected config file to exist")
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Config file is empty")
	}

	if !contains(string(data), `"config_mode":`) {
		t.Error("Config file does not contain config_mode field")
	}
}

func TestLoadJSONConfig(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() string
		wantTools int
		wantErr   bool
	}{
		{
			name: "load existing config",
			setup: func() string {
				configFile := filepath.Join(t.TempDir(), "config.json")
				config := ProtoConfig{
					Tools: map[string]IconConfig{
						"node": {Icon: "e718", Color: "green"},
						"go":   {Icon: "e627", Color: "cyan"},
					},
					Template: "test",
					Cache:    CacheConfig{TTL: 300},
				}
				jsonData, _ := json.Marshal(config)
				os.WriteFile(configFile, jsonData, 0644)
				return configFile
			},
			wantTools: 2,
			wantErr:   false,
		},
		{
			name: "create default config",
			setup: func() string {
				return filepath.Join(t.TempDir(), "config.json")
			},
			wantTools: 13,
			wantErr:   false,
		},
		{
			name: "invalid json",
			setup: func() string {
				configFile := filepath.Join(t.TempDir(), "config.json")
				os.WriteFile(configFile, []byte("{invalid json"), 0644)
				return configFile
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFile := tt.setup()

			config, err := loadJSONConfig(configFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadJSONConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(config.Tools) != tt.wantTools {
					t.Errorf("Expected %d tools, got %d", tt.wantTools, len(config.Tools))
				}
			}
		})
	}
}

func TestProtoInstalled(t *testing.T) {
	installed := protoInstalled()
	if !installed {
		t.Log("Proto not installed in test environment")
	}
}

func TestRunProtoCommand(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		mockOutput []byte
		mockError  error
		wantErr    bool
	}{
		{
			name:       "successful command",
			args:       []string{"status", "--json"},
			mockOutput: []byte(`{"node": {"is_installed": true, "resolved_version": "24.0.0"}}`),
			mockError:  nil,
			wantErr:    false,
		},
		{
			name:       "command fails",
			args:       []string{"nonexistent"},
			mockOutput: nil,
			mockError:  fmt.Errorf("command failed"),
			wantErr:    true,
		},
		{
			name:       "empty output",
			args:       []string{"outdated", "--json"},
			mockOutput: []byte(`{}`),
			mockError:  nil,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldRunProtoCommand := runProtoCommand
			defer func() { runProtoCommand = oldRunProtoCommand }()

			runProtoCommand = func(args []string) ([]byte, error) {
				if len(args) != len(tt.args) {
					t.Errorf("got %d args, want %d", len(args), len(tt.args))
				}
				return tt.mockOutput, tt.mockError
			}

			output, err := runProtoCommand(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("runProtoCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && string(output) != string(tt.mockOutput) {
				t.Errorf("runProtoCommand() = %s, want %s", string(output), string(tt.mockOutput))
			}
		})
	}
}

func TestGetProtoStatus_ProtoNotInstalled(t *testing.T) {
	oldProtoInstalled := protoInstalled
	defer func() { protoInstalled = oldProtoInstalled }()

	protoInstalled = func() bool { return false }

	output := getProtoStatus()

	if output != "" {
		t.Errorf("getProtoStatus() = %q, want empty", output)
	}
}

func TestGetProtoStatus_ConfigError(t *testing.T) {
	oldProtoInstalled := protoInstalled
	oldLoadConfig := loadConfig
	defer func() {
		protoInstalled = oldProtoInstalled
		loadConfig = oldLoadConfig
	}()

	protoInstalled = func() bool { return true }
	loadConfig = func() (ProtoConfig, error) {
		return ProtoConfig{}, fmt.Errorf("config error")
	}

	output := getProtoStatus()

	if output != "" {
		t.Errorf("getProtoStatus() = %q, want empty", output)
	}
}

func TestGetProtoStatus_ToolStatusError(t *testing.T) {
	oldProtoInstalled := protoInstalled
	oldLoadConfig := loadConfig
	oldGetToolStatus := getToolStatus
	defer func() {
		protoInstalled = oldProtoInstalled
		loadConfig = oldLoadConfig
		getToolStatus = oldGetToolStatus
	}()

	protoInstalled = func() bool { return true }
	loadConfig = func() (ProtoConfig, error) {
		return ProtoConfig{
			Tools:    map[string]IconConfig{},
			Template: "",
			Cache:    CacheConfig{TTL: 300},
		}, nil
	}
	getToolStatus = func(config ProtoConfig) (map[string]ToolStatus, error) {
		return nil, fmt.Errorf("status error")
	}

	output := getProtoStatus()

	if output != "" {
		t.Errorf("getProtoStatus() = %q, want empty", output)
	}
}

func TestGetProtoStatus_EmptyToolList(t *testing.T) {
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
			Tools:    map[string]IconConfig{},
			Template: "",
			Cache:    CacheConfig{TTL: 300},
		}, nil
	}
	getToolStatus = func(config ProtoConfig) (map[string]ToolStatus, error) {
		return map[string]ToolStatus{}, nil
	}
	getOutdatedStatus = func(config ProtoConfig) map[string]OutdatedStatus {
		return map[string]OutdatedStatus{}
	}
	formatOutput = func(tools map[string]ToolStatus, outdatedTools map[string]OutdatedStatus, config ProtoConfig) string {
		return "empty"
	}

	output := getProtoStatus()

	if output != "empty" {
		t.Errorf("getProtoStatus() = %q, want empty", output)
	}
}
