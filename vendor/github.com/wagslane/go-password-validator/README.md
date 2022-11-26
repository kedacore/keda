# go-password-validator

Simple password validator using raw entropy values. Hit the project with a star if you find it useful ⭐

Supported by [Qvault](https://qvault.io)

[![](https://godoc.org/github.com/wagslane/go-password-validator?status.svg)](https://godoc.org/github.com/wagslane/go-password-validator) ![Deploy](https://github.com/wagslane/go-password-validator/workflows/Tests/badge.svg)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

This project can be used to front a password strength meter, or simply validate password strength on the server. Benefits:

* No stupid rules (doesn't require uppercase, numbers, special characters, etc)
* Everything is based on entropy (raw cryptographic strength of the password)
* Doesn't load large sets of data into memory - very fast and lightweight
* Doesn't contact any API's or external systems
* Inspired by this [XKCD](https://xkcd.com/936/)

![XKCD Passwords](https://imgs.xkcd.com/comics/password_strength.png)

## ⚙️ Installation

Outside of a Go module:

```bash
go get github.com/wagslane/go-password-validator
```

## 🚀 Quick Start

```go
package main

import (
    passwordvalidator "github.com/wagslane/go-password-validator"
)

func main(){
    entropy := passwordvalidator.GetEntropy("a longer password")
    // entropy is a float64, representing the strength in base 2 (bits)

    const minEntropyBits = 60
    err := passwordvalidator.Validate("some password", minEntropyBits)
    // if the password has enough entropy, err is nil
    // otherwise, a formatted error message is provided explaining
    // how to increase the strength of the password
    // (safe to show to the client)
}
```

## What Entropy Value Should I Use?

It's up to you. That said, here is a graph that shows some common timings for different values, somewhere in the 50-70 range seems "reasonable".

Keep in mind that attackers likely aren't just brute-forcing passwords, if you want protection against common passwords or [PWNed passwords](https://haveibeenpwned.com/) you'll need to do additional work. This library is lightweight, doesn't load large datasets, and doesn't contact external services.

![entropy](https://external-preview.redd.it/rhdADIZYXJM2FxqNf6UOFqU5ar0VX3fayLFpKspN8uI.png?auto=webp&s=9c142ebb37ed4c39fb6268c1e4f6dc529dcb4282)

## How It Works

First, we determine the "base" number. The base is a sum of the different "character sets" found in the password.

We've *arbitrarily* chosen the following character sets:

* 26 lowercase letters
* 26 uppercase letters
* 10 digits
* 5 replacement characters - `!@$&*`
* 5 seperator characters - `_-., `
* 22 less common special characters - `"#%'()+/:;<=>?[\]^{|}~`


Using at least one character from each set your base number will be 94: `26+26+10+5+5+22 = 94`

Every unique character that doesn't match one of those sets will add `1` to the base.

If you only use, for example, lowercase letters and numbers, your base will be 36: `26+10 = 36`.

After we have calculated a base, the total number of brute-force-guesses is found using the following formulae: `base^length`

A password using base 26 with 7 characters would require `26^7`, or `8031810176` guesses.

Once we know the number of guesses it would take, we can calculate the actual entropy in bits using `log2(guesses)`. That calculation is done in log space in practice to avoid numeric overflow.

### Additional Safety

We try to err on the side of reporting *less* entropy rather than *more*.

#### Same Character

With repeated characters like `aaaaaaaaaaaaa`, or `111222`, we modify the length of the sequence to count as no more than `2`.

* `aaaa` has length 2
* `111222` has length 4

#### Common Sequences

Common sequences of length three or greater count as length `2`.

* `12345` has length 2
* `765432` has length 2
* `abc` has length 2
* `qwerty` has length 2

The sequences are checked from back->front and front->back. Here are the sequences we've implemented so far, and they're case-insensitive:

* `0123456789`
* `qwertyuiop`
* `asdfghjkl`
* `zxcvbnm`
* `abcdefghijklmnopqrstuvwxyz`

## Not ZXCVBN

There's another project that has a similar purpose, [zxcvbn](https://github.com/dropbox/zxcvbn), and you may want to check it out as well. Our goal is not to be zxcvbn, because it's already good at what it does. `go-password-validator` doesn't load any large datasets of real-world passwords, we write simple rules to calculate an entropy score. It's up to the user of this library to decide how to use that entropy score, and what scores constitute "secure enough" for their application.

## 💬 Contact

[![Twitter Follow](https://img.shields.io/twitter/follow/wagslane.svg?label=Follow%20Wagslane&style=social)](https://twitter.com/intent/follow?screen_name=wagslane)

Submit an issue (above in the issues tab)

## Transient Dependencies

None! And it will stay that way, except of course for the standard library.

## 👏 Contributing

I love help! Contribute by forking the repo and opening pull requests. Please ensure that your code passes the existing tests and linting, and write tests to test your changes if applicable.

All pull requests should be submitted to the `main` branch.

```bash
make test
make fmt
make vet
make lint
```
