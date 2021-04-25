package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
)

func TestListenURL(t *testing.T) {
	u := ListenURL()
	if !strings.HasPrefix(u.String(), "http://") {
		t.Errorf("ListenURL didn't start with http:// '%s'", u)
	}
}

func TestIsSpecial(t *testing.T) {
	dotPrefix := Keyword(".keywordstring")
	slashSuffix := Keyword("keywordstring/")
	dotAndSlash := Keyword(".keywordstring/")

	if dotPrefix.IsSpecial() {
		t.Fail()
	}
	if !slashSuffix.IsSpecial() {
		t.Fail()
	}
	if !dotAndSlash.IsSpecial() {
		t.Fail()
	}
}

func TestGpath(t *testing.T) {
	len1, err := ParsePath("keywordstring/")
	if err != nil {
		t.Fail()
	}
	len2, err := ParsePath(".keywordstring/tag1")
	if err != nil {
		t.Fail()
	}
	len3, err := ParsePath("keywordstring/tag1/param1")
	if err != nil {
		t.Fail()
	}

	if len1.Len() != 1 {
		fmt.Println(len1)
		t.Errorf("Gpath len == %d, expected: 1", len1.Len())
	}
	if len2.Len() != 2 {
		t.Fail()
	}
	if len3.Len() != 3 {
		t.Fail()
	}

	// keywords get stripped of . or /
	if len1.Keyword != "keywordstring" {
		t.Errorf("slash suffix was not stripped off keyword")
	}
	if len2.Keyword != "keywordstring" {
		t.Errorf("period prefix was not stripped off keyword")
	}
}

func TestDecouple(t *testing.T) {
	l, err := MakeNewlink("localhost", "a title")
	if err != nil {
		t.Fail()
	}
	k, err := MakeNewKeyword("decoupleme")
	if err != nil {
		t.Fail()
	}
	ll := MakeNewList(k)
	LinkDataBase.Couple(ll, l)
	if _, exists := LinkDataBase.Lists[k]; !exists {
		t.Fail()
	}
	LinkDataBase.Decouple(ll, l)
	// list has no more links, should be destroyed
	if _, exists := LinkDataBase.Lists[k]; exists {
		t.Fail()
	}
	// link has no more memberships, should be destroyed
	if _, exists := LinkDataBase.Links[l.ID]; exists {
		t.Fail()
	}
}

func TestGetLink(t *testing.T) {
	l, err := MakeNewlink("localhost/getlink", "whatever")
	if err != nil {
		t.Fail()
	}
	newLinkID, err := LinkDataBase.CommitNewLink(l)
	if err != nil {
		t.Fail()
	}

	// look for an ID we definitely don't have in the database
	result1 := LinkDataBase.GetLink(8923982, "")
	if result1.ID != 0 {
		t.Fail()
	}
	// -1 is used when we are searching by URL only
	result2 := LinkDataBase.GetLink(-1, "http://localhost/getlink")
	if result2.ID != newLinkID {
		t.Fail()
	}
	// This should result in getting the real link we have in the DB
	result3 := LinkDataBase.GetLink(newLinkID, "")
	if result3.ID != newLinkID {
		t.Fail()
	}
}

func TestTagFunctions(t *testing.T) {
	l, err := MakeNewlink("localhost", "a title")
	if err != nil {
		t.Fail()
	}
	newLinkID, err := LinkDataBase.CommitNewLink(l)
	if err != nil {
		t.Fail()
	}
	k, err := MakeNewKeyword("yayfortags")
	if err != nil {
		t.Fail()
	}
	ll := MakeNewList(k)
	LinkDataBase.Couple(ll, l)

	result1 := ll.GetTag(99998)
	if len(result1) != 0 {
		t.Fail()
	}
	ll.TagBindings[newLinkID] = []string{"newtag"}
	result2 := ll.TagBindings[newLinkID]
	if result2[0] != "newtag" {
		t.Fail()
	}
	ll.TagBindings[newLinkID] = []string{"newtag", "tag2"}
	result3 := ll.GetTagString(newLinkID, "|")
	if result3 != "newtag|tag2" {
		t.Fail()
	}
}

