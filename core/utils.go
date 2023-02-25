package core

import (
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	LogInfo  *log.Logger
	LogError *log.Logger
	LogDebug *log.Logger
)

// ConfigureLogging will set debug logging up with the -d flag when this program is run.
func ConfigureLogging(debug bool, w io.Writer) {
	LogInfo = log.New(w, "INFO: ", log.Ldate|log.Ltime|log.Lmsgprefix)
	LogError = log.New(w, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)
	if debug {
		LogDebug = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)
	} else {
		LogDebug = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)
		LogDebug.SetOutput(io.Discard)
	}
}

/*
	Utility functions
*/

// ByClicks implements sorting on an array of ListOfLinks by click count (descending).
type ByClicks []*ListOfLinks

func (a ByClicks) Len() int           { return len(a) }
func (a ByClicks) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByClicks) Less(i, j int) bool { return a[j].Clicks < a[i].Clicks }

// ByAtime implements sorting by access time on an array of links.
type ByAtime []*Link

func (a ByAtime) Len() int           { return len(a) }
func (a ByAtime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAtime) Less(i, j int) bool { return a[j].Atime.Before(a[i].Atime) }

// ByMtime implements sorting by modify time on an array of links.
type ByMtime []*Link

func (a ByMtime) Len() int           { return len(a) }
func (a ByMtime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByMtime) Less(i, j int) bool { return a[j].Mtime.Before(a[i].Mtime) }

// ByLinkClicks implements sorting by click count on each link.
type ByLinkClicks []*Link

func (a ByLinkClicks) Len() int           { return len(a) }
func (a ByLinkClicks) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLinkClicks) Less(i, j int) bool { return a[j].Clicks < a[i].Clicks }

// Return a URL of this redirector, used for redirects back to index
// If ExternalPort and ListenPort are different in the configuration, return only the ExternalAddress.
// Otherwise, return the explicit listen address/port.
func ListenURL() *url.URL {
	var s string
	if ExternalPort != ListenPort {
		s = fmt.Sprintf("http://%s", ExternalAddress)
	} else {
		s = fmt.Sprintf("http://%s:%d", ListenAddress, ListenPort)
	}
	url, err := url.Parse(s)
	if err != nil {
		log.Fatalf("Config URL would not parse. Check configured addresses and ports. '%s'\n", s)
	}
	return url
}

func TopLink(ll ListOfLinks) *Link {
	return ll.ClickSort()[0]
}

/*
Shutdown is meant to handle graceful termination of the redirector.

Ctrl+C/sigterm is the signal this runs on.
*/
func Shutdown(d *LinkDatabase, s chan int) {
	log.Println("Signal caught. Shutting down...")

	// prevent destruction of data by writing to a backup first.
	tmpfile := fmt.Sprintf("%s.tmp", GodbFileName)
	fh, err := os.Create(tmpfile)
	if err != nil {
		log.Fatalf("Could not export to %s\n", tmpfile)
	}
	defer fh.Close()
	d.Export(fh, s)

	// now move over the existing db file so it can be used on the next startup
	err = os.Rename(tmpfile, GodbFileName)
	if err != nil {
		log.Fatalf("Could not move %s to %s\n", tmpfile, GodbFileName)
	}

	RedirectorMetadata.Export("go2metadata.json")
}

// CheckpointDB saves a copy of the link database at a provided interval (a time duration string).
// This also syncs the DB to the failover peer through a TCP connection at the same interval.
func CheckpointDB(duration string, s chan int) {
	d, err := time.ParseDuration(duration)
	if err != nil {
		LogError.Fatalf("Specified duration of '%s' could not be parsed\n", duration)
	}
	for {
		// purpose 1: sync db to peer on a time interval

		//

		// purpose 2: export the db to local disk as a backup
		fileName := fmt.Sprintf("%s.bak", GodbFileName)
		fh, err := os.Create(GodbFileName)
		if err != nil {
			LogInfo.Printf("%s\n", err)
			LogError.Fatalf("Could not export to %s\n", GodbFileName)
		}

		err = LinkDataBase.Export(fh, s)
		if err != nil {
			LogError.Fatalf("DB checkpoint to file '%s' failed. %s", fileName, err)
		}
		time.Sleep(d)
	}
}

// rotateSlice adds val at s[0], rotating all existing elements 1 position rightward
// This limits capacity of s to LinkLogCapacity, as defined in the config file.
func RotateSlice(s []string, val string) []string {
	// This inserts at the front of the slice.
	s = append(s, "")
	copy(s[1:], s[0:])
	s[0] = val
	if len(s) > LinkLogCapacity {
		s = s[:LinkLogCapacity]
	}
	return s
}

