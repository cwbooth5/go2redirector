package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/cwbooth5/go2redirector/core"
)

/*
	HTTP handling and routes
*/

// Provide an external URL used to get the entire DB in JSON format
func RouteGetDB(w http.ResponseWriter, r *http.Request) {
	data, err := json.Marshal(core.LinkDataBase)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	core.LogDebug.Println("_db_ route hit")
	w.Write(data)
}

// func routeCustomHandler(w http.ResponseWriter, r *http.Request) {
// 	// We need a custom function here to handle all the interesting paths in our normal use cases.

// 	//log.Println("Handling request for path:", r.URL.Path)
// 	// if the URL path does not start with an underscore, treat it as an attempted redirect.

// 	//TODO this might be a place where we just 404 them on garbage input.

// 	fmt.Println(core.FormatRequest(r))

// 	var genParam []string
// 	if r.Method == http.MethodGet && !strings.HasPrefix(r.URL.Path, "/_") && r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/.") {

// 		var safeKeyword core.Keyword
// 		var err error

// 		k := r.URL.EscapedPath() // Example result: /kwdname/param1/param2
// 		// log.Printf("escaped path: %q", k)
// 		m := strings.TrimPrefix(k, "/")
// 		// log.Printf("keyword: %v", m)
// 		p := strings.SplitAfterN(m, "/", 2) // ["kwdname/", "param1/param2"]
// 		core.LogDebug.Printf("Incoming URL escaped path items: %q\n", p)
// 		core.LogDebug.Printf("Cookies provided on this direct keyword: %s\n", r.Cookies())

// 		// this used to go looking for the slash on the end. Now we look at all terms.
// 		// zero terms provided: they get the index page (shouldn't ever get to this function)
// 		// one term provided: they entered a keyword. look it up, redirect or create if needed
// 		// two or more terms: treat the first as the keyword, all the rest as the dynamic variables in left-to-right order across the URL

// 		// If the keyword has a trailing slash, treat it as a special keyword.
// 		// TODO: How do we want the user experience to be?
// 		// A. We show them exactly what they typed in the address bar the entire time, no redirecting with URL params. -OR-
// 		// B. We redirect here to the keyword handler. http://go2.local/?keyword=kwdname%2Fparam1%2Fparam2
// 		if strings.HasSuffix(p[0], "/") { // this is a dynamic/special keyword
// 			// If they entered a bare "kwdname/" and no params
// 			if len(p) > 1 { // they provided one or more params
// 				genParam = strings.Split(p[1], "/")
// 			}

// 			if p[1] == "" { //TODO there has to be a better way to do this..
// 				core.LogDebug.Printf("Dynamic redirect was requested, but no parameters were supplied, showing usage page...\n")
// 				http.Redirect(w, r, fmt.Sprintf("/.%s", p[0]), 302)
// 				return
// 			}

// 			safeKeyword, _ := core.MakeNewKeyword(p[0]) //TODO, err handling
// 			core.LogDebug.Printf("Special redirect requested. Keyword: '%s', Parameters: %s", p[0], genParam)
// 			HandleSpecial(w, r, safeKeyword, genParam)
// 			return
// 		}

// 		// look to see if we have a keyword for this.
// 		safeKeyword, _ = core.MakeNewKeyword(p[0])

// 		// If the keyword exists. follow the redirect behavior
// 		// otherwise, go to the dot page for the keyword
// 		if ll, exists := core.LinkDataBase.Lists[safeKeyword]; exists {
// 			ultimateURL := ll.GetRedirectURL()
// 			core.LogDebug.Printf("URL on keyword '%s' is '%s'\n", ll.Keyword, ultimateURL)
// 			// Links in regular lists can have variables.
// 			l := core.LinkDataBase.GetLink(-1, ultimateURL)
// 			if strings.ContainsAny(l.URL, "{}") {
// 				ultimateURL, _, err = RenderSpecial(r, []string{}, l, ll)
// 				if err != nil {
// 					// They didn't supply enough parameters.
// 					core.LogError.Println(err)
// 					routeSpecialListPage(w, r, safeKeyword, genParam, err.Error())
// 					return
// 				}
// 			}

