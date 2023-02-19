package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/cwbooth5/go2redirector/core"
)

/*
	HTTP handling and routes
*/

/*
The string variables handler
*/
func RouteStrings(w http.ResponseWriter, r *http.Request) {
	if core.ExtractUser(r) == "" { // not logged in
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	core.LogDebug.Println("strings route hit")
	model := ModelIndex{
		Title:          "String Variables",
		LinkDB:         core.LinkDataBase,
		RedirectorName: core.RedirectorName,
		ActiveUser:     core.ExtractUser(r),
	}
	var varName, varValue string
	sPath := strings.Split(r.URL.Path, "/")
	varName = sPath[len(sPath)-1]
	core.LogDebug.Printf("Incoming string name: %s\n", varName)

	switch r.Method {
	case http.MethodGet:
		// Pull up a blank create page (default)
		// or pre-fill fields to edit a variable
		varValue = core.LinkDataBase.Variables.Strings[varName]
		model.Variable = []string{varName, varValue}
	}

	err := RenderTemplate(w, "strings.gohtml", &model)
	if err != nil {
		core.LogError.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

/*
The maps page shows the input boxes for a new map or in the case of an
existing map, those same edit boxes populated with values from that map.
*/
func RouteMaps(w http.ResponseWriter, r *http.Request) {
	if core.ExtractUser(r) == "" { // not logged in
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	core.LogDebug.Println("maps route hit")
	model := ModelIndex{
		Title:          "Map Variables",
		LinkDB:         core.LinkDataBase,
		RedirectorName: core.RedirectorName,
		ActiveUser:     core.ExtractUser(r),
	}
	sPath := strings.Split(r.URL.Path, "/")
	varName := sPath[len(sPath)-1]
	core.LogDebug.Printf("Incoming map name: %s\n", varName)

	switch r.Method {
	case http.MethodGet:
		// Pull up a blank create page (default)
		// or pre-fill fields to edit a variable
		// they get here if hitting the edit button for a map or by going to _maps_
		valuesAndNewlines := ""
		values := []string{}
		for key, val := range core.LinkDataBase.Variables.Maps[varName] {
			values = append(values, strings.Join([]string{key, val}, ":"))
		}
		valuesAndNewlines = strings.Join(values, "\n")
		// Overload the Variable field in the model.
		// The name field will be filled out with the first string in the array, value is second
		model.Variable = []string{varName, valuesAndNewlines}
	}

	err := RenderTemplate(w, "maps.gohtml", &model)
	if err != nil {
		core.LogError.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

/*
The link check feature: for debugging precisely what happens behind the scenes

This particular function is here solely to catch requests using the "/check"
prefix on their queries. This redirects the user to the same input they
provided with an added "?check=true" URL parameter so we can handle checks
consistently in one place.
*/
func RouteCheck(w http.ResponseWriter, r *http.Request) {
	// run through all redirect logic and show a list of what we would do
	var requestURL string
	if core.ExternalPort == 0 {
		requestURL = fmt.Sprintf("%s://%s/%s?check=true", core.ExternalProto, core.ExternalAddress, strings.TrimPrefix(r.RequestURI, "/check/"))
	} else {
		requestURL = fmt.Sprintf("%s://%s:%d/%s?check=true", core.ExternalProto, core.ExternalAddress, core.ExternalPort, strings.TrimPrefix(r.RequestURI, "/check/"))
	}
	http.Redirect(w, r, requestURL, http.StatusFound)
	core.LogInfo.Println("Check mode activated for a keyword")
}

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

func RouteLink(w http.ResponseWriter, r *http.Request) {
	// GET requests will have the editlink template returned.

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
		keyword, err := core.MakeNewKeyword(r.URL.Query().Get("returnto"))
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

		// They have a good keyword and provided a link URL to add. Do we have it already?
		id := core.NewLinkID(path.Base(r.URL.Path)) // This is a GET so the link ID is the first thing after the slash.
		var existingLink *core.Link

		if existingLink, exists := core.LinkDataBase.Links[id]; !exists {
			core.LogDebug.Printf("We don't have a link for the provided ID: %d\n", id)
			if existingLink == nil || id != 0 {
				model := ModelIndex{
					Title:              fmt.Sprintf("link ID %d does not exist", id),
					LinkDB:             core.LinkDataBase,
					Keyword:            keyword,
					KeywordExists:      exists,
					KeywordBeingEdited: true,
					LinkExists:         exists,
					LinkBeingEdited:    existingLink,
					RedirectorName:     core.RedirectorName,
					ActiveUser:         core.ExtractUser(r),
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
			le = true
			lnk = *existingLink
		} else {
			// We render the edit page with the placeholder link to 'edit'. The template will not render this special linkZero.
			// They will get the same edit form as usual.
			title = "Add New Link"
			lnk = *core.LinkZero
		}

		overrides := make(map[string]string)

		ddd, err := url.ParseRequestURI(r.URL.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		core.LogDebug.Printf("Raw Query: %s", ddd)

		order := 1 // indexed at 1 since we use param1, param2...
		for kee, val := range r.URL.Query() {
			if strings.HasPrefix(kee, "param") {
				overrides[fmt.Sprint(order)] = val[0]
				order++
			}
		}

		// Does the URL they provided have any variable substitutions?
		core.LogDebug.Printf("COOKIES SEEN: %s\n", r.Cookies())
		core.LogDebug.Printf("overrides sent to template: %s\n", overrides)

		model = ModelIndex{
			Title:              title,
			LinkDB:             core.LinkDataBase,
			Keyword:            keyword,
			KeywordExists:      kwdExists,
			KeywordBeingEdited: kbe,
			LinkExists:         le,
			LinkBeingEdited:    &lnk,
			RedirectorName:     core.RedirectorName,
			Overrides:          overrides,
			ActiveUser:         core.ExtractUser(r),
			Variable:           []string{core.ExternalProto, core.ExternalAddress, fmt.Sprintf("%d", core.ExternalPort)},
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
		http.Redirect(w, r, s, http.StatusFound)
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

func RouteLogin(w http.ResponseWriter, r *http.Request) {
	// Right now, this only supports POST requests to change their cookie.
	if r.Method == "POST" {
		// Login interface
		// We get their name, set their cookie, then redirect back to Referer
		// To log them out, we set their cookie TTL to expire.

		core.LogDebug.Println("post to _login_")
		// get their name from the POST form
		r.ParseForm()
		var theirName string
		theirPostName := r.PostFormValue("loginname")
		theirCookieName := core.ExtractUser(r)
		if theirPostName != "" {
			theirName = theirPostName
		} else {
			theirName = theirCookieName
		}
		ttl := time.Unix(0, 0) // a sensible default: expire immediately

		if r.PostFormValue("delete") == "true" {
			// They are logging out. We will set their cookie to expire now.
			core.LogInfo.Printf("User '%s' is logging out\n", theirName)
			// ttl will just be the default value of "immediately"
		} else if theirName != "" {
			day, _ := time.ParseDuration("24h")
			ttl = time.Now().Add(day)
			core.LogInfo.Printf("User %s is logging in\n", theirName)
		}

		cookie := http.Cookie{
			Name:    "redirectorlogin",
			Value:   theirName,
			Expires: ttl,
		}

		// We always set a cookie. TTL determines login/logout
		http.SetCookie(w, &cookie)
		http.Redirect(w, r, fmt.Sprintf("%s/", r.Referer()), http.StatusFound)
	}
}

/*
	page functions
*/

func IndexPage(w http.ResponseWriter, r *http.Request) {
	kwd := core.Keyword("")
	var err error

	activeUser := core.ExtractUser(r)

	// model passed into index is the entire DB for now
	model := ModelIndex{
		Title:              "The GO2 Redirector",
		LinkDB:             core.LinkDataBase,
		Keyword:            kwd,
		KeywordExists:      false,
		KeywordBeingEdited: false,
		LinkExists:         false,
		LinkBeingEdited:    core.LinkZero,
		RedirectorName:     core.RedirectorName,
		ActiveUser:         activeUser,
	}

	err = RenderTemplate(w, "index.gohtml", &model)
	if err != nil {
		core.LogError.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// This performs substitutions on the URL. It returns the URL string, whether it is complete, and an error value.
func RenderSpecial(params []string, l *core.Link, ll *core.ListOfLinks, check chan<- string) (string, bool, error) {

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
	inputLinkVariables := make(map[string]string)

	// This is more or less an assertion of a condition that should never happen.
	if len(l.Lists) > 1 {
		core.LogDebug.Println("Special was a member of more than 1 list!!")
		core.PrintList(*ll)
		for i, v := range l.Lists {
			core.LogDebug.Printf("\tmembership %d: %s\n", i, v)
		}
	}

	/*
		This range shouldn't be needed if we are only allowing a single param.
		It will stay here to get {1} now and be compatible with {2} if we ever decide to support that.
		We can't remove this {1} feature now that we've released with it. It's probably fine.
	*/
	for idx, val := range params {
		if val == "" {
			continue // empty string provided, ignore.
		}
		// The index is used here so they can sub {1}, {2}, and so on...
		// Note this key is a string of a number. kinda dumb.
		inputLinkVariables[fmt.Sprint(idx+1)] = val
	}
	core.LogDebug.Printf("Variables before parameters: %s\n", inputLinkVariables)

	// look on the list at this linkid, get the extraction regex.
	// We need to check it here to make sure it compiles. This is also checked when they submit it.
	extractionRegex, err := regexp.Compile(ll.Extractions[l.ID].Regex)
	if err != nil {
		fmt.Printf("regex compilation failed! %s", err) //TODO: proper internal error
		return finalURL, complete, err
	}
	// run regex using the entire parameter {1} as a string.
	matches := extractionRegex.FindStringSubmatch(inputLinkVariables["1"])
	check <- fmt.Sprintf("Extraction regex being used: %s", extractionRegex)
	check <- fmt.Sprintf("Matches made on parameter '%s': %s", inputLinkVariables["1"], matches)

	/* We need the named group's name and the value we captured.
	The name shows us were we are going to sub data into the URL string.
	The captured value gives us something to use in the variable lookup.
	*/
	res := make(map[string]string)
	for i, name := range matches {
		if i == 0 {
			continue // first element is not useful, it's the matched string itself
		}
		res[extractionRegex.SubexpNames()[i]] = name
	}

	check <- fmt.Sprintf("named parameters and extracted values: %s", res)
	linkDefaultVars := l.LinkVariables

	finalURL, complete, err = core.GetURL(l.URL, res, inputLinkVariables, linkDefaultVars, check)
	core.LogDebug.Printf("link: %s\n", finalURL)

	if !complete {
		err = fmt.Errorf("not all substitutions were completed on the URL")
		check <- err.Error()
	}

	return finalURL, complete, err
}

/*
Right now, this is a wrapper for renderListPage just in case we want to ever
do something special here.
*/
func RenderDotPage(r *http.Request) (string, ModelIndex, error) {
	return RenderListPage(r)
}

/*
Return everything needed to get to the list page
*/
func RenderListPage(r *http.Request) (string, ModelIndex, error) {
	var tmpl string
	var model ModelIndex
	var err error
	var kwdExists = false

	pth, err := core.ParsePath(r.URL.Path)
	inputKeyword := r.URL.Query().Get("keyword") // only set if they entered a keyword in the input box

	if err != nil && inputKeyword == "" {
		core.LogError.Println(err)
		model.ErrorMessage = err.Error()
		return "list.gohtml", model, err
	}

	if inputKeyword != "" {
		// core.LogDebug.Printf("User supplied keyword: %s\n", inputKeyword)
		k := strings.Split(inputKeyword, "/")[0]
		pth.Keyword, _ = core.MakeNewKeyword(k)
	}

	for kwd := range core.SearchKeywordsData {
		core.Similar(string(pth.Keyword), kwd)
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

	activeUser := core.ExtractUser(r)

	model = ModelIndex{
		Title:              "list",
		LinkDB:             core.LinkDataBase,
		Keyword:            pth.Keyword,
		KeywordExists:      kwdExists,
		KeywordBeingEdited: bEdited,
		LinkExists:         false,
		LinkBeingEdited:    core.LinkZero,
		RedirectorName:     core.RedirectorName,
		ErrorMessage:       "",
		ActiveUser:         activeUser,
	}

	// regular lists go to list, special goes to the special page
	if pth.Keyword.IsSpecial() {
		model.KeywordBeingEdited = false // abusing this to get another boolean in the template
		model.UsageLog = core.LinkLog[pth.Keyword]
		tmpl = "listspecial.gohtml"
	} else {
		tmpl = "list.gohtml"
	}

	return tmpl, model, err
}

// Used when a link is going to be edited. The link can be new or existing.
// They can also land on this page if their keyword had a . prefix or / suffix.
// /keyword/.absent || /keyword/absent/ || /keyword/absent == edit and couple new link tagged 'absent' on this list (note stripped)
// /keyword/.present || /keyword/present/ || /keyword/present == edit existing link on editlink page

func RenderLinkPage(r *http.Request) (string, ModelIndex, error) {
	var model ModelIndex
	var err error

	pth, err := core.ParsePath(r.URL.Path)
	inputKeyword := r.URL.Query().Get("keyword") // only set if they entered a keyword in the input box
	if inputKeyword != "" {
		core.LogDebug.Printf("User supplied keyword: %s\n", inputKeyword)
		// TODO: the "keyword" here could be something like "keyword/.tag"

		inputSplit := strings.Split(inputKeyword, "/")
		pth.Keyword, err = core.MakeNewKeyword(inputSplit[0])
	}

	// Determine if the keyword already exists.
	_, kwdExists := core.LinkDataBase.Lists[pth.Keyword]

	url := r.URL.Query().Get("url")
	link := core.LinkDataBase.GetLink(-1, url)

	activeUser := core.ExtractUser(r)

	if link.ID > 0 {
		core.LogDebug.Printf("Link already exists. We are returning the existing link and modify page.%v", link)
		// if the link is already there, they can submit a modification to the link.
		// re-render the add page with all their form data and the existing link with the warning.
		// Variable field is being abused to sneak in the external addr/port for the js to redirect browsers to the api properly
		model = ModelIndex{
			Title:              "Edit Existing Link",
			LinkDB:             core.LinkDataBase,
			Keyword:            pth.Keyword,
			KeywordExists:      kwdExists,
			KeywordBeingEdited: false,
			LinkExists:         true,
			LinkBeingEdited:    link,
			RedirectorName:     core.RedirectorName,
			ActiveUser:         activeUser,
			Variable:           []string{core.ExternalProto, core.ExternalAddress, fmt.Sprintf("%d", core.ExternalPort)},
		}
	} else {
		model = ModelIndex{
			Title:              "Add New Link",
			LinkDB:             core.LinkDataBase,
			Keyword:            pth.Keyword,
			KeywordExists:      kwdExists,
			KeywordBeingEdited: false,
			LinkExists:         false,
			LinkBeingEdited:    core.LinkZero,
			RedirectorName:     core.RedirectorName,
			ActiveUser:         activeUser,
			Variable:           []string{core.ExternalProto, core.ExternalAddress, fmt.Sprintf("%d", core.ExternalPort)},
		}
	}
	return "editlink.gohtml", model, err
}

// Take a template name, like help.gohtml, and render it down to the base template.
// Execute it, sending it to the client.
func RenderTemplate(w http.ResponseWriter, name string, data *ModelIndex) error {
	// Ensure the template exists in the map.
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

/*
Using a template file, substitute in some config variables to form
the opensearch.xml so it functions with this redirector instance.

Browsers will request this if the HTML says we have it. The rendered
version of this file will contain all the needed substitutions, as
dictated from the config file.
*/
func RenderOpenSearch(templatePath string, outPath string) error {
	var u string
	var err error
	if core.ExternalPort == 0 {
		u = fmt.Sprintf("%s://%s", core.ExternalProto, core.ExternalAddress)
	} else {
		u = fmt.Sprintf("%s://%s:%d", core.ExternalProto, core.ExternalAddress, core.ExternalPort)
	}

	searchURL := fmt.Sprintf("%s/.{searchTerms}", u)
	suggestURL := fmt.Sprintf("%s/_suggest_/?q={searchTerms}", u)
	baseURL := u

	t, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}

	defer f.Close()

	config := map[string]string{
		"searchURL":      searchURL,
		"suggestURL":     suggestURL,
		"baseURL":        baseURL,
		"redirectorName": core.RedirectorName,
	}

	err = t.Execute(f, config)
	if err != nil {
		return err
	}
	return err
}

/*
RouteSuggest implements part of the opensearch protocol, specifically the search
suggestions extension.

Every time the user types a character in their URL bar, the browser is going to
send a request to this route. The route responds with an array of information
the browser uses to populate the seach suggestions.
*/
func RouteSuggest(w http.ResponseWriter, r *http.Request) {
	var suggestionPrefix string
	// the characters they typed in the search bar
	if r.URL.RawQuery == "" || len(r.URL.Query()["q"]) < 1 {
		http.Error(w, "query required (use ?q=searchterm)", http.StatusBadRequest)
		return
	}
	suggestionPrefix = r.URL.Query()["q"][0]

	searchTerms := core.SearchDB(suggestionPrefix, 15)
	core.LogDebug.Printf("suggest prefix: %s, terms: %s\n", suggestionPrefix, searchTerms)
	// Other browsers *could* use these arrays, they're part of the standard/protocol.
	reply := []interface{}{suggestionPrefix, searchTerms}

	// check before sending back
	data, err := json.Marshal(reply)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
