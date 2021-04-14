package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/cwbooth5/go2redirector/core"
)

func TestNewKeyword(t *testing.T) {
	// we lowercase everything
	ans, _ := core.MakeNewKeyword("OTTers")
	if ans != "otters" {
		t.Fail()
	}
	// all these characters are legal and acceptable
	ans, _ = core.MakeNewKeyword("OTTers__-~")
	if ans != "otters__-~" {
		t.Fail()
	}
}

func TestMyNewLink(t *testing.T) {
	_, err := core.MakeNewlink("http://website.net/index.php", "a title")
	if err != nil {
		t.Logf("This vanilla new link should not have an error of: %s\n", err)
		t.Fail()
	}
}

func TestRemoveEmptyKeyword(t *testing.T) {
	db := core.MakeNewLinkDatabase()
	aLink, err := core.MakeNewlink("www.reddit.com", "probably reddit")
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	// db.CommitNewLink(aLink)
	akw, _ := core.MakeNewKeyword("sites")
	aList := core.MakeNewList(akw)
	// newList := db.CommitNewList(akw, aList)
	db.Couple(aList, aLink)
	if len(aList.Links) != 1 {
		t.Fail()
	}
	db.Decouple(aList, aLink)

	// Decouple will remove the link from the keyword and vice-versa.
	if _, exists := db.Lists[akw]; exists {
		t.Logf("Delete of last link did not delete the list entirely from the DB.")
		t.Fail()
	}
	// TODO: need to figure out here if we will remove links with no more associated keywords
	if linkval, exists := db.Links[aLink.ID]; exists {
		t.Logf("Link should have no remaining memberships! Found value: %v", linkval.Lists)
		t.Fail()
	}
}

func TestAddMultipleLinksToList(t *testing.T) {
	/*
		create a new link
		create another
		create a keyword and commit it
		add both links to the list
	*/
	db := core.MakeNewLinkDatabase()
	aLink, _ := core.MakeNewlink("www.reddit.com", "probably reddit")
	bLink, _ := core.MakeNewlink("www.digg.com", "yeah I just went there")
	cLink, _ := core.MakeNewlink("www.geocities.com", "wow")

	db.CommitNewLink(aLink)
	db.CommitNewLink(bLink)
	db.CommitNewLink(cLink)
	akw, _ := core.MakeNewKeyword("otters")
	aList := core.MakeNewList(akw)
	if len(aList.Links) != 0 {
		t.Fail()
	}
	// now add that second link to the list.
	db.Couple(aList, bLink)
	if len(aList.Links) != 1 {
		core.PrintList(*aList)
		t.Fail()
	}

	// and add a third
	db.Couple(aList, cLink)
	if len(aList.Links) != 2 {
		core.PrintList(*aList)
		t.Fail()
	}
}

func TestBehaviorChange(t *testing.T) {
	/*
		Change behavior from the default of 'list' to 'random'.
	*/
	db := core.MakeNewLinkDatabase()
	aLink, _ := core.MakeNewlink("www.reddit.com", "probably reddit")
	db.CommitNewLink(aLink)
	akw, _ := core.MakeNewKeyword("otters")
	aList := core.MakeNewList(akw)
	db.Couple(aList, aLink)
	aList.Behavior = core.RedirectToRandom

	if aList.Behavior != -4 {
		core.PrintList(*aList)
		t.Fail()
	}
}

func TestBasicAddRemove(t *testing.T) {
	db := core.MakeNewLinkDatabase()

	// create a link with no list yet
	aLink, err := core.MakeNewlink("www.example.com", "a title")
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	db.CommitNewLink(aLink)

	if db.NextLinkID != 2 {
		t.Logf("Next link ID expected: 2, got: %d", db.NextLinkID)
		t.FailNow()
	}

	// create a new keyword
	aKw, _ := core.MakeNewKeyword("missiles")
	// create a new list of links and add that link to the above keyword
	aList := core.MakeNewList(aKw)
	db.Couple(aList, aLink)

	if aList.Clicks != 0 || aList.Behavior != -2 || len(aList.Links) != 1 {
		t.Log("initial values. clicks == 0, behavior == list, exactly one link with id 0")
		t.FailNow()
	}

	fmt.Printf("LEN: %d\n\n", len(aList.Links))
	if aList.Links[1].ID != 1 {
		t.Logf("Link ID seen: %d, expected: 1", aList.Links[1].ID)
		t.FailNow()
	}

	db.Decouple(aList, aLink)

	// The list should be gone now, since it holds zero links.
	if _, exists := db.Lists[aKw]; exists {
		t.Logf("Keyword/list was still present after decoupling the last link: %s", aKw)
		t.FailNow()
	}
}

