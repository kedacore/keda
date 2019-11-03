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

- .Site.Menus.copyright_footer

### Variables

#### copyright
*type: string*  
*default: Copyright {{ $Year }} {{ .Site.params.name }} (i18n enabled)*

#### attribution
*type: boolean*  
*default: false*

If set to true, the Syna theme name and link would be shown using the `attribution` snippet from `i18n`.

[Global variables](/docs/global-variables) are documented as well and have been omitted from this page.
