+++
fragment = "content"
weight = 100

title = "Installation"

[sidebar]
  sticky = true
+++

### Adding Syna to your site

You can use Syna by adding it as a submodule to your website repository and pointing the submodule to the latest release.
This way whenever you want to update the theme you can just pull the updates and checkout to the latest tag.

```bash
git submodule init # If you haven't initialized before
git submodule add https://git.okkur.org/syna themes/syna
cd themes/syna
git checkout v0.15 # Latest release as of now is v0.15
```

*You can visit [Hugo's documentation](https://gohugo.io/themes/installing-and-using-themes/) on installing themes for more information.*

### Using starter

If you don't have a site yet, you can use our [starter](https://git.okkur.org/syna-start).
Using the starter you will have a sample page with several fragment examples and you can use them to start building your own website.

```bash
git clone --recurse-submodules https://git.okkur.org/syna-start # --recurse-submodules will clone the theme
cd syna-start
hugo server -D # to build your website, run hugo instead
```
