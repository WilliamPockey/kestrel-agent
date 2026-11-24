# Go 基础补课（Python 开发者版）

> 这份文档不追求覆盖 Go 的全部语法，只覆盖 Kestrel 项目会用到的 80%。
> 读法：每节先看最小例子，自己敲一遍；再看「和 Python 的区别」；最后看「项目落点」。
> 遇到看不懂的符号，先回来查，不要硬猜。

---

## 0. 先建立 5 个直觉

1. **Go 是编译型语言**：`go run` 先编译再运行；`go build` 只编译。
2. **Go 没有异常**：错误是普通返回值，必须显式检查 `if err != nil`。
3. **Go 的并发很便宜**：一个 goroutine 初始栈很小，可以开成千上万个；但共享内存仍然要加锁或用 channel。
4. **Go 的接口是隐式实现**：不需要 `implements`，只要方法集匹配就自动满足接口。
5. **Go 喜欢显式**：没有隐式类型转换，没有 `try/except`，没有构造函数语法糖，很多事要自己写清楚。

---

## 1. 工具链与项目结构

### 1.1 最常用的 5 条命令

```bash
go run ./cmd/server      # 编译并运行
go build ./...           # 编译所有包，不运行
go test ./...            # 跑测试
go vet ./...             # 静态检查
go fmt ./...             # 格式化
```

### 1.2 `go.mod`

```go
module kestrel

go 1.26.0
```

- `module` 是模块路径，也是 import 的前缀。
- 同模块内导入：`import "kestrel/internal/httpapi"`.
- 发布到 GitHub 前，把 module 改成 `github.com/<你的ID>/kestrel-agent`，并同步替换 import。

### 1.3 包（package）

- 一个目录 = 一个包；同一目录下所有 `.go` 文件的 `package` 名必须一致。
- `package main` + `func main()` 才能生成可执行程序。
- `internal/` 下的包只能被本模块导入，外部模块无法引用。

---

## 2. 变量、常量、类型

### 2.1 变量

```go
var name string = "kestrel" // 显式类型
var count = 3               // 类型推断
age := 18                   // 只能在函数内用，短变量声明
```

- `:=` 只能在函数内部使用。
- 同一个作用域里，`:=` 左边至少要有一个新变量。
- 未使用的局部变量会**编译报错**（不是警告）。

### 2.2 常量

```go
const MaxIterations = 10
const (
    StatusOK      = "ok"
    StatusFailed  = "failed"
)
```

常量在编译期确定，不能运行时赋值。

### 2.3 基本类型

```go
bool
string
int  int8 int16 int32 int64
uint uint8 uint16 uint32 uint64
float32 float64
byte   // uint8 的别名
rune   // int32 的别名，表示一个 Unicode 码点
```

- 没有隐式转换：`int` 和 `int64` 不能直接相加。
- 转换要显式：`int64(n)`、`int(x)`。
- `len("你好")` 返回字节数；`utf8.RuneCountInString("你好")` 返回字符数（rune 数）。

### 2.4 零值

声明但没赋值的变量有零值：

| 类型 | 零值 |
|---|---|
| 数值 | `0` |
| `string` | `""` |
| `bool` | `false` |
| 指针、接口、map、slice、channel、函数 | `nil` |

> Python 的 `None` 只有一个；Go 的 `nil` 可以表示多种「空」。

### 2.5 时间

```go
time.Second        // 1 秒
5 * time.Second    // 5 秒
time.Duration(1500) * time.Millisecond
```

`time.Duration` 本质是 `int64` 纳秒。写 `5 * time.Second` 比写 `5000000000` 清楚。

### 2.6 Python 对比

| Python | Go |
|---|---|
| `x = 1` | `x := 1`（函数内）或 `var x = 1` |
| `None` | `nil`（有限类型） |
| `int` 自动变大 | `int` 有固定位数；需要大数用 `math/big` |
| `str` 是 Unicode | `string` 是字节序列；`rune` 是字符 |
| `len(s)` 字符数 | `len(s)` 字节数 |

**项目落点**：`internal/config` 里的超时、端口、日志级别；`time.Duration` 会大量出现。

---

## 3. 函数、多返回值、错误

### 3.1 函数签名

```go
func add(a, b int) int {
    return a + b
}

func divide(a, b float64) (float64, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}
```

