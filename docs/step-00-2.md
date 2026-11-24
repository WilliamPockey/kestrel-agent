# Step 0.2：配置、日志、中间件、优雅退出

> 这一步开始像一个真正的后端服务：配置可注入、日志可检索、请求可追踪、panic 不拖垮进程、退出不丢请求。
> 同时也是你第一次正式接触 **goroutine、channel、select、context、signal、sync**。

---

## 1. 本步目标

- 用环境变量配置服务，并在启动时校验。
- 用 `log/slog` 输出结构化日志（开发用 text，生产用 JSON）。
- 给每个请求生成/透传 `X-Request-ID`，并写进日志。
- panic 时返回 500，而不是让进程崩掉。
- 用中间件记录方法、路径、状态码、耗时、字节数。
- 收到 `Ctrl+C` / `SIGTERM` 时优雅退出，给在途请求完成时间。
- 为后面的多副本、readiness、滚动更新打基础。

---

## 2. 新增/修改的文件

```
cmd/server/main.go                 # 组装：config → logger → signal → app.Run
internal/config/config.go          # 环境变量、默认值、校验
internal/config/config_test.go
internal/observability/logger.go   # slog 初始化
internal/observability/logger_test.go
internal/httpapi/router.go         # 路由 + 中间件链
internal/httpapi/response.go       # writeJSON / errorResponse
internal/httpapi/middleware.go     # RequestID / Logging / Recovery / SecurityHeaders
internal/httpapi/middleware_test.go
internal/httpapi/router_test.go
internal/app/app.go                # NewServer / Run（生命周期）
internal/app/app_test.go
```

---

## 3. 先理解 8 个概念

### 3.1 struct、方法、指针接收者

```go
type HTTPConfig struct {
    Addr string
    ReadTimeout time.Duration
}

func (c Config) Validate() error { ... }
```

- `type X struct` 定义结构体；字段首字母大写才能被其他包访问。
- `func (c Config) Validate()` 是值接收者：方法内拿到一份拷贝。
- 如果方法要修改原对象，或者结构体很大，用指针接收者 `func (c *Config)`。
- 我们的 `Config` 很小，`Validate` 只读，所以用值接收者。

### 3.2 构造函数

Go 没有 `__init__`，习惯写一个 `NewXxx` 函数：

```go
func NewLogger(cfg config.LogConfig) (*slog.Logger, error) {
    // ...
}
```

- 返回 `(对象, error)`，让调用方决定失败怎么处理。
- 不要在里面 `os.Exit` 或 `panic`；库代码只返回错误。

### 3.3 环境变量与 12-factor

```go
func env(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}
```

- 配置不写死在代码里，部署时用环境变量覆盖。
- 空字符串视为「没设置」，走默认值。
- `time.ParseDuration("5s")` 解析 `time.Duration`；解析失败要带上是哪个变量。

### 3.4 `log/slog`

```go
logger.InfoContext(ctx, "http request",
    "method", r.Method,
    "status", 200,
    "duration_ms", 3,
    "request_id", id,
)
```

- `slog` 是标准库结构化日志。
- 参数是「键, 值, 键, 值...」，不是格式化字符串。
- `JSONHandler` 适合生产（日志采集），`TextHandler` 适合本地阅读。
- `InfoContext` 会把 context 传给 handler；以后可以自动带 trace_id。

### 3.5 context

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

- `context` 用来传递取消信号、截止时间、请求级元数据。
- 函数第一个参数通常是 `ctx context.Context`。
- `context.WithValue` 只放请求级元数据（request id、user id），不要放数据库连接、logger 这类依赖。
- 自定义 key 类型，避免不同包用字符串 key 冲突：

```go
type contextKey string
const requestIDKey contextKey = "request_id"
```

### 3.6 goroutine、channel、select

```go
errCh := make(chan error, 1)
go func() {
    errCh <- srv.Serve(listener)
}()
```

