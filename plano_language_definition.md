# plano 语言定义草案 V0.2

> 实现状态：当前仓库里已经有第一版可运行实现，但它只覆盖这份草案的一部分。  
> 当前支持范围与差异请看 [docs/implementation_status.md](./docs/implementation_status.md)。

> 定位：plano 是一门 **可嵌入、可扩展、强类型、schema-driven 的脚本 DSL 语言**。  
> 它的核心目标不是写死某一种构建脚本或基础设施描述格式，而是提供一套语言内核，让宿主可以注册自己的 DSL 关键字、表单、类型、函数、校验规则和 lowering 逻辑。

---

## 1. 设计目标

plano 的目标是成为一个可嵌入语言内核，用 Go 实现，用于承载不同宿主的 DSL。

它可以支撑：

- 构建脚本 DSL
- 基础设施任务 DSL
- CI Pipeline DSL
- 轻量编排 DSL
- 部署描述 DSL
- 风控规则 DSL
- 缓存策略 DSL

语言核心只提供通用能力：

- 词法分析
- 语法解析
- AST 构建
- 符号绑定
- 类型检查
- 表达式求值
- typed document 生成
- schema 驱动的 DSL 校验

具体业务概念由宿主注册，例如：

- `workspace`
- `task`
- `plugin`
- `toolchain`
- `go.binary`
- `go.test`
- `service`
- `job`
- `volume`
- `network`
- `pipeline`
- `stage`

这些不应该是语言内核写死的关键字，而应该是宿主注册的 **soft keyword / form**。

---

## 2. 核心理念

### 2.1 少量硬关键字，大量宿主软关键字

plano 语言保留少量硬关键字：

```text
import
const
let
fn
return
if
else
for
in
true
false
null
```

除此之外，DSL 领域词汇都应由宿主注册。

例如 bu1ld 宿主可以注册：

```text
workspace
task
plugin
toolchain
run
go.test
go.binary
```

基础设施宿主可以注册：

```text
service
job
network
volume
resources
env
healthcheck
placement
```

CI 宿主可以注册：

```text
pipeline
stage
step
matrix
cache
artifact
```

### 2.2 宿主注册 Form，而不是修改语言内核

plano 不建议支持动态 lexer keyword。也就是说，`task`、`service`、`pipeline` 这类词在词法阶段都只是普通 `Ident`。

解析器统一把它们解析为：

```text
FormDecl
```

然后在 binder/typechecker 阶段通过宿主注册的 `FormSpec` 判断其是否合法。

例如：

```plano
task build {
  deps = [test]
}
```

语法层只解析为：

```text
FormDecl {
  head  = task
  label = build
  body  = [...]
}
```

语义层再根据宿主注册信息判断：

```text
task 是合法 form
build 是 task symbol
deps 字段类型是 list<ref<task>>
test 必须是已定义 task ref
```

### 2.3 类似 Kotlin Script DSL 的思路

Kotlin DSL 的核心不是让用户自定义 Kotlin 语法，而是通过宿主 API、receiver scope、函数和属性提供 DSL 体验。

plano 借鉴这个思路：

- 语言语法保持稳定
- 宿主注册 DSL form
- form 可以像关键字一样使用
- schema 提供补全、校验、文档和 lowering

因此 plano 的扩展点不是“让宿主改 parser 语法”，而是：

```text
宿主注册语言 Form
```

---

## 3. 基础示例

### 3.1 bu1ld 构建脚本示例

```plano
workspace {
  name = "bu1ld"
  default = build
}

plugin go {
  source = builtin
  id = "builtin.go"
}

const goVersion: string = env("GO_VERSION", "1.26.2")
const target: string = os + "/" + arch

toolchain go {
  version = goVersion
  settings = {
    mode = "module"
    platform = target
  }
}

task prepare {
  outputs = ["dist"]

  run {
    exec("mkdir", "-p", "dist")
  }
}

go.test test {
  packages = ["./..."]
}

go.binary build {
  deps = [prepare, test]
  main = "./cmd/cli"
  out = "dist/bu1ld"
}

import "tasks/*.plano"
```

这里：

- `workspace` 是宿主注册的 form
- `plugin` 是宿主注册的 form
- `toolchain` 是宿主注册的 form
- `task` 是宿主注册的 form
- `go.test` / `go.binary` 是插件注册的 form
- `prepare`、`test`、`build` 是 `task` 类型的 symbol
- `deps = [prepare, test]` 的类型是 `list<ref<task>>`

---

## 4. 词法定义

### 4.1 注释