func PrependEdit(e []*EditRecord, val *EditRecord) []*EditRecord {
	e = append(e, &EditRecord{})
	copy(e[1:], e[0:])
	e[0] = val
	if len(e) > 5 {
		e = e[:5]
	}
	return e
}

// FormatRequest generates ascii representation of a request
// It useful for printing incoming requests on the console.
func FormatRequest(r *http.Request) string {
	var request []string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	// Add the host
	request = append(request, fmt.Sprintf("Host: %v", r.Host))
	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == "POST" {
		r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	}
	// Return the request as a string
	return strings.Join(request, "\n")
}

// This is used to find keywords which are similar to others.
// Levenshtein distance algorithm
func calcLevDist(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	if a == b {
		return 0
	}

	// ensure the shortest of the two is first (optimization)
	if len(a) > len(b) {
		a, b = b, a
	}

	arr := make([]int, len(a)+1)

	for i := 1; i < len(arr); i++ {
		arr[i] = int(i)
	}

	for i := 1; i <= len(b); i++ {
		p := i
		for j := 1; j <= len(a); j++ {
			c := arr[j-1]
			if b[i-1] != a[j-1] {
				c = min(min(arr[j-1]+1, p+1), arr[j]+1)
			}
			arr[j-1] = p
			p = c
		}
		arr[len(a)] = p
	}
	return arr[len(a)]
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// Print out/log similarity ratio, returning the ratio.
func Similar(reference, comparison string) float64 {

	ld := calcLevDist(reference, comparison)

	// substring matches are helpful for keywords which mirror special redirects or plurals.
	// This can catch nonsensical substrings that have nothing to do with larger strings. example: 'it' in 'bitbucket'
	// It might be possible to compare lengths here to tune out these kinds of matches...

	// This is the ratio of levdist / len(reference)
	// Lower LD alone gets too many similarity matches as reference length decreases.
	// This accounts for very short terms.
	ratio := float64(ld) / float64(len(reference))

	if ld == 0 {
		return 0
	}
	return ratio
}

// just a pretty print of this struct
func PrintList(ll ListOfLinks) {
	v := reflect.ValueOf(ll)
	typeOF := v.Type()

	for i := 0; i < v.NumField(); i++ {
		LogDebug.Printf("Field: %s\tValue: %v\n", typeOF.Field(i).Name, v.Field(i).Interface())
	}
}

// destroyLink will remove a link object, then decouple it from all
// lists that link is a member of.
func DestroyLink(l *Link) {
	for _, list := range l.Lists {
		lol := LinkDataBase.Lists[list]
		LogInfo.Printf("Decoupling link %d from %v\n", l.ID, lol)
		LinkDataBase.Decouple(lol, l)
	}
	delete(LinkDataBase.Links, l.ID) // Remove link object entirely
}

// pruneExpiringLinks will look through the link database and delete links which
// have a Dtime in the past.
func PruneExpiringLinks(c chan int) {
	duration, _ := time.ParseDuration(PruneInterval)
	for {
		<-c
		LinkDataBase.Prune()
		c <- 1
		time.Sleep(duration)
	}
}

// Populate easily-searchable structures with information from the real linkDB.
// This is meant to be run in a goroutine every 30 seconds or so.
func IndexSearchDB(interval string, s chan int) {
	duration, _ := time.ParseDuration(interval)
	for {
		<-s
		LinkDataBase.IndexKeywords()
		s <- 1
		time.Sleep(duration)
	}
}

/*
GetURL takes a URL string with substitutions and returns a final URL with all substitutions
performed. The "complete" boolean indicates whether no more substitutions need to be done.

The {1} is still supposed to be the entire parameter incoming from the user's request., so we still
need to substitute it into the URL where specified.
TODO: make it impossible to make a variable named "1" to eliminate confusion
*/
func GetURL(url string, mapLookups map[string]string, mappings map[string]string, lv map[string]string, check chan<- string) (string, bool, error) {
	// take an array of linkvariablesmaps and perform substitutions on the URL, returning a final URL.
	finalURL := url
	var complete bool
	var err error
	LogDebug.Printf("URL prior to substitutions: '%s'\n", url)

	/*
		This is where we parse the different go2redirector-specific fields out of their template URL.
		{1} - Any integer surrounded in braces: the entire parameter string
		{stringname} - Lookup for a string variable of this name
		{mapname[key]} - Lookup for a map variable, get value stored at key in mapname

		We have to build another regex here to match all the potential types of variables embedded in their URL.
	*/
	check <- fmt.Sprintf("Default behaviors for captured fields: <code>%s</code>", lv)
	// for this one, we already have their input parameter(s) so we aren't looking anything up. Just do the replacement now.
	for pattern, replacement := range mappings {
		searchStr := fmt.Sprintf("{%s}", pattern)
		msg := fmt.Sprintf("Search pattern for positional parameter: <code>%s</code>", searchStr)
		check <- msg
		finalURL = strings.Replace(finalURL, searchStr, replacement, -1) // replace all instances
	}

	// return early if these were the only substitution(s).
	if !strings.ContainsAny(finalURL, "{}") {
		complete = true
		return finalURL, complete, err
	}

	/* Step 2 is a pass over the URL for the {$string} subs
	{$string} means to look up a string variable so we can do a replacement.
	{string} means look for the capture group 'string' and use whatever we captured as the substitution.
	*/
	re2 := regexp.MustCompile(`\{(\$[-\w]+)\}`)
	stringsresult := re2.FindAllStringSubmatch(finalURL, -1)
	check <- fmt.Sprintf("string lookup(s) found in URL: <code>%s</code>", stringsresult)
	for _, v := range stringsresult { // v[0] == full match, v[1] == capture group
		if strings.HasPrefix(v[1], "$") {
			// This is an explicit string variable lookup.
			// look for a hit (or not) and print that, changing the URL only if it is a non-nil hit
			LogDebug.Printf("string lookup requested: %s", v[1])
			variableName := strings.TrimLeft(v[1], "$")
			replacement := LinkDataBase.Variables.Strings[variableName]
			if replacement == "" {
				msg := fmt.Sprintf("string variable '%s' blank or unset, url not modified\n", variableName)
				check <- msg
				LogDebug.Printf(msg)
				// This is a semantic misconfiguration on their link template or variable. Tell them.
				err = errors.New(msg)
			} else { // success
				searchStr := fmt.Sprintf("{$%s}", variableName)
				msg := fmt.Sprintf("String variable %s will be replaced with '%s' in the URL", searchStr, replacement)
				check <- msg
				LogDebug.Println(msg)
				finalURL = strings.Replace(finalURL, searchStr, replacement, -1) // replace all instances
				LogDebug.Printf("new URL: %s\n", finalURL)
			}
		} else {
			// TODO: figure out if this alternate syntax of no $ on strings even makes sense to keep.
			// We already have {1}...
		}
	}

	// return early if there are no more substitutions
	if !strings.ContainsAny(finalURL, "{}") {
		complete = true
		return finalURL, complete, err
	}

	/*
		Step 3 is a pass over the URL for the {$map[key]} subs
		Only allow map names containing hyphens and letters

		LinkVariables is now defining fallbacks/defaults like python's
		dict.get() function. The potential options for each variable are:
		"none" == No default, fail with no error if lookup failed
		"error" == no default, fail with error if lookup failed
		"input" == use input value as replacement if lookup failed
	*/
	re3 := regexp.MustCompile(`\{(\$?[a-zA-Z-]+)\[(\w+)\]\}`)
	mapsresult := re3.FindAllStringSubmatch(finalURL, -1)
	check <- fmt.Sprintf("map lookup(s) found in URL: <code>%s</code>", mapsresult)
	for _, v := range mapsresult { // array of [entire match, key, value] we only need the latter two
		searchStr, mapname, mapkey := v[0], v[1], v[2]
		mapname = strings.TrimLeft(mapname, "$") // strip off the "$" variable syntax
		check <- fmt.Sprintf("map name[key]: <code>%s[%s]</code>", mapname, mapkey)
		// mapname here is accurate. mapkey is the visual string, the name of the named capture group.
		LogDebug.Printf("map lookup requested: %s[%s]\n", mapname, mapLookups[mapkey])
		replacement := LinkDataBase.Variables.Maps[mapname][mapLookups[mapkey]]
		check <- fmt.Sprintf("result of map lookup: %s", replacement)
		if replacement == "" {
			// The lookup failed. Variable not there, key missing...
			msg := fmt.Sprintf("map lookup did not return a hit, taking default action of: <code>%s</code>", lv[mapkey])
			check <- msg
			LogDebug.Println(msg)
			switch lv[mapkey] { // Check to see if a default is set.
			case "none":
				msg = fmt.Sprintf("map variable '%s' blank or unset, no default is set, no error to the user", mapname)
				LogInfo.Println(msg)
			case "error":
				msg = "Map lookup failed, sending error to user: check if map/key exist"
				LogInfo.Println(msg)
				err = fmt.Errorf("lookup for map '%s' failed, check if map name or key exist", mapname)
			case "input":
				msg = fmt.Sprintf("map lookup $%s['%s'] failed, using default", mapname, mapLookups[mapkey])
				LogInfo.Println(msg)
				replacement = mapLookups[mapkey]                                 // just pass their input straight through
				finalURL = strings.Replace(finalURL, searchStr, replacement, -1) // replace all instances
			}
			check <- msg
		} else {
			// lookup success
			finalURL = strings.Replace(finalURL, searchStr, replacement, -1) // replace all instances
		}
		LogDebug.Printf("New/final URL: %s\n", finalURL)
	}

	// Let the caller know if there are remaining substitutions to be done.
	// At this point a failed lookup could yield a URL missing replacement fields entirely.
	complete = !strings.ContainsAny(finalURL, "{}")
	msg := fmt.Sprintf("URL substitutions finished: '%s'", finalURL)
	check <- msg
	LogDebug.Println(msg)

	// we can never redirect to a URL which is empty or which still has subs, redirect to link page
	if finalURL == "" {
		msg = "ERROR: URL was empty after substitutions"
		err = fmt.Errorf(msg)
	} else if strings.ContainsAny(finalURL, "{}") {
		msg = "ERROR: URL had leftover substitutions (check for typos, variable existence)"
		err = fmt.Errorf(msg)
	}
	check <- msg
	return finalURL, complete, err
}

// NewLinkID is for templates' string -> linkID conversion
func NewLinkID(id string) int {
	i, err := strconv.Atoi(id)
	if err != nil {
		LogError.Printf("Input ID could not be converted to an integer. '%s'\n", id)
	}
	return i
}

// GetExpireTime returns a date used to set the delete time on a link.
// The only reason this exists is to add error checking around parsing of the duration string.
func GetExpireTime(start time.Time, duration string) (time.Time, error) {
	var e error
	// We need a delta from the date they gave us, using the duration they wanted to use.
	dur, err := time.ParseDuration(duration)
	if err != nil {
		msg := fmt.Errorf("this duration string could not be parsed: %s", duration)
		LogError.Println(msg)
		e = msg
	}
	return start.Add(dur), e
}

// Return true if a string indicates 'edit mode', meaning the user wanted to force
// an edit page for something.
func EditMode(s string) bool {
	return strings.HasPrefix(s, ".") || strings.HasPrefix(s, "/.") || strings.HasSuffix(s, "/")
}

// Check whether a string would create a valid keyword
func IsValidKeyword(s string) bool {
	_, err := MakeNewKeyword(s)
	return err == nil
}

// Clean up anomalies in a URL string due to omissions or user inputs
func SanitizeURL(u string) string {
	var cleaned string
	cleaned = strings.TrimSpace(u)
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "")
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "slack://") {
		// This prefix has to be here because redirects require it. Browsers need it to understand it's a URL.
		cleaned = fmt.Sprintf("http://%s", cleaned)
	}
	return cleaned
}

