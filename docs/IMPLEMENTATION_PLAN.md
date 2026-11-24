# Kestrel Agent — 实施路线图（内部规划）

> 这份文档写给实现者，也写给未来的自己。每次开工前先读当前阶段；每次收工更新「进度」和「决策记录」。
>
> 项目代号 **Kestrel**（红隼），仓库名 `kestrel-agent`。刻意不用 `nexus-agent` 这个名字，也不复制它的代码、命名和目录结构。

---

## 0. 一句话定位

用 Go 从零实现一个**可演示、可测试、可解释**的 Agent 平台，作为 Go 后端 + AI 工程能力的载体。

**核心闭环（P0）**

```
HTTP/SSE → LLM streaming → tool calling → ReAct → SQLite 持久化
        → 认证 → 最小 RAG → 可观测 → 测试/CI/Docker → demo
```

**差异化（P1/P2，面试可深挖）**

- 手写 Agent runtime，再加一个 `eino` adapter 做对照，能讲清楚「框架帮我做了什么」。
- 实现 LangGraph 风格的 `StateGraph`：条件边、并行、checkpoint、interrupt/HITL、断点恢复。
- 混合检索：dense + sparse（BM25）+ RRF，加检索评估（recall@k / MRR / nDCG）。
- MCP client：JSON-RPC 2.0、stdio 与 Streamable HTTP、工具发现/调用。
- 评测集 + 回归门禁：不只「能跑」，而是「可度量」。
- 并发/背压/超时/熔断/幂等：每个点都能在大厂面试里展开。

**明确不做**

- 不做低代码平台，不做通用工作流 SaaS。
- 不追求功能数量，不早期引入 Milvus / Redis / K8s / 微服务。
- 不把 `nexus-agent` 的代码复制进来改名字。可以读它做对照，但提交历史必须是我们自己的。
- 不先学 React 再做后端。先把 Go 和 Agent 核心闭环打穿。

---

## 1. 学习目标（比功能更重要）

项目只是载体。完成后应该能做到：

1. 不查资料写出一个带超时、可取消、可优雅退出的 HTTP 服务。
2. 解释 goroutine、channel、select、context、WaitGroup、Mutex 的使用场景和坑。
3. 手写 OpenAI 兼容的流式解析，能处理 `tool_calls` 分片、`usage`、`reasoning_content`。
4. 设计一个 ReAct 循环，并说清楚：终止条件、迭代上限、token 预算、工具失败、背压、幂等。
5. 用 `database/sql` 写仓储层，解释索引、事务、N+1、分页和迁移。
6. 解释 RAG 的 chunk、embedding、ANN、混合检索、RRF、评估指标。
7. 解释状态图/工作流：条件边、并行、checkpoint、HITL、断点恢复、幂等。
8. 用 OTel/Prometheus 定位一次慢请求，能说清楚 trace / metric / log 的分工。
9. 给核心包写表驱动测试、httptest、fake LLM、race test、fuzz test、benchmark。
10. 在 3 分钟内讲清楚项目取舍，并能回答「如果流量涨 10 倍怎么办」。

> 面试提醒：项目不能替代 Go 基础、网络、MySQL/Redis、操作系统和算法。每天留固定时间刷基础题和手写代码。

---

## 2. 技术选型

| 领域 | 选择 | 理由 | 什么时候换/加 |
|---|---|---|---|
| HTTP | `net/http`（Go 1.22+ 方法路由） |  stdlib 足够，面试可讲底层；避免先学框架魔法 | 需要更强路由/中间件生态时再评估 chi/Hertz |
| 配置 | 环境变量 + struct 校验 | 12-factor，简单可测 | 配置复杂后再加 YAML |
| 日志 | `log/slog` | 标准库、结构化、可加 trace_id | 需要采样/多 sink 时再加 zap |
| 数据库 | 本地/测试 SQLite（`modernc.org/sqlite`，纯 Go）；生产 Postgres 16（`pgx/v5`） | 本地零运维；生产验证连接池、事务、索引、只读副本 | Phase 3 起同时保留 SQLite/Postgres 适配 |
| SQL | `database/sql` + 手写 SQL | 真正理解 SQL、索引、事务、慢查询 | 需要复杂 CRUD 再评估 sqlc/GORM |
| 迁移 | 内嵌 SQL + 简易 runner → goose/golang-migrate | 学事务、版本表、幂等、向后兼容迁移 | Phase 3 后期切生产迁移工具 |
| 缓存/协调 | Redis 7 | 分布式限流、幂等键、缓存、分布式锁、Streams 队列 | Phase 8–9 |
| 部署 | 多副本 + Nginx/HAProxy；K8s 可选 | 验证无状态、健康检查、优雅摘流、水平扩展 | Phase 9 |
| LLM | OpenAI 兼容协议 | DeepSeek/Ollama/vLLM 都能接 | 多供应商时加 adapter |
| 测试 LLM | Fake Provider | 无 API Key、可重复、CI 稳定 | 真实模型只做可选 E2E |
| RAG | 自写 chunk/BM25/cosine/RRF | 面试可讲原理，不依赖黑盒 | 数据量大时加 pgvector/Milvus |
| 状态图 | 自写 `internal/graph` | LangGraph 是 Python/JS；我们用 Go 实现同等思想 | 需要分布式执行再评估 Temporal |
| eino | `cloudwego/eino` adapter | 对照学习框架抽象，不把核心绑死 | 以当时官方 examples 为准，pin 版本 |
| MCP | 自写 JSON-RPC client | 理解协议，而不是只会调 SDK | 稳定后可换 mcp-go |
| 可观测 | `log/slog` + OTel + Prometheus | 大厂标配，能讲 SLI/SLO | 部署后加 Jaeger/Grafana |
| 前端 | CLI 优先，可选最小 Web | 先打穿后端；UI 不抢时间 | P2 再做 React/HTMX |
| CI | GitHub Actions | 免费、可展示 | 需要时加 release/docker |

