package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/xarunoba/omp-prototools/jsonc"
)

const (
	ANSIBlack   = "30"
	ANSIRed     = "31"
	ANSIGreen   = "32"
	ANSIYellow  = "33"
	ANSIBlue    = "34"
	ANSIMagenta = "35"
	ANSICyan    = "36"
	ANSIWhite   = "37"
	ANSIDefault = "0"
)

const (
	defaultTemplate   = `{{.ToolIcon}} {{if .IsInstalled}}{{if eq .ResolvedVersion .NewestVersion}}{{fgColor "#1c5f2a"}}{{else}}{{if .IsOutdated}}{{fgColor "#8b6914"}}{{else}}{{fgColor "#1c5f2a"}}{{end}}{{end}} {{.ResolvedVersion}} {{reset}}{{end}}{{else}}{{fgColor "red"}} Missing {{reset}}{{end}}`
	defaultCacheTTL   = 300
	defaultConfigMode = "upwards"
	ResetColor        = "\x1b[0m"
)

func getConfigMode(configMode string) string {
	if configMode == "" {
		return defaultConfigMode
	}
	return configMode
}

func getConfigModeFlags(configMode string) []string {
	mode := getConfigMode(configMode)
	switch mode {
	case "global", "local":
		return []string{"--config-mode", mode}
	case "upwards-global", "all":
		return []string{"--config-mode", "all"}
	default:
		return nil
	}
}

var (
	forceRefresh     bool
	silentMode       bool
	configPath       string
	cachedConfig     ProtoConfig
	cachedConfigPath string
	cachedConfigMod  time.Time
)

func init() {
	flag.BoolVar(&forceRefresh, "refresh", false, "Bypass cache and fetch fresh data from proto")
	flag.BoolVar(&silentMode, "silent", false, "Suppress output (useful for hooks/caching)")
	flag.StringVar(&configPath, "config", "", "Path to custom config file (overrides default location)")
}

type ToolStatus struct {
	IsInstalled     bool   `json:"is_installed"`
	ConfigSource    string `json:"config_source,omitempty"`
	ConfigVersion   string `json:"config_version,omitempty"`
	ResolvedVersion string `json:"resolved_version,omitempty"`
	ProductDir      string `json:"product_dir,omitempty"`
}

type OutdatedStatus struct {
	IsLatest       bool   `json:"is_latest"`
	IsOutdated     bool   `json:"is_outdated"`
	ConfigSource   string `json:"config_source,omitempty"`
	ConfigVersion  string `json:"config_version,omitempty"`
	CurrentVersion string `json:"current_version,omitempty"`
	NewestVersion  string `json:"newest_version,omitempty"`
	LatestVersion  string `json:"latest_version,omitempty"`
}

