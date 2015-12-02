Rivet
=====

[![Go Walker](https://gowalker.org/api/v1/badge)](https://gowalker.org/github.com/typepress/rivet)[![GoDoc](https://godoc.org/github.com/typepress/rivet?status.svg)](https://godoc.org/github.com/typepress/rivet)

专注路由.
[简洁](#简洁), [顺序匹配](#顺序匹配), [支持注入](#注入), [深度解耦](#深度解耦)的 http 路由管理器.

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
func hi(c rivet.Context) {
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

Rivet 支持 "*" 匹配和可自定义匹配模式的具名路由. 通用格式为

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