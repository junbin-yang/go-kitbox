.PHONY: test lint fmt vet clean

test:
	go test -v -coverprofile=coverage.out ./pkg/... ./internal/...

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -f coverage.out