type IconConfig struct {
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

type CacheConfig struct {
	TTL int `json:"ttl,omitempty"` // Cache TTL in seconds, default 300 (5 min)
}

type DirectoryCacheData struct {
	StatusData   map[string]ToolStatus     `json:"status"`
	OutdatedData map[string]OutdatedStatus `json:"outdated"`
	Timestamp    int64                     `json:"timestamp"`
}

type CachedData struct {
	Entries map[string]DirectoryCacheData `json:"entries"` // Keyed by directory hash
}

type CachedResult struct {
	StatusData   map[string]ToolStatus
	OutdatedData map[string]OutdatedStatus
}

type ProtoConfig struct {
	ConfigMode string                `json:"config_mode,omitempty"` // global, local, upwards (default), upwards-global
	Tools      map[string]IconConfig `json:"tools"`
	Template   string                `json:"template,omitempty"`
	Cache      CacheConfig           `json:"cache,omitzero"`
}

type TemplateData struct {
	Tool            string
	ToolIcon        string
	IsInstalled     bool
	ResolvedVersion string
	IsLatest        bool
	IsOutdated      bool
	ConfigVersion   string
	NewestVersion   string
	LatestVersion   string
}

var getDirectoryContext = func(configMode string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write([]byte(wd))
	normalizedMode := getConfigMode(configMode)
	h.Write([]byte(normalizedMode))

	dir := wd
	for {
		prototoolsPath := filepath.Join(dir, ".prototools")
		if info, err := os.Stat(prototoolsPath); err == nil && !info.IsDir() {
			data, err := os.ReadFile(prototoolsPath)
			if err == nil {
				h.Write(data)
			}
		}

		if dir == homeDir || dir == "/" {
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

var getConfigFilePath = func() string {
	if configPath != "" {
		return configPath
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	configDir := filepath.Join(cacheDir, "oh-my-posh", "integrations", "omp-prototools")
	configFile := filepath.Join(configDir, "config.jsonc")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		configFile = filepath.Join(configDir, "config.json")
	}
	return configFile
}

var getCacheFile = func() string {
	configFile := getConfigFilePath()
	if configFile == "" {
		return ""
	}
	configDir := filepath.Dir(configFile)
	configBase := filepath.Base(configFile)
	configName := strings.TrimSuffix(configBase, filepath.Ext(configBase))
	return filepath.Join(configDir, configName+".cache.json")
}

func readCache() (CachedData, error) {
	cacheFile := getCacheFile()
	if cacheFile == "" {
		return CachedData{}, fmt.Errorf("cannot determine cache directory")
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return CachedData{}, err
	}

	var cached CachedData
	if err := json.Unmarshal(data, &cached); err != nil {
		return CachedData{}, err
	}

	return cached, nil
}

func writeCache(cached CachedData) error {
	cacheFile := getCacheFile()
	if cacheFile == "" {
		return fmt.Errorf("cannot determine cache directory")
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

func isCacheValid(cached CachedData) bool {
	if cached.Entries == nil {
		return false
	}
	return true
}

func isCacheEntryValid(entry DirectoryCacheData, ttlSeconds int) bool {
	if entry.Timestamp == 0 {
		return false
	}
	elapsed := time.Since(time.Unix(entry.Timestamp, 0))
	return elapsed.Seconds() < float64(ttlSeconds)
}

func getCachedData(config ProtoConfig, configMode string) (CachedResult, bool) {
	if forceRefresh {
		return CachedResult{}, false
	}

	ttl := config.Cache.TTL
	if ttl == 0 {
		ttl = defaultCacheTTL
	}

	cached, err := readCache()
	if err != nil || !isCacheValid(cached) {
		return CachedResult{}, false
	}

	dirHash, err := getDirectoryContext(configMode)
	if err != nil {
		return CachedResult{}, false
	}

	entry, exists := cached.Entries[dirHash]
	if !exists || !isCacheEntryValid(entry, ttl) {
		return CachedResult{}, false
	}

	return CachedResult{
		StatusData:   entry.StatusData,
		OutdatedData: entry.OutdatedData,
	}, true
}

func main() {
	flag.Parse()
	output := getProtoStatus()
	if !silentMode {
		fmt.Print(output)
	}
}

func getProtoStatus() string {
	if !protoInstalled() {
		return ""
	}

	config, err := loadConfig()
	if err != nil {
		return ""
	}

	var (
		tools         map[string]ToolStatus
		outdatedTools map[string]OutdatedStatus
		toolsErr      error
	)

	type result struct {
		status   map[string]ToolStatus
		outdated map[string]OutdatedStatus
		err      error
	}

	resultChan := make(chan result, 1)

	cached, ok := getCachedData(config, config.ConfigMode)
	if ok {
		close(resultChan)
		tools = cached.StatusData
		outdatedTools = cached.OutdatedData
	} else {
		go func() {
			var r result

			type statusResult struct {
				data map[string]ToolStatus
				err  error
			}
			statusChan := make(chan statusResult, 1)
			go func() {
				data, err := getToolStatus(config)
				statusChan <- statusResult{data, err}
			}()

			type outdatedResult struct {
				data map[string]OutdatedStatus
			}
			outdatedChan := make(chan outdatedResult, 1)
			go func() {
				data := getOutdatedStatus(config)
				outdatedChan <- outdatedResult{data}
			}()

			sRes := <-statusChan
			r.status = sRes.data
			r.err = sRes.err

			oRes := <-outdatedChan
			r.outdated = oRes.data

			resultChan <- r
		}()

		r := <-resultChan
		tools = r.status
		outdatedTools = r.outdated
		toolsErr = r.err

		if toolsErr == nil && (len(tools) > 0 || len(outdatedTools) > 0) {
			updateCache(tools, outdatedTools, config.ConfigMode)
		}
	}

	if toolsErr != nil {
		return ""
	}

	if len(tools) == 0 {
		return ""
	}

	return formatOutput(tools, outdatedTools, config)
}

var protoInstalled = func() bool {
	_, err := exec.LookPath("proto")
	return err == nil
}

var loadConfig = func() (ProtoConfig, error) {
	// Determine config file path
	var configFile string
	if configPath != "" {
		configFile = configPath
	} else {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return ProtoConfig{}, err
		}

		configDir := filepath.Join(cacheDir, "oh-my-posh", "integrations", "omp-prototools")

		// Try .jsonc first, then fall back to .json
		configFile = filepath.Join(configDir, "config.jsonc")
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			configFile = filepath.Join(configDir, "config.json")
		}
	}

	// Check if we have a cached config that's still valid
	if len(cachedConfig.Tools) > 0 || cachedConfig.Template != "" {
		if cachedConfigPath == configFile {
			if info, err := os.Stat(configFile); err == nil {
				if info.ModTime().Equal(cachedConfigMod) || info.ModTime().Before(cachedConfigMod) {
					return cachedConfig, nil
				}
			}
		}
	}

	// Load fresh config
	config, err := loadJSONConfig(configFile)
	if err != nil {
		return ProtoConfig{}, err
	}

	// Cache the config
	if info, statErr := os.Stat(configFile); statErr == nil {
		cachedConfig = config
		cachedConfigPath = configFile
		cachedConfigMod = info.ModTime()
	}

	return config, nil
}

var getToolStatus = func(config ProtoConfig) (map[string]ToolStatus, error) {
	cached, ok := getCachedData(config, config.ConfigMode)
	if ok {
		if cached.StatusData != nil {
			return cached.StatusData, nil
		}
	}

	args := []string{"status", "--json"}
	if flags := getConfigModeFlags(config.ConfigMode); len(flags) > 0 {
		args = append(args, flags...)
	}

	output, err := runProtoCommand(args)
	if err != nil {
		return nil, err
	}

	var tools map[string]ToolStatus
	if err := json.Unmarshal(output, &tools); err != nil {
		return nil, err
	}

	return tools, nil
}

var getOutdatedStatus = func(config ProtoConfig) map[string]OutdatedStatus {
	cached, ok := getCachedData(config, config.ConfigMode)
	if ok {
		if cached.OutdatedData != nil {
			return cached.OutdatedData
		}
	}

	args := []string{"outdated", "--json"}
	if flags := getConfigModeFlags(config.ConfigMode); len(flags) > 0 {
		args = append(args, flags...)
	}

	output, err := runProtoCommand(args)
	if err != nil {
		return make(map[string]OutdatedStatus)
	}

	var tools map[string]OutdatedStatus
	if err := json.Unmarshal(output, &tools); err != nil {
		return make(map[string]OutdatedStatus)
	}

	return tools
}

func updateCache(statusData map[string]ToolStatus, outdatedData map[string]OutdatedStatus, configMode string) {
	cached, _ := readCache()
	if cached.Entries == nil {
		cached.Entries = make(map[string]DirectoryCacheData)
	}

	dirHash, err := getDirectoryContext(configMode)
	if err != nil {
		return
	}

	entry := DirectoryCacheData{
		StatusData:   statusData,
		OutdatedData: outdatedData,
		Timestamp:    time.Now().Unix(),
	}
	cached.Entries[dirHash] = entry
	writeCache(cached)
}

var runProtoCommand = func(args []string) ([]byte, error) {
	cmd := exec.Command("proto", args...)
	return cmd.Output()
}

var formatOutput = func(tools map[string]ToolStatus, outdatedTools map[string]OutdatedStatus, config ProtoConfig) string {
	var formatted strings.Builder
	tmplStr := config.Template
	if tmplStr == "" {
		tmplStr = defaultTemplate
	}

	funcMap := template.FuncMap{
		"eq":      func(a, b any) bool { return a == b },
		"ne":      func(a, b any) bool { return a != b },
		"fgColor": templateFgColor,
		"bgColor": templateBgColor,
		"reset":   func() string { return ResetColor },
	}

	tmpl, err := template.New("output").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return ""
	}

	// Sort tool names for consistent output order
	toolNames := make([]string, 0, len(tools))
	for tool := range tools {
		toolNames = append(toolNames, tool)
	}
	sort.Strings(toolNames)

	for _, tool := range toolNames {
		status := tools[tool]

		var display string
		if iconConfig, ok := config.Tools[tool]; ok {
			icon := decodeUnicodeHex(iconConfig.Icon)
			iconColor := formatColor(iconConfig.Color, true)
			display = fmt.Sprintf("%s%s%s", iconColor, icon, "\x1b[0m")
		} else {
			display = tool
		}

		var outdated *OutdatedStatus
		if out, exists := outdatedTools[tool]; exists {
			outdated = &out
		}

		var configVersion string
		var newestVersion string
		var latestVersion string

		configVersion = status.ConfigVersion
		newestVersion = status.ResolvedVersion
		latestVersion = status.ResolvedVersion

		if outdated != nil {
			if outdated.NewestVersion != "" {
				newestVersion = outdated.NewestVersion
			}
			if outdated.LatestVersion != "" {
				latestVersion = outdated.LatestVersion
			}
		}

		data := TemplateData{
			Tool:            tool,
			ToolIcon:        display,
			IsInstalled:     status.IsInstalled,
			ResolvedVersion: status.ResolvedVersion,
			IsLatest:        outdated != nil && outdated.IsLatest,
			IsOutdated:      outdated != nil && outdated.IsOutdated,
			ConfigVersion:   configVersion,
			NewestVersion: func() string {
				if newestVersion != "" {
					return newestVersion
				}
				return status.ResolvedVersion
			}(),
			LatestVersion: func() string {
				if latestVersion != "" {
					return latestVersion
				}
				return status.ResolvedVersion
			}(),
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			continue
		}

		formatted.WriteString(buf.String())
	}

	return strings.TrimRight(formatted.String(), " ")
}

func formatColor(color string, foreground bool) string {
	if strings.HasPrefix(color, "#") {
		ansiColor := hexToANSI256(color)
		code := 38
		if !foreground {
			code = 48
		}
		return fmt.Sprintf("\x1b[%d;5;%sm", code, ansiColor)
	}
	ansiCode := resolveColorName(color)
	if !foreground {
		return fmt.Sprintf("\x1b[4%sm", ansiCode)
	}
	return fmt.Sprintf("\x1b[%sm", ansiCode)
}

func templateFgColor(color string) string {
	return formatColor(color, true)
}

func templateBgColor(color string) string {
	return formatColor(color, false)
}

func resolveColorName(color string) string {
	colorMap := map[string]string{
		"black":   ANSIBlack,
		"red":     ANSIRed,
		"green":   ANSIGreen,
		"yellow":  ANSIYellow,
		"blue":    ANSIBlue,
		"magenta": ANSIMagenta,
		"cyan":    ANSICyan,
		"white":   ANSIWhite,
		"default": ANSIDefault,
	}
	if ansiCode, ok := colorMap[strings.ToLower(color)]; ok {
		return ansiCode
	}
	return color
}

func hexToANSI256(hex string) string {
	hex = strings.TrimPrefix(hex, "#")

	r, _ := strconv.ParseInt(hex[0:2], 16, 64)
	g, _ := strconv.ParseInt(hex[2:4], 16, 64)
	b, _ := strconv.ParseInt(hex[4:6], 16, 64)

	if r == g && g == b {
		if r < 8 {
			return "16"
		}
		if r > 248 {
			return "231"
		}
		return fmt.Sprintf("%d", int(((r-8)/10)+232))
	}

	ri := int((r * 5) / 255)
	gi := int((g * 5) / 255)
	bi := int((b * 5) / 255)

	return fmt.Sprintf("%d", 16+(ri*36)+(gi*6)+bi)
}

func getDefaultConfigContent() string {
	return `{
	// Proto config mode: determines which .prototools files to use
	// "global" - Only load ~/.proto/.prototools
	// "local" - Only load ./.prototools in current directory
	// "upwards" - Load .prototools while traversing upwards, but do not load ~/.proto/.prototools (default)
	// "upwards-global" or "all" - Load .prototools while traversing upwards, and do load ~/.proto/.prototools
	"config_mode": ` + fmt.Sprintf("%q", defaultConfigMode) + `,

 	// Custom Go template for formatting output
 	// Available variables: .Tool, .ToolIcon, .IsInstalled, .ResolvedVersion, .IsLatest, .IsOutdated
 	// ConfigVersion, NewestVersion, and LatestVersion are available for all tools
 	// - .ConfigVersion - Configured version constraint (e.g., "~22", "^1.20") from proto status
 	// - .NewestVersion - Newest version matching the constraint (e.g., "22.10.1") from proto outdated
 	// - .LatestVersion - Absolute latest version (e.g., "25.3.1") from proto outdated
 	// Available functions: eq (equal), ne (not equal), fgColor, bgColor, reset
	"template": ` + fmt.Sprintf("%q", defaultTemplate) + `,

	// Tool-specific icon and color configuration
	// Use hex colors (e.g., "#61AFEF") or color names (e.g., "blue", "red", "green")
	// Icons use Nerd Font hex codes (e.g., "e76f", "e627")
	"tools": {
		"bun": {
			"icon": "e76f",
			"color": "magenta"
		},
		"deno": {
			"icon": "e7c0",
			"color": "white"
		},
		"go": {
			"icon": "e627",
			"color": "cyan"
		},
		"moon": {
			"icon": "e38d",
			"color": "white"
		},
		"node": {
			"icon": "ed0d",
			"color": "green"
		},
		"npm": {
			"icon": "e71e",
			"color": "yellow"
		},
		"pnpm": {
			"icon": "e865",
			"color": "yellow"
		},
		"poetry": {
			"icon": "e867",
			"color": "cyan"
		},
		"python": {
			"icon": "e73c",
			"color": "yellow"
		},
		"ruby": {
			"icon": "e23e",
			"color": "red"
		},
		"rust": {
			"icon": "e7a8",
			"color": "red"
		},
		"uv": {
			"icon": "f0b02",
			"color": "magenta"
		},
		"yarn": {
			"icon": "e6a7",
			"color": "cyan"
		}
	},

	// Cache configuration
	// TTL: Time-to-live for cached data in seconds (default: ` + fmt.Sprintf("%d", defaultCacheTTL) + ` = 5 minutes)
	// Set to 0 to disable caching, or increase for longer intervals
	"cache": {
		"ttl": ` + fmt.Sprintf("%d", defaultCacheTTL) + `
	}
}`
}

func createDefaultConfig(configFile string) error {
	configDir := filepath.Dir(configFile)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(configFile, []byte(getDefaultConfigContent()), 0644)
}

func loadJSONConfig(configFile string) (ProtoConfig, error) {
	config := ProtoConfig{
		Tools: make(map[string]IconConfig),
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		// Config doesn't exist, create default config
		if os.IsNotExist(err) {
			if createErr := createDefaultConfig(configFile); createErr != nil {
				return config, fmt.Errorf("failed to create default config: %w", createErr)
			}
			data, err = os.ReadFile(configFile)
			if err != nil {
				return config, err
			}
		} else {
			return config, err
		}
	}

	// Convert JSONC to standard JSON
	jsonData := jsonc.ToJSON(data)

	if err := json.Unmarshal(jsonData, &config); err != nil {
		return config, err
	}

	return config, nil
}

func decodeUnicodeHex(hex string) string {
	hex = strings.TrimPrefix(hex, "\\u")

	val, err := strconv.ParseInt(hex, 16, 32)
	if err != nil {
		return ""
	}

	return string(rune(val))
}
