package core

import (
	"fmt"
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
	EditDate time.Time
	EditMsg  string
	EditUser string
}

// Metadata holds all list/link edits.
// It can be marshaled to JSON and saved to disk.
type Metadata struct {
	ListEdits map[Keyword][]*EditRecord
	LinkEdits map[int][]*EditRecord
}

// This is called to initialize the in-memory metadata for all lists and links.
func MakeNewMetadata(d *LinkDatabase) *Metadata {
	// go through all lists and links, creating their corresponding metadata with empty values.
	var m *Metadata
	m = new(Metadata)
	m.ListEdits = make(map[Keyword][]*EditRecord)
	m.LinkEdits = make(map[int][]*EditRecord)
	fmt.Printf("Metadata created: %s\n", m)

	return m
}

func LoadMetaData(filepath string) error {
	return nil
}