func TestGetUsages(t *testing.T) {
	RedirectorName = "go2" // Set this artificially, no config here
	k, _ := MakeNewKeyword("usagestuff")
	ll := MakeNewList(k)
	l1, _ := MakeNewlink("localhost/kt", "keyword and a tag")
	l2, _ := MakeNewlink("localhost/ktp-{1}.php", "keyword, tag, and parameter")
	keywordTag, _ := LinkDataBase.CommitNewLink(l1)
	keywordTagParameter, _ := LinkDataBase.CommitNewLink(l2)

	LinkDataBase.Couple(ll, l1)
	LinkDataBase.Couple(ll, l2)

	ll.TagBindings[keywordTag] = []string{"kt"}
	ll.TagBindings[keywordTagParameter] = []string{"ktp"}

	// test cases
	result1 := ll.GetUsages(keywordTag)
	if result1[0] != "go2 usagestuff/kt" {
		fmt.Println(result1)
		t.Fail()
	}
	result2 := ll.GetUsages(keywordTagParameter)
	if result2[0] != "go2 usagestuff/ktp/parameter" {
		fmt.Println(result2)
		t.Fail()
	}
}

// This is the function for determining edit distance between two terms.
func TestLevDistance(t *testing.T) {
	a := calcLevDist("", "term")
	if a != 4 {
		t.Fail()
	}
	b := calcLevDist("term", "")
	if b != 4 {
		t.Fail()
	}
	c := calcLevDist("term", "term")
	if c != 0 {
		t.Fail()
	}
	d := calcLevDist("missiles", "missile")
	if d != 1 {
		t.Fail()
	}
	e := calcLevDist("something", "anotherthing")
	if e != 5 {
		t.Fail()
	}
}

func TestMakeNewKeyword(t *testing.T) {
	k, err := MakeNewKeyword("<script>")
	if err == nil {
		t.Errorf("This was a bad keyword and it did not indicate an error: %s", k)
	}
	// url.QueryUnescape should indicate an error and pass it up
	k, err = MakeNewKeyword("invalid%escape")
	if err == nil {
		t.Errorf("This was a bad keyword (cannot escape) and it did not indicate an error: %s", k)
	}

}

func TestMakeNewLink(t *testing.T) {
	l, _ := MakeNewlink(" localhostwithspaces  ", "this is a title")
	if l.ID != 0 {
		t.Fail()
	}
	if l.URL != "http://localhostwithspaces" {
		fmt.Println(l)
		t.Fail()
	}
	l.URL = "http://now-with-substitutions/{1}.php"
	if !l.Special() {
		t.Fail()
	}
}

func TestCommitNewLink(t *testing.T) {
	l, _ := MakeNewlink("localhost", "this is a title")
	if l.ID != 0 {
		t.Fail()
	}
	l.ID = 56
	id, _ := LinkDataBase.CommitNewLink(l)
	if id != 56 {
		t.Fail()
	}
}

