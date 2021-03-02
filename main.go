package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"

	"github.com/cwbooth5/go2redirector/api"
	gohttp "github.com/cwbooth5/go2redirector/http"

	"github.com/cwbooth5/go2redirector/core"
	"github.com/oxtoacart/bpool"
)

// This is essentially a unit test.
func MakeStuff() {
	lnk, _ := core.MakeNewlink("www.reddit.com", "reddit")
	core.LinkDataBase.CommitNewLink(lnk)

	ll := core.MakeNewList(core.Keyword("r"), lnk)
	core.LinkDataBase.Couple(ll, lnk)
	ll.TagBindings[lnk.ID] = "pics"

	lnk, _ = core.MakeNewlink("www.127.0.0.1/foo", "local")
	core.LinkDataBase.CommitNewLink(lnk)
	core.LinkDataBase.Couple(ll, lnk)
	ll.TagBindings[lnk.ID] = "l"

	// wikipedia test of tagging
	lnk, _ = core.MakeNewlink("https://en.wikipedia.org/wiki/{subject}", "english wikipedia")
	core.LinkDataBase.CommitNewLink(lnk)
	ll = core.MakeNewList(core.Keyword("wiki"), lnk)
	core.LinkDataBase.Couple(ll, lnk)
	ll.TagBindings[lnk.ID] = "en"

	lnk, _ = core.MakeNewlink("https://it.wikipedia.org", "italian wikipedia")
	core.LinkDataBase.CommitNewLink(lnk)
	core.LinkDataBase.Couple(ll, lnk)
	ll.TagBindings[lnk.ID] = "it"

	lnk, _ = core.MakeNewlink("https://es.wikipedia.org", "spanish wikipedia")
	core.LinkDataBase.CommitNewLink(lnk)
	core.LinkDataBase.Couple(ll, lnk)
	ll.TagBindings[lnk.ID] = "es"

	lnk, _ = core.MakeNewlink("https://de.wikipedia.org", "german wikipedia")
	core.LinkDataBase.CommitNewLink(lnk)
	core.LinkDataBase.Couple(ll, lnk)
	ll.TagBindings[lnk.ID] = "de"

}

