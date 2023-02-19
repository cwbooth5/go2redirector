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
	"time"

	"github.com/cwbooth5/go2redirector/api"
	gohttp "github.com/cwbooth5/go2redirector/http"

	"github.com/cwbooth5/go2redirector/core"
	"github.com/oxtoacart/bpool"
)

func handleKeyword(w http.ResponseWriter, r *http.Request, check chan<- string) (string, gohttp.ModelIndex, bool, error) {

	var tmpl string
	var model gohttp.ModelIndex
	var redirect, complete bool
	var err error

	request, err := core.MakeNewGoRequest(r)
	if err != nil || !request.Valid {
		check <- err.Error()
		return tmpl, model, redirect, err
	}

	msg := fmt.Sprintf("incoming URL path: %s", request.Path)
	core.LogDebug.Println(msg)
	check <- msg

	if request.EditMode {
		tmpl, model, _ = gohttp.RenderListPage(r)
	}

	// based on path length, handle as a redirect
	check <- fmt.Sprintf("parsed path length: %d", request.Path.Len())
	msg = fmt.Sprintf("Parsed keyword: '%s', tag: '%s', parameter: '%s'", request.Path.Keyword, request.Path.Tag, request.Path.Params)
	check <- msg

	ll, exists := core.LinkDataBase.Lists[request.Path.Keyword]
	if !exists {
		msg = fmt.Sprintf("keyword '%s' does not exist, rendering list page", request.Path.Keyword)
		core.LogDebug.Println(msg)
		check <- msg
		tmpl, model, err = gohttp.RenderListPage(r)
		return tmpl, model, redirect, err
	} else { //deboog
		msg = "keyword found, proceeding to follow path..."
		core.LogDebug.Println(msg)
		check <- msg
	}

	check <- fmt.Sprintf("This list's behavior: %d", ll.Behavior)

	switch {
	case request.Path.Len() == 1:
		var url string
		// first use case: /keyword (bare, no additional slash-delimited fields)
		// We returned early if this didn't exist, so we know it is in the db now.
		if core.EditMode(string(request.Path.Keyword)) {
			tmpl, model, err = gohttp.RenderListPage(r) // force to the list edit page
		} else {
			// It's a real redirect, follow the list's behavior now.
			lnk := core.LinkDataBase.GetLink(-1, ll.GetRedirectURL())
			if lnk.Special() {
				// If the link has substitutions, attempt to complete them using getURL.
				// If that completes the URL, redirect them.
				url, _, err = core.GetURL(lnk.URL, make(map[string]string), make(map[string]string), lnk.LinkVariables, check)
				if err != nil {
					tmpl, model, _ = gohttp.RenderListPage(r)
					// They screwed up their input - left out a parameter maybe?
					model.ErrorMessage = err.Error()
					return tmpl, model, redirect, err
				}
				msg = "This link is special (has substitutions in its URL)"
			} else {
				msg = "This link contains no substitutions in its URL"
			}
			check <- msg

			// check mode: render check page early
			if core.GetCheckMode(r) {
				core.LogDebug.Printf("CHECK MODE (%s): returning early\n", request.StringPath())
				return tmpl, model, redirect, err
			}

			if lnk.Special() && !complete {
				// listpage := fmt.Sprintf("/.%s", ll.Keyword)
				// http.Redirect(w, r, listpage, http.StatusTemporaryRedirect)
				errmsg := fmt.Sprintf("Generated URL is incomplete: '%s'", url)

				tmpl, model, _ = gohttp.RenderListPage(r) // force to the list edit page
				model.ErrorMessage = errmsg
				return tmpl, model, redirect, err
			}

			lnk.Clicks++
			core.LogDebug.Printf("Bare keyword redirect on '%s', clicks: %d\n", ll.Keyword, lnk.Clicks)
			core.LogInfo.Printf("Path '%s' redirect rendered: %s\n", request.Path.Keyword, ll.GetRedirectURL())
			check <- fmt.Sprintf("The URL this will redirect to: %s", lnk.URL)

			// Note we need to redirect THEN destroy the link.
			redirect = true
			if url != "" && !lnk.Special() {
				http.Redirect(w, r, url, http.StatusTemporaryRedirect)
			} else {
				http.Redirect(w, r, ll.GetRedirectURL(), http.StatusTemporaryRedirect)
			}
			if lnk.Dtime == core.BurnTime {
				core.LogInfo.Printf("Link %d is being burned.\n", lnk.ID)
				core.DestroyLink(lnk)
			}
		}
		return tmpl, model, redirect, err

	case request.Path.Len() == 2:
		// second use case: /keyword/param||tag
		// Check to see if the second term in the array starts with . or ends with /
		//     If so, we are rendering the link edit page for that link.
		//     If not, it is a bare tag.

		// The tag could have a leading dot or trailing slash. If so, that is an EDIT.
		if core.EditMode(request.Path.Tag) {
			tmpl, model, err = gohttp.RenderLinkPage(r)
			return tmpl, model, redirect, err
		}

		// Check for the use case of "existing keyword but missing tag"
		var tagfound bool
		for id, tagList := range ll.TagBindings {
			// This check is a stopgap for the effects of a bug where decoupling
			// left orphaned tagbindings on a list of links.
			if _, ok := ll.Links[id]; !ok {
				core.LogInfo.Printf("orphaned tag %d found on list %s\n", id, ll.Keyword)
				continue
			}
			for _, tag := range tagList {
				if request.Path.Tag == tag {
					var url string
					tagfound = true
					core.LogDebug.Printf("Tag '%s' located on list, link ID %d\n", request.Path.Tag, id)
					l := core.LinkDataBase.GetLink(id, "")
					if l.Special() { // special links get their URL changed with all the replacements.
						// If the link has substitutions, attempt to complete them using getURL.
						// If that completes the URL, redirect them.
						url, _, err = core.GetURL(l.URL, make(map[string]string), make(map[string]string), l.LinkVariables, check)
						if err != nil {
							tmpl, model, _ = gohttp.RenderListPage(r)
							// They screwed up their input - left out a parameter maybe?
							model.ErrorMessage = err.Error()
							return tmpl, model, redirect, err
						}
						msg = "This link is special (has substitutions in its URL)"
					} else {
						url = l.URL
						msg = "link is not special, has no replacements to do"
					}

					// common things to do when we found a matching tag and got our URL to redirect to.
					check <- msg

					// check mode: render check page early
					if core.GetCheckMode(r) {
						check <- fmt.Sprintf("The URL this will redirect to: %s", url)
						core.LogDebug.Printf("CHECK MODE (%s): returning early\n", request.StringPath())
						return tmpl, model, redirect, err
					}

					l.Clicks++
					core.LogInfo.Printf("Path '%s/%s' redirect rendered: %s\n", request.Path.Keyword, request.Path.Tag, url)
					core.LogDebug.Println("Redirecting based on tag")
					http.Redirect(w, r, url, http.StatusTemporaryRedirect)

					redirect = true
					if l.Dtime == core.BurnTime {
						msg = fmt.Sprintf("Link %d is being burned.\n", l.ID)
						core.LogInfo.Println(msg)
						core.DestroyLink(l)
					}

					return tmpl, model, redirect, err
				}
			}
		}
		if !tagfound {
			// This is the main branch for go2 thing/fancy-dynamic-parameter
			msg := fmt.Sprintf("Tag '%s' was not found under keyword '%s'", request.Path.Tag, request.Path.Keyword)
			check <- msg
			core.LogDebug.Println(msg)
			// Now we try to use their input as a parameter.
			url := ll.GetRedirectURL()

			core.LogDebug.Printf("Redirect URL for this list of links: '%s'\n", url)
			/*
				This could be a confusing spot for the user if they have redirect to "this page" and
				a substitution in their URL.
			*/
			if ll.Behavior == core.RedirectToList {
				check <- "behavior is 'this page' so user will land on edit page for this keyword"
			}

			if strings.ContainsAny(url, "{}") {

				// pth.Tag is being treated as a substitution parameter/variable
				msg = "final field is not a tag, so it is being treated as an input parameter"
				check <- msg
				l := core.LinkDataBase.GetLink(-1, url)
				l.Clicks++
				url, complete, err = gohttp.RenderSpecial([]string{request.Path.Tag}, l, ll, check)

				if err != nil {
					return tmpl, model, redirect, err
				}
				if complete { // substitution complete, they are redirected to the URL.

					msg = fmt.Sprintf("Path '%s/%s' redirect rendered: %s\n", request.Path.Keyword, request.Path.Tag, url)
					core.LogInfo.Println(msg)
					check <- msg
					redirect = true

					msg = fmt.Sprintf("Redirecting user to: %s\n", url)
					core.LogInfo.Println(msg)
					check <- msg

					// If check mode enabled, don't modify anything. Send to check page.
					if core.GetCheckMode(r) {
						core.LogDebug.Printf("CHECK MODE (%s): returning early\n", request.StringPath())
					} else {
						if l.Dtime == core.BurnTime {
							core.LogInfo.Printf("Link %d is being burned.\n", l.ID)
							core.DestroyLink(l)
						}
						http.Redirect(w, r, url, http.StatusTemporaryRedirect)
					}

					return tmpl, model, redirect, err
				}
			} else {
				// pth.Tag is being treated as a tag to look up a link in this list.
				msg = fmt.Sprintf("resolved redirectURL contains no variables and no tag was found: %s\n", url)
				core.LogDebug.Println(msg)
				check <- msg
				model.ErrorMessage = msg
			}
		}
		tmpl, model, err = gohttp.RenderListPage(r)

		if !redirect {
			err = fmt.Errorf("tag '%s' was not found on this list", request.Path.Tag)
			model.ErrorMessage = err.Error()
		}

		return tmpl, model, redirect, err

	case request.Path.Len() == 3:
		// Third use case: keyord/tag/param
		// We already know the list exists at this keyword.
		var url string
		for id, tagList := range ll.TagBindings {
			// This check is a stopgap for the effects of a bug where decoupling
			// left orphaned tagbindings on a list of links.
			if _, ok := ll.Links[id]; !ok {
				core.LogInfo.Printf("orphaned tag %d found on list %s\n", id, ll.Keyword)
				continue
			}
			for _, tag := range tagList {
				if request.Path.Tag == tag {
					for _, l := range ll.Links {
						if id == l.ID {
							// We have a link URL now at that tag on this list.
							url, complete, err = gohttp.RenderSpecial([]string{request.Path.Params[0]}, l, ll, check)
							check <- fmt.Sprintf("URL after special rendering: <code>%s</code>", url)

							if complete {
								// check mode: render check page early
								if core.GetCheckMode(r) {
									core.LogDebug.Println("CHECK MODE: returning without redirect")
								} else {
									core.LogInfo.Printf("Path '%s/%s' redirect rendered: %s\n", request.Path.Keyword, request.Path.Tag, url)
									l.Clicks++
									http.Redirect(w, r, url, http.StatusTemporaryRedirect)
									redirect = true
									if l.Dtime == core.BurnTime {
										core.LogInfo.Printf("Link %d is being burned.\n", l.ID)
										core.DestroyLink(l)
									}
								}
								return tmpl, model, redirect, err
							} // if incomplete, I guess we can error out?
						}
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
		get requests: core use case, humans do this
		if url path == / : return, render index page
		if url path has prefix . or suffix / == return, render dotpage/listpage
		else treat as keyword, sending to keyword handleFunc

		The "check" interface: They send a redirect in with check=true in the url parameters
	*/

	if r.RequestURI == "/favicon.ico" {
		// This is requested by so many browsers, we can handle it specifically
		// here to avoid nonsensical keyword lookups.
		http.Redirect(w, r, "/static/img/favicon.ico", http.StatusPermanentRedirect)
		return
	}

	/*
		New logic:
		This function is only responsible for http request handling and response sending.
	*/

	if r.Method != http.MethodGet {
		http.Error(w, "GET requests only", http.StatusBadRequest)
		return
	}

	request, reqerr := core.MakeNewGoRequest(r)
	if reqerr != nil {
		gohttp.IndexPage(w, r)
		return
	}

	// If for any reason their request didn't look like a go2 keyword/tag, they get index
	if r.URL.Path == "/" && !request.Valid {
		gohttp.IndexPage(w, r)
		return
	}

	// edge case: they typed 'check' and nothing else. moondog
	// if request.Path.Keyword == "" {
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	return
	// }

	// This turns their request around and sends it through the redirector as a checked redirect.
	if request.WantsCheck {
		var u string
		if core.ExternalPort == 0 {
			u = fmt.Sprintf("http://%s/%s?check=true", core.ExternalAddress, request.StringPath())
		} else {
			u = fmt.Sprintf("http://%s:%d/%s?check=true", core.ExternalAddress, core.ExternalPort, request.StringPath())
		}
		http.Redirect(w, r, u, http.StatusTemporaryRedirect)
		return
	}

	// The check interface
	// This is done in the main handler so we can follow the same code path
	// as a typical redirect.
	checkChan := make(chan string, 40) // buffer of 40 until blocking

	if request.CheckMode {
		core.LogInfo.Printf("check requested: %s\n", r.RequestURI)
		// If this was a check, we render the check page and include our model
		// The "variables" portion of the struct is []string so we can abuse that here
		// by filling it with whatever we had in our check channel.
		// call to handleKeyword is synchronous here, channel is buffered to allow it to run/return

		_, model, _, _ := handleKeyword(w, r, checkChan)
		tmpl := "check.gohtml"
		model.Title = "Check a redirect"
		model.ActiveUser = request.User
		model.RedirectorName = core.RedirectorName
		close(checkChan)
		for item := range checkChan {
			model.Variable = append(model.Variable, item)
		}
		gohttp.RenderTemplate(w, tmpl, &model)
		return
	}

	// We aren't in check mode.
	// They have a valid keyword at this point.
	// if in edit mode, go to edit on the page
	// Otherwise, normal request

	if request.EditMode {
		tmpl, model, _ := gohttp.RenderDotPage(r)
		// user input error (probably, among other things)
		// TODO on that user error...what's possible here?
		err := gohttp.RenderTemplate(w, tmpl, &model)

		if err != nil {
			core.LogError.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	} else {
		// call to handleKeyword is synchronous here, channel is buffered to allow it to run/return
		tmpl, model, redirect, err := handleKeyword(w, r, checkChan)

		if err != nil {
			model.ErrorMessage = err.Error()
		}

		if !redirect {
			gohttp.RenderTemplate(w, tmpl, &model)
			return
		}
	}

}

// Run the webserver frontend. This is only done when this instance of the redirector
// is the active member of the pair.
func configureWebserver(a string, p int) string {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/_suggest_/", gohttp.RouteSuggest)
	http.HandleFunc("/check/", gohttp.RouteCheck)
	http.HandleFunc("/api/", api.RouteAPI)
	http.HandleFunc("/404.html", gohttp.RouteNotFound)
	http.HandleFunc("/_link_/", gohttp.RouteLink)
	http.HandleFunc("/_login_", gohttp.RouteLogin)
	http.HandleFunc("/_db_", gohttp.RouteGetDB)
	http.HandleFunc("/_strings_/", gohttp.RouteStrings)
	http.HandleFunc("/_maps_/", gohttp.RouteMaps)
	http.HandleFunc("/", routeHappyHandler)
	core.LogInfo.Printf(fmt.Sprintf("Server starting with arguments: %s:%d", core.ListenAddress, core.ListenPort))
	return fmt.Sprintf("%s:%d", a, p)
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
}

func main() {
	/*
		User errors must be sent back to the users and must not panic anywhere here
		anything hitting URLs we cannot handle or objective errors must not log
	*/

	// TODO: flags for log levels
	go2Config, e := core.RenderConfig("go2config.json")
	if e != nil {
		log.Println("error loading local configuration file")
	}
	core.ListenAddress = go2Config.LocalListenAddress
	core.ListenPort = go2Config.LocalListenPort
	core.ExternalAddress = go2Config.ExternalAddress
	core.ExternalPort = go2Config.ExternalPort
	core.ExternalProto = go2Config.ExternalProto
	core.GodbFileName = go2Config.GodbFilename
	core.RedirectorName = go2Config.RedirectorName
	core.PruneInterval = go2Config.PruneInterval
	core.NewListBehavior = go2Config.NewListBehavior
	core.LevDistRatio = go2Config.LevDistRatio
	core.LinkLogNewKeywords = go2Config.LinkLogNewKeywords
	core.LinkLogCapacity = go2Config.LinkLogCapacity
	core.FailoverPeer = go2Config.FailoverPeer
	core.FailoverLocal = go2Config.FailoverLocal
	var logFile = go2Config.LogFile

	var importPath string
	var debugMode bool
	var listenAddress string
	var listenPort int
	flag.StringVar(&importPath, "i", core.GodbFileName, "Existing go2 redirector JSON DB to import")
	flag.BoolVar(&debugMode, "d", false, "Debug mode, set this to send debug logging to STDOUT")
	flag.StringVar(&listenAddress, "l", core.ListenAddress, "local TCP address to listen on, overrides LocalListenAddress in the config file")
	flag.IntVar(&listenPort, "p", core.ListenPort, "local TCP port to listen on, overrides LocalListenPort in the config file")
	flag.Parse()

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	core.ConfigureLogging(debugMode, file)

	// Render the opensearch XML template using config values.
	gohttp.RenderOpenSearch("templates/opensearch.goxml", "static/xml/opensearch.xml")

	// This directory isn't created in the repo by default because the opensearch.xml file
	// is generated by a template. The directory is created here at startup if it doesn't already exist.
	os.MkdirAll("static/xml", 0755)
	err = gohttp.RenderOpenSearch("templates/opensearch.goxml", "static/xml/opensearch.xml")
	if err != nil {
		log.Fatal(err)
	}

	/*
		This is a simple active-standby failover mechanism.
		If we see a value in the configuration file for the failover peer,
		We attempt to connect to it. If the peer is up, we assume a STANDBY role.
		If the peer is down, we assume an ACTIVE role.

		Active == web server is turned on, we are sending updates to the standby
		Standby == web server is off, we are receiving linkdb updates from the peer

		This is designed specifically as a simple failover mechanism. It only supports
		two systems in a coordinated pair.

		When the active system shuts off, it will miss its heartbeat on the standby.
		After a few seconds, the standby will load in the latest copy of the linkdb
		and it will turn on its webserver.
	*/
	updateChan := make(chan *core.LinkDatabase, 1)
	if core.FailoverPeer != "" {
		core.Synchronize()
		go core.RunFailoverMonitor(updateChan)
	}

	if !core.IsActiveRedirector {
		// This is the standby loop.
		core.LogInfo.Println("We are starting in STANDBY mode")
		for {
			select {
			case incomingDB := <-updateChan:
				core.LogDebug.Println("got a link DB update")
				core.LinkDataBase = incomingDB
			case <-time.After(2 * time.Second):
				core.LogError.Println("Timeout hit! Active did not sync with us. Assuming ACTIVE role...")
				core.IsActiveRedirector = true
				// This is the standby -> active transition. Note we are loading our linkdb
				// not from the disk, but from our core.LinkDataBase object.
				go core.SendUpdates(core.LinkDataBase)
				go core.PruneExpiringLinks()
				go core.CheckpointDB("300s")
				s := configureWebserver(listenAddress, listenPort)
				err := http.ListenAndServe(s, nil)
				if err != nil {
					core.LogError.Fatal(err)
				}
				break
			}
		}
	} else {
		// This is the active execution path.
		core.LogDebug.Println("We are starting in ACTIVE mode")

		// load the link database off the disk.
		core.LogDebug.Printf("Loading link database from file: %s", importPath)
		fh, err := os.Open(importPath)
		if err != nil {
			fmt.Printf("DB file '%s' could not be opened! Run the install script to create one.\n", core.GodbFileName)
		}
		err = core.LinkDataBase.Import(fh)
		if err != nil {
			core.LogError.Fatal(err)
		}

		// Initialize string and map variable structures
		if core.LinkDataBase.Variables == nil {
			core.LinkDataBase.Variables = &core.UserVariables{}
			core.LogDebug.Println("Variables data structure initialized")
		}
		// The upgrade case from a db containing variables but no Uses
		if core.LinkDataBase.Variables.Uses == nil {
			core.LinkDataBase.Variables.Uses = make(map[string][]*core.Link)
		}
		if core.LinkDataBase.Variables.Strings == nil {
			core.LinkDataBase.Variables.Strings = make(map[string]string)
			core.LogDebug.Println("String variables initialized")
		}
		if core.LinkDataBase.Variables.Maps == nil {
			core.LinkDataBase.Variables.Maps = make(map[string]map[string]string)
			core.LogDebug.Println("Map variables initialized")
		}
		// Init LinkZero fields not created in previous revisions of the DB schema
		// LinkVariables must not be null
		if core.LinkDataBase.Links[0].LinkVariables == nil {
			core.LinkDataBase.Links[0].LinkVariables = make(map[string]string)
			core.LogDebug.Println("LinkZero modified to init LinkVariables")
		}

		// If the file is found on disk, init metadata with that file.
		// Metadata is initialized empty before this, so an error is ignored for now and we default to "empty metadata"
		meta, metaerr := core.RedirectorMetadata.Import("go2metadata.json")
		if metaerr == nil {
			core.LogDebug.Println("Edit metadata found on disk and loaded: go2metadata.json")
			core.RedirectorMetadata = meta
		}

		// When we go active and we have a peer, we will start sending regular updates to
		// that peer indefinitely.
		if core.FailoverPeer != "" {
			go core.SendUpdates(core.LinkDataBase)
		}
		go core.PruneExpiringLinks()
		go core.CheckpointDB("300s")
		go core.IndexSearchDB("30s")

		// handle ctrl+c and sigterm - try to shut down gracefully and dump the db
		shutdownChan := make(chan os.Signal, 1)
		signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-shutdownChan
			core.Shutdown(core.LinkDataBase)
			os.Exit(1)
		}()

		ipPort := configureWebserver(listenAddress, listenPort)
		err = http.ListenAndServe(ipPort, nil)
		// most common errors are:
		// - port already in-use by another process
		// - insufficient privileges to bind to requested port
		if err != nil {
			log.Print(err.Error())
			os.Exit(1)
		}
	}
}