**关键原则：先 stdlib，后框架；先接口，后实现；先 fake，后真实服务；先无状态，后有状态；先单机闭环，再多副本演练。**

---

## 3. 目标架构

```
                    ┌─────────────────────────────┐
   CLI / Web ──────▶│  HTTP API (net/http)        │
                    │  auth / rate limit / SSE    │
                    └──────────────┬──────────────┘
                                   │
                    ┌──────────────▼──────────────┐
                    │  Service Layer              │
                    │  chat / kb / workflow / eval│
                    └───────┬─────────┬───────────┘
                            │         │
          ┌─────────────────▼──┐  ┌───▼─────────────────┐
          │ Agent Runtime      │  │ Graph Runtime       │
          │ ReAct loop         │  │ StateGraph          │
          │ events / budget    │  │ checkpoint / HITL   │
          └───────┬───────┬────┘  └───┬─────────┬───────┘
                  │       │           │         │
          ┌───────▼──┐ ┌──▼───────┐ ┌─▼─────┐ ┌─▼──────┐
          │ LLM GW   │ │ Tool Exec│ │ Store │ │ MCP    │
          │ stream   │ │ registry │ │ SQL   │ │ client │
          │ retry    │ │ sandbox  │ │       │ │        │
          └───────┬──┘ └──┬───────┘ └───────┘ └────────┘
                  │       │
          ┌───────▼───────▼────────────────────────────┐
          │ RAG: loader → splitter → embed → hybrid    │
          │      dense + BM25 + RRF → rerank(optional) │
          └────────────────────────────────────────────┘
```

**事件流是主线**：Agent 不直接写 HTTP 响应，而是产出一串 `Event`；SSE、CLI、未来的 WebSocket 都消费同一份事件。这样 runtime 可测试、可复用。

**状态外置是水平扩展的前提**：HTTP 进程不保存会话、队列、限流计数和 SSE 订阅关系；这些放到 Postgres/Redis。这样加副本只需要加进程，不需要改代码。

---

## 3.5 高可用/高并发能力地图

> 我们不会假装有百万 QPS。目标是：**单机可演示，多副本可验证，故障可演练，指标可量化**。面试时能讲清楚「为什么这样设计、故障时会发生什么」。

| 面试主题 | 项目落点 | 怎么验证 |
|---|---|---|
| 无状态水平扩展 | 多副本 API + 负载均衡；会话/事件/队列外置 | compose 起 3 个副本，kill 一个，压测无 5xx |
| 优雅摘流 | readiness + SIGTERM + `Server.Shutdown` + 请求超时 | 滚动重启时观察在途请求完成 |
| 连接池与慢查询 | Postgres + pgx pool；`EXPLAIN ANALYZE`；索引 | 压测前后 p95、连接数、慢查询日志 |
| 缓存 | Redis cache-aside；TTL + jitter；singleflight 防击穿 | 缓存命中率、穿透/雪崩故障注入 |
| 分布式限流 | Redis 滑动窗口/令牌桶；per-user/per-tenant | 超过配额返回 429，多副本计数一致 |
| 幂等 | `Idempotency-Key` + DB 唯一约束 + Redis 去重 | 重复请求/并发重复请求只产生一次副作用 |
| 异步任务 | Redis Streams consumer group、pending、XAUTOCLAIM、DLQ | kill worker 后任务被重新领取，不丢不重 |
| 背压与过载保护 | 有界队列、semaphore、worker pool、load shedding | 超过容量时快速失败/降级，而不是 OOM |
| 熔断/重试/舱壁 | per-provider breaker、指数退避 + jitter、并发隔离 | LLM 超时/429 时不拖垮整个服务 |
| SSE 长连接 | 心跳、连接上限、`Last-Event-ID` 续传、Redis 扇出 | 压测连接数、断线重连、跨副本续传 |
| 可观测与 SLO | RED/USE 指标、p95/p99、trace、错误预算 | 一次慢请求能从 trace 定位到 LLM/DB |
| 故障演练 | Redis 挂、PG 慢、LLM 超时、worker 崩、副本被杀 | 演练记录 + 改进项 |
| 容量模型 | 每副本连接数、内存/SSE、队列深度、token 成本 | 给出 QPS/连接数/成本的估算和实测 |
| 一致性 | at-least-once、幂等消费、outbox（可选）、HITL | 重复投递/进程崩溃后的状态正确 |

