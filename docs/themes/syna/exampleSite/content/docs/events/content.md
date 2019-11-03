+++
fragment = "content"
weight = 100

title = "Events"

[sidebar]
  sticky = true
+++

Syna has a built-in communication stream that is mostly used to send messages between fragments. Although the stream is not limited to fragments, it can be used to publish anything and anywhere in the code, a subscriber can be added that can listen to the published events.

The stream has three main functions, `publish`, `subscribe` and `unsubscribe`. Any event published using the `publish` function can be listened to by `subscribe` function, allowing easy, decoupled and isolated functionalities inside fragments and the whole page.

Events can be triggered either by directly calling the `publish` function on the `window.syna.stream` object or by an special url query which is explained [below](#event-urls).

### `publish` function

You can call `publish` function anywhere in your code. In the built-in fragments, it's mostly called on buttons' click handlers.

The function accepts two arguments, `topic` and `args`

#### topic
*type: string*  

Event topic.

#### args
*type: string | object*

When using the string format, the string should look like the following format:

```
key:value,key2:value2
```

The string will be converted to an object and will be published as paramaters along with the topic.

### Event urls

There is an easier way to publish events and that is to open a url. This is more useful when the event is going to be published when an `a` tag is clicked on. The url event is a url with a query appended to it that has an `event` attribute in it. For example:

```
/fragments/stripe/?event=pricing:change&product=Starting plan&price=$9.99/mo&currency=usd
```

In this example, when the page `/fragments/stripe/` is opened, the event stream will translate the query, publishing a `pricing:change` event with `product`, `price` and `currency` parameters attached to it. If there is an `event` attribute in the query, every other attribute will be published as it's parameter.

**NOTE:** Event urls need to be Base64 encrypted. In order for the example above to work you need to add `unsafeEvents = true` to your `config.toml` file. Base64 events need to have `e` attribute inside them with the entire encrypted query as it's value. The url above will look like this in the Base64 format.

```
/fragments/stripe/?e=P2V2ZW50PXByaWNpbmc6Y2hhbmdlJnByb2R1Y3Q9U3RhcnRpbmcgcGxhbiZwcmljZT0kOS45OS9tbyZjdXJyZW5jeT11c2Q=
```

*To convert your event to Base64, use an online service such as [base64encode.org](base64encode.org) or you can call `btoa` function in your browser's devtools console. For example: `btoa('?event=...&key=...&key2=...`)*

### `subscribe` function

The subscribe function adds a listener for the specified `topic`. This function receives the following arguments.

#### topic
*type: string*

Event topic.

#### listener

The listener function. The function is invoked with event parameters.

Example: The following published event:

```
/fragments/stripe/?event=pricing:change&product=Starting plan&price=$9.99/mo&currency=usd

// or by calling the publish function

window.syna.stream.publish('pricing:change', {
    product: 'Starting plan',
    price: '$9.99/mo',
    currency: 'usd'
})
```

will trigger the following subscriber:

```
window.syna.stream.subscribe('pricing:change', function(params) {
    alert('You have selected ' + params.product)
})
```
