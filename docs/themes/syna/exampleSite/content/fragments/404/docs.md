+++
fragment = "content"
weight = 300

title = "Documentation"

[sidebar]
  sticky = true
+++

404 fragment looks a lot like the `hero` fragment, with a default call to action button to return to the homepage and a description in addition to subtitle and title.

In order to customize the fragment, which only appears on 404 pages, you have to place it in your `/content/_global` directory.

### Variables

#### redirect_text
*type: string*  
*default: i18n "404.direction"*

Description of the page.

#### button_text
*type: string*  
*default: i18n "404.button"*

Title of the call to action button.

#### redirect_url
*type: string*  
*default: "/"*

URL for the call to action button.

#### asset
*type: [asset object](/docs/global-variables/#asset)*

The asset will be displayed on top of the fragment, before title and subtitle.

Asset in this fragment only support images that are supported in `img` tag (`.png`, `.jpg`, `.gif`, `.svg`, etc.).

[Global variables](/docs/global-variables) are documented as well and have been omitted from this page.
