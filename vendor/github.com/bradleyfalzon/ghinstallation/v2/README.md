# ghinstallation

[![GoDoc](https://godoc.org/github.com/bradleyfalzon/ghinstallation?status.svg)](https://godoc.org/github.com/bradleyfalzon/ghinstallation/v2)

`ghinstallation` provides `Transport`, which implements `http.RoundTripper` to
provide authentication as an installation for GitHub Apps.

This library is designed to provide automatic authentication for
https://github.com/google/go-github or your own HTTP client.

See
https://developer.github.com/apps/building-integrations/setting-up-and-registering-github-apps/about-authentication-options-for-github-apps/

# Installation

Get the package:

```bash
GO111MODULE=on go get -u github.com/bradleyfalzon/ghinstallation/v2
```

# GitHub Example

```go
import "github.com/bradleyfalzon/ghinstallation/v2"

func main() {
    // Shared transport to reuse TCP connections.
    tr := http.DefaultTransport

    // Wrap the shared transport for use with the app ID 1 authenticating with installation ID 99.
    itr, err := ghinstallation.NewKeyFromFile(tr, 1, 99, "2016-10-19.private-key.pem")
    if err != nil {
        log.Fatal(err)
    }

    // Use installation transport with github.com/google/go-github
    client := github.NewClient(&http.Client{Transport: itr})
}
```

You can also use [`New()`](https://pkg.go.dev/github.com/bradleyfalzon/ghinstallation/v2#New) to load a key directly from a `[]byte`.

# GitHub Enterprise Example

For clients using GitHub Enterprise, set the base URL as follows:

```go
import "github.com/bradleyfalzon/ghinstallation/v2"

const GitHubEnterpriseURL = "https://github.example.com/api/v3"

func main() {
    // Shared transport to reuse TCP connections.
    tr := http.DefaultTransport

    // Wrap the shared transport for use with the app ID 1 authenticating with installation ID 99.
    itr, err := ghinstallation.NewKeyFromFile(tr, 1, 99, "2016-10-19.private-key.pem")
    if err != nil {
        log.Fatal(err)
    }
    itr.BaseURL = GitHubEnterpriseURL

    // Use installation transport with github.com/google/go-github
    client := github.NewEnterpriseClient(GitHubEnterpriseURL, GitHubEnterpriseURL, &http.Client{Transport: itr})
}
```

## What is app ID and installation ID

`app ID` is the GitHub App ID. \
You can check as following : \
Settings > Developer > settings > GitHub App > About item

`installation ID` is a part of WebHook request. \
You can get the number to check the request. \
Settings > Developer > settings > GitHub Apps > Advanced > Payload in Request
tab

```
WebHook request
...
  "installation": {
    "id": `installation ID`
  }
```

# Customizing signing behavior

Users can customize signing behavior by passing in a
[Signer](https://pkg.go.dev/github.com/bradleyfalzon/ghinstallation/v2#Signer)
implementation when creating an
[AppsTransport](https://pkg.go.dev/github.com/bradleyfalzon/ghinstallation/v2#AppsTransport).
For example, this can be used to create tokens backed by keys in a KMS system.

```go
signer := &myCustomSigner{
  key: "https://url/to/key/vault",
}
atr := NewAppsTransportWithOptions(http.DefaultTransport, 1, WithSigner(signer))
tr := NewFromAppsTransport(atr, 99)
```

# License

[Apache 2.0](LICENSE)

# Dependencies

- [github.com/golang-jwt/jwt-go](https://github.com/golang-jwt/jwt-go)