- 参数类型写在变量名后面。
- 多返回值很常见，尤其是 `(结果, error)`。
- 调用：

```go
result, err := divide(10, 2)
if err != nil {
    // 处理错误
}
```

### 3.2 错误不是异常

```go
f, err := os.Open("config.yaml")
if err != nil {
    return fmt.Errorf("open config: %w", err)
}
defer f.Close()
```

- `%w` 包装错误，保留错误链。
- `errors.Is(err, os.ErrNotExist)` 判断是否为某个哨兵错误。
- `errors.As(err, &target)` 判断是否为某个具体错误类型。
- 不要用 `err.Error() == "..."` 做判断。

### 3.3 命名返回值（先少用）

```go
func f() (n int, err error) {
    return 0, nil
}
```

命名返回值能让 `defer` 修改返回值，但可读性差。初学阶段显式 `return` 更清楚。

### 3.4 Python 对比

| Python | Go |
|---|---|
| `try/except` | `if err != nil` |
| `raise ValueError(...)` | `return fmt.Errorf(...)` |
| `finally` | `defer` |
| 异常会向上冒泡 | 错误必须逐层返回 |

**项目落点**：所有 IO、HTTP、数据库、LLM 调用都会返回 `error`；中间件用 `recover` 兜底 panic。

---

## 4. 复合类型：slice、map、struct

### 4.1 数组 vs 切片

```go
var arr [3]int          // 数组，长度是类型的一部分
nums := []int{1, 2, 3}  // 切片，最常用
nums = append(nums, 4)  // append 可能分配新底层数组
```

切片是「指针 + 长度 + 容量」：

```go
s := nums[1:3] // 左闭右开
len(s)         // 2
cap(s)         // 从起点到底层数组末尾的容量
```

常见坑：

```go
a := []int{1, 2, 3}
b := a[:2]
b[0] = 99
// a 也会变成 [99 2 3]，因为共享底层数组
```

需要独立副本时用 `append([]int(nil), a...)` 或 `slices.Clone(a)`（Go 1.21+）。

### 4.2 map

```go
m := map[string]int{"a": 1}
m["b"] = 2
v, ok := m["c"] // ok=false，v=0
delete(m, "a")

for k, v := range m {
    // 遍历顺序是随机的
}
```

- 声明但没 `make` 的 map 是 `nil`；读可以，写会 panic。
- map 不是并发安全的；并发读写要加锁或用 `sync.Map`（大多数场景更适合 `Mutex + map`）。

### 4.3 struct

```go
type User struct {
    ID       int64  `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email,omitempty"`
}
```

- 字段首字母大写才能被其他包访问，也才能被 `encoding/json` 导出。
- `` `json:"..."` `` 是 struct tag，控制 JSON 字段名。
- `omitempty` 表示零值时省略该字段。

构造：

```go
u := User{ID: 1, Username: "alice"}
p := &User{ID: 2} // 指针
```

### 4.4 方法

```go
func (u User) DisplayName() string {
    return u.Username
}

func (u *User) Rename(name string) {
    u.Username = name
}
```

- `(u User)` 是值接收者，方法内拿到副本。
- `(u *User)` 是指针接收者，可以修改原对象，也避免大结构体拷贝。
- 经验：需要修改状态、结构体较大、或者有其他指针接收者时，用指针接收者；否则值接收者也可以。**同一类型的方法最好统一。**

### 4.5 Python 对比

| Python | Go |
|---|---|
| `list` | slice |
| `dict` | map |
| `@dataclass` | struct + tag |
| `obj.method()` | `func (r Receiver) Method()` |
| `self` | 接收者，显式命名 |

**项目落点**：`llm.Message`、`agent.Event`、`config.Config` 都是 struct；消息列表、工具列表都是 slice；元数据用 map。

---

## 5. 指针

```go
x := 42
p := &x      // p 是 *int
fmt.Println(*p) // 42，解引用
*p = 100
fmt.Println(x)  // 100
```

- `&` 取地址，`*` 解引用。
- 指针零值是 `nil`；解引用 `nil` 会 panic。
- 函数传参默认是值拷贝；想修改原对象或避免大对象拷贝，传指针。

```go
func increment(n *int) { *n++ }
```

> Python 的变量更像「对象引用」，但没有显式指针。Go 把「值 / 指针」摆到台面上，需要你明确选择。

**项目落点**：`*http.Server`、`*slog.Logger`、`*sql.DB`、`*config.Config`。这些通常只创建一次，传指针避免拷贝。

---

## 6. 接口（Go 最重要的抽象）

### 6.1 隐式实现

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}

type File struct{}

func (f *File) Read(p []byte) (int, error) { return 0, nil }
```

