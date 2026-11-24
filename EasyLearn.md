# EasyLearn —— Kestrel Agent 学习交接

> 换电脑、换窗口、隔几天回来时，先读这份。
> 给 AI 看的详细上下文在 [`docs/AI-HANDOFF.md`](docs/AI-HANDOFF.md)；工程路线图在 [`docs/IMPLEMENTATION_PLAN.md`](docs/IMPLEMENTATION_PLAN.md)。

---

## 1. 项目是什么

- **项目名**：Kestrel Agent
- **仓库**：https://github.com/WilliamPockey/kestrel-agent
- **模块名**：`kestrel`
- **Go 版本**：1.26+
- **定位**：从零用 Go 实现一个可演示、可测试、可解释的 Agent 平台，用来学习 Go 后端 + AI 工程，并作为实习作品集。
- **原则**：不复制 `nexus-agent`；先 stdlib 后框架；先单机闭环，再多副本/高可用演练。

---

## 2. 当前进度（准确状态）

### 2.1 工程代码

已经完成：

- Step 0.1：HTTP 服务、`/healthz`、表驱动测试。
- Step 0.2：环境变量配置、`slog` 结构化日志、请求 ID、安全响应头、访问日志、panic 恢复、`/readyz`、优雅退出。
- `go test ./...`、`go vet ./...`、`go build ./...` 通过。
- Windows 本地没有 gcc，`go test -race` 暂时跑不了；以后放 Linux CI / WSL2。

**还没有做**：

- CI / GitHub Actions
- LLM Gateway
- 工具调用 / ReAct
- 数据库 / 认证 / RAG / 工作流
- goroutine / channel / context 的正式学习（代码里已经用了，但还没讲）

### 2.2 学习进度

你已经完成：

1. 跑通现有服务，用 `curl` 看到真实 HTTP 请求。
2. 练习 `lessons/01_functions`：函数、参数、返回值。
3. 练习 `lessons/02_http_server` 的第 1、2 步：
   - 用 `http.HandleFunc` 注册 `/hello`、`/version`；
   - 看到 `/missing` 返回 404；
   - 看到 `POST /hello` 仍然返回 200，因为只按路径注册，没有限制方法。

**当前停在这里：第 3 步还没做。**

第 3 步的任务是：把 `lessons/02_http_server/main.go` 改成显式 `mux` + 方法限制，观察 405：

```go
func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "hello")
	})

	mux.HandleFunc("GET /version", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "v0.1.0")
	})

	fmt.Println("listen on http://localhost:8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		fmt.Println(err)
	}
}
```

然后运行并观察：

```bash
curl.exe -i http://localhost:8081/hello
curl.exe -i -X POST http://localhost:8081/hello
curl.exe -i http://localhost:8081/version
curl.exe -i -X POST http://localhost:8081/version
curl.exe -i http://localhost:8081/missing
```

应该看到：

- `GET /hello`、`GET /version` → 200
- `POST /hello`、`POST /version` → 405 `Method Not Allowed`
- `GET /missing` → 404

第 3 步做完后，再进入：

4. JSON 响应与 `writeJSON`（对应项目 `/healthz`）
5. 为什么需要 `http.Handler` / `http.HandlerFunc`
6. 中间件为什么长成 `func(http.Handler) http.Handler`（对应 `NewRouter`）
7. goroutine / channel / context + 优雅退出（对应 `app.Run`）
8. `httptest` + 表驱动测试

---

## 3. 新电脑怎么开始

### 3.1 安装

- 安装 Go 1.26+：https://go.dev/dl/
- 终端检查：

```bash
go version
```

应该看到 `go1.26.x`。

### 3.2 克隆仓库

```bash
git clone https://github.com/WilliamPockey/kestrel-agent.git
cd kestrel-agent
```

### 3.3 验证工程

```bash
go test ./...
go vet ./...
go build ./...
```

全部无报错后，运行服务：

```bash
go run ./cmd/server
```

另开终端：

```bash
curl.exe -i http://localhost:8080/healthz
curl.exe -i http://localhost:8080/readyz
curl.exe -i -X POST http://localhost:8080/healthz
```

预期：

- `/healthz` → 200 + JSON
- `/readyz` → 200 + JSON
- `POST /healthz` → 405

按 `Ctrl+C` 停止，应看到 `shutdown requested` / `server stopped`。

### 3.4 运行学习练习

```bash
go run ./lessons/01_functions
go run ./lessons/02_http_server
```

`lessons/02_http_server` 会占用 8081 端口；另开终端用 `curl`。

---

## 4. 目录说明

