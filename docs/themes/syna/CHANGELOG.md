<!--
# v0.16.0
_2018_
  - [Downloads for v0.16.0](https://github.com/okkur/syna/releases/tag/v0.16.0)
  - [Changelog since v0.15.0](#changes-since-v015)

## Documentation for v0.16
[Documentation](https://syna.okkur.org/docs) *Documentation defaults to latest release*

## Changes since v0.15.0

## Fixes since v0.15.0

---

-->

# v0.15.2
_2019-10-16_
  - [Downloads for v0.15.2](https://github.com/okkur/syna/releases/tag/v0.15.2)
  - [Changelog since v0.15.1](#changes-since-v0151)

## Documentation for v0.15.2
[Documentation](https://syna.okkur.org/docs) *Documentation defaults to latest release*

## Changes since v0.15.1
  - Add support for sort order in the list fragment #583
  - Add minHeight param to hero fragment #571
  - Bump Hugo's minimum required version to v0.58 #588

## Fixes since v0.15.1
  - Fix fragments not showing in the 404 page #575
  - Fix global fragments with subitems showing warning #582
  - Fix toc list displaying pagination in docs
  - Fix extra padding on code blocks #603
  - Filter special pages out of list fragment's displayed pages #595
  - Fix children deptch in toc #596
  - Fix bad split in event params (pubsub) #594

---

# v0.15.1
_2019-05-27_
  - [Downloads for v0.15.1](https://github.com/okkur/syna/releases/tag/v0.15.1)
  - [Changelog since v0.15.0](#changes-since-v015)

## Documentation for v0.15.1
[Documentation](https://syna.okkur.org/docs) *Documentation defaults to latest release*

## Changes since v0.15.0

## Fixes since v0.15.0
  - Add the new auto generated files (have been missed in a previous commit)
  - Move syna-grid.css from user side to theme side (bad approach, removes it from user side at least)
  - Fix the .Dir deprecation warning, I have no idea how these are popping up

---
# v0.15.0
_2019-05_
  - [Downloads for v0.15.0](https://github.com/okkur/syna/releases/tag/v0.15.0)
  - [Changelog since v0.14.0](#changes-since-v014)

## Documentation for v0.15
[Documentation](https://syna.okkur.org/docs) *Documentation defaults to latest release*

## Changes since v0.14.0
  - BREAKING: `item`: Item URL configuration is now `item_url` instead of `url`
  - BREAKING: Custom JS and CSS within config.toml are replaced by `config` fragment
  - Huge documentation overhaul
  - Accesibility improvements via `alt` and `sr-only` tags
  - Code snippets and inline code are more readable (invert background)
  - Contrast improvements for text, buttons and backgrounds
  - Add Title_align for better control of headers
  - Enable and document usage of FontAwesome Pro
  - Upgrade FontAwesome
  - Show scrollspy and active page to navbar and sidebar based navbars
  - Resize images automatically with the exception of `static/` based ones
  - Use favicon.svg and favicon.png, if defined
  - Add slot feature to combine various fragments
  - Add support for social media cards
  - Ability to create documentation via `content` and `list` using sidebar slots
  - `list`: Add collapsible items
  - `nav`: Add sticky option
  - `hero`: Ability to customize particle.js
  - `pricing`: Add plan:change event
  - New: `events`: Client side pubsub like event framework including triggering events via URL
  - `events`: Base64 obfuscated event URLs
  - New: `stripe`: Add payments fragment based on stripe
  - `stripe`: Prevent double charges by disabling button
  - `stripe`: Add multiple price option
  - `stripe`: Add custom price option
  - New: `graph`: Add chart.js fragment
  - New: `TOC`: Add table of contents fragment
  - New: `config`: New config fragment to inject custom assets such as `meta`, `link` or `script`
  - New: `header`: Add separate header fragment

## Fixes since v0.14.0
  - Fix consistency of header margins
  - Retriggering an event will clear fields
  - `stripe`: Fix multiple Stripe fragments on a single page
  - `contact`: Fix Recaptcha positioning
  - `contact`: Make contact form async even within Firefox
  - `table`: Optimize darker background colors
  - `content`: Sidebar margin fixes
  - `404`: Fix layout issues
  - `editor`: Fix editor not always loading
  - `react-portal`: Fix portal not always loading

---

# v0.14.0
_2018-10-15_
  - [Downloads for v0.14.0](https://github.com/okkur/syna/releases/tag/v0.14.0)
  - [Changelog since v0.13.0](#changes-since-v013)

## Documentation for v0.14
[Documentation](/tree/v0.14/docs)

## Changes since v0.13.0
  - BREAKING: Fragment lookup order was broken in v0.13. Please check your fragments are overwritten as expected.
  - BREAKING: `item`: Icons should be placed under `asset` table
  - BREAKING: `items`: Icon for each column should be placed under `asset` table
  - BREAKING: `logos`: Deprecated `logos` fragment in favor of `items` fragment
  - BREAKING: `header`: `align` variable is changed to `header_align`
  - Some colors have slightly changed. The change is a major internal overall. It's not considered a breaking change but please review your design.
  - `nav`: Breadcrumb support added using `breadcrumb: false/true` and `breadcrumb_level: 1`
  - `table`/`item`: Extract table into its own helper
  - `list`: Display date and category for pages
  - `list`: Pagination is now supported
  - `list`: Change page title size based on visibility of summary
  - `list`: Add ability to use a custom summary using `.Params.summary`
  - `content`: `.Params.summary` added with markdown support
  - `content`: Display date
  - `content`: Fix max-width of images in the content
  - `pricing`: Add warning message in case there are no items available
  - `global`: All fragments now support alignment of the title and subtitle
  - `faq`: Add `faq` fragment to list questions and answers
  - Header (title and subtitle) code extracted into helper partial
  - Text-color code extracted into helper partial
  - Theme colors are now customizable through `config.toml`
  - Hugo resource pipelines now builds sass files instead of Webpack
  - Make build command minify by default

## Fixes since v0.13.0
  - `contact`: Fix contact form not submitting data to Netlify

---

# v0.13.0
_2018-09-10_
  - [Downloads for v0.13.0](https://github.com/okkur/syna/releases/tag/v0.13.0)
  - [Changelog since v0.12.0](#changes-since-v012)

## Documentation for v0.13.0
[Documentation](/tree/v0.13.0/docs)

## Changes since v0.12.0
  - BREAKING: Page declaration from `_index/index.md` needs to be moved to `content/_index.md`
  - BREAKING: Page declaration for `_index/index.md` needs to be set to `headless = true`
  - BREAKING: Image declaration changed from `[branding]`, `image = ""` to `[asset]` using consistent asset declaration
  - BREAKING: Image declaration changed from `header = ""` to `[header]` using consistent asset declaration
  - BREAKING: Image declaration changed from `[[logos]]` to `[[assets]]` using consistent asset declaration
  - Hugo resource pipelines now builds sass files instead of Webpack
  - Theme colors are now customizable through `config.toml`
  - New: `list` fragment for section pages and page lists
  - New: `pricing` fragment to show pricing and features
  - New: `react-portal` fragment to embed react based features
  - New: `editor` fragment to create an editor from JSON schemas
  - New: `search` fragment enabling search as part of a page
  - `navbar`: Support search in navbar
  - `content`: Optionally show date and category in content fragment
  - `footer`: `asset.title` is moved to `.Params.title`
  - `portfolio`: Support image fallthrough
  - `404`: Add ability to change and resize image
  - `member`, `items`, `portfolio`: Display error messages, when no item is configured
  - Refactor fragment lookup strategy
  - Restructure exampleSite (showcase fragments and use as actual page for Syna)
  - Add `/dev/` section to exampleSite for testing and development
  - Extract code into helper partials

## Fixes since v0.12.0
  - `footer`: Subtitle is now linked when there is no logo
  - Use relLangURL for all links
  - Fix recaptcha support for Netlify contact form
  - Optimize asset sizes in exampleSite

---

# v0.12.0
_2018-08-06_
  - [Downloads for v0.12.0](https://github.com/okkur/syna/releases/tag/v0.12.0)
  - [Changelog since v0.11.0](#changes-since-v011)

## Documentation for v0.12.0
[Documentation](/tree/v0.12.0/docs)
[Getting started](/tree/v0.12.0/docs#using-starter)

## Changes since v0.11.0
  - BREAKING: `content-single` and `content-split` merged into `content` fragment
  - BREAKING: Moving to `_index` and `_global` as special directories and headless bundles
  - BREAKING: Subpath handling made consistent with Hugo
  - New: `header` fragment for easier section bundling and linking
  - New: `portfolio` fragment to showcase projects etc.
  - New: Categories for `content` fragment
  - `member`: Company affiliation for single member mode
  - `member`: Redesign single member mode
  - Getting started guide
  - Update documentation
  - Bundle JS files and register them within each fragment
  - `404`: Refactor 404 to be fragment based

## Fixes since v0.11.0
  - Improve naming consistency
  - Cleanup bootstrap files
  - Add attribution for inspiration
  - `table`: Align table cells using `align` variable
  - `items`: Remove icon, if not set
  - `item/table`: Fix icon + url
  - `item`: Fix align = center

---

# v0.11.0

> Note: This version includes major breaking changes.
> With v0.11.0 most breaking changes are already settled.
> We expect a few more breaking changes in the coming releases, but nothing major.
> Our recommendation is to build your side from our release tags instead of master.

_2018-06-06_
  - [Downloads for v0.11.0](https://github.com/okkur/syna/releases/tag/v0.11.0)
  - [Changelog since v0.10.0](#changes-since-v010)

## Documentation
[Documentation](/tree/v0.11.0/docs)
[Examples](/tree/v0.11.0/exampleSite)


## Changes since v0.10.0

  - BREAKING: Remove split layout in favour of content-split fragment
  - BREAKING: Change all frontmatter variables named `link` to `url`
  - BREAKING: Contact fragment configuration are loaded within the fragment controller
  - NOTE: jQuery and jQuery Form Validator and BootstrapJS have been replaced with much smaller replacements
  - NOTE: Nav and Footer are now fragments and should be configured
  - Full rework of contact fragment
  - Add support for global fragments
  - Scroll to top button
  - Netlify contact form support
  - Use snake_case variable names
  - Use nesting for frontmatter variables
  - Default attribution to opt-in
  - Settable jumbotron background
  - Auto hide navbar (no menu items) with optional overwrite
  - Single member mode for Member fragment
  - Makefile to build and run a development server
  - Add resource fallthrough to all images
  - Remove extra whitespace in layout files
  - Automatically set lastmod for content files
  - Upgrade to Bootstrap v4
  - Load all assets locally and remove usage of CDNs
  - Introduce webpack for development
  - Upgrade to latest Bootstrap v4.1
  - Auto hide empty navigation bar

## Fixes since v0.10.0

  - Recaptcha support
  - Jumbotron corners
  - Add links support for logo in footer fragment
  - Fix full width coverage for particle.js
  - Fontawesome icons now need to declare the full icon class: `fab fa-facebook` instead of `fa-facebook`
  - Fix the default hidden contact fields.
  - ParticleJS fixes

---

# v0.10.0
_2018-03-09_
  - [Downloads for v0.10.0](https://github.com/okkur/syna/releases/tag/v0.10.0)
  - [Changelog since v0.9.0](#changes-since-v090)

## Documentation
[Examples](/tree/v0.10.0/exampleSite)

Notes: This version includes a major breaking change.

## Changes since v0.9.0

  - Migrate data files to Page Bundles
  - Use individual content files for member fragment
  - Use individual content files for items fragment
  - Reorganize Content structure

## Fixes since v0.9.0

  - Split up member files into individual files (#13)
  - Move from `<p>` to `<div>` for anything that could contain markdown content (#31)

---

# v0.9.0
_2017-12-08_
  - [Downloads for v0.9.0](https://github.com/okkur/syna/releases/tag/v0.9.0)
  - [Changelog since v0.8.0](#changes-since-v080)

## Documentation
[Examples](/tree/v0.9.0/exampleSite)

Notes: Member and Footer fragments only support brand icons for now.

## Changes since v0.8.0

  - Subscribe fragment reusing embed fragment
  - Pre and Post subtitle for item for item fragment
  - Migrate to Fontawesome v5
  - Unchanged bootstrap v4 scss files
  - Syna specific color overwrite

## Fixes since v0.8.0

  - Page-top anchor (#20)

---

# v0.8.0
_2017-10-23_
  - [Downloads for v0.8.0](https://github.com/okkur/syna/releases/tag/v0.8.0)
  - [Changelog since v0.7.0](#changes-since-v070)

## Documentation
[Examples](/tree/v0.8.0/exampleSite)

## Changes since v0.7.0

  - Add icons to member fragment
  - Color option for hero
  - Item fragment with button and image/icon
  - Cleanup example data
  - Automatic push to demo via gitlab ci
  - Update basefiles via reposeed

## Fixes since v0.7.0

  - Member icon hover
  - Print error on captcha inaccessible

---

# v0.7.0
_2017-10-18_
  - [Downloads for v0.7.0](https://github.com/okkur/syna/releases/tag/v0.7.0)
  - [Changelog since v0.6.0](#changes-since-v060)

## Documentation
[Examples](/tree/v0.7.0/exampleSite)

## Changes since v0.6.0

  - Option to hide unimportant columns on smaller devices
  - Option to center table headers
  - German translation
  - Member fragment
  - Source code note about syna
  - Visual attribution

## Fixes since v0.6.0

  - Table responsiveness
  - Alignment legal footer
  - Improve readability on mobile

---

# v0.6.0
_2017-10-08_
  - [Downloads for v0.6.0](https://github.com/okkur/syna/releases/tag/v0.6.0)
  - [Changelog since v0.5.0](#changes-since-v050)

## Documentation
[Examples](/tree/v0.6.0/exampleSite)

## Changes since v0.5.0

  - Merge item based fragments into item fragment
  - Restructure example data
  - Optional table for item fragment
  - Table fragment

## Fixes since v0.5.0

  - Alignment improvements for item fragment
  - Responsiveness for item images

---

# v0.5.0
_2017-10-08_
  - [Downloads for v0.5.0](https://github.com/okkur/syna/releases/tag/v0.5.0)
  - [Changelog since v0.4.0](#changes-since-v040)

## Documentation
[Examples](/tree/v0.5.0/exampleSite)

## Changes since v0.4.0

  - Height and width option to hero logo
  - Bind hero background image position to bottom
  - Button fragment for call to action
  - Reorganize fragments
  - Cleanup data files
  - Item fragment with cal to action

## Fixes since v0.4.0

  - Fragment include conditionals

---

# v0.4.0
_2017-10-07_
  - [Downloads for v0.4.0](https://github.com/okkur/syna/releases/tag/v0.4.0)
  - [Changelog since v0.3.0](#changes-since-v030)

## Documentation
[Examples](/tree/v0.4.0/exampleSite)

## Changes since v0.3.0

  - Background color for body
  - 404 page
  - Multiple button option for hero
  - Embed fragment for videos or other media

---

# v0.3.0
_2017-10-05_
  - [Downloads for v0.3.0](https://github.com/okkur/syna/releases/tag/v0.3.0)
  - [Changelog since v0.2.0](#changes-since-v020)

## Documentation
[Examples](/tree/v0.3.0/exampleSite)

## Changes since v0.2.0

  - Cleanup exampleSite
  - Two column single page
  - More example navigation

---

# v0.2.0
_2017-10-05_
  - [Downloads for v0.2.0](https://github.com/okkur/syna/releases/tag/v0.2.0)
  - [Changelog since v0.1.0](#changes-since-v010)

## Documentation
[Examples](/tree/v0.2.0/exampleSite)

## Changes since v0.1.0

  - Background color options
  - Simple one column single page

---

# v0.1.0
_2017-10-04_

  - [Downloads for v0.1.0](https://github.com/okkur/syna/releases/tag/v0.1.0)
  - [Changelog since v0.0.0](#changes-since-v000)

## Documentation
[Examples](/tree/v0.1.0/exampleSite)

## Changes since v0.0.0

  - Bootstrap 4 support
  - Logo fragment
  - Contact fragment
  - Legal footer fragment
  - Footer fragment
  - Hero fragment
  - Row based item fragment
  - Column based item fragment
