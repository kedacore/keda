+++
fragment = "content"
weight = 100

title = "Image Resource Fallthrough"

[sidebar]
  sticky = true
+++

Some fragments may display images, if configured in their content files.
The configuration accepts a filename and will search for the actual image using a fallthrough mechanism.
The lookup order is shown below:

- 1. Lookup within the fragment's subdirectory such as `content/[page]/[fragment]/[filename].md`).
- 2. Lookup, if the above doesn't match any files, it will try to match a file within the page directory such as `content/[page]`.
- 3. Lookup, if none of the above match any files, it will try the global `static/images/` directory.

So the fragment will look in the following order `fragment > page > global`. If you need to use an image in several pages you can put it in the `static/images/` directory and the image would be available globally. But if an image may differ between two pages or even two fragments of same the type, it's possible to collocate it with the content files either on a per page or per fragment level.

Syna supports custom favicons in config.toml allowing for ICO, PNG or SVG image formats. In order to use one of the custom favicon formats, you can specify the image file name in config.toml and save the image file in the 'static/' directory.
