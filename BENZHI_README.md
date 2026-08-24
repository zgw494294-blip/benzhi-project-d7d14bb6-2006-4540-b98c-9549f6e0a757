# BENZHI_README

基于 Go 实现的古建木构修缮勘察与施工放行台 HTTP API 项目，一款后端服务，已完整实现古建木构修缮勘察与施工放行服务，以修缮档案串联构件基线、不可变观察修订、风险校核、方案退回闭环、范围冻结和不可变放行凭据，并以本地哈希链事件账本和原子快照支持恢复。

## 项目说明
- 项目：benzhi-project-d7d14bb6-2006-4540-b98c-9549f6e0a757
- 项目用途：已完整实现古建木构修缮勘察与施工放行服务，以修缮档案串联构件基线、不可变观察修订、风险校核、方案退回闭环、范围冻结和不可变放行凭据，并以本地哈希链事件账本和原子快照支持恢复。
- Go 工具链：`golang:1.22`
- 前端工具链：无

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...
cd /app && GOTOOLCHAIN=local go run ./cmd/server -selfcheck -addr=127.0.0.1:19081
cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-project-d7d14bb6-2006-4540-b98c-9549f6e0a757-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-project-d7d14bb6-2006-4540-b98c-9549f6e0a757-arm64 linux/arm64
docker run -it benzhi-project-d7d14bb6-2006-4540-b98c-9549f6e0a757-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/server -selfcheck -addr=127.0.0.1:19081`
