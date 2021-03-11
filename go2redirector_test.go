package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
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
	aList := core.MakeNewList(akw, aLink)
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
	aList := core.MakeNewList(akw, aLink)
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
	aList := core.MakeNewList(akw, aLink)
	db.Couple(aList, aLink)
	aList.Behavior = core.RedirectToRandom

	if aList.Behavior != -4 {
		core.PrintList(*aList)
		t.Fail()
	}

	// test behavior change to specific link
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
	aList := core.MakeNewList(aKw, aLink)
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

// func TestGetSimilar(t *testing.T) {
// 	db := core.MakeNewLinkDatabase()
// 	aLink, _ := core.MakeNewlink("www.example.com/TestGetSimilar", "probably TestGetSimilar")
// 	db.CommitNewLink(aLink)
// 	akw, _ := core.MakeNewKeyword("site")
// 	aList := core.MakeNewList(akw, aLink)
// 	db.Couple(aList, aLink)

// 	firstSim := aList.GetSimilar(akw)
// 	if len(firstSim) != 0 {
// 		t.Log("there shouldn't be any similar links since this is the only link")
// 		t.Fail()
// 	}

// 	// make another similar keyword with an edit distance of 1. This should be tagged as similar.
// 	bkw, _ := core.MakeNewKeyword("sites")
// 	bList := core.MakeNewList(bkw, aLink)
// 	db.Couple(bList, aLink)
// 	secondSim := bList.GetSimilar(bkw)
// 	if len(secondSim) != 1 {
// 		t.Logf("expected 1 similar link, got: %d", len(secondSim))
// 		for i, v := range secondSim {
// 			fmt.Printf("num: %d, value: %s\n", i, v)
// 		}
// 		t.Fail()
// 	}
// 	fmt.Println(db)
// }

// This is the function for determining edit distance between two terms.
// func TestLevDistance(t *testing.T) {
// 	a := core.calcLevDist("", "term")
// 	if a != 4 {
// 		t.Fail()
// 	}
// 	b := calcLevDist("term", "")
// 	if b != 4 {
// 		t.Fail()
// 	}
// 	c := calcLevDist("term", "term")
// 	if c != 0 {
// 		t.Fail()
// 	}
// 	d := calcLevDist("missiles", "missile")
// 	if d != 1 {
// 		t.Fail()
// 	}
// 	e := calcLevDist("something", "anotherthing")
// 	if e != 5 {
// 		t.Fail()
// 	}

// }

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
	_, e := core.RenderConfig("go2redirector.go")
	if e == nil {
		t.Error("this config was not supposed to load (malformed")
	}
}

func init() {
	core.ConfigureLogging(true, os.Stdout)
}
