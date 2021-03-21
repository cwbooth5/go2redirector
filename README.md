# The Go2 Redirector

[![Build Status](https://github.com/cwbooth5/go2redirector/actions/workflows/build.yml/badge.svg)](https://github.com/cwbooth5/go2redirector/actions/workflows/build.yml)

This is a mnemonic URL database, redirector, and search engine.

The primary function of this tool is to HTTP redirect users straight to a URL associated with the keyword they were accessing. URLs are difficult to remember and type, but they are easier to use if named concisely. Links are named or **tagged** and placed into lists identified by **keywords**. These keywords are well-known, intuitive strings the users come up with which describe the list or link being sought after.

The secondary function is to provide a place to curate lists of the most up-to-date links to a given subject.

If users want to search for information on Mars, they could type `go2 mars` in their browser's URL/search bar. This could redirect directly to a link or a list of links about the planet. Imagine there are multiple links or articles about Mars which exist on this list. How could a user get more information on the moon Phobos, going directly to it in one search? List curators could *tag* one of the links in the list with `phobos`. Now users can type `go2 mars/phobos` in their URL/search bar. That link has now become the canonical redirect for anyone looking for more information on this moon of Mars.

## Try out a [DEMO](https://go2redirector.us.to/)

## Usage

The go2 redirector follows a Wikipedia-like model of community-driven addition, deletion, and curation of data. If users collectively agree on what keywords mean and in turn agree on what the list for that keyword should be, the result is the group's most accurate understanding of these mnemonic keywords at any given moment. The more people who use the redirector, the more editors you have keeping things up to date.

## Setup

### Configuration

To set up the application, an initial configuration and empty link database need to be created. To do this, run the install script from the command line.

`./install.sh`

This will place a `godb.json` file on disk in the project root directory, then it will write a generic configuration file `go2config.json` in the same directory. The default settings in the config file are enough to get started, but look it over to understand the settings available.

### Building

The redirector needs to be compiled as the second and final step of setup. A simple `go build` in the project root should yield an executable. Run that executable with no arguments to see the redirector start, listening on an ephemeral port.

### Build and Run in a Container

A multi-stage `Dockerfile` is included here to ease the process of building the binary and providing a runnable container. The initial building container is huge and is thrown away in favor of a smaller alpine linux-based runtime container. There are three elements to building and running the container: building, persisting the link database, and running the container.

Building the container can be done with: `docker build -t go2redirector .`

If you would like to inspect the build container itself to check out/debug the build environment, you can access it by targeting the build container by name.

`docker build --target builder -t go2build .`

You can create a volume locally for outside-the-container persistent space for `godb.json`. This will allow your container to user the same godb every time it executes. Leave this procedure out if you do not wish to save the database between container runs.

1. Create a local storage volume using `docker volume create go2`
2. Inspect your created volume to verify it is created. `docker volume inspect go2`

Now run the container, using the volume. This will run the container in daemon mode and remove it when it stops.

`docker run --rm -p 8080:8080 -d -v go2:/home/gouser/data go2redirector`

Note that you don't have use a volume like this. A bind mount to another existing directory would work as well.

The redirector is going to be listening on `0.0.0.0:8080` inside the container, as opposed to the default of `127.0.0.1` from go2config.json (the default).

To see the logs from the container, they're all redirected to stdout, so you can do a `docker logs <name of running container>`

### Browser Configuration

The go2redirector should be running on `localhost:8080` now. You can do directly to that, or to make things easier, you can configure your browser with a new search keyword like `go2`.

Each browser has a slightly different configuration procedure to enable keyword search engines.

#### Firefox

1. Open `localhost:8080` (or whatever URL you run your redirector on) in Firefox.
2. Right-click on the search box next to **go2** on the upper left of the page.
3. Select "Add Keyword for this Search"
4. Set a keyword of `go2`

#### Chrome

1. Open settings.
2. Navigate to "Manage search engines".
3. Click the "Add' button.
4. For the **Search engine** field, choose a descriptive name.
5. for the **Keyword** field, use `go2`.
6. For the URL, enter `http://localhost:8080/?keyword=%s`

Now your browser can be used to access the go2redirector like a search engine. Set the keyword to `go2` and use the search box

`go2 wiki/es`

If your browser redirected to the Spanish version of Wikipedia, you're all set.

## Design

### Terminology

* **Keywords** uniquely identify a list of links. It's a list name you can pronounce.
* **Tags** are names for a link and are local to the list/keyword that link resides in.
* **Parameters** are positional arguments users can supply to subsititute directly into a link's URL.
* **Dotpage** is the 'edit' page for a given list and can be accessed by adding a dot `.` prefix to any keyword.
* **Fields** are the elements between each forward slash `/` in a redirect string user's enter. For example, `go2 planets/mars/weather` would have fields "planets" (the keyword), "mars" (the tag), and "weather" the parameter.

### Keywords/Lists

Curation of a list of links starts with selecting an intuitive keyword. This is the name people will remember this list of links by. Think about the keyword and how general it is. Does it apply to other potential lists? If so, maybe come up with a more specific keyword name or combine the two lists.

### Tags

A tag is a name for a link within a list of links. The tag is the second (optional) field a user types in a go2 redirect. If you have a list of moons of Mars, you might tag one with "phobos" and another with "deimos", resulting in a redirect like `go2 mars/phobos` to go straight to whatever link describes that moon. Tags are optional. If a second field is specified by the user, the redirector attempts to locate a tag in the list by that name. If it fails to find one, the second field is treated as a subsititution parameter for the link.

### Links

If a link is added with a URL we already have a link for under some other keyword, we allow you to create a completely new link because you might have another title and different keyword associations. If you attempt to add a duplicate link, it will show you other keywords already using the link when looking at the dotpage.

### The Search Box

The input field in the upper left of the index page is the main entry point for the application. This is the field users can type keyword/tag/parameter combos into to get to a redirect or create a new ones.

### "Burn after Reading" and Link Lifetime

Links can have a date set which specifies the link's lifetime in the link database. By default, links never expire. Users can input various link lifetimes. The most unique of all selected link lifetimes is "burn after reading" which is exactly what it sounds like. The application will destroy the link after a single person has used it as a redirect. This is useful for links you'll only use or share once. You should select a reasonable lifetime for a link if it's not going to be eternal. This is the passive form of curation in the application, removing links as their expiration dates come up.

### Editing and the Dotpage

To force access to the list page of a keyword (regardless of list behavior), you simply prefix that keyword with a period or suffix it with a forward slash. Doing so will render the list page where the links can be changed around or tagged.

### User-Provided Parameters

The links in a list can have a `{1}` placed anywhere in the URL to serve as a substitution string for a single positional parameter supplied by the user. Right now, we only support one parameter, but this could change if there is a compelling reason for two or more. In the previous version of the redirector, these types of links with substitutions were called "special" links and they used `{*}` as a substitution string. For example, the keyword `go2 planets` can have a few links tagged with various planet names. Each link URL can contain the subsititution string {1}.

For the user input of `go2 planets/mars/weather` the go2redirector would locate the `planets` keyword, look up the link tagged with `mars`, get its URL of `www.nasa.gov/planets/mars/{1}.php`, and perform a substitution to `www.nasa.gov/planets/mars/weather.php`. Finally, the user would be redirected to that URL.

## Contributing

I need all the help I can get making my novice level golang look nicer. There are new features we want to add and not enough people to do it. If you'd like to contribute, just fork the repository and submit a PR! File any enhancement requests or bugs on the issue tracker here in the go2redirector project.

## Credits

The original redirector [f5go](https://github.com/f5devcentral/f5go) was designed by Saul Pwanson, with assistance from Bryce Bockman and Treebird(tm).
