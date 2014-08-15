package rivet

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type debugParams struct {
	maps map[string]interface{}
	diff int
}

func newDebugParams() *debugParams {
	return &debugParams{maps: map[string]interface{}{}}
}
func (p *debugParams) ParamsReceiver(key, text string, val interface{}) {
	p.maps[key] = val
}

// ParamsNames 接收合法的参数名
func (p *debugParams) ParamsNames(names map[string]bool) {
	if len(p.maps) != len(names) {
		clear := len(names) == 0
		for k, _ := range p.maps {
			if clear || !names[k] {
				p.diff++
			}
		}
		for k, _ := range names {
			if p.maps[k] == nil {
				p.diff++
			}
		}

	}
}

func assert(t *testing.T, got interface{}, want interface{}, s ...string) {
	if fmt.Sprint(got) != fmt.Sprint(want) {

		t.Fatal(
			s,
			"\ngot :", got,
			"\nwant:", want,
		)
	}
}

func TestTrie(t *testing.T) {
	var child *Trie

	root := NewRootTrie()

	routes := []string{
		"/b/",
		"/search/:query",
		"/cmd/:tool/",
		"/src/*filepath",
		"/x",
		"/x/y",
		"/y/",
		"/y/z",
		"/0/:id",
		"/0/:id/1",
		"/1/:id/",
		"/1/:id/2",
		"/aa",
		"/a/",
		"/do",
		"/doc",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/no/a",
		"/no/b",
		"/api/hello/:name",
		"/empty",
		"/",
		"/hi",
		"/hi/**",
		"/hi/path/to",
		"/hi/:name/to",
		"/:name",
		"/:name/path",
		"/:name/path/to",
		"/:name/path/**",
		"/:name/**",
	}

	for i, path := range routes {
		recv := catchPanic(func() {
			child = root.Add(path, nil)
			child.id = i + 1
		})
		if recv != nil {
			t.Fatalf("panic *trie.Add '%s': %v", path, recv)
		}
	}

	for i, path := range routes {
		p := newDebugParams()
		child := root.Match(path, p, nil, nil)

		if child == nil {
			t.Errorf("*trie.Match failed '%s'", path)
		}
		if child.id != i+1 {
			t.Errorf("*trie.Match route is nil'%s'", path)
		}

		_, names := analyzePath(path)
		if p.diff != 0 {
			t.Errorf(
				"*trie.Match: incorrect Params:\n %s\n child.name: %s\n child.names: %#v\n path names: %#v\n params: %#v\n",
				path, child.name, child.names, names, p.maps)
		}
	}

}

var badParams = []string{
	"GET", "/:mad uint", "/123a",
}

var hasParams = []string{
	"GET", "/:mad uint", "/12387",
	"GET", "/catch/all**", "/catch/all12387",
}

func Test_BadParams(t *testing.T) {
	mux := NewRouter(nil)
	for i := 0; i < len(badParams); i += 3 {
		mux.Handle(badParams[i], badParams[i+1])
	}

	for i := 0; i < len(badParams); i += 3 {

		method, urlPath := badParams[i], badParams[i+2]

		node := mux.Match(method, urlPath, nil, nil, nil)
		if node.Id() != 0 {
			t.Fatal("want got NotFound, but got ", node.Id(), urlPath)
		}
	}
}

func Test_HasParams(t *testing.T) {
	mux := NewRouter(nil)
	for i := 0; i < len(hasParams); i += 3 {
		mux.Handle(hasParams[i], hasParams[i+1])
	}

	for i := 0; i < len(hasParams); i += 3 {

		p := newDebugParams()
		method, urlPath := hasParams[i], hasParams[i+2]

		node := mux.Match(method, urlPath, p, nil, nil)

		if node.Id() == 0 {
			t.Fatalf("NotFound : %s", urlPath)
		}
		if len(p.maps) == 0 {
			t.Fatal("want Params , but got nil:", node.Id(), urlPath)
		}
		if p.diff != 0 {
			t.Fatalf("NotFound : params received to more %s", urlPath)
		}
	}
}

var otsRoutes = []string{
	"GET", "/:mad uint/?", "/12387",
	"GET", "/:mad uint/?", "/12387/",
	"GET", "/catch/all/?", "/catch/all",
	"GET", "/catch/all/?", "/catch/all/",
}

func Test_OTS(t *testing.T) {
	routes := otsRoutes
	mux := NewRouter(nil)

	i := 0
	for i = 0; i < len(routes); i += 3 {
		method, urlPath := routes[i], routes[i+1]
		recv := catchPanic(func() {
			mux.Handle(method, urlPath)
		})

		if recv != nil {
			t.Fatalf("panic Handle '%s': %v", urlPath, recv)
		}
	}
	// mux.RootTrie("GET").Print("")
	for i := 0; i < len(routes); i += 3 {

		p := newDebugParams()
		method, urlPath := routes[i], routes[i+2]

		node := mux.Match(method, urlPath, p, nil, nil)

		if node.Id() == 0 {
			t.Fatalf("NotFound : %s", urlPath)
		}
		if i < 2 && len(p.maps) == 0 {
			t.Fatal("want Params , but got nil:", node.Id(), urlPath)
		}
		if p.diff != 0 {
			t.Fatalf("NotFound : params received to more %s", urlPath)
		}
	}
}

