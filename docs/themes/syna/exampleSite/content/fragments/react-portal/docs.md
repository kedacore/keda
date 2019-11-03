+++
fragment = "content"
weight = 300

title = "Documentation"

[sidebar]
  sticky = true
+++

This fragment doesn't make use of any variables except for global variables.

This fragment makes it possible to load a React component and render it on the page. To do so, you need to create a React component as usual and register it in the page and then the protal will render the component.

To register a React component in the portal, you need to add your component and name of the fragment's controller file to `window.synaPortals` object.

```
window.synaPortals['UNIQUE_IDENTIFIER] = {
  component: YOUR_COMPONENT,
  container: '#FRAGMENT_FILENAME [data-portal]',
};
```

*Replace capitalized phrases in the code above with what you need.*

Add this code to your script (the file you have defined your React component). If your component doesn't show up, make sure your code is being loaded before `syna-react.js`.
An easy way to make sure of that is to use the `config` fragment.

[Global variables](/docs/global-variables) are documented as well and have been omitted from this page.
