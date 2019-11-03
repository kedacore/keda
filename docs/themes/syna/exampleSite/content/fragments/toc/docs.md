+++
fragment = "content"
weight = 300

title = "Documentation"

[sidebar]
  sticky = true
+++

This fragment renders table of contents of a content fragment. It can be used standalone, or in a slot in `list` or `content` fragment.

If it's standalone, then the `content` fragment is required.

### Variables

#### content
*type: string*  
*required*

Path to the content fragment that you need the table of contents of. This path can be relative to the page or relative to `content/` directory.

[Global variables](/docs/global-variables) are documented as well and have been omitted from this page.
