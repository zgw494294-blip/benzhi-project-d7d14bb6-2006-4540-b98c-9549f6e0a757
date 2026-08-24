# 古建木构修缮放行台

本项目面向古建筑勘察工程师、木作修缮方案编制人员和文物保护责任审核员，提供从木构件基线登记、病害观察、风险校核、方案审核、范围冻结到施工放行凭据签发的单进程 JSON HTTP API。档案写入使用 `expectedVersion` 和 `Idempotency-Key`，本地 JSON Lines 账本带审计哈希链并可从快照恢复。所有命令先在档案副本上完成业务校验，成功后才一次写入版本、快照和审计事件。

## 构建、运行与测试

```text
go build ./...
go run ./cmd/server -addr=127.0.0.1:19081
go test ./...
go run ./cmd/server -selfcheck -addr=127.0.0.1:19081
```

服务默认监听 `127.0.0.1:19081`，可使用 `-addr=127.0.0.1:<port>` 或 `PORT` 环境变量指定端口。数据目录可用 `TIMBER_DATA_DIR` 指定。

主要接口包括 `POST /api/v1/dossiers` 建档、`POST /api/v1/dossiers/{id}/components` 登记构件、`POST /api/v1/dossiers/{id}/observations` 提交观察、`POST /api/v1/dossiers/{id}/assess` 校核、`POST /api/v1/dossiers/{id}/plans` 提交方案、`POST /api/v1/dossiers/{id}/reviews` 审核、`POST /api/v1/dossiers/{id}/freeze` 冻结、`POST /api/v1/dossiers/{id}/release` 放行，以及档案详情、时间线、风险矩阵和凭据查询。

构件登记既接受原有单构件 JSON，也接受构件数组或 `{ "components": [...] }`；批次会统一检查编号唯一性、承重上级解析和拓扑环，任一失败都不改变档案。观察请求必须提供有效的 `observedAt`、位置、严重度和证据引用，证据会去空白、去重，历史按 `componentCode` 和 `revision` 排序并标记 `current`。方案动作必须锁定最新观察，并提供 `materialConstraint` 与 `acceptanceStandard`。

`GET /api/v1/dossiers/{id}/risk` 支持 `componentCode`、`severity`、`status` 和 `covered` 查询参数，响应同时给出构件总数、高风险构件数、覆盖数、开放阻断数和当前方案 revision。档案详情包含相邻方案差异和组合放行视图；凭据查询会重新核验冻结清单、凭据摘要和磁盘审计链。`FROZEN` 或 `RELEASED` 档案的业务写入统一返回 `FROZEN`。