/*
ParsePath takes a URL path entered by a user and breaks
down the path into its constituent parts.
This will never return a keyword with the leading /. if it is provided to this function.
*/
func ParsePath(s string) (Gpath, error) {
	var gp Gpath
	var err error
	var k Keyword
	var t string
	var p []string

	s = html.EscapeString(s)
	// check mode - strip it if it's there.
	tr := strings.TrimPrefix(s, "check")
	tr = strings.TrimPrefix(tr, "/")
	tr = strings.TrimPrefix(tr, ".")

	if tr == "" {
		err = errors.New("invalid keyword, blank")
		gp := Gpath{k, t, p}
		return gp, err
	}

	sp := strings.Split(tr, "/")
	k, err = MakeNewKeyword(sp[0])
	if len(sp) > 2 {
		t = sp[1]
		p = sp[2:]
	} else if len(sp) > 1 {
		t = sp[1]
	}
	gp = Gpath{
		Keyword: k,
		Tag:     t,
		Params:  p,
	}

	return gp, err
}

/*
Extract the user login name from any cookies presented in their requests.
The cookie name will be 'redirectorlogin' and the value is their login name.
*/
func ExtractUser(r *http.Request) string {
	if len(r.Cookies()) > 0 {
		for _, c := range r.Cookies() {
			if c.Name == "redirectorlogin" {
				return c.Value
			}
		}
	}
	return ""
}

