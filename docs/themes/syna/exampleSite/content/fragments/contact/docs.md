+++
fragment = "content"
weight = 300

title = "Documentation"

[sidebar]
  sticky = true
+++

Contact form can be used to receive messages.

Various methods and providers are supported. You can use Netlify's form service, [formspree](formspree.io) or a custom endpoint.

**NOTE:** If `netlify` variable is set to true then value of `post_url` will be dismissed since Contact fragment can send the data to a single endpoint only.

This fragment uses internationalized text snippets. Customize them in the fragment or in your own `i18n/` website directory.

Contributions for translations are welcome.

### Events

<!-- TODO: revise later -->
This fragment uses the [Events](/docs/events) api by default.

#### Subscribers

##### contact:update

This event is not published by any fragment by default. But you can make use of it through [event urls](/docs/events/#event-urls).

###### name
*type: string*

Fills the name input.

###### email
*type: string*

Fills the email input.

###### phone
*type: string*

Fills the phone input.

###### message
*type: string*

Fills the message input.

### Variables

#### form_name
*type: string*  
*default: contact-form-{{ file_name }}*

Unique name for the form used to identify the form in scripts and on the page.

#### post_url
*type: string*  
*default: formspree.io*  
*Requires email to be set*

URL to your own backend or a service you are using.

#### email
*type: string*

Email used in case of error or if javascript is not available for a functioning contact form.

#### button_text
*type: string*  
*default: i18n contact.defaultButton*

Submit button text of the form.

#### netlify
*type: boolean*  
*default: false*

Setting netlify to `true` will enable Netlify's form handling and it will override any `post_url` configuration.

Using the Netlify form service simplifies form handling. Enable it and submissions should be showing up in your Netlify interface. It includes spam prevention including reCaptcha usage and can be connected to various triggers such as email, slack and more.

**NOTE:** Your website needs to be hosted on [Netlify](https://netlify.com) to take advantage of this.

#### recaptcha
*type: object*  
*default: Not set*

In the case `post_url` is used a reCaptcha can be added to the form by setting recaptcha to `true` and providing a `recaptcha.sitekey`.

This reduces spam submissions to your contact form.

##### recaptcha.sitekey
*type: string*

Your specific Google reCaptcha  sitekey generated within the [recaptcha dashboard](https://www.google.com/recaptcha/intro/v3.html).

#### message
*type: object*

These messages would be shown after submission in case of an error or success.

##### message.success
*type: string*  
*default: i18n contact.defaultGenericSuccess*

##### message.error
*type: string*  
*default: i18n contact.defaultGenericError*

#### fields
*type: Object of objects*

Each object defined under `fields` will be added to the form. You can remove any of the fields and they would not appear.

#### fields.name
##### fields.name.text
*type: string*

##### fields.name.error
*type: string*  
*default: i18n contact.defaultNameError*

#### fields.email
##### fields.email.text
*type: string*

##### fields.email.error
*type: string*  
*default: i18n contact.defaultEmailError*

#### fields.phone
##### fields.phone.text
*type: string*

##### fields.phone.error
*type: string*  
*default: i18n contact.defaultPhoneError*

#### fields.message
##### fields.message.text
*type: string*

##### fields.message.error
*type: string*  
*default: i18n contact.defaultButton*

#### fields.hidden
*type: Array of objects*

You can use this array to add new hidden fields to the form.

##### fields.hidden.name
*type: string*

This field can have any value and it would be registered in the form with the custom value you provide, unless the value is one of the following:

- `site`: If the name of the hidden field is set to site, `value` is overridden with permalink of the current page.
- `page`: `value` would be overridden with URL of the current page.

Any other value for the `name` variable in the `fields.hidden` object will have normal behavior and will not affect the `value` variable.

##### fields.hidden.value
*type: string*

Value of the hidden field. This field will be overridden if `name` is set to `site` or `page`.

[Global variables](/docs/global-variables) are documented as well and have been omitted from this page.
