+++
fragment = "content"
weight = 300

title = "Documentation"

[sidebar]
  sticky = true
+++

### Variables

#### Content
*type: string*  
*Default Hugo variable. Defined simply by entering it's value under the frontmatter section.*

This variable is optional.

#### align
*type: string*  
*accepted values: right, center, left*  
*default: center*

Defines the layout of the fragment.

#### pre
*type: string*

The value shown before the sidebar content. This variable is optional.

#### post
*type: string*

The value shown after the sidebar content. This variable is optional.

#### buttons
*type: array of objects*

Buttons can be defined using the same [variables as buttons fragment](/fragments/buttons#buttons).

If an asset is defined or the fragment is centered, buttons are displayed in the main section of the fragment. If not, they're displayed in the siderbar.

This variable is optional.

#### asset
*type: [asset object](/docs/global-variables/#asset)*

Either an image or an icon. The asset is displayed in the sidebar unless the fragment is centered.

#### header, rows (table fragment)
*type: [table object](/fragments/table/#docs)*

A table can be added to the item fragment using the same variables as the table fragment. Please read more about tables in the [table fragment's docs](/fragments/table/#docs).

[Global variables](/docs/global-variables) are documented as well and have been omitted from this page.
