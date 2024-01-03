# Envtest Binaries Manager

This is a small tool that manages binaries for envtest. It can be used to
download new binaries, list currently installed and available ones, and
clean up versions.

To use it, just go-install it on 1.19+ (it's a separate, self-contained
module):

```shell
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
```

For full documentation, run it with the `--help` flag, but here are some
examples:

```shell
# download the latest envtest, and print out info about it
setup-envtest use

# download the latest 1.19 envtest, and print out the path
setup-envtest use -p path 1.19.x!

# switch to the most recent 1.21 envtest on disk
source <(setup-envtest use -i -p env 1.21.x)

# list all available local versions for darwin/amd64
setup-envtest list -i --os darwin --arch amd64

# remove all versions older than 1.16 from disk
setup-envtest cleanup <1.16

# use the value from $KUBEBUILDER_ASSETS if set, otherwise follow the normal
# logic for 'use'
setup-envtest --use-env

# use the value from $KUBEBUILDER_ASSETS if set, otherwise use the latest
# installed version
setup-envtest use -i --use-env

# sideload a pre-downloaded tarball as Kubernetes 1.16.2 into our store
setup-envtest sideload 1.16.2 < downloaded-envtest.tar.gz
```

## Where does it put all those binaries?

By default, binaries are stored in a subdirectory of an OS-specific data
directory, as per the OS's conventions.

On Linux, this is `$XDG_DATA_HOME`; on Windows, `%LocalAppData`; and on
OSX, `~/Library/Application Support`.

There's an overall folder that holds all files, and inside that is
a folder for each version/platform pair.  The exact directory structure is
not guarnateed, except that the leaf directory will contain the names
expected by envtest.  You should always use `setup-envtest fetch` or
`setup-envtest switch` (generally with the `-p path` or `-p env` flags) to
get the directory that you should use.

## Why do I have to do that `source <(blah blah blah)` thing

This is a normal binary, not a shell script, so we can't set the parent
process's environment variables.  If you use this by hand a lot and want
to save the typing, you could put something like the following in your
`~/.zshrc` (or similar for bash/fish/whatever, modified to those):

```shell
setup-envtest() {
    if (($@[(Ie)use])); then
        source <($GOPATH/bin/setup-envtest "$@" -p env)
    else
        $GOPATH/bin/setup-envtest "$@"
    fi
}
```

## What if I don't want to talk to the internet?

There are a few options.

First, you'll probably want to set the `-i/--installed` flag. If you want
to avoid forgetting to set this flag, set  the `ENVTEST_INSTALLED_ONLY`
env variable, which will switch that flag on by default.

Then, you have a few options for managing your binaries:

- If you don't *really* want to manage with this tool, or you want to
  respect the $KUBEBUILDER_ASSETS variable if it's set to something
  outside the store, use the `use --use-env -i` command.

  `--use-env` makes the command unconditionally use the value of
  KUBEBUILDER_ASSETS as long as it contains the required binaries, and
  `-i` indicates that we only ever want to work with installed binaries
  (no reaching out the remote GCS storage).

  As noted about, you can use `ENVTEST_INSTALLED_ONLY=true` to switch `-i`
  on by default, and you can use `ENVTEST_USE_ENV=true` to switch
  `--use-env` on by default.

- If you want to use this tool, but download your gziped tarballs
  separately, you can use the `sideload` command.  You'll need to use the
  `-k/--version` flag to indicate which version you're sideloading.

  After that, it'll be as if you'd installed the binaries with `use`.

- If you want to talk to some internal source, you can use the
  `--remote-bucket` and `--remote-server` options.  The former sets which
  GCS bucket to download from, and the latter sets the host to talk to as
  if it were a GCS endpoint. Theoretically, you could use the latter
  version to run an internal "mirror" -- the tool expects

  - `HOST/storage/v1/b/BUCKET/o` to return JSON like

    ```json
    {"items": [
        {"name": "kubebuilder-tools-X.Y.Z-os-arch.tar.gz", "md5Hash": "<base-64-encoded-md5-hash>"},
        {"name": "kubebuilder-tools-X.Y.Z-os-arch.tar.gz", "md5Hash": "<base-64-encoded-md5-hash>"},
    ]}
    ```

  - `HOST/storage/v1/b/BUCKET/o/TARBALL_NAME` to return JSON like
    `{"name": "kubebuilder-tools-X.Y.Z-os-arch.tar.gz", "md5Hash": "<base-64-encoded-md5-hash>"}`

  - `HOST/storage/v1/b/BUCKET/o/TARBALL_NAME?alt=media` to return the
    actual file contents
