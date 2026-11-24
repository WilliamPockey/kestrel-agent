# 新窗口交接文档（给下一个 AI）

> 用途：换会话、换模型、换电脑时，把这份文档和指定文件交给 AI，让它快速恢复上下文，不用你重新解释一遍。
> 这份文档是「给 AI 看的」，不是项目介绍。

---

## 1. 一分钟项目摘要

- **项目名**：Kestrel Agent（仓库名 `kestrel-agent`）
- **路径**：`D:\C\Projects\FeelFree\kestrel-agent`
- **module**：`kestrel`，Go 1.26
- **定位**：从零用 Go 实现一个可演示、可测试、可解释的 Agent 平台，作为 Go 后端 + AI 工程能力的作品集；不是复制 `nexus-agent`。
- **当前进度**：Phase 0 / Step 0.2 完成（配置、slog、中间件、request id、panic recovery、readiness、优雅退出）。
- **下一步**：Step 0.3 CI（Linux 跑 `go test -race`、`go vet`、`go build`、格式检查），然后 Phase 1 LLM Gateway。
- **不是玩具**：路线图包含高可用/高并发专项（多副本、Redis 限流/幂等/Streams、Postgres 连接池/慢查询、SSE 长连接、故障演练、容量模型）、eino adapter 对照、Go 版 StateGraph、RAG 评估、MCP、OTel/Prometheus。
- **明确不做**：不复制 `nexus-agent` 的代码/命名/目录；不早期上 Milvus/K8s/微服务；不追求功能堆砌。

---

## 2. 学习者背景与讲解要求

- **Python 熟，Go 只入门**：知道变量、基本语句；goroutine/channel/context 刚开始学。
- **代码和相关知识基础较弱**：讲解必须**全面、清晰、由浅入深**。
- 讲解方式要求：
  - 先讲「为什么」，再讲「怎么写」；
  - 多用 Python 对照；
  - 代码逐行解释，术语第一次出现要解释；
  - 不要只说「照抄这个」；
  - 一次只推进一个 Step，不要一次生成几千行；
  - 每步给：概念、最小可运行代码、验收命令、预期输出、练习；
  - 每步必须能 `go test`、`go vet`、`go build`；
  - 遇到报错先解释原因，再给修复。
- **面试导向**：项目不仅要能跑，还要能讲清楚取舍、故障场景、指标和高可用设计。

---

## 3. 必读文件（按顺序）

如果新 AI 能访问项目目录，让它按顺序读这些文件：

1. `docs/AI-HANDOFF.md`（本文件）
2. `docs/IMPLEMENTATION_PLAN.md`（路线图、每步验收、面试点、技术选型、HA 能力地图）
3. `docs/GO-BASICS.md`（Go 基础补课，重点第 9 节并发）
4. `docs/step-00-2.md`（当前 Step 的详细讲义）
5. `README.md`、`go.mod`
6. 当前代码：
   - `cmd/server/main.go`
   - `internal/config/config.go`
   - `internal/config/config_test.go`
   - `internal/observability/logger.go`
   - `internal/observability/logger_test.go`
   - `internal/httpapi/router.go`
   - `internal/httpapi/middleware.go`
   - `internal/httpapi/response.go`
   - `internal/httpapi/router_test.go`
   - `internal/httpapi/middleware_test.go`
   - `internal/app/app.go`
   - `internal/app/app_test.go`
7. 需要了解上一步：`docs/step-00.md`

---

## 4. 新窗口开场白（直接复制给 AI）

```text
你是我的 Go + 后端 + AI Agent 导师。请先不要写代码，按顺序读这些文件：

- docs/AI-HANDOFF.md
- docs/IMPLEMENTATION_PLAN.md
- docs/GO-BASICS.md
- docs/step-00-2.md
- README.md
- cmd/server/main.go
- internal/config/config.go
- internal/observability/logger.go
- internal/httpapi/router.go
- internal/httpapi/middleware.go
- internal/httpapi/response.go
- internal/app/app.go
- 对应的 *_test.go

读完后先用 10 行以内总结：项目目标、当前进度、代码结构、下一步。确认后我们从 Step 0.3（CI）继续。

我的背景：Python 熟，Go 只入门，goroutine/channel/context 刚开始学。讲解要全面、清晰、由浅入深，多用 Python 对照和逐行解释；一次只做一个 Step；每步给验收命令、预期输出和练习；不要一次性生成大量代码；不要复制 nexus-agent。

项目要求：最终要能讲高可用/高并发（多副本、Redis 限流/幂等/Streams、Postgres 连接池、SSE 长连接、故障演练），不是玩具；所有功能要有测试和可量化验收。新增依赖前先解释理由。

如果文件读不到，告诉我需要我贴哪几个文件，不要凭空猜代码。
```

---

## 5. 文件怎么给：三种情况

### 情况 A：新窗口能访问同一个项目目录（推荐）

不需要上传代码。直接把项目路径和上面的开场白发给 AI，让它自己读文件。

需要说明：

