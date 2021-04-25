package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	ID            int // This is the one value users can never change.
	URL           string
	Title         string
	Lists         []Keyword
	Ctime         time.Time
	Mtime         time.Time
	Atime         time.Time
	Dtime         time.Time
	LinkVariables map[string]string
	Clicks        int
}

// ListOfLinks most notably contains a map of [int]*link referring to all links coupled
// with this list.
// Bindings keep track of link ID-to-code word mappings. A link in a given list of links can have a different
// code, because each link can be thought of like its own context
type ListOfLinks struct {
	Keyword     Keyword
	Links       map[int]*Link
	Behavior    int // negative IDs are special cases
	Clicks      int
	Usage       string
	Logging     bool
	TagBindings map[int][]string
}

type LinkDatabase struct {
	Lists      map[Keyword]*ListOfLinks
	Links      map[int]*Link
	NextLinkID int
}

// Gpath holds a Keyword, a Tag, and an array of any Params supplied by the user.
// This is used to structure their input to the redirector (from their browser URL bar).
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
// func MakeNewList(keyword Keyword, linkobj *Link) *ListOfLinks {
func MakeNewList(keyword Keyword) *ListOfLinks {
	defaultBehavior := RedirectToFreshest
	return &ListOfLinks{
		Keyword:     keyword,
		Links:       make(map[int]*Link),
		Behavior:    defaultBehavior,
		Usage:       "",
		Logging:     LinkLogNewKeywords,
		TagBindings: make(map[int][]string),
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

	// nil case - there's nothing in this list yet
	if len(ll.Links) == 0 {
		return ""
	}

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

// Return a tag []string for a given link ID in this list of links.
func (ll *ListOfLinks) GetTag(i int) []string {
	return ll.TagBindings[i]
}

func (ll *ListOfLinks) GetTagString(i int, delimiter string) string {
	return strings.Join(ll.TagBindings[i], delimiter)
}

// ClickSort will sort a list of links by each link's click count. It will not return
// a modified list, but an array of *Link pointers. This is used mostly for list
// display purposes.
func (ll *ListOfLinks) ClickSort() []*Link {
	sorted := []*Link{}
	for _, lnk := range ll.Links {
		sorted = append(sorted, lnk)
	}
	sort.Sort(ByLinkClicks(sorted))

	return sorted
}

// Check a tag on a list of links and return a string describing any problems (if any).
// Currently, the only problem users can create is a duplicate tag in a list.
func (ll *ListOfLinks) CheckTag(inputTag string) string {
	// look through the tag bindings. Is this tag already present?
	dupes := make(map[string]bool)
	for _, taglist := range ll.TagBindings {
		for _, tag := range taglist {
			if _, exists := dupes[tag]; !exists {
				dupes[tag] = false
			} else {
				dupes[tag] = true
			}
		}
	}
	if dupes[inputTag] == true {
		return "Duplicate tag! This could lead to undefined list behavior."
	} else {
		return "" // no problems found
	}
}

// GetUsages is run in the templates to provide all possible usage strings
// for a given link in this list.
func (ll *ListOfLinks) GetUsages(linkid int) []string {
	var usages []string
	l := ll.Links[linkid]

	// first use case: tagbindings are completely empty
	if len(ll.TagBindings) == 0 {
		// temporary fix for lists which were already created with {} as the tagbindings for this linkid
		LogError.Printf("NOTICE: Usages hack enabled for list '%s' and link %d\n", ll.Keyword, linkid)
		ll.TagBindings[l.ID] = []string{""}
	}

	// second use case: tagbindings are not empty, but we don't have a map entry for this link ID.
	if _, exists := ll.TagBindings[linkid]; !exists {
		LogError.Printf("NOTICE: Usages hack enabled for list '%s' and link %d\n", ll.Keyword, linkid)
		ll.TagBindings[l.ID] = []string{""}
	}

	if ll.TagBindings[linkid][0] != "" {
		tags := ll.TagBindings[linkid]
		for _, tag := range tags {
			if !l.Special() { // go2 keyword/tag
				usages = append(usages, fmt.Sprintf("%s %s/%s", RedirectorName, ll.Keyword, tag))
			} else { //go2 keyword/tag/parameter
				usages = append(usages, fmt.Sprintf("%s %s/%s/parameter", RedirectorName, ll.Keyword, tag))
			}
		}
	} else {
		if !l.Special() { // go2 keyword
			usages = append(usages, fmt.Sprintf("%s %s", RedirectorName, ll.Keyword))
		} else { // go2 keyword/parameter
			usages = append(usages, fmt.Sprintf("%s %s/parameter", RedirectorName, ll.Keyword))
		}
	}
	return usages
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

// Import will read data from the provided io.Reader into memory at the global
// LinkDataBase variable.
func (d *LinkDatabase) Import(fh io.Reader) error {
	var tempdb LinkDatabase
	var err error
	data, _ := io.ReadAll(fh)

	err = json.Unmarshal(data, &tempdb)
	if err != nil {
		LogError.Printf("json parsing error: %s", err)
		return err
	}

	LinkDataBase = &tempdb
	return err
}

// Export will marshal the current LinkDataBase into JSON and write it to the provided
// io.Writer.
func (d *LinkDatabase) Export(fh io.Writer) error {
	file, err := json.Marshal(*d)
	if err != nil {
		LogError.Println("JSON marshal error:", err)
		return err
	}
	_, err = fh.Write(file)
	if err != nil {
		LogError.Fatal(err)
	}
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

	// tag bindings - use case for "add an existing link to an existing otherlist"
	if _, exists := ll.TagBindings[linkObj.ID]; !exists {
		ll.TagBindings[linkObj.ID] = []string{""}
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

// Get a link object by ID or URL.
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
	// Check for the < count case
	if len(linkPile) < count {
		return linkPile
	}
	return linkPile[:count]
}

// Sort by modification time (Mtime)
func (d *LinkDatabase) LinksByMtime(count int) []*Link {
	linkPile := []*Link{}
	for _, link := range d.Links {
		linkPile = append(linkPile, link)
	}
	sort.Sort(ByMtime(linkPile))
	// Check for the < count case
	if len(linkPile) < count {
		return linkPile
	}
	return linkPile[:count]
}

func (d *LinkDatabase) LinksByClicks(count int) []*Link {
	linkPile := []*Link{}
	for _, link := range d.Links {
		linkPile = append(linkPile, link)
	}
	sort.Sort(ByLinkClicks(linkPile))
	// Check for the < count case
	if len(linkPile) < count {
		return linkPile
	}
	return linkPile[:count]
}

// TopLists returns a collection of lists of links, sorted by click count.
// The desired length of the returned result is the input.
// Use -1 to return all lists in the linkDB, sorted by click count.
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

var RedirectorMetadata = MakeNewMetadata()