支持单行注释：

```plano
// this is a comment
```

支持块注释：

```plano
/*
 multi line comment
*/
```

### 4.2 标识符

```ebnf
Ident = Letter { Letter | Digit | "_" | "-" }
```

示例：

```plano
task build {}
health-check {}
go.binary build {}
```

建议允许 `-`，方便 DSL 语义表达，例如 `health-check`、`pre-build`。

### 4.3 字符串

```ebnf
String = '"' { Character | Escape } '"'
```

示例：

```plano
"hello"
"dist/bu1ld"
"registry.local/api:latest"
```

### 4.4 数字

```ebnf
Int   = Digit { Digit }
Float = Digit { Digit } "." Digit { Digit }
```

示例：

```plano
1
100
3.14
```

### 4.5 时间字面量

```ebnf
Duration     = Int DurationUnit
DurationUnit = "ms" | "s" | "m" | "h"
```

示例：

```plano
500ms
10s
5m
1h
```

### 4.6 容量字面量

```ebnf
Size     = Int SizeUnit
SizeUnit = "B" | "Ki" | "Mi" | "Gi" | "Ti"
```

示例：

```plano
512Mi
2Gi
10Gi
```

---

## 5. 顶层语法定义

```ebnf
File        = { Statement }

Statement   = CoreStatement
            | FormDecl

CoreStatement
            = ImportDecl
            | ConstDecl
            | LetDecl
            | FnDecl
            | ReturnStmt
            | IfStmt
            | ForStmt

ImportDecl  = "import" String

ConstDecl   = "const" Ident [ ":" Type ] "=" Expr

LetDecl     = "let" Ident [ ":" Type ] "=" Expr

FnDecl      = "fn" Ident "(" [ Params ] ")" [ ":" Type ] BlockBody

Params      = Param { "," Param }

Param       = Ident ":" Type

ReturnStmt  = "return" Expr

IfStmt      = "if" Expr BlockBody [ "else" BlockBody ]

ForStmt     = "for" Ident "in" Expr BlockBody

BlockBody   = "{" { Statement } "}"

FormDecl    = FormHead [ Label ] FormBody

FormHead    = QualifiedIdent

QualifiedIdent
            = Ident { "." Ident }

Label       = Ident | String

FormBody    = "{" { FormItem } "}"

FormItem    = Assignment
            | FormDecl
            | CallStmt
            | CoreStatement

Assignment  = Ident "=" Expr

CallStmt    = QualifiedIdent "(" [ Expr { "," Expr } ] ")"

Expr        = OrExpr
```

---

## 6. 类型语法定义

```ebnf
Type        = SimpleType
            | ListType
            | MapType
            | RefType
            | QualifiedIdent

SimpleType  = "string"
            | "int"
            | "float"
            | "bool"
            | "duration"
            | "size"
            | "path"
            | "any"

ListType    = "list" "<" Type ">"

MapType     = "map" "<" Type ">"

RefType     = "ref" "<" QualifiedIdent ">"
```

示例：

```plano
const name: string = "bu1ld"
const timeout: duration = 10s
const memory: size = 512Mi
const packages: list<string> = ["./..."]
const targetTask: ref<task> = build
```

宿主可以定义类型别名：

```text
TaskRef = ref<task>
ServiceRef = ref<service>
ToolchainRef = ref<toolchain>
```

---

## 7. 表达式语法定义

```ebnf
Expr        = OrExpr

OrExpr      = AndExpr { "||" AndExpr }

AndExpr     = EqualityExpr { "&&" EqualityExpr }

EqualityExpr
            = CompareExpr { ( "==" | "!=" ) CompareExpr }

CompareExpr = AddExpr { ( ">" | ">=" | "<" | "<=" ) AddExpr }

AddExpr     = MulExpr { ( "+" | "-" ) MulExpr }

MulExpr     = UnaryExpr { ( "*" | "/" | "%" ) UnaryExpr }

UnaryExpr   = [ "!" | "-" ] PostfixExpr

PostfixExpr = PrimaryExpr { Selector | Index | CallSuffix }

Selector    = "." Ident

Index       = "[" Expr "]"

CallSuffix  = "(" [ Expr { "," Expr } ] ")"

PrimaryExpr = String
            | Int
            | Float
            | Bool
            | Duration
            | Size
            | Null
            | Ident
            | ArrayExpr
            | ObjectExpr
            | "(" Expr ")"

ArrayExpr   = "[" [ Expr { "," Expr } [ "," ] ] "]"

ObjectExpr  = "{" [ ObjectEntry { "," ObjectEntry } [ "," ] ] "}"

ObjectEntry = Ident "=" Expr

Bool        = "true" | "false"

Null        = "null"
```