`*File` 自动满足 `Reader`，不需要写 `implements`。

### 6.2 空接口与 `any`

```go
var v any = 42 // any 是 interface{} 的别名
```

`any` 可以装任何值，但取出时要类型断言：

```go
n, ok := v.(int)
if !ok {
    // 不是 int
}
```

类型 switch：

```go
switch x := v.(type) {
case int:
    fmt.Println("int", x)
case string:
    fmt.Println("string", x)
default:
    fmt.Println("other")
}
```

### 6.3 接口的最佳实践

- 接口由**使用方**定义，而不是实现方。需要什么方法就定义什么接口。
- 接口越小越好，`io.Reader` / `io.Writer` 只有一个方法。
- 返回具体类型，接收接口；构造函数可以返回接口。
- 不要为了「看起来像 Java」而提前抽接口；先写具体实现，出现第二个实现或测试需要时再抽。

### 6.4 项目落点

```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}

type Provider interface {
    Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

type Tool interface {
    Name() string
    Invoke(ctx context.Context, args json.RawMessage) (Result, error)
}
```

**面试题**：Go 接口和 Python 的 duck typing 有什么区别？
答：Python 运行时看有没有方法；Go 编译期检查方法集，性能更好，错误更早暴露。

---

## 7. 控制流

### 7.1 if

```go
if err := doSomething(); err != nil {
    return err
}
```

`if` 可以带一个初始化语句，变量作用域只在 `if/else` 内。这是 Go 很常见的写法。

### 7.2 for（Go 只有 for）

```go
for i := 0; i < 3; i++ { }
for i := range 3 { }          // Go 1.22+，等价于 0,1,2
for _, v := range nums { }    // 只要值
for i, v := range nums { }    // 索引和值
for { break }                 // 无限循环
```

没有 `while`，用 `for condition { }`。

### 7.3 switch

```go
switch status {
case "ok":
    // ...
case "failed", "timeout":
    // 多个值
default:
    // ...
}
```

Go 的 `switch` 默认不穿透，不需要 `break`；想穿透用 `fallthrough`（很少用）。

### 7.4 defer

```go
f, err := os.Open("x")
if err != nil { return err }
defer f.Close()
```

- `defer` 在函数返回时执行，后进先出（LIFO）。
- 参数在 `defer` 语句执行时求值，不是函数返回时。
- 常用来释放资源、解锁、关闭连接、记录耗时。

```go
func timed() {
    start := time.Now()
    defer func() {
        fmt.Println(time.Since(start))
    }()
    // ...
}
```

### 7.5 Python 对比

| Python | Go |
|---|---|
| `for x in xs` | `for _, x := range xs` |
| `while cond` | `for cond` |
| `try/finally` | `defer` |
| `match` | `switch` |

---

## 8. panic 与 recover

```go
func safe() {
    defer func() {
        if r := recover(); r != nil {
            fmt.Println("recovered:", r)
        }
    }()
    panic("boom")
}
```

- `panic` 用于「程序无法继续」的 bug，不是业务错误。
- `recover` 只能在 `defer` 的函数里生效，而且只能恢复**当前 goroutine** 的 panic。
- HTTP 中间件用 `recover` 防止一个请求打崩整个进程。
- 库代码不要随便 `panic`；把错误返回给调用方。

**项目落点**：`Recovery` 中间件；工具执行器的 panic 保护。

---

## 9. 并发（重点）

### 9.1 goroutine

```go
go func() {
    fmt.Println("running in background")
}()
```

- `go` 关键字启动一个 goroutine；它由 Go runtime 调度，不绑定 OS 线程。
- 主 goroutine 退出，整个程序就退出；不会等待其他 goroutine。
- goroutine 很便宜，但不是无限：每个都有栈和调度成本，泄漏会吃内存。

### 9.2 channel

```go
ch := make(chan int)     // 无缓冲：发送和接收必须同时准备好
ch <- 1                  // 发送
v := <-ch                // 接收

buf := make(chan int, 3) // 有缓冲：缓冲满之前发送不阻塞
```