- `go func(){}()` 启动一个 goroutine，主流程继续往下走。
- `errCh` 有 1 个缓冲：即使主流程已经返回，goroutine 也能把错误放进 channel 后退出，不会永久阻塞（goroutine 泄漏）。
- `select` 同时等待多个事件：

```go
select {
case <-ctx.Done():
    // 收到退出信号
case err := <-errCh:
    // 服务启动失败/退出
}
```

- 多个 case 同时就绪时随机选一个。
- `ctx.Done()` 是一个 channel，取消/超时会关闭它。

### 3.7 signal

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
```

- `Ctrl+C` 在 Windows/Linux 都是 `os.Interrupt`。
- 容器/K8s 停止 Pod 时通常发 `SIGTERM`。
- `NotifyContext` 把信号转成 context 取消，后续代码只需要看 `ctx.Done()`。
- `defer stop()` 释放信号监听。

### 3.8 `Shutdown` vs `Close`

- `Server.Shutdown(ctx)`：
  - 停止接受新连接；
  - 关闭空闲连接；
  - 等待正在处理的请求完成；
  - 直到 ctx 截止时间。
- `Server.Close()`：立即关闭所有连接，不等在途请求。
- 优雅退出的顺序：先 `Shutdown`，超时才 `Close`。

---

## 4. 代码逐文件讲解

### 4.1 `internal/config/config.go`

```go
type Config struct {
    Env  string
    HTTP HTTPConfig
    Log  LogConfig
}
```

配置按领域分组。以后会加 `DB`、`Redis`、`LLM` 等。

```go
func Default() Config {
    return Config{
        Env: "dev",
        HTTP: HTTPConfig{
            Addr:              ":8080",
            ReadHeaderTimeout: 5 * time.Second,
            ReadTimeout:       15 * time.Second,
            WriteTimeout:      0,
            IdleTimeout:       60 * time.Second,
            ShutdownTimeout:   10 * time.Second,
        },
        Log: LogConfig{Level: "info", Format: "text"},
    }
}
```

每个超时的作用：

| 字段 | 作用 | 为什么重要 |
|---|---|---|
| `ReadHeaderTimeout` | 读完请求头的上限 | 防 Slowloris：客户端一点点发头，占住连接 |
| `ReadTimeout` | 读完整个请求的上限 | 防慢请求体拖住 goroutine |
| `WriteTimeout` | 写完整响应的上限 | 普通 API 要有；SSE 长连接必须设为 0 |
| `IdleTimeout` | keep-alive 空闲连接保留时间 | 平衡连接复用和资源占用 |
| `ShutdownTimeout` | 优雅退出的等待预算 | 太短会丢在途请求，太长会拖慢发布 |

`WriteTimeout: 0` 是刻意选择：SSE 是一条长时间不结束的响应，非零的 `WriteTimeout` 会在到点时切断它。代价是慢客户端可能占住连接，后面用 per-request deadline、心跳、连接上限来解决。

```go
func Load() (Config, error) {
    cfg := Default()
    cfg.Env = env("KESTREL_ENV", cfg.Env)
    cfg.HTTP.Addr = env("KESTREL_HTTP_ADDR", cfg.HTTP.Addr)
    // ... duration 解析
    cfg.Log.Level = env("KESTREL_LOG_LEVEL", cfg.Log.Level)
    cfg.Log.Format = env("KESTREL_LOG_FORMAT", defaultLogFormat(cfg.Env))
    if err := cfg.Validate(); err != nil {
        return Config{}, err
    }
    return cfg, nil
}
```

注意：

- `Load` 先构造默认值，再逐个覆盖。
- 所有错误都返回，不在配置包里 `log.Fatal`。
- `defaultLogFormat` 让 `prod` 默认 JSON，本地默认 text。

`Validate` 做了这些检查：

- env/addr 不能为空（`strings.TrimSpace` 防全是空格）；
- 超时必须是正数；
- `WriteTimeout` 允许 0，但不能为负；
- 日志级别/格式必须是支持的值。

**面试点**：配置校验应该在启动时做，失败快速退出（fail fast），而不是运行到一半才发现。

### 4.2 `internal/config/config_test.go`

```go
setEnv(t, map[string]string{
    "KESTREL_HTTP_READ_TIMEOUT": "soon",
})

