# Gonzo Skins Documentation

Gonzo supports customizable color schemes (skins) that allow you to personalize the appearance of the terminal user interface. This document explains how to use, customize, and create your own skins.

## Table of Contents

- [Quick Start](#quick-start)
- [Available Skins](#available-skins)
- [Using Skins](#using-skins)
- [Getting Example Skins](#getting-example-skins)
- [Creating Custom Skins](#creating-custom-skins)
- [Color Reference](#color-reference)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)

## Quick Start

Be sure you download and place in the Gonzo config directory so Gonzo can find them.

```bash
# Use a built-in skin
gonzo --skin=dracula

# Use via environment variable
export GONZO_SKIN=github-light
gonzo -f application.log

# Use in config file
echo "skin: nord" >> ~/.config/gonzo/config.yml
```

## Available Skins

### Dark Themes 🌙
- **`default`** - Original Gonzo dark theme
- **`controltheory-dark`** - Original [ControlTheory](https://www.controltheory.com) based dark theme
- **`dracula`** - Purple-accented vampire theme
- **`gruvbox`** - Retro groove colors
- **`monokai`** - Warm high-contrast theme
- **`nord`** - Arctic blue palette
- **`solarized-dark`** - Precision colors for reduced eye strain

### Light Themes ☀️
- **`controltheory`** - Original [ControlTheory](https://www.controltheory.com) based light theme
- **`github-light`** - Clean GitHub-inspired light mode
- **`solarized-light`** - Precision light colors
- **`vs-code-light`** - Professional VS Code style
- **`spring`** - Fresh nature-inspired colors

## Using Skins

### Command Line
```bash
# Short form
gonzo -s dracula

# Long form
gonzo --skin=github-light

# Multiple ways to specify
gonzo --skin nord -f /var/log/app.log --follow
```

### Environment Variable
```bash
export GONZO_SKIN=monokai
gonzo -f application.log
```

### Configuration File
Add to `~/.config/gonzo/config.yml`:
```yaml
skin: solarized-dark
memory-size: 10000
update-interval: 2s
```

## Getting Example Skins

The example skins are available in the [Gonzo repository](https://github.com/control-theory/gonzo) under the `skins/` directory.

### Method 1: Download Individual Skins
```bash
# Create skins directory
mkdir -p ~/.config/gonzo/skins

# Download a specific skin (replace 'dracula' with desired skin)
curl -o ~/.config/gonzo/skins/dracula.yaml \
  https://raw.githubusercontent.com/control-theory/gonzo/main/skins/dracula.yaml
```

### Method 2: Clone Repository and Copy
```bash
# Clone the repository
git clone https://github.com/control-theory/gonzo.git
cd gonzo

# Copy all skins
cp skins/*.yaml ~/.config/gonzo/skins/
```

## Creating Custom Skins

### Basic Structure
Create a YAML file in `~/.config/gonzo/skins/` with the following structure:

```yaml
name: my-custom-skin
description: My awesome custom skin
author: Your Name
colors:
  # UI Component Colors
  primary: "#0066cc"           # Main accent color
  secondary: "#00cc66"         # Secondary accent
  background: "#ffffff"        # Main background
  surface: "#f8f9fa"          # Secondary background
  border: "#dee2e6"           # Default borders
  border_active: "#0066cc"     # Active section borders
  text: "#212529"             # Primary text
  text_secondary: "#6c757d"    # Muted text
  text_inverse: "#ffffff"      # Text on colored backgrounds
  
  # Chart and Data Colors
  chart_title: "#0066cc"       # Chart titles
  chart_bar: "#00cc66"        # Bar chart bars
  chart_accent: "#ff6600"      # Chart accents
  
  # Log Entry Colors
  log_timestamp: "#6c757d"     # Log timestamps
  log_message: "#212529"       # Log message text
  log_background: "#ffffff"    # Log entry background
  log_selected: "#e3f2fd"      # Selected log entry
  
  # Severity Level Colors
  severity_trace: "#adb5bd"    # TRACE level
  severity_debug: "#6c757d"    # DEBUG level
  severity_info: "#0066cc"     # INFO level
  severity_warn: "#ff9500"     # WARN level
  severity_error: "#dc3545"    # ERROR level
  severity_fatal: "#6f42c1"    # FATAL/CRITICAL level
  
  # Status Colors
  success: "#00cc66"          # Success states
  warning: "#ff9500"          # Warning states
  error: "#dc3545"            # Error states
  info: "#0066cc"             # Info states
  
  # Special Elements
  help: "#6c757d"             # Help text
  highlight: "#fff3cd"        # Search highlights
  disabled: "#adb5bd"         # Disabled elements
```

### Save and Use
```bash
# Save as ~/.config/gonzo/skins/my-custom-skin.yaml
gonzo --skin=my-custom-skin
```

## Color Reference

### UI Component Colors
| Color | Purpose | Example |
|-------|---------|---------|
| `primary` | Main accent color | Active borders, highlights |
| `secondary` | Secondary accent | Alternative highlights |
| `background` | Main background | Dashboard background |
| `surface` | Secondary background | Modal backgrounds, panels |
| `border` | Default borders | Section borders |
| `border_active` | Active borders | Selected section borders |
| `text` | Primary text | Main readable text |
| `text_secondary` | Muted text | Timestamps, help text |
| `text_inverse` | Inverse text | Text on colored backgrounds |

### Chart and Data Colors
| Color | Purpose |
|-------|---------|
| `chart_title` | Chart section titles |
| `chart_bar` | Bar chart bars |
| `chart_accent` | Chart accent elements |

### Log Entry Colors
| Color | Purpose |
|-------|---------|
| `log_timestamp` | Log entry timestamps |
| `log_message` | Log message text |
| `log_background` | Log entry background |
| `log_selected` | Selected log entry highlight |

### Severity Level Colors
| Color | Purpose | Log Levels |
|-------|---------|------------|
| `severity_trace` | Lowest priority | TRACE |
| `severity_debug` | Debug information | DEBUG |
| `severity_info` | Informational | INFO |
| `severity_warn` | Warnings | WARN, WARNING |
| `severity_error` | Errors | ERROR |
| `severity_fatal` | Critical errors | FATAL, CRITICAL |

### Status Colors
| Color | Purpose |
|-------|---------|
| `success` | Success indicators |
| `warning` | Warning indicators |
| `error` | Error indicators |
| `info` | Information indicators |

### Special Elements
| Color | Purpose |
|-------|---------|
| `help` | Help text and instructions |
| `highlight` | Search highlights, emphasis |
| `disabled` | Disabled UI elements |

## Examples

### Light Theme Example
```yaml
name: my-light-theme
description: A clean light theme
author: Me
colors:
  primary: "#0066cc"
  background: "#ffffff"
  text: "#212529"
  text_secondary: "#6c757d"
  severity_error: "#dc3545"
  severity_warn: "#fd7e14"
  severity_info: "#0066cc"
  # ... other colors
```

### Dark Theme Example
```yaml
name: my-dark-theme
description: A sleek dark theme
author: Me
colors:
  primary: "#66d9ef"
  background: "#272822"
  text: "#f8f8f2"
  text_secondary: "#75715e"
  severity_error: "#f92672"
  severity_warn: "#e6db74"
  severity_info: "#66d9ef"
  # ... other colors
```

### High Contrast Example
```yaml
name: high-contrast
description: High contrast for accessibility
author: Me
colors:
  primary: "#0000ff"
  background: "#000000"
  text: "#ffffff"
  text_secondary: "#cccccc"
  severity_error: "#ff0000"
  severity_warn: "#ffff00"
  severity_info: "#00ffff"
  # ... other colors
```

## Design Guidelines

### Light Themes
- Use **dark text** (`#212529`, `#24292e`) on **light backgrounds** (`#ffffff`, `#fafafa`)
- Ensure sufficient contrast ([WCAG AA: 4.5:1 minimum](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html))
- Use muted colors for secondary elements
- Avoid pure white backgrounds in favor of subtle grays

### Dark Themes
- Use **light text** (`#ffffff`, `#f8f8f2`) on **dark backgrounds** (`#282828`, `#2e3440`)
- Avoid pure black backgrounds; use dark grays instead
- Use vibrant colors for accents and highlights
- Ensure readability in low-light conditions

### Color Accessibility
- Test with color blindness simulators
- Ensure adequate contrast ratios
- Don't rely solely on color to convey information
- Consider users with visual impairments

## Troubleshooting

### Skin Not Loading
```bash
# Check if skin file exists
ls ~/.config/gonzo/skins/

# Verify YAML syntax
cat ~/.config/gonzo/skins/my-skin.yaml

# Test with default skin
gonzo --skin=default
```

### Missing Colors
If a color is not specified in your skin, Gonzo will use the default value. All colors are optional, but it's recommended to specify all for consistent appearance.

### Color Format
- Use hex colors: `#ff0000`, `#RGB`, `#RRGGBB`
- RGB values: `rgb(255, 0, 0)`
- Named colors: `red`, `blue` (limited support)

### Performance
- Skins are loaded once at startup
- No performance impact during runtime

## Contributing Skins

We welcome community-contributed skins! To submit a skin:

1. Create your skin following the guidelines above
2. Test it thoroughly with different log types
3. Submit a pull request to the [Gonzo repository](https://github.com/control-theory/gonzo)
4. Include screenshots if possible

## License

All skins in this repository are released under the same license as Gonzo. Community-contributed skins retain their original author attribution.