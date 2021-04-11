package core

import (
	"encoding/json"
	"io/ioutil"
	"time"
)

/*
List and link metadata

We maintain a data structure with two halves, one for lists and the other for links.
The more interesting edits are coupling/decoupling, tag modifications, and new/destroyed things.

The data is nice to have if you want to persist it along with a corresponding link database.
For lists, the keyword is used to locate edit records.
For links, the id is used to locate edit records.
*/

// EditRecord holds what amounts to a line in a log file.
// It contains the date things were done, the person who did it, and the message.
type EditRecord struct {
	EditDate time.Time `json:"edit_date"`
	EditMsg  string    `json:"edit_msg"`
	EditUser string    `json:"edit_user"`
}

// Metadata holds all list/link edits.
// It can be marshaled to JSON and saved to disk.
type Metadata struct {
	ListEdits map[Keyword][]*EditRecord
	LinkEdits map[int][]*EditRecord
}

// This is called to initialize the in-memory metadata for all lists and links.
func MakeNewMetadata() *Metadata {
	// go through all lists and links, creating their corresponding metadata with empty values.
	var m *Metadata
	m = new(Metadata)
	m.ListEdits = make(map[Keyword][]*EditRecord)
	m.LinkEdits = make(map[int][]*EditRecord)
	return m
}

func (m *Metadata) Import(f string) (Metadata, error) {
	var tempdata Metadata
	var err error
	data, err := ioutil.ReadFile(f)
	if err != nil {
		LogError.Printf("No edit metadata file '%s' was found\n", f)
		return tempdata, err
	}

	err = json.Unmarshal(data, &tempdata)
	if err != nil {
		LogError.Printf("json parsing error: %s", err)
		return tempdata, err
	}
	return tempdata, err
}

func (m *Metadata) Export(f string) error {
	file, err := json.Marshal(*m)
	if err != nil {
		LogError.Println("JSON marshal error:", err)
		return err
	}
	err = ioutil.WriteFile(f, file, 0644)
	if err != nil {
		LogError.Fatal(err)
	}
	LogInfo.Printf("Link metadata exported to %s.\n", f)
	return err
}