**实现顺序**：Phase 0 先把无状态、超时、优雅退出做对；Phase 3 把状态放进数据库；Phase 8 做可观测和单机可靠性；Phase 9 专门做多副本、Redis、故障演练和容量测试。不要等到最后才补。

---

## 4. 目录结构（目标态）

```
kestrel-agent/
├── cmd/
│   ├── server/          # API 服务入口
│   ├── cli/             # 终端聊天，快速验证 runtime
│   ├── worker/          # 异步任务（文档解析/评测）
│   └── eval/            # 评测入口
├── internal/
│   ├── config/          # 环境变量、校验
│   ├── httpapi/         # 路由、中间件、handler、SSE
│   ├── auth/            # 密码哈希、JWT、刷新轮换
│   ├── domain/          # 领域模型（不依赖框架）
│   ├── store/           # repository 接口 + SQL 实现
│   ├── llm/             # provider 接口、OpenAI 兼容、fake、重试、熔断
│   ├── tool/            # 工具接口、注册表、执行器、内置工具、沙箱
│   ├── agent/           # ReAct runtime、事件、预算
│   ├── graph/           # StateGraph、checkpoint、interrupt
│   ├── rag/             # loader、splitter、embedding、vector、bm25、fusion、eval
│   ├── memory/          # 短期/长期记忆
│   ├── mcp/             # JSON-RPC、stdio、HTTP
│   ├── worker/          # 任务队列、重试、死信
│   └── observability/   # slog、OTel、Prometheus
├── migrations/          # SQL 迁移
├── eval/                # 数据集与基线
├── docs/                # 路线图、Step 讲义、ADR
├── deploy/              # Dockerfile、compose
├── scripts/             # 本地辅助脚本
├── .github/workflows/   # CI
├── Makefile
└── README.md
```

---

## 5. Go 概念地图（边做边学）

| Go 概念 | 在哪一步学 | 项目里的落点 |
|---|---|---|
| module / package / import / 导出规则 | Step 0 | `go.mod`、`internal/httpapi` |
| struct / method / 指针接收者 | Step 0–1 | `http.Server`、config、handler 依赖 |
| interface / 隐式实现 | Step 1–2 | `llm.Provider`、`tool.Tool`、`store.Repository` |
| error / `%w` / `errors.Is` / `errors.As` | Step 1–2 | 错误码、HTTP 错误响应、重试判断 |
| slice / map / 值拷贝 | Step 1–3 | 事件、消息、工具参数 |
| defer / panic / recover | Step 1–2 | 中间件恢复、工具执行保护 |
| goroutine / channel / select | Step 1–2 | 请求处理、流式事件、worker |
| context / 取消 / 超时 | Step 1–2 | LLM 调用、工具、请求链路 |
| sync / atomic | Step 2–3 | 并发工具执行、指标、缓存 |
| testing / httptest / table-driven | Step 0 起 | 每个包 |
| `go test -race` | Step 1 起 | CI 必跑 |
| `database/sql` / 事务 / 索引 | Step 3 | 会话、消息、运行记录 |
| `embed` | Step 3 | 内嵌迁移 SQL |
| generics | Step 4–5 | 通用分页/结果类型（谨慎使用） |
| fuzz / benchmark / pprof | Step 8 | SSE parser、chunker、RRF、热路径 |
| `log/slog` / OTel / Prometheus | Step 8 | 全链路 |

---

## 6. 分阶段计划

> 状态标记：`[ ]` 未开始，`[~]` 进行中，`[x]` 完成。
> 每个 Step 的产出必须能：编译、测试、解释、演示。

### Phase 0 — Walking Skeleton（先让服务跑起来）

**目标**：建立工程习惯，理解一个 Go 服务从 `main` 到 HTTP 响应的完整路径。

#### Step 0.1 项目骨架 + 健康检查 + 第一个测试
- 概念：module、package、`main`、`http.Handler`、`http.ServeMux`、JSON、`httptest`。
- 交付：`go.mod`、`cmd/server/main.go`、`internal/httpapi/router.go`、`router_test.go`、Makefile、`.gitignore`、README。
- 验收：`go test ./...` 通过；`go vet ./...` 通过；`curl /healthz` 返回 JSON；`POST /healthz` 返回 405。
- 面试点：为什么用 `http.Handler` 而不是全局函数？`ServeMux` 的方法路由怎么工作？

