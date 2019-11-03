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

- .Site.Menus.main

### Variables

#### search
*type: boolean*  
*default: false*

If set to `true`, search is enabled within the navbar.  
**Note:** The additional input field used for search will alter the menu layout.

#### sticky
*type: boolean*  
*default: false*

If set to `true`, navbar will stick to the top of the screen whenever page scrolls past it.

#### prepend, postpend
*type: array of objects*

Menu like objects that are used to add menus before and after the main menu.

##### prepend/postpend.url
*type: string*

##### prepend/postpend.name
*type: string*

#### breadcrumb
*type: object*

If object is set, breadcrumbs will be shown under the navbar.

##### breadcrumb.display
*type: boolean*  
*default: true*

##### breadcrumb.level
*type: number*  
*default: 1*

Define the section level the breadcrumb will start being shown.
 
The default value `1` would lead to the following being defined: 

```
content/_index # level 0, not shown
content/blog/_index # level 1, shown
content/blog/article-1 # level 2, shown
```

##### breadcrumb.background
*type:  string*  
*recommended: primary, secondary, white, light, dark*  
*accepted values: primary, secondary, white, light, dark, warning, success, danger, info, transparent*

#### asset
*type: [asset object](/docs/global-variables/#asset)*

Asset will be shown as a clickable logo directing to the main page.

#### repo_button
*type: object*

Enable a button on the top right navbar. Usually used to link to your repository such as Github or Gitlab.  
The icon can be customized via `repo_button.icon`.

##### repo_button.url
*type: string*

##### repo_button.text
*type: string*  
*default: star*

##### repo_button.icon
*type: string*  
*default: fab fa-github*

[Global variables](/docs/global-variables) are documented as well and have been omitted from this page.
