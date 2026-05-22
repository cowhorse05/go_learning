# Go 项目从零到运行（小白向）

## 0. 前置：确认 Go 已安装

```bash
go version
# 正常输出类似：go version go1.22.5 linux/amd64
```

如果没有安装，去 https://go.dev/dl/ 下载安装。

---

## 1. 新建项目

Go 项目就是一个**文件夹**，里面放 `.go` 源文件。关键是要有一个 `go.mod` 文件来声明这个项目叫啥名。

### 步骤

```bash
# 1. 创建项目文件夹
mkdir my-project
cd my-project

# 2. 初始化 module（就是给项目起个名）
go mod init 项目名

# 例如：
go mod init hello-world
```

`go mod init` 执行后，文件夹里会多一个 `go.mod` 文件：

```
my-project/
├── go.mod    ← 自动生成的，记录项目名和依赖
└── main.go   ← 你自己写的代码
```

---

## 2. 写代码

新建 `main.go`，内容如下（最基本的 hello world）：

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
```

解释几个关键点：

| 概念 | 是什么 | 必须这样写吗 |
|------|--------|--------------|
| `package main` | 告诉 Go：这个文件属于 `main` 包 | **是**，可执行程序的入口必须在 `main` 包里 |
| `func main()` | 程序的入口函数，程序从这里开始执行 | **是**，有且只有一个 |
| `import "fmt"` | 引入标准库 `fmt`（格式化输出用） | 用到啥引啥 |

---

## 3. 运行代码

### 方式一：直接运行（日常开发用）

```bash
go run main.go
```

`go run` = 临时编译 + 立刻运行，**不会生成可执行文件**。改完代码立刻看效果，推荐。

如果要运行整个包（当项目有多个 `.go` 文件时）：

```bash
go run .
```

### 方式二：先编译再运行（发布部署用）

```bash
# 编译：生成一个叫 myapp 的可执行文件
go build -o myapp main.go

# 运行
./myapp
```

`go build` 会生成一个**二进制文件**，可以直接拿到别的机器上跑（同平台的话）。

---

## 4. 常用命令速查

| 命令 | 作用 | 什么时候用 |
|------|------|-----------|
| `go mod init 名字` | 初始化新项目 | 新建项目时，只跑一次 |
| `go run main.go` | 直接运行 | 开发调试 |
| `go run .` | 运行整个包 | 多个 `.go` 文件时 |
| `go build -o 输出名 main.go` | 编译成二进制文件 | 准备发布/部署 |
| `go build .` | 编译整个包 | 多个文件时 |
| `go mod tidy` | 自动整理依赖（下载缺失的、删掉多余的） | `go.mod` 里有报红的时候 |

---

## 5. 一个完整的实操例子

```bash
# 1. 创建项目
mkdir ~/hello-app
cd ~/hello-app

# 2. 初始化
go mod init hello-app

# 3. 写代码（用 vim/nano/vscode 创建 main.go）
cat > main.go << 'EOF'
package main

import "fmt"

func main() {
    fmt.Println("Hello, Go!")
}
EOF

# 4. 运行
go run main.go
# 输出: Hello, Go!

# 5. 编译
go build -o hello-app main.go

# 6. 运行编译后的文件
./hello-app
# 输出: Hello, Go!
```

---

## 6. 关于 go.mod 文件

初始化后长这样：

```
module hello-app    ← 项目名

go 1.22.5           ← Go 版本
```

当你 `import` 了第三方包（比如 `github.com/gin-gonic/gin`），运行 `go mod tidy` 后，`go.mod` 会自动记录依赖：

```
module hello-app

go 1.22.5

require github.com/gin-gonic/gin v1.9.1    ← 自动加上的
```

同时会生成 `go.sum`（校验文件，不用手动管它）。

---

## 7. 新手常见问题

**Q: `package xxx is not in GOROOT` 报错？**
A: 说明你 import 的包不存在或没下载。用 `go mod tidy` 自动下载。

**Q: 一个文件夹下能有多个 `main.go` 吗？**
A: 不能。一个文件夹下，所有 `.go` 文件的 `package` 声明必须一致。如果你想写多个独立的程序，需要分别放在不同的文件夹里。

**Q: `go run` vs `go build`，日常用哪个？**
A: 开发时用 `go run`，方便。要发布给别人用时用 `go build`，生成一个独立可执行文件。
