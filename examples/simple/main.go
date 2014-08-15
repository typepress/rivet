package main

import (
	"net/http"

	"github.com/typepress/rivet"
)

func empty() { println("empty") }

func HelloWorld(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Hello World"))
}

func Hello(params rivet.Params, w http.ResponseWriter) {
	w.Write([]byte("Hello " + params["name"].(string)))
}

func CatchAll(c rivet.Context) {
	c.WriteString("CatchAll:" + c.GetParams().Get("*"))
}

func Go(c rivet.Context) {
	c.Map("Death is coming. Let's Go!")
}

func GoGo(c rivet.Context) {
	// 获取 string 类型标识
	id := rivet.TypeIdOf("string")

	c.WriteString(c.GetParams().Get("name") + "! " +
		c.Get(id).(string)) // 类型转换
}

func main() {

	// 传递 nil, NewRouter 使用内部的 Rivet 实现
	mux := rivet.NewRouter(nil)

	mux.Get("/", HelloWorld)
	mux.Get("/empty", empty)
	mux.Get("/hi/:name/path/to", Hello)
	mux.Get("/hi/:name/path", Hello)
	mux.Get("/:name", Go, GoGo)
	mux.Get("/hi/**", CatchAll)

	// rivet.Router 符合 http.Handler 接口
	http.ListenAndServe(":3000", mux)
}
