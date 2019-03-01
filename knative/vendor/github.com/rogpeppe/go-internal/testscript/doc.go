// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package testscript provides support for defining filesystem-based tests by
creating scripts in a directory.

To invoke the tests, call testscript.Run. For example:

	func TestFoo(t *testing.T) {
		testscript.Run(t, testscript.Params{
			Dir: "testdata",
		})
	}

A testscript directory holds test scripts *.txt run during 'go test'.
Each script defines a subtest; the exact set of allowable commands in a
script are defined by the parameters passed to the Run function.
To run a specific script foo.txt

	go test cmd/go -run=TestName/^foo$

where TestName is the name of the test that Run is called from.

To define an executable command (or several) that can be run as part of the script,
call RunMain with the functions that implement the command's functionality.
The command functions will be called in a separate process, so are
free to mutate global variables without polluting the top level test binary.

	func TestMain(m *testing.M) {
		os.Exit(testscript.RunMain(m, map[string] func() int{
			"testscript": testscriptMain,
		}))
	}

In general script files should have short names: a few words, not whole sentences.
The first word should be the general category of behavior being tested,
often the name of a subcommand to be tested or a concept (vendor, pattern).

Each script is a text archive (go doc github.com/rogpeppe/go-internal/txtar).
The script begins with an actual command script to run
followed by the content of zero or more supporting files to
create in the script's temporary file system before it starts executing.

As an example:

	# hello world
	exec cat hello.text
	stdout 'hello world\n'
	! stderr .

	-- hello.text --
	hello world

Each script runs in a fresh temporary work directory tree, available to scripts as $WORK.
Scripts also have access to these other environment variables:

	HOME=/no-home
	PATH=<actual PATH>
	TMPDIR=$WORK/tmp
	devnull=<value of os.DevNull>
	goversion=<current Go version; for example, 1.12>

The environment variable $exe (lowercase) is an empty string on most
systems, ".exe" on Windows.

The script's supporting files are unpacked relative to $WORK
and then the script begins execution in that
directory as well. Thus the example above runs in $WORK
with $WORK/hello.txt containing the listed contents.

The lines at the top of the script are a sequence of commands to be
executed by a small script engine in the testscript package (not the system
shell).  The script stops and the overall test fails if any particular
command fails.

Each line is parsed into a sequence of space-separated command words,
with environment variable expansion and # marking an end-of-line comment.
Adding single quotes around text keeps spaces in that text from being
treated as word separators and also disables environment variable
expansion.  Inside a single-quoted block of text, a repeated single
quote indicates a literal single quote, as in:

	'Don''t communicate by sharing memory.'

A line beginning with # is a comment and conventionally explains what is
being done or tested at the start of a new phase in the script.

A special form of environment variable syntax can be used to quote
regexp metacharacters inside environment variables. The "@R" suffix
is special, and indicates that the variable should be quoted.

	${VAR@R}

The command prefix ! indicates that the command on the rest of the line
(typically go or a matching predicate) must fail, not succeed. Only certain
commands support this prefix. They are indicated below by [!] in the synopsis.

The command prefix [cond] indicates that the command on the rest of the line
should only run when the condition is satisfied. The predefined conditions are:

 - [short] for testing.Short()
 - [net] for whether the external network can be used
 - [link] for whether the OS has hard link support
 - [symlink] for whether the OS has symbolic link support
 - [exec:prog] for whether prog is available for execution (found by exec.LookPath)

A condition can be negated: [!short] means to run the rest of the line
when testing.Short() is false.

Additional conditions can be added by passing a function to Params.Condition.

The predefined commands are:

- cd dir
  Change to the given directory for future commands.

- chmod mode file

  Change the permissions of file or directory to the given octal mode (000 to 777).

- cmp file1 file2
  Check that the named files have the same content.
  By convention, file1 is the actual data and file2 the expected data.
  File1 can be "stdout" or "stderr" to use the standard output or standard error
  from the most recent exec or wait command.
  (If the files have differing content, the failure prints a diff.)

- cmpenv file1 file2
  Like cmp, but environment variables in file2 are substituted before the
  comparison. For example, $GOOS is replaced by the target GOOS.

- cp src... dst
  Copy the listed files to the target file or existing directory.

- env [key=value...]
  With no arguments, print the environment (useful for debugging).
  Otherwise add the listed key=value pairs to the environment.

- [!] exec program [args...] [&]
  Run the given executable program with the arguments.
  It must (or must not) succeed.
  Note that 'exec' does not terminate the script (unlike in Unix shells).

  If the last token is '&', the program executes in the background. The standard
  output and standard error of the previous command is cleared, but the output
  of the background process is buffered — and checking of its exit status is
  delayed — until the next call to 'wait', 'skip', or 'stop' or the end of the
  test. At the end of the test, any remaining background processes are
  terminated using os.Interrupt (if supported) or os.Kill.

  Standard input can be provided using the stdin command; this will be
  cleared after exec has been called.

- [!] exists [-readonly] file...
  Each of the listed files or directories must (or must not) exist.
  If -readonly is given, the files or directories must be unwritable.