```text
kestrel-agent/
├── cmd/server/                 # 服务入口
├── internal/
│   ├── app/                    # 服务器生命周期、优雅退出
│   ├── config/                 # 环境变量配置与校验
│   ├── httpapi/                # 路由、中间件、JSON 响应
│   └── observability/          # slog 日志
├── lessons/                    # 学习用的最小程序
│   ├── 01_functions/
│   └── 02_http_server/
├── docs/
│   ├── IMPLEMENTATION_PLAN.md  # 工程路线图（11 个 Phase）
│   ├── GO-BASICS.md            # Go 基础补课
│   ├── AI-HANDOFF.md           # 给 AI 的交接
│   ├── step-00.md              # Step 0.1 讲义
│   └── step-00-2.md            # Step 0.2 讲义
├── Makefile
├── README.md
└── EasyLearn.md                # 本文件
```

---

## 5. 我的基础与教学约定（给下一个 AI 看）

### 5.1 我会什么

- Go：变量、函数、多返回值、`error`、struct、方法、指针、interface、类型断言、泛型。
- HTTP：方法、状态码、JSON、REST 的基本含义。
- 我不会：goroutine、channel、context、sync；Go 的 Handler/路由/中间件；SQL、Redis、高可用。

### 5.2 怎么教我

- **先讲为什么，再讲是什么**；不要突然丢一个新概念。
- 一次只引入一个新概念，每个概念先给最小可运行例子。
- 多用「当前代码 → 下一行代码」的演进方式，不要一次性贴大段代码。
- 讲完一个点，给我 1–3 个小问题或小练习，确认我懂了再往下。
- 不要假设我已经跑过命令；让我实际运行，并把输出发给你。
- 不要让我同时面对 `Handler`、`HandlerFunc`、`ServeMux`、中间件五个新词。
- 代码要能 `go test` / `go vet` / `go build`。

### 5.3 给新 AI 的开场白

```text
请先读 EasyLearn.md、docs/AI-HANDOFF.md、docs/IMPLEMENTATION_PLAN.md、
docs/step-00-2.md，以及 lessons/02_http_server/main.go。

我刚完成到「第 3 步之前」：/hello 和 /version 已经注册，
GET 返回 200，POST /hello 也返回 200，因为还没有方法限制。
下一步是改成显式 mux + "GET /..."，观察 405。

我的基础：Go 会变量/函数/多返回值/error、struct/方法/指针/interface/类型断言；
HTTP 会方法/状态码/JSON/REST；不会 goroutine/channel/context，也不会 Go 的 Handler/中间件。

请先检查我是否已经完成第 3 步，再按「先讲为什么、一次一个概念、给最小例子和 checkpoint」的方式继续。
不要一次性写很多代码，不要跳过我的基础。
```

---

## 6. 常用命令

```bash
# 测试 / 静态检查 / 构建
go test ./...
go vet ./...
go build ./...

# 运行服务
go run ./cmd/server

# 运行学习练习
go run ./lessons/01_functions
go run ./lessons/02_http_server

# 格式化
go fmt ./...

# 查看测试覆盖
go test -cover ./...
```

Makefile：

```bash
make test
make vet
make run
make build
```

---

## 7. 重要注意事项

- 仓库已经独立成自己的 git 仓库，不再属于 `D:\C\Projects` 那个仓库。
- `bin/`、`*.exe`、日志、`.env`、`.commandcode/` 不会提交。
- 不要把真实 API Key、密码、token 写进代码或提交。
- Windows 本地没有 gcc，`go test -race` 会报 `-race requires cgo`；以后在 Linux CI 或 WSL2 跑。
- 目前没有 CI；Step 0.3 会补 GitHub Actions。
- 不要直接复制 `nexus-agent` 的代码；可以读它做对照，但要写自己的实现和测试。

---

## 8. 下一步路线（简要）

| 步骤 | 内容 | 对应项目代码 |
|---|---|---|
| 第 3 步（当前） | 显式 mux + `GET /...` + 405 | `router.go` 的 `mux.HandleFunc` |
| 第 4 步 | JSON 响应、状态码 | `response.go` 的 `writeJSON` |
| 第 5 步 | 为什么需要 Handler / HandlerFunc | `http.Handler`、`mux.HandleFunc` |
| 第 6 步 | 中间件套娃 | `NewRouter` 的 `RequestID(...)` |
| 第 7 步 | goroutine / channel / context / 优雅退出 | `app.Run`、`main.go` |
| 第 8 步 | `httptest` + 表驱动测试 | `*_test.go` |
| 之后 | LLM Gateway → 工具/ReAct → 持久化/认证 → RAG → StateGraph → eino → MCP → 高可用 | `docs/IMPLEMENTATION_PLAN.md` |

---

## 9. 换电脑后的第一件事

```bash
git clone https://github.com/WilliamPockey/kestrel-agent.git
cd kestrel-agent
go test ./...
```

如果测试通过，打开 `lessons/02_http_server/main.go`，从第 3 步继续。
