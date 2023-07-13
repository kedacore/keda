package scalers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const ghLoadCount = 2 // the size of the pretend pool completed of job requests

var testGhWorkflowResponse = `{"total_count":1,"workflow_runs":[{"id":30433642,"name":"Build","node_id":"MDEyOldvcmtmbG93IFJ1bjI2OTI4OQ==","check_suite_id":42,"check_suite_node_id":"MDEwOkNoZWNrU3VpdGU0Mg==","head_branch":"master","head_sha":"acb5820ced9479c074f688cc328bf03f341a511d","run_number":562,"event":"push","status":"queued","conclusion":null,"workflow_id":159038,"url":"https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642","html_url":"https://github.com/octo-org/octo-repo/actions/runs/30433642","pull_requests":[],"created_at":"2020-01-22T19:33:08Z","updated_at":"2020-01-22T19:33:08Z","actor":{"login":"octocat","id":1,"node_id":"MDQ6VXNlcjE=","avatar_url":"https://github.com/images/error/octocat_happy.gif","gravatar_id":"","url":"https://api.github.com/users/octocat","html_url":"https://github.com/octocat","followers_url":"https://api.github.com/users/octocat/followers","following_url":"https://api.github.com/users/octocat/following{/other_user}","gists_url":"https://api.github.com/users/octocat/gists{/gist_id}","starred_url":"https://api.github.com/users/octocat/starred{/owner}{/repo}","subscriptions_url":"https://api.github.com/users/octocat/subscriptions","organizations_url":"https://api.github.com/users/octocat/orgs","repos_url":"https://api.github.com/users/octocat/repos","events_url":"https://api.github.com/users/octocat/events{/privacy}","received_events_url":"https://api.github.com/users/octocat/received_events","type":"User","site_admin":false},"run_attempt":1,"run_started_at":"2020-01-22T19:33:08Z","triggering_actor":{"login":"octocat","id":1,"node_id":"MDQ6VXNlcjE=","avatar_url":"https://github.com/images/error/octocat_happy.gif","gravatar_id":"","url":"https://api.github.com/users/octocat","html_url":"https://github.com/octocat","followers_url":"https://api.github.com/users/octocat/followers","following_url":"https://api.github.com/users/octocat/following{/other_user}","gists_url":"https://api.github.com/users/octocat/gists{/gist_id}","starred_url":"https://api.github.com/users/octocat/starred{/owner}{/repo}","subscriptions_url":"https://api.github.com/users/octocat/subscriptions","organizations_url":"https://api.github.com/users/octocat/orgs","repos_url":"https://api.github.com/users/octocat/repos","events_url":"https://api.github.com/users/octocat/events{/privacy}","received_events_url":"https://api.github.com/users/octocat/received_events","type":"User","site_admin":false},"jobs_url":"https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642/jobs","logs_url":"https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642/logs","check_suite_url":"https://api.github.com/repos/octo-org/octo-repo/check-suites/414944374","artifacts_url":"https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642/artifacts","cancel_url":"https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642/cancel","rerun_url":"https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642/rerun","workflow_url":"https://api.github.com/repos/octo-org/octo-repo/actions/workflows/159038","head_commit":{"id":"acb5820ced9479c074f688cc328bf03f341a511d","tree_id":"d23f6eedb1e1b9610bbc754ddb5197bfe7271223","message":"Create linter.yaml","timestamp":"2020-01-22T19:33:05Z","author":{"name":"Octo Cat","email":"octocat@github.com"},"committer":{"name":"GitHub","email":"noreply@github.com"}},"repository":{"id":1296269,"node_id":"MDEwOlJlcG9zaXRvcnkxMjk2MjY5","name":"Hello-World","full_name":"octocat/Hello-World","owner":{"login":"octocat","id":1,"node_id":"MDQ6VXNlcjE=","avatar_url":"https://github.com/images/error/octocat_happy.gif","gravatar_id":"","url":"https://api.github.com/users/octocat","html_url":"https://github.com/octocat","followers_url":"https://api.github.com/users/octocat/followers","following_url":"https://api.github.com/users/octocat/following{/other_user}","gists_url":"https://api.github.com/users/octocat/gists{/gist_id}","starred_url":"https://api.github.com/users/octocat/starred{/owner}{/repo}","subscriptions_url":"https://api.github.com/users/octocat/subscriptions","organizations_url":"https://api.github.com/users/octocat/orgs","repos_url":"https://api.github.com/users/octocat/repos","events_url":"https://api.github.com/users/octocat/events{/privacy}","received_events_url":"https://api.github.com/users/octocat/received_events","type":"User","site_admin":false},"private":false,"html_url":"https://github.com/octocat/Hello-World","description":"This your first repo!","fork":false,"url":"https://api.github.com/repos/octocat/Hello-World","archive_url":"https://api.github.com/repos/octocat/Hello-World/{archive_format}{/ref}","assignees_url":"https://api.github.com/repos/octocat/Hello-World/assignees{/user}","blobs_url":"https://api.github.com/repos/octocat/Hello-World/git/blobs{/sha}","branches_url":"https://api.github.com/repos/octocat/Hello-World/branches{/branch}","collaborators_url":"https://api.github.com/repos/octocat/Hello-World/collaborators{/collaborator}","comments_url":"https://api.github.com/repos/octocat/Hello-World/comments{/number}","commits_url":"https://api.github.com/repos/octocat/Hello-World/commits{/sha}","compare_url":"https://api.github.com/repos/octocat/Hello-World/compare/{base}...{head}","contents_url":"https://api.github.com/repos/octocat/Hello-World/contents/{+path}","contributors_url":"https://api.github.com/repos/octocat/Hello-World/contributors","deployments_url":"https://api.github.com/repos/octocat/Hello-World/deployments","downloads_url":"https://api.github.com/repos/octocat/Hello-World/downloads","events_url":"https://api.github.com/repos/octocat/Hello-World/events","forks_url":"https://api.github.com/repos/octocat/Hello-World/forks","git_commits_url":"https://api.github.com/repos/octocat/Hello-World/git/commits{/sha}","git_refs_url":"https://api.github.com/repos/octocat/Hello-World/git/refs{/sha}","git_tags_url":"https://api.github.com/repos/octocat/Hello-World/git/tags{/sha}","git_url":"git:github.com/octocat/Hello-World.git","issue_comment_url":"https://api.github.com/repos/octocat/Hello-World/issues/comments{/number}","issue_events_url":"https://api.github.com/repos/octocat/Hello-World/issues/events{/number}","issues_url":"https://api.github.com/repos/octocat/Hello-World/issues{/number}","keys_url":"https://api.github.com/repos/octocat/Hello-World/keys{/key_id}","labels_url":"https://api.github.com/repos/octocat/Hello-World/labels{/name}","languages_url":"https://api.github.com/repos/octocat/Hello-World/languages","merges_url":"https://api.github.com/repos/octocat/Hello-World/merges","milestones_url":"https://api.github.com/repos/octocat/Hello-World/milestones{/number}","notifications_url":"https://api.github.com/repos/octocat/Hello-World/notifications{?since,all,participating}","pulls_url":"https://api.github.com/repos/octocat/Hello-World/pulls{/number}","releases_url":"https://api.github.com/repos/octocat/Hello-World/releases{/id}","ssh_url":"git@github.com:octocat/Hello-World.git","stargazers_url":"https://api.github.com/repos/octocat/Hello-World/stargazers","statuses_url":"https://api.github.com/repos/octocat/Hello-World/statuses/{sha}","subscribers_url":"https://api.github.com/repos/octocat/Hello-World/subscribers","subscription_url":"https://api.github.com/repos/octocat/Hello-World/subscription","tags_url":"https://api.github.com/repos/octocat/Hello-World/tags","teams_url":"https://api.github.com/repos/octocat/Hello-World/teams","trees_url":"https://api.github.com/repos/octocat/Hello-World/git/trees{/sha}","hooks_url":"http://api.github.com/repos/octocat/Hello-World/hooks"},"head_repository":{"id":217723378,"node_id":"MDEwOlJlcG9zaXRvcnkyMTc3MjMzNzg=","name":"octo-repo","full_name":"octo-org/octo-repo","private":true,"owner":{"login":"octocat","id":1,"node_id":"MDQ6VXNlcjE=","avatar_url":"https://github.com/images/error/octocat_happy.gif","gravatar_id":"","url":"https://api.github.com/users/octocat","html_url":"https://github.com/octocat","followers_url":"https://api.github.com/users/octocat/followers","following_url":"https://api.github.com/users/octocat/following{/other_user}","gists_url":"https://api.github.com/users/octocat/gists{/gist_id}","starred_url":"https://api.github.com/users/octocat/starred{/owner}{/repo}","subscriptions_url":"https://api.github.com/users/octocat/subscriptions","organizations_url":"https://api.github.com/users/octocat/orgs","repos_url":"https://api.github.com/users/octocat/repos","events_url":"https://api.github.com/users/octocat/events{/privacy}","received_events_url":"https://api.github.com/users/octocat/received_events","type":"User","site_admin":false},"html_url":"https://github.com/octo-org/octo-repo","description":null,"fork":false,"url":"https://api.github.com/repos/octo-org/octo-repo","forks_url":"https://api.github.com/repos/octo-org/octo-repo/forks","keys_url":"https://api.github.com/repos/octo-org/octo-repo/keys{/key_id}","collaborators_url":"https://api.github.com/repos/octo-org/octo-repo/collaborators{/collaborator}","teams_url":"https://api.github.com/repos/octo-org/octo-repo/teams","hooks_url":"https://api.github.com/repos/octo-org/octo-repo/hooks","issue_events_url":"https://api.github.com/repos/octo-org/octo-repo/issues/events{/number}","events_url":"https://api.github.com/repos/octo-org/octo-repo/events","assignees_url":"https://api.github.com/repos/octo-org/octo-repo/assignees{/user}","branches_url":"https://api.github.com/repos/octo-org/octo-repo/branches{/branch}","tags_url":"https://api.github.com/repos/octo-org/octo-repo/tags","blobs_url":"https://api.github.com/repos/octo-org/octo-repo/git/blobs{/sha}","git_tags_url":"https://api.github.com/repos/octo-org/octo-repo/git/tags{/sha}","git_refs_url":"https://api.github.com/repos/octo-org/octo-repo/git/refs{/sha}","trees_url":"https://api.github.com/repos/octo-org/octo-repo/git/trees{/sha}","statuses_url":"https://api.github.com/repos/octo-org/octo-repo/statuses/{sha}","languages_url":"https://api.github.com/repos/octo-org/octo-repo/languages","stargazers_url":"https://api.github.com/repos/octo-org/octo-repo/stargazers","contributors_url":"https://api.github.com/repos/octo-org/octo-repo/contributors","subscribers_url":"https://api.github.com/repos/octo-org/octo-repo/subscribers","subscription_url":"https://api.github.com/repos/octo-org/octo-repo/subscription","commits_url":"https://api.github.com/repos/octo-org/octo-repo/commits{/sha}","git_commits_url":"https://api.github.com/repos/octo-org/octo-repo/git/commits{/sha}","comments_url":"https://api.github.com/repos/octo-org/octo-repo/comments{/number}","issue_comment_url":"https://api.github.com/repos/octo-org/octo-repo/issues/comments{/number}","contents_url":"https://api.github.com/repos/octo-org/octo-repo/contents/{+path}","compare_url":"https://api.github.com/repos/octo-org/octo-repo/compare/{base}...{head}","merges_url":"https://api.github.com/repos/octo-org/octo-repo/merges","archive_url":"https://api.github.com/repos/octo-org/octo-repo/{archive_format}{/ref}","downloads_url":"https://api.github.com/repos/octo-org/octo-repo/downloads","issues_url":"https://api.github.com/repos/octo-org/octo-repo/issues{/number}","pulls_url":"https://api.github.com/repos/octo-org/octo-repo/pulls{/number}","milestones_url":"https://api.github.com/repos/octo-org/octo-repo/milestones{/number}","notifications_url":"https://api.github.com/repos/octo-org/octo-repo/notifications{?since,all,participating}","labels_url":"https://api.github.com/repos/octo-org/octo-repo/labels{/name}","releases_url":"https://api.github.com/repos/octo-org/octo-repo/releases{/id}","deployments_url":"https://api.github.com/repos/octo-org/octo-repo/deployments"}}]}`
var ghDeadJob = `{"id":30433642,"name":"Build","node_id":"MDEyOldvcmtmbG93IFJ1bjI2OTI4OQ==","check_suite_id":42,"check_suite_node_id":"MDEwOkNoZWNrU3VpdGU0Mg==","head_branch":"master","head_sha":"acb5820ced9479c074f688cc328bf03f341a511d","run_number":562,"event":"push","status":"completed","conclusion":null,"workflow_id":159038,"url":"https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642","html_url":"https://github.com/octo-org/octo-repo/actions/runs/30433642","pull_requests":[],"created_at":"2020-01-22T19:33:08Z","updated_at":"2020-01-22T19:33:08Z","actor":{"login":"octocat","id":1,"node_id":"MDQ6VXNlcjE=","avatar_url":"https://github.com/images/error/octocat_happy.gif","gravatar_id":"","url":"https://api.github.com/users/octocat","html_url":"https://github.com/octocat","followers_url":"https://api.github.com/users/octocat/followers","following_url":"https://api.github.com/users/octocat/following{/other_user}","gists_url":"https://api.github.com/users/octocat/gists{/gist_id}","starred_url":"https://api.github.com/users/octocat/starred{/owner}{/repo}","subscriptions_url":"https://api.github.com/users/octocat/subscriptions","organizations_url":"https://api.github.com/users/octocat/orgs","repos_url":"https://api.github.com/users/octocat/repos","events_url":"https://api.github.com/users/octocat/events{/privacy}","received_events_url":"https://api.github.com/users/octocat/received_events","type":"User","site_admin":false},"run_attempt":1,"run_started_at":"2020-01-22T19:33:08Z","triggering_actor":{"login":"octocat","id":1,"node_id":"MDQ6VXNlcjE=","avatar_url":"https://github.com/images/error/octocat_happy.gif","gravatar_id":"","url":"https://api.github.com/users/octocat","html_url":"https://github.com/octocat","followers_url":"https://api.github.com/users/octocat/followers","following_url":"https://api.github.com/users/octocat/following{/other_user}","gists_url":"https://api.github.com/users/octocat/gists{/gist_id}","starred_url":"https://api.github.com/users/octocat/starred{/owner}{/repo}","subscriptions_url":"https://api.github.com/users/octocat/subscriptions","organizations_url":"https://api.github.com/users/octocat/orgs","repos_url":"https://api.github.com/users/octocat/repos","events_url":"https://api.github.com/users/octocat/events{/privacy}","received_events_url":"https://api.github.com/users/octocat/received_events","type":"User","site_admin":false},"jobs_url":"https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642/jobs","logs_url":"https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642/logs","check_suite_url":"https://api.github.com/repos/octo-org/octo-repo/check-suites/414944374","artifacts_url":"https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642/artifacts","cancel_url":"https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642/cancel","rerun_url":"https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642/rerun","workflow_url":"https://api.github.com/repos/octo-org/octo-repo/actions/workflows/159038","head_commit":{"id":"acb5820ced9479c074f688cc328bf03f341a511d","tree_id":"d23f6eedb1e1b9610bbc754ddb5197bfe7271223","message":"Create linter.yaml","timestamp":"2020-01-22T19:33:05Z","author":{"name":"Octo Cat","email":"octocat@github.com"},"committer":{"name":"GitHub","email":"noreply@github.com"}},"repository":{"id":1296269,"node_id":"MDEwOlJlcG9zaXRvcnkxMjk2MjY5","name":"Hello-World","full_name":"octocat/Hello-World","owner":{"login":"octocat","id":1,"node_id":"MDQ6VXNlcjE=","avatar_url":"https://github.com/images/error/octocat_happy.gif","gravatar_id":"","url":"https://api.github.com/users/octocat","html_url":"https://github.com/octocat","followers_url":"https://api.github.com/users/octocat/followers","following_url":"https://api.github.com/users/octocat/following{/other_user}","gists_url":"https://api.github.com/users/octocat/gists{/gist_id}","starred_url":"https://api.github.com/users/octocat/starred{/owner}{/repo}","subscriptions_url":"https://api.github.com/users/octocat/subscriptions","organizations_url":"https://api.github.com/users/octocat/orgs","repos_url":"https://api.github.com/users/octocat/repos","events_url":"https://api.github.com/users/octocat/events{/privacy}","received_events_url":"https://api.github.com/users/octocat/received_events","type":"User","site_admin":false},"private":false,"html_url":"https://github.com/octocat/Hello-World","description":"This your first repo!","fork":false,"url":"https://api.github.com/repos/octocat/Hello-World","archive_url":"https://api.github.com/repos/octocat/Hello-World/{archive_format}{/ref}","assignees_url":"https://api.github.com/repos/octocat/Hello-World/assignees{/user}","blobs_url":"https://api.github.com/repos/octocat/Hello-World/git/blobs{/sha}","branches_url":"https://api.github.com/repos/octocat/Hello-World/branches{/branch}","collaborators_url":"https://api.github.com/repos/octocat/Hello-World/collaborators{/collaborator}","comments_url":"https://api.github.com/repos/octocat/Hello-World/comments{/number}","commits_url":"https://api.github.com/repos/octocat/Hello-World/commits{/sha}","compare_url":"https://api.github.com/repos/octocat/Hello-World/compare/{base}...{head}","contents_url":"https://api.github.com/repos/octocat/Hello-World/contents/{+path}","contributors_url":"https://api.github.com/repos/octocat/Hello-World/contributors","deployments_url":"https://api.github.com/repos/octocat/Hello-World/deployments","downloads_url":"https://api.github.com/repos/octocat/Hello-World/downloads","events_url":"https://api.github.com/repos/octocat/Hello-World/events","forks_url":"https://api.github.com/repos/octocat/Hello-World/forks","git_commits_url":"https://api.github.com/repos/octocat/Hello-World/git/commits{/sha}","git_refs_url":"https://api.github.com/repos/octocat/Hello-World/git/refs{/sha}","git_tags_url":"https://api.github.com/repos/octocat/Hello-World/git/tags{/sha}","git_url":"git:github.com/octocat/Hello-World.git","issue_comment_url":"https://api.github.com/repos/octocat/Hello-World/issues/comments{/number}","issue_events_url":"https://api.github.com/repos/octocat/Hello-World/issues/events{/number}","issues_url":"https://api.github.com/repos/octocat/Hello-World/issues{/number}","keys_url":"https://api.github.com/repos/octocat/Hello-World/keys{/key_id}","labels_url":"https://api.github.com/repos/octocat/Hello-World/labels{/name}","languages_url":"https://api.github.com/repos/octocat/Hello-World/languages","merges_url":"https://api.github.com/repos/octocat/Hello-World/merges","milestones_url":"https://api.github.com/repos/octocat/Hello-World/milestones{/number}","notifications_url":"https://api.github.com/repos/octocat/Hello-World/notifications{?since,all,participating}","pulls_url":"https://api.github.com/repos/octocat/Hello-World/pulls{/number}","releases_url":"https://api.github.com/repos/octocat/Hello-World/releases{/id}","ssh_url":"git@github.com:octocat/Hello-World.git","stargazers_url":"https://api.github.com/repos/octocat/Hello-World/stargazers","statuses_url":"https://api.github.com/repos/octocat/Hello-World/statuses/{sha}","subscribers_url":"https://api.github.com/repos/octocat/Hello-World/subscribers","subscription_url":"https://api.github.com/repos/octocat/Hello-World/subscription","tags_url":"https://api.github.com/repos/octocat/Hello-World/tags","teams_url":"https://api.github.com/repos/octocat/Hello-World/teams","trees_url":"https://api.github.com/repos/octocat/Hello-World/git/trees{/sha}","hooks_url":"http://api.github.com/repos/octocat/Hello-World/hooks"},"head_repository":{"id":217723378,"node_id":"MDEwOlJlcG9zaXRvcnkyMTc3MjMzNzg=","name":"octo-repo","full_name":"octo-org/octo-repo","private":true,"owner":{"login":"octocat","id":1,"node_id":"MDQ6VXNlcjE=","avatar_url":"https://github.com/images/error/octocat_happy.gif","gravatar_id":"","url":"https://api.github.com/users/octocat","html_url":"https://github.com/octocat","followers_url":"https://api.github.com/users/octocat/followers","following_url":"https://api.github.com/users/octocat/following{/other_user}","gists_url":"https://api.github.com/users/octocat/gists{/gist_id}","starred_url":"https://api.github.com/users/octocat/starred{/owner}{/repo}","subscriptions_url":"https://api.github.com/users/octocat/subscriptions","organizations_url":"https://api.github.com/users/octocat/orgs","repos_url":"https://api.github.com/users/octocat/repos","events_url":"https://api.github.com/users/octocat/events{/privacy}","received_events_url":"https://api.github.com/users/octocat/received_events","type":"User","site_admin":false},"html_url":"https://github.com/octo-org/octo-repo","description":null,"fork":false,"url":"https://api.github.com/repos/octo-org/octo-repo","forks_url":"https://api.github.com/repos/octo-org/octo-repo/forks","keys_url":"https://api.github.com/repos/octo-org/octo-repo/keys{/key_id}","collaborators_url":"https://api.github.com/repos/octo-org/octo-repo/collaborators{/collaborator}","teams_url":"https://api.github.com/repos/octo-org/octo-repo/teams","hooks_url":"https://api.github.com/repos/octo-org/octo-repo/hooks","issue_events_url":"https://api.github.com/repos/octo-org/octo-repo/issues/events{/number}","events_url":"https://api.github.com/repos/octo-org/octo-repo/events","assignees_url":"https://api.github.com/repos/octo-org/octo-repo/assignees{/user}","branches_url":"https://api.github.com/repos/octo-org/octo-repo/branches{/branch}","tags_url":"https://api.github.com/repos/octo-org/octo-repo/tags","blobs_url":"https://api.github.com/repos/octo-org/octo-repo/git/blobs{/sha}","git_tags_url":"https://api.github.com/repos/octo-org/octo-repo/git/tags{/sha}","git_refs_url":"https://api.github.com/repos/octo-org/octo-repo/git/refs{/sha}","trees_url":"https://api.github.com/repos/octo-org/octo-repo/git/trees{/sha}","statuses_url":"https://api.github.com/repos/octo-org/octo-repo/statuses/{sha}","languages_url":"https://api.github.com/repos/octo-org/octo-repo/languages","stargazers_url":"https://api.github.com/repos/octo-org/octo-repo/stargazers","contributors_url":"https://api.github.com/repos/octo-org/octo-repo/contributors","subscribers_url":"https://api.github.com/repos/octo-org/octo-repo/subscribers","subscription_url":"https://api.github.com/repos/octo-org/octo-repo/subscription","commits_url":"https://api.github.com/repos/octo-org/octo-repo/commits{/sha}","git_commits_url":"https://api.github.com/repos/octo-org/octo-repo/git/commits{/sha}","comments_url":"https://api.github.com/repos/octo-org/octo-repo/comments{/number}","issue_comment_url":"https://api.github.com/repos/octo-org/octo-repo/issues/comments{/number}","contents_url":"https://api.github.com/repos/octo-org/octo-repo/contents/{+path}","compare_url":"https://api.github.com/repos/octo-org/octo-repo/compare/{base}...{head}","merges_url":"https://api.github.com/repos/octo-org/octo-repo/merges","archive_url":"https://api.github.com/repos/octo-org/octo-repo/{archive_format}{/ref}","downloads_url":"https://api.github.com/repos/octo-org/octo-repo/downloads","issues_url":"https://api.github.com/repos/octo-org/octo-repo/issues{/number}","pulls_url":"https://api.github.com/repos/octo-org/octo-repo/pulls{/number}","milestones_url":"https://api.github.com/repos/octo-org/octo-repo/milestones{/number}","notifications_url":"https://api.github.com/repos/octo-org/octo-repo/notifications{?since,all,participating}","labels_url":"https://api.github.com/repos/octo-org/octo-repo/labels{/name}","releases_url":"https://api.github.com/repos/octo-org/octo-repo/releases{/id}","deployments_url":"https://api.github.com/repos/octo-org/octo-repo/deployments"}}`
var testGhUserReposResponse = `[{"id":1296269,"node_id":"MDEwOlJlcG9zaXRvcnkxMjk2MjY5","name":"Hello-World-2","full_name":"octocat/Hello-World","owner":{"login":"octocat","id":1,"node_id":"MDQ6VXNlcjE=","avatar_url":"https://github.com/images/error/octocat_happy.gif","gravatar_id":"","url":"https://api.github.com/users/octocat","html_url":"https://github.com/octocat","followers_url":"https://api.github.com/users/octocat/followers","following_url":"https://api.github.com/users/octocat/following{/other_user}","gists_url":"https://api.github.com/users/octocat/gists{/gist_id}","starred_url":"https://api.github.com/users/octocat/starred{/owner}{/repo}","subscriptions_url":"https://api.github.com/users/octocat/subscriptions","organizations_url":"https://api.github.com/users/octocat/orgs","repos_url":"https://api.github.com/users/octocat/repos","events_url":"https://api.github.com/users/octocat/events{/privacy}","received_events_url":"https://api.github.com/users/octocat/received_events","type":"User","site_admin":false},"private":false,"html_url":"https://github.com/octocat/Hello-World","description":"This your first repo!","fork":false,"url":"https://api.github.com/repos/octocat/Hello-World","archive_url":"https://api.github.com/repos/octocat/Hello-World/{archive_format}{/ref}","assignees_url":"https://api.github.com/repos/octocat/Hello-World/assignees{/user}","blobs_url":"https://api.github.com/repos/octocat/Hello-World/git/blobs{/sha}","branches_url":"https://api.github.com/repos/octocat/Hello-World/branches{/branch}","collaborators_url":"https://api.github.com/repos/octocat/Hello-World/collaborators{/collaborator}","comments_url":"https://api.github.com/repos/octocat/Hello-World/comments{/number}","commits_url":"https://api.github.com/repos/octocat/Hello-World/commits{/sha}","compare_url":"https://api.github.com/repos/octocat/Hello-World/compare/{base}...{head}","contents_url":"https://api.github.com/repos/octocat/Hello-World/contents/{+path}","contributors_url":"https://api.github.com/repos/octocat/Hello-World/contributors","deployments_url":"https://api.github.com/repos/octocat/Hello-World/deployments","downloads_url":"https://api.github.com/repos/octocat/Hello-World/downloads","events_url":"https://api.github.com/repos/octocat/Hello-World/events","forks_url":"https://api.github.com/repos/octocat/Hello-World/forks","git_commits_url":"https://api.github.com/repos/octocat/Hello-World/git/commits{/sha}","git_refs_url":"https://api.github.com/repos/octocat/Hello-World/git/refs{/sha}","git_tags_url":"https://api.github.com/repos/octocat/Hello-World/git/tags{/sha}","git_url":"git:github.com/octocat/Hello-World.git","issue_comment_url":"https://api.github.com/repos/octocat/Hello-World/issues/comments{/number}","issue_events_url":"https://api.github.com/repos/octocat/Hello-World/issues/events{/number}","issues_url":"https://api.github.com/repos/octocat/Hello-World/issues{/number}","keys_url":"https://api.github.com/repos/octocat/Hello-World/keys{/key_id}","labels_url":"https://api.github.com/repos/octocat/Hello-World/labels{/name}","languages_url":"https://api.github.com/repos/octocat/Hello-World/languages","merges_url":"https://api.github.com/repos/octocat/Hello-World/merges","milestones_url":"https://api.github.com/repos/octocat/Hello-World/milestones{/number}","notifications_url":"https://api.github.com/repos/octocat/Hello-World/notifications{?since,all,participating}","pulls_url":"https://api.github.com/repos/octocat/Hello-World/pulls{/number}","releases_url":"https://api.github.com/repos/octocat/Hello-World/releases{/id}","ssh_url":"git@github.com:octocat/Hello-World.git","stargazers_url":"https://api.github.com/repos/octocat/Hello-World/stargazers","statuses_url":"https://api.github.com/repos/octocat/Hello-World/statuses/{sha}","subscribers_url":"https://api.github.com/repos/octocat/Hello-World/subscribers","subscription_url":"https://api.github.com/repos/octocat/Hello-World/subscription","tags_url":"https://api.github.com/repos/octocat/Hello-World/tags","teams_url":"https://api.github.com/repos/octocat/Hello-World/teams","trees_url":"https://api.github.com/repos/octocat/Hello-World/git/trees{/sha}","clone_url":"https://github.com/octocat/Hello-World.git","mirror_url":"git:git.example.com/octocat/Hello-World","hooks_url":"https://api.github.com/repos/octocat/Hello-World/hooks","svn_url":"https://svn.github.com/octocat/Hello-World","homepage":"https://github.com","language":null,"forks_count":9,"stargazers_count":80,"watchers_count":80,"size":108,"default_branch":"master","open_issues_count":0,"is_template":true,"topics":["octocat","atom","electron","api"],"has_issues":true,"has_projects":true,"has_wiki":true,"has_pages":false,"has_downloads":true,"archived":false,"disabled":false,"visibility":"public","pushed_at":"2011-01-26T19:06:43Z","created_at":"2011-01-26T19:01:12Z","updated_at":"2011-01-26T19:14:43Z","permissions":{"admin":false,"push":false,"pull":true},"allow_rebase_merge":true,"template_repository":null,"temp_clone_token":"ABTLWHOULUVAXGTRYU7OC2876QJ2O","allow_squash_merge":true,"allow_auto_merge":false,"delete_branch_on_merge":true,"allow_merge_commit":true,"subscribers_count":42,"network_count":0,"license":{"key":"mit","name":"MIT License","url":"https://api.github.com/licenses/mit","spdx_id":"MIT","node_id":"MDc6TGljZW5zZW1pdA==","html_url":"https://github.com/licenses/mit"},"forks":1,"open_issues":1,"watchers":1},{"id":1296269,"node_id":"MDEwOlJlcG9zaXRvcnkxMjk2MjY5","name":"Hello-World","full_name":"octocat/Hello-World","owner":{"login":"octocat","id":1,"node_id":"MDQ6VXNlcjE=","avatar_url":"https://github.com/images/error/octocat_happy.gif","gravatar_id":"","url":"https://api.github.com/users/octocat","html_url":"https://github.com/octocat","followers_url":"https://api.github.com/users/octocat/followers","following_url":"https://api.github.com/users/octocat/following{/other_user}","gists_url":"https://api.github.com/users/octocat/gists{/gist_id}","starred_url":"https://api.github.com/users/octocat/starred{/owner}{/repo}","subscriptions_url":"https://api.github.com/users/octocat/subscriptions","organizations_url":"https://api.github.com/users/octocat/orgs","repos_url":"https://api.github.com/users/octocat/repos","events_url":"https://api.github.com/users/octocat/events{/privacy}","received_events_url":"https://api.github.com/users/octocat/received_events","type":"User","site_admin":false},"private":false,"html_url":"https://github.com/octocat/Hello-World","description":"This your first repo!","fork":false,"url":"https://api.github.com/repos/octocat/Hello-World","archive_url":"https://api.github.com/repos/octocat/Hello-World/{archive_format}{/ref}","assignees_url":"https://api.github.com/repos/octocat/Hello-World/assignees{/user}","blobs_url":"https://api.github.com/repos/octocat/Hello-World/git/blobs{/sha}","branches_url":"https://api.github.com/repos/octocat/Hello-World/branches{/branch}","collaborators_url":"https://api.github.com/repos/octocat/Hello-World/collaborators{/collaborator}","comments_url":"https://api.github.com/repos/octocat/Hello-World/comments{/number}","commits_url":"https://api.github.com/repos/octocat/Hello-World/commits{/sha}","compare_url":"https://api.github.com/repos/octocat/Hello-World/compare/{base}...{head}","contents_url":"https://api.github.com/repos/octocat/Hello-World/contents/{+path}","contributors_url":"https://api.github.com/repos/octocat/Hello-World/contributors","deployments_url":"https://api.github.com/repos/octocat/Hello-World/deployments","downloads_url":"https://api.github.com/repos/octocat/Hello-World/downloads","events_url":"https://api.github.com/repos/octocat/Hello-World/events","forks_url":"https://api.github.com/repos/octocat/Hello-World/forks","git_commits_url":"https://api.github.com/repos/octocat/Hello-World/git/commits{/sha}","git_refs_url":"https://api.github.com/repos/octocat/Hello-World/git/refs{/sha}","git_tags_url":"https://api.github.com/repos/octocat/Hello-World/git/tags{/sha}","git_url":"git:github.com/octocat/Hello-World.git","issue_comment_url":"https://api.github.com/repos/octocat/Hello-World/issues/comments{/number}","issue_events_url":"https://api.github.com/repos/octocat/Hello-World/issues/events{/number}","issues_url":"https://api.github.com/repos/octocat/Hello-World/issues{/number}","keys_url":"https://api.github.com/repos/octocat/Hello-World/keys{/key_id}","labels_url":"https://api.github.com/repos/octocat/Hello-World/labels{/name}","languages_url":"https://api.github.com/repos/octocat/Hello-World/languages","merges_url":"https://api.github.com/repos/octocat/Hello-World/merges","milestones_url":"https://api.github.com/repos/octocat/Hello-World/milestones{/number}","notifications_url":"https://api.github.com/repos/octocat/Hello-World/notifications{?since,all,participating}","pulls_url":"https://api.github.com/repos/octocat/Hello-World/pulls{/number}","releases_url":"https://api.github.com/repos/octocat/Hello-World/releases{/id}","ssh_url":"git@github.com:octocat/Hello-World.git","stargazers_url":"https://api.github.com/repos/octocat/Hello-World/stargazers","statuses_url":"https://api.github.com/repos/octocat/Hello-World/statuses/{sha}","subscribers_url":"https://api.github.com/repos/octocat/Hello-World/subscribers","subscription_url":"https://api.github.com/repos/octocat/Hello-World/subscription","tags_url":"https://api.github.com/repos/octocat/Hello-World/tags","teams_url":"https://api.github.com/repos/octocat/Hello-World/teams","trees_url":"https://api.github.com/repos/octocat/Hello-World/git/trees{/sha}","clone_url":"https://github.com/octocat/Hello-World.git","mirror_url":"git:git.example.com/octocat/Hello-World","hooks_url":"https://api.github.com/repos/octocat/Hello-World/hooks","svn_url":"https://svn.github.com/octocat/Hello-World","homepage":"https://github.com","language":null,"forks_count":9,"stargazers_count":80,"watchers_count":80,"size":108,"default_branch":"master","open_issues_count":0,"is_template":true,"topics":["octocat","atom","electron","api"],"has_issues":true,"has_projects":true,"has_wiki":true,"has_pages":false,"has_downloads":true,"archived":false,"disabled":false,"visibility":"public","pushed_at":"2011-01-26T19:06:43Z","created_at":"2011-01-26T19:01:12Z","updated_at":"2011-01-26T19:14:43Z","permissions":{"admin":false,"push":false,"pull":true},"allow_rebase_merge":true,"template_repository":null,"temp_clone_token":"ABTLWHOULUVAXGTRYU7OC2876QJ2O","allow_squash_merge":true,"allow_auto_merge":false,"delete_branch_on_merge":true,"allow_merge_commit":true,"subscribers_count":42,"network_count":0,"license":{"key":"mit","name":"MIT License","url":"https://api.github.com/licenses/mit","spdx_id":"MIT","node_id":"MDc6TGljZW5zZW1pdA==","html_url":"https://github.com/licenses/mit"},"forks":1,"open_issues":1,"watchers":1}]`
var testGhWFJobResponse = `{"total_count":1,"jobs":[{"id":399444496,"run_id":29679449,"run_url":"https://api.github.com/repos/octo-org/octo-repo/actions/runs/29679449","node_id":"MDEyOldvcmtmbG93IEpvYjM5OTQ0NDQ5Ng==","head_sha":"f83a356604ae3c5d03e1b46ef4d1ca77d64a90b0","url":"https://api.github.com/repos/octo-org/octo-repo/actions/jobs/399444496","html_url":"https://github.com/octo-org/octo-repo/runs/399444496","status":"queued","conclusion":"success","started_at":"2020-01-20T17:42:40Z","completed_at":"2020-01-20T17:44:39Z","name":"build","steps":[{"name":"Set up job","status":"completed","conclusion":"success","number":1,"started_at":"2020-01-20T09:42:40.000-08:00","completed_at":"2020-01-20T09:42:41.000-08:00"},{"name":"Run actions/checkout@v2","status":"queued","conclusion":"success","number":2,"started_at":"2020-01-20T09:42:41.000-08:00","completed_at":"2020-01-20T09:42:45.000-08:00"},{"name":"Set up Ruby","status":"completed","conclusion":"success","number":3,"started_at":"2020-01-20T09:42:45.000-08:00","completed_at":"2020-01-20T09:42:45.000-08:00"},{"name":"Run actions/cache@v3","status":"completed","conclusion":"success","number":4,"started_at":"2020-01-20T09:42:45.000-08:00","completed_at":"2020-01-20T09:42:48.000-08:00"},{"name":"Install Bundler","status":"completed","conclusion":"success","number":5,"started_at":"2020-01-20T09:42:48.000-08:00","completed_at":"2020-01-20T09:42:52.000-08:00"},{"name":"Install Gems","status":"completed","conclusion":"success","number":6,"started_at":"2020-01-20T09:42:52.000-08:00","completed_at":"2020-01-20T09:42:53.000-08:00"},{"name":"Run Tests","status":"completed","conclusion":"success","number":7,"started_at":"2020-01-20T09:42:53.000-08:00","completed_at":"2020-01-20T09:42:59.000-08:00"},{"name":"Deploy to Heroku","status":"completed","conclusion":"success","number":8,"started_at":"2020-01-20T09:42:59.000-08:00","completed_at":"2020-01-20T09:44:39.000-08:00"},{"name":"Post actions/cache@v3","status":"completed","conclusion":"success","number":16,"started_at":"2020-01-20T09:44:39.000-08:00","completed_at":"2020-01-20T09:44:39.000-08:00"},{"name":"Complete job","status":"completed","conclusion":"success","number":17,"started_at":"2020-01-20T09:44:39.000-08:00","completed_at":"2020-01-20T09:44:39.000-08:00"}],"check_run_url":"https://api.github.com/repos/octo-org/octo-repo/check-runs/399444496","labels":["self-hosted","foo","bar"],"runner_id":1,"runner_name":"my runner","runner_group_id":2,"runner_group_name":"my runner group","workflow_name":"CI","head_branch":"main"}]}`