#### Step 0.2 配置 + 结构化日志 + 中间件 + 优雅退出
- 概念：struct、env 解析、`slog`、中间件链、`context`、`signal.NotifyContext`、`goroutine`、`channel`、`sync.WaitGroup`。
- 交付：`internal/config`、请求 ID、panic recovery、访问日志、`/readyz`、`Server.Shutdown`。
- 验收：Ctrl+C 后进程在超时内退出；日志带 request_id；测试覆盖非法配置；`-race` 通过。
- 面试点：为什么不能只 `ListenAndServe`？`Shutdown` 和 `Close` 的区别？如何避免 goroutine 泄漏？

#### Step 0.3 CI 与质量门禁
- 概念：GitHub Actions、`gofmt`、`go vet`、`staticcheck`/`golangci-lint`、coverage。
- 交付：`.github/workflows/ci.yml`；README 增加徽章/命令。
- 验收：push 后 CI 绿；故意写一个格式错误能被拦住。
- 面试点：为什么 CI 要跑 `-race`？覆盖率应该看什么？

---

### Phase 1 — LLM Gateway（把模型接进来）

**目标**：不依赖任何 Agent 框架，手写一个可靠的 OpenAI 兼容客户端。

#### Step 1.1 Chat 类型与 Provider 接口
- 概念：struct、interface、JSON tag、`omitempty`、错误包装。
- 交付：`llm.Message`、`ChatRequest/Response`、`Provider` 接口、Fake Provider。
- 验收：表驱动测试覆盖消息序列化；fake 能按脚本返回。
- 面试点：为什么先定义接口再写实现？如何让测试不依赖网络？

#### Step 1.2 非流式 Chat Completions
- 概念：`http.Client`、`context`、超时、状态码处理、错误体解析。
- 交付：OpenAI 兼容 provider（`/v1/chat/completions`）、配置 base_url/api_key/model。
- 验收：`httptest.Server` 模拟成功/401/429/500；错误可被 `errors.Is/As` 识别。
- 面试点：超时应该设在哪几层？重试哪些错误、不重试哪些？

#### Step 1.3 流式 SSE 与 tool_calls 分片
- 概念：`bufio.Scanner`、SSE 格式、`chan`、`select`、context 取消、增量 JSON。
- 交付：`ChatStream` 返回 `<-chan StreamEvent`；正确拼接 `tool_calls` 的 index/name/arguments 分片；读取 `usage`。
- 验收：golden SSE 文件测试；客户端取消后服务端 goroutine 能退出；`-race` 通过。
- 面试点：SSE 和 WebSocket 的取舍？如何处理慢消费者/背压？为什么事件 channel 要有缓冲？

#### Step 1.4 可靠性：重试、限流、熔断、超时
- 概念：指数退避 + jitter、令牌桶、`context` 截止时间、状态机。
- 交付：`retry`、`rate limiter`、`circuit breaker`、`stream first-token timeout`。
- 验收：用 fake clock/可注入 sleep 测试退避；熔断半开恢复测试。
- 面试点：重试放大雪崩怎么办？幂等键怎么设计？

---

### Phase 2 — Tools + ReAct Agent（核心）

**目标**：实现一个能真正调用工具、可取消、可观测的 ReAct 循环。

#### Step 2.1 工具接口、注册表、执行器
- 概念：interface、JSON Schema、`json.RawMessage`、panic recovery、超时。
- 交付：`tool.Tool`、`Registry`、`Executor`；内置 `calculator`、`now`、`http_get`、`read_file`。
- 验收：未知工具/坏参数/超时/panic 都返回结构化错误；`read_file` 防路径穿越；`http_get` 防 SSRF。
- 面试点：工具参数为什么用 JSON Schema？如何防止 prompt injection 诱导危险工具？

#### Step 2.2 ReAct 循环与事件流
- 概念：状态机、循环终止、token 预算、事件、`chan`、context。
- 交付：`agent.Runtime`，输入消息/工具/模型，输出事件流；max iterations、step/run timeout、token budget。
- 验收：fake LLM 脚本「工具→工具→回答」；无限工具调用会被截断；取消能停止工具。
- 面试点：ReAct vs Plan-and-Execute vs Router？如何避免死循环？工具错误要不要回填给模型？

#### Step 2.3 SSE 对话接口 + CLI
- 概念：POST + SSE、事件编码、心跳、客户端断开。
- 交付：`POST /v1/chat` 流式返回；`cmd/cli` 可交互；事件含 thought/action/observation/answer。
- 验收：curl 能看到分块输出；断开后服务端停止；测试不依赖真实模型。
- 面试点：POST SSE 与 EventSource 的差异？如何做断线重连/续传？

---

### Phase 3 — 持久化、认证、API

**目标**：从「能跑」变成「有状态、可登录、可回放」。

#### Step 3.1 Schema 与迁移
- 概念：SQL、主键/外键、索引、事务、`embed`。
- 交付：users / agents / conversations / messages / runs / run_events 表；迁移 runner。
- 验收：空库一键迁移；重复执行幂等；测试用临时 SQLite。
- 面试点：为什么消息表要按 conversation_id + id 建索引？软删除怎么设计？

