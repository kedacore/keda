+++
fragment = "stripe"
weight = 162
background = "secondary"

title = "Payment Fragment with Custom Value"
subtitle = "Doesn't work in demo"

post_url = "https://us-central1-syna-222118.cloudfunctions.net/function-1/charge"
stripe_token = "pk_test_36PckiAlsGm9KmHj9b034GAW"

product = "Example Product"

[user_input]
  default = "20.00"
  currencies = ['usd', 'eur', 'cad'] # First currency will be used as default

[fields.email]
  text = "Your email address"
+++

You can pay for the product by filling this form (provided by Stripe).