// 			// A link can potentially have a 'burn after reading' Dtime. (the nil value of time.Time)
// 			// If it does, we are going to blow away the link in the database now.
// 			// They're still getting redirected below since we have the URL.
// 			if l.Dtime.Equal(core.BurnTime) {
// 				core.LogDebug.Printf("Burning link and redirecting user to: %s\n", ultimateURL)
// 				core.DestroyLink(l)
// 			} else {
// 				// Update Atime and register a 'click' on this specific keyword.
// 				l.Atime = time.Now().UTC()
// 				ll.Clicks++
// 				core.LogDebug.Printf("Redirecting user to: %s\n", ultimateURL)
// 			}

// 			// This unimportant set of lines is the central purpose of
// 			// this entire program - the redirect to their URL.
// 			http.Redirect(w, r, ultimateURL, 307)
// 			return
// 		}
// 		// Go to the list/dot page.
// 		listPage(w, r, safeKeyword)
// 		return

// 	} else if strings.HasPrefix(r.URL.Path, "/.") {
// 		// use case: They went straight to the dotpage in the address bar.
// 		kwdStr := strings.TrimPrefix(r.URL.Path, "/.") //lower this?
// 		safeKeyword, _ := core.MakeNewKeyword(kwdStr)
// 		listPage(w, r, safeKeyword)
// 		return

// 	} else if strings.HasPrefix(r.URL.Query().Get("keyword"), ".") {
// 		// Use case: They asked for a keyword with a leading dot (entered on index page form).
// 		s := fmt.Sprintf("%s/%s", core.ListenURL(), r.URL.Query().Get("keyword"))
// 		http.Redirect(w, r, s, 302)
// 		return

// 	} else if r.URL.Query().Get("keyword") != "" {
// 		// path == "/anything?keyword=thing"
// 		// two paths:
// 		// 1. it was a regular keyword, no arguments
// 		// 2. it was a special keyword with N arguments separated by slashes
// 		keyword := r.URL.Query().Get("keyword")

// 		// TODO: this is duplicated logic from above. needs a function
// 		m := strings.TrimPrefix(keyword, "/")
// 		p := strings.SplitAfterN(m, "/", 2) // ["kwdname/", "param1/param2"]
// 		core.LogDebug.Printf("Incoming URL escaped path items: %q\n", p)

// 		// These are things they typed in the go2/[] box at the top of the page.
// 		if strings.Contains(p[0], "/") {
// 			if len(p) > 1 {
// 				genParam = strings.Split(p[1], "/")
// 			}
// 			safeKeyword, _ := core.MakeNewKeyword(p[0])
// 			HandleSpecial(w, r, safeKeyword, genParam)
// 			return
// 		}

// 		safeKeyword, _ := core.MakeNewKeyword(keyword)
// 		listPage(w, r, safeKeyword)
// 		return
// 	} else if r.URL.Path == "/" {
// 		IndexPage(w, r)
// 		return
// 	} else {
// 		// log.Printf("No handler found for this URL! %s\n", r.URL)
// 		// log.Printf(FormatRequest(r))
// 		RouteNotFound(w, r)
// 	}
// }

