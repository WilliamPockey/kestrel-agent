# Step 0：把项目跑起来（Walking Skeleton）

> 目标不是「写很多功能」，而是先建立一个**最小可运行、可测试、可解释**的 Go 服务。
> 完成后你应该能回答：一个 HTTP 请求从 `main` 到返回 JSON，中间经过了什么？

---

## 1. 本步目标

- 建立 Go module 和目录结构。
- 写一个 `GET /healthz` 接口。
- 写第一个表驱动测试。
- 跑通 `go run`、`go test`、`go vet`、`go build`。
- 理解 Go 的 package、handler、interface、JSON、测试的基本写法。

---

## 2. 目录

```
kestrel-agent/
├── go.mod
├── .gitignore
├── Makefile
├── README.md
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   └── httpapi/
│       ├── router.go
│       └── router_test.go
└── docs/
    ├── IMPLEMENTATION_PLAN.md
    └── step-00.md
```

两个新目录的含义：

- `cmd/`：可执行程序入口。一个项目可以有多个 `main`（server、cli、worker、eval）。
- `internal/`：只允许本 module 内部 import 的包。Go 会在编译期阻止外部 module 引用它，适合放业务代码。

---

## 3. 先理解 7 个概念

### 3.1 module、package、import

- **module**：一个版本化的代码集合，由 `go.mod` 定义。我们的 module 名是 `kestrel`。
- **package**：同一目录下的 `.go` 文件属于同一个包。包名写在每个文件第一行。
- **import**：按 **module path + 目录** 导入，不是按文件名。

`cmd/server/main.go` 导入：

```go
import "kestrel/internal/httpapi"
```

意思是：导入 `kestrel` 这个 module 下 `internal/httpapi` 目录的包。包名是 `httpapi`，调用时写 `httpapi.NewRouter()`。

> 和 Python 的区别：Python 的 `import` 更接近文件/模块；Go 的 import 是包路径。一个包可以由多个文件组成。

### 3.2 `package main` 和 `func main`

```go
package main

func main() {
    // 程序入口
}
```

- 只有 `package main` 且包含 `func main()` 的包，才能被编译成可执行程序。
- `httpapi` 是普通库包，不能直接 `go run`。

### 3.3 导出规则：首字母大小写

Go 没有 `public/private` 关键字，靠首字母：

- `NewRouter`、`ServeHTTP`：大写，**导出**，包外可见。
- `handleHealthz`、`writeJSON`：小写，**包内私有**。

这是 Go 的硬规则，不是风格建议。

### 3.4 `http.Handler` 和 `http.ServeMux`

`net/http` 的核心抽象是一个接口：

```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
```

任何类型，只要实现了 `ServeHTTP` 方法，就是一个 handler。`http.ServeMux` 也实现了它，所以可以层层嵌套。

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /healthz", handleHealthz)
return mux
```

- `"GET /healthz"` 是 Go 1.22+ 的方法路由语法。
- `GET` 模式也会匹配 `HEAD` 请求。
- 同一个路径收到未注册的方法（比如 `POST`）时，`ServeMux` 会自动返回 `405 Method Not Allowed`，并带 `Allow` 头。

> 和 Python 的区别：Flask/FastAPI 用装饰器注册路由；Go 这里是显式调用 `HandleFunc`。没有魔法，容易追踪。

### 3.5 `http.ResponseWriter` 和 JSON

`http.ResponseWriter` 是接口，负责写状态码、响应头和 body：

```go
w.Header().Set("Content-Type", "application/json; charset=utf-8")
w.WriteHeader(http.StatusOK)
_ = json.NewEncoder(w).Encode(v)
```

顺序很重要：

1. 先设置 header。
2. 再 `WriteHeader`。
3. 最后写 body。

一旦 body 开始写，再改状态码就无效了。

`json.NewEncoder(w).Encode(v)` 会把 Go 值编码成 JSON 并写入 `w`，末尾带一个换行。`any` 是 `interface{}` 的别名（Go 1.18+）。

### 3.6 `go test` 和 `httptest`

```go
req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
rec := httptest.NewRecorder()
NewRouter().ServeHTTP(rec, req)
```

- `httptest.NewRequest` 构造一个请求对象，不经过真实网络。
- `httptest.NewRecorder` 实现了 `http.ResponseWriter`，把响应记录在内存里。
- 直接调用 `ServeHTTP`，就能在测试里跑完整的 handler 逻辑。

表驱动测试的结构：

```go
tests := []struct{ name, method string; wantStatus int }{ ... }

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // 用 tt 跑一遍
    })
}
```

好处：新增用例只加一行，不用复制测试函数。

### 3.7 错误处理

Go 没有异常，错误是返回值：

```go
if err := srv.ListenAndServe(); err != nil {
    log.Fatal(err)
}
```

- `err != nil` 表示出错。
- `log.Fatal` 打印并 `os.Exit(1)`。
- 后续我们会用 `errors.Is` / `errors.As` / `%w` 做错误链，而不是字符串比较。

> 和 Python 的区别：Go 用返回值显式处理错误；`defer` 类似 `finally`，但没有 `try/except` 的隐式跳转。

---

## 4. 代码逐行

### 4.1 `go.mod`

```go
module kestrel