关闭：

```go
close(ch)
v, ok := <-ch // ok=false 表示已关闭且没有数据
for v := range ch { } // 自动在关闭后退出
```

规则：

- 只有发送方关闭 channel。
- 向已关闭的 channel 发送会 panic。
- 从已关闭的 channel 接收会立刻返回零值。
- nil channel 发送/接收会永久阻塞（有时可用来禁用 select 分支）。

### 9.3 select

```go
select {
case v := <-ch:
    fmt.Println("got", v)
case <-time.After(time.Second):
    fmt.Println("timeout")
case <-ctx.Done():
    fmt.Println("canceled")
default:
    fmt.Println("no event")
}
```

- 多个 case 同时就绪时随机选一个。
- `default` 让 select 非阻塞。
- 常用于超时、取消、多路事件合并。

### 9.4 context

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
```

- `context.Context` 用来传递取消信号、截止时间、请求范围的值。
- 函数第一个参数通常是 `ctx context.Context`。
- `context.Background()` 是根；`WithCancel` / `WithTimeout` / `WithDeadline` 派生。
- 不把 `context` 存进 struct 字段；随请求传递。
- `context.Value` 只放请求级元数据（request id、user id），不要放业务依赖。

### 9.5 sync

```go
var mu sync.Mutex
mu.Lock()
// 临界区
mu.Unlock()

var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    // ...
}()
wg.Wait()
```

- `sync.Mutex`：互斥锁。
- `sync.RWMutex`：读多写少。
- `sync.WaitGroup`：等待一组 goroutine 结束；`Add` 必须在启动 goroutine 前调用。
- `sync.Once`：只执行一次（初始化）。
- `sync/atomic`：原子计数、标志位。

### 9.6 常见并发坑

1. **数据竞争**：多个 goroutine 同时读写同一个变量，没锁。用 `go test -race` 抓。
2. **goroutine 泄漏**：启动了 goroutine，但 channel 永远没人读/写，导致永久阻塞。
3. **循环变量捕获**：Go 1.22 之前，`for` 变量是复用的；闭包会共享同一个变量。Go 1.22+ 每次迭代新变量，但仍建议显式传参。
4. **关闭已关闭的 channel**：panic。
5. **在有缓冲 channel 满时阻塞**：消费者慢，生产者卡死；需要背压或丢弃策略。
6. **锁内做 IO**：持锁调用网络/数据库，导致其他 goroutine 长时间等待。

### 9.7 Python 对比

| Python | Go |
|---|---|
| `asyncio` / `await` | goroutine + channel + context |
| `threading.Lock` | `sync.Mutex` |
| `queue.Queue` | channel |
| `concurrent.futures` | goroutine + WaitGroup / errgroup |
| `asyncio.timeout` | `context.WithTimeout` |

**项目落点**：HTTP 服务每个请求一个 goroutine；SSE 事件流用 channel；worker pool 用 goroutine + 有界队列；优雅退出用 context + WaitGroup。

---

## 10. 测试

### 10.1 基本测试

```go
func TestAdd(t *testing.T) {
    got := add(1, 2)
    if got != 3 {
        t.Fatalf("add(1,2) = %d, want 3", got)
    }
}
```

- 测试文件 `*_test.go`。
- 函数名 `TestXxx(t *testing.T)`。
- `t.Fatalf` 立即停止当前测试；`t.Errorf` 继续。

### 10.2 表驱动

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name string
        a, b int
        want int
    }{
        {"positive", 1, 2, 3},
        {"negative", -1, -2, -3},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := add(tt.a, tt.b); got != tt.want {
                t.Fatalf("got %d, want %d", got, tt.want)
            }
        })
    }
}
```

### 10.3 httptest

```go
req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
rec := httptest.NewRecorder()
handler.ServeHTTP(rec, req)
```

不需要真实端口，不起网络。

### 10.4 其他命令

```bash
go test -v ./...
go test -run TestHealthz/GET -v ./internal/httpapi
go test -race ./...          # 需要 cgo；Windows 无 gcc 时放 CI/WSL2
go test -cover ./...
go test -bench=. -benchmem
go test -fuzz=FuzzParse -fuzztime=10s
```

### 10.5 测试原则

