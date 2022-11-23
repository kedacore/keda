// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 The Kubernetes Authors

package main

import (
	goflag "flag"
	"fmt"
	"os"
	"runtime"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/spf13/afero"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"

	envp "sigs.k8s.io/controller-runtime/tools/setup-envtest/env"
	"sigs.k8s.io/controller-runtime/tools/setup-envtest/remote"
	"sigs.k8s.io/controller-runtime/tools/setup-envtest/store"
	"sigs.k8s.io/controller-runtime/tools/setup-envtest/versions"
	"sigs.k8s.io/controller-runtime/tools/setup-envtest/workflows"
)

const (
	// envNoDownload is an env variable that can be set to always force
	// the --installed-only, -i flag to be set.
	envNoDownload = "ENVTEST_INSTALLED_ONLY"
	// envUseEnv is an env variable that can be set to control the --use-env
	// flag globally.
	envUseEnv = "ENVTEST_USE_ENV"
)

var (
	force         = flag.Bool("force", false, "force re-downloading dependencies, even if they're already present and correct")
	installedOnly = flag.BoolP("installed-only", "i", os.Getenv(envNoDownload) != "",
		"only look at installed versions -- do not query the remote API server, "+
			"and error out if it would be necessary to")
	verify = flag.Bool("verify", true, "verify dependencies while downloading")
	useEnv = flag.Bool("use-env", os.Getenv(envUseEnv) != "", "whether to return the value of KUBEBUILDER_ASSETS if it's already set")

	targetOS   = flag.String("os", runtime.GOOS, "os to download for (e.g. linux, darwin, for listing operations, use '*' to list all platforms)")
	targetArch = flag.String("arch", runtime.GOARCH, "architecture to download for (e.g. amd64, for listing operations, use '*' to list all platforms)")

	// printFormat is the flag value for -p, --print.
	printFormat = envp.PrintOverview
	// zapLvl is the flag value for logging verbosity.
	zapLvl = zap.WarnLevel

	binDir = flag.String("bin-dir", "",
		"directory to store binary assets (default: $OS_SPECIFIC_DATA_DIR/envtest-binaries)")
	remoteBucket = flag.String("remote-bucket", "kubebuilder-tools", "remote GCS bucket to download from")
	remoteServer = flag.String("remote-server", "storage.googleapis.com",
		"remote server to query from.  You can override this if you want to run "+
			"an internal storage server instead, or for testing.")
)

// TODO(directxman12): handle interrupts?

// setupLogging configures a Zap logger.
func setupLogging() logr.Logger {
	logCfg := zap.NewDevelopmentConfig()
	logCfg.Level = zap.NewAtomicLevelAt(zapLvl)
	zapLog, err := logCfg.Build()
	if err != nil {
		envp.ExitCause(1, err, "who logs the logger errors?")
	}
	return zapr.NewLogger(zapLog)
}

// setupEnv initializes the environment from flags.
func setupEnv(globalLog logr.Logger, version string) *envp.Env {
	log := globalLog.WithName("setup")
	if *binDir == "" {
		dataDir, err := store.DefaultStoreDir()
		if err != nil {
			envp.ExitCause(1, err, "unable to deterimine default binaries directory (use --bin-dir to manually override)")
		}

		*binDir = dataDir
	}
	log.V(1).Info("using binaries directory", "dir", *binDir)

	env := &envp.Env{
		Log: globalLog,
		Client: &remote.Client{
			Log:    globalLog.WithName("storage-client"),
			Bucket: *remoteBucket,
			Server: *remoteServer,
		},
		VerifySum:     *verify,
		ForceDownload: *force,
		NoDownload:    *installedOnly,
		Platform: versions.PlatformItem{
			Platform: versions.Platform{
				OS:   *targetOS,
				Arch: *targetArch,
			},
		},
		FS:    afero.Afero{Fs: afero.NewOsFs()},
		Store: store.NewAt(*binDir),
		Out:   os.Stdout,
	}

	switch version {
	case "", "latest":
		env.Version = versions.LatestVersion
	case "latest-on-disk":
		// we sort by version, latest first, so this'll give us the latest on
		// disk (as per the contract from env.List & store.List)
		env.Version = versions.AnyVersion
		env.NoDownload = true
	default:
		var err error
		env.Version, err = versions.FromExpr(version)
		if err != nil {
			envp.ExitCause(1, err, "version be a valid version, or simply 'latest' or 'latest-on-disk'")
		}
	}

	env.CheckCoherence()

	return env
}

