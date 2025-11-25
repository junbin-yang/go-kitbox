.PHONY: run build test clean

# 运行应用
run:
	go run cmd/server/main.go

# 构建应用
build:
	go build -o bin/server cmd/server/main.go

# 运行测试
test:
	go test -v ./...

# 清理构建产物
clean:
	rm -rf bin/
