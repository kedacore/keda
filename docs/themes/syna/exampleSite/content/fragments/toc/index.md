+++
fragment = "content"
weight = 100

title = "ToC"
background = "light"
+++

Add table of contents of any content fragment to the page.

<!--more-->

ToC fragment can be used to render Table of Contents of any content fragment
from any page. The default behavior is to get the name of the content fragment
on the same page:

```
content = "content.md"
```

But if you need to get the Table of Contents of other pages, just address them
relative to the `content` directory:

```
content = "fragments/hero/index.md"
```