func RouteLink(w http.ResponseWriter, r *http.Request) {
	// GET requests will have the editlink template returned.
	// POST requests will
	requestDump, err := httputil.DumpRequest(r, true)

	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(requestDump))
	switch r.Method {
	case http.MethodGet:
		/*
			If the link they GET exists, render editlink in edit mode.
			If the link they GET is new, render editlink in add mode.
		*/
		var model ModelIndex
		// TODO - keyword sanitization? How could this err?
		// This bit is only applicable if a new link is getting added to a new keyword.
		var kwdExists = false
		keyword, _ := core.MakeNewKeyword(r.URL.Query().Get("returnto"))
		if err != nil {
			// TODO: User needs to know why their keyword was bad.
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		if _, exists := core.LinkDataBase.Lists[keyword]; exists {
			// TODO: need to figure out how to handle this edge case. add called when it already exists
			kwdExists = true
		}

		var title string
		var kbe, le bool
		var lnk core.Link

		core.LogDebug.Printf("EDITLINK KEYWORD: %s\n", keyword)
		// They have a good keyword and provided a link URL to add. Do we have it already?
		id := core.NewLinkID(path.Base(r.URL.Path)) // This is a GET so the link ID is the first thing after the slash.
		var existingLink *core.Link

		if existingLink, exists := core.LinkDataBase.Links[id]; !exists {
			core.LogDebug.Printf("We don't have a link for the provided ID: %d\n", id)
			if existingLink == nil || id != 0 {
				model := ModelIndex{
					Title:              fmt.Sprintf("link ID %d does not exist", id),
					LinkDB:             *core.LinkDataBase,
					Keyword:            keyword,
					KeywordExists:      exists,
					KeywordBeingEdited: true,
					LinkExists:         exists,
					LinkBeingEdited:    existingLink,
					RedirectorName:     core.RedirectorName,
				}
				err := RenderTemplate(w, "404.gohtml", &model)
				if err != nil {
					core.LogError.Println(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}
		}

		existingLink = core.LinkDataBase.GetLink(id, "") // look up the existing link by ID
		if existingLink.ID > 0 {
			// if the link is already there, they can submit a modification to the link.
			// re-render the add page with all their form data and the existing link with the warning.
			title = "Edit Existing Link"
			kbe = false
			le = true
			lnk = *existingLink
		} else {
			// We render the edit page with the placeholder link to 'edit'. The template will not render this special linkZero.
			// They will get the same edit form as usual.
			title = "Add New Link"
			kbe = false
			le = false
			lnk = *core.LinkZero
		}

		overrides := make(map[string]string)

		ddd, err := url.ParseRequestURI(r.URL.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		core.LogDebug.Printf("Raw Query: %s", ddd)
		// Links can have cookie overrides.
		if len(r.Cookies()) > 0 {
			for _, cookie := range r.Cookies() {
				// names are pipe-delimted ex. 'keyword|name'
				nameFields := strings.Split(cookie.Name, "|")
				k := nameFields[0] // keyword
				p := nameFields[1] // pattern
				i := nameFields[2] // linkid

				// slashes cannot be in cookies so we use underscore. Change it back, if present.
				k = strings.ReplaceAll(k, "_", "/")

				kwd := core.Keyword(k)
				fmt.Printf("COOKIE SHIT: %s\n", cookie)
				if kwd == keyword && fmt.Sprint(lnk.ID) == i {
					// Cookie name indicated it was for this keyword AND link ID.
					overrides[p] = cookie.Value
				}
			}
		}

		order := 1 // indexed at 1 since we use param1, param2...
		for kee, val := range r.URL.Query() {
			if strings.HasPrefix(kee, "param") {
				overrides[fmt.Sprint(order)] = val[0]
				// overrides[order] = LinkVariablesMap{Pattern: val[0], Replacement: ""}
				order++
			}
		}

		// Does the URL they provided have any variable substitutions?
		core.LogDebug.Printf("COOKIES SEEN: %s\n", r.Cookies())
		core.LogDebug.Printf("overrides sent to template: %s\n", overrides)

		model = ModelIndex{
			Title:              title,
			LinkDB:             *core.LinkDataBase,
			Keyword:            keyword,
			KeywordExists:      kwdExists,
			KeywordBeingEdited: kbe,
			LinkExists:         le,
			LinkBeingEdited:    &lnk,
			RedirectorName:     core.RedirectorName,
			Overrides:          overrides,
		}

		err = RenderTemplate(w, "editlink.gohtml", &model)
		if err != nil {
			core.LogError.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	default:
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)

	}
	// TODO: arghh this can't be under the delete method? Maybe have this redirect using delete?
	// This is the removal of a link from a keyword.
	if r.PostFormValue("delete") == "decouple" {
		returnto := r.PostFormValue("returnto")
		keyword := core.Keyword(returnto)
		id := core.NewLinkID(r.PostFormValue("linkid"))
		core.LogDebug.Printf("We are decoupling link ID: %d from keyword: %s\n", id, keyword)

		if ll, exists := core.LinkDataBase.Lists[keyword]; exists {
			linkPtr := core.LinkDataBase.Links[id]
			core.LinkDataBase.Decouple(ll, linkPtr)
		}

		// send them back to the list page for that keyword.
		s := fmt.Sprintf("%s/.%s", core.ListenURL(), keyword)
		http.Redirect(w, r, s, 302)
	}
}

// If you go looking for something weird, you get moondog.
func RouteNotFound(w http.ResponseWriter, r *http.Request) {
	model := ModelIndex{
		Title:              "Whoopsies!",
		KeywordBeingEdited: true,
	}
	// TODO: maybe don't even use a template, just make a static 404 page here. no error checking needed
	err := RenderTemplate(w, "404.gohtml", &model)
	if err != nil {
		core.LogError.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func routeSpecialListPage(w http.ResponseWriter, r *http.Request, keyword core.Keyword, params []string, errMsg string) {
	var model ModelIndex
	var kwdExists = false

	core.LogDebug.Printf("list page hit for special/ keyword: '%s', parameters: '%s'\n", keyword, params)

	// check to see if this keyword exists.
	// model changes based on existence of a keyword input from the form
	if _, exists := core.LinkDataBase.Lists[keyword]; exists {
		kwdExists = true
	}

	// Regarding the params: It's going to come in as an array of strings. (might want to change to just string later - TODO)
	model = ModelIndex{
		Title:              "special list page",
		LinkDB:             *core.LinkDataBase,
		Keyword:            keyword,
		KeywordExists:      kwdExists,
		KeywordBeingEdited: true,
		LinkExists:         false,
		LinkBeingEdited:    core.LinkZero,
		RedirectorName:     core.RedirectorName,
		KeywordParams:      params,
		UsageLog:           core.LinkLog[keyword],
		ErrorMessage:       errMsg,
	}
	core.LogDebug.Printf("Usage strings: %s\n", model.UsageLog)
	err := RenderTemplate(w, "listspecial.gohtml", &model)
	if err != nil {
		core.LogError.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

/*
	page functions
*/

func IndexPage(w http.ResponseWriter, r *http.Request) {
	kwd := core.Keyword("")
	var err error
	// if err != nil {
	// 	core.LogError.Println(err)
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// }
	// model passed into index is the entire DB for now
	model := ModelIndex{
		Title:              "The GO2 Redirector",
		LinkDB:             *core.LinkDataBase,
		Keyword:            kwd,
		KeywordExists:      false,
		KeywordBeingEdited: false,
		LinkExists:         false,
		LinkBeingEdited:    core.LinkZero,
		RedirectorName:     core.RedirectorName,
	}

	err = RenderTemplate(w, "index.gohtml", &model)
	if err != nil {
		core.LogError.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// func listPage(w http.ResponseWriter, r *http.Request, keyword core.Keyword) {
// 	var model ModelIndex
// 	var kwdExists = false
// 	var err error

// 	// modify clicks and atime (do this on special too)

// 	//log.Printf("list page hit for keyword '%s'\n", keyword)
// 	for _, val := range core.LinkDataBase.Lists {
// 		core.Similar(string(keyword), string(val.Keyword))
// 	}
// 	// check to see if this keyword exists.
// 	// model changes based on existence of a keyword input from the form
// 	if k, exists := core.LinkDataBase.Lists[keyword]; exists {
// 		kwdExists = true
// 		// keyword is going to get a click, plus an Atime update
// 		k.Clicks++
// 	}

// 	var bEdited = true

// 	// What if the keyword they are entering is similar?
// 	// What if any links in their list are functionally identical to what others added?

// 	model = ModelIndex{
// 		Title:              "list page",
// 		LinkDB:             *core.LinkDataBase,
// 		Keyword:            keyword,
// 		KeywordExists:      kwdExists,
// 		KeywordBeingEdited: bEdited,
// 		LinkExists:         false,
// 		LinkBeingEdited:    core.LinkZero,
// 		RedirectorName:     core.RedirectorName,
// 		ErrorMessage:       "",
// 	}

// 	// regular lists go to list, special goes to the special page
// 	if keyword.IsSpecial() {
// 		model.KeywordBeingEdited = false // abusing this to get another boolean in the template
// 		model.UsageLog = core.LinkLog[keyword]
// 		err = RenderTemplate(w, "listspecial.gohtml", &model)
// 	} else {
// 		err = RenderTemplate(w, "list.gohtml", &model)
// 	}
// 	if err != nil {
// 		core.LogError.Println(err)
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 	}
// }

// encapsulate the logic for calculating a special redirect and performing that redirect
func HandleSpecial(w http.ResponseWriter, r *http.Request, keyword core.Keyword, params []string) {
	// param is whatever was following that trailing slash.
	// look up the keyword for this list (of one special link) (if not there, create it, return kw create page)
	// Get the URL for that link
	// run the substitution on the variable fields
	// return the redirect to the client with the string-replaced URL
	if ll, exists := core.LinkDataBase.Lists[keyword]; exists {
		core.LogDebug.Println("Special keyword does already exist")
		// we need the link object and the URL to render a special link.
		url := ll.GetRedirectURL()
		core.LogDebug.Printf("URL returned from GetRedirectURL: %s\n", url)
		l := core.LinkDataBase.GetLink(-1, url)
		core.LogDebug.Printf("COOKES SEEN: %s\n", r.Cookies())
		core.LogDebug.Printf("existing list of links: %v\n", ll)
		core.LogDebug.Printf("existing link: %v\n", l)

		// This will take the request and any cookies provided to return a real URL.
		ultimateURL, _, err := RenderSpecial(r, params, l, ll)
		if err != nil {
			// They didn't supply enough parameters.
			core.LogError.Println(err)
			routeSpecialListPage(w, r, keyword, params, err.Error())
			return
		}
		core.LogInfo.Printf("URL rendered: %s\n", ultimateURL)

		// register a 'click' on this specific keyword.
		ll.Clicks++
		l.Atime = time.Now().UTC()

		// log the current usage on this particular list of links
		usage := fmt.Sprintf("%s%s", keyword, strings.Join(params, "/"))
		core.LogDebug.Printf("Adding usage for special keyword: %s\n", usage)
		if ll.Logging {
			core.LinkLog[keyword] = core.RotateSlice(core.LinkLog[keyword], usage)
		}
		// core.LogDebug.Println("LinkLog Entries:")
		// for _, v := range LinkLog[keyword] {
		// 	core.LogDebug.Printf("\t%s\n", v)
		// }

		core.LogDebug.Printf("Special redirect rendered: %s\n", ultimateURL)
		core.LogInfo.Printf("Redirecting user to: %s\n", ultimateURL)
		http.Redirect(w, r, ultimateURL, 307)
		return
	}
	// send them off to the create page for a new keyword, as we'd do with a normal keyword
	core.LogDebug.Println("Special keyword does not already exist, sending to special list page...")

	routeSpecialListPage(w, r, keyword, params, "")
}

// This performs substitutions on the URL. It returns the URL string, whether it is complete, and an error value.
func RenderSpecial(r *http.Request, params []string, l *core.Link, ll *core.ListOfLinks) (string, bool, error) {

	// This is a "usage" URL or one with all the variable names.
	// http://www.example.com/{planet}/{moon}
	// If they had a special keyword of go2/info/mars/phobos, that would become:
	// http://www.example.com/mars/phobos
	// The usage employs words people can read to understand order of terms.

	/*
		another idea
		the only time you can define the patterns is when you initially enter the special in your browser.
		The entered params become the patters(keys) and the values are entered as blank or nil initially on the link.
		the user is redirected to the edit page where they enter all information.
		one of the things they submit are the replacements (values) for the patterns initially defined above
		The usage block remains read only because it can only be changed at dynamic link create time
		- maybe use italics and not curly braces in the usage to make more readable

	*/

	/*
		get the list this link belongs to.
		get the usage of the list.
		make a map of string:string
		For each usage term, in order:
		 - usage term becomes key
		 - first incoming param becomes value
		repeat until all usage terms are mapped to an incoming param

		for each map item:
		- string.Replace(unSubURL, key, value, -1) // replace all occurrences
		return final url


		build a link variables map and ask for replacements to be performed.

		cookies take pecedence over everything. You could override {2} if you like.



	*/
	var finalURL string
	var complete bool
	var err error

	core.LogDebug.Printf("%v", l)

	// break specials for now. We need to switch params to be positional and numbered starting at 1.
	// params is currently coming in at len==1 and empty string.
	// if len(params) >= 1 && params[0] != "" {
	// 	return "", fmt.Errorf("specials broken for a bit")
	// }

	if len(l.Lists) > 1 {
		// This is more or less an assertion of a condition that should never happen.
		core.LogDebug.Println("Special was a member of more than 1 list!!")
		core.PrintList(*ll)
		for i, v := range l.Lists {
			core.LogDebug.Printf("\tmembership %d: %s\n", i, v)
		}
	}

	inputLinkVariables := make(map[string]string)

	// params provided in the URL take precedence over everything else. If they provided positional
	// parameters, we are using those as substitutions in numbered positions (if this is a special keyword).

	for idx, val := range params {
		if val == "" {
			continue // empty string provided, ignore.
		}
		// The index is used here so they can sub {1}, {2}, and so on...
		inputLinkVariables[fmt.Sprint(idx+1)] = val
	}
	if len(inputLinkVariables) != 0 {
		finalURL, complete = core.GetURL(l.URL, inputLinkVariables)
		// return early if their params provided the remaining substitutions.
		if complete {
			return finalURL, complete, err
		}
	}
	core.LogDebug.Printf("Variables after parameters: %s\n", inputLinkVariables)

	// Check cookies first and perform replacements.
	// For any substitutions in the usage URL, we allow the user to override anything.
	if len(r.Cookies()) > 0 {
		// If the client has cookies, use them.
		core.LogDebug.Printf("User had custom cookies to override values: %s\n", r.Cookies())
		for _, c := range r.Cookies() {
			nameFragments := strings.Split(c.Name, "|")
			k := nameFragments[0]
			p := nameFragments[1]
			i := nameFragments[2]

			// slashes cannot be in cookies so we use underscore. Change it back, if present.
			k = strings.ReplaceAll(k, "_", "/")

			kwd := core.Keyword(k)
			if kwd == ll.Keyword && fmt.Sprint(l.ID) == i {
				inputLinkVariables[p] = c.Value
			}
		}
	}
	core.LogDebug.Printf("Variables after cookies: %s\n", inputLinkVariables)
	finalURL, complete = core.GetURL(l.URL, inputLinkVariables)
	// Return early if we find no more substitutions to be done.
	if complete {
		return finalURL, complete, err
	}

	// Finally, perform substitutions on the remaining URL using the defaults set on the link.
	finalURL, complete = core.GetURL(finalURL, l.LinkVariables)

	if !complete {
		err = fmt.Errorf("not all substitutions were completed on the URL")
	}

	return finalURL, complete, err
}

func RenderDotPage(r *http.Request) (string, ModelIndex, error) {
	// core.LogDebug.Println("this is the dotpage function")
	// right now, this is a wrapper for renderListPage just in case we want to ever
	// do something special here.
	return RenderListPage(r)
}

func RenderListPage(r *http.Request) (string, ModelIndex, error) {
	// core.LogDebug.Println("this is the listpage function")
	var tmpl string
	var model ModelIndex
	var err error

	var kwdExists = false

	pth, err := core.ParsePath(r.URL.Path)
	if err != nil {
		// early return, they botched the path
		return tmpl, model, err
	}

	inputKeyword := r.URL.Query().Get("keyword") // only set if they entered a keyword in the input box
	if inputKeyword != "" {
		core.LogDebug.Printf("User supplied keyword: %s\n", inputKeyword)
		k := strings.Split(inputKeyword, "/")[0]
		pth.Keyword, _ = core.MakeNewKeyword(k)
	}

	//log.Printf("list page hit for keyword '%s'\n", keyword)
	for _, val := range core.LinkDataBase.Lists {
		core.Similar(string(pth.Keyword), string(val.Keyword))
	}
	// check to see if this keyword exists.
	// model changes based on existence of a keyword input from the form
	if k, exists := core.LinkDataBase.Lists[pth.Keyword]; exists {
		kwdExists = true
		// keyword is going to get a click, plus an Atime update
		k.Clicks++
	}

	var bEdited = true

	// What if the keyword they are entering is similar?
	// What if any links in their list are functionally identical to what others added?

	model = ModelIndex{
		Title:              "list",
		LinkDB:             *core.LinkDataBase,
		Keyword:            pth.Keyword,
		KeywordExists:      kwdExists,
		KeywordBeingEdited: bEdited,
		LinkExists:         false,
		LinkBeingEdited:    core.LinkZero,
		RedirectorName:     core.RedirectorName,
		ErrorMessage:       "",
	}

	// regular lists go to list, special goes to the special page
	if pth.Keyword.IsSpecial() {
		model.KeywordBeingEdited = false // abusing this to get another boolean in the template
		model.UsageLog = core.LinkLog[pth.Keyword]
		// err = RenderTemplate(w, "listspecial.gohtml", &model)
		tmpl = "listspecial.gohtml"
	} else {
		// err = RenderTemplate(w, "list.gohtml", &model)
		tmpl = "list.gohtml"
	}

	return tmpl, model, err
}

// Used when a link is going to be edited. The link can be new or existing.
// They can also land on this page if their keyword had a . prefix or / suffix.
// /keyword/.absent || /keyword/absent/ || /keyword/absent == edit and couple new link tagged 'absent' on this list (note stripped)
// /keyword/.present || /keyword/present/ || /keyword/present == edit existing link on editlink page
//

func RenderLinkPage(r *http.Request) (string, ModelIndex, error) {
	var model ModelIndex
	var err error

	pth, err := core.ParsePath(r.URL.Path)
	inputKeyword := r.URL.Query().Get("keyword") // only set if they entered a keyword in the input box
	if inputKeyword != "" {
		core.LogDebug.Printf("User supplied keyword: %s\n", inputKeyword)
		pth.Keyword, _ = core.MakeNewKeyword(inputKeyword)
	}

	// Determine if the keyword already exists.
	_, kwdExists := core.LinkDataBase.Lists[pth.Keyword]

	url := r.URL.Query().Get("url")
	link := core.LinkDataBase.GetLink(-1, url)

	if link.ID > 0 {
		core.LogDebug.Printf("Link already exists. We are returning the existing link and modify page.%v", link)
		// if the link is already there, they can submit a modification to the link.
		// re-render the add page with all their form data and the existing link with the warning.
		model = ModelIndex{
			Title:              "Edit Existing Link",
			LinkDB:             *core.LinkDataBase,
			Keyword:            pth.Keyword,
			KeywordExists:      kwdExists,
			KeywordBeingEdited: false,
			LinkExists:         true,
			LinkBeingEdited:    link,
			RedirectorName:     core.RedirectorName,
		}
	} else {
		model = ModelIndex{
			Title:              "Add New Link",
			LinkDB:             *core.LinkDataBase,
			Keyword:            pth.Keyword,
			KeywordExists:      kwdExists,
			KeywordBeingEdited: false,
			LinkExists:         false,
			LinkBeingEdited:    core.LinkZero,
			RedirectorName:     core.RedirectorName,
		}
	}
	return "editlink.gohtml", model, err
}

// Take a template name, like help.gohtml, and render it down to the base template.
// Execute it, sending it to the client.
func RenderTemplate(w http.ResponseWriter, name string, data *ModelIndex) error {
	// Ensure the template exists in the map.
	core.LogDebug.Printf("Rendering template named: '%s'\n", name)
	tmpl, ok := Templates[name]
	if !ok {
		return fmt.Errorf("the template %s does not exist", name)
	}

	// Write into a temp buffer to check for errors.
	// This allows a proper header to be sent back, and for the error to be clearer.
	buf := Bufpool.Get()
	defer Bufpool.Put(buf)

	err := tmpl.ExecuteTemplate(buf, "base", data)
	if err != nil {
		return err
	}

	// Set the header and write the buffer to the http.ResponseWriter
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
	return nil
}
