# D2 to Site

A simple tool built to scan a file system, compile any D2 files there, and combine them with the Markdown files to generate a simple static website.

## Assumptions

This project started as a need to host a bunch of D2 architecture and user flow images. Placing them in PDFs was painful, as they would span multiple pages uncomfortably. Placing them on Confluence and other wiki-style tools was better, but keeping them up to date was a challenge. Instead, the desire would be to have a repo that, every time a commit was made, "something" would compile the images and generate a site for reference. This tool came out of the desire for that "something."

To get started, you will need a filesystem. Let's start with something like this:

```bash
./src
-- index.md
-- logo.png
-- ./level1
-- -- index.md
-- -- user_flow.d2
-- -- admin_flow.d2
```

By running the tool with the proper flags (see the next section), you should end up with something like:

```bash
./build
-- index.html
-- logo.png
-- ./level1
-- -- index.html
-- -- user_flow.svg
-- -- admin_flow.svg
```

What happens is that the tool walks the filesystem target, compiles every .d2 file to an svg, compiles every Markdown file to an HTML file with the specified template, and copies any other file directly. In the Markdown files, {{filename}} will take the output `filename.d2` -> `filename.svg` and convert it into an `<img>` tag in the HTML.

The Markdown files can begin with YAML-based metadata. The currently accepted metadata are `title`, `summary`, and `tags`. For example:

```markdown
---
title: Sample
tags:
  - tag1
  - tag2
summary: A basic description, free of HTML, used for search.
---

# A sample page

{{sample}}

```

This will generate the title and tags as meta data, render the content as Markdown, and result in `{{sample}}` turning into `<img src='/sample.svg' alt='diagram' />`

## Running the Tool

```bash
$ ./d2tosite -h
NAME:
   d2tosite - Use it

USAGE:
    [global options] command [command options] [arguments...]

DESCRIPTION:
   A simple CLI that traverses a directory and generates a basic HTML site from Markdown and D2 files

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --d2-theme value          the D2 theme ID to use (default: 1)
   --d2-output-type value    the output type for the d2 compiler; can only be svg at this time and is otherwise ignored (default: "svg")
   --input-directory value   the directory to read from and walk to build the site (default: "./src")
   --output-directory value  the output directory to publish the site to (default: "./build")
   --page-template value     the template to use for each page (default: "./default_templates/page.html")
   --index-template value    the template to use for the content of the diagram index (default: "./default_templates/diagram_index.html")
   --tag-template value      the template to use for each tag page content (default: "./default_templates/tag.html")
   --clean                   if true, removes the target build directory prior to build
   --continue-errors         if true, continues to build site after parsing and compiling errors are found
   --help, -h                show help
```

## Templates

Templates are Go-style HTML templates that are applied to the compiled Markdown files. For an example, see the `./cmd/default_templates/leaf.html` file. For a code-based definition, the following data is populated and passed to the templates for use as variables:

```go
type SiteData struct {
   Title       string
   Content     template.HTML
   Links       []d2s.LeafData
   Tags        []string
   SiteTags    map[string][]d2s.LeafData
   AllDiagrams map[string]*d2s.LeafData
}
```

The `LeafData` struct looks like:

```go
type LeafData struct {
   Title    string
   FileName string
   Tags     []string
   SiteTags map[string][]LeafData // needed for the nav
   Links    []LeafData            // needed for the nav
   Diagrams []string              // needed for the index
   Content  template.HTML         // used for converting to an html template
   Summary  string                // used for search displays, found in the meta
}
```

Currently, the main template variables that are supported are:

- .Title -> the title of the page
- .Links -> a map of titles to leaf information, which importantly includes the .Filepath.
- .Tags -> a slice of tags on the site. Special pages will be generated that shows each HTML file that contains a specific tag
- .Content -> them HTML-ized content to display on a page

## Search

Search is a local index of pages keyed by their path with the content, title, tags, and summary indexed. It is then parsed through `lunr.js`. If the query param of `search` is present, the search content is displayed. Note that this is purely client-side and driven in the `leaf.html` template, so if you provide your own template, you will need to ensure you either support search as laid out or remove it from your site.

## Roadmap

These are not in any specific order. If you are interested in working on, send a message or open an issue.

[ ] GitHub actions to build binaries

[ ] Dockerize the binaries for easier use

[ ] Expose more D2 functionality

[ ] Code cleanup to move beyond POC

[ ] UI testing for HTML generation

[ ] Logging and verbosity

[ ] Add static page with output of the test data on main push

[ ] Better error handling if templates are missing

[ ] Config file support
