#!/bin/bash

# Run OpenTelemetry Collector natively
# Assumes you have otelcol installed locally

echo "Starting OpenTelemetry Collector..."
echo "Configuration: otel-collector-native.yaml"
echo ""
echo "The collector will:"
echo "  - Accept OTLP logs on ports 4317 (gRPC) and 4318 (HTTP)"
echo "  - Forward logs to Loki at http://localhost:3100/otlp"
echo "  - Output logs to console for debugging"
echo ""

# Check if otelcol is installed
if ! command -v otelcol-contrib &> /dev/null; then
    echo "Error: otelcol is not installed!"
    echo ""
    echo "download from:"
    echo "  https://github.com/open-telemetry/opentelemetry-collector-releases/releases"
    echo ""
    exit 1
fi

# Run the collector
otelcol-contrib --config=otel-collector-native.yaml