func main() {
	// exit with appropriate error codes -- this should be the first defer so
	// that it's the last one executed.
	defer envp.HandleExitWithCode()

	// set up flags
	flag.Usage = func() {
		name := os.Args[0]
		fmt.Fprintf(os.Stderr, "Usage: %s [FLAGS] use|list|cleanup|sideload [VERSION]\n", name)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr,
			`
Note: this command is currently alpha, and the usage/behavior may change from release to release.

Examples:

	# download the latest envtest, and print out info about it
	%[1]s use

	# download the latest 1.19 envtest, and print out the path
	%[1]s use -p path 1.19.x!

	# switch to the most recent 1.21 envtest on disk
	source <(%[1]s use -i -p env 1.21.x)

	# list all available local versions for darwin/amd64
	%[1]s list -i --os darwin --arch amd64

	# remove all versions older than 1.16 from disk
	%[1]s cleanup <1.16

	# use the value from $KUBEBUILDER_ASSETS if set, otherwise follow the normal
	# logic for 'use'
	%[1]s --use-env

	# use the value from $KUBEBUILDER_ASSETS if set, otherwise use the latest
	# installed version
	%[1]s use -i --use-env

	# sideload a pre-downloaded tarball as Kubernetes 1.16.2 into our store
	%[1]s sideload 1.16.2 < downloaded-envtest.tar.gz

Commands:

	use:
		get information for the requested version, downloading it if necessary and allowed.
		Needs a concrete platform (no wildcards), but wilcard versions are supported.

	list:
		list installed *and* available versions matching the given version & platform.
		May have wildcard versions *and* platforms.
		If the -i flag is passed, only installed versions are listed.

	cleanup:
		remove all versions matching the given version & platform selector.
		May have wildcard versions *and* platforms.

	sideload:
		reads a .tar.gz file from stdin and expand it into the store.
		must have a concrete version and platform.

Versions:

	Versions take the form of a small subset of semver selectors.

	Basic semver whole versions are accepted: X.Y.Z.
	Z may also be '*' or 'x' to match a wildcard.
	You may also just write X.Y, which means X.Y.*.

	A version may be prefixed with '~' to match the most recent Z release
	in the given Y release ( [X.Y.Z, X.Y+1.0) ).

	Finally, you may suffix the version with '!' to force checking the
	remote API server for the latest version.

	For example:

		1.16.x / 1.16.* / 1.16 # any 1.16 version
		~1.19.3                # any 1.19 version that's at least 1.19.3
		<1.17                  # any release 1.17.x or below
		1.22.x!                # the latest one 1.22 release available remotely

Output:

	The fetch & switch commands respect the --print, -p flag.

	overview: human readable information
	path: print out the path, by itself
	env: print out the path in a form that can be sourced to use that version with envtest

	Other command have human-readable output formats only.

Environment Variables:

	KUBEBUILDER_ASSETS:
		--use-env will check this, and '-p/--print env' will return this.
		If --use-env is true and this is set, we won't check our store
		for versions -- we'll just immediately return whatever's in
		this env var.

	%[2]s:
		will switch the default of -i/--installed to true if set to any value

	%[3]s:
		will switch the default of --use-env to true if set to any value

`, name, envNoDownload, envUseEnv)
	}
	flag.CommandLine.AddGoFlag(&goflag.Flag{Name: "v", Usage: "logging level", Value: &zapLvl})
	flag.VarP(&printFormat, "print", "p", "what info to print after fetch-style commands (overview, path, env)")
	needHelp := flag.Bool("help", false, "print out this help text") // register help so that we don't get an error at the end
	flag.Parse()

	if *needHelp {
		flag.Usage()
		envp.Exit(2, "")
	}

	// check our argument count
	if numArgs := flag.NArg(); numArgs < 1 || numArgs > 2 {
		flag.Usage()
		envp.Exit(2, "please specify a command to use, and optionally a version selector")
	}

	// set up logging
	globalLog := setupLogging()

	// set up the environment
	var version string
	if flag.NArg() > 1 {
		version = flag.Arg(1)
	}
	env := setupEnv(globalLog, version)

	// perform our main set of actions
	switch action := flag.Arg(0); action {
	case "use":
		workflows.Use{
			UseEnv:      *useEnv,
			PrintFormat: printFormat,
			AssetsPath:  os.Getenv("KUBEBUILDER_ASSETS"),
		}.Do(env)
	case "list":
		workflows.List{}.Do(env)
	case "cleanup":
		workflows.Cleanup{}.Do(env)
	case "sideload":
		workflows.Sideload{
			Input:       os.Stdin,
			PrintFormat: printFormat,
		}.Do(env)
	default:
		flag.Usage()
		envp.Exit(2, "unknown action %q", action)
	}
}
