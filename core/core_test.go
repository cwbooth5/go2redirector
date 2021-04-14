package core

import (
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

// func TestGetUsages(t *testing.T) {
// 	l, err := MakeNewlink("localhost", "a title")
// 	if err != nil {
// 		t.Fail()
// 	}
// 	newLinkID, err := LinkDataBase.CommitNewLink(l)
// 	if err != nil {
// 		t.Fail()
// 	}
// 	k, err := MakeNewKeyword("usagestuff")
// 	if err != nil {
// 		t.Fail()
// 	}
// 	ll := MakeNewList(k)
// 	LinkDataBase.Couple(ll, l)

// ll.TagBindings[newLinkID] = []string{"newtag"}

// result1 := ll.GetUsages(newLinkID)
// if result1[0] != "usagestuff/newtag" {
// 	t.Fail()
// }
// fmt.Println(result1)
// }

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

// If this isn't here, logging calls during functions we are testing cause a SEGV
func init() {
	ConfigureLogging(true, os.Stdout)
}
