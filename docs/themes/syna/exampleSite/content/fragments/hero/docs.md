+++
fragment = "content"
weight = 300

title = "Documentation"

[sidebar]
  sticky = true
+++

### Extra Files

Hero fragment supports extra files to configure how it appears. Therefore you need
to create a directory for the hero fragment and name your fragment `index.md` inside that directory.

Extra files can be provided for background, logo and particles effect configuration.

#### [hero]/config.json

If `particles` variable is set to true, a default configuration for the `particles-js` library
will be used. You can customize this configuration by providing a `config.json` file
inside the hero fragment's directory.

### Variables

#### title_page
*type: boolean*  
*default: false*

If set to `true` and `asset` is not set, title will be the same as page title.

#### particles
*type: boolean*  
*default: false*

If set to `true`, Particles.js would be added to the page and displayed in the Hero fragment. 

#### minHeight
*type: string*  
*accepted values: css size values*  
*default: initial*

Sets minimum height of hero fragment.

#### header
*type: [asset object](/docs/global-variables/#asset)*

Background image of the Hero fragment.

#### asset
*type: [asset object](/docs/global-variables/#asset)*

The asset is displayed on the Hero fragment instead of the `title` and can be used to display a logo. 

**Note:** If set, title will not be shown and subtitle will be displayed in a bigger size.

#### buttons
*type: array of objects*

Call to action buttons displayed after title or asset and subtitle. 

Visit [Buttons fragment page (documentation section)](/fragments/buttons#docs) for documentation on which variables to use.

[Global variables](/docs/global-variables) are documented as well and have been omitted from this page.
