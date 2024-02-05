# Guide to release a new version

1. Update the `buildVersion` constant in [connection.go](https://github.com/rabbitmq/amqp091-go/blob/4886c35d10b273bd374e3ed2356144ad41d27940/connection.go#L31)
2. Commit and push. Include the version in the commit message e.g. [this commit](https://github.com/rabbitmq/amqp091-go/commit/52ce2efd03c53dcf77d5496977da46840e9abd24)
3. Create a new [GitHub Release](https://github.com/rabbitmq/amqp091-go/releases). Create a new tag as `v<MAJOR>.<MINOR>.<PATCH>`
   1. Use auto-generate release notes feature in GitHub
4. Generate the change log, see [Changelog Generation](#changelog-generation)
5. Review the changelog. Watch out for issues closed as "not-fixed" or without a PR
6. Commit and Push. Pro-tip: include `[skip ci]` in the commit message to skip the CI run, since it's only documentation
7. Send an announcement to the mailing list. Take inspiration from [this message](https://groups.google.com/g/rabbitmq-users/c/EBGYGOWiSgs/m/0sSFuAGICwAJ)

## Changelog Generation

```
github_changelog_generator --token GITHUB-TOKEN -u rabbitmq -p amqp091-go --no-unreleased --release-branch main
```
