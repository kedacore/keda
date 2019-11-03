+++
fragment = "content"
weight = 300

title = "Documentation"

[sidebar]
  sticky = true
+++

### Note

Use one instance of this fragment per page. Running more might lead to unexpected issues.

### Menus

- .Site.Menus.footer
- .Site.Menus.footer_social

### Variables

#### menu_title
*type: string*

Title of the menu displayed in the footer.

#### asset
*type: [asset object](/docs/global-variables/#asset)*  
*size: 220x100*

The asset such as images or graphics is displayed on top left of the fragment and can be used for a logo.

The global variables `subtitle`, `title_align` are not supported in this fragment.

[Global variables](/docs/global-variables) are documented as well and have been omitted from this page.