func TestSortingInterfaces(t *testing.T) {
	numLinks := 20
	db := MakeNewLinkDatabase()
	for i := 0; i < numLinks; i++ {
		l, _ := MakeNewlink(fmt.Sprintf("example.com/%d", i), fmt.Sprintf("title %d", i))
		db.CommitNewLink(l)
	}

	result1 := db.LinksByAtime(10)
	if result1[0].ID != numLinks {
		t.Fail()
	}

	result1_1 := db.LinksByAtime(999) // test asking for count > found links
	if len(result1_1) != numLinks {
		t.Fail()
	}

	result2 := db.LinksByAtime(2)
	if result2[0].ID != numLinks {
		t.Fail()
	}
	if len(result2) != 2 {
		t.Fail()
	}

	result3 := db.LinksByMtime(10)
	if result3[0].ID != numLinks {
		t.Fail()
	}

	if len(result3) != 10 {
		t.Fail()
	}
	result3_1 := db.LinksByMtime(999)
	if len(result3_1) != numLinks {
		t.Fail()
	}

	// assign a bunch of click counts to the links
	for i, link := range db.Links {
		f := float64(i)
		link.Clicks = int(math.Floor(math.Pow(2, f)))
	}

	// pull the top ten by clicks
	result4 := db.LinksByClicks(10)

	if result4[0].Clicks != 1048576 {
		t.Fail()
	}

	result4_1 := db.LinksByClicks(999)
	if len(result4_1) != numLinks {
		t.Fail()
	}

	// lists can have click counts too
	for i := 0; i < 20; i++ {
		l, _ := MakeNewlink(fmt.Sprintf("example.com/%d", i), fmt.Sprintf("title %d", i))
		db.CommitNewLink(l)
		k, _ := MakeNewKeyword(fmt.Sprintf("mykeyword%d", i))
		ll := MakeNewList(k)
		f := float64(i)
		ll.Clicks = int(math.Floor(math.Pow(2, f)))
		db.Couple(ll, l)
	}

	result5 := db.TopLists(10)
	if result5[0].Clicks != 524288 {
		t.Fail()
	}

	result5_1 := db.TopLists(-1)
	if len(result5_1) != len(db.Lists) {
		t.Fail()
	}
}

func TestModifyLogging(t *testing.T) {
	k, _ := MakeNewKeyword("duh")
	ll := MakeNewList(k)

	if ll.Logging != LinkLogNewKeywords {
		t.Fail()
	}

	// turn off logging
	ll.ModifyLogging(false)
	if ll.Logging == true {
		t.Fail()
	}
	if len(LinkLog[ll.Keyword]) != 0 {
		t.Fail()
	}

	ll.ModifyLogging(true)
	if ll.Logging == false {
		t.Fail()
	}
	if len(LinkLog[ll.Keyword]) != 0 {
		t.Fail()
	}
}

func TestAKA(t *testing.T) {
	l1, _ := MakeNewlink("example.com", "this is a title")
	l2, _ := MakeNewlink("example.com", "this is a title as well")

	empty_aka := l2.AKA()
	if len(empty_aka) != 0 {
		t.Fail()
	}
	// now commit both and check again.
	LinkDataBase.CommitNewLink(l1)
	LinkDataBase.CommitNewLink(l2)
	single_aka := l2.AKA()
	if len(single_aka) != 1 {
		t.Fail()
	}
}

func TestMakeNewList(t *testing.T) {
	k, _ := MakeNewKeyword("mykeyword")
	ll := MakeNewList(k)

	// test for default list behavior
	if ll.Behavior != RedirectToFreshest {
		t.Error("new list creation did not default to behavior: freshest")
	}
	// new lists should have a nil link map
	if len(ll.Links) != 0 {
		fmt.Printf("length: %v", ll.Links)
		t.Fail()
	}
	if ll.Logging != LinkLogNewKeywords {
		t.Fail()
	}
}

func TestGetRedirectURL(t *testing.T) {
	k, _ := MakeNewKeyword("penguins")
	ll := MakeNewList(k)

	// nil case - you have a list with zero links!
	nil_url := ll.GetRedirectURL()
	if nil_url != "" {
		t.Fail()
	}

	// create a bunch of links
	for i := 0; i < 10; i++ {
		l, _ := MakeNewlink(fmt.Sprintf("http://localhost/%d", i), fmt.Sprintf("link %d", i))
		LinkDataBase.CommitNewLink(l)
		LinkDataBase.Couple(ll, l)
	}

	newurl := ll.GetRedirectURL()
	// mtimesort should indicate the freshest is the last one we added.
	if newurl != "http://localhost/9" {
		fmt.Println(newurl)
		t.Error("URL returned was not the freshest!")
	}

	ll.Behavior = RedirectToList
	listurl := ll.GetRedirectURL()
	if listurl != fmt.Sprintf("%s/.penguins", ListenURL()) {
		t.Fail()
	}

	ll.Behavior = RedirectToRandom
	var urls []string
	for i := 1; i < 40; i++ {
		urls = append(urls, ll.GetRedirectURL())
	}

	// It'd be weird if these were all equal.
	if urls[5] == urls[25] && urls[1] == urls[19] && urls[12] == urls[20] {
		t.Fail()
	}

	// case for 'direct to a link in the list'
	l, _ := MakeNewlink("localhost/hellothere", "weeee")
	linkID, _ := LinkDataBase.CommitNewLink(l)
	LinkDataBase.Couple(ll, l)
	ll.Behavior = linkID
	directURL := ll.GetRedirectURL()

	if directURL != "http://localhost/hellothere" {
		t.Fail()
	}

	// 'top' list behavior
	for i, link := range ll.Links {
		f := float64(i)
		link.Clicks = int(math.Floor(math.Pow(2, f)))
	}

	ll.Behavior = RedirectToTop
	topURL := ll.GetRedirectURL()
	fmt.Println(topURL)
	if topURL != "http://localhost/hellothere" {
		t.Fail()
	}
}