脚本循环支持可选过滤子句：

```ebnf
ForStmt     = "for" [ Ident "," ] Ident "in" Expr [ "where" Expr ] Block
```

`where` 表达式在循环作用域内求值，因此可以引用当前循环变量；它必须是 `bool`，为 `false` 时跳过本轮循环。

示例：

```plano
os + "/" + arch

env("GO_VERSION", "1.26.2")

["cmd/**", "internal/**", "go.mod", "go.sum"]

{
  mode = "module"
  platform = os + "/" + arch
}

replicas > 1 && env("CI", "false") == "true"
```

### 7.1 Expr-lang 桥接表达式

实现层还提供一个显式的 expr-lang 桥接入口：

```plano
expr("slug(prefix + '/' + branch)")
expr_eval("dir + '/' + name", { dir = "dist", name = "app" })
```

这不是替换 plano 自身表达式语法，而是给宿主和用户一个 opt-in 的动态表达式入口。`expr(...)` / `expr_eval(...)` 的运行环境包含：

- 宿主通过 `RegisterExprVar` 注册的变量
- 宿主通过 `RegisterExprFunc` / `RegisterExprFunction` 注册的函数
- plano 全局常量
- 已解析的顶层 `const`
- 当前 script 作用域里的局部变量
- 第二个参数传入的 object override

---

## 8. Form 模型

### 8.1 Form 是宿主 DSL 的基本扩展单元

plano 不直接使用 `BlockDecl` 作为核心抽象，而使用更通用的 `FormDecl`。

例如：

```plano
task build {
  deps = [test]
}
```

语法层只认为它是：

```text
FormDecl
  head  = task
  label = build
  body  = [...]
```

语义层再根据宿主注册的 `FormSpec` 解释。

### 8.2 Qualified Form

插件可以注册带命名空间的 form：

```plano
go.binary build {
  deps = [test]
  main = "./cmd/cli"
  out = "dist/bu1ld"
}
```

这里：

```text
head = go.binary
label = build
```

`go.binary` 由 `go` 插件注册。

### 8.3 无 Label Form

```plano
workspace {
  name = "bu1ld"
  default = build
}
```

如果 `workspace` 的 schema 定义为 `NoLabel`，用户写：

```plano
workspace main {
}
```

则类型检查阶段报错：

```text
workspace does not accept label
```

---

## 9. FormSpec 设计

宿主通过 `FormSpec` 注册 soft keyword / form。

概念结构：

```go
type FormSpec struct {
    Name        QName
    Label       LabelSpec
    AllowedAt   PlacementSpec
    BodyMode    BodyMode
    Fields      []FieldSpec
    NestedForms []QName
    Declares    *SymbolSpec
    Lowerer     Lowerer
    Docs        string
}
```

### 9.1 LabelSpec

```text
NoLabel
SymbolLabel(kind)
StringLabel
TypedLabel(type)
```

示例：

```plano
workspace {
}
```

`workspace` 使用 `NoLabel`。

```plano
task build {
}
```

`task` 使用 `SymbolLabel("task")`。

```plano
stage "build-and-test" {
}
```

`stage` 可以使用 `StringLabel`。

### 9.2 BodyMode

不同 form 的 body 模式不同：

```text
FieldOnlyBody      只允许字段
FormOnlyBody       只允许嵌套 form
MixedBody          字段 + 嵌套 form
CallOnlyBody       只允许调用语句
ScriptBody         允许语句、if、for、let 等脚本能力
```

示例：

```plano
workspace {
  name = "demo"
  default = build
}
```

`workspace` 可以是 `FieldOnlyBody`。

```plano
task build {
  deps = [test]

  run {
    exec("go", "build", "./...")
  }
}
```

`task` 可以是 `MixedBody`。

```plano
run {
  exec("go", "build", "./...")
  shell("echo done")
}
```

`run` 可以是 `CallOnlyBody`。

---

## 10. FieldSpec 设计

字段由宿主 schema 定义。

概念结构：

```go
type FieldSpec struct {
    Name     string
    Type     Type
    Required bool
    Default  Value
    Docs     string
}
```

示例：`task` form：

```text
form task:
  label: symbol<task>
  fields:
    deps: list<ref<task>> = []
    inputs: list<path> = []
    outputs: list<path> = []
  nested:
    run
```

示例：`go.binary` form：

