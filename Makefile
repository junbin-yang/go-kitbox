.PHONY: test coverage lint fmt vet clean

test:
	@echo "Running tests..."
	@go test -coverprofile=coverage.out ./pkg/...
	@echo "\nAnalyzing coverage..."
	@python3 scripts/analyze_coverage.py
	@echo "\nGenerating HTML report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Done! Coverage report: coverage.html"

coverage:
	@python3 scripts/analyze_coverage.py

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -f coverage.out coverage.html