func TestAddToOtherList(t *testing.T) {
	/*
		When a user couples a link with one of the 'otherlists', there are two possibilities.
		1. They are coupling with a list that already exists.
		2. They are coupling with a list which does not exist yet.

		The coupling should result in a new tagbinding map entry for the added link.
	*/
	db := core.MakeNewLinkDatabase()

	// create a link with no list yet
	aLink, _ := core.MakeNewlink("www.example.com", "a title")
	db.CommitNewLink(aLink)

	// create a new list and add the above link.
	// link isn't important, just has to be there to make the list whole
	aKw, _ := core.MakeNewKeyword("missiles")
	aList := core.MakeNewList(aKw)
	db.Couple(aList, aLink)
	if len(db.Lists[aKw].TagBindings) == 0 {
		t.Logf("Tagbindings were len == 0 on new list: %s", aKw)
		t.FailNow()
	}

	bLink, _ := core.MakeNewlink("www.example.com/foo", "another title")
	db.CommitNewLink(bLink)
	// create a second list and add a new link to it.
	bKw, _ := core.MakeNewKeyword("tanks")
	bList := core.MakeNewList(bKw)
	db.Couple(bList, bLink)

	// Coupling link A with list B should result in a tagbinding on list B
	db.Couple(bList, aLink)
	if _, exists := bList.TagBindings[aLink.ID]; !exists {
		t.Logf("tag binding was not present in otherlist: %s", bList.Keyword)
		t.FailNow()
	}

}

/*
path len == 1
setup: keyword untagged with URL with no subs
test: "keyword" from user, they get redirected based on behavior (freshest)

setup: change the URL to have a param
test: "keyword" with no 2nd field provided, nice error message, list page
test: "keyword/param" and they get a URL back
*/

/*
path len == 2
setup: keyword with single link, link has no params
test: HTTP GET from user with "keyword/tag", asserting that URL returned
test: another request with "keyword/tag/garbage", nice error message with list page
test:
*/

/*
path len == 3
setup: keyword with a single link, link has substitution
test: HTTP GET from user with "keyword/tag/param", asserting valid URL returned with param
*/

func TestRedirectSingleMissing(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", fmt.Sprintf("%s/mykeyword", core.ListenURL()), nil)
	helpHandle := http.HandlerFunc(routeHappyHandler)
	helpHandle.ServeHTTP(w, r)
	// keyword isn't there, we should get the list page back.
	if !strings.Contains(w.Body.String(), "/_link_/0?returnto=mykeyword") {
		fmt.Println(w.Body)
		t.Error("Body didn't indicate this was a new keyword")
	}
}

func TestRedirectSingleExists(t *testing.T) {
	// create the keyword we will request
	aLink, _ := core.MakeNewlink("www.example.com/singleexists", "does a single link exist?")
	core.LinkDataBase.CommitNewLink(aLink)
	aKw, _ := core.MakeNewKeyword("spacelasers")
	aList := core.MakeNewList(aKw)
	core.LinkDataBase.Couple(aList, aLink)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", fmt.Sprintf("%s/spacelasers", core.ListenURL()), nil)
	helpHandle := http.HandlerFunc(routeHappyHandler)
	helpHandle.ServeHTTP(w, r)
	// keyword isn't there, we should get the list page back.
	if w.Code != 307 {
		t.Errorf("We expected a 307 redirect but got: %d", w.Code)
	}

	// edit mode on the same keyword
	w2 := httptest.NewRecorder()
	r2, _ := http.NewRequest("GET", fmt.Sprintf("%s/.spacelasers", core.ListenURL()), nil)
	h2 := http.HandlerFunc(routeHappyHandler)
	h2.ServeHTTP(w2, r2)

	if w2.Code != 200 {
		t.Errorf("We expected a 200 OK but got: %d", w2.Code)
	}

	// Keyword is there, but we are in edit mode so should get the list page
	if !strings.Contains(w2.Body.String(), "does a single link exist?") {
		fmt.Println(w2.Body)
		t.Error("Body didn't indicate this was a new keyword")
	}

	// path len == 1, special link but they didn't provide a parameter == list page
	// so they can sort out all the mistakes in their miserable lives
	bLink, _ := core.MakeNewlink("www.example.com/singleexists/{1}/whatever", "this is so special yay")
	core.LinkDataBase.CommitNewLink(bLink)
	core.LinkDataBase.Couple(aList, bLink)
	w3 := httptest.NewRecorder()
	r3, _ := http.NewRequest("GET", fmt.Sprintf("%s/spacelasers", core.ListenURL()), nil)
	h3 := http.HandlerFunc(routeHappyHandler)
	h3.ServeHTTP(w3, r3)

	if w3.Code != 200 {
		t.Fail()
	}

	// now they finally get their life in order and use a damn parameter
	w4 := httptest.NewRecorder()
	r4, _ := http.NewRequest("GET", fmt.Sprintf("%s/spacelasers/foobar", core.ListenURL()), nil)
	h4 := http.HandlerFunc(routeHappyHandler)
	h4.ServeHTTP(w4, r4)

	if w4.Code != 307 {
		t.Fail()
	}

	if w4.HeaderMap.Get("Location") != "http://www.example.com/singleexists/foobar/whatever" {
		fmt.Println(w4)
		t.Fail()
	}
}

