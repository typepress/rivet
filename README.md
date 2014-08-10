rivet
=====

[![Go Walker](http://gowalker.org/api/v1/badge)](http://gowalker.org/github.com/typepress/rivet)

是一个简单的 http 路由

特性
====

开放和自由是 rivet 的设计初衷

* Context 接口设计, 更自由. Injector 设计. 灵感源自 [Martini](https://github.com/go-martini).
* Handler 泛函数支持, 定义为 interface{}.
* Node    接口设计, 只为存储 Handler.
* Pattern 接口设计, 路由期进行 URL 参数检查和转换.
* Router  的 Match 方法使它可独立使用.
* Trie    高效的路由匹配, 可独立使用. 灵感源自 [httprouter](https://github.com/julienschmidt/httprouter).
* Rivet   是预置的 Context, 内部使用伪 http.Flusher 兼容不同需求.

上述特性事实上开放了路由所有环节, rivet 实现了开放的路由设计.

路由风格
========

示例:

```
"/path/to/prefix:pattern/:pattern/:"
```

以 "/" 分割成段. ":" 号为开始定界符
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
    简化风格, 等同于 ": *". 允许空值, 只匹配不提取参数
::
    尾部全匹配, 只能用于模式尾部, 提取参数, 参数名为 "*". 例如:
    "/path/to/catch/all/::"
    会匹配 "/path/to/catch/all/paths", 并以 "*" 为名提取 "paths".
*
    "*" 可替代 ":" 作为开始定界符, 某些情况 "*" 更符合常规思维, 如:
    "/path/to*"
    "/path/to/catch/all/**"
```

Rivet 在路由匹配上做了很多工作, 支持下列路由同时存在, 并正确匹配:

```
"/path/to:name"
"/path/to:name/"
"/path/to:name/suffix"
"/:name"
"/path/**"
```

即便如此, 相信仍然会有一些路由无法支持.

使用
====

示例: 运行此代码后点击 [这里](http://127.0.0.1:3000/hello/Rivet)

```go
package main

import (
    "net/http"

    "github.com/typepress/rivet"
)

// 简单的 handler
func Hello(c rivet.Context) {
    c.WriteString("Hello " + c.Params().Get("name"))
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