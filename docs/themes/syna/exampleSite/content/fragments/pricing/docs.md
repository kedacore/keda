+++
fragment = "content"
weight = 300

title = "Documentation"

[sidebar]
  sticky = true
+++

This fragment needs a fragment controller file and subitems. You need to create a directory for this fragment and put `index.md` with `fragment = "pricing"` and subitems next to that file.

### Events

Pricing fragment doesn't subscribe to any events by default and only publishes events in [special circumstances](#usage-with-events). Please checkout [usage with events](#usage-with-events) to learn more. The published event is listened to in the [Stripe fragment](/fragments/stripe) and will cause the Stripe fragment to change it's properties.

### Variables

`index.md` doesn't use any variables. Following variables are for subitems.

#### price
*type: string*

#### highlight
*type: boolean*  
*default: false*

If set to `true`, the column will have more `z-index`, width and stay a bit on top of other columns.

#### button_text
*type: string*

Title of the button on the column.

#### button_url
*type: string*

URL of the button on the column.

##### Usage with events

You can make use of the Events api through this variable.

- Not setting the `button_url` variable will publish an event with `title`, `subtitle`, `price` and `currency` variables.
- Setting it to an event url such as: `/fragments/stripe/?event=pricing:change&product=Starting plan&price=$9.99/mo&currency=usd` will redirect the page and publish a custom event.

#### features
*type: array of objects*

This array will be displayed on the pricing column, listing what is aviable in the current plan.

##### features.text
*type: string*

##### features.icon
*type: string*

[Global variables](/docs/global-variables) are documented as well and have been omitted from this page.
