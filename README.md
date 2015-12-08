Rivet
=====

[![Go Walker](https://gowalker.org/api/v1/badge)](https://gowalker.org/github.com/typepress/rivet)
[![GoDoc](https://godoc.org/github.com/typepress/rivet?status.svg)](https://godoc.org/github.com/typepress/rivet)

[性能](#performance) 可观的 http 路由管理器. 特征:

[简洁](#简洁), [顺序匹配](#顺序匹配), [支持注入](#注入), [深度解耦](#深度解耦), [HostRouter](#hostrouter).

[examples][] 目录中有几个例子, 方便您了解 Rivet.

任何问题, 分享可至 [issues][] , [wiki][]

这里有个路由专项评测 [go-http-routing-benchmark][benchmark].

Rivet 版本号采用 [Semantic Versioning](http://semver.org/).


简洁
====

常规风格示例: 在本地运行此代码, 然后后点击 [这里](http://127.0.0.1:3282/Rivet)

```go
package main

import (
    "net/http"

    "github.com/typepress/rivet"
)

// 常规风格 handler
func helloWord(rw http.ResponseWriter, req *http.Request) {
    io.WriteString(rw, "Hello Word")
}

// Context handler.
func hi(c *rivet.Context) {
    c.WriteString("Hi "+c.Get("who")) // 提取参数 who
}

func main() {
    
    // 新建 rivet 实现的 http.Handler
    mux := rivet.New()

    // 注册路由
    mux.Get("/", helloWord)
    mux.Get("/:who", hi) // 参数名设定为 "who"
    
    http.ListenAndServe(":3282", mux) 
}
```


路由风格
========

Rivet 支持 `"*"`, `"?"` 通配符和可自定义匹配模式的具名路由. 通用格式为

```
:name MatcherName exp
```

其中 ":" 为定界符, name 为参数名, MatcherName 为自定义匹配模式名, exp 为 MatcherName 对应的表达式.
Rivet 内建了一些路由, 它们保存在全局对象 Matches 中.

最简形式 ":name" 等同 ":name string", "string" 是内建的 Matcher.

正则支持
--------

当无法确认匹配模式时被当做正则对待. 比如

```
"/:id ^id(\d+)$"
```

把 "^id(\d+)$" 作为模式名在 Matches 中是找不到的, 事实上使用者也不会这样命名.

单星通配
--------

单个星号可匹配任意个非分割(通常是 "/", 可定制)字符

双星通配
---------

以 `"**"` 结尾的模式称作 Catch-All. Trie.Match 总是以 `"**"` 为名保存匹配字符串到返回的 Params 中.

形如 `"/src/github.com/**.go"`, `"**"` 后还有定值的称作后缀匹配.

问号单配
--------

单个问号可匹配零或一个问号之前的字符. "/flavors?" 可匹配 "/flavor" 和 "/flavors".

可选尾斜线匹配只是问号单配得一个实例 "/flavors/?" 可匹配 "/flavors" 和 "/flavors/".
建议在 http.Handler 中处理可选尾斜线, 而不是路由中.


顺序匹配
========

通常路由都能支持静态路由, 参数路由, 正则匹配, Catch-All 等, Rivet 也同样支持, 而且做的更好.

```
"/hi/**",
"/",
"/**",
"/hi*",
"/hi*/",
"/hi/*",
"/hi",
```

显然匹配顺序影响匹配结果. Rivet 采用的匹配顺序为:

 1. 静态字符串优先, 
 2. 按添加路由的顺序进行匹配, `"*"`, `"**"` 除外.
 3. 最后匹配 `"*"` 和 `"**"`
 4. 直到所有的 URL.Path 被消耗完且 Trie.Word 非 nil


深度解耦
========

解耦可以让应用切入到路由执行的每一个环节. Rivet 对解耦做了很多工作. 大概可以分三个级别.

 1. 底层级别 Trie, Params, Matcher 是 Rivet 的核心, 它们可以独立工作.
 2. 路由级别 Router 简单实现了注册和匹配路由.
 3. 注入级别 Rivet 实现了 http.Handler, 内部使用 Context 和 Dispatcher 支持注入.

使用者依据喜好决定如何使用.


注入
====

对于静态类型语言来说, 类型约束造成应用需要大量继承框架定义的类型, 注入改善这种状况, 原理是:

    多数情况下对于路由 Handler 可以通过参数类型自动匹配事先准备好的变量完成调用.

Context 起到变量容器作用, 支持注入(Injector)反射调用, 有三个关键方法:

```go
    // MapTo 以 t 的类型为 key 把变量 v 关联到 context. 相同 t 值只保留一个.
    MapTo(v interface{}, t interface{})

    // Pick 以类型指针 t 为键值返回关联到 context 的变量.
    Pick(t unsafe.Pointer) interface{}

    // Map 等同调用 MapTo(v, v).
    Map(v interface{})
```

如果注入变量不是为了反射调用, 那么直接操作 Context.Store 更轻量.
使用 Context.Store 前您需要先 make 它.

Dispatch 方法包装路由 handler, 结合 Context 实现支持注入的路由调用器 Dispatcher.

HostRouter
==========

HostRouter 是个简单的 Host 路由. 如果和 Rivet 配合可实现完整的 http 路由, 比如:

```
hr := rivet.NewHostRouter()

// 为域名分配独立的 *Rivet, 当然别忘记给他们注册路由
golang := rivet.New()
godoc := rivet.New()

hr.Add("*.golang.org", golang)
hr.Add("*.godoc.org", godoc)
```


分组路由
========

Rivet 中没有独立的分组路由方法, 但可以通过组合几个 Trie 使用 Trie.Add 方法来实现.
HostRouter 就是这样的例子.


Performance
===========

以下是 Rivet 未使用注入时与 [Echo][], [Gin][] Benchmark 对比结果.

```
#GithubAPI Routes: 203
   Echo: 76040 Bytes
   Gin: 52672 Bytes
   Rivet: 46088 Bytes

#GPlusAPI Routes: 13
   Echo: 6296 Bytes
   Gin: 3856 Bytes
   Rivet: 3272 Bytes

#ParseAPI Routes: 26
   Echo: 7216 Bytes
   Gin: 6880 Bytes
   Rivet: 6096 Bytes

#Static Routes: 157
   Echo: 60192 Bytes
   Gin: 30544 Bytes
   Rivet: 26392 Bytes

PASS
BenchmarkEcho_Param         20000000          97 ns/op         0 B/op        0 allocs/op
BenchmarkGin_Param          20000000          90 ns/op         0 B/op        0 allocs/op
BenchmarkRivet_Param         5000000         233 ns/op        48 B/op        1 allocs/op
BenchmarkEcho_Param5        10000000         176 ns/op         0 B/op        0 allocs/op
BenchmarkGin_Param5         10000000         154 ns/op         0 B/op        0 allocs/op
BenchmarkRivet_Param5        2000000         697 ns/op       240 B/op        1 allocs/op
BenchmarkEcho_Param20        3000000         474 ns/op         0 B/op        0 allocs/op
BenchmarkGin_Param20         5000000         378 ns/op         0 B/op        0 allocs/op
BenchmarkRivet_Param20       1000000        2214 ns/op      1024 B/op        1 allocs/op
BenchmarkEcho_ParamWrite    10000000         206 ns/op        16 B/op        1 allocs/op
BenchmarkGin_ParamWrite     10000000         195 ns/op         0 B/op        0 allocs/op
BenchmarkRivet_ParamWrite    3000000         421 ns/op       112 B/op        2 allocs/op
BenchmarkEcho_GithubStatic  10000000         115 ns/op         0 B/op        0 allocs/op
BenchmarkGin_GithubStatic   20000000         113 ns/op         0 B/op        0 allocs/op
BenchmarkRivet_GithubStatic 10000000         127 ns/op         0 B/op        0 allocs/op
BenchmarkEcho_GithubParam   10000000         197 ns/op         0 B/op        0 allocs/op
BenchmarkGin_GithubParam    10000000         187 ns/op         0 B/op        0 allocs/op
BenchmarkRivet_GithubParam   3000000         526 ns/op        96 B/op        1 allocs/op
BenchmarkEcho_GithubAll        30000       45353 ns/op         0 B/op        0 allocs/op
BenchmarkGin_GithubAll         30000       39494 ns/op         0 B/op        0 allocs/op
BenchmarkRivet_GithubAll       20000       97124 ns/op     16272 B/op      167 allocs/op
BenchmarkEcho_GPlusStatic   20000000          88 ns/op         0 B/op        0 allocs/op
BenchmarkGin_GPlusStatic    20000000          87 ns/op         0 B/op        0 allocs/op
BenchmarkRivet_GPlusStatic  20000000          81 ns/op         0 B/op        0 allocs/op
BenchmarkEcho_GPlusParam    10000000         122 ns/op         0 B/op        0 allocs/op
BenchmarkGin_GPlusParam     20000000         117 ns/op         0 B/op        0 allocs/op
BenchmarkRivet_GPlusParam    5000000         298 ns/op        48 B/op        1 allocs/op
BenchmarkEcho_GPlus2Params  10000000         170 ns/op         0 B/op        0 allocs/op
BenchmarkGin_GPlus2Params   10000000         154 ns/op         0 B/op        0 allocs/op
BenchmarkRivet_GPlus2Params  3000000         438 ns/op        96 B/op        1 allocs/op
BenchmarkEcho_GPlusAll       1000000        2322 ns/op         0 B/op        0 allocs/op
BenchmarkGin_GPlusAll        1000000        1916 ns/op         0 B/op        0 allocs/op
BenchmarkRivet_GPlusAll       300000        4395 ns/op       768 B/op       11 allocs/op
BenchmarkEcho_ParseStatic   20000000          91 ns/op         0 B/op        0 allocs/op
BenchmarkGin_ParseStatic    20000000          87 ns/op         0 B/op        0 allocs/op
BenchmarkRivet_ParseStatic  2000000           85 ns/op         0 B/op        0 allocs/op
BenchmarkEcho_ParseParam    20000000         105 ns/op         0 B/op        0 allocs/op
BenchmarkGin_ParseParam     20000000          96 ns/op         0 B/op        0 allocs/op
BenchmarkRivet_ParseParam    5000000         261 ns/op        48 B/op        1 allocs/op
BenchmarkEcho_Parse2Params  10000000         129 ns/op         0 B/op        0 allocs/op
BenchmarkGin_Parse2Params   10000000         115 ns/op         0 B/op        0 allocs/op
BenchmarkRivet_Parse2Params  3000000         397 ns/op        96 B/op        1 allocs/op
BenchmarkEcho_ParseAll        300000        4223 ns/op         0 B/op        0 allocs/op
BenchmarkGin_ParseAll         300000        3491 ns/op         0 B/op        0 allocs/op
BenchmarkRivet_ParseAll       200000        6877 ns/op       912 B/op       16 allocs/op
BenchmarkEcho_StaticAll        50000       30386 ns/op         0 B/op        0 allocs/op
BenchmarkGin_StaticAll         50000       27437 ns/op         0 B/op        0 allocs/op
BenchmarkRivet_StaticAll       50000       32593 ns/op         0 B/op        0 allocs/op
```


Acknowledgements
================

Inspiration from Julien Schmidt's [httprouter][], about Trie struct.

Trie 算法灵感来自 Julien Schmidt's [httprouter][].


LICENSE
=======
Copyright (c) 2014 The TypePress Authors. All rights reserved.
Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file.

[issues]: //github.com/typepress/rivet/issues
[wiki]: //github.com/typepress/rivet/wiki
[httprouter]: //github.com/julienschmidt/httprouter
[benchmark]: //github.com/julienschmidt/go-http-routing-benchmark
[examples]: //github.com/typepress/rivet/tree/master/examples
[Echo]: //github.com/labstack/echo
[Gin]: //github.com/gin-gonic/gin