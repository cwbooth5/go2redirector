package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/cwbooth5/go2redirector/core"
)

func TestRouteAPIBasic(t *testing.T) {
	w1 := httptest.NewRecorder()
	r1, _ := http.NewRequest("GET", "", nil)
	helpHandle := http.HandlerFunc(RouteAPI)
	helpHandle.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Errorf("Home page didn't return %v", http.StatusOK)
	}
}

func TestRouteAPIKeywords(t *testing.T) {
	// /api/keywords
	w2 := httptest.NewRecorder()
	r2, _ := http.NewRequest("GET", "/api/keywords", nil)
	helpHandle := http.HandlerFunc(RouteAPI)
	helpHandle.ServeHTTP(w2, r2)
	if w2.Header().Get("Content-Type") != "application/json" {
		t.Fail()
	}
	if w2.Header().Get("Cache-Control") != "max-age=60" {
		t.Fail()
	}
	if w2.Code != http.StatusFound {
		t.Fail()
	}
}

/*
	Notable test points:

- link ID is 0
- burn is the expiretime
- type == internal
*/
func TestRouteAPILink(t *testing.T) {
	// api/link
	helpHandle := http.HandlerFunc(RouteAPI)
	linkPost := url.Values{}
	linkPost.Set("internal", "true")
	linkPost.Set("returnto", "mykeyword")
	linkPost.Set("linkid", "0") // new link
	linkPost.Set("title", "my hovercraft is full of eels")
	linkPost.Set("tag", "tag1 tag tag3")
	linkPost.Set("url", "www.example.com")
	linkPost.Set("linklog", "enable")
	linkPost.Set("expiretime", "burn") // test of burn after reading feature
	linkPost.Set("otherlists", "nonexistentlist")

	core.LogDebug.Printf("%s", linkPost)
	encoded := strings.NewReader(linkPost.Encode())
	core.LogDebug.Println(encoded)
	r3, _ := http.NewRequest("POST", "/api/link/", encoded)
	r3.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w3 := httptest.NewRecorder()
	helpHandle.ServeHTTP(w3, r3)

	core.LogDebug.Println(core.FormatRequest(r3))

	// "internal" posts get a 302
	if w3.Code != 302 {
		t.Errorf("expected a 302, got: %d", w3.Code)
	}
}

// This tests for a nasty bug if a bad link ID is provided in the form.
func TestRouteAPIBadLinkID(t *testing.T) {
	// api/link
	helpHandle := http.HandlerFunc(RouteAPI)

	linkPost := url.Values{}
	linkPost.Set("internal", "true")
	linkPost.Set("returnto", "mykeyword")
	linkPost.Set("linkid", "392394") // this does not exist
	linkPost.Set("title", "my hovercraft is full of eels")
	linkPost.Set("tag", "tag1 tag tag3")
	linkPost.Set("url", "www.example.com")
	linkPost.Set("linklog", "enable")
	linkPost.Set("expiretime", "1h")
	linkPost.Set("otherlists", "nonexistentlist")

	core.LogDebug.Printf("%s", linkPost)
	encoded := strings.NewReader(linkPost.Encode())
	core.LogDebug.Println(encoded)
	r3, _ := http.NewRequest("POST", "/api/link/", encoded)
	r3.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w3 := httptest.NewRecorder()
	helpHandle.ServeHTTP(w3, r3)

	core.LogDebug.Println(core.FormatRequest(r3))

	// "internal" posts get a 302
	if w3.Code != 404 {
		t.Errorf("expected a 404, got: %d", w3.Code)
	}
}

// func TestRouteAPIDeleteLink(t *testing.T) {
// 	// api/link
// 	// A linkdb is needed to get a link we can delete
// 	core.LinkDataBase = core.MakeNewLinkDatabase()
// 	core.LinkDataBase.NextLinkID = 1
// 	aLink, err := core.MakeNewlink("www.reddit.com", "probably reddit")
// 	if err != nil {
// 		t.Log(err)
// 		t.FailNow()
// 	}
// 	akw, _ := core.MakeNewKeyword("mykeyword")
// 	aList := core.MakeNewList(akw, aLink)
// 	core.LinkDataBase.Couple(aList, aLink)

// 	helpHandle := http.HandlerFunc(RouteAPI)

// 	linkPost := url.Values{}
// 	linkPost.Set("internal", "true")
// 	linkPost.Set("returnto", "mykeyword")
// 	linkPost.Set("linkid", "1") // this does not exist
// 	linkPost.Set("title", "my hovercraft is full of eels")
// 	linkPost.Set("tag", "tag1 tag tag3")
// 	linkPost.Set("url", "www.example.com")
// 	linkPost.Set("delete", "true") // delete the link
// 	linkPost.Set("expiretime", "1h")
// 	linkPost.Set("otherlists", "nonexistentlist anotherlist")

// 	core.LogDebug.Printf("%s", linkPost)
// 	encoded := strings.NewReader(linkPost.Encode())
// 	core.LogDebug.Println(encoded)
// 	r3, _ := http.NewRequest("POST", "/api/link/", encoded)
// 	r3.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	w3 := httptest.NewRecorder()
// 	helpHandle.ServeHTTP(w3, r3)

// 	core.LogDebug.Println(core.FormatRequest(r3))

// 	// "internal" posts get a 302
// 	if w3.Code != 410 {
// 		t.Errorf("expected a 410, got: %d", w3.Code)
// 	}
// }

// If this isn't here, logging calls during functions we are testing cause a SEGV
func init() {
	core.ConfigureLogging(true, os.Stdout)
	core.SYNC <- 1
}
