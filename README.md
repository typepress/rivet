rivet
=====

是一个简单的 http 路由, 特点是 Context 实例由用户生成.

路由风格
========

示例:

```
"/path/to/prefix<pattern>suffix/prefix:name/prefix*suffix/*"
```

经过 "/" 分割, 最多支持 255 段. 
示例中的 "path", "to","prefix", "suffix" 是字面值, 称为字面风格.
支持三种模式风格: 同一个段内最多使用一种风格.

```
<pattern>
    一个 pattern 两端以 "<", ">" 定界, 以 " " 作为分隔符.
    第一段是参数名, 第二段是类型名, 后续为参数.
    示例: <cat string 6>
   cat
    为参数名, 省略表示只验证不提取参数, 形如 < string>
        string
            为类型名, 可以注册自定义 class 到 PatternFactory 变量.
        6
            为参数, 所有内建类型可以设置一个限制长度参数, 最大值 255. 例如
            <name string 6>
            <name int 9>
            <name hex 32>
:name
    简化风格, 等同于: <name string>.
*
    简化风格, 等同于: < *>. 允许空值, 不垮段, 只匹配不提取参数

    注意: <name string 0> 中的 0 无法产生作用, 应该用 <name *> 替代.
```

使用
====

实现这两个接口就可以自主控制 Context 的生成.
rivet 提供了一个实现, 您也可以直接使用.

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


LICENSE
=======
Copyright 2014 The TypePress Authors. All rights reserved.
Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file.