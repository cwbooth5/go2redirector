package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"
)

/*
	Primitives
*/

// used for Mtime, date set ridiculously far in the future
var Never = time.Date(2081, 7, 17, 7, 12, 0, 0, time.UTC)

// used to indicate links to be 'burned after reading', date set well in the past
var BurnTime = time.Date(1, 1, 1, 1, 1, 1, 1, time.UTC)

// LinkLog structure for special link usages on all special keywords
// This grows to the size of len(keywords) in the link database.
// Each array of strings is a list of most recent to oldest usages of that keyword.
// This is thrown away when the redirector shuts down.
var LinkLog = make(map[Keyword][]string)

var LinkZero = newEmptyLink(LinkDataBase, "127.0.0.1", "This is link zero.", "link zero!")

// Keyword is a string representing a list name.
type Keyword string

// custom internal url type
type InternalURL string

// This is an enum for different list of links redirect behaviors.
// Named cases get negative integers.
// Zero is going to be "unset" (should never be seen)
// One or greater will be a link ID.
const (
	RedirectToList     = -1
	RedirectToFreshest = -2 // the default when new lists are created
	RedirectToTop      = -3
	RedirectToRandom   = -4
	// specific links are going to use their >0 link IDs
)

// Links are identified by their global LinkID.
// The URL is a string because it might have substitutions within, not being a valid URL while stored here.
// Ctime == created, Mtime == modified, Atime == last time clicked/redirected
type Link struct {
	ID    int // This is the one value users can never change.
	URL   string
	Title string
	// Lists         map[Keyword]*ListOfLinks // old: Lists []Keyword
	Lists []Keyword
	Ctime time.Time
	Mtime time.Time
	Atime time.Time
	Dtime time.Time
	//Special       bool
	LinkVariables map[string]string
	Clicks        int
}

// ListOfLinks most notably contains a map of [int]*link referring to all links coupled
// with this list.
// Bindings keep track of link ID-to-code word mappings. A link in a given list of links can have a different
// code, because each link can be thought of like its own context
type ListOfLinks struct {
	Keyword Keyword
	// Links    []*link
	Links       map[int]*Link
	Behavior    int // negative IDs are special cases
	Clicks      int
	Usage       string
	Logging     bool
	TagBindings map[int]string
}

type LinkDatabase struct {
	Lists      map[Keyword]*ListOfLinks
	Links      map[int]*Link
	NextLinkID int
}

// Gpath holds a Keyword, a Tag, and an array of any Params supplied by the user.
type Gpath struct {
	Keyword Keyword
	Tag     string
	Params  []string
}

/*
	Operations
*/

func (k Keyword) IsSpecial() bool {
	return strings.HasSuffix(string(k), "/")
}

// How many {variable} blocks are in the URL string?
func (i InternalURL) VarCount() int {
	return 0 // TODO, implement
}

// Is the URL fully formed? Can it be rendered as it currently stands in a browser?
func (i InternalURL) Valid() bool {
	return true
}

// Len of the Gpath struct is the total count of path items.
func (g Gpath) Len() int {
	i := 1 //keyword always counts as 1
	if g.Tag != "" {
		i++
	}
	return i + len(g.Params)
}