// path len == 2
func TestRedirectDoubleExists(t *testing.T) {
	// create the keyword we will request
	aLink, _ := core.MakeNewlink("www.example.com/mars", "all about mars")
	core.LinkDataBase.CommitNewLink(aLink)
	aKw, _ := core.MakeNewKeyword("planets")
	aList := core.MakeNewList(aKw)
	core.LinkDataBase.Couple(aList, aLink)

	// update tag bindings to tag this link with "mars"
	aList.TagBindings[aLink.ID][0] = "mars"

	// first test: they provided an existing keyword/tag combo via their URL bar
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", fmt.Sprintf("%s/planets/mars", core.ListenURL()), nil)
	h := http.HandlerFunc(routeHappyHandler)
	h.ServeHTTP(w, r)
	// keyword isn't there, we should get the list page back.
	if w.Code != 307 {
		t.Errorf("We expected a 307 redirect but got: %d", w.Code)
	}

	if w.HeaderMap.Get("Location") != "http://www.example.com/mars" {
		fmt.Println(w)
		t.Fail()
	}

	// next test: they provided an existing keyword/tag combo via their URL bar
	w11 := httptest.NewRecorder()
	r11, _ := http.NewRequest("GET", fmt.Sprintf("%s/planets/neptune", core.ListenURL()), nil)
	h11 := http.HandlerFunc(routeHappyHandler)
	h11.ServeHTTP(w11, r11)
	// keyword isn't there, we should get the list page back.
	if w11.Code != 200 {
		t.Errorf("We expected a 200 OK but got: %d", w11.Code)
	}

	// second test: edit mode on the keyword - you just get the list page
	w2 := httptest.NewRecorder()
	r2, _ := http.NewRequest("GET", fmt.Sprintf("%s/.planets/mars", core.ListenURL()), nil)
	h2 := http.HandlerFunc(routeHappyHandler)
	h2.ServeHTTP(w2, r2)

	if w2.Code != 200 {
		t.Errorf("We expected a 200 OK but got: %d", w2.Code)
	}

	// third test is same as the first test, but simulating them entering "planets/mars" in the search box.
	w3 := httptest.NewRecorder()
	r3, _ := http.NewRequest("GET", fmt.Sprintf("%s/?keyword=planets/mars", core.ListenURL()), nil)
	h3 := http.HandlerFunc(routeHappyHandler)
	h3.ServeHTTP(w3, r3)
	if w3.Code != 307 {
		t.Errorf("We expected a 307 redirect but got: %d", w3.Code)
	}

	if w3.HeaderMap.Get("Location") != "http://www.example.com/mars" {
		fmt.Println(w3)
		t.Fail()
	}

	// fourth test: user input a garbage keyword in the search box
	w4 := httptest.NewRecorder()
	r4, _ := http.NewRequest("GET", fmt.Sprintf("%s/?keyword=</planets>>>>/mars", core.ListenURL()), nil)
	h4 := http.HandlerFunc(routeHappyHandler)
	h4.ServeHTTP(w4, r4)
	// They get a 400 back to remind them not to send us all their garbage
	if w4.Code != 400 {
		t.Errorf("We expected a 400 BAD REQUEST but got: %d", w4.Code)
	}

	// fifth test: properly formed keyword/tag in the search box in edit mode
	w5 := httptest.NewRecorder()
	r5, _ := http.NewRequest("GET", fmt.Sprintf("%s/?keyword=.planets/mars", core.ListenURL()), nil)
	h5 := http.HandlerFunc(routeHappyHandler)
	h5.ServeHTTP(w5, r5)

	if w5.Code != 200 {
		t.Errorf("We expected a 200 OK but got: %d", w5.Code)
	}

	// final test: burn the link to destroy it
	aLink.Dtime = core.BurnTime
	w6 := httptest.NewRecorder()
	r6, _ := http.NewRequest("GET", fmt.Sprintf("%s/?keyword=planets/mars", core.ListenURL()), nil)
	h6 := http.HandlerFunc(routeHappyHandler)
	h6.ServeHTTP(w6, r6)

	if w6.Code != 307 {
		fmt.Println(w6)
		t.Errorf("We expected a 307 redirect but got: %d", w6.Code)
	}

}

