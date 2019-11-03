+++
fragment = "content"
weight = 300

title = "Documentation"

[sidebar]
  sticky = true
+++

### Variables

#### media
*type: string*

URL to embed in the fragment. Any value would override `media_source`.

#### media_source
*type: string*

Custom HTML code for your `iframe` or embed objects.

If `media` is set, this value would not be used.

#### responsive
*type: boolean*  
*default: true*

Make the embed object or iframe responsive. It can be customized with help of the `ratio` variable.  
Using `responsive` any `size` variable would be overridden.

#### ratio
*type: string*  
*accepted values: 21by9, 16by9, 4by3, 1by1*  
*default: 4by3*

If `responsive` is set to `false`, this value would be ignored.

#### size
*type: number*  
*default: 75*

Percentage value to force embed object or iframe to be a specific width of its parent.

If `responsive` is set to `true` then this value would be ignored.

[Global variables](/docs/global-variables) are documented as well and have been omitted from this page.