// MakeNewKeyword is constructor and sanitizer for keywords so they can be used safely in lookups.
// valid characters are ALPHA, DIGIT, or any of "-_ ~" (note that space is in there)
// All keywords successfully created will be lower-cased because case-sensitive would drive people NUTS.
func MakeNewKeyword(kwd string) (Keyword, error) {
	// In order to allow spaces, unescape the keywords with spaces because they will be
	// coming in from the entered URL using %20 to encode the spaces.
	escaped, err := url.QueryUnescape(kwd)
	if err != nil {
		LogError.Printf("Keyword could not be unescaped. '%s'\n", kwd)
	}

	// first strip known leading and trailing characters users are known to add
	// first check for valid characters for a URL string
	escaped = strings.TrimPrefix(escaped, ".")
	escaped = strings.TrimPrefix(escaped, "/")
	escaped = strings.TrimPrefix(escaped, ".")
	escaped = strings.TrimSuffix(escaped, "/")

	// then check for valid characters for a URL string
	for idx, l := range escaped {
		if unicode.IsLetter(l) || unicode.IsDigit(l) || strings.ContainsAny(string(l), "-_ ~") {
			continue
		} else {
			msg := "valid characters are ALPHA, DIGIT, or any of -_ ~"
			LogDebug.Printf("Bad character at position %d: '%s'\n", idx, string(l))
			LogDebug.Printf(msg)
			return Keyword(""), errors.New(msg)
		}
	}

	k := Keyword(strings.ToLower(escaped))
	return k, err
}

func MakeNewlink(incomingURL string, title string) (*Link, error) {
	var err error
	// what could err be here? What could be wrong with their input URL and title
	// normalize the URL here in prep for comparison
	listMembership := []Keyword{} // no memberships initially, done when adding to lists, array of keywords to prevent cycles
	createTime := time.Now().UTC()
	newLink := Link{
		ID:    0,
		URL:   SanitizeURL(incomingURL),
		Title: title,
		Lists: listMembership,
		Ctime: createTime,
		Mtime: createTime,
		Atime: createTime,
		Dtime: Never,
	}
	return &newLink, err
}

// Combine a keyword and a link pointer to generate a new listofLinks
func MakeNewList(keyword Keyword, linkobj *Link) *ListOfLinks {
	var behavior int
	switch NewListBehavior {
	case "RedirectToFreshest":
		behavior = RedirectToFreshest
	case "RedirectToTop":
		behavior = RedirectToTop
	case "RedirectToRandom":
		behavior = RedirectToRandom
	case "RedirectToList":
		behavior = RedirectToList
	default:
		behavior = RedirectToFreshest
	}
	return &ListOfLinks{
		Keyword:     keyword,
		Links:       make(map[int]*Link),
		Behavior:    behavior,
		Usage:       "",
		Logging:     LinkLogNewKeywords,
		TagBindings: make(map[int]string),
	}
}

// If someone adds a link with an identical URL to an existing link, provide a way to
// show other links and how people titled them.
func (l Link) AKA() []*Link {
	// TODO: #34 linear, could be improved
	found := []*Link{}
	for _, lnk := range LinkDataBase.Links {
		if lnk.URL == l.URL && lnk.ID != l.ID {
			found = append(found, lnk)
		}
	}
	return found
}

func (l Link) Special() bool {
	return strings.ContainsAny(l.URL, "{}")
}

// GetRedirectURL will return a URL string for given keyword based on its current behavior.
func (ll *ListOfLinks) GetRedirectURL() string {
	/*
		freshest == most recent mtime
		top == most clicks
		random == throw a dart
		list == list page for the keyword
		default == direct to a specific link, based on current LinkID set (Behavior > 0)
	*/

	// copy of the array of links used for iteration in the below cases
	temp := []*Link{}
	for _, v := range ll.Links {
		temp = append(temp, v)
	}

	switch ll.Behavior {
	case RedirectToFreshest:
		sort.Sort(ByMtime(temp))
		return fmt.Sprintf("%s", temp[0].URL)
	case RedirectToTop:
		// Locate the link with the most clicks
		// TODO: the sort interface available is on arrays of *listoflinks.
		// That won't work here. We need to first get a click count on each link itself.
		// Then we need to sort by that click count to find the link with the highest number.
		return TopLink(*ll).URL
	case RedirectToRandom:
		// Just pick a random link under this list of links.
		temp := []*Link{}
		for _, v := range ll.Links {
			temp = append(temp, v)
		}
		randURL := temp[rand.Intn(len(temp))]
		return fmt.Sprintf("%s", randURL.URL)
	case RedirectToList:
		return fmt.Sprintf("%s/.%s", ListenURL(), ll.Keyword)
	default:
		// If the behavior int is above 0, it's a link ID.
		linkFromId := LinkDataBase.GetLink(ll.Behavior, "")
		return linkFromId.URL
	}
}

