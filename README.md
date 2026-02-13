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

### Config Structure

```jsonc
{
  // Proto config mode: determines which .prototools files to use
  // "global" - Only load ~/.proto/.prototools
  // "local" - Only load ./.prototools in current directory
  // "upwards" - Load .prototools while traversing upwards, but do not load ~/.proto/.prototools (default)
  // "upwards-global" or "all" - Load .prototools while traversing upwards, and do load ~/.proto/.prototools
  // This field is optional - if omitted, defaults to "upwards"
  "config_mode": "upwards",

  // Tool-specific icon and color configuration
  // Use hex colors (e.g., "#61AFEF") or color names (e.g., "blue", "red", "green")
  // Icons use Nerd Font hex codes (e.g., "e76f", "e627")
  "tools": {
    "node": {
      "icon": "ed0d",
      "color": "green"
    },
    "go": {
      "icon": "e627",
      "color": "cyan"
    }
  },

  // Custom Go template for formatting output
  // Available variables: .Tool, .ToolIcon, .IsInstalled, .ResolvedVersion, .IsOutdated
  // Available functions: ne (not equal), fgColor, bgColor, reset
  "template": "{{.ToolIcon}} {{if .IsInstalled}}{{if ne .ResolvedVersion \"\"}}{{if .IsOutdated}}{{bgColor \"#8b6914\"}}{{else}}{{bgColor \"#1c5f2a\"}}{{end}} {{.ResolvedVersion}} {{reset}}{{end}}{{else}}{{bgColor \"red\"}} Missing {{reset}}{{end}}",

  // Cache configuration
  // TTL: Time-to-live for cached data in seconds (default: 300 = 5 minutes)
  // Set to 0 to disable caching, or increase for longer intervals
  "cache": {
    "ttl": 300
  }
}
```

### Configuring Proto Config Mode

To show all configured tools instead of just those in your current directory:

To show all configured tools including those in the global proto config:

```json
{
  "config_mode": "all"
}
```

Or to only show tools from the global proto config:

```json
{
  "config_mode": "global"
}
```

### Customizing Icons

Find Nerd Font icons and add their hex codes:

```json
{
  "tools": {
    "rust": {
      "icon": "e7a8",
      "color": "red"
    },
    "python": {
      "icon": "e73c",
      "color": "yellow"
    }
  }
}
```

### Custom Templates

The `template` field uses Go's template syntax:

```json
{
  "template": "{{if .IsInstalled}}✓ {{.ResolvedVersion}}{{else}}✗ Missing{{end}}"
}
```

**Available variables:**
- `.Tool` - Tool name (e.g., "node", "go")
- `.ToolIcon` - Formatted icon with ANSI color codes
- `.IsInstalled` - Boolean, true if tool is installed
- `.ResolvedVersion` - Version string (e.g., "24.13.1")
- `.IsOutdated` - Boolean, true if a newer version exists

**Available functions:**
- `ne(a, b)` - Returns true if a != b
- `fgColor("color")` - Apply foreground color (hex or name)
- `bgColor("color")` - Apply background color (hex or name)
- `reset()` - Reset all formatting

### Color Options

**Named colors:** `black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`, `default`

**Hex colors:** Any 6-digit hex code (e.g., `#FF0000`, `#61AFEF`)

## Cache

The tool caches Proto command output to improve performance:

- **Location:** `~/.cache/oh-my-posh/integrations/omp-prototools/cache.json`
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
