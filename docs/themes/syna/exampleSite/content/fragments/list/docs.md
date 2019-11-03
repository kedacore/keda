+++
fragment = "content"
weight = 300

title = "Documentation"

[sidebar]
  sticky = true
+++

### Variables

#### count
*type: number*  
*default: 10*

Number of pages displayed in the list.  
If List fragment is registered in a list page, `count` will be used for pagination as well.

#### section
*type: string*  
*default: current section*

This value will be used for finding pages. The value by default is the path to the current section. If you want to change the default value, enter the path to the section relative to `content/` directory. For example to list pages from `content/blog` directory (`blog` section), the value would be `blog`.

#### summary
*type: boolean*  
*default: true*

If set to `true`, it will show the summary of the page. It works by looking for a Content fragment in the page and fetching the summary, either by looking for a `summary` variable in that Content fragment or by using Hugo to summarize the content.

#### sort
*type: string*
*accepted values: All Hugo [page variables](https://gohugo.io/variables/page/)*
*default: PublishDate*

Sets the sort property. Accepted values are all the public fields on the page variable. Please refer to [*Page variables*](https://gohugo.io/variables/page/) documentation page on Hugo.

#### sort_order
*type: string*
*accepted values: asc, desc*
*default: asc*

Sets the sort order.

#### images
*type: boolean*  
*default: true*

If set to `true`, it will show the image of the page. It works by looking for a Content fragment in the page and fetching the `image` value in it.

#### read_more
*type: boolean*
*default: empty*

- empty: show when content is truncated
- false: never show
- true: always show

#### tiled
*type: boolean*  
*default: false*

If set to `true`, the fragment will show each page as a card in a two column layout. By default, pages are listed below each other.

#### display_date
*type: boolean*  
*default: false*

#### collapsible
*type: boolean*  
*default: false*

Content of each page (everything after the title) will be collapsible. If there is no content after the title, then that item will not be collapsible and will not show the arrow indicating collapsible functionlity.

#### subsections
*type: boolean*  
*default: true*

If enabled list pages from nested/child sections will be shown.

#### subsection_leaves
*type: boolean*  
*default: false*

Shows subsection leaf pages. Like `subsections` variable but will only show normal pages in nested sections.

[Global variables](/docs/global-variables) are documented as well and have been omitted from this page.