#### Step 3.2 Repository 层
- 概念：接口、依赖注入、`database/sql`、`sql.Null*`、分页。
- 交付：`store.Repository`；SQLite 实现；内存 fake 供 handler 测试。
- 验收：CRUD、分页、事务回滚测试；无 N+1。
- 面试点：Repository 是不是过度设计？什么时候直接写 SQL 更好？

#### Step 3.3 认证与安全
- 概念：Argon2id/bcrypt、JWT、access/refresh、轮换、中间件、常量时间比较。
- 交付：注册/登录/刷新/登出；refresh token 哈希入库；请求体大小限制。
- 验收：过期/篡改/重放测试；密码不落日志；生产配置拒绝默认密钥。
- 面试点：JWT 无法主动失效怎么办？refresh rotation 如何防重放？

#### Step 3.4 REST API 与错误规范
- 概念：REST、状态码、错误 envelope、request id、幂等键、分页。
- 交付：会话/消息/运行记录 API；统一错误码；OpenAPI 或 API.md。
- 验收：httptest 覆盖 200/400/401/403/404/409/429/500；错误响应稳定。
- 面试点：如何设计幂等创建接口？分页用 offset 还是 cursor？

---

### Phase 4 — RAG（检索增强）

**目标**：从文档到可评估的混合检索。

#### Step 4.1 文档加载与分块
- 概念：文件类型、UTF-8、递归切分、overlap、边界条件。
- 交付：txt/md/pdf loader；chunk size/overlap 可配；保留元数据。
- 验收：空文件/超长行/中文/英文/代码块测试；chunk 不丢字符。
- 面试点：固定长度 vs 语义分块？overlap 的代价？

#### Step 4.2 Embedding 与向量存储
- 概念：向量、余弦相似度、批量请求、维度校验、失败重试。
- 交付：Embedding 接口 + fake（确定性）+ OpenAI 兼容实现；内存 vector store。
- 验收：相似度排序正确；维度不一致报错；批量失败可恢复。
- 面试点：ANN 索引（HNSW/IVF）怎么选？归一化后点积=余弦？

#### Step 4.3 混合检索与 RRF
- 概念：BM25、倒排索引、分数归一化、RRF、top-k、过滤。
- 交付：BM25（自写）+ dense + RRF；元数据过滤；检索测试台。
- 验收：黄金集 recall@k / MRR / nDCG 可计算；RRF 排序测试。
- 面试点：为什么 hybrid 通常优于纯向量？RRF 的 k 怎么选？

#### Step 4.4 RAG 工具与评估
- 概念：上下文拼接、引用、幻觉、评估集、回归门禁。
- 交付：`knowledge_search` 工具；`cmd/eval`；基线指标。
- 验收：回答带引用；检索退化能被 eval 发现。
- 面试点：怎么衡量 RAG 好不好？召回和生成分别怎么评？

---

### Phase 5 — StateGraph（LangGraph 风格的 Go 实现）

**目标**：实现可检查点、可中断、可恢复的图执行引擎。

#### Step 5.1 图模型与执行器
- 概念：节点、边、条件边、状态、循环、最大步数。
- 交付：`graph.Graph`、`Node`、`Edge`、`ConditionalEdge`、`Run`；Mermaid 导出。
- 验收：线性/条件/循环图测试；超过 max steps 返回错误。
- 面试点：DAG 和带环状态图的区别？如何防死循环？

#### Step 5.2 并行与错误处理
- 概念：fan-out/fan-in、`errgroup`、context 取消、部分失败。
- 交付：并行分支；分支结果合并；重试/超时；事件流。
- 验收：`-race` 通过；一个分支失败时的策略可配置。
- 面试点：并行写共享 map 会怎样？如何做结构化并发？

#### Step 5.3 Checkpoint 与 HITL
- 概念：快照、版本、恢复、幂等、interrupt。
- 交付：SQLite checkpointer；`waiting_approval`；approve/reject 恢复；运行时间线。
- 验收：进程重启后能恢复；重复恢复不重复执行副作用；拒绝分支正确。
- 面试点：断点恢复如何保证 exactly-once/幂等？副作用节点怎么设计？

#### Step 5.4 示例工作流
- 交付：research → summarize → review（HITL）；条件路由；子图。
- 验收：CLI/HTTP 可跑；事件可回放；Mermaid 图正确。
- 面试点：什么场景该用工作流而不是自由 Agent？

---

### Phase 6 — eino 对照（框架适配）

**目标**：不是「会用 eino」，而是能讲清楚「手写 vs 框架」的边界。

#### Step 6.1 Spike：读官方 examples，跑通最小示例
- 交付：`docs/adr/0002-eino-adapter.md` 草稿；最小 ChatModel + Tool + Graph。
- 验收：能离线/用 fake 跑通；记录版本与 API 变化。
- 面试点：eino 的 Node/Edge/Graph 抽象与我们手写版的差异？

#### Step 6.2 Adapter：同一场景双实现
- 交付：`internal/llm/eino`、`internal/agent/eino`；同一组测试跑两套 runtime。
- 验收：fake 模型下行为一致；性能/代码量对比记录。
- 面试点：什么时候该用框架？框架带来的锁定和调试成本？