var zRoutes = []string{
	"/",
	"/",
	"/hi",
	"/hi",
	"/hi/**",
	"/hi/z",
	"/hi/path/to",
	"/hi/path/to",
	"/hi/:name/to",
	"/hi/:name/to",
	"/:name",
	"/:name",
	"/:name/path",
	"/:name/path",
	"/:name/path/to",
	"/:name/path/to",
	"/:name/path/**",
	"/:name/path/z",
	"/:name/**",
	"/:name/z",
}

func Test_Z(t *testing.T) {
	routes := zRoutes
	mux := NewRouter(nil)

	i := 0
	for i = 0; i < len(routes); i += 2 {
		urlPath := routes[i]
		recv := catchPanic(func() {
			mux.Get(urlPath)
		})

		if recv != nil {
			t.Fatalf("panic Handle '%s': %v", urlPath, recv)
		}
	}

	root := mux.RootTrie("GET")
	//root.Print("")

	for i := 0; i < len(routes); i += 2 {

		urlPath := routes[i+1]

		trie := root.Match(urlPath, nil, nil, nil)

		if trie.id != i/2+1 {
			t.Fatalf("missing : %d %s", trie.id, urlPath)
		}
	}
}

func Test_Routing(t *testing.T) {
	mux := NewRouter(nil)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	want := func(g interface{}, w interface{}) {
		assert(t, g, w)
	}

	var restr string
	Do := func(res, method, urlStr string) {
		restr = res
		req, _ := http.NewRequest(method, srv.URL+urlStr, nil)
		http.DefaultClient.Do(req)
	}

	result := ""
	mux.Get("/repos/:owner/:repo", func(params Params) {
		want(params["owner"], ":git")
		want(params["repo"], ":hub")
		result += restr + "github"
	})

	mux.Post("/bar/:cat", func(params Params) {
		want(params["cat"], "bat")
		result += restr + ":cat"
	})

	mux.Get("/foo", func(req *http.Request) {
		result += restr + "fix"
	})
	mux.Get("/foo/*", func(req *http.Request) {
		result += restr + ":"
	})

	mux.Get("/foo/prefix:", func(req *http.Request) {
		result += restr + "prefix*"
	})

	mux.Post("/foo/post:id", func(params Params) {
		want(params["id"], 6)
		result += restr + "post"
	})

	mux.Patch("/bar/:id", func(params Params) {
		want(params["id"], "foo")
		result += restr + "id"
	})

	mux.Any("/any/foo:ID uint", func(params Params) {
		want(params["ID"], 6000)
		result += restr + "ID"
	})

	mux.Any("/any/catch**", func(params Params) {
		want(params["*"], "all")
		result += restr + ":all"
	})
	Do("1", "POST", "/bar/bat")
	Do("2", "GET", "/foo")
	Do("3", "GET", "/foo/a")
	Do("4", "GET", "/foo/prefix*")
	Do("5", "PATCH", "/bar/foo")
	Do("6", "POST", "/foo/post6")
	Do("7", "POST", "/any/foo6000")
	Do("8", "GET", "/any/foo6000")
	Do("9", "GET", "/repos/:git/:hub")
	Do("0", "GET", "/any/catchall")

	want(result, "1:cat2fix3:4prefix*5id6post7ID8ID9github0:all")
}

func catchPanic(testFunc func()) (recv interface{}) {
	defer func() {
		recv = recover()
	}()

	testFunc()
	return
}

func Test_Context(t *testing.T) {
	mux := NewRouter(nil)

	mux.Get("/:name", func(p Params) {
		if p.Get("name") != "PathParams" {
			t.Fatal(p)
		}
	})

	mux.Get("/map/:name", func(p map[string]interface{}) {
		if p["name"].(string) != "PathParams" {
			t.Fatal(p)
		}
	})

	req, _ := http.NewRequest("GET", "/PathParams", nil)
	mux.ServeHTTP(nil, req)
	req, _ = http.NewRequest("GET", "/map/PathParams", nil)
	mux.ServeHTTP(nil, req)
}

func Test_Scene(t *testing.T) {
	mux := NewRouter(NewScene)

	mux.Get("/:name", func(p PathParams) {
		if p["name"] != "PathParams" {
			t.Fatal(p)
		}
	})

	mux.Get("/map/:name", func(p map[string]string) {
		if p["name"] != "PathParams" {
			t.Fatal(p)
		}
	})

	req, _ := http.NewRequest("GET", "/PathParams", nil)
	mux.ServeHTTP(nil, req)
	req, _ = http.NewRequest("GET", "/map/PathParams", nil)
	mux.ServeHTTP(nil, req)
}

func Test_Invoke(t *testing.T) {
	result := ""
	invoke := func(c Context) {
		result = c.Get(TypeIdOf("")).(string)
	}

	mux := NewRouter(nil)

	mux.Get("/", func(c Context) {
		c.Map("Invoke")
	}, func(c Context) {
		c.Invoke(invoke)
	})

	req, _ := http.NewRequest("GET", "/", nil)
	mux.ServeHTTP(nil, req)
	if result != "Invoke" {
		t.Fatalf("want `Invoke`, bug got ", result)
	}
}
