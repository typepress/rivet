Rivet
=====

[![Go Walker](https://gowalker.org/api/v1/badge)](https://gowalker.org/github.com/typepress/rivet)
[![GoDoc](https://godoc.org/github.com/typepress/rivet?status.svg)](https://godoc.org/github.com/typepress/rivet)

[性能](#Performance) 可观的 http 路由管理器. 特征:

[简洁](#简洁), [顺序匹配](#顺序匹配), [支持注入](#注入), [深度解耦](#深度解耦), [HostRouter](#HostRouter).

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

Rivet 支持 "*", "?" 通配符和可自定义匹配模式的具名路由. 通用格式为

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

星号通配
--------

单个星号可匹配任意个非 "/" 字符

问号单配
--------

单个问号可匹配零或一个问号之前的字符. "/flavors?" 可匹配 "/flavor" 和 "/flavors".

可选尾斜线
----------

问号单配支持可选尾斜线, "/flavors/?" 可匹配 "/flavors" 和 "/flavors/".
建议在 http.Handler 中处理可选尾斜线, 而不是路由中.

Catch-All
---------

以 "**" 结尾的模式称作 Catch-All. Trie.Match 总是以 "**" 为名保存匹配字符串到返回的 Params 中.



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
 2. 按添加路由的顺序进行匹配, "*", "**" 除外.
 3. 最后匹配 "*" 和 "**"
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

Context 起到变量容器作用, 支持注入(Injector)变量, 有三个关键方法:

```go
    // MapTo 以 t 的类型为 key 把变量 v 关联到 context. 相同 t 值只保留一个.
    MapTo(v interface{}, t interface{})

    // Pick 以类型指针 t 为键值返回关联到 context 的变量.
    Pick(t unsafe.Pointer) interface{}

    // Map 等同调用 MapTo(v, v).
    Map(v interface{})
```

Dispatch 方法包装路由 handler, 结合 Context 实现支持注入的路由调用器 Dispatcher.


HostRouter
==========

HostRouter 实现了一个简单的 Host 路由.


Performance
===========

以下是与 [Echo][] 对比结果, Rivet 未使用注入.

```
#GithubAPI Routes: 203
   Echo: 76040 Bytes
   Rivet: 42840 Bytes

#GPlusAPI Routes: 13
   Echo: 6296 Bytes
   Rivet: 3064 Bytes

#ParseAPI Routes: 26
   Echo: 7216 Bytes
   Rivet: 5680 Bytes

#Static Routes: 157
   Echo: 60192 Bytes
   Rivet: 23880 Bytes

PASS
BenchmarkEcho_Param         20000000            97.7 ns/op         0 B/op          0 allocs/op
BenchmarkRivet_Param         5000000           254 ns/op          48 B/op          1 allocs/op
BenchmarkEcho_Param5        10000000           175 ns/op           0 B/op          0 allocs/op
BenchmarkRivet_Param5        2000000           731 ns/op         240 B/op          1 allocs/op
BenchmarkEcho_Param20        3000000           481 ns/op           0 B/op          0 allocs/op
BenchmarkRivet_Param20        500000          2538 ns/op        1024 B/op          1 allocs/op
BenchmarkEcho_ParamWrite    10000000           212 ns/op          16 B/op          1 allocs/op
BenchmarkRivet_ParamWrite    5000000           347 ns/op          48 B/op          1 allocs/op
BenchmarkEcho_GithubStatic  10000000           122 ns/op           0 B/op          0 allocs/op
BenchmarkRivet_GithubStatic 10000000           142 ns/op           0 B/op          0 allocs/op
BenchmarkEcho_GithubParam   10000000           199 ns/op           0 B/op          0 allocs/op
BenchmarkRivet_GithubParam   3000000           567 ns/op          96 B/op          1 allocs/op
BenchmarkEcho_GithubAll        30000         51867 ns/op           0 B/op          0 allocs/op
BenchmarkRivet_GithubAll       10000        118267 ns/op       16272 B/op        167 allocs/op
BenchmarkEcho_GPlusStatic   20000000            94.5 ns/op         0 B/op          0 allocs/op
BenchmarkRivet_GPlusStatic  10000000           102 ns/op           0 B/op          0 allocs/op
BenchmarkEcho_GPlusParam    10000000           151 ns/op           0 B/op          0 allocs/op
BenchmarkRivet_GPlusParam    5000000           353 ns/op          48 B/op          1 allocs/op
BenchmarkEcho_GPlus2Params  10000000           176 ns/op           0 B/op          0 allocs/op
BenchmarkRivet_GPlus2Params  3000000           584 ns/op          96 B/op          1 allocs/op
BenchmarkEcho_GPlusAll        500000          2420 ns/op           0 B/op          0 allocs/op
BenchmarkRivet_GPlusAll       300000          4769 ns/op         768 B/op         11 allocs/op
BenchmarkEcho_ParseStatic   20000000            96.3 ns/op         0 B/op          0 allocs/op
BenchmarkRivet_ParseStatic  20000000            98.9 ns/op         0 B/op          0 allocs/op
BenchmarkEcho_ParseParam    20000000           111 ns/op           0 B/op          0 allocs/op
BenchmarkRivet_ParseParam    5000000           282 ns/op          48 B/op          1 allocs/op
BenchmarkEcho_Parse2Params  10000000           135 ns/op           0 B/op          0 allocs/op
BenchmarkRivet_Parse2Params  3000000           425 ns/op          96 B/op          1 allocs/op
BenchmarkEcho_ParseAll        300000          4270 ns/op           0 B/op          0 allocs/op
BenchmarkRivet_ParseAll       200000          7446 ns/op         912 B/op         16 allocs/op
BenchmarkEcho_StaticAll        50000         31607 ns/op           0 B/op          0 allocs/op
BenchmarkRivet_StaticAll       50000         34967 ns/op           0 B/op          0 allocs/op
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