// Set link logging to on(true) or off(false) for the provided keyword.
// true == empty linklog is created for the list
// false == linklog entry for the keyword is deleted, can be used to clear out history
// The reason this exists is the user has the right to be forgotten. They should be able to
// both record and delete recordings of usages of keywords.
func (ll *ListOfLinks) ModifyLogging(setting bool) {
	ll.Logging = setting
	if setting == true {
		var a []string
		LinkLog[ll.Keyword] = a // empty slice initially
	} else {
		delete(LinkLog, ll.Keyword)
		LogDebug.Printf("Linklog for '%s' destroyed due to user request to disable logging\n", ll.Keyword)
	}
}

// Return a tag string for a given link ID in this list of links.
func (ll *ListOfLinks) GetTag(i int) string {
	return ll.TagBindings[i]
}

func (ll *ListOfLinks) ClickSort() []*Link {
	sorted := []*Link{}
	for _, lnk := range ll.Links {
		sorted = append(sorted, lnk)
	}
	sort.Sort(ByLinkClicks(sorted))

	return sorted
}

// GetSimilar locates similarly-named keywords from an existing list of links.
func (ll *ListOfLinks) GetSimilar(kwd Keyword) []Keyword {
	targets := []Keyword{}
	s1 := string(kwd)
	allLists := LinkDataBase.Lists // TODO: rmutex
	for _, val := range allLists {
		s2 := string(val.Keyword)
		if s1 == s2 {
			continue
		}
		ratio := Similar(s1, s2)
		if ratio < LevDistRatio && s1 != s2 {
			// low ratio
			targets = append(targets, val.Keyword)
		} else if strings.Contains(s1, s2) || strings.Contains(s2, s1) {
			// substring match
			targets = append(targets, val.Keyword)
		}
	}
	return targets
}

/*
	Core linkdatabase operations
*/

var LinkDataBase = MakeNewLinkDatabase()

// NewLinkDatabase is an exported constructor for making that first links db
func MakeNewLinkDatabase() *LinkDatabase {
	return &LinkDatabase{
		Lists:      make(map[Keyword]*ListOfLinks),
		Links:      make(map[int]*Link),
		NextLinkID: 1,
	}
}

func (d *LinkDatabase) Import(filename string) {
	var tempdb LinkDatabase
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		LogDebug.Fatal("DB file was not found! Run the install script to create one.")
	}

	err = json.Unmarshal(data, &tempdb)
	if err != nil {
		LogError.Printf("json parsing error: %s", err)
	}

	LinkDataBase = &tempdb
}

// export the entire DB into a JSON file on disk.
func (d *LinkDatabase) Export(f string) error {
	file, err := json.Marshal(*d)
	if err != nil {
		LogError.Println("JSON marshal error:", err)
		return err
	}
	err = ioutil.WriteFile(f, file, 0644)
	if err != nil {
		LogError.Fatal(err)
	}
	// LogInfo.Printf("Link DB exported to %s.\n", f)
	return err
}

// convenience function used to create an empty link for the DB at ID == 0.
func newEmptyLink(d *LinkDatabase, incomingURL string, title string, keyword Keyword) *Link {
	createTime := time.Now().UTC()
	newLink := Link{
		ID:    0,
		URL:   incomingURL,
		Title: title,
		Lists: make([]Keyword, 1),
		Ctime: createTime,
		Mtime: createTime,
		Atime: createTime,
		Dtime: Never,
	}
	// create the link in the DB at ID == 0, which is a unique link object.
	d.Links[0] = &newLink
	return &newLink
}

