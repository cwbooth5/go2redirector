package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cwbooth5/go2redirector/core"
)

/*
disbled until checkpoint can be run within testing timeframe

	func TestGetSimilar(t *testing.T) {
		db := core.MakeNewLinkDatabase()
		aLink, _ := core.MakeNewlink("www.example.com/TestGetSimilar", "probably TestGetSimilar")
		db.CommitNewLink(aLink)
		akw, _ := core.MakeNewKeyword("site")
		aList := core.MakeNewList(akw)
		db.Couple(aList, aLink)

		// create a model
		model := ModelIndex{Keyword: akw, LinkDB: *db}

		firstSim := model.GetSimilar()
		if len(firstSim) != 0 {
			t.Log("there shouldn't be any similar links since this is the only link")
			t.Fail()
		}

		// make another similar keyword with an edit distance of 1. This should be tagged as similar.
		bLink, _ := core.MakeNewlink("www.example.com/TestGetSimilar/2.php", "moar TestGetSimilar")
		db.CommitNewLink(aLink)
		bkw, _ := core.MakeNewKeyword("sites")
		bList := core.MakeNewList(bkw)
		db.Couple(bList, bLink)
		// populate the search-related data structures (done in goroutine normally)
		core.IndexSearchDB("1s")
		time.Sleep(2)
		secondSim := model.GetSimilar()
		fmt.Printf("edit data: %s", secondSim)
		if len(secondSim) != 1 {
			t.Logf("expected 1 similar link, got: %d", len(secondSim))
			for i, v := range secondSim {
				fmt.Printf("num: %d, value: %s\n", i, v)
			}
			t.Fail()
		}
		fmt.Println(db)
	}
*/
func TestGetExternalRedirectorAddress(t *testing.T) {
	model := ModelIndex{Keyword: core.Keyword("test")}
	if model.GetExternalRedirectorAddress() != core.ExternalAddress {
		t.Fail()
	}
}

func TestGetExternalRedirectorProto(t *testing.T) {
	model := ModelIndex{Keyword: core.Keyword("test")}
	if model.GetExternalRedirectorProto() != core.ExternalProto {
		t.Fail()
	}
}

func TestRouteLogin(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("POST", "/", nil)
	cookie := http.Cookie{
		Name:    "redirectorlogin",
		Value:   "trogdor",
		Expires: time.Unix(0, 0), // expiry not relevant in this test
	}
	http.SetCookie(w, &cookie)

	h := http.HandlerFunc(RouteLogin)
	h.ServeHTTP(w, r)
}

// Fuzz the login cookies (the field name and value)
func FuzzRouteLogin(f *testing.F) {
	f.Add("000", "somename")
	f.Add("redirectorlogin", "batman") // the happy path
	f.Add("]][[|/@#%&@#%1a]]", "glorp boo")
	f.Fuzz(func(t *testing.T, field string, inputname string) {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("POST", "/", nil)
		cookie := http.Cookie{
			Name:    field,
			Value:   inputname,
			Expires: time.Unix(0, 0), // expiry not relevant in this test
		}
		http.SetCookie(w, &cookie)

		h := http.HandlerFunc(RouteLogin)
		h.ServeHTTP(w, r)
	})
}

// If this isn't here, logging calls during functions we are testing cause a SEGV
func init() {
	core.ConfigureLogging(true, os.Stdout)
}
