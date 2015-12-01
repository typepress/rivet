package main

import (
	"io"
	"net/http"

	"github.com/typepress/rivet"
)

func empty() { println("empty") }

func HelloWorld(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Hello World"))
}

func Hello(params rivet.Params, w http.ResponseWriter) {
	io.WriteString(w, "Hello ")
	w.Write([]byte(params.Get("name")))
}

func CatchAll(c rivet.Context) {
	c.WriteString("CatchAll:" + c.Get("**"))
}

func Go(c rivet.Context) {
	c.Map("Death is coming. Let's Go!")
}

func GoGo(c rivet.Context) {
	var s string

	i, has := c.Var(rivet.TypePointerOf("string"))
	if has {
		s, _ = i.(string)
	}

	c.WriteString(c.Get("name") + "! " + s)
}

func main() {

	mux := rivet.New()

	mux.Get("/", HelloWorld)
	mux.Get("/empty", empty)
	mux.Get("/hi/:name/path/to", Hello)
	mux.Get("/hi/:name/path", Hello)
	mux.Get("/:name", Go, GoGo)
	mux.Get("/hi/**", CatchAll)

	// rivet.Router 符合 http.Handler 接口
	http.ListenAndServe(":3000", mux)
}