func TestCheckTag(t *testing.T) {
	k, _ := MakeNewKeyword("tagcheck")
	ll := MakeNewList(k)

	// Add two links to the list using distinct tags
	l1, _ := MakeNewlink("localhost/tagcheck", "about that hovercraft...")
	linkID1, _ := LinkDataBase.CommitNewLink(l1)
	LinkDataBase.Couple(ll, l1)
	ll.TagBindings[linkID1] = []string{"tag1", "tag2"}
	l2, _ := MakeNewlink("localhost/tagchecker", "oh my eels!")
	linkID2, _ := LinkDataBase.CommitNewLink(l2)
	LinkDataBase.Couple(ll, l2)
	ll.TagBindings[linkID2] = []string{"tag1"} // note we have a dupe now

	// Verify no problems found, tag is unique
	result1 := ll.CheckTag("newtag")
	if result1 != "" {
		t.Fail()
	}

	// Verify existing tag indicates a duplicate was found
	result2 := ll.CheckTag("tag1")
	if result2 == "" { // returned string isn't important to test
		t.Fail()
	}
}

func TestImport(t *testing.T) {
	dbString := strings.NewReader(`{"Lists":{"wiki":{"Keyword":"wiki","Links":{"4":{"ID":4,"URL":"https://en.wikipedia.org/wiki/{1}","Title":"english wikipedia","Lists":["wiki"],"Ctime":"2021-03-02T21:29:53.190845282Z","Mtime":"2021-03-02T21:29:53.190846649Z","Atime":"2021-03-02T21:29:53.190845282Z","Dtime":"2081-07-17T07:12:00Z","LinkVariables":{"1":"burrito"},"Clicks":0},"5":{"ID":5,"URL":"https://it.wikipedia.org","Title":"italian wikipedia","Lists":["wiki"],"Ctime":"2021-04-02T19:25:13.0299362-07:00","Mtime":"2021-04-03T02:25:13.0299745Z","Atime":"2021-04-02T19:25:13.0299362-07:00","Dtime":"2081-07-17T07:12:00Z","LinkVariables":{},"Clicks":0},"6":{"ID":6,"URL":"https://es.wikipedia.org","Title":"spanish wikipedia","Lists":["wiki"],"Ctime":"2021-03-02T21:28:42.998908802Z","Mtime":"2021-03-02T21:28:42.998915106Z","Atime":"2021-03-02T21:28:42.998908802Z","Dtime":"2081-07-17T07:12:00Z","LinkVariables":null,"Clicks":0},"7":{"ID":7,"URL":"https://de.wikipedia.org","Title":"german wikipedia","Lists":["wiki"],"Ctime":"2021-04-02T19:25:23.7522075-07:00","Mtime":"2021-04-03T02:25:23.752248Z","Atime":"2021-04-02T19:25:23.7522075-07:00","Dtime":"2081-07-17T07:12:00Z","LinkVariables":{},"Clicks":0}},"Behavior":-2,"Clicks":7,"Usage":"","Logging":true,"TagBindings":{"4":["en"],"5":["it","italian"],"6":["es"],"7":["de","german"]}}},"Links":{"0":{"ID":1,"URL":"http://127.0.0.1","Title":"This is a default.","Lists":[""],"Ctime":"2021-02-22T07:41:36.914130327Z","Mtime":"2021-02-22T07:41:36.914130327Z","Atime":"2021-02-22T07:41:36.914130327Z","Dtime":"2081-07-17T07:12:00Z","LinkVariables":null,"Clicks":0},"1":{"ID":1,"URL":"http://127.0.0.1","Title":"This is a default.","Lists":[""],"Ctime":"2021-02-22T07:41:36.914130327Z","Mtime":"2021-02-22T07:41:36.914130327Z","Atime":"2021-02-22T07:41:36.914130327Z","Dtime":"2081-07-17T07:12:00Z","LinkVariables":null,"Clicks":0},"4":{"ID":4,"URL":"https://en.wikipedia.org/wiki/{1}","Title":"english wikipedia","Lists":["wiki"],"Ctime":"2021-03-02T21:29:53.190845282Z","Mtime":"2021-03-02T21:29:53.190846649Z","Atime":"2021-03-02T21:29:53.190845282Z","Dtime":"2081-07-17T07:12:00Z","LinkVariables":{"1":"burrito"},"Clicks":0},"5":{"ID":5,"URL":"https://it.wikipedia.org","Title":"italian wikipedia","Lists":["wiki"],"Ctime":"2021-04-02T19:25:13.0299362-07:00","Mtime":"2021-04-03T02:25:13.0299745Z","Atime":"2021-04-02T19:25:13.0299362-07:00","Dtime":"2081-07-17T07:12:00Z","LinkVariables":{},"Clicks":0},"6":{"ID":6,"URL":"https://es.wikipedia.org","Title":"spanish wikipedia","Lists":["wiki"],"Ctime":"2021-03-02T21:28:42.998908802Z","Mtime":"2021-03-02T21:28:42.998915106Z","Atime":"2021-03-02T21:28:42.998908802Z","Dtime":"2081-07-17T07:12:00Z","LinkVariables":null,"Clicks":0},"7":{"ID":7,"URL":"https://de.wikipedia.org","Title":"german wikipedia","Lists":["wiki"],"Ctime":"2021-04-02T19:25:23.7522075-07:00","Mtime":"2021-04-03T02:25:23.752248Z","Atime":"2021-04-02T19:25:23.7522075-07:00","Dtime":"2081-07-17T07:12:00Z","LinkVariables":{},"Clicks":0}},"NextLinkID":8}`)
	db := MakeNewLinkDatabase()
	LinkDataBase = db

	err := db.Import(dbString)
	if err != nil {
		t.FailNow() // this should have worked
	}
	if len(LinkDataBase.Lists) != 1 {
		fmt.Println(LinkDataBase)
		t.FailNow()
	}
	if len(LinkDataBase.Links) != 6 {
		fmt.Println(LinkDataBase)
		t.FailNow()
	}
}

