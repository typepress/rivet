Rivet
=====

[![Go Walker](http://gowalker.org/api/v1/badge)](http://gowalker.org/github.com/typepress/rivet) [![status](https://sourcegraph.com/api/repos/github.com/typepress/rivet/.badges/status.png)](https://sourcegraph.com/github.com/typepress/rivet)

专注路由.
[简洁](#简洁), [贪心匹配](#贪心匹配), [支持注入](#注入), [可定制](#定制), [深度解耦](#深度解耦)的 http 路由管理器.

[examples][] 目录中有几个例子, 方便您了解 Rivet.

任何问题, 分享可至 [issues](/typepress/rivet/issues) , [wiki](/typepress/rivet/wiki) 

这里有个路由专项评测 [go-http-routing-benchmark][benchmark].

Rivet 版本号采用 [Semantic Versioning](http://semver.org/).

简洁
====

Rivet 使用常规风格.

示例: 复制到本地运行此代码, 然后后点击 [这里](http://127.0.0.1:3000/Rivet)

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
带参数的 handler.
params 是从 URL.Path 中提取到的参数.
params 的另一种风格是 PathParams/Scene. 参见 Scene.
*/
func Hi(params rivet.Params, rw http.ResponseWriter) {
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

上例中 `"/"` 是无参数路由. `"/:who"` 是有参数路由, 参数名为 "who".

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

以 api.github.com 真实路由为例:
```go
mux.Get("/users/:user/events", Events)
mux.Get("/users/:user/events/orgs/:org", Events)
```

因为都用 Events 函数作为 handler,  可以这样写:
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

*提示: 如果 Params 类型不适合您, 请看 [Scene](#scene) 部分.*

贪心匹配
========

通常 Router 库都能支持静态路由, 参数路由, 可选尾部斜线等, Rivet 也同样支持, 而且做的更好. 下面这些路由并存, 同样能正确匹配:

```
"/",
"/**",
"/hi",
"/hi/**",
"/hi/path/to",
"/hi/:name/to",
"/:name",
"/:name/path/?",
"/:name/path/to",
"/:name/path/**",
"/:name/**",
```

当 URL.Path 为
```
"/xx/zzz/yyy"
```

时 `"/:name/**` 会被匹配, 它的层级比较深, 这符合贪心匹配原则.
使用者有可能困惑, 因为 `"/**"` 和 `"/:name/**"` 都可以匹配 `"/xx/zzz/yyy"`. 记住贪心匹配原则, 否则避免这种用法即可. 后文会详细介绍路由风格.

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

现实中会有一些需求, 比如服务器对不同用户在相同 URL.Path 下有不同响应, 也就是用户角色控制. 使用注入后会很简单.

```go
// 用户角色控制示意, 简单的定义为 string
type Role string

/**
在 handler 函数中加上 rivet.Context 参数即可用注入标记用户角色,
*/
func UserRole(c rivet.Context) {
    // Context.Request() 返回 *http.Request
    req := c.Request()

    // 通常根据 session 确定用户角色.
    session := req.Cookie("session").Value

    /**
    这里只是示意代码, 现实中的逻辑更复杂.
    用注入函数 Map, 把用户角色关联到上下文.
    */
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
DelComments 删除评论, role 参数由前面的 UserRole 注入上下文.
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

func main() {
    // ...
    //注册路由:
    mux.Get("/del/comments/:id", UserRole, DelComments)
    // ...
}
```

这个例子中, `"/del/comments/:id"` 被匹配后, 先执行 UserRole, 把用户角色关联到 Context, 因为 UserRole 没有对 http.ResponsWriter 进行写操作, DelComments 会被执行. Rivet 负责传递 DelComments 需要的参数 UserRole 等. DelComments 获得 role 变量进行相应的处理, 完成角色控制.

*提示: 如果 Rivet 发现 ResponsWriter 写入任何内容, 认为响应已经完成, 不再执行后续 handler*

定制
====

事实上, 上例中的 UserRole 很多地方都要用, 每次注册路由都带上 UserRole 很不方便.
通常在路由匹配之前执行 UserRole. 可以这样用:

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

方法也很多, 这只是最简单的一种.

*提示: 善用 [Filter][] 可真正起到滤器请求的作用.*

深度解耦
========

解耦使应用能切入到路由执行流程中的每一个环节, 达到高度定制. Rivet 在不失性能的前提下, 对解耦做了很多努力. 了解 Rivet 的类型和接口有助于深度定制路由流程.

* [Params][] 保存 URL.Path 中的参数
* [Filter][] 检查/转换 URL.Path 参数, 亦可过滤请求.
* [Node][] 保存 handler, 每个 Node 都拥唯一 id.
* [Trie][] 匹配 URL.Path, 调用 Filter, 调用 Params 生成器.
    匹配到的 Trie.id 和 Node.id 是对应的.
* [Context][] 维护上下文, 处理 handler. 内置 Rivet 实现了它.
* [Router][] 路由管理器, 把上述对象联系起来, 完成路由功能.

他们是如何解耦:

Params 无其它依赖, 有 PathParams 风格可选. 自定义 [ParamsReceiver][] 定制.

Filter 接口无其它依赖. 自定义 [FilterBuilder][] 定制.

Node 接口依赖 Context. 自定义 [NodeBuilder][] 定制. 可以建立独立的 Context.

Trie 是路由匹配的核心, 依赖 Filter, ParamsReceiver. 它们都可定制.

Context 接口依赖 ParamsReceiver, 这只是个函数, 最终也是无依赖的. Context 用了注入, 可能您的应用并不需要注入, 不用它即可.

[Rivet][] 是内置的 Context 实现, 是个 struct, 可以扩展.

*提示: 注入是透明的, 不使用不产生开销, 使用了开销也不高.*

Router 依赖上述所有. 了解函数类型 [NodeBuilder][] 和 [Riveter][] 定制自己的 Node, Context.

定制使用大概分两类:

    底层: 直接使用 Trie, 构建自己的 Node, ParamsReceiver, Context, Router.
          需要了解 TypeIdOf, NewContext, NewNode, ParamsFunc, FilterFunc.
    扩展: 使用 Router, 自定义 Context 生成器, 或者扩展 Rivet.

*提示: 底层定制 Trie 需要 FilterBuilder, 如果 Path 参数无类型. 直接用 nil 替代, Trie 可以正常工作.*

下文展示扩展定制方法.

自定义 Context 生成器:
```go
// 自定义 Context 生成器, 实现真正的 http.Flusher
func MyRiveter(rw http.ResponseWriter, req *http.Request) rivet.Context {

    // 构建自己的 http.Flusher
    rw = MyResponseWriterFlusher(rw) 
    c := new(rivet.NewContext(rw, req)) // 依旧使用 rivet.Rivet
    return c
}
```

rivet 内置的 ResponseWriteFakeFlusher 是个伪 http.Flusher, 只是有个 Flush() 方法, 没有真的实现 http.Flusher 功能. 如果您需要真正的 Flusher 需要自己实现.

实现自己的 Context 很容易, 善用 Next 和 Invoke 方法即可.

举例:

```go
/**
扩展 Context, 实现 Before Handler.
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
        // 执行 Before Handler
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
调用 Context.Invoke, 插入执行另外的 handler.
MyInvoke 插入执行 SendStaticFile, 这和直接调用 SendStaticFile 不同.
这样的 SendStaticFile 可以使用上下文关联变量, 就像上文讲的角色控制.
而 MyInvoke 不必关心 SendStaticFile 所需要的参数, 那可以由别的代码负责.
*/
func MyInvoke(c rivet.Context) {
    c.Invoke(SendStaticFile)
}

/**
发送静态文件, 参数 root 是前期代码关联好的.
现实中简单的改写 req.URL.Path, 无需 root 参数也是可行的.
*/
func SendStaticFile(root http.Dir, rw http.ResponseWriter, req *http.Request) {
    // send ...
}
```


路由风格
========

Rivet 对路由 pattern 支持丰富.

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

上面的路由只有参数名, 数据类型都是 string. Rivet 还支持带类型的 pattern.

示例:
```
"/news/:cat/:id uint"
```

":id uint" 表示参数名是 "id", 数据要符合 "uint" 的要求.

"uint" 是内置的 Filter class, 参见 [FilterClass][], 您可以注册新的 class.

示例: 可选尾斜线
```
"/news/?"
```

可匹配:
```
"/news"
"/news/"
```

*提示: "/?" 只能在尾部出现*.

除了可选尾斜线, 路由风格可归纳为:

```
"/path/to/prefix:pattern/:pattern/:"
```

其中 "path", "to","prefix" 是占位符, 表示固定字符, 称为定值.
":pattern" 格式为:

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

*提示: 含有 class 才会生成 Filter, 否则被优化处理*

您也许注意到, 这里没有正则, 自定义 Filter 怎么执行由定制者控制, 包括正则.

*提示: 正则中不能含有 "/".*

Scene
=====

路由风格支持带类型的参数, Filter 检查时可能会对参数进行类型转换, interface{} 方便保存转换后的结果, 后续代码无需再次检查转换, 所以 Params 定义成这样:

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
params 类型为 PathParams, 是从 URL.Path 中提取到的参数.
PathParams 和 Scene 配套使用.
*/
func Hi(params rivet.PathParams, rw http.ResponsWriter) {
    io.WriteString(rw, "Hi "+params["who"])
}

func main() {
    
    // 传递 NewScene, 采用 PathParams 风格
    mux := rivet.NewRouter(rivet.NewScene)

    mux.Get("/:who", Hi) // 参数名设定为 "who"
    
    http.ListenAndServe(":3000", mux) 
}
```

*提示: PathParams 和 NewScene 配套使用. 事实上 Context 采用 All-In-One 的设计方式, 具体实现不必未完成所有接口, 使用方法配套即可.*

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


[Node]: //gowalker.org/github.com/typepress/rivet#Node
[NodeBuilder]: //gowalker.org/github.com/typepress/rivet#NodeBuilder
[Filter]: //gowalker.org/github.com/typepress/rivet#Filter
[FilterBuilder]: //gowalker.org/github.com/typepress/rivet#FilterBuilder
[FilterClass]: //gowalker.org/github.com/typepress/rivet#_variables
[Params]: //gowalker.org/github.com/typepress/rivet#Params
[ParamsReceiver]: //gowalker.org/github.com/typepress/rivet#ParamsReceiver
[Scene]: //gowalker.org/github.com/typepress/rivet#Scene
[Trie]: //gowalker.org/github.com/typepress/rivet#Trie
[Context]: //gowalker.org/github.com/typepress/rivet#Context
[Router]: //gowalker.org/github.com/typepress/rivet#Router
[Rivet]: //gowalker.org/github.com/typepress/rivet#Rivet
[Rivet.Get]: //gowalker.org/github.com/typepress/rivet#Rivet_Get
[Rivet.Invoke]: //gowalker.org/github.com/typepress/rivet#Rivet_Invoke
[Riveter]: //gowalker.org/github.com/typepress/rivet#Riveter
[benchmark]: //github.com/julienschmidt/go-http-routing-benchmark
[examples]: //github.com/typepress/rivet/tree/master/examples