---

### Phase 7 — MCP 与多 Agent（扩展）

#### Step 7.1 MCP client
- 概念：JSON-RPC 2.0、initialize、tools/list、tools/call、stdio、HTTP、会话。
- 交付：client + fake server 测试 + 健康检查。
- 验收：工具发现/调用/错误/超时；无死锁。
- 面试点：MCP 和普通 HTTP 工具的区别？如何做能力协商？

#### Step 7.2 多 Agent 路由
- 概念：Host/Expert、路由、上下文隔离、A2A 基础。
- 交付：host 根据意图选 expert；expert 只能用自己的工具；A2A agent card（可选）。
- 验收：路由准确率测试；专家失败可回退。
- 面试点：多 Agent 什么时候是过度设计？如何避免上下文爆炸？

---

### Phase 8 — 可靠性与可观测性

#### Step 8.1 OTel + Prometheus + slog
- 交付：HTTP/LLM/tool/retrieval/graph span；指标；trace_id 注入日志。
- 验收：Jaeger 能看到一次完整调用链；`/metrics` 可抓取。
- 面试点：trace/metric/log 的分工？如何定位 p99 慢请求？

#### Step 8.2 限流、熔断、舱壁、降级
- 交付：per-user/per-provider 限流；熔断；并发舱壁；降级响应。
- 验收：故障注入下服务不雪崩；指标能反映。
- 面试点：重试 + 熔断 + 限流如何配合？什么是 retry storm？

#### Step 8.3 异步 worker 与幂等
- 交付：内存队列 → Redis Streams；任务重试、可见性超时、死信、幂等键。
- 验收：重复投递不重复处理；worker 崩溃可恢复。
- 面试点：至少一次 vs 恰好一次？Redis Streams 的 pending/claim 怎么用？

#### Step 8.4 压测与性能
- 交付：k6 场景；benchmark；pprof 报告；优化记录。
- 验收：给出 p95/p99、吞吐、内存；能解释瓶颈。
- 面试点：Go GC/逃逸分析/锁竞争怎么影响延迟？

---

### Phase 9 — 高可用与高并发专项

> 目标不是「堆中间件」，而是让每个 HA 设计都有可验证的故障场景和指标。先单机闭环，再多副本演练。

#### Step 9.1 无状态化与多副本
- 交付：compose 起 3 个 API 副本 + Nginx/HAProxy；readiness/liveness；`/readyz` 检查 DB/Redis；优雅摘流。
- 验收：压测中 kill 一个副本，无 5xx；滚动重启期间在途请求完成；日志能区分副本。
- 面试点：为什么不能把会话/限流计数/队列放进程内？readiness 和 liveness 的区别？

#### Step 9.2 Postgres 生产化
- 交付：pgx pool；连接池参数；索引；慢查询日志；迁移向后兼容；事务边界；分页。
- 验收：`EXPLAIN ANALYZE` 前后对比；连接池耗尽演练；迁移可回滚。
- 面试点：连接池大小怎么定？长事务有什么危害？读写分离的延迟怎么处理？

#### Step 9.3 Redis 缓存、限流、幂等
- 交付：cache-aside；TTL + jitter；singleflight；分布式滑动窗口/令牌桶；`Idempotency-Key`。
- 验收：缓存穿透/击穿/雪崩故障注入；多副本限流计数一致；重复请求只执行一次。
- 面试点：缓存一致性怎么保证？Redis 限流的原子性？时钟回拨怎么办？

#### Step 9.4 Redis Streams 分布式 worker
- 交付：consumer group；pending；XAUTOCLAIM；重试；DLQ；幂等消费；worker 多副本。
- 验收：kill worker 后任务被重新领取；重复投递不重复处理；毒任务进 DLQ。
- 面试点：至少一次 vs 恰好一次？pending 堆积怎么排查？

#### Step 9.5 SSE 长连接与背压
- 交付：心跳；连接上限；事件序列号 + `Last-Event-ID` 续传；Redis Pub/Sub 或 Streams 扇出；慢消费者策略。
- 验收：断线重连不丢关键事件；慢消费者不拖垮服务；多副本下能续传。
- 面试点：100k SSE 连接怎么估内存/文件描述符？为什么需要心跳？粘性会话能不能解决？

#### Step 9.6 故障演练与容量模型
- 交付：故障注入脚本；k6 场景；容量模型文档；演练记录。
- 验收：Redis 挂 / PG 慢 / LLM 超时 / worker 崩 / 副本被杀五类演练有结论；给出 p95/p99、QPS、连接数、成本。
- 面试点：如何做容量规划？故障时先降级什么？SLO/错误预算怎么定？

#### Step 9.7 K8s 清单（可选）
- 交付：Deployment/Service/Ingress/HPA/PDB/readiness/liveness；资源 requests/limits。
- 验收：滚动更新不丢请求；HPA 能扩容；PDB 不阻塞维护。
- 面试点：preStop + graceful shutdown 的顺序？HPA 看 CPU 还是自定义指标？