```text
form go.binary:
  label: symbol<task>
  fields:
    deps: list<ref<task>> = []
    main: path required
    out: path required
```

---

## 11. Symbol 与 Ref

### 11.1 Symbol 定义

带 label 的 form 可以声明 symbol。

```plano
task prepare {
}

task build {
}
```

产生：

```text
prepare: ref<task>
build: ref<task>
```

### 11.2 Symbol 引用

```plano
task build {
  deps = [prepare]
}
```

这里 `prepare` 不是字符串，而是符号引用。

如果未定义：

```plano
task build {
  deps = [missing]
}
```

报错：

```text
undefined symbol "missing"
```

如果类型不匹配：

```plano
service api {
}

task build {
  deps = [api]
}
```

报错：

```text
field "deps" expects list<ref<task>>, got list<ref<service>>
```

---

## 12. 函数系统

函数由 runtime 注册，也可以后续支持用户定义。

内置函数示例：

```text
env(name: string, fallback?: string): string
concat(...string): string
basename(path): string
dirname(path): string
join_path(...string): path
```

运行时常量：

```text
os: string
arch: string
```

示例：

```plano
const target = os + "/" + arch
const version = env("GO_VERSION", "1.26.2")
```

函数需要声明副作用：

```text
pure
reads_env
reads_file
exec
network
```

例如：

```text
env(...)       reads_env
basename(...)  pure
join_path(...) pure
```

这样可以支持配置缓存和增量编译。

---

## 13. Run 不是硬编码关键字

`run` 不应该是 parser 写死的特殊结构，而应该是宿主注册的 form。

```plano
task prepare {
  run {
    exec("mkdir", "-p", "dist")
  }
}
```

语法层解析为：

```text
FormDecl head=task label=prepare
  FormDecl head=run
    CallStmt exec(...)
```

语义层根据 schema 判断：

```text
task 允许嵌套 run
run 使用 CallOnlyBody
exec(...) 返回 Action
```

这样其他宿主也可以定义自己的类似结构：

```plano
stage build {
  steps {
    shell("go test ./...")
  }
}
```

---

## 14. Import 与模块系统

```plano
import "tasks/go.plano"
import "tasks/**/*.plano"
```

规则：

1. 相对路径基于当前文件。
2. 是否支持 glob 由宿主或 ImportResolver 决定。
3. import 文件共享同一个 module scope。
4. import 结果参与配置缓存指纹。
5. import 循环应报错。

示例错误：

```text
import cycle detected: a.plano -> b.plano -> a.plano
```

ImportResolver 概念接口：

```go
type ImportResolver interface {
    Resolve(ctx context.Context, from source.File, spec string) ([]source.File, error)
}
```

---

## 15. 作用域规则

建议作用域规则如下：

1. 每个入口文件和它的 imports 组成一个 compilation unit。
2. 顶层 `const`、`fn`、带 label 的 form symbol 位于 module scope。
3. block/form 内字段不产生顶层 symbol。
4. form label 是否产生 symbol 由 `FormSpec.Declares` 决定。
5. 函数参数只在函数体内可见。
6. `for` 变量只在循环体内可见。

---

## 16. 编译流程

### 16.1 Parse

```text
source -> AST
```

只做语法解析，不做领域判断。

### 16.2 Bind

```text
AST -> Symbol Table
```

收集：

- const
- fn
- form label symbol
- imports

检查：

- 重复定义
- 未定义引用
- import cycle
- 作用域冲突

### 16.3 Type Check

根据 schema 和 function signature 检查：

- form 是否存在
- label 是否允许
- 字段是否存在
- 字段是否必填
- 字段类型是否匹配
- nested form 是否允许
- body mode 是否符合
- 函数参数类型是否匹配
- ref 类型是否匹配

### 16.4 Evaluate

求值表达式：

- 字符串拼接
- 函数调用
- expr-lang 桥接表达式
- 数组
- 对象
- 运行时常量
- 环境变量读取

生成 typed value。

### 16.5 Typed Document

语言内核输出通用 typed document，而不是直接输出宿主 IR。

```go
type Document struct {
    Forms   []FormInstance
    Symbols SymbolTable
}
```

```go
type FormInstance struct {
    Kind   QName
    Label  *Symbol
    Fields map[string]Value
    Forms  []FormInstance
    Calls  []CallInstance
    Range  source.Range
}
```

### 16.6 Lowering

宿主把 typed document 转成自己的 IR。

示例：

```text
bu1ld:
  Document -> build.Project

编排系统:
  Document -> DeploymentPlan

CI 系统:
  Document -> PipelineSpec
```