// If a page needs to be rendered, that is returned along with a model.
// If a redirect can be performed, it is done straight from this function.
func handleKeyword(w http.ResponseWriter, r *http.Request) (string, gohttp.ModelIndex, bool, error) {
	// If the keyword has nothing following it, perform a lookup.
	//    doesn't exist - editlink page
	//    does exist - follow behavior
	// LogDebug.Println("this is the handleKeyword function")
	var tmpl string
	var model gohttp.ModelIndex
	var redirect bool
	var err error
	var complete bool

	//var workingKeyword string

	// var complete bool
	core.LogDebug.Printf("URL path being parsed: %s\n", r.URL.Path)
	pth, err := core.ParsePath(r.URL.Path)
	core.LogDebug.Printf("Resulting keyword: %s\n", pth.Keyword)
	inputKeyword := r.URL.Query().Get("keyword") // only set if they entered a keyword in the input box
	if inputKeyword != "" {
		core.LogDebug.Printf("User supplied/input box keyword: %s\n", inputKeyword)
		// if the keyword has/slashes/within then we need to just use the first field here.
		inputSplit := strings.Split(inputKeyword, "/")
		pth.Keyword, _ = core.MakeNewKeyword(inputSplit[0])
		if len(inputSplit) > 1 {
			pth.Tag = inputSplit[1]
		}
	}
	core.LogDebug.Printf("parsed path in handle keyword: %s\n", pth.Keyword)
	ll, exists := core.LinkDataBase.Lists[pth.Keyword]
	if !exists {
		core.LogDebug.Printf("keyword '%s' does not exist.\n", pth.Keyword)
		tmpl, model, err = gohttp.RenderListPage(r)
		return tmpl, model, redirect, err
	} else { //deboog
		fmt.Println("keyword does exist, proceeding to follow path...")
		core.PrintList(*ll)
	}

	switch {
	case pth.Len() == 1:
		// first use case: /keyword (bare, no additional slash-delimited fields)
		// We returned early if this didn't exist, so we know it is in the db now.
		if core.EditMode(string(pth.Keyword)) {
			tmpl, model, err = gohttp.RenderListPage(r) // force to the list edit page
		} else {
			// It's a real redirect, follow the list's behavior now.
			redirect = true
			core.LogDebug.Printf("Bare keyword redirect on '%s'\n", ll.Keyword)
			http.Redirect(w, r, ll.GetRedirectURL(), 307)
		}

		return tmpl, model, redirect, err

	case pth.Len() == 2:
		// second use case: /keyword/param||tag
		// Check to see if the second term in the array starts with . or ends with /
		//     If so, we are rendering the link edit page for that link.
		//     If not, it is a bare tag.

		// The tag could have a leading dot or trailing slash. If so, that is an EDIT.
		if core.EditMode(pth.Tag) {
			tmpl, model, err = gohttp.RenderLinkPage(r)
			return tmpl, model, redirect, err
		}

		// What if they entered /keyword/param, where param was really a substitution they want to do?
		// algorithim: Look on the list to see if the redirectURL contains a substitution for this.
		//    if any {variable} is present in the url, treat this second value as a _parameter_
		//    if the sub is not present, treat this value as a _tag_ identifying a link in the list.

		url := ll.GetRedirectURL()
		if strings.ContainsAny(url, "{}") {
			// pth.Tag is being treated as a substitution parameter/variable
			l := core.LinkDataBase.GetLink(-1, url)
			url, complete, err = gohttp.RenderSpecial(r, []string{pth.Tag}, l, ll)
			if !complete {
				// pth.Tag is now being treated as a link tag.
				// Look in the ll tag mappings.
				// if lid, exists := ll.TagBindings[pth.Tag]; exists {
				// 	// redirect to that link's URL.
				// 	// what if the link has {variables}?
				// 	lnk := core.LinkDataBase.GetLink(lid, "")
				// 	if url, cmp := GetURL(lnk.URL, lnk.LinkVariables); cmp {
				// 		// subs complete using default link variables on the link, redirect.
				// 		http.Redirect(w, r, url, 307)
				// 	} else {
				// 		// substitutions were somehow not completed
				// 		http.Error(w, "default link variable substitutions did not yield a working link", http.StatusBadRequest)
				// 	}

				// 	// TODO, renderspecial with cookies code needed here
				// }
			} else {
				// substitution complete, they are redirected to the URL.
				http.Redirect(w, r, url, 307)
				redirect = true
				return tmpl, model, redirect, err
			}
		} else {
			// pth.Tag is being treated as a tag to look up a link in this list.
			core.LogDebug.Println("resolved redirectURL contains no variables.")
			// search for the tag in this list. If it is there, redirect to that link. If not, edit list page is rendered
			for id, tag := range ll.TagBindings {
				if pth.Tag == tag {
					core.LogDebug.Printf("Tag '%s' was found on list '%s'\n", pth.Tag, ll.Keyword)
					http.Redirect(w, r, ll.Links[id].URL, 307)
					redirect = true
					return tmpl, model, redirect, err
				}
			}
		}

	case pth.Len() == 3:
		// Third use case: keyord/tag/param
		// We already know the list exists at this keyword.
		fmt.Println("path len 3")
		// tag indicated the link we need to get a URL for.
		var url string
		var complete bool
		for id, tag := range ll.TagBindings {
			if pth.Tag == tag {
				for _, l := range ll.Links {
					if id == l.ID {
						// We have a link URL now at that tag on this list.
						url, complete, err = gohttp.RenderSpecial(r, []string{pth.Params[0]}, l, ll)
						if complete {
							core.LogDebug.Printf("Tag '%s' lookup url: %s\n", pth.Tag, url)
							http.Redirect(w, r, url, 307)
							redirect = true
							return tmpl, model, redirect, err
						}
						// if incomplete, I guess we can error out?
					}
				}
			}
		}

	default:
		// nonsensical input, figure out what to tell them
	}

	return tmpl, model, redirect, err
}