---

### Phase 10 — 评测、交付、演示

#### Step 10.1 评测体系
- 交付：JSONL 数据集；tool accuracy、task success、recall@k、MRR、nDCG、延迟/成本；CI 回归门禁。
- 验收：改动导致指标下降会失败；报告可读。
- 面试点：离线评测和线上指标怎么闭环？

#### Step 10.2 Docker 与部署
- 交付：多阶段 Dockerfile、compose（app + postgres + redis）、健康检查、非 root、graceful shutdown。
- 验收：一条命令起服务；数据持久化；日志可读。
- 面试点：镜像为什么小？如何传配置/密钥？

#### Step 10.3 作品集包装
- 交付：英文 README、架构图、demo GIF/视频、benchmark 表、面试讲稿。
- 验收：陌生人 5 分钟能跑起来；3 分钟能听懂亮点。
- 面试点：项目最难的问题是什么？如果重做会怎么改？

---

### Phase 11 — 可选：最小 Web 控制台

- 聊天流式输出 + 决策链时间线 + RAG 检索测试台 + 工作流审批。
- 技术：React + TypeScript 或 HTMX；只在 P0/P1 稳定后做。
- 验收：不引入新的后端复杂度；前端有类型和构建。

---

## 7. 测试策略

| 层级 | 工具 | 覆盖什么 | 何时跑 |
|---|---|---|---|
| 单元 | `testing` + 表驱动 | 纯函数、解析、排序、状态机 | 每次保存 |
| HTTP | `httptest` | handler、中间件、错误码、SSE | CI |
| 集成 | 临时 SQLite + fake LLM | repository、service、runtime | CI |
| 竞态 | `go test -race` | 并发/事件流/worker | CI 必跑 |
| 模糊 | `go test -fuzz` | SSE parser、JSON 参数、chunker | 定期/CI 短跑 |
| 基准 | `go test -bench` | RRF、chunker、事件分发 | 优化时 |
| E2E | compose + 真实模型（可选） | 全链路 | 手动/定时 |
| 评测 | `cmd/eval` | 检索/工具/任务成功率 | 回归门禁 |

原则：

- 每个 bug 先写失败测试，再修。
- 不追求 100% 覆盖率；核心包 70%+，边界和错误路径优先。
- 所有外部依赖都有 fake；真实模型只做可选 E2E。
- 测试不 sleep 等结果；用 channel/条件变量/可注入时钟。

---

## 8. 质量门禁与工程习惯

- `gofmt` / `go vet` / `staticcheck`（或 golangci-lint）必须过。
- `go test -race ./...` 必须过。
- 每个 Step 一个小 commit，消息用 conventional commits：`feat: ...`、`fix: ...`、`test: ...`、`docs: ...`。
- 关键决策写 ADR（`docs/adr/NNNN-*.md`）。
- 不提交密钥；提供 `.env.example`；日志脱敏。
- README 始终可运行；命令复制即可执行。
- 不引入无法解释的依赖；新增依赖前写清理由。

---

## 9. 面试叙事（提前准备）

**3 分钟版本**

> 我用 Go 从零写了一个 Agent 平台 Kestrel：手写 OpenAI 兼容流式客户端和 tool calling，ReAct runtime 产出统一事件流，通过 SSE 推给客户端；用 SQLite/Postgres 做会话、运行和事件持久化；RAG 实现 chunk + dense/BM25 混合检索 + RRF，并带 recall@k 评估；工作流用自研 StateGraph，支持条件边、并行、checkpoint 和 HITL 断点恢复；服务保持无状态，多副本部署，Redis 负责分布式限流、幂等和 Streams 队列，最后用 OTel/Prometheus 做全链路观测，k6 做故障演练和容量测试，CI 跑 race/lint/eval。另一个版本用 eino 适配同一套接口，用来对照框架和手写的取舍。

**必须能展开的深挖题**

- 为什么 ReAct 循环要有 max iterations + token budget + step timeout？分别防什么？
- 事件 channel 满了怎么办？丢事件、阻塞、还是背压？你的选择是什么？
- 工具调用失败要不要回填给模型？如何避免模型反复调用坏工具？
- 如何防 prompt injection 诱导 `read_file` / `http_get` 访问敏感资源？
- RAG 的 recall 和 answer quality 怎么分别评估？RRF 的 k 怎么选？
- StateGraph 的 checkpoint 如何保证恢复幂等？副作用节点怎么处理？
- 重试、限流、熔断、舱壁如何配合？什么是 retry storm？
- `context` 取消如何传到 LLM 流、工具执行和数据库查询？
- 为什么用 SQLite 起步？换 Postgres 要改哪些层？
- eino 帮你解决了什么？哪些地方你仍然选择手写？
- 服务如何做到无状态水平扩展？会话、SSE 订阅、限流计数分别放在哪？
- readiness 和 liveness 的区别？滚动更新时如何保证不丢在途请求？
- 多副本下定时任务/worker 如何避免重复执行？什么时候需要 leader election？
- 缓存穿透、击穿、雪崩分别怎么处理？缓存和数据库的一致性怎么取舍？
- 分布式限流用 Redis 怎么做？滑动窗口和令牌桶的差异？时钟回拨怎么办？
- 幂等键如何设计？并发重复请求怎么保证只执行一次？
- Redis Streams 的 consumer group、pending、XAUTOCLAIM 怎么用？至少一次怎么变成业务上的恰好一次？
- Postgres 连接池大小怎么定？长事务、慢查询、索引失效怎么定位？
- 10 万 SSE 长连接怎么估算内存和文件描述符？为什么需要心跳和断线续传？
- 熔断、重试、限流、舱壁如何配合？什么是 retry storm 和级联失败？

