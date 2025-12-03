#!/bin/bash

# CPU 性能分析 - 单参数路由
echo "=== CPU Profile: Zallocrout_Param ==="
go test -bench="^BenchmarkZallocrout_Param$" -benchtime=3s -cpuprofile=cpu_param.prof -memprofile=mem_param.prof
go tool pprof -top -lines cpu_param.prof | head -20
echo ""

# CPU 性能分析 - 20参数路由
echo "=== CPU Profile: Zallocrout_Param20 ==="
go test -bench="^BenchmarkZallocrout_Param20$" -benchtime=3s -cpuprofile=cpu_param20.prof -memprofile=mem_param20.prof
go tool pprof -top -lines cpu_param20.prof | head -20
echo ""

# CPU 性能分析 - GitHub All
echo "=== CPU Profile: Zallocrout_GithubAll ==="
go test -bench="^BenchmarkZallocrout_GithubAll$" -benchtime=3s -cpuprofile=cpu_github.prof -memprofile=mem_github.prof
go tool pprof -top -lines cpu_github.prof | head -20
echo ""

# 对比 Gin 的性能
echo "=== CPU Profile: Gin_Param (for comparison) ==="
go test -bench="^BenchmarkGin_Param$" -benchtime=3s -cpuprofile=cpu_gin_param.prof
go tool pprof -top -lines cpu_gin_param.prof | head -20
echo ""

echo "分析完成！可以使用以下命令查看详细分析："
echo "  go tool pprof -http=:8080 cpu_param.prof      # 查看 CPU 火焰图"
echo "  go tool pprof -http=:8080 mem_param.prof      # 查看内存分配"
echo "  go tool pprof -list=Match cpu_github.prof     # 查看 Match 函数详情"
