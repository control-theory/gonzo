# Gonzo Architecture Diagrams

These PlantUML diagrams illustrate the Gonzo experimental architecture and plugin system.

## Viewing the Diagrams

### Option 1: PlantUML Online Server
Visit http://www.plantuml.com/plantuml/uml/ and paste the content of any `.puml` file to render it.

### Option 2: VS Code Extension
Install the PlantUML extension in VS Code to preview diagrams directly.

### Option 3: Generate PNGs with PlantUML
```bash
# Install PlantUML
brew install plantuml

# Generate all PNGs
for file in *.puml; do
  plantuml -tpng "$file"
done
```

## Diagram Descriptions

1. **gonzo_architecture.puml** - Overall experimental architecture flow showing how plugins integrate with the processing pipeline

2. **gonzo_docker_plugin.puml** - Detailed Docker plugin flow showing container discovery, log tailing, and source indicator assignment

3. **gonzo_stdin_plugin.puml** - Explains why piped input shows "I0" - because it correctly identifies the source as stdin

4. **gonzo_loki_plugin.puml** - Loki plugin flow showing query execution and log streaming

5. **gonzo_problem_diagnosis.puml** - Diagnoses the current issue where Docker plugin shows "??" due to context cancellation

6. **gonzo_all_plugins.puml** - Maps all plugins to their source type indicators (D, L, V, O, F, I)

## Source Indicator Mapping

| Plugin | Indicator | Example | Description |
|--------|-----------|---------|-------------|
| Docker | D | D1, D2 | Docker containers |
| Loki | L | L1, L2 | Loki queries |
| VMLogs | V | V1, V2 | Victoria Logs sources |
| OTLP | O | O1 | OTLP receiver |
| Files | F | F1 | File reader |
| Stdin | I | I0 | Standard input (always 0) |

## Key Insights

- When you pipe Docker logs (`docker logs container | gonzo`), it shows "I0" because the source IS stdin
- To get "D1" indicators, use the Docker plugin directly: `--source='docker:{"follow":["*"]}'`
- Currently there's a bug where Docker plugin's context gets cancelled before container discovery completes
