**DON'T USE THIS PACKAGE** - use [`xdg-go/scram`](https://pkg.go.dev/github.com/xdg-go/scram) instead!

I renamed this to [`xdg-go/scram`](https://pkg.go.dev/github.com/xdg-go/scram) in October 2018.  This didn't break dependencies at the time because Github redirected requests.  In March 2021, I made `xdg-go/scram` a module, which can't be used as `xdg/scram` with Github redirects.  This repository has been recreated to support legacy dependencies.

See my article [How I broke the MongoDB Go driver ecosystem](https://xdg.me/i-broke-the-mongodb-go-driver-ecosystem/) for more details.