- 项目路径：`D:\C\Projects\FeelFree\kestrel-agent`
- 先读 `docs/AI-HANDOFF.md` 和 `docs/IMPLEMENTATION_PLAN.md`
- 当前做到 Step 0.2，下一步 Step 0.3
- 你的背景和讲解要求（开场白里已包含）

### 情况 B：新 AI 不能访问目录，但支持上传文件

最少上传这 4 个：

1. `docs/AI-HANDOFF.md`
2. `docs/IMPLEMENTATION_PLAN.md`
3. `docs/step-00-2.md`
4. `docs/GO-BASICS.md`

再上传当前代码（很小，建议全传）：

- `go.mod`
- `README.md`
- `cmd/server/main.go`
- `internal/config/*.go`
- `internal/observability/*.go`
- `internal/httpapi/*.go`
- `internal/app/*.go`

如果上传数量有限，优先顺序：

1. `AI-HANDOFF.md`
2. `IMPLEMENTATION_PLAN.md`
3. `step-00-2.md`
4. `go.mod` + `main.go`
5. 当前要改的包（例如下一 Step 的代码）

### 情况 C：只能粘贴文本

不要粘贴整个仓库。按这个顺序贴：

1. `AI-HANDOFF.md` 全文
2. `IMPLEMENTATION_PLAN.md` 的「目录结构 + 当前 Phase + 下一步」
3. 当前 Step 的讲义
4. 当前要改的文件（一次只贴一个包）
5. 报错全文（不要只贴最后一行）

---

## 6. 不要发给 AI 的东西

- `bin/`、`*.exe`、日志文件、临时文件
- `.env`、真实 API Key、密码、token
- `node_modules`、构建产物
- 整个 `nexus-agent` 原仓库（我们只把它当对照，不复制）
- `.commandcode/taste/`（系统管理的偏好文件，不要手动改）

---

## 7. 当前状态核验命令

新 AI 应该先让你跑这些，确认环境没变：

```bash
cd D:\C\Projects\FeelFree\kestrel-agent
go test ./...
go vet ./...
go build ./...
```

当前预期：

- `go test ./...` 全部通过
- `go vet ./...` 无输出
- `go build ./...` 无输出
- `go test -cover ./...` 大约：
  - `internal/config` 80.8%
  - `internal/httpapi` 89.1%
  - `internal/observability` 87.5%
  - `internal/app` 58.3%
- `go test -race ./...` 在 Windows 本地会因为没有 gcc 失败；Step 0.3 配 Linux CI 后跑。

运行验证：

```bash
go run ./cmd/server
# 另开终端
curl.exe -i http://localhost:8080/healthz
curl.exe -i http://localhost:8080/readyz
# Ctrl+C 后应看到 shutdown requested / server stopped
```

---

## 8. 当前重要的工程约定

- module 名是 `kestrel`；发布到 GitHub 前再改 `github.com/<你的ID>/kestrel-agent`，并同步替换 import。
- 目前**没有独立 git 仓库/提交**；项目在 `D:\C\Projects` 这个 git 仓库下面。不要擅自 `git init` 或 commit，先和用户确认项目位置。
- 先 stdlib，后框架；先接口，后实现；先 fake，后真实服务。
- 本地 SQLite/内存，生产 Postgres/Redis；用接口隔离。
- 手写 Agent runtime + eino adapter 对照，不做 eino-only。
- LangGraph 是 Python/JS 生态；我们在 Go 里实现同类 `StateGraph`，不引入 Python 服务。
- 测试以 Fake LLM 为主，真实模型只做可选 E2E。
- 事件流是 runtime 的唯一输出；HTTP/CLI/未来 WebSocket 都消费同一份事件。
- 服务保持无状态；会话、队列、限流计数、SSE 订阅外置到 Postgres/Redis。

---

## 9. 下一步（新 AI 应该从这里开始）

### Step 0.3 CI

- 新增 `.github/workflows/ci.yml`
- Linux 上跑：`gofmt` 检查、`go vet`、`go test -race ./...`、`go build ./...`
- 可选：`staticcheck` / `golangci-lint`
- 验收：push 后 CI 绿；故意写格式错误能被拦住。

### Phase 1 LLM Gateway

- Step 1.1 Chat 类型与 Provider 接口
- Step 1.2 非流式 Chat Completions
- Step 1.3 流式 SSE 与 tool_calls 分片
- Step 1.4 重试、限流、熔断、超时

详细验收见 `docs/IMPLEMENTATION_PLAN.md`。

---

## 10. 交接检查清单

新 AI 开始写代码前，确认它做到了：

- [ ] 读完了 `AI-HANDOFF.md` 和 `IMPLEMENTATION_PLAN.md`
- [ ] 用几句话复述了项目目标、当前进度、下一步
- [ ] 没有假设不存在的文件或依赖
- [ ] 没有复制 `nexus-agent` 的代码
- [ ] 先解释概念，再给代码
- [ ] 每步给验收命令和练习
- [ ] 新增依赖前解释理由
- [ ] 不擅自 commit / 改 git 历史
