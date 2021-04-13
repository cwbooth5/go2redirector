package core

import (
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

// If this isn't here, logging calls during functions we are testing cause a SEGV
func init() {
	ConfigureLogging(true, os.Stdout)
}
