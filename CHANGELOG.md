# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.7]

- Add support for Victoria Logs (#34)
- Nix-support (#20)

## [0.1.6]

- Add 'f' key for full screen log viewing

## [0.1.2] Initial Release

### Added
- Initial release of Gonzo
- Real-time log stream analysis
- OTLP (OpenTelemetry) native support with HTTP and gRPC protocols
- Interactive k9s-inspired dashboard with 4 quadrants
- Word frequency analysis and charts
- Attribute tracking and visualization
- Severity level distribution
- Time-series log count visualization
- Modal detail view for individual logs
- AI-powered log analysis (OpenAI API compatible)
- OTLP HTTP receiver (port 4318) alongside gRPC receiver (port 4317)
- Improved severity extraction from OTLP records with proper fallback priority
- Green-to-light-blue gradient "Gonzo!" branding in footer
- Chat interface for AI interaction
- Regex filtering support
- Vim-style keyboard navigation
- Mouse support for selection
- Multiple log format detection (JSON, logfmt, plain text)
- Configurable update intervals and buffer sizes
- Test mode for non-TTY environments
- Cross-platform support (Linux, macOS, Windows)

### Technical
- Built with Bubble Tea framework
- Uses Lipgloss for styling
- Implements Bubbles components for UI elements
- Clean architecture with separated concerns
- Comprehensive error handling
- Memory-efficient frequency tracking