// Decouple a list of links and a specific link.
// If the removal of a link from the list results in a zero-length list, the list is deleted.
func (d *LinkDatabase) Decouple(ll *ListOfLinks, linkObj *Link) {
	LogInfo.Printf("Link ID %d has been decoupled from keyword '%s'\n", linkObj.ID, ll.Keyword)
	delete(ll.Links, linkObj.ID)

	// If the length of the list now is zero, the list should be removed entirely.
	if len(ll.Links) == 0 {
		delete(d.Lists, ll.Keyword)
		// remove the usage log for this keyword
		//delete(LinkLog, ll.Keyword)  TODO, turn this back on
	}

	// Remove the list's keyword from the link's memberships.
	updatedMemberships := []Keyword{}
	for _, kwd := range linkObj.Lists {
		if ll.Keyword != kwd {
			updatedMemberships = append(updatedMemberships, kwd) // keep this one
		}
	}
	// mtime update, new memberships with keyword removed
	linkObj.Mtime = time.Now().UTC()
	linkObj.Lists = updatedMemberships

	// If the link holds no memberships, it is deleted from the database.
	if len(linkObj.Lists) == 0 {
		LogInfo.Printf("Link %d has been removed (no remaining list memberships)", linkObj.ID)
		delete(d.Links, linkObj.ID)
	}
}

// Couple a an existing link's pointer to a list of links. The list can be existing or will be committed here if new.
// When you combine a list and a link, the list gets this link included and the link gets its memberships updated.
func (d *LinkDatabase) Couple(ll *ListOfLinks, linkObj *Link) {
	LogInfo.Printf("Link ID %d has been coupled with keyword '%s'\n", linkObj.ID, ll.Keyword)
	linkObj.Mtime = time.Now().UTC()

	if _, exists := d.Lists[ll.Keyword]; !exists {
		d.Lists[ll.Keyword] = ll // create the list of links
	}

	// Update memberships in both the list and the link.
	present := false
	for _, kwd := range linkObj.Lists {
		if kwd == ll.Keyword {
			present = true
		}
	}
	if !present { // do not add membership if it is already present
		linkObj.Lists = append(linkObj.Lists, ll.Keyword)
	}

	ll.Links[linkObj.ID] = linkObj
}

// CommitNewLink adds a Link object to the database.
// We only advance the linkid counter if we are actually about to commit a link.
func (d *LinkDatabase) CommitNewLink(l *Link) (int, error) {
	var err error
	var id int
	if l.ID == 0 {
		id = d.NextLinkID
		l.ID = id
		d.Links[id] = l
		d.NextLinkID++
	} else {
		msg := "The link being added was not ID=0/new!"
		LogError.Println(msg)
		err = errors.New(msg)
		id = l.ID
	}
	LogDebug.Printf("New link with ID %d was added.\n", l.ID)
	return id, err
}

/* Get a link object by ID or URL. */
func (d *LinkDatabase) GetLink(id int, url string) *Link {

	for _, lnk := range d.Links {
		if lnk.ID == id || lnk.URL == url {
			return lnk
		}
	}
	// special case: used to create the initial LinkZero.
	// secondary case: they asked for an ID or URL which we could not find.
	return LinkZero
}

// Sort by access time (Atime)
func (d *LinkDatabase) LinksByAtime(count int) []*Link {
	linkPile := []*Link{}
	for _, link := range d.Links {
		linkPile = append(linkPile, link)
	}
	sort.Sort(ByAtime(linkPile))
	return linkPile[:count]
}

// Sort by modification time (Mtime)
func (d *LinkDatabase) LinksByMtime(count int) []*Link {
	linkPile := []*Link{}
	for _, link := range d.Links {
		linkPile = append(linkPile, link)
	}
	sort.Sort(ByMtime(linkPile))
	return linkPile[:count]
}

func (d *LinkDatabase) TopLists(count int) []*ListOfLinks {
	listPile := []*ListOfLinks{}
	for _, list := range d.Lists {
		listPile = append(listPile, list)
	}
	sort.Sort(ByClicks(listPile))
	if count < 0 {
		return listPile
	}
	return listPile[:count]
}
