Rivet
=====

[![Go Walker](http://gowalker.org/api/v1/badge)](http://gowalker.org/github.com/typepress/rivet)

简洁, 支持注入, 可定制, 深度解耦的 http 路由管理器.

Rivet 专注路由相关功能, 未来不会增加非路由相关的功能.

任何问题, 分享可至 [issues](issues) , [wiki](wiki) 

简洁
====

Rivet 使用和常规风格一致.

示例: 运行此代码后点击 [这里](http://127.0.0.1:3000/hello/Rivet)

```go
package main

import (
    "io"
    "net/http"

    "github.com/typepress/rivet"
)

// 常规风格 handler
func HelloWord(rw http.ResponseWriter, req *http.Request) {
    io.WriteString(rw, "Hello Word")
}

/**
带参数的 handler
params 是从 URL.Path 中提取到的参数
*/
func Hi(params rivet.Params, rw http.ResponsWriter) {
    io.WriteString(rw, "Hi "+params.Get("who")) // 提取参数 who
}

func main() {
    
    // 新建路由管理器
    mux := rivet.NewRouter(nil) // 下文解释参数 nil

    // 注册路由
    mux.Get("/", HelloWord)
    mux.Get("/:who", Hi) // 参数名设定为 "who"
    
    // rivet.Router 符合 http.Handler 接口
    http.ListenAndServe(":3000", mux) 
}
```

上例中 `"/"` 是无 URL.Path 参数路由. `"/:who"` 是有参数的, 参数名为 "who".

访问 "/" 输出:
```
Hello Word
```

访问 "/Boy" 输出:
```
Hi Boy
```

访问 "/Girl" 输出:
```
Hi Girl
```

访问 "/news/sports" 会得到 404 NotFound 页面.

下面以 api.github.com 真实路由为例:
```go
mux.Get("/users/:user/events", Events)
mux.Get("/users/:user/events/orgs/:org", Events)
```

因为都用同一个 handler, Events 可以这样写:
```go
func Events(params rivet.Params, rw http.ResponsWriter) {
    user := params.Get("owner")
    if user == "github" {
        // 用户 github 很受欢迎, 需要特别处理
        // do something
        return 
    }
    
    // 因为两个路由 path 都用 Events 处理, 可根据参数进行区分
    org := params.Get("org")
    if org != "" {
        // 对应 "/users/:user/events/orgs/:org" 的处理
        return
    }

    // 对应 "/users/:user/events" 的处理
}
```

事实上 api.github.com 路由很多, 分开用不同的 handler 处理才是好方法:
```go
mux.Get("/users/:user/events", userEvents)
mux.Get("/users/:user/events/orgs/:org", userOrgEvents)
```

注入
====

rivet.Context 支持注入(Injector), 有三个关键方法:

```go
    // MapTo 以 t 为 key 把变量 v 关联到 context. 相同 t 值只保留一个.
    MapTo(v interface{}, t uint)

    // Get 以类型标识 t 为 key, 返回关联到 context 的变量.
    Get(t uint) interface{}

    // Map 自动提取 v 的类型标识作为 t, 调用 MaptTo. 通常使用 Map.
    Map(v interface{})
```

实际中的需求更复杂, 比如不同用户在相同 URL.Path 下有不同响应, 用户角色控制.
使用注入后会很简单.

```go
// 用户角色, 示意, 简单的定义为 string
type Role string

/**
使用注入的方法确定用户角色.
只需要给 handler 一个 rivet.Context 参数就可以使用注入.
*/
func UserRole(c rivet.Context) {
    // Context.Request() 返回 *http.Request
    req := c.Request()

    // 通常根据 session 确定用户角色.
    session := req.Cookie("session").Value

    // 这里只是示意代码, 现实中不可能这么做.
    switch session {
    default: // 游客
        c.Map(Role(""))

    case "admin": // 管理员
        c.Map(Role("admin"))

    case "signOn": // 已经登录
        c.Map(Role("signOn"))
    }
}

/**
DelComments 删除评论, 需要的参数由前面的 UserRole 准备.
*/
func DelComments(role Role, params rivet.Params, rw http.ResponsWriter) {
    if role == "" {
        // 拒绝游客
        rw.WriteHeader(http.StatusForbidden)
        return
    }

    if role == "admin" {
        // 允许 admin
        // do delete
        return
    }

    // 其他角色,需要更多的判断
    // do something
}
```

注册路由:
```go
mux.Get("/del/comments/:id", UserRole, DelComments)
```

这个例子中, `"/del/comments/:id"` 被匹配后, 先执行 UserRole, 把用户角色关联到 Context, 因为 UserRole 没有对 http.ResponsWriter 进行写操作, DelComments 被执行.

定制
====

事实上, 上例中的 UserRole 很多地方都要用, 每次注册路由都带上 UserRole 很不方便.
通常 UserRole 是在路由匹配之前以先执行. 可以这样用:

```go
// 定义自己的 rivet.Context 生成器
func MyRiveter(rw http.ResponseWriter, req *http.Request) rivet.Context {
    c := new(rivet.NewContext(rw, req))
    // 先执行角色控制
    UserRole(c)
    return c
}

func main() {

    // 使用 MyRiveter
    mux := rivet.NewRouter(MyRiveter)

    mux.Get("/del/comments/:id", DelComments)

    http.ListenAndServe(":3000", mux)
}
```

其他方法也很多, 这只是最简单的一种.

深度解耦
========

解耦可以让应用切入到 Rivet 执行路由流程中的每一个环节, 达到高度定制. Rivet 在不失性能的前提下, 对解耦做了很多努力. 了解下列 Rivet 的设计接口有助于定制您自己的路由规则.

* [Params][] 保存 URL.Path 中的参数
* [Filter][] 检查/转换 URL.Path 参数, 亦可过滤请求.
* [Node][] 保存 handler, 二次过滤 Params, 每个 Node 都拥唯一 id.
    二次过滤很重要, 路由匹配过程中可能发生回溯, 会产生一些多余参数.
* [Trie][] 匹配 URL.Path, 调用 Filter, 调用 Params 生成器.
    匹配到的 Trie.id 和 Node.id 是对应的.
* [Context][] 维护上下文, 处理 handler. 内置 Rivet 实现了它.
* [Router][] 路由管理器, 把上述对象联系起来, 完成路由功能.

他们是如何解耦的:

Params 和其他无关, 无其它依赖, 唯一的约束是有固定的类型定义.

Filter 接口无其它依赖, 还有便捷 FilterFunc 形式.

Node 接口依赖 Context.

Trie 依赖 Filter 接口, 是路由匹配的核心. 生成 Params 用的独立函数接口 ParamsReceiver, ParamsReceiver 无其它依赖, 甚至和 Params 也无关.

Context 接口依赖 ParamsReceiver, 间接来说也是无依赖的. 但是 Context 用了注入, 可能您的应用并不需要注入.

Rivet 是内置的 Context 实现, 是个 struct, 可以扩展. 并且接口丰富.

Router 依赖上述所有. 可以通过两个函数 NodeBuilder 和 Riveter 定制自己的 Node, Context.

因此大概有分两种深度使用级别:

    底层: 直接使用 Trie, 构建自己的 Node, ParamsReceiver, Context, Router.
    扩展: 使用 Router, 自定义 Context 生成器, 或者扩展 Rivet.

    深度使用, 这有几个函数和类型需要您了解.
    TypeIdOf, NewContext, NewNode, ParamsFunc, FilterFunc,

虽然底层使用仍然依赖 Filter, 需要传递 FilterBuilder, 如果您的路由 Path 不含有复杂的参数匹配. 直接用 nil 替代即可. 本文不展示底层使用示例.

自定义 Context 生成器:
```go
// 自定义 Context 生成器
func MyRiveter(rw http.ResponseWriter, req *http.Request) rivet.Context {

    // 构建自己的 rw,  比如实现一个真正的 http.Flusher
    rw = MyResponseWriterFlusher(rw) 
    c := new(rivet.NewContext(rw, req)) // 依旧使用 rivet.Rivet
    return c
}
```

rivet 内置的 ResponseWriteFakeFlusher 是个伪 http.Flusher, 只是有个 Flus() 方法, 并没有真的实现 http.Flusher 功能. 如果您需要真正的 Flusher 需要自己实现.

实现自己的 Context 很容易, 善用 Next 和 Invoke 方法即可.

举例:

```go
/**
扩展 Context, 实现 Before.
*/
type MyContext struct {
    rivet.Context
    beforeIsRun true
}

/**
MyContext 生成器
使用:
    reivt.Router(MyRiveter)
*/
func MyRiveter(res http.ResponseWriter, req *http.Request) rivet.Context {
    c := new(MyContext)
    c.Context = rivet.NewContext(res, req)
    return c
}

func (c *MyContext) Next() {
    if !beforeIsRun {
        // 执行 Before 处理
        // do something
        beforeIsRun = true
    }
    c.Context.Next()
}

// 观察者模式
func Observer(c rivet.Context) {
    defer func() {
        if err := recover(); err != nil {
            // 捕获 panic
            // do something
            return
        }
        // 其他操作, 比如写日志, 统计执行时间等等
        // do something
    }()
    c.Next()
}

/**
MyInvoke 是个 Handler, 执行时可以调用 Context.Invoke. 例如:
插入执行 SendStaticFile, 这和直接调用 SendStaticFile 不同.
这样的 SendStaticFile 可以使用上下文关联变量
*/
func MyInvoke(c rivet.Context) {
    c.Invoke(SendStaticFile)
}

/**
发送静态文件, 参数 root 是前期执行的某个 Handler 关联好的.
现实中简单的改写 req.URL.Path, 无需 root 参数也是可行的.
*/
func SendStaticFile(root http.Dir, rw http.ResponseWriter, req *http.Request) {
    // send ...
}

```


路由风格
========

Rivet 对路由 pattern 的支持很丰富.

示例:
```
"/news/:cat"
```

可匹配:
```
"/news/sprots"
"/news/health"
```

示例:
```
"/news/:cat/:id"
```

可匹配:
```
"/news/sprots/9527"
"/news/health/1024"
```

当然您可以把这两条路由都注册到 Router, 它们会被正确匹配.
上面的路由只有参数名, 数据类型都是 string. Rivet 还支持带类型的 pattern.

示例:
```
"/news/:cat/:id uint"
```

uint 是内置的 class, 参见 [FilterClass][].

":id uint" 表示参数名是 "id", 数据必须是 uint 字符串.

路由风格:

```
"/path/to/prefix:pattern/:pattern/:"
```

其中 "path", "to","prefix" 是占位符, 表示固定字符, 称为定值.
":pattern" 表示匹配模式, 格式为:

```
:name class arg1 arg2 argN

    以 ":" 开始, 以 " " 作为分隔符.
    第一段是参数名, 第二段是类型名, 后续为参数.
    
    示例: ":cat string 6"

    cat
        为参数名, 如果省略只验证不提取参数, 形如 ": string 6"
    string
        为类型名, 可以自定义 class 注册到 FilterClass 变量.
    6
        为长度参数, 可以设置一个限制长度参数. 例如
        ":name string 5"
        ":name uint 9"
        ":name hex 32"

:name class
    提取参数, 以 "name" 为 key, 根据 class 对值进行合法性检查.

:name
    提取参数, 不对值进行合法检查, 值不能为空.
    如果允许空值要使用 ":name *". "*" 是个 class, 允许空值.

:
    不提取参数, 不检查值, 允许空值, 等同于 ": *".
::
    只能用于模式尾部. 提取参数, 不检查值, 允许空值, 参数名为 "*".
    例如:
        "/path/to/::"
    可匹配:
        "/path/to/",          "*" 为参数名, 值为 "".
        "/path/to/paths",     "*" 为参数名, 值为 "paths".
        "/path/to/path/path", "*" 为参数名, 值为 "path/path".
*
    "*" 可替代 ":" 作为开始定界符, 某些情况 "*" 更符合习惯, 如:
    "/path/to*"
    "/path/to/**"
```

Rivet 在路由匹配上做了很多工作, 支持下列路由同时存在, 并正确匹配:

```
"/path/to:name"
"/path/to:name/"
"/path/to:name/suffix"
"/:name"
"/path/**"
```

即便如此, 还会有这些路不能并存.

Scene
=====

路由风格支持类型, Filter 检查时可能需要对数据进行类型转换, interface{} 方便保存转换后的结果, 避免后续代码再次转换, 所以 Params 定义成这样:

```go
type Params map[string]interface{}
```

一些应用场景无转换需求, 只需要简单定义:

```go
type PathParams map[string]string
```

是的, 这种场景也很普遍. [Scene][] 就是为此准备的 Context.

Scene 的使用很简单:
```go
package main

import (
    "io"
    "net/http"

    "github.com/typepress/rivet"
)

/**
带 PathParams 参数的 handler
params 是从 URL.Path 中提取到的参数
*/
func Hi(params rivet.PathParams, rw http.ResponsWriter) {
    io.WriteString(rw, "Hi "+params["who"])
}

func main() {
    
    // 传递 NewScene, handler 可以采用 PathParams 风格
    mux := rivet.NewRouter(rivet.NewScene)

    mux.Get("/:who", Hi) // 参数名设定为 "who"
    
    http.ListenAndServe(":3000", mux) 
}
```

注意 PathParams 只能和 NewScene 配套使用. 事实上 Context 采用的是 All-In-One 的设计方式, 实现有可能未完成所有接口, 使用方式对应变更即可.


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


[Node]: https://gowalker.org/github.com/typepress/rivet#Node
[Filter]: https://gowalker.org/github.com/typepress/rivet#Filter
[Params]: https://gowalker.org/github.com/typepress/rivet#Params
[Trie]: https://gowalker.org/github.com/typepress/rivet#Trie
[Context]: https://gowalker.org/github.com/typepress/rivet#Context
[Router]: https://gowalker.org/github.com/typepress/rivet#Router
[Scene]: https://gowalker.org/github.com/typepress/rivet#Scene
[Rivet.Get]: https://gowalker.org/github.com/typepress/rivet#Rivet_Get
[Rivet.Invoke]: https://gowalker.org/github.com/typepress/rivet#Rivet_Invoke
[FilterClass]: https://gowalker.org/github.com/typepress/rivet#_variables