func TestRedirectTripleExists(t *testing.T) {
	// create the keyword we will request, note the substitution in the URL here
	aLink, _ := core.MakeNewlink("www.example.com/continents/asia/{1}.php", "different data for a continent")
	core.LinkDataBase.CommitNewLink(aLink)
	aKw, _ := core.MakeNewKeyword("continents")
	aList := core.MakeNewList(aKw)
	core.LinkDataBase.Couple(aList, aLink)

	// update tag bindings to tag this link with "asia"
	aList.TagBindings[aLink.ID][0] = "asia"

	// first test: they provided an existing keyword/tag/parameter combo via their URL bar
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", fmt.Sprintf("%s/continents/asia/population", core.ListenURL()), nil)
	h := http.HandlerFunc(routeHappyHandler)
	h.ServeHTTP(w, r)

	if w.Code != 307 {
		t.Errorf("We expected a 307 redirect but got: %d", w.Code)
	}

	if w.HeaderMap.Get("Location") != "http://www.example.com/continents/asia/population.php" {
		fmt.Println(w)
		t.Fail()
	}
}

func TestBurnAfterReading(t *testing.T) {
	// create a keyword and a link inside which will detonate after one redirect
	aLink, _ := core.MakeNewlink("www.example.com/burned", "the arsonist has oddly-shaped feet")
	aLink.Dtime = core.BurnTime // this is important
	core.LinkDataBase.CommitNewLink(aLink)
	aKw, _ := core.MakeNewKeyword("burner")
	aList := core.MakeNewList(aKw)
	core.LinkDataBase.Couple(aList, aLink)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", fmt.Sprintf("%s/burner", core.ListenURL()), nil)
	helpHandle := http.HandlerFunc(routeHappyHandler)
	helpHandle.ServeHTTP(w, r)
	// keyword isn't there, we should get the list page back.
	if w.Code != 307 {
		t.Errorf("We expected a 307 redirect but got: %d", w.Code)
	}

	// Request it again. It should be gone.
	// The link should now be missing in both the links map and the memberships on this keyword.
	if _, exists := aList.Links[aLink.ID]; exists {
		t.Fail()
	}
	if _, exists := core.LinkDataBase.Links[aLink.ID]; exists {
		t.Fail()
	}
}

/*

HTTP handler and API calls

*/

// simple test of the index page
func TestRouteIndex(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "", nil)
	helpHandle := http.HandlerFunc(routeHappyHandler)
	helpHandle.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("Home page didn't return %v", http.StatusOK)
	}
}

/*

utility functions

*/

func TestConfigLoad(t *testing.T) {
	cfg, _ := core.RenderConfig("go2config.json")
	fmt.Println(cfg)
	if cfg.LocalListenAddress == "" {
		t.Error("local listen address config option was unset")
	}
	if cfg.LocalListenPort == 0 {
		t.Error("local listen port config option was unset")
	}
	_, err := core.RenderConfig("nonexistent.file")
	if err == nil {
		t.Error("this config was not supposed to load (not found")
	}
	// TODO: render a config with bad JSON
	_, e := core.RenderConfig("README.md")
	if e == nil {
		t.Error("this config was not supposed to load (malformed")
	}
}

func init() {
	core.ConfigureLogging(true, os.Stdout)
}
