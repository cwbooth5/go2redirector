package core

/*
Users can set variables globally which can be looked up and used in URL replacement operations.

This data is held in memory while running and exported to disk when shutting down.
These variables sit in the link database alongside the lists and links.
Uses is a structure holding variable names and arrays of link IDs employing those variables.
*/
type UserVariables struct {
	Strings map[string]string            `json:"strings"`
	Maps    map[string]map[string]string `json:"maps"`
	Uses    map[string][]*Link           `json:"uses"`
}

func CreateStringVar(n, v string) {
	// name and value come in from the user input
	// TODO: what's the max length we will accept for name and value?
	// LinkDataBase.Variables.Strings = make(map[string]string)
	if LinkDataBase.Variables.Strings == nil {
		LinkDataBase.Variables.Strings = make(map[string]string)
		LogDebug.Println("String variables initialized")
	}
	LinkDataBase.Variables.Strings[n] = v
}

func CreateMapVar(n string) {
	// temp := make(map[string]string)
	if LinkDataBase.Variables.Maps == nil {
		LinkDataBase.Variables.Maps = make(map[string]map[string]string)
		LogDebug.Println("Map variables initialized")
	}
	LogDebug.Printf("initializing mapvar named '%s'\n", n)
	LinkDataBase.Variables.Maps[n] = make(map[string]string)
}
