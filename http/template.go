package http

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/cwbooth5/go2redirector/core"
	"github.com/oxtoacart/bpool"
)

// Many functions have to get to the templates stack, so it is made global.
var Templates = make(map[string]*template.Template)

var Bufpool *bpool.BufferPool

type ModelIndex struct {
	Title              string
	LinkDB             core.LinkDatabase
	Keyword            core.Keyword
	KeywordExists      bool
	KeywordBeingEdited bool
	LinkExists         bool
	LinkBeingEdited    *core.Link
	RedirectorName     string
	Overrides          map[string]string
	KeywordParams      []string
	UsageLog           []string
	ErrorMessage       string
	ActiveUser         string // empty string means not logged in
}

// GetBehavior returns a string representation of the behavior for a model's keyword.
// Strings are returned because they are being used in HTML by the template.
func (m *ModelIndex) GetBehavior() string {
	// If the keyword does not exist, this is the default.
	if !m.KeywordExists {
		return strconv.Itoa(core.RedirectToFreshest)
	}
	// TODO: read mutex
	behavior := core.LinkDataBase.Lists[m.Keyword].Behavior
	res := strconv.Itoa(behavior)
	return res
}

func (m *ModelIndex) PrettyBehavior() string {
	switch m.GetBehavior() {
	case "-1":
		return "this page"
	case "-2":
		return "freshest link"
	case "-3":
		return "most used link"
	case "-4":
		return "random link"
	default:
		// The list redirects to a specific link. Get its title.
		return m.LinkDB.GetLink(core.LinkDataBase.Lists[m.Keyword].Behavior, "").Title
	}
}

func (m *ModelIndex) ListURL() string {
	var result string
	if ll, exists := core.LinkDataBase.Lists[m.Keyword]; exists {
		result = ll.GetRedirectURL()
	}
	return result
}

func (m *ModelIndex) ClickSort() []*core.ListOfLinks {
	target := core.LinkDataBase.Lists
	temp := []*core.ListOfLinks{}
	for _, v := range target {
		temp = append(temp, v)
	}
	sort.Sort(core.ByClicks(temp))
	return temp
}

// //sort on the most recent atime, descending
func (m *ModelIndex) AtimeSort(k core.Keyword) []*core.Link {
	target := core.LinkDataBase.Links
	temp := []*core.Link{}
	for _, v := range target {
		temp = append(temp, v)
	}
	sort.Sort(core.ByAtime(temp))
	return temp
}

// //sort on the most recent mtime, descending
func (m *ModelIndex) MtimeSort(k core.Keyword) []*core.Link {
	target := core.LinkDataBase.Lists[k].Links
	temp := []*core.Link{}
	for _, v := range target {
		temp = append(temp, v)
	}
	sort.Sort(core.ByMtime(temp))
	return temp
}

// The template can use this to get a nicer string explaining mtime.
func (m *ModelIndex) PrettyTime(t time.Time) string {
	// Dates in the future
	if t == core.Never {
		return "never"
	}

	// This is used in the template to print expiration times in the future.
	now := time.Now()
	if t.After(now) {
		sub := t.Sub(now)
		return fmt.Sprintf("%s", sub.Truncate(time.Second))
	}

	// Dates in the past - This uses 30 days/720 hours for a month
	delta := time.Now().Sub(t)
	if delta.Minutes() < 1 {
		return fmt.Sprintf("%d seconds ago", int(delta.Seconds()))
	} else if delta.Minutes() < 2 {
		return fmt.Sprintf("%d minute ago", int(delta.Minutes()))
	} else if delta.Hours() < 1 {
		return fmt.Sprintf("%d minutes ago", int(delta.Minutes()))
	} else if delta.Hours() < 24 {
		return "today"
	} else if delta.Hours() < 48 {
		return "yesterday"
	} else if delta.Hours() < 720 {
		return fmt.Sprintf("%d days ago", int(delta.Hours()/24))
	} else {
		if t == core.BurnTime {
			return "This link will be burned after one reading."
		}
		if int(delta.Hours()/720) == 1 {
			return fmt.Sprintf("%d month ago", int(delta.Hours()/720))
		}
		return fmt.Sprintf("%d months ago", int(delta.Hours()/720))
	}
}

// return the usage strings for special keywords/lists
func (m *ModelIndex) GetUsage() string {
	// new keywords: params they provided become the usage statement
	if !m.KeywordExists {
		if m.KeywordParams == nil || m.KeywordParams[0] == "" {
			return "" //TODO, for now...this is where we'd get existing usage
		}
		return fmt.Sprintf("{%s}", strings.Join(m.KeywordParams, "}/{"))
	}
	// existing keywords: we read the values out of the ll
	return core.LinkDataBase.Lists[m.Keyword].Usage
}

// return true if a keyword is special/dynamic, false if it is a regular keyword
func (m *ModelIndex) IsSpecial() bool {
	// If the keyword exists and is within the model, trailing slash means it's special.
	return m.Keyword.IsSpecial()
}

// IdentifyBuild simply returns the sha256sum of the redirector binary for simple versioning.
func (m *ModelIndex) IdentifyBuild() string {
	f, err := os.Open("main.go")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	hsh := sha256.New()
	if _, err := io.Copy(hsh, f); err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf("%x", hsh.Sum(nil))
}

// give keyword, get list of links
func (m *ModelIndex) GetMyList(k core.Keyword) *core.ListOfLinks {
	return core.LinkDataBase.Lists[k]
}

// Return the configuration external_address value for use in templates.
// This does not contain the http or https prefixes, which would be nice to have
// since the external full URL could potentially use either protocol.
func (m *ModelIndex) GetExternalRedirectorAddress() string {
	return core.ExternalAddress
}

func (m *ModelIndex) GetListEdits(k core.Keyword) []*core.EditRecord {
	return core.RedirectorMetadata.ListEdits[k]
}

func (m *ModelIndex) GetLinkEdits(id int) []*core.EditRecord {
	return core.RedirectorMetadata.LinkEdits[id]
}

// GetSimilar locates similarly-named keywords from an existing list of links.
func (m *ModelIndex) GetSimilar() []core.Keyword {
	targets := []core.Keyword{}
	s1 := string(m.Keyword)
	allLists := m.LinkDB.Lists
	// allLists := core.LinkDataBase.Lists
	for _, val := range allLists {
		s2 := string(val.Keyword)
		if s1 == s2 {
			continue
		}
		ratio := core.Similar(s1, s2)
		if ratio < core.LevDistRatio && s1 != s2 {
			// low ratio
			targets = append(targets, val.Keyword)
		} else if strings.Contains(s1, s2) || strings.Contains(s2, s1) {
			// substring match
			targets = append(targets, val.Keyword)
		}
	}
	return targets
}
