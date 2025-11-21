#!/bin/bash
set -e

echo "Running tests..."
go test -v -coverprofile=coverage.out $(go list ./... | grep -v '/examples/')

echo "Generating coverage report..."
go tool cover -html=coverage.out -o coverage.html

echo "Coverage report generated: coverage.html"