- [!] grep [-count=N] pattern file
  The file's content must (or must not) match the regular expression pattern.
  For positive matches, -count=N specifies an exact number of matches to require.

- mkdir path...
  Create the listed directories, if they do not already exists.

- rm file...
  Remove the listed files or directories.

- skip [message]
  Mark the test skipped, including the message if given.

- stdin file
  Set the standard input for the next exec command to the contents of the given file.

- [!] stderr [-count=N] pattern
  Apply the grep command (see above) to the standard error
  from the most recent exec or wait command.

- [!] stdout [-count=N] pattern
  Apply the grep command (see above) to the standard output
  from the most recent exec or wait command.

- stop [message]
  Stop the test early (marking it as passing), including the message if given.

- symlink file -> target
  Create file as a symlink to target. The -> (like in ls -l output) is required.

- wait
  Wait for all 'exec' and 'go' commands started in the background (with the '&'
  token) to exit, and display success or failure status for them.
  After a call to wait, the 'stderr' and 'stdout' commands will apply to the
  concatenation of the corresponding streams of the background commands,
  in the order in which those commands were started.

When TestScript runs a script and the script fails, by default TestScript shows
the execution of the most recent phase of the script (since the last # comment)
and only shows the # comments for earlier phases. For example, here is a
multi-phase script with a bug in it (TODO: make this example less go-command
specific):

	# GOPATH with p1 in d2, p2 in d2
	env GOPATH=$WORK/d1${:}$WORK/d2

	# build & install p1
	env
	go install -i p1
	! stale p1
	! stale p2

	# modify p2 - p1 should appear stale
	cp $WORK/p2x.go $WORK/d2/src/p2/p2.go
	stale p1 p2

	# build & install p1 again
	go install -i p11
	! stale p1
	! stale p2

	-- $WORK/d1/src/p1/p1.go --
	package p1
	import "p2"
	func F() { p2.F() }
	-- $WORK/d2/src/p2/p2.go --
	package p2
	func F() {}
	-- $WORK/p2x.go --
	package p2
	func F() {}
	func G() {}

The bug is that the final phase installs p11 instead of p1. The test failure looks like:

	$ go test -run=Script
	--- FAIL: TestScript (3.75s)
	    --- FAIL: TestScript/install_rebuild_gopath (0.16s)
	        script_test.go:223:
	            # GOPATH with p1 in d2, p2 in d2 (0.000s)
	            # build & install p1 (0.087s)
	            # modify p2 - p1 should appear stale (0.029s)
	            # build & install p1 again (0.022s)
	            > go install -i p11
	            [stderr]
	            can't load package: package p11: cannot find package "p11" in any of:
	            	/Users/rsc/go/src/p11 (from $GOROOT)
	            	$WORK/d1/src/p11 (from $GOPATH)
	            	$WORK/d2/src/p11
	            [exit status 1]
	            FAIL: unexpected go command failure

	        script_test.go:73: failed at testdata/script/install_rebuild_gopath.txt:15 in $WORK/gopath/src

	FAIL
	exit status 1
	FAIL	cmd/go	4.875s
	$

Note that the commands in earlier phases have been hidden, so that the relevant
commands are more easily found, and the elapsed time for a completed phase
is shown next to the phase heading. To see the entire execution, use "go test -v",
which also adds an initial environment dump to the beginning of the log.

Note also that in reported output, the actual name of the per-script temporary directory
has been consistently replaced with the literal string $WORK.

If Params.TestWork is true, it causes each test to log the name of its $WORK directory and other
environment variable settings and also to leave that directory behind when it exits,
for manual debugging of failing tests:

	$ go test -run=Script -work
	--- FAIL: TestScript (3.75s)
	    --- FAIL: TestScript/install_rebuild_gopath (0.16s)
	        script_test.go:223:
	            WORK=/tmp/cmd-go-test-745953508/script-install_rebuild_gopath
	            GOARCH=
	            GOCACHE=/Users/rsc/Library/Caches/go-build
	            GOOS=
	            GOPATH=$WORK/gopath
	            GOROOT=/Users/rsc/go
	            HOME=/no-home
	            TMPDIR=$WORK/tmp
	            exe=

	            # GOPATH with p1 in d2, p2 in d2 (0.000s)
	            # build & install p1 (0.085s)
	            # modify p2 - p1 should appear stale (0.030s)
	            # build & install p1 again (0.019s)
	            > go install -i p11
	            [stderr]
	            can't load package: package p11: cannot find package "p11" in any of:
	            	/Users/rsc/go/src/p11 (from $GOROOT)
	            	$WORK/d1/src/p11 (from $GOPATH)
	            	$WORK/d2/src/p11
	            [exit status 1]
	            FAIL: unexpected go command failure

	        script_test.go:73: failed at testdata/script/install_rebuild_gopath.txt:15 in $WORK/gopath/src

	FAIL
	exit status 1
	FAIL	cmd/go	4.875s
	$

	$ WORK=/tmp/cmd-go-test-745953508/script-install_rebuild_gopath
	$ cd $WORK/d1/src/p1
	$ cat p1.go
	package p1
	import "p2"
	func F() { p2.F() }
	$
*/
package testscript
