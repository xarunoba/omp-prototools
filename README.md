# omp-prototools

A custom integration tool that bridges [oh-my-posh](https://ohmyposh.dev) and [proto](https://moonrepo.dev/proto), allowing you to display your proto-managed tool versions directly in your shell prompt. 

## Installation

Install via `go install github.com/xarunoba/omp-prototools@latest`

### Requirements
- **proto** - [https://moonrepo.dev/proto](https://moonrepo.dev/proto)
- **oh-my-posh** - [https://ohmyposh.dev](https://ohmyposh.dev)
- **Go 1.25+** - For installation and building from source

## Usage

**For Bash/Zsh, add to `~/.bashrc` or `~/.zshrc`:**

```bash
# Run OMP_PROTOTOOLS on every prompt
_run_omp_prototools() {
    export OMP_PROTOTOOLS=$(omp-prototools)
}

# Inject to PROMPT_COMMAND
if [[ -z "$PROMPT_COMMAND" ]]; then
    PROMPT_COMMAND="_run_omp_prototools"
else
    PROMPT_COMMAND="$PROMPT_COMMAND;_run_omp_prototools"
fi

# Refresh omp-prototools cache when installing
proto() {
    command proto "$@"
    if [[ "$1" == "install" ]];then
        command omp-prototools --refresh --silent
    fi
}
```

**Then add this segment to your oh-my-posh config:**

```json
{
  "segments": [
    {
      "type": "env",
      "style": "plain",
      "template": "{{ .Env.OMP_PROTOTOOLS }}",
    }
  ]
}
```

### Command Line Options

```bash
# Display proto tool status
./omp-prototools

# Bypass cache and fetch fresh data
./omp-prototools --refresh

# Use custom config file
./omp-prototools --config /path/to/config.json

# Suppress output (useful for scripts/hooks)
./omp-prototools --silent
```

## Configuration

The tool automatically creates a default config file on first run at:

```
~/.cache/oh-my-posh/integrations/omp-prototools/config.jsonc
```

> **Note:** The config file contains detailed documentation for all configuration options. Open the generated config file to see complete instructions for templates, icons, colors, and cache settings.

### Customizing Icons

Find Nerd Font icons and add their hex codes. Colors support hex codes, named colors, or ANSI codes:

```json
{
  "tools": {
    "rust": {
      "icon": "e7a8",
      "color": "red"          // named color
    },
    "python": {
      "icon": "e73c",
      "color": "yellow"       // named color
    },
    "node": {
      "icon": "ed0d",
      "color": "#61AFEF"      // hex color
    },
    "go": {
      "icon": "e627",
      "color": "208"          // ANSI code (orange)
    }
  }
}
```

### Custom Templates

The `template` field uses Go's template syntax:

```json
{
  "template": "{{if .IsInstalled}}✓ {{if eq .ResolvedVersion .NewestVersion}}{{fgColor \"#1c5f2a\"}}{{else}}{{if .IsOutdated}}{{fgColor \"#8b6914\"}}{{else}}{{fgColor \"#1c5f2a\"}}{{end}}{{end}} {{.ResolvedVersion}} {{reset}}{{else}}✗ Missing{{end}}"
}
```

 **Available variables:**
 - `.Tool` - Tool name (e.g., "node", "go")
 - `.ToolIcon` - Formatted icon with ANSI color codes (falls back to tool name if not configured)
 - `.IsInstalled` - Boolean, true if tool is installed
 - `.ResolvedVersion` - Current installed version string (e.g., "24.13.1")
 - `.ConfigVersion` - Configured version constraint (e.g., "~22", "^1.20") - available for all tools
 - `.NewestVersion` - Newest version matching the constraint (e.g., "22.10.1") - available for all tools
 - `.LatestVersion` - Absolute latest version (e.g., "25.3.1") - available for all tools
 - `.IsLatest` - Boolean, true if current version is the newest matching the constraint
 - `.IsOutdated` - Boolean, true if a newer version exists

**Available functions:**
- `eq(a, b)` - Returns true if a == b
- `ne(a, b)` - Returns true if a != b
- `fgColor("color")` - Apply foreground color (hex code, name, or ANSI code)
- `bgColor("color")` - Apply background color (hex code, name, or ANSI code)
- `reset()` - Reset all formatting

### Color Options

**Named colors:** `black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`, `default`

**Hex colors:** Any 6-digit hex code (e.g., `#FF0000`, `#61AFEF`)

**ANSI codes:** Any valid ANSI color code (e.g., `33` for yellow, `208` for orange)

## Cache

The tool caches Proto command output to improve performance:

- **Location:** Same directory as config file, named `{config_name}.cache.json`
  - Default: `~/.cache/oh-my-posh/integrations/omp-prototools/config.cache.json`
  - With custom config: Uses same directory and base name as config file
- **Format:** JSONC-compatible (indented JSON)
- **Default TTL:** 300 seconds (5 minutes)
- **Configurable:** Via `cache.ttl` in config (set to 0 to disable)

Cache is updated when fresh data is fetched by omp-prototools. Use `--refresh` or run after proto operations to ensure cache is current.

## Default Tools

The default configuration includes icons for these popular tools:

- **bun** (JavaScript runtime)
- **deno** (JavaScript runtime)
- **go** (Go language)
- **moon** (Project management)
- **node** (JavaScript runtime)
- **npm** (Node package manager)
- **pnpm** (Fast, disk space efficient package manager)
- **poetry** (Python packaging)
- **python** (Python language)
- **ruby** (Ruby language)
- **rust** (Rust language)
- **uv** (Fast Python package installer)
- **yarn** (JavaScript package manager)

Add more tools by adding entries to the `tools` section in your config.

## License

MIT License - see LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Acknowledgments

- [oh-my-posh](https://ohmyposh.dev) - The cross-shell prompt theme engine
- [proto](https://moonrepo.dev/proto) - The tool manager this integrates with
- [Nerd Fonts](https://www.nerdfonts.com/) - Font glyphs used for icons
