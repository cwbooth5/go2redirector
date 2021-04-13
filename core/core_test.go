package core

import (
	"fmt"
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
}

// If this isn't here, logging calls during functions we are testing cause a SEGV
func init() {
	ConfigureLogging(true, os.Stdout)
}
