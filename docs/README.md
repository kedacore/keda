# Syna Start

This is a sample project that can be used to jump start your Syna project. It uses Syna theme and Hugo with sample files that introduce two simple pages, one for landing and one for the about page.

## Prerequisites
- [Install Git](https://git-scm.com/downloads).
- [Install Go](https://golang.org/doc/install).
- [Install Hugo](https://gohugo.io/getting-started/installing/). Depending on your system, this might require Scoop, Choclatey, or other software.

## Installation

To start using this starter you need to clone or download this repository and update it's git submodules (Syna theme is added as a submodule).

```
git clone https://git.okkur.org/syna-start project-name && cd project-name
git submodule init
git submodule update
```

## Usage

To start your website run the following commands:

**Development**:
```
$ hugo server -D # This command starts the Hugo server and watches the site directory for changes.
```

**Production**:
```
$ hugo # This command generates the static website in the public/ directory. If you do not have a site, then it gives errors about missing layout files.
```

> Prerequisites: Go, Hugo

## Directory Structure

We're using the standard directory structure using content pages.

```
├─ content/
|  └ _global/ # All global fragments are located in this directory
|  └ _index/ # Landing page is in this directory and it's url is changed to **/**.
|  └ about/ # About page
├ layouts/ # You can add extra layout files here. A sample custom fragment is available as a sample.
├ static/ # Your static files are in this directory.
├ themes/ # Hugo uses this directory as a default to look for themes. Syna theme is a git submodule available in this directory.
├ .gitignore
├ .gitmodules
├ config.toml # Hugo config file containing general settings and menu configs.
```

For storing images in the static directory, note that Syna fragments look for
images in their own fragment directory, page directory and `static/images`
directory. Read our [image fallthrough documentation](https://syna.okkur.org/docs/image-fallthrough/) for more info.

Further details read our [full documentation](https://syna.okkur.org/docs).

## First Steps

Open index.md and type. The changes are visible almost immediately at http://localhost:1313/.