**可量化指标（做完后填真实数字）**

- `go test -race` 覆盖包数 / 核心包覆盖率
- 检索黄金集 recall@5、MRR
- Agent 任务成功率 / 平均工具调用次数
- `/healthz` 与聊天接口 p95/p99
- 单次对话 token 成本
- 多副本压测 QPS、错误率、429 比例、副本故障恢复时间
- 连接池使用率、Redis 命中率、队列深度、pending 数
- 每副本 SSE 连接数 / 内存 / 文件描述符
- 二进制大小、冷启动时间、内存占用

---

## 10. 协作方式（我们怎么一步步做）

1. **每一步先讲概念和验收标准**，再写代码。
2. **你先自己写**；卡住先给提示，不直接给答案；实在卡住再看参考实现。
3. 每步结束必须：`gofmt`、`go vet`、`go test -race`、能运行、能解释。
4. 一个 Step 一个 commit；提交前你确认。
5. 不复制 `nexus-agent`；可以读它做对照，在文档里注明「参考了哪个设计」。
6. 每步更新本文件的「进度」和「决策记录」。

---

## 11. 风险与取舍

| 风险 | 表现 | 对策 |
|---|---|---|
| 范围蔓延 | 想一次做完 RAG/工作流/MCP/前端 | 严守 P0；每个 Phase 有验收，不通过不进入下一个 |
| 基础设施过重 | 一开始就上 Milvus/Redis/K8s | SQLite/内存优先；接口留好，后换实现 |
| 真实模型成本/不稳定 | 测试依赖 API Key、结果漂移 | Fake Provider 为主；真实模型只做可选 E2E |
| eino API 变化 | 示例跑不通 | 先 spike，pin 版本，adapter 隔离 |
| Windows 差异 | 路径、信号、Docker、`-race` 需要 cgo | 路径用 `filepath`；信号用 `os.Interrupt`；race 放 Linux CI / WSL2；Docker Desktop/WSL2 |
| 只写不测 | 功能多但一改就坏 | 每个 Step 先写测试；CI 门禁 |
| 面试讲不清 | 代码是抄的/框架堆的 | 每个核心文件自己重写；写 ADR；做 code review |

---

## 12. 进度

- [x] 路线图文档（含高可用/高并发能力地图）
- [x] Phase 0 / Step 0.1 项目骨架 + 健康检查 + 测试
- [x] Phase 0 / Step 0.2 配置 + 日志 + 中间件 + 优雅退出
- [ ] Phase 0 / Step 0.3 CI
- [ ] Phase 1 LLM Gateway
- [ ] Phase 2 Tools + ReAct
- [ ] Phase 3 持久化 + 认证 + API
- [ ] Phase 4 RAG
- [ ] Phase 5 StateGraph
- [ ] Phase 6 eino 对照
- [ ] Phase 7 MCP + 多 Agent
- [ ] Phase 8 可靠性与可观测
- [ ] Phase 9 高可用与高并发专项
- [ ] Phase 10 评测与交付
- [ ] Phase 11 可选 Web

---

## 13. 决策记录（ADR 索引）

- ADR-001：先 stdlib，后框架；HTTP 用 `net/http`。
- ADR-002：SQLite 起步，Postgres 作为生产适配；SQL 手写。
- ADR-003：手写 Agent runtime + eino adapter 对照，不做 eino-only。
- ADR-004：LangGraph 是 Python/JS 生态；我们在 Go 中实现同类 StateGraph，而不是引入 Python 服务。
- ADR-005：测试以 Fake Provider 为主，真实模型只做可选 E2E。
- ADR-006：事件流是 runtime 的唯一输出，HTTP/CLI/未来 WebSocket 都消费同一份事件。
- ADR-007：服务保持无状态；会话、队列、限流计数、SSE 订阅外置到 Postgres/Redis，支持多副本。
- ADR-008：本地 SQLite/内存，生产 Postgres/Redis；用接口隔离两套实现，不让业务代码依赖具体存储。
- ADR-009：HA 用 compose 多副本 + 故障演练验证，不假装有真实大规模流量；所有结论必须有实测指标。
- ADR-010：SSE 长连接单独设计：心跳、连接上限、事件序号 + `Last-Event-ID` 续传、跨副本扇出。