go 1.26.0
```

- `module kestrel`：本地 module 名。发布到 GitHub 前可以改成 `github.com/<你的ID>/kestrel-agent`，并同步替换 import。
- `go 1.26.0`：语言版本。环境是 Go 1.26.5，满足要求。

### 4.2 `cmd/server/main.go`

```go
package main

import (
    "log"
    "net/http"
    "time"

    "kestrel/internal/httpapi"
)

func main() {
    srv := &http.Server{
        Addr:              ":8080",
        Handler:           httpapi.NewRouter(),
        ReadHeaderTimeout: 5 * time.Second,
    }

    log.Printf("kestrel listening on %s", srv.Addr)
    if err := srv.ListenAndServe(); err != nil {
        log.Fatal(err)
    }
}
```

关键点：

- `&http.Server{...}` 创建结构体指针。`&` 取地址，等价于 `new` + 初始化。
- `Addr: ":8080"` 表示监听所有网卡的 8080 端口。
- `Handler` 接收任意 `http.Handler`；`NewRouter()` 返回的 `ServeMux` 满足它。
- `ReadHeaderTimeout` 防止慢速请求头攻击（Slowloris）。生产服务还应该加 `ReadTimeout`、`WriteTimeout`、`IdleTimeout`。
- `ListenAndServe` 是阻塞调用；`net/http` 会为每个连接/请求启动 goroutine。现在先不用自己写 goroutine，Step 1 会正式讲。
- `log.Fatal` 只适合 `main`。库代码不要 `Fatal`，要把 error 返回给调用方。

### 4.3 `internal/httpapi/router.go`

```go
package httpapi

import (
    "encoding/json"
    "net/http"
)

func NewRouter() http.Handler {
    mux := http.NewServeMux()
    mux.HandleFunc("GET /healthz", handleHealthz)
    return mux
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
    writeJSON(w, http.StatusOK, map[string]string{
        "status":  "ok",
        "service": "kestrel",
    })
}