_, err := Load()
if err == nil {
    t.Fatal("Load() accepted an invalid duration")
}
```

- `t.Setenv` 会在测试结束后恢复环境变量，避免测试之间互相污染。
- 先把所有相关变量设成空字符串，模拟「没设置」，再设置需要覆盖的值。
- 测试错误时不仅看 `err != nil`，还看错误信息是否包含变量名，方便排查。

### 4.3 `internal/observability/logger.go`

```go
func NewLogger(cfg config.LogConfig) (*slog.Logger, error) {
    level, err := parseLevel(cfg.Level)
    if err != nil {
        return nil, err
    }
    opts := &slog.HandlerOptions{Level: level}
    switch cfg.Format {
    case "json":
        handler = slog.NewJSONHandler(os.Stdout, opts)
    case "text":
        handler = slog.NewTextHandler(os.Stdout, opts)
    }
    return slog.New(handler), nil
}
```

- 日志是依赖，不是全局变量；通过参数传入，测试时可以换成写入 buffer 的 logger。
- `slog.Level` 控制最低输出级别。
- `os.Stdout` 是标准输出；容器里通常由日志采集器收集。
- 不在这里 `slog.SetDefault`；由 `main` 决定是否设为全局。

### 4.4 `internal/httpapi/middleware.go`

#### RequestID

```go
func RequestID(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := r.Header.Get(requestIDHeader)
        if !validRequestID(id) {
            id = newRequestID()
        }
        w.Header().Set(requestIDHeader, id)
        ctx := context.WithValue(r.Context(), requestIDKey, id)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

- 如果上游（网关/客户端）传了合法的 `X-Request-ID`，复用它；否则生成一个。
- 写回响应头，客户端可以用它报障。
- 放进 context，后续 handler/日志/LLM 调用都能拿到。
- `validRequestID` 限制 1–64 个字符，只允许字母、数字、`-_.`，防止 header 注入和日志注入。

`newRequestID` 用 `crypto/rand` 生成 16 字节随机数并 hex 编码；随机源失败时用纳秒时间戳兜底。

#### statusRecorder

```go
type statusRecorder struct {
    http.ResponseWriter
    status int
    bytes  int
}
```

这是本步最重要的 Go 技巧之一：**包装 `http.ResponseWriter`，记录状态码和字节数**。

```go
func (r *statusRecorder) WriteHeader(code int) {
    if r.status != 0 {
        return
    }
    r.status = code
    r.ResponseWriter.WriteHeader(code)
}
```

- 防止重复 `WriteHeader`（重复会触发 `superfluous response.WriteHeader call`）。
- `Write` 前如果没写状态码，默认 200。

包装的陷阱：`http.ResponseWriter` 背后可能还实现了 `http.Flusher`、`http.Hijacker`、`io.ReaderFrom` 等可选接口。你把它包成一个新类型后，这些接口就「消失」了。SSE 必须能 `Flush`，所以我们要显式转发：

```go
func (r *statusRecorder) Flush() {
    if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
        flusher.Flush()
    }
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
    return r.ResponseWriter
}
```

- `Unwrap` 让 `http.ResponseController` 能拿到底层 writer，设置写超时、flush 等。
- 以后 WebSocket 需要 `Hijacker`，也要类似转发，或者用 `ResponseController`。

#### Logging

```go
start := time.Now()
recorder := &statusRecorder{ResponseWriter: w}
next.ServeHTTP(recorder, r)
logger.InfoContext(r.Context(), "http request",
    "method", r.Method,
    "path", r.URL.Path,
    "status", status,
    "bytes", recorder.bytes,
    "duration_ms", time.Since(start).Milliseconds(),
    "request_id", RequestIDFromContext(r.Context()),
    "remote_addr", r.RemoteAddr,
)
```

- `duration_ms` 是排查慢请求的起点。
- `status`/`bytes` 来自 recorder。
- `request_id` 来自 context。
- 真实生产还会记录 `trace_id`、`user_id`、`route`、`error_code`，但不要在日志里放密码/token/完整 prompt。

#### Recovery

```go
defer func() {
    recovered := recover()
    if recovered == nil {
        return
    }
    logger.ErrorContext(r.Context(), "panic recovered",
        "panic", recovered,
        "stack", string(debug.Stack()),
        "request_id", RequestIDFromContext(r.Context()),
    )
    if recorder, ok := w.(*statusRecorder); ok && recorder.status != 0 {
        return
    }
    writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
}()
```

- `recover` 只能在 `defer` 里生效，且只能恢复当前 goroutine 的 panic。
- 记录堆栈方便定位，但不要把堆栈返回给客户端。
- 如果响应已经开始写（`status != 0`），不能再写 500，否则会触发 superfluous 警告；这时只记录日志。
- panic 是 bug，不是业务错误；业务错误应该返回 `error`。

#### SecurityHeaders

```go
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("Referrer-Policy", "no-referrer")
```

- `nosniff` 防止浏览器猜 MIME 类型。
- `Referrer-Policy` 减少 Referer 泄露。
- 以后有前端时再加 CSP/HSTS 等。

### 4.5 `internal/httpapi/router.go` 与中间件顺序

```go
return RequestID(SecurityHeaders(Logging(logger)(Recovery(logger)(mux))))
```

从外到内：

```
RequestID → SecurityHeaders → Logging → Recovery → mux
```

为什么这样排：

- `RequestID` 最外层：后面所有日志都能拿到 ID。
- `SecurityHeaders` 在写响应前设置头。
- `Logging` 包住 `Recovery`：Recovery 捕获 panic 并写 500 后，Logging 能记录到 500。
- `Recovery` 在 mux 外：任何 handler panic 都能兜住。

中间件本质是 `func(http.Handler) http.Handler`，一层层包起来。Go 没有框架魔法，顺序就是代码顺序。

### 4.6 `internal/app/app.go`：生命周期

```go
listener, err := net.Listen("tcp", cfg.HTTP.Addr)
if err != nil {
    return fmt.Errorf("listen on %s: %w", cfg.HTTP.Addr, err)
}
```

显式 `net.Listen` 的好处：

- 端口占用时立刻返回错误，不会假装启动成功；
- 测试时可以用 `127.0.0.1:0` 让系统分配随机端口；
- 日志里能打印实际监听地址。

为了能测试完整生命周期，`Run` 只负责 `net.Listen`，真正等待信号、关闭服务的逻辑放在 `serve` 里：

```go
func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
    listener, err := net.Listen("tcp", cfg.HTTP.Addr)
    if err != nil {
        return fmt.Errorf("listen on %s: %w", cfg.HTTP.Addr, err)
    }
    return serve(ctx, cfg, logger, listener)
}
```

测试可以直接传入一个 `net.Listener`，不需要真实固定端口。下面 `serve` 里的逻辑才是核心。

```go
errCh := make(chan error, 1)
go func() {
    errCh <- srv.Serve(listener)
}()
```

- `Serve` 是阻塞的，必须放 goroutine。
- channel 缓冲 1：主流程提前返回时，goroutine 仍能发送并退出，避免泄漏。

```go
select {
case <-ctx.Done():
    logger.Info("shutdown requested")
case err := <-errCh:
    if err == nil || errors.Is(err, http.ErrServerClosed) {
        return nil
    }
    return fmt.Errorf("serve: %w", err)
}
```

- 正常运行时，两个 case 都不满足，阻塞等待。
- 收到信号 → `ctx.Done()`。
- 服务自己挂掉 → `errCh`。

```go
shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
defer cancel()

if err := srv.Shutdown(shutdownCtx); err != nil {
    _ = srv.Close()
    return fmt.Errorf("shutdown: %w", err)
}
```

- 用新的 `context.Background()`，因为原来的 ctx 已经被取消；如果直接用原 ctx，`Shutdown` 会立刻超时。
- `Shutdown` 超时才 `Close`，保证尽量优雅。
- `Shutdown` 成功后，`Serve` 返回 `http.ErrServerClosed`，从 `errCh` 读出来确认。

**面试点**：为什么 `Shutdown` 要用一个全新的 context？
答：触发退出的 ctx 已经 canceled，`Shutdown` 需要自己的超时预算来等待在途请求。

### 4.7 `cmd/server/main.go`

```go
cfg, err := config.Load()
if err != nil {
    slog.Error("load config", "err", err)
    os.Exit(1)
}

logger, err := observability.NewLogger(cfg.Log)
if err != nil {
    slog.Error("init logger", "err", err)
    os.Exit(1)
}
slog.SetDefault(logger)

ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

if err := app.Run(ctx, cfg, logger); err != nil {
    logger.Error("server exited", "err", err)
    os.Exit(1)
}
```

- 配置/日志初始化失败时，还没有自定义 logger，所以用默认 `slog`。
- `slog.SetDefault` 让第三方库的 `slog` 调用也用我们的格式。
- `signal.NotifyContext` 把 `Ctrl+C`/`SIGTERM` 变成 ctx 取消。
- `app.Run` 返回 nil 表示正常退出。

---

## 5. 请求生命周期

```
client
  │
  ▼
net/http (每个请求一个 goroutine)
  │
  ▼
RequestID        生成/校验 X-Request-ID，放进 context
  │
  ▼
SecurityHeaders  设置安全响应头
  │
  ▼
Logging          包一层 statusRecorder，记录状态/字节/耗时
  │
  ▼
Recovery         defer recover，panic 时写 500
  │
  ▼
mux              GET /healthz → handler
  │
  ▼
writeJSON        设置 Content-Type → WriteHeader → Encode
```

---

## 6. 优雅退出时序

```
Ctrl+C / SIGTERM
  │
  ▼
signal.NotifyContext 取消 ctx
  │
  ▼
app.Run 的 select 命中 ctx.Done()
  │
  ▼
Server.Shutdown(ctxWithTimeout)
  ├─ 停止接受新连接
  ├─ 关闭空闲连接
  └─ 等待在途请求完成
  │
  ▼
Serve 返回 http.ErrServerClosed
  │
  ▼
app.Run 返回 nil
  │
  ▼
main 退出，defer stop() 释放信号监听
```

---

## 7. 运行与验证

```bash
# 1. 默认启动（text 日志）
go run ./cmd/server

# 2. 另开终端
curl.exe -i http://localhost:8080/healthz
curl.exe -i http://localhost:8080/readyz
curl.exe -i -H "X-Request-ID: test-123" http://localhost:8080/healthz

# 3. 让日志变成 JSON
set KESTREL_ENV=prod
set KESTREL_LOG_FORMAT=json
go run ./cmd/server

# 4. 改端口
set KESTREL_HTTP_ADDR=127.0.0.1:18081
go run ./cmd/server

# 5. 非法配置应该启动失败
set KESTREL_LOG_LEVEL=verbose
go run ./cmd/server
```

Windows cmd 用 `set`；PowerShell 用 `$env:KESTREL_ENV="prod"`；bash 用 `export KESTREL_ENV=prod`。

期望日志（text）：

```text
time=... level=INFO msg="server started" addr=127.0.0.1:18081 env=dev
time=... level=INFO msg="http request" method=GET path=/healthz status=200 bytes=36 duration_ms=0 request_id=... remote_addr=127.0.0.1:...
```

按 `Ctrl+C` 后应该看到：

```text
level=INFO msg="shutdown requested"
level=INFO msg="server stopped"
```

---

## 8. 测试与验收

```bash
go test ./...
go test -v ./...
go vet ./...
go build ./...
```

覆盖点：

- 配置默认值、覆盖、非法值、prod 默认 JSON。
- RequestID 生成/透传/拒绝非法值。
- Recovery 返回 500 JSON。
- statusRecorder 保留 Flush。
- Logging 记录状态码和 request_id。
- SecurityHeaders。
- `/healthz` 200、POST 405、`/readyz` 200、未知路由 404。
- 服务器能启动、请求成功、取消 context 后 `serve` 正常返回。
- `Shutdown` 后 `Serve` 返回 `http.ErrServerClosed`。

测试并发代码时尽量**不要 sleep 猜时间**。`TestServeGracefulShutdown` 用了一个包装过的 listener：第一次 `Accept` 时关闭 `ready` channel，测试等到这个事件再发请求。这是用事件同步代替 sleep 的例子。

验收清单：

- [ ] `go test ./...` 通过。
- [ ] `go vet ./...` 无输出。
- [ ] 启动后 `/healthz`、`/readyz` 正常。
- [ ] 响应头有 `X-Request-ID`；传入合法 ID 会复用。
- [ ] 日志包含 method/path/status/bytes/duration_ms/request_id。
- [ ] `Ctrl+C` 能看到 shutdown 日志并正常退出。
- [ ] 非法日志级别启动失败且退出码非 0。
- [ ] 你能解释：为什么 `WriteTimeout` 默认是 0；为什么 `Shutdown` 要用新的 context；为什么 `Recovery` 要在 `Logging` 里面。

---

## 9. 练习

1. 给 `/version` 增加 `X-Request-ID` 测试，确认每个请求 ID 不同。
2. 写一个会 panic 的测试 handler，确认返回 500 且日志里有 `panic recovered`。
3. 把 `SecurityHeaders` 加到中间件链的不同位置，观察响应头是否还在；解释为什么。
4. 把 `KESTREL_HTTP_SHUTDOWN_TIMEOUT` 设成 `1ns`，写一个慢 handler（sleep 100ms），观察 `Shutdown` 超时和日志。
5. 用 `go test -cover ./...` 看覆盖率，找出没测到的分支。
6. 不看代码，画出「一个请求从进入到写日志」的中间件顺序。

---

## 10. 常见错误

| 现象 | 原因 | 解决 |
|---|---|---|
| `listen on :8080: bind: address already in use` | 端口被占用 | 换端口或停掉旧进程 |
| `superfluous response.WriteHeader call` | 重复写状态码 | 用 statusRecorder 防重，或检查 handler |
| `panic: runtime error: invalid memory address` | nil 指针解引用 | 检查 error 后再用对象 |
| 日志没有 request_id | 中间件顺序错了，或 context 没传下去 | RequestID 放最外层，handler 用 `r.Context()` |
| `Shutdown` 立刻超时 | 用了已经取消的 ctx | 用 `context.WithTimeout(context.Background(), ...)` |
| 退出时卡住 | 有 goroutine 没退出/没关连接 | 检查 channel、WaitGroup、长连接 |
| `-race requires cgo` | Windows 没 C 工具链 | Linux CI / WSL2，或安装 gcc |

---

## 11. 这一步和高可用/高并发的关系

- **readiness**：多副本部署时，负载均衡只把流量发给 ready 的实例。现在 `/readyz` 只返回 ready；Phase 3 会检查数据库/Redis。
- **优雅退出**：滚动发布/杀副本时，先停止接新请求，再等在途请求完成，避免 5xx。
- **超时**：每个连接、每个请求都有边界，防止慢客户端耗尽 goroutine/连接。
- **request id**：多副本下能把一次请求的日志串起来；以后接 OTel 会变成 trace_id。
- **panic 隔离**：一个坏请求不会打崩整个进程。
- **无状态**：目前没有任何进程内会话状态，为后面加副本做准备。

---

## 12. 下一步

Step 0.3：GitHub Actions CI（Linux 上跑 `go test -race`、`go vet`、`go build`、`gofmt` 检查），把质量门禁固定下来。

然后进入 Phase 1：LLM Gateway。你会学到：

- `http.Client`、context 超时、错误分类；
- OpenAI 兼容协议；
- SSE 流式解析；
- `tool_calls` 分片拼接；
- 重试、退避、限流、熔断；
- 用 `httptest` 和 fake provider 做可重复测试。
