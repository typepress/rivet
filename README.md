rivet
=====

[![Go Walker](http://gowalker.org/api/v1/badge)](http://gowalker.org/github.com/typepress/rivet)

是一个简单的 http 路由

特性
====

开放和自由是 rivet 的设计初衷

* Context 实例由使用者控制生成, 更自由.
* Handler 定义为 interface{}.
* Handler 可接管 Context 控制权.
* Injector 设计. 灵感源自 [Martini](https://github.com/go-martini).
* Pattern 可自定义, 路由期进行 URL 参数检查和转换.
* Router 的 Match 方法使它可自由组合.
* Trie 实现的 Route 可独立使用. 灵感源自 [httprouter](https://github.com/julienschmidt/httprouter).
* 预置的 Rivet 使用 ResponseWriteFakeFlusher 实例. 伪 http.Flusher 更符合对 Flusher 的不同需求.

上述特性事实上开放了路由所有环节, rivet 实现了独立, 开放的路由设计.
rivet 未提供 Before Handler, 因为上述特性足够开放, 自由组合可实现多种需求.

路由风格
========

示例:

```
"/path/to/prefix:pattern/:pattern/:"
```

以 "/" 分割成段.
示例中的 "path", "to","prefix" 是字面值, 称为定值.

```
:pattern
    一个 pattern 以 ":" 开始, 以 " " 作为分隔符.
    第一段是参数名, 第二段是类型名, 后续为参数.
    示例: :cat string 6:
   cat
    为参数名, 省略表示只验证不提取参数, 形如 ": string"
        string
            为类型名, 可以注册自定义 class 到 PatternClass 变量.
        6
            为参数, 所有内建类型可以设置一个限制长度参数, 最大值 255. 例如
            ":name string 6"
            ":name int 9"
            ":name hex 32"
:name
    简化风格, 用于段尾部, 等同于 ":name string".
    注意: ":name string 0" 中的 0 不能使空值生效, 应该用 ":name *".
:
    等同 "*" 模式
*
    简化风格, 等同于 ": *". 允许空值, 只匹配不提取参数
::
    等同 "**" 模式
**
    尾部全匹配, 只能用于模式尾部, 提取参数, 参数名为 "*". 例如:
    "/path/to/catch/all/**"
    会匹配 "/path/to/catch/all/paths", 并以 "*" 为名提取 "paths".
```


使用
====

实现这两个接口就可以自主控制 Context 的生成.
rivet 提供类型 Rivet 实现了这两个接口, 您也可以直接使用.

```go
/**
Riveter 用于生成 Context 实例, 需要用户实现.
*/
type Riveter interface {
    // Context 生成 Context 实例
    Context(res http.ResponseWriter, req *http.Request) Context
}

/**
Context 是实际的 http Request 处理对象.
*/
type Context interface {
    // Source 返回产生 Context 的参数
    Source() (http.ResponseWriter, *http.Request)
    /**
    Invoke 负责调用 http.Request Handler
    参数:
        params 含有路由匹配模式提取到的参数
            为 nil, 那一定是匹配失败.
            即便 len(params) 为 0 也表示匹配成功.
        handlers 由 Router 匹配得到.
            当设置了 NotFound Handler 时, 也会通过此方法传递.
            如果匹配失败, 且没有设置 NotFound Handler, 此值为 nil.
    */
    Invoke(params Params, handlers ...Handler)
}
```

只需要完成这两个简单的接口就可以使用 rivet 了.

示例: 运行此代码后点击 [这里](http://127.0.0.1:3000/hello/Rivet)

```go
package main

import (
    "net/http"

    "github.com/typepress/rivet"
)

// 简单的 handler
func Hello(w http.ResponseWriter, params rivet.Params) {
    w.Write([]byte("Hello " + params["name"].(string)))
}

func main() {

    mux := rivet.NewRouter(nil) // 传递 nil, 会使用内部的 Rivet 实现
    mux.Get("/hello/:name", Hello) // 设置 GET 路由

    http.ListenAndServe(":3000", mux) // rivet.Router 符合 http.Handler 接口
}
```


Acknowledgements
================

Inspiration from Julien Schmidt's [httprouter](https://github.com/julienschmidt/httprouter), about Trie struct.

Trie 算法和结构灵感来自 Julien Schmidt's [httprouter](https://github.com/julienschmidt/httprouter).


LICENSE
=======
Copyright (c) 2013 Julien Schmidt. All rights reserved.
Copyright (c) 2014 The TypePress Authors. All rights reserved.
Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file.