func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(v)
}
```

关键点：

- `NewRouter` 返回接口 `http.Handler`，而不是具体类型 `*http.ServeMux`。调用方只依赖接口，后续换路由实现不影响上层。
- `_ *http.Request`：这个 handler 不需要请求内容，用 `_` 明确表示忽略。
- `map[string]string` 编码后 key 按字典序输出，所以看到的是 `{"service":"kestrel","status":"ok"}`。
- `writeJSON` 目前忽略编码错误（`_ =`）。对于这个固定的 map，编码不会失败；后面处理复杂结构时会改成先 `json.Marshal` 再写，避免半截响应。

### 4.4 `internal/httpapi/router_test.go`

```go
func TestHealthz(t *testing.T) {
    tests := []struct {
        name       string
        method     string
        wantStatus int
    }{
        {name: "GET returns 200", method: http.MethodGet, wantStatus: http.StatusOK},
        {name: "POST returns 405", method: http.MethodPost, wantStatus: http.StatusMethodNotAllowed},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(tt.method, "/healthz", nil)
            rec := httptest.NewRecorder()

            NewRouter().ServeHTTP(rec, req)

            if rec.Code != tt.wantStatus {
                t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
            }
            if tt.wantStatus != http.StatusOK {
                return
            }

            var body map[string]string
            if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
                t.Fatalf("decode body: %v", err)
            }
            if body["status"] != "ok" {
                t.Fatalf("body.status = %q, want %q", body["status"], "ok")
            }
        })
    }
}
```

关键点：

- 测试文件必须以 `_test.go` 结尾，包名可以和被测包相同（`package httpapi`），这样能访问包内私有函数。
- `t.Run` 创建子测试，失败时能定位到具体用例。
- `t.Fatalf` 会立即终止当前子测试；`t.Errorf` 会继续执行。
- `%d` 打印整数，`%q` 打印带引号的字符串，调试时比 `%v` 更清楚。
- `POST /healthz` 的 405 是 `ServeMux` 自动产生的，不需要我们写分支。

---

## 5. 运行

在 `kestrel-agent` 目录下：

```bash
# 1. 编译并运行
go run ./cmd/server
```

另开一个终端：

```bash
curl http://localhost:8080/healthz
```

期望：

```json
{"service":"kestrel","status":"ok"}
```

再试一个错误方法：

```bash
curl -i -X POST http://localhost:8080/healthz
```

期望状态行包含 `405 Method Not Allowed`。

停止服务：在运行 `go run` 的终端按 `Ctrl+C`。

> 当前版本按 `Ctrl+C` 会直接退出，还没有优雅关闭。Step 1 会加 `context` + `signal.NotifyContext` + `Server.Shutdown`。

---

## 6. 测试与检查

```bash
go test ./...
go test -v ./...
go test -run TestHealthz/GET -v ./internal/httpapi
go vet ./...
go fmt ./...
go build ./...
```

- `go test ./...`：递归测试所有包。
- `-run` 支持正则，`TestHealthz/GET` 只跑 GET 子测试。
- `go vet`：静态检查，能发现格式化字符串、可疑锁拷贝等问题。
- `go fmt`：按官方格式重写文件；提交前必须干净。
- `go build ./...`：只编译，不运行。
- `go test -race ./...`：竞态检测。Windows 默认没有 C 工具链时会报 `-race requires cgo`；本地先用普通测试，race 放到 Linux CI / WSL2 跑。CI 会从 Step 0.3 开始配置。

---

## 7. 验收标准

- [ ] `go test ./...` 全部通过。
- [ ] `go test -race ./...` 通过（Step 1 起 CI 必跑）。
- [ ] `go vet ./...` 无输出。
- [ ] `go run ./cmd/server` 能启动。
- [ ] `GET /healthz` 返回 200 + JSON。
- [ ] `POST /healthz` 返回 405。
- [ ] 你能解释：`http.Handler` 是什么、`ServeMux` 为什么能返回 405、`httptest` 为什么不需要真实网络。
- [ ] 你能解释：`NewRouter` 为什么返回接口而不是 `*http.ServeMux`。

---

## 8. 练习（先自己写，再看下一步）

1. 增加 `GET /version`，返回：

   ```json
   {"version":"dev"}
   ```

   并写测试覆盖 200、JSON 字段、`Content-Type`。

2. 给 `/healthz` 的测试增加一条断言：`Content-Type` 必须是 `application/json; charset=utf-8`。

3. 把 `handleHealthz` 的状态码临时改成 `503`，观察哪个测试失败；改回来。

4. 用 `go test -v` 和 `go test -run TestHealthz/GET -v` 各跑一次，观察输出差异。

5. 不看代码，口头回答：
   - 为什么 `POST /healthz` 会返回 405，而不是 404？
   - `http.ResponseWriter` 是接口还是结构体？
   - `json.NewEncoder(w).Encode` 和 `json.Marshal` 的区别是什么？

---

## 9. 常见错误

| 现象 | 原因 | 解决 |
|---|---|---|
| `package kestrel/internal/httpapi is not in std` | 在错误的目录运行，或 module 名不一致 | 确认在 `kestrel-agent` 根目录，`go.mod` 的 module 是 `kestrel` |
| `http: superfluous response.WriteHeader call` | 重复调用 `WriteHeader` | 只写一次；在写 body 前写 |
| `cannot use mux (variable of type *http.ServeMux) as http.Handler` | 方法集不匹配 | `*http.ServeMux` 实现了 `ServeHTTP`，正常可用；检查是否返回了错误类型 |
| `no required module provides package` | import 路径拼错 | 目录名、包名、module 名三者对齐 |
| 端口被占用 | 8080 已有进程 | 换端口或停掉旧进程 |

---

## 10. 下一步：Step 0.2

下一步会引入这些 Go 核心概念：

- `struct` + 方法 + 构造函数
- 环境变量配置与校验
- `log/slog` 结构化日志
- 中间件：request ID、panic recovery、访问日志
- `goroutine`、`channel`、`sync.WaitGroup`
- `context`、`signal.NotifyContext`
- `http.Server.Shutdown` 优雅退出
- `go test -race`

完成后，服务会从「能返回 JSON」升级为「像一个真正的后端服务」。
