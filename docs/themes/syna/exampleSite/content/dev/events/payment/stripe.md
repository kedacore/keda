+++
fragment = "stripe"
weight = 120

post_url = "https://us-central1-syna-222118.cloudfunctions.net/function-1/charge"
stripe_token = "pk_test_36PckiAlsGm9KmHj9b034GAW"

[[prices]]
  text = "20.00$"
  currency = "usd"

[[prices]]
  text = "30.00$"
  currency = "usd"

[fields.email]
  text = "Your email address"
+++
