package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cwbooth5/go2redirector/core"
)

// Incoming link struct containing POST form data
type apiLink struct {
	Keyword    core.Keyword `json:"keyword"`
	ID         int          `json:"id"`
	Title      string       `json:"title"`
	Tag        string       `json:"tag"`
	Url        string       `json:"url"`
	Expiretime string       `json:"expiretime"`
	Lists      []string     `json:"lists"`
}

/*
RouteAPI is the API handler used primarily to modify lists and links.

This is a single route that matches all the different request URLs. It's not
incredibly sophisticated, but it gets the job done.
*/

func RouteAPI(w http.ResponseWriter, r *http.Request) {
	/*
		/api/link - GET, POST
		/api/list - POST

		To differentiate between automated external users, who should get an API response from
		this program using its own API, we will use a hidden form value to identify requests
		needing a special response - like a redirect to a page as opposed to JSON and a 202.
		It will be a Form element key of "internal" with any non-null value.
	*/
	var err error

	// CORS header since browsers will check this for cross-origin accesses
	if _, exists := w.Header()["Access-Control-Allow-Origin"]; !exists {
		w.Header().Add("Access-Control-Allow-Origin", "*")
	}
	// Classification of API paths
	<-core.SYNC
	// link
	if strings.HasPrefix(r.URL.RequestURI(), "/api/link") {
		var internal bool // Is this going to get a page returned(internal == true) or a JSON response?
		var inboundLink *core.Link
		now := time.Now()
		switch r.Method {
		case "POST":
			r.ParseForm()
			for k, v := range r.Form {
				fmt.Printf("%s: %s\n", k, v)
				if k == "internal" && v[0] != "" {
					internal = true
				}
			}

			kw, _ := core.MakeNewKeyword(r.FormValue("returnto"))
			id := core.NewLinkID(r.PostFormValue("linkid")) // the link ID the form said we were editing: 0 == new
			outboundLink := apiLink{
				Keyword: kw,
				Title:   r.FormValue("title"),
				Tag:     strings.ToLower(r.FormValue("tag")), // note this is a space-delimited string of tags at this point
				Url:     core.SanitizeURL(r.FormValue("url")),
				ID:      id,
			}

			if id == 0 {
				// the working new link copy
				inboundLink, _ = core.MakeNewlink(outboundLink.Url, outboundLink.Title)
				// The only time expiretime can be set is on link creation.
				delta := r.FormValue("expiretime")
				if err != nil {
					msg := fmt.Sprintf("this duration string could not be parsed: '%s'", delta)
					core.LogError.Println(msg)
					http.Error(w, msg, http.StatusBadRequest)
				}
				outboundLink.Expiretime = delta
				// New links being created have an expire date set.
				// Existing links cannot have this value edited, so it only exists here.
				var exptime time.Time
				if delta == "burn" {
					// special case: burn after reading
					// We set the date to time.Time nil value to encode this.
					// Link pruning code should not remove this special case, even though it's well in the past.
					exptime = core.BurnTime
					core.LogDebug.Printf("burn time of %s set on link\n", exptime)
				} else {
					exptime, err = core.GetExpireTime(inboundLink.Ctime, delta)
					if err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest) // this would only fail if the template has bad values
					}
				}
				inboundLink.Dtime = exptime
				core.LogDebug.Printf("inbound link Dtime %s\n", inboundLink.Dtime)
			} else {
				// Check to see if we even have a link at this ID.
				if _, exists := core.LinkDataBase.Links[id]; !exists {
					w.WriteHeader(http.StatusNotFound)
					core.SYNC <- 1
					return
				}
				inboundLink = core.LinkDataBase.Links[id]
				inboundLink.Title = outboundLink.Title
				inboundLink.URL = outboundLink.Url
			}

			// Check for keyword in db
			ll, exists := core.LinkDataBase.Lists[outboundLink.Keyword]
			if !exists {
				// We need to create the keyword and link.
				ll = core.MakeNewList(outboundLink.Keyword)
				core.LogInfo.Printf("New keyword created: '%s'\n", outboundLink.Keyword)
			}

			// If they were decoupling the link, do it now and return
			if r.PostFormValue("delete") == "true" {
				core.LinkDataBase.Decouple(ll, inboundLink)
				// link edit metadata
				deleteEdit := core.EditRecord{EditDate: now, EditUser: core.ExtractUser(r), EditMsg: fmt.Sprintf("link decoupled: %s", inboundLink.URL)}
				core.RedirectorMetadata.ListEdits[ll.Keyword] = core.PrependEdit(core.RedirectorMetadata.ListEdits[ll.Keyword], &deleteEdit)

				core.LogInfo.Printf("user %s deleted link ID %d\n", deleteEdit.EditUser, inboundLink.ID)
				if internal {
					// The template called this, so 302 to the dotpage for this keyword.
					http.Redirect(w, r, fmt.Sprintf("/.%s", outboundLink.Keyword), http.StatusFound)
					core.SYNC <- 1
					return
				}
				// this isn't rendering a template, just an http response
				w.WriteHeader(http.StatusGone)
				core.SYNC <- 1
				return
			}

			// TAGS: The form input is a single field of space-delimited strings. Those are the tags.
			// The entire list of tags for the given link is overwritten by whatever they enter here.
			fmt.Printf("outbound link tag: %s\n", outboundLink.Tag)
			allTags := strings.Split(outboundLink.Tag, " ")
			fmt.Printf("alltags: %v\n", allTags)
			var newLinkEdit core.EditRecord

			if id == 0 {
				core.LogDebug.Printf("POST is adding a new link.")
				// We are editing an existing link.
				// Commit the inbound link, then grab the resulting new link ID we assigned
				lid, _ := core.LinkDataBase.CommitNewLink(inboundLink)
				// inbound link has its new linkid now.
				outboundLink.ID = lid
				ll.TagBindings[lid] = allTags
				newLinkEdit = core.EditRecord{EditDate: now, EditUser: core.ExtractUser(r), EditMsg: fmt.Sprintf("link created: %s", inboundLink.URL)}
				core.LogInfo.Printf("New link with ID %d was added to the DB by user %s.\n", lid, newLinkEdit.EditUser)
			} else {
				ll.TagBindings[id] = allTags
				newLinkEdit = core.EditRecord{EditDate: now, EditUser: core.ExtractUser(r), EditMsg: fmt.Sprintf("link modified: %s", inboundLink.URL)}
				core.LogInfo.Printf("Existing link with ID %d was modified by user %s.\n", id, newLinkEdit.EditUser)
			}
			// link edit metadata
			core.RedirectorMetadata.LinkEdits[outboundLink.ID] = core.PrependEdit(core.RedirectorMetadata.LinkEdits[outboundLink.ID], &newLinkEdit)

			// existing links will be coupled further down

			// timestamps
			inboundLink.Ctime = now
			inboundLink.Atime = now
			inboundLink.Mtime = now

			// list memberships
			var allMemberships = core.LinkDataBase.Links[inboundLink.ID].Lists
			for _, kw := range strings.Fields(r.PostFormValue("otherlists")) {
				kwd, _ := core.MakeNewKeyword(kw)
				// link edit metadata
				otherListEdit := core.EditRecord{EditDate: now, EditUser: core.ExtractUser(r), EditMsg: fmt.Sprintf("link coupled: %s, tags: %s", inboundLink.URL, allTags)}
				if ll, exists := core.LinkDataBase.Lists[kwd]; exists {
					core.LinkDataBase.Couple(ll, inboundLink)
					core.RedirectorMetadata.ListEdits[ll.Keyword] = core.PrependEdit(core.RedirectorMetadata.ListEdits[ll.Keyword], &otherListEdit)
				} else {
					// The other list they were trying to add to doesn't exist. No problem. Create it.
					newList := core.MakeNewList(kwd)
					core.LinkDataBase.Couple(newList, inboundLink)
					core.RedirectorMetadata.ListEdits[newList.Keyword] = core.PrependEdit(core.RedirectorMetadata.ListEdits[newList.Keyword], &otherListEdit)
				}
				allMemberships = append(allMemberships, kwd)
				core.LogDebug.Printf("Coupling link to otherlist '%s'", kwd)
			}

			/*
				link variables
				Users can name capture groups in a regex and map the captured strings to an internal variable lookup.

				This no longer supports user-local variables. This is being repurposed.
			*/
			formVariableValues := make(map[string]string) // making a new one forces a refresh of this data on the link
			for k := range r.Form {
				if strings.HasPrefix(k, "urlvar~") {
					name := strings.TrimPrefix(k, "urlvar~")
					// "name" is whatever they named the capture group. "k" is their input form value, which is the default action.
					// TODO: we need to check user input here
					formVariableValues[name] = r.FormValue(k)
				}
			}

			// list of links now needs the param/regex
			// Users are providing a regex here, validate it.
			inputRegex := r.FormValue("paramregexinput")
			_, err := regexp.Compile(inputRegex)
			if err != nil {
				// give both debug log and user feedback on the failed input
				core.LogError.Printf("regex compilation of '%s' failed! %s", inputRegex, err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				core.SYNC <- 1
				return
			}
			// only change these if validation passed
			inboundLink.LinkVariables = formVariableValues // They're all overwritten. Users can always change these defaults.
			// Initialize extractions struct to cover pre-extractions-feature lists of links
			if ll.Extractions == nil {
				ll.Extractions = make(map[int]core.ExtractionCapture)
			}
			ll.Extractions[inboundLink.ID] = core.ExtractionCapture{ExampleParam: r.FormValue("paraminput"), Regex: inputRegex}

			core.LinkDataBase.Couple(ll, inboundLink)

			// link edit metadata
			listEdit := core.EditRecord{EditDate: now, EditUser: core.ExtractUser(r), EditMsg: fmt.Sprintf("link coupled: %s, tags: %s", inboundLink.URL, allTags)}
			core.RedirectorMetadata.ListEdits[ll.Keyword] = core.PrependEdit(core.RedirectorMetadata.ListEdits[kw], &listEdit)

			if internal {
				// The template called this, so 302 to the dotpage for this keyword.
				http.Redirect(w, r, fmt.Sprintf("/.%s", outboundLink.Keyword), http.StatusFound)
				core.SYNC <- 1
				return
			}
			// this isn't rendering a template, just an http response
			// 202/Accepted
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(outboundLink)
		case "GET":
			// 200 OK or 404
			// This is here for the js requesting already-set link variables so it can populate input.value fields.
			core.LogDebug.Println("this is a GET to the link API")

			// incoming request will contain linkid
			linkid, err := strconv.Atoi(r.URL.Query().Get("linkid"))
			core.LogDebug.Printf("Incoming link id: %d\n", linkid)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				core.SYNC <- 1
				return
			}

			data, err := json.Marshal(core.LinkDataBase.Links[linkid])
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				core.SYNC <- 1
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(data)
		}

	} else if r.URL.RequestURI() == "/api/behavior/" {
		var internal bool
		switch r.Method {
		case "POST":
			r.ParseForm()
			core.LogDebug.Println("Incoming POST fields/values:")
			for k, v := range r.Form {
				core.LogDebug.Printf("%s: %s\n", k, v)
				if k == "internal" && v[0] != "" {
					internal = true
				}
			}

			kw, _ := core.MakeNewKeyword(r.FormValue("keyword"))
			requestedBehavior, err := strconv.Atoi(r.FormValue("behavior"))
			if err != nil { // err would indicate the form is broken or manual bad input from clients
				msg := "Behavior entered was malformed"
				core.LogError.Print(msg)
				http.Error(w, msg, http.StatusBadRequest)
				core.SYNC <- 1
				return
			}

			previousBehavior := core.LinkDataBase.Lists[kw].Behavior
			core.LinkDataBase.Lists[kw].Behavior = requestedBehavior
			core.LogInfo.Printf("Behavior on keyword '%s' changed to %d by user %s\n", kw, requestedBehavior, core.ExtractUser(r))

			if previousBehavior != requestedBehavior { // handle the case where they just clicked the button with no changes
				// edit metadata on the list
				editmsg := fmt.Sprintf("behavior changed from '%s' to '%s'", core.GetPrettyBehaviorString(previousBehavior), core.GetPrettyBehaviorString(requestedBehavior))
				core.RedirectorMetadata.ListEdits[kw] = core.PrependEdit(core.RedirectorMetadata.ListEdits[kw], &core.EditRecord{EditDate: time.Now(), EditUser: core.ExtractUser(r), EditMsg: editmsg})
			}

			if internal {
				// The template called this, so 302 to the dotpage for this keyword.
				http.Redirect(w, r, fmt.Sprintf("/.%s", kw), http.StatusFound)
				core.SYNC <- 1
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotImplemented)
		case "GET":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotImplemented)
		}
	} else if r.URL.RequestURI() == "/api/keywords" {
		// Keywords API, used initially just to get the data for the search box. proof-of-concept
		switch r.Method {
		case "POST":
			core.LogDebug.Printf("post to keyword API, TODO")
		case "GET":
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "max-age=60") // cache locally to speed things up
			w.WriteHeader(http.StatusFound)

			data, err := json.Marshal(core.LinkDataBase.Lists)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				core.SYNC <- 1
				return
			}
			w.Write(data)
		}
	} else if strings.HasPrefix(r.URL.RequestURI(), "/api/variables/strings") {
		/*
			strings api
			This one is much simpler than maps.
			**namespaces are included for forward-compatibility when those are added
			"/api/variables/strings/{stringname}"
			GET: Get a string back as JSON by name
			DELETE: Delete a string by name
			POST: create new string, name and value fields required, namespace global by default
			{
				name: "mystring",
				value: "myvalue",
				namespace: "global"
			}
		*/
		var strName string
		type stringsPayload struct {
			Namespace, Name, Value string
		}
		split := strings.Split(r.RequestURI, "/")
		if len(split) < 5 {
			http.Error(w, "ERROR: string name required in URL", http.StatusBadRequest)
			core.SYNC <- 1
			return
		}
		strName = strings.Trim(split[4], "\r\n")

		switch r.Method {
		case "DELETE":
			// The delete operation is on the entire string/value variable.
			if len(split) == 5 {
				core.LogInfo.Printf("String %s is being deleted by user %s\n", strName, core.ExtractUser(r))
				delete(core.LinkDataBase.Variables.Strings, strName)
			}

			data, err := json.Marshal("deleted")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				core.SYNC <- 1
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
		case "POST":
			// creation, updates
			pl := stringsPayload{}
			body, err := io.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil { // problem reading request body
				http.Error(w, err.Error(), http.StatusBadRequest)
				core.SYNC <- 1
				return
			}
			err = json.Unmarshal(body, &pl)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				core.SYNC <- 1
				return
			}
			// input sanitization
			pl.Name = strings.Trim(pl.Name, "\r\n")
			pl.Value = strings.Trim(pl.Value, "\r\n")
			core.CreateStringVar(pl.Name, pl.Value)
			core.LogInfo.Printf("String %s is being created by user %s\n", pl.Name, core.ExtractUser(r))

			// bullshit reply for testing, TODO change this to something sensible
			data, err := json.Marshal(core.LinkDataBase.Variables.Maps[strName])
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				core.SYNC <- 1
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
		}
	} else if strings.HasPrefix(r.URL.RequestURI(), "/api/variables/maps") {
		/*
			**namespaces are included for forward-compatibility when those are added
			"/api/variables/maps/{mapname}"
			GET: Get a map back as JSON by name
			DELETE: Delete a map by name
			POST: create new map, values field is mandatory, array can be empty, namespace global by default
			{
				values: "phx:phoenix-32\niad:virginia-12",
				namespace: "global"
			}

			PUT: Update N elements of an existing map
			every value needs to be included, extra values can be added
			{
				values: [
					phx:phoenix-1234,
					iad:virginia-99,
					icn:seoul-88
				],
				namespace: global
			}

			"/api/variables/maps/{mapname}/{key}"
			GET: Get a value from a map name and key
			the return from the GET is going to be identical to what you POST
			POST: Add a key/value to a map
			{
				value: "some-value"
			}
			DELETE: Delete a key/value entry from a map

		*/
		type mapsPayload struct {
			Namespace, Values string
		}
		core.LogDebug.Println("maps route hit")
		split := strings.Split(r.RequestURI, "/")
		if len(split) < 5 {
			http.Error(w, "ERROR: map name required in URL", http.StatusBadRequest)
			core.SYNC <- 1
			return
		}
		mapName := split[4]

		switch r.Method {
		case "DELETE":
			if len(split) == 5 {
				core.LogInfo.Printf("Map %s is being deleted by user %s\n", mapName, core.ExtractUser(r))
				// The first case is they are deleting an entire map by name.
				delete(core.LinkDataBase.Variables.Maps, mapName)
			} else if len(split) == 6 {
				// The second case is they are deleting a specific key:value pair from a map.
				// These delete requests will have a request body indicating what is being removed.
				core.LogInfo.Println("key is being deleted from map")
				keyName := split[len(split)-1]
				delete(core.LinkDataBase.Variables.Maps[mapName], keyName)
			}
			data, err := json.Marshal("deleted")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				core.SYNC <- 1
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)

		case "POST":
			pl := mapsPayload{}
			body, err := io.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil { // problem reading request body
				http.Error(w, err.Error(), http.StatusBadRequest)
				core.SYNC <- 1
				return
			}
			err = json.Unmarshal(body, &pl)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				core.SYNC <- 1
				return
			}

			// temporary holding structure for incoming data, just in case errors on input
			tempInput := make(map[string]string)

			for _, item := range strings.Split(pl.Values, "\n") {
				// now we have key:value
				pair := strings.SplitN(item, ":", 2)
				// key is the first element, value is the second
				if len(pair) <= 1 { // they didn't provide a separator
					http.Error(w, "no separator was specified", http.StatusBadRequest)
					core.SYNC <- 1
					return
				}
				// fmt.Printf("map  - %s\n", pair) // TODO: space crashes this
				tempInput[pair[0]] = pair[1]
			}

			// This destroys the entire map and creates it new with incoming values.
			core.LogInfo.Printf("Map %s is being created/modified by user %s\n", mapName, core.ExtractUser(r))
			core.LinkDataBase.Variables.Maps[mapName] = tempInput

			// bullshit reply for testing
			data, err := json.Marshal(core.LinkDataBase.Variables.Maps[mapName])
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				core.SYNC <- 1
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
		}
	}
	core.SYNC <- 1
}