func TestExport(t *testing.T) {
	dbString := strings.NewReader(`{Lists":{"wiki":{"Keyword":"wiki","Links":{"4":{"ID":4,"URL":"https://en.wikipedia.org/wiki/{1}","Title":"english wikipedia","Lists":["wiki"],"Ctime":"2021-03-02T21:29:53.190845282Z","Mtime":"2021-03-02T21:29:53.190846649Z","Atime":"2021-03-02T21:29:53.190845282Z","Dtime":"2081-07-17T07:12:00Z","LinkVariables":{"1":"burrito"},"Clicks":0},"5":{"ID":5,"URL":"https://it.wikipedia.org","Title":"italian wikipedia","Lists":["wiki"],"Ctime":"2021-04-02T19:25:13.0299362-07:00","Mtime":"2021-04-03T02:25:13.0299745Z","Atime":"2021-04-02T19:25:13.0299362-07:00","Dtime":"2081-07-17T07:12:00Z","LinkVariables":{},"Clicks":0},"6":{"ID":6,"URL":"https://es.wikipedia.org","Title":"spanish wikipedia","Lists":["wiki"],"Ctime":"2021-03-02T21:28:42.998908802Z","Mtime":"2021-03-02T21:28:42.998915106Z","Atime":"2021-03-02T21:28:42.998908802Z","Dtime":"2081-07-17T07:12:00Z","LinkVariables":null,"Clicks":0},"7":{"ID":7,"URL":"https://de.wikipedia.org","Title":"german wikipedia","Lists":["wiki"],"Ctime":"2021-04-02T19:25:23.7522075-07:00","Mtime":"2021-04-03T02:25:23.752248Z","Atime":"2021-04-02T19:25:23.7522075-07:00","Dtime":"2081-07-17T07:12:00Z","LinkVariables":{},"Clicks":0}},"Behavior":-2,"Clicks":7,"Usage":"","Logging":true,"TagBindings":{"4":["en"],"5":["it","italian"],"6":["es"],"7":["de","german"]}}},"Links":{"0":{"ID":1,"URL":"http://127.0.0.1","Title":"This is a default.","Lists":[""],"Ctime":"2021-02-22T07:41:36.914130327Z","Mtime":"2021-02-22T07:41:36.914130327Z","Atime":"2021-02-22T07:41:36.914130327Z","Dtime":"2081-07-17T07:12:00Z","LinkVariables":null,"Clicks":0},"1":{"ID":1,"URL":"http://127.0.0.1","Title":"This is a default.","Lists":[""],"Ctime":"2021-02-22T07:41:36.914130327Z","Mtime":"2021-02-22T07:41:36.914130327Z","Atime":"2021-02-22T07:41:36.914130327Z","Dtime":"2081-07-17T07:12:00Z","LinkVariables":null,"Clicks":0},"4":{"ID":4,"URL":"https://en.wikipedia.org/wiki/{1}","Title":"english wikipedia","Lists":["wiki"],"Ctime":"2021-03-02T21:29:53.190845282Z","Mtime":"2021-03-02T21:29:53.190846649Z","Atime":"2021-03-02T21:29:53.190845282Z","Dtime":"2081-07-17T07:12:00Z","LinkVariables":{"1":"burrito"},"Clicks":0},"5":{"ID":5,"URL":"https://it.wikipedia.org","Title":"italian wikipedia","Lists":["wiki"],"Ctime":"2021-04-02T19:25:13.0299362-07:00","Mtime":"2021-04-03T02:25:13.0299745Z","Atime":"2021-04-02T19:25:13.0299362-07:00","Dtime":"2081-07-17T07:12:00Z","LinkVariables":{},"Clicks":0},"6":{"ID":6,"URL":"https://es.wikipedia.org","Title":"spanish wikipedia","Lists":["wiki"],"Ctime":"2021-03-02T21:28:42.998908802Z","Mtime":"2021-03-02T21:28:42.998915106Z","Atime":"2021-03-02T21:28:42.998908802Z","Dtime":"2081-07-17T07:12:00Z","LinkVariables":null,"Clicks":0},"7":{"ID":7,"URL":"https://de.wikipedia.org","Title":"german wikipedia","Lists":["wiki"],"Ctime":"2021-04-02T19:25:23.7522075-07:00","Mtime":"2021-04-03T02:25:23.752248Z","Atime":"2021-04-02T19:25:23.7522075-07:00","Dtime":"2081-07-17T07:12:00Z","LinkVariables":{},"Clicks":0}},"NextLinkID":8}`)
	db := MakeNewLinkDatabase()
	LinkDataBase = db
	db.Import(dbString)
	buf := new(bytes.Buffer)

	err := LinkDataBase.Export(buf)

	fmt.Printf("Buffer: %v", buf)
	if err != nil {
		t.FailNow()
	}
	_, err = json.Marshal(buf)
	if err != nil {
		t.Fail()
	}
}

// If this isn't here, logging calls during functions we are testing cause a SEGV
func init() {
	ConfigureLogging(true, os.Stdout)
}
