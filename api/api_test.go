package api

import (
	"fmt"
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
	apiHandle := http.HandlerFunc(RouteAPI)

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
	apiHandle.ServeHTTP(w3, r3)

	core.LogDebug.Println(core.FormatRequest(r3))

	// "internal" posts get a 302
	if w3.Code != 404 {
		t.Errorf("expected a 404, got: %d", w3.Code)
	}
}

func FuzzTestRouteAPI(f *testing.F) {
	srv := httptest.NewServer(http.HandlerFunc(RouteAPI))
	defer srv.Close()

	f.Add("true", "akeyword", 823498, "a title", "spaced tags", "http://www.example.com", "enable", "burn", "otherlists")
	f.Add("false", "akeyword", -56, "a title", "spaced tags", "http://www.example.com", "disable", "6d2h", "otherlists")

	f.Fuzz(func(t *testing.T, i string, k string, id int, ti string, tt string, u string, ll string, e string, o string) {
		linkPost := url.Values{}
		linkPost.Set("internal", i)
		linkPost.Set("returnto", k)
		linkPost.Set("linkid", fmt.Sprint(id))
		linkPost.Set("title", ti)
		linkPost.Set("tag", tt)
		linkPost.Set("url", u)
		linkPost.Set("linklog", ll)
		linkPost.Set("expiretime", e)
		linkPost.Set("otherlists", o)
		encoded := strings.NewReader(linkPost.Encode())

		_, err := http.DefaultClient.Post(fmt.Sprintf("%s/api/link/", srv.URL), "application/x-www-form-urlencoded", encoded)

		if err != nil {
			t.Errorf("Error: %v", err)
		}
		// TODO: check for status codes with certain inputs
		// For now, fuzzing for crashes is fine.
	})
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