// This returns a human-readable behavior or a link title if direct is selected as the behavior.
func GetPrettyBehaviorString(b int) string {
	switch b {
	case -1:
		return "this page"
	case -2:
		return "freshest link"
	case -3:
		return "most used link"
	case -4:
		return "random link"
	default:
		// The list redirects to a specific link. Get its title.
		return LinkDataBase.Links[b].Title
	}
}

// GetCheckMode reads an HTTP request object and returns a boolean indicating
// whether or not check mode is enabled on the request.
// The only indicator of check mode is a URL parameter called "check"
// with a value of "true" or "false".
func GetCheckMode(r *http.Request) bool {
	switch r.URL.Query().Get("check") {
	case "true":
		return true
	case "false":
		return false
	default:
		return false
	}
}

type GoRequest struct {
	WantsCheck bool   // keyword starts with 'check'
	CheckMode  bool   // request has check=true URL parameter
	EditMode   bool   // true if keyword starts with '.'
	Path       Gpath  // internal path representation
	Valid      bool   // Is the keyword valid?
	User       string // pulled from the cookie their browser sent
}

// We need the full path they entered for reconstructing internal links to redirect to.
// Main use case is redirecting to the check interface using the url param check=true
func (g *GoRequest) StringPath() string {
	var a []string
	if g.Path.Keyword != "" {
		a = append(a, string(g.Path.Keyword))
	}
	if g.Path.Tag != "" {
		a = append(a, g.Path.Tag)
	}
	p := strings.Join(a, "/")
	return p
}

