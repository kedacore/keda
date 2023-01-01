# Azure/go-amqp Contributing Guide

Thank you for your interest in contributing to go-amqp.

- For reporting bugs, requesting features, or asking for support, please file an issue in the [issues](https://github.com/Azure/go-amqp/issues) section of the project.

- If you would like to become an active contributor to this project please follow the instructions provided in [Microsoft Azure Projects Contribution Guidelines](https://azure.github.io/azure-sdk/policies_opensource.html).

- To make code changes, or contribute something new, please follow the [GitHub Forks / Pull requests model](https://help.github.com/articles/fork-a-repo/): Fork the repo, make the change and propose it back by submitting a pull request.

## Pull Requests

- **DO** follow the API design and implementation [Go Guidelines](https://azure.github.io/azure-sdk/golang_introduction.html).
  - When submitting large changes or features, **DO** have an issue or spec doc that describes the design, usage, and motivating scenario.
- **DO** submit all code changes via pull requests (PRs) rather than through a direct commit. PRs will be reviewed and potentially merged by the repo maintainers after a peer review that includes at least one maintainer.
- **DO** review your own PR to make sure there are no unintended changes or commits before submitting it.
- **DO NOT** submit "work in progress" PRs. A PR should only be submitted when it is considered ready for review and subsequent merging by the contributor.
  - If the change is work-in-progress or an experiment, **DO** start off as a temporary draft PR.
- **DO** give PRs short-but-descriptive names (e.g. "Improve code coverage for sender by 10%", not "Fix #1234") and add a description which explains why the change is being made.
- **DO** refer to any relevant issues, and include [keywords](https://help.github.com/articles/closing-issues-via-commit-messages/) that automatically close issues when the PR is merged.
- **DO** tag any users that should know about and/or review the change.
- **DO** ensure each commit successfully builds. The entire PR must pass all tests in the Continuous Integration (CI) system before it'll be merged.
- **DO** address PR feedback in an additional commit(s) rather than amending the existing commits, and only rebase/squash them when necessary. This makes it easier for reviewers to track changes.
- **DO** assume that ["Squash and Merge"](https://github.com/blog/2141-squash-your-commits) will be used to merge your commit unless you request otherwise in the PR.
- **DO NOT** mix independent, unrelated changes in one PR. Separate real product/test code changes from larger code formatting/dead code removal changes. Separate unrelated fixes into separate PRs, especially if they are in different modules or files that otherwise wouldn't be changed.
- **DO** comment your code focusing on "why", where necessary. Otherwise, aim to keep it self-documenting with appropriate names and style.
- **DO** add [GoDoc style comments](https://azure.github.io/azure-sdk/golang_introduction.html#documentation-style) when adding new APIs or modifying header files.
- **DO** make sure there are no typos or spelling errors, especially in user-facing documentation.
- **DO** verify if your changes have impact elsewhere. For instance, do you need to update other docs or exiting markdown files that might be impacted?
- **DO** add relevant unit tests to ensure CI will catch future regressions.

## Merging Pull Requests (for project contributors with write access)

- **DO** use ["Squash and Merge"](https://github.com/blog/2141-squash-your-commits) by default for individual contributions unless requested by the PR author.
  Do so, even if the PR contains only one commit. It creates a simpler history than "Create a Merge Commit".
  Reasons that PR authors may request "Merge and Commit" may include (but are not limited to):

  - The change is easier to understand as a series of focused commits. Each commit in the series must be buildable so as not to break `git bisect`.
  - Contributor is using an e-mail address other than the primary GitHub address and wants that preserved in the history. Contributor must be willing to squash
    the commits manually before acceptance.

## Developer Guide

### Logging

To enable debug logging, build with `-tags debug`. This enables debug level 1 by default. You can increase the level by setting the `DEBUG_LEVEL` environment variable to 2 or higher. (Debug logging is disabled entirely without `-tags debug`, regardless of `DEBUG_LEVEL` setting.)

To add additional logging, use the `debug.Log(level int, format string, v ...any)` function, which is similar to `fmt.Printf` but takes a level as its first argument.

### Packet Capture

Wireshark can be very helpful in diagnosing interactions between client and server. If the connection is not encrypted Wireshark can natively decode AMQP 1.0. If the connection is encrypted with TLS you'll need to log out the keys.

Example of logging the TLS keys:

```go
// Create the file
f, err := os.OpenFile("key.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

// Configure TLS
tlsConfig := &tls.Config{
    KeyLogWriter: f,
}

// Dial the host
const host = "my.amqp.server"
conn, err := tls.Dial("tcp", host+":5671", tlsConfig)

// Create the connections
client, err := amqp.New(conn,
    amqp.ConnSASLPlain("username", "password"),
    amqp.ConnServerHostname(host),
)
```

You'll need to configure Wireshark to read the key.log file in Preferences > Protocols > SSL > (Pre)-Master-Secret log filename.
