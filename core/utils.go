package core

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
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
func ConfigureLogging(debug bool, logFile string) {
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	LogInfo = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lmsgprefix)
	LogError = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)
	if debug {
		LogDebug = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)
	} else {
		LogDebug = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)
		LogDebug.SetOutput(ioutil.Discard)
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

func Shutdown() {
	//TODO: actual cleanup code. dump db to disk
	LogInfo.Println("Signal caught. Shutting down...")
	LinkDataBase.Export(GodbFileName)
}

// CheckpointDB saves a copy of the link database at a provided interval (a time duration string).
func CheckpointDB(duration string) {
	fileName := fmt.Sprintf("%s.bak", GodbFileName)
	d, err := time.ParseDuration(duration)
	if err != nil {
		LogError.Fatalf("Specified duration of '%s' could not be parsed\n", duration)
	}
	for {
		err := LinkDataBase.Export(fileName)
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
	} else if ratio < 0.30 {
		LogDebug.Printf("These are VERY similar '%s' and '%s' ld: %d ratio: %.2f\n", reference, comparison, ld, ratio)
	} else if ratio < LevDistRatio {
		LogDebug.Printf("These are SOMEWHAT similar '%s' and '%s' ld: %d ratio: %.2f\n", reference, comparison, ld, ratio)
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
func PruneExpiringLinks() {
	duration, _ := time.ParseDuration(PruneInterval)
	for {
		now := time.Now()
		for id, lnk := range LinkDataBase.Links {
			if lnk.Dtime.Before(now) {
				// special case: If it's a burner, leave it where it is.
				if lnk.Dtime.Equal(BurnTime) {
					continue
				}
				DestroyLink(lnk)
				LogInfo.Printf("Pruning link from database: %d", id)
			}
		}
		time.Sleep(duration)
	}
}

// GetURL takes a URL string with substitutions and returns a final URL with all substitutions
// performed. The "complete" boolean indicates whether no more substitutions need to be done.
func GetURL(url string, mappings map[string]string) (string, bool) {
	// take an array of linkvariablesmaps and perform substitutions on the URL, returning a final URL.
	finalURL := url
	var complete bool
	LogDebug.Println(mappings)
	LogDebug.Printf("URL prior to substitutions: '%s'\n", url)
	for pattern, replacement := range mappings {
		// searchStr := fmt.Sprintf("{%s}", l.LinkVariables[idx].Pattern)
		searchStr := fmt.Sprintf("{%s}", pattern)
		LogDebug.Printf("Search pattern: %s ", searchStr)
		LogDebug.Printf("Replace pattern: %s\n", replacement)
		finalURL = strings.Replace(finalURL, searchStr, replacement, -1) // replace all instances
	}

	// For convenience, let the caller know if there are remaining substitutions to be done.
	complete = !strings.ContainsAny(finalURL, "{}")
	LogDebug.Printf("url after subs: '%s'\n", finalURL)
	if finalURL == "" {
		panic("EMPTY URL WAS RENDERED")
	}
	return finalURL, complete
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
	return strings.HasPrefix(s, ".") || strings.HasSuffix(s, "/")
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

// ParsePath takes a URL path entered by a user and consults the link DB to break
// down the path into its constituent parts.
// This will never return a keyword with the leading /. if it is provided to this function.
func ParsePath(s string) (Gpath, error) {
	var gp Gpath
	var err error
	var k Keyword
	var t string
	var m []string
	tr := strings.TrimPrefix(s, "/")
	tr = strings.TrimPrefix(tr, ".")
	LogDebug.Printf("trimmed kwd: '%s'\n", tr)
	// error checking: paths cannot start with unders Those are special pages.
	if strings.HasPrefix(tr, "_") {
		err = fmt.Errorf("Path could not be parsed due to underscore")
	}
	sp := strings.Split(tr, "/")
	k, _ = MakeNewKeyword(sp[0])
	if len(sp) > 2 {
		t = sp[1]
		m = sp[2:]
	} else if len(sp) > 1 {
		t = sp[1]
	}
	gp = Gpath{
		Keyword: k,
		Tag:     t,
		Params:  m,
	}
	// LogDebug.Printf("gpath: %s\n", gp)
	return gp, err
}
