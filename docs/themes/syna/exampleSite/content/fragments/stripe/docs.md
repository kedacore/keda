+++
fragment = "content"
weight = 300

title = "Documentation"

[sidebar]
  sticky = true
+++

Stripe fragment creates a form with a Stripe element inside, which is used to create payments for the users.

It requires a backend for creating the customer and charge for Stripe. An example can be found at [here](https://syna.okkur.org/stripe-gcf) which can be deployed on Google Cloud Functions and works out of the box.

The use of a backend is also useful for saving other information from the user such as the product they selected (in case you are providing more than one products) or other metadata.

### Events

This fragment uses the [Events](/docs/events) api by default.

#### Subscribers

##### pricing:change

This event is published by the [Pricing fragment](/fragments/pricing/#events). It can also be published by [event urls](/docs/events/#event-urls).

###### product
*type: string*

If provided, this variable will override the preconfigured `product` variable and will be displayed on top of the form.

###### description
*type: string*

Payment metadata. This variable is not displayed to the user but will be sent to the server with the form data.

###### price
*type: string*

The price that will be charged to the user's account. The value should follow the same rules as [prices.text](#prices-text).

###### currency
*type: string*

The currency in which the user will be charged. The value should follow the same rules as [prices.currency](#prices-currency).

### Variables

#### post_url
*type: string*

Form data will be sent to this url. The url should point to an api which can handle Stripe's charge customer api. An [example](https://syna.okkur.org/stripe-gcf) is provided for more information.

#### stripe_token
*type: string*

Publishable API token from your Stripe account should

#### product
*type: string*

Your product's name. This variable is optional.

#### fields.email
*type: object*

##### fields.email.text
*type: string*

Placeholder text shown in the email input.

#### prices
*type: array of objects*

This variable is optional. If the array contains a single item, the form's design will change to indicate that the price cannot be changed. If no values are provided, the form will fallback to `user_input`.

##### prices.text
*type: string*

The price that will be charged to the user's account. This variable is shown to the user as is but it will be converted to a number with no decimals when it is submitted. The value can be anything that can be formatted to a number, for example: `20.00$`, `20`, `19.95$/mo`.

##### prices.currency
*type: string*

The currency in which the user will be charged. The value should be the three-letter ISO coded of the currency. You can find the appropriate ISO code [here](https://stripe.com/docs/currencies#presentment-currencies).

#### user_input
*type: object*

This variable will create a field and will display either the field itself or a button that will switch the form to editable mode. The field is displayed based on whether the `prices` field contains any items or not.

##### user_input.default
*type: string*

Default value in the editable price field. The value should follow the same rules as [prices.text](#prices-text).

##### user_input.currencies
*type: array of strings*

The currency in which the user will be charged. The value should follow the same rules as [prices.currency](#prices-currency).

[Global variables](/docs/global-variables) are documented as well and have been omitted from this page.
