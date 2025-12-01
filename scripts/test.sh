#!/bin/bash
set -e

echo "Running tests..."
go test -coverprofile=coverage.out ./pkg/...

echo "Analyzing coverage..."
python3 scripts/analyze_coverage.py

echo "Generating HTML report..."
go tool cover -html=coverage.out -o coverage.html

echo "Done! Coverage report: coverage.html"