type parseGitHubRunnerMetadataTestData struct {
	testName      string
	metadata      map[string]string
	hasEnvs       bool
	isError       bool
	expectedError string
}

var testGitHubRunnerResolvedEnv = map[string]string{
	"GITHUB_API_URL": "https://api.github.com",
	"ACCESS_TOKEN":   "sample",
	"RUNNER_SCOPE":   ORG,
	"ORG_NAME":       "ownername",
	"OWNER":          "ownername",
	"LABELS":         "foo,bar",
	"REPOS":          "reponame,otherrepo",
	"APP_ID":         "1",
	"INST_ID":        "2",
}

var testGitHubRunnerTokenEnv = map[string]string{
	"ACCESS_TOKEN_DIFF": "sample",
}

var testAuthParams = map[string]string{
	"personalAccessToken": "sample",
}

var testGitHubRunnerMetadata = []parseGitHubRunnerMetadataTestData{
	// nothing passed
	{"empty", map[string]string{}, true, true, "no runnerScope given"},
	// properly formed
	{"properly formed", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": ORG, "owner": "ownername", "repos": "reponame,otherrepo", "labels": "golang", "targetWorkflowQueueLength": "1"}, true, false, ""},
	// properly formed with no labels and no repos
	{"properly formed, no labels or repos", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": REPO, "owner": "ownername", "targetWorkflowQueueLength": "1"}, true, false, ""},
	// string for int64
	{"string for int64-1", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": REPO, "owner": "ownername", "targetWorkflowQueueLength": "a"}, true, false, ""},
	// formed from env
	{"formed from env", map[string]string{"githubApiURLFromEnv": "GITHUB_API_URL", "runnerScopeFromEnv": "RUNNER_SCOPE", "ownerFromEnv": "OWNER", "reposFromEnv": "REPOS", "targetWorkflowQueueLength": "1"}, true, false, ""},
	// missing runnerScope
	{"missing runnerScope", map[string]string{"githubApiURL": "https://api.github.com", "owner": "ownername", "repos": "reponame,otherrepo", "labels": "golang", "targetWorkflowQueueLength": "1"}, true, true, "no runnerScope given"},
	// empty runnerScope
	{"empty runnerScope", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": "", "owner": "ownername", "repos": "reponame,otherrepo", "labels": "golang", "targetWorkflowQueueLength": "1"}, true, true, "no runnerScope given"},
	// missing owner
	{"missing owner", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": REPO, "repos": "reponame", "targetWorkflowQueueLength": "1"}, true, true, "no owner given"},
	// empty owner
	{"empty owner", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": REPO, "owner": "", "repos": "reponame", "targetWorkflowQueueLength": "1"}, true, true, "no owner given"},
	// empty token
	{"empty targetWorkflowQueueLength", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": REPO, "owner": "ownername", "repos": "reponame"}, true, false, ""},
	// missing installationID From Env
	{"missing installationID Env", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": ORG, "owner": "ownername", "repos": "reponame,otherrepo", "labels": "golang", "targetWorkflowQueueLength": "1", "applicationIDFromEnv": "APP_ID"}, true, true, "applicationID, installationID and applicationKey must be given"},
	// missing applicationID From Env
	{"missing applicationId Env", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": ORG, "owner": "ownername", "repos": "reponame,otherrepo", "labels": "golang", "targetWorkflowQueueLength": "1", "installationIDFromEnv": "INST_ID"}, true, true, "applicationID, installationID and applicationKey must be given"},
	// nothing passed
	{"empty, no envs", map[string]string{}, false, true, "no runnerScope given"},
	//  empty githubApiURL
	{"empty githubApiURL, no envs", map[string]string{"githubApiURL": "", "runnerScope": ORG, "owner": "ownername", "repos": "reponame,otherrepo", "labels": "golang", "targetWorkflowQueueLength": "1"}, false, false, ""},
	// properly formed
	{"properly formed, no envs", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": ORG, "owner": "ownername", "repos": "reponame,otherrepo", "labels": "golang", "targetWorkflowQueueLength": "1"}, false, false, ""},
	// properly formed with no labels and no repos
	{"properly formed, no envs, labels or repos", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": ENT, "owner": "ownername", "targetWorkflowQueueLength": "1"}, false, false, ""},
	// formed from env
	{"formed from env, no envs", map[string]string{"githubApiURLFromEnv": "GITHUB_API_URL", "ownerFromEnv": "OWNER", "repos": "reponame", "targetWorkflowQueueLength": "1"}, false, true, "no runnerScope given"},
	// formed from default env
	{"formed from default env, no envs", map[string]string{"owner": "ownername", "repos": "reponame", "targetWorkflowQueueLength": "1"}, false, true, "no runnerScope given"},
	// missing runnerScope
	{"missing runnerScope, no envs", map[string]string{"githubApiURL": "https://api.github.com", "owner": "ownername", "repos": "reponame,otherrepo", "labels": "golang", "targetWorkflowQueueLength": "1"}, false, true, "no runnerScope given"},
	// empty runnerScope
	{"empty runnerScope, no envs", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": "", "owner": "ownername", "repos": "reponame,otherrepo", "labels": "golang", "targetWorkflowQueueLength": "1"}, false, true, "no runnerScope given"},
	// empty owner
	{"empty owner, no envs", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": REPO, "owner": "", "repos": "reponame", "targetWorkflowQueueLength": "1"}, false, true, "no owner given"},
	// missing owner
	{"missing owner, no envs", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": REPO, "repos": "reponame", "targetWorkflowQueueLength": "1"}, false, true, "no owner given"},
	// missing labels, no envs
	{"missing labels, no envs", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": ORG, "owner": "ownername", "repos": "reponame,otherrepo", "targetWorkflowQueueLength": "1"}, false, false, ""},
	// empty labels, no envs
	{"empty labels, no envs", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": ORG, "owner": "ownername", "labels": "", "repos": "reponame,otherrepo", "targetWorkflowQueueLength": "1"}, false, false, ""},
	// missing repos, no envs
	{"missing repos, no envs", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": ORG, "owner": "ownername", "labels": "golang", "targetWorkflowQueueLength": "1"}, false, false, ""},
	// empty repos, no envs
	{"empty repos, no envs", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": ORG, "owner": "ownername", "labels": "golang", "repos": "", "targetWorkflowQueueLength": "1"}, false, false, ""},
	// missing installationID
	{"missing installationID", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": ORG, "owner": "ownername", "repos": "reponame,otherrepo", "labels": "golang", "targetWorkflowQueueLength": "1", "applicationID": "1"}, true, true, "applicationID, installationID and applicationKey must be given"},
	// missing applicationID
	{"missing applicationID", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": ORG, "owner": "ownername", "repos": "reponame,otherrepo", "labels": "golang", "targetWorkflowQueueLength": "1", "installationID": "1"}, true, true, "applicationID, installationID and applicationKey must be given"},
	// all good
	{"missing applicationKey", map[string]string{"githubApiURL": "https://api.github.com", "runnerScope": ORG, "owner": "ownername", "repos": "reponame,otherrepo", "labels": "golang", "targetWorkflowQueueLength": "1", "applicationID": "1", "installationID": "1"}, true, true, "applicationID, installationID and applicationKey must be given"},
}

func TestGitHubRunnerParseMetadata(t *testing.T) {
	for _, testData := range testGitHubRunnerMetadata {
		t.Run(testData.testName, func(t *testing.T) {
			var err error
			if testData.hasEnvs {
				_, err = parseGitHubRunnerMetadata(&ScalerConfig{ResolvedEnv: testGitHubRunnerResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testAuthParams})
			} else {
				_, err = parseGitHubRunnerMetadata(&ScalerConfig{ResolvedEnv: testGitHubRunnerTokenEnv, TriggerMetadata: testData.metadata, AuthParams: testAuthParams})
			}
			if testData.isError && err == nil {
				t.Fatal("expected error but got none")
			}
			if testData.isError && err != nil && err.Error() != testData.expectedError {
				t.Fatal("expected error " + testData.expectedError + " but got error " + err.Error())
			}
			if !testData.isError && err != nil {
				t.Fatalf("expected no error but got %s", err)
			}
		})
	}
}

func getGitHubTestMetaData(url string) *githubRunnerMetadata {
	testpat := "testpat"

	meta := githubRunnerMetadata{
		githubAPIURL:              url,
		runnerScope:               REPO,
		owner:                     "testOwner",
		personalAccessToken:       &testpat,
		targetWorkflowQueueLength: 1,
	}

	return &meta
}

func buildQueueJSON() []byte {
	output := testGhWorkflowResponse[0 : len(testGhWorkflowResponse)-2]
	for i := 1; i < ghLoadCount; i++ {
		output = output + "," + ghDeadJob
	}

	output += "]}"

	return []byte(output)
}

func apiStubHandler(hasRateLeft bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		futureReset := time.Now()
		futureReset = futureReset.Add(time.Minute * 30)
		w.Header().Set("X-RateLimit-Reset", fmt.Sprint(futureReset.Unix()))
		if hasRateLeft {
			w.Header().Set("X-RateLimit-Remaining", "50")
		} else {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.WriteHeader(http.StatusForbidden)
		}
		if strings.HasSuffix(r.URL.String(), "jobs") {
			_, _ = w.Write([]byte(testGhWFJobResponse))
			w.WriteHeader(http.StatusOK)
		}
		if strings.HasSuffix(r.URL.String(), "runs") {
			if strings.Contains(r.URL.String(), "BadRepo") {
				w.WriteHeader(http.StatusNotFound)
			} else {
				_, _ = w.Write(buildQueueJSON())
				w.WriteHeader(http.StatusOK)
			}
		}
		if strings.HasSuffix(r.URL.String(), "repos") {
			_, _ = w.Write([]byte(testGhUserReposResponse))
			w.WriteHeader(http.StatusOK)
		}
	}))
}

func apiStubHandler404() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
}

func TestNewGitHubRunnerScaler_QueueLength_NoRateLeft(t *testing.T) {
	var apiStub = apiStubHandler(false)

	meta := getGitHubTestMetaData(apiStub.URL)

	mockGitHubRunnerScaler := githubRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	tRepo := []string{"test"}
	mockGitHubRunnerScaler.metadata.repos = tRepo

	_, err := mockGitHubRunnerScaler.GetWorkflowQueueLength(context.TODO())

	if err == nil {
		t.Fail()
	}

	if !strings.HasPrefix(err.Error(), "GitHub API rate limit exceeded") {
		t.Fail()
	}
}

func TestNewGitHubRunnerScaler_QueueLength_SingleRepo(t *testing.T) {
	var apiStub = apiStubHandler(true)

	meta := getGitHubTestMetaData(apiStub.URL)

	mockGitHubRunnerScaler := githubRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	mockGitHubRunnerScaler.metadata.repos = []string{"test"}
	mockGitHubRunnerScaler.metadata.labels = []string{"foo", "bar"}

	queueLen, err := mockGitHubRunnerScaler.GetWorkflowQueueLength(context.TODO())

	if err != nil {
		t.Fail()
	}

	if queueLen != 1 {
		t.Fail()
	}
}

func TestNewGitHubRunnerScaler_QueueLength_SingleRepo_ExtraRunnerLabels(t *testing.T) {
	var apiStub = apiStubHandler(true)

	meta := getGitHubTestMetaData(apiStub.URL)

	mockGitHubRunnerScaler := githubRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	mockGitHubRunnerScaler.metadata.repos = []string{"test"}
	mockGitHubRunnerScaler.metadata.labels = []string{"foo", "bar", "other", "more"}

	queueLen, err := mockGitHubRunnerScaler.GetWorkflowQueueLength(context.TODO())

	if err != nil {
		t.Fail()
	}

	if queueLen != 1 {
		t.Fail()
	}
}

func TestNewGitHubRunnerScaler_QueueLength_SingleRepo_LessRunnerLabels(t *testing.T) {
	var apiStub = apiStubHandler(true)

	meta := getGitHubTestMetaData(apiStub.URL)

	mockGitHubRunnerScaler := githubRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	mockGitHubRunnerScaler.metadata.repos = []string{"test"}
	mockGitHubRunnerScaler.metadata.labels = []string{"foo"}

	queueLen, err := mockGitHubRunnerScaler.GetWorkflowQueueLength(context.TODO())

	if err != nil {
		t.Fail()
	}

	if queueLen != 0 {
		t.Fail()
	}
}

func TestNewGitHubRunnerScaler_404(t *testing.T) {
	var apiStub = apiStubHandler404()

	meta := getGitHubTestMetaData(apiStub.URL)

	mockGitHubRunnerScaler := githubRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	mockGitHubRunnerScaler.metadata.labels = []string{"foo", "bar"}

	_, err := mockGitHubRunnerScaler.GetWorkflowQueueLength(context.TODO())

	if err == nil {
		t.Fail()
	}

	if !strings.HasSuffix(err.Error(), "404 response: not found") {
		t.Fail()
	}
}

func TestNewGitHubRunnerScaler_BadConnection(t *testing.T) {
	meta := getGitHubTestMetaData("http://127.0.0.1:9999")

	mockGitHubRunnerScaler := githubRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	mockGitHubRunnerScaler.metadata.repos = []string{"test"}
	mockGitHubRunnerScaler.metadata.labels = []string{"foo", "bar"}

	_, err := mockGitHubRunnerScaler.GetWorkflowQueueLength(context.TODO())

	if err == nil {
		t.Fail()
	}

	if !strings.HasSuffix(err.Error(), "connect: connection refused") {
		fmt.Println(err.Error())
		t.Fail()
	}
}

func TestNewGitHubRunnerScaler_BadURL(t *testing.T) {
	meta := getGitHubTestMetaData(string([]byte{199, 199, 199, 199, 199, 199, 199, 199})) // invalid URL

	mockGitHubRunnerScaler := githubRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	mockGitHubRunnerScaler.metadata.repos = []string{"test"}
	mockGitHubRunnerScaler.metadata.labels = []string{"foo", "bar"}

	_, err := mockGitHubRunnerScaler.GetWorkflowQueueLength(context.TODO())

	if err == nil {
		t.Fail()
	}

	if !strings.HasSuffix(err.Error(), "unsupported protocol scheme \"\"") {
		fmt.Println(err.Error())
		t.Fail()
	}
}

func TestNewGitHubRunnerScaler_QueueLength_NoRunnerLabels(t *testing.T) {
	var apiStub = apiStubHandler(true)

	meta := getGitHubTestMetaData(apiStub.URL)

	mockGitHubRunnerScaler := githubRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	mockGitHubRunnerScaler.metadata.repos = []string{"test"}

	queueLen, err := mockGitHubRunnerScaler.GetWorkflowQueueLength(context.TODO())

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	if queueLen != 0 {
		t.Fail()
	}
}

func TestNewGitHubRunnerScaler_QueueLength_MultiRepo_Assigned(t *testing.T) {
	var apiStub = apiStubHandler(true)

	meta := getGitHubTestMetaData(apiStub.URL)

	mockGitHubRunnerScaler := githubRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	tRepo := []string{"test", "test2"}
	mockGitHubRunnerScaler.metadata.repos = tRepo
	mockGitHubRunnerScaler.metadata.runnerScope = ORG
	mockGitHubRunnerScaler.metadata.labels = []string{"foo", "bar"}

	queueLen, err := mockGitHubRunnerScaler.GetWorkflowQueueLength(context.TODO())

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	if queueLen != 2 {
		fmt.Println(queueLen)
		t.Fail()
	}
}

func TestNewGitHubRunnerScaler_QueueLength_MultiRepo_Assigned_OneBad(t *testing.T) {
	var apiStub = apiStubHandler(true)

	meta := getGitHubTestMetaData(apiStub.URL)

	mockGitHubRunnerScaler := githubRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	tRepo := []string{"test", "test2", "BadRepo"}
	mockGitHubRunnerScaler.metadata.repos = tRepo
	mockGitHubRunnerScaler.metadata.runnerScope = ORG
	mockGitHubRunnerScaler.metadata.labels = []string{"foo", "bar"}

	queueLen, err := mockGitHubRunnerScaler.GetWorkflowQueueLength(context.TODO())

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	if queueLen != 2 {
		fmt.Println(queueLen)
		t.Fail()
	}
}

func TestNewGitHubRunnerScaler_QueueLength_MultiRepo_PulledUserRepos(t *testing.T) {
	var apiStub = apiStubHandler(true)

	meta := getGitHubTestMetaData(apiStub.URL)

	mockGitHubRunnerScaler := githubRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	mockGitHubRunnerScaler.metadata.labels = []string{"foo", "bar"}

	queueLen, err := mockGitHubRunnerScaler.GetWorkflowQueueLength(context.TODO())

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	if queueLen != 2 {
		t.Fail()
	}
}

func TestNewGitHubRunnerScaler_QueueLength_MultiRepo_PulledOrgRepos(t *testing.T) {
	var apiStub = apiStubHandler(true)

	meta := getGitHubTestMetaData(apiStub.URL)

	mockGitHubRunnerScaler := githubRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	mockGitHubRunnerScaler.metadata.runnerScope = ORG
	mockGitHubRunnerScaler.metadata.labels = []string{"foo", "bar"}

	queueLen, err := mockGitHubRunnerScaler.GetWorkflowQueueLength(context.TODO())

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	if queueLen != 2 {
		t.Fail()
	}
}

func TestNewGitHubRunnerScaler_QueueLength_MultiRepo_PulledEntRepos(t *testing.T) {
	var apiStub = apiStubHandler(true)

	meta := getGitHubTestMetaData(apiStub.URL)

	mockGitHubRunnerScaler := githubRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	mockGitHubRunnerScaler.metadata.runnerScope = ENT
	mockGitHubRunnerScaler.metadata.labels = []string{"foo", "bar"}

	queueLen, err := mockGitHubRunnerScaler.GetWorkflowQueueLength(context.TODO())

	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	if queueLen != 2 {
		t.Fail()
	}
}

func TestNewGitHubRunnerScaler_QueueLength_MultiRepo_PulledBadRepos(t *testing.T) {
	var apiStub = apiStubHandler(true)

	meta := getGitHubTestMetaData(apiStub.URL)

	mockGitHubRunnerScaler := githubRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	mockGitHubRunnerScaler.metadata.runnerScope = "bad"

	_, err := mockGitHubRunnerScaler.GetWorkflowQueueLength(context.TODO())

	if err == nil {
		t.Fail()
	}

	if !strings.HasPrefix(err.Error(), "runnerScope bad not supported") {
		t.Fail()
	}
}

func TestNewGitHubRunnerScaler_QueueLength_MultiRepo_PulledRepos_NoRate(t *testing.T) {
	var apiStub = apiStubHandler(false)

	meta := getGitHubTestMetaData(apiStub.URL)

	mockGitHubRunnerScaler := githubRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	_, err := mockGitHubRunnerScaler.GetWorkflowQueueLength(context.TODO())

	if err == nil {
		t.Fail()
	}

	if !strings.HasPrefix(err.Error(), "GitHub API rate limit exceeded") {
		t.Fail()
	}
}

type githubRunnerMetricIdentifier struct {
	metadataTestData *map[string]string
	scalerIndex      int
	name             string
}

var githubRunnerMetricIdentifiers = []githubRunnerMetricIdentifier{
	{&testGitHubRunnerMetadata[1].metadata, 0, "s0-github-runner-ownername"},
	{&testGitHubRunnerMetadata[1].metadata, 1, "s1-github-runner-ownername"},
}

func TestGithubRunnerGetMetricSpecForScaling(t *testing.T) {
	for i, testData := range githubRunnerMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseGitHubRunnerMetadata(&ScalerConfig{ResolvedEnv: testGitHubRunnerResolvedEnv, TriggerMetadata: *testData.metadataTestData, AuthParams: testAuthParams, ScalerIndex: testData.scalerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockGitHubRunnerScaler := githubRunnerScaler{
			metadata:   meta,
			httpClient: http.DefaultClient,
		}

		metricSpec := mockGitHubRunnerScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		t.Log(i)
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName+"!="+testData.name)
		}
	}
}