- 每个 bug 先写失败测试。
- 外部依赖用 fake；测试不依赖网络/API Key。
- 测错误路径，不只测 happy path。
- 不 sleep 等异步结果；用 channel、条件变量、可注入时钟。

---

## 11. JSON

```go
type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
    Name    string `json:"name,omitempty"`
}

b, err := json.Marshal(m)       // 编码成 []byte
err = json.Unmarshal(b, &m)     // 解码到结构体
err = json.NewEncoder(w).Encode(m) // 直接写 writer，末尾带换行
err = json.NewDecoder(r.Body).Decode(&m)
```

- 只有大写字段会被导出。
- `json.RawMessage` 可以延迟解析嵌套 JSON。
- `json.Decoder` 适合流式读取；`json.Unmarshal` 适合已有 `[]byte`。
- 解码未知字段默认忽略；需要严格校验时用 `DisallowUnknownFields`。

**项目落点**：OpenAI 协议、工具参数、SSE 事件、配置文件。

---

## 12. 泛型（先会用，不滥用）

```go
func Map[T, U any](xs []T, f func(T) U) []U {
    out := make([]U, 0, len(xs))
    for _, x := range xs {
        out = append(out, f(x))
    }
    return out
}
```

- 类型参数写在方括号里。
- 约束用 `any`、`comparable`、自定义 interface。
- 不要为了炫技而泛型；普通接口/具体类型更清楚。

---

## 13. 项目里的 Go 概念地图

| 代码位置 | 用到的 Go 知识 |
|---|---|
| `cmd/server/main.go` | package main、struct、指针、goroutine、signal、context |
| `internal/config` | struct、方法、错误、环境变量、time.Duration |
| `internal/httpapi` | interface、handler、中间件、JSON、httptest |
| `internal/llm` | interface、http.Client、context、channel、SSE |
| `internal/tool` | interface、JSON Schema、panic/recover、超时 |
| `internal/agent` | goroutine、channel、select、context、状态机 |
| `internal/store` | database/sql、事务、错误处理、分页 |
| `internal/graph` | map、状态机、并发、checkpoint |
| `internal/worker` | goroutine、channel、WaitGroup、context、幂等 |

---

## 14. 练习顺序（配合项目）

1. 用 `for` + `slice` 写一个去重函数，并写表驱动测试。
2. 用 `map` 统计一段文本的词频，按次数排序输出。
3. 定义一个 `Shape` 接口，实现 `Circle` 和 `Rectangle`，写测试。
4. 用 `error` + `fmt.Errorf("%w")` 包装错误，用 `errors.Is` 判断。
5. 启动 3 个 goroutine，用 channel 汇总结果；用 `context.WithTimeout` 取消。
6. 用 `sync.Mutex` 保护一个计数器，开 100 个 goroutine 累加，跑 `-race`。
7. 用 `httptest` 测一个 handler 的 200/400/405。
8. 用 `json.Marshal/Unmarshal` 处理一个带 `omitempty` 的结构体。

---

## 15. 常见编译错误速查

| 报错 | 原因 | 解决 |
|---|---|---|
| `undefined: xxx` | 变量/函数未定义，或首字母小写导致包外不可见 | 检查拼写、作用域、导出规则 |
| `declared and not used` | 局部变量声明未使用 | 删除或使用 `_` |
| `cannot use x (type int) as type int64` | 没有隐式转换 | 显式 `int64(x)` |
| `missing return` | 有返回值函数存在未返回的分支 | 补 `return` |
| `invalid operation: ... (mismatched types)` | 类型不一致 | 转换类型 |
| `panic: assignment to entry in nil map` | map 未初始化 | `make(map[K]V)` |
| `fatal error: all goroutines are asleep - deadlock!` | channel 互相等待 | 检查发送/接收配对、缓冲、close |
| `http: superfluous response.WriteHeader call` | 重复写状态码 | 只写一次 |
| `-race requires cgo` | Windows 没有 C 工具链 | 用 Linux CI / WSL2，或安装 gcc |

---

## 16. 一句话总结

> Go 的难点不是语法，而是：**错误处理、接口设计、并发与生命周期**。
> 这三件事会在 Kestrel 里反复出现；每写一个包，都问自己：
> 1. 错误往哪里返回？
> 2. 这个接口由谁使用、需要几个方法？
> 3. 谁启动 goroutine，谁负责让它退出？
