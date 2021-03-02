# The Go2 Redirector

This is a mnemonic URL database, redirector, and search engine.

The primary function of this tool is to HTTP redirect users straight to a URL associated with the keyword they were accessing. URLs are difficult to remember and type, but they are easier to use if named concisely. Links are named or **tagged** and placed into lists identified by **keywords**. These keywords are well-known, intuitive strings the users come up with which describe the list or link being sought after.

The secondary function is to provide a place to curate lists of the most up-to-date links to a given subject.

If users want to search for information on Mars, they could type `go2 mars` in their browser's URL/search bar. This could redirect directly to a link or a list of links about the planet. Imagine there are multiple links or articles about Mars which exist on this list. How could a user get more information on the moon Phobos, going directly to it in one search? List curators could *tag* one of the links in the list with `phobos`. Now users can type `go2 mars/phobos` in their URL/search bar. That link has now become the canonical redirect for anyone looking for more information on this moon of Mars.

## Usage

The go2 redirector follows a Wikipedia-like model of community-driven addition, deletion, and curation of data. If users collectively agree on what keywords mean and in turn agree on what the list for that keyword should be, the result is the group's most accurate understanding of these mnemonic keywords at any given moment. The more people who use the redirector, the more editors you have keeping things up to date.

## Setup

### Configuration

To set up the application, an initial configuration and empty link database need to be created. To do this, run the install script from the command line.

`./install.sh`

This will place a `godb.json` file on disk in the project root directory, then it will write a generic configuration file `go2config.json` in the same directory. The default settings in the config file are enough to get started, but look it over to understand the settings available.

### Building

The redirector needs to be compiled as the second and final step of setup. A simple `go build` in the project root should yield an executable. Run that executable with no arguments to see the redirector start, listening on an ephemeral port.

### Browser Configuration

The go2redirector should be running on `localhost:8080` now. You can do directly to that, or to make things easier, you can configure your browser with a new search keyword like `go2`. Now your browser can be used to access the go2redirector like a search engine.