语言内核不绑定业务 IR。

---

## 17. 可嵌入 Runtime API 设计

宿主使用方式：

```go
rt := runtime.New(runtime.Options{
    FS: afero.NewOsFs(),
})

rt.RegisterFrontend(plano.Frontend())

rt.Use(std.Module())
rt.Use(builddsl.Module())
rt.Use(godsl.Module())
rt.Use(dockerdsl.Module())

doc, diags := rt.CompileFile(ctx, "build.plano")
if diags.HasError() {
    return diags.Err()
}

project, err := builddsl.Lower(ctx, doc)
if err != nil {
    return err
}

plan, err := planner.Plan(project, "build")
if err != nil {
    return err
}

return executor.Execute(ctx, plan)
```

自定义 DSL：

```go
mod := schema.NewModule("company")

mod.Form("deploy.service").
    Label(schema.SymbolLabel("service")).
    Field("image", types.String).
    Field("replicas", types.Int.Default(1)).
    Field("env", types.Map(types.String)).
    Lower(func(ctx lower.Context, form schema.FormInstance) (any, error) {
        return CompanyServiceSpec{...}, nil
    })

rt.Use(mod)
```

---

## 18. 可插拔解析格式

调用方可以自定义解析格式，但建议它们最终输出统一 AST。

Frontend 接口：

```go
type Frontend interface {
    Name() string
    Extensions() []string
    Parse(ctx context.Context, src source.File) (*ast.File, diag.Diagnostics)
}
```

默认提供：

```text
plano syntax frontend
```

调用方可以注册：

```go
rt.RegisterFrontend(plano.Frontend())
rt.RegisterFrontend(companyyaml.Frontend())
```

这样可以做到：

```text
不同输入格式
  -> 同一个 AST
  -> 同一个 binder
  -> 同一个 type checker
  -> 同一个 schema registry
  -> 不同宿主 lowering
```

---

## 19. LSP 支持

LSP 能力应该来自 schema registry。

支持：

- form completion
- field completion
- function completion
- type diagnostics
- symbol references
- go to definition
- hover schema 文档
- function signature help

例如：

```plano
go.binary build {
  ma|
}
```

可以补全：

```text
main
out
deps
```

因为 `go.binary` schema 中注册了这些字段。

---

## 20. 错误诊断要求

错误信息必须包含：

- 文件
- 行
- 列
- 错误类型
- 具体字段或 symbol
- 期望类型
- 实际类型

示例：

```text
build.plano:22:10: field "deps" expects list<ref<task>>, got list<string>
```

示例：

```text
build.plano:8:13: undefined symbol "build"
```

示例：

```text
tasks/go.plano:3:3: missing required field "main" for form "go.binary"
```

---

## 21. V1 推荐范围

V1 应保持克制。

### V1 支持

语法：

- `import`
- `const`
- `FormDecl`
- `Assignment`
- `CallStmt`
- 表达式

表达式：

- string
- int
- bool
- array
- object
- ident
- call
- binary `+`

类型：

- string
- int
- bool
- list<T>
- object
- ref<T>

扩展：

- form schema registry
- function registry
- symbol table
- typed document
- lowerer API
- 默认 plano parser frontend

### V1 暂不支持

- `let` 可变赋值
- 用户自定义 `fn`
- `for`
- `while`
- `if statement`
- class
- exception
- 任意 IO
- 动态 parser hook

---

## 22. V2/V3 演进方向

### V2

- 用户自定义函数 `fn`
- `if` expression / statement
- `for` expression / statement
- formatter
- LSP 深度补全
- multi frontend
- remote plugin

### V3

- daemon incremental compile
- distributed execution
- agent integration
- infra service DSL
- policy DSL
- import module cache
- plugin lock / checksum

---

## 23. 最终总结

plano 应该是一门：

```text
schema-driven 的可嵌入强类型脚本 DSL。
```

它的核心设计边界是：

```text
语言核心不认识业务
业务语义通过 FormSpec 注册
宿主关键字是 soft keyword，不是硬编码 keyword
parser 统一解析为 FormDecl
binder/typechecker 根据宿主 registry 做语义判断
表达式是语言内建能力
symbol/ref 是强类型依赖关系的基础
typed document 是语言内核输出
lowering 由宿主完成
```

一句话：

```text
plano 的 DSL 扩展点不是自定义 parser 关键字，而是宿主注册语言 Form。
```

这样既可以达到类似 Kotlin Script DSL 的扩展体验，又不会让语言内核失控。
