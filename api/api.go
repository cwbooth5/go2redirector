package api

import (
	"encoding/json"
	"fmt"
	"net/http"
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

// RouteAPI is the API handler used primarily to modify lists and links.
func RouteAPI(w http.ResponseWriter, r *http.Request) {
	/*
		/api/link - GET, POST
		/api/list - POST

		To differentiate between automated external users, who should get an API response from
		this program using its own API, we will use a hidden form value to identify requests
		needing a special response - like a redirect to a page as opposed to JSON and a 202.
		It will be a Form element key of "internal" with any non-null value.
	*/
	// core.LogDebug.Printf("API request URL: %s\n", r.URL.Path)
	// requestDump, err := httputil.DumpRequest(r, true)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Println(string(requestDump))
	var err error

	// Classification of API paths

	// link
	if r.URL.RequestURI() == "/api/link/" {
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

				if internal {
					// The template called this, so 302 to the dotpage for this keyword.
					http.Redirect(w, r, fmt.Sprintf("/.%s", outboundLink.Keyword), 302)
					return
				}
				// this isn't rendering a template, just an http response
				// 410/Gone
				w.WriteHeader(http.StatusGone)
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
				core.LogInfo.Printf("New link with ID %d was added to the DB.\n", lid)
				ll.TagBindings[lid] = allTags
				newLinkEdit = core.EditRecord{EditDate: now, EditUser: core.ExtractUser(r), EditMsg: fmt.Sprintf("link created: %s", inboundLink.URL)}
			} else {
				ll.TagBindings[id] = allTags
				newLinkEdit = core.EditRecord{EditDate: now, EditUser: core.ExtractUser(r), EditMsg: fmt.Sprintf("link modified: %s", inboundLink.URL)}
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

			// link variables
			formVariableValues := make(map[string]string)
			for k := range r.Form {
				if strings.HasPrefix(k, "urlvar~") {
					name := strings.TrimPrefix(k, "urlvar~")
					formVariableValues[name] = r.FormValue(k)
				}
			}
			inboundLink.LinkVariables = formVariableValues // They're all overwritten. Users can always change these defaults.

			core.LinkDataBase.Couple(ll, inboundLink)

			// link edit metadata
			listEdit := core.EditRecord{EditDate: now, EditUser: core.ExtractUser(r), EditMsg: fmt.Sprintf("link coupled: %s, tags: %s", inboundLink.URL, allTags)}
			core.RedirectorMetadata.ListEdits[ll.Keyword] = core.PrependEdit(core.RedirectorMetadata.ListEdits[kw], &listEdit)

			if internal {
				// The template called this, so 302 to the dotpage for this keyword.
				http.Redirect(w, r, fmt.Sprintf("/.%s", outboundLink.Keyword), 302)
				return
			}
			// this isn't rendering a template, just an http response
			// 202/Accepted
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(outboundLink)
		case "GET":
			// 200 OK or 404
			core.LogDebug.Println("this is a GET..")

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
				return
			}

			previousBehavior := core.LinkDataBase.Lists[kw].Behavior
			core.LinkDataBase.Lists[kw].Behavior = requestedBehavior
			core.LogDebug.Printf("Behavior on keyword '%s' changed to %d\n", kw, requestedBehavior)

			if previousBehavior != requestedBehavior { // handle the case where they just clicked the button with no changes
				// edit metadata on the list
				editmsg := fmt.Sprintf("behavior changed from '%s' to '%s'", core.GetPrettyBehaviorString(previousBehavior), core.GetPrettyBehaviorString(requestedBehavior))
				core.RedirectorMetadata.ListEdits[kw] = core.PrependEdit(core.RedirectorMetadata.ListEdits[kw], &core.EditRecord{EditDate: time.Now(), EditUser: core.ExtractUser(r), EditMsg: editmsg})
			}

			if internal {
				// The template called this, so 302 to the dotpage for this keyword.
				http.Redirect(w, r, fmt.Sprintf("/.%s", kw), http.StatusFound)
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
			w.Header().Set("Cache-Control", "max-age=60")
			w.WriteHeader(http.StatusFound)

			data, err := json.Marshal(core.LinkDataBase.Lists)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			//core.LogDebug.Println("/api/keywords route hit")
			w.Write(data)
		}
	}
}