// This is the happy path handler for normal requests coming in.
func routeHappyHandler(w http.ResponseWriter, r *http.Request) {
	/*
		get requests only

		if url path == / : return, render index page
		if url path has prefix . or suffix / == return, render dotpage/listpage
		else treat as keyword, sending to keyword handleFunc
	*/

	if r.Method != http.MethodGet {
		// This is the interface for humans. If you want to post, hit the API.
		http.Error(w, "GET requests only", http.StatusBadRequest)
		return
	}

	p := r.URL.Path
	inputKeyword := r.URL.Query().Get("keyword")
	switch {
	case p == "/" && inputKeyword == "":
		core.LogDebug.Printf("\tindex page processing on path: %s\n", p)
		gohttp.IndexPage(w, r)
	case core.EditMode(strings.TrimPrefix(p, "/")):
		core.LogDebug.Printf("\tDotpage rendering for path: %s\n", p)
		tmpl, model, err := gohttp.RenderDotPage(r)
		// user input error (probably, among other things)
		// TODO on that user error...what's possible here?

		err = gohttp.RenderTemplate(w, tmpl, &model)

		// template rendering error
		if err != nil {
			core.LogError.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return

	default:
		// process it as a keyword
		var redirect bool
		core.LogDebug.Printf("\tDefault handling hit for path: %s\n", p)
		// special cases for common defaults browsers request
		if p == "/favicon.ico" {
			http.Error(w, "nope", http.StatusNotFound)
			return // until we have a favicon..
		}
		tmpl, model, redirect, _ := handleKeyword(w, r)
		if !redirect {
			gohttp.RenderTemplate(w, tmpl, &model)
			return
		} // otherwise, they've already been redirected via handleKeyword
	}
}

/*
	Initialization
*/

func init() {

	layouts, err := filepath.Glob("templates/*.gohtml")
	if err != nil {
		core.LogError.Fatal(err)
	}

	// We use this so errors during template renderings get buffered instead of blowing up
	// in the user's face. We get a chance to react to or handle problems.
	gohttp.Bufpool = bpool.NewBufferPool(64)

	for _, layout := range layouts {
		gohttp.Templates[filepath.Base(layout)] = template.Must(template.ParseFiles(layout, "templates/base.gohtml"))
	}

	// logging setup
	core.LogInfo = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lmsgprefix)
	core.LogError = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)
	core.LogDebug = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)

	// handle ctrl+c and sigterm - try to shut down gracefully and dump the db
	shutdownChan := make(chan os.Signal, 1)

	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-shutdownChan
		core.Shutdown()
		os.Exit(1)
	}()
}

func main() {
	/*
		User errors must be sent back to the users and must not panic anywhere here
		anything hitting URLs we cannot handle or objective errors must not log
	*/

	// TODO: flags for log levels
	go2Config, e := core.RenderConfig("go2config.json")
	if e != nil {
		core.LogError.Fatal("error loading local configuration file")
	}
	core.ListenAddress = go2Config.LocalListenAddress
	core.ListenPort = go2Config.LocalListenPort
	core.ExternalAddress = go2Config.ExternalAddress
	core.ExternalPort = go2Config.ExternalPort
	core.GodbFileName = go2Config.GodbFilename
	core.RedirectorName = go2Config.RedirectorName
	core.PruneInterval = go2Config.PruneInterval
	core.NewListBehavior = go2Config.NewListBehavior
	core.LevDistRatio = go2Config.LevDistRatio
	core.LinkLogNewKeywords = go2Config.LinkLogNewKeywords
	core.LinkLogCapacity = go2Config.LinkLogCapacity

	var importPath string
	flag.StringVar(&importPath, "i", core.GodbFileName, "Existing go2 redirector JSON DB to import")
	flag.Parse()

	core.LogInfo.Printf("Loading link database from file: %s", importPath)
	core.LinkDataBase.Import(importPath)

	s := fmt.Sprintf("Server starting with arguments: %s:%d", core.ListenAddress, core.ListenPort)
	core.LogInfo.Println(s)

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/_link_/", gohttp.RouteLink)
	http.HandleFunc("/api/", api.RouteAPI)
	http.HandleFunc("/_db_", gohttp.RouteGetDB)
	http.HandleFunc("/404.html", gohttp.RouteNotFound)
	http.HandleFunc("/", routeHappyHandler) // golden happy path because why not?

	// LogDebug.Printf(IdentifyBuild())
	go core.PruneExpiringLinks()

	// MakeStuff()

	p := fmt.Sprintf("%s:%d", core.ListenAddress, core.ListenPort)
	err := http.ListenAndServe(p, nil)
	// err := http.ListenAndServeTLS(p, "go2.home.cwb.crt", "go2.home.cwb.key", nil)
	if err != nil {
		core.LogError.Fatal(err)
	}

}
