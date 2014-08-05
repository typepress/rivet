package rivet

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func assert(t *testing.T, got interface{}, want interface{}, s ...string) {
	if fmt.Sprint(got) != fmt.Sprint(want) {

		t.Fatal(
			s,
			"\ngot :", got,
			"\nwant:", want,
		)
	}
}

func hasMatch(t *testing.T, mux Router, method, urlPath string) {
	p, r := mux.Match(method, urlPath)
	if r == nil {
		t.Fatal("want an Route , but got nil:", urlPath)
	}
	if len(p) == 0 {
		t.Fatal("want Params , but got nil:", urlPath)
	}
}

func badMatch(t *testing.T, mux Router, method, urlPath string) {
	_, r := mux.Match(method, urlPath)
	if r != nil {
		t.Fatal("want nil, but got an Route:")
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
		mux.Add(badParams[i], badParams[i+1])
	}
	for i := 0; i < len(badParams); i += 3 {
		badMatch(t, mux, badParams[i], badParams[i+2])
	}
}

func Test_HasParams(t *testing.T) {
	mux := NewRouter(nil)
	for i := 0; i < len(hasParams); i += 3 {
		mux.Add(hasParams[i], hasParams[i+1])
	}
	for i := 0; i < len(hasParams); i += 3 {
		hasMatch(t, mux, hasParams[i], hasParams[i+2])
	}
}

func Test_Routing(t *testing.T) {
	mux := NewRouter(nil).(*router)
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

func TestTrie(t *testing.T) {
	var child *Trie
	root := NewRootRoute()

	routes := []string{
		"/",
		"/hi",
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
	}

	for _, path := range routes {
		recv := catchPanic(func() {
			child = root.Add(path)
			child.base = new(base)
		})
		if recv != nil {
			t.Fatalf("panic *trie.Add '%s': %v", path, recv)
		}
		if child == nil {
			t.Errorf("*trie.Add failed '%s'", path)
		}
	}

	for _, path := range routes {
		_, child := root.Match(path)

		if child == nil {
			t.Errorf("*trie.Match failed '%s'", path)
		}
		if child.base == nil {
			t.Errorf("*trie.Match route is nil'%s'", path)
		}
	}
	//p, _ := root.Match("/1/:id/2")
	//println(p)
}

func catchPanic(testFunc func()) (recv interface{}) {
	defer func() {
		recv = recover()
	}()

	testFunc()
	return
}
