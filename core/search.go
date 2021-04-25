package core

import (
	"fmt"
	"strings"
)

/*
Full text search through the link database
*/

// SearchResults holds lists and links matching search criteria
// provided by the user.
type SearchResults struct {
	Lists map[Keyword]*ListOfLinks
	Links map[int]*Link
}

// TODO: thread out the search on a copy of the db

// TODO: some searches are stronger matches. For example,
// an exact keyword match should be higher than a fuzzy match.

/*
Edit distance calculation is used on keywords and tags

Substring matches are used on links
*/
func (d *LinkDatabase) Search(term string) SearchResults {
	result := &SearchResults{
		Lists: make(map[Keyword]*ListOfLinks),
		Links: make(map[int]*Link),
		// Strength
	}
	for _, list := range d.Lists {
		// Matching keyword names from lists
		editDistanceKeyword := Similar(string(list.Keyword), term)
		// lev distance of 0 == exact match
		// TODO: this needs to be a higher-ordered search result
		if editDistanceKeyword == 0 {
			fmt.Printf("Keyword: %s - Exact match on keyword\n", list.Keyword)
			result.Lists[list.Keyword] = list
			continue // no need to look further on the list
		}

		// lev distance of < .3 == fuzzy match
		if editDistanceKeyword < 0.3 {
			fmt.Printf("Keyword: %s - Fuzzy match on keyword\n", list.Keyword)
			result.Lists[list.Keyword] = list
			continue
		}

		// Matching tag bindings within a list of links
		tags := []string{}
		for _, tb := range list.TagBindings {
			tags = append(tags, tb...)
		}
		for _, t := range tags {
			editDistanceTag := Similar(t, term)
			// This tag is close.
			if editDistanceTag < 0.3 {
				fmt.Printf("Keyword: %s, Tag: %s - Fuzzy match on tag\n", list.Keyword, t)
				result.Lists[list.Keyword] = list
				continue
			}
		}

		// Matching on a redirect URL from the list (changes based on list's behavior)
		// We do a substring match on this due to the length/complexity of the URLs
		url := list.GetRedirectURL()
		if strings.Contains(url, term) {
			fmt.Printf("Keyword: %s, RedirectURL: %s - Substring match on RedirectURL\n", list.Keyword, url)
			result.Lists[list.Keyword] = list
			continue
		}
	}

	/*
		Iteration through all links in the DB
		Precedence:
		1. Title
		2. URL
		3. memberships
		4. edits
	*/
	for _, link := range d.Links {
		// substring match on title
		if strings.Contains(link.Title, term) {
			fmt.Printf("Link: %d, Title: %s - Title substring match\n", link.ID, link.Title)
			result.Links[link.ID] = link
		}

		// TODO: fuzzy match on words in the title? IDK...

		// URL
		if strings.Contains(link.URL, term) {
			fmt.Printf("Link: %d, URL: %s - URL substring match\n", link.ID, link.URL)
			result.Links[link.ID] = link
		}

	}
	return *result
}