/*
Parse an incoming request for go2 semantics and any errors doing so
*/
func MakeNewGoRequest(r *http.Request) (GoRequest, error) {
	var err error
	var path Gpath
	var valid, edit, wants bool
	check := GetCheckMode(r)
	user := ExtractUser(r)
	path, err = ParsePath(r.URL.Path) // will trim 'check', if it exists
	if err == nil {
		valid = true
		edit = EditMode(r.URL.Path)
	} else {
		// problem locating keyword, check params
		path, err = ParsePath(r.URL.Query().Get("keyword"))
		if err == nil {
			valid = true
			edit = EditMode(r.URL.Query().Get("keyword"))
		}
	}
	// If the request has a leading 'check' they want to follow check behavior
	if len(r.URL.Query().Get("keyword")) > 0 {
		if strings.Split(r.URL.Query().Get("keyword"), "/")[0] == "check" {
			wants = true
		}
	}

	return GoRequest{
		WantsCheck: wants,
		CheckMode:  check,
		EditMode:   edit,
		User:       user,
		Path:       path,
		Valid:      valid,
	}, err
}

// search the entire link database using a string as input
// This can be used both for suggestions and full search
// This search algorithm weights the results based on how much of a match we find.
// Results are ordered from most to least relevant in the returned array.
func SearchDB(term string, maxresults int, s chan int) []string {
	term = strings.ToLower(term)
	term = strings.TrimSpace(term)

	// using a map for uniqueness
	// values are weight from 1-100 (1 is most relevant)
	targets := make(map[string]int)

	<-s

	// exact match on a keyword
	if SearchKeywordsTrie.Search(term) {
		targets[term] = 1
	} else {
		if SearchKeywordsTrie.Startswith(term) {
			targets[term] = 30
		}
	}

	/*
		This data is a map of keyword: big-ass string of tags, link titles, and variables
	*/
	for kwd, searchdata := range SearchKeywordsData {
		// substring match _inside_ keyword
		if strings.Contains(kwd, term) {
			targets[kwd] = 10
			continue
		}
		// substring match inside keyword _data_
		if strings.Contains(searchdata, term) {
			if _, exists := targets[kwd]; !exists {
				targets[kwd] = 20
				continue
			}
		} else {
			if term == kwd {
				continue // already matched/added above
			}
			ratio := Similar(term, kwd)
			if ratio < LevDistRatio && term != kwd {
				// low ratio
				if _, exists := targets[kwd]; !exists {
					targets[kwd] = 80
					continue
				}
			} else if strings.Contains(term, kwd) || strings.Contains(kwd, term) {
				// substring match
				if _, exists := targets[kwd]; !exists {
					targets[kwd] = 60
					continue
				}
			}
		}
	}

	keys := make([]string, 0, len(targets)) // keywords

	buckets := make(map[int][]string)

	for k, v := range targets {
		buckets[v] = append(buckets[v], k)
	}

	weights := make([]int, 0, len(buckets))
	for w := range buckets {
		weights = append(weights, w)
	}
	sort.Ints(weights) // ascending

	for _, w := range weights {
		// for all terms at that weight
		keys = append(keys, buckets[w]...) // extend
	}

	var searchTerms []string // Firefox uses only this array

	if len(keys) > maxresults {
		searchTerms = keys[:maxresults]
	} else {
		searchTerms = keys
	}
	LogDebug.Printf("search term: %s, results: %s\n", term, searchTerms)
	s <- 1

	return searchTerms
}
