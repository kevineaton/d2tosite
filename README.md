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
   d2tosite

USAGE:
    [global options] command [command options] [arguments...]

DESCRIPTION:
   A simple CLI that traverses a directory and generates a basic HTML site from Markdown and D2 files

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config value            a config file that can be used to configure the build; if other flags are sent as well, they will override the file
   --d2-theme value          the D2 theme ID to use (default: 1)
   --input-directory value   the directory to read from and walk to build the site (default: "./src")
   --output-directory value  the output directory to publish the site to (default: "./build")
   --page-template value     the template to use for each page; if not provided, it will used the embedded template file at compile time
   --index-template value    the template to use for the content of the diagram index; if not provided, it will used the embedded template file at compile time
   --tag-template value      the template to use for each tag page content; if not provided, it will used the embedded template file at compile time
   --clean                   if true, removes the target build directory prior to build
   --continue-errors         if true, continues to build site after parsing and compiling errors are found
   --help, -h                show help
```

## Configuration

You may choose to pass in a `.json` or `.yml` file as a configuration option. The extension will determine the parsing. Although there are many different naming conventions available, snake_case was chosen for simplicity. Effectively, the keys are just the binary command line options with `-` changed to `_`.

## Templates

Templates are Go-style HTML templates that are applied to the compiled Markdown files. For an example, see the `./cmd/default_templates/page.html` file. Each template will be built with the `LeafData` filled out for that leaf AFTER all of the filesystem is walked. This is to ensure that each page can generate a navigation panel and search.

Each template may be specified at the command line as an argument that is a relative-path. On start, the files will be checked to see if they exist. If they do not, embedded templates shipped with the binary at compile-time will be used. You can find a copy of them in the repo in `cmd/default_templates/*.html`.

## Search

Search is a local index of pages keyed by their path with the content, title, tags, and summary indexed. It is then parsed through `lunr.js`. If the query param of `search` is present, the search content is displayed. Note that this is purely client-side and driven in the `page.html` template, so if you provide your own template, you will need to ensure you either support search as laid out or remove it from your site.

## Roadmap

These are not in any specific order. If you are interested in working on, send a message or open an issue.

[ ] GitHub actions to build binaries

[ ] Dockerize the binaries for easier use

[ ] Expose more D2 functionality

[ ] Code cleanup to move beyond POC

[ ] UI testing for HTML generation

[ ] Logging and verbosity

[ ] Add static page with output of the test data on main push

[ ] Add output types of site (default), single-page (all on the same page, in order), or JSON (for consumption elsewhere)
