package github

import (
	"encoding/json"
	"testing"
)

func TestModel_deserialize_large_github_blob(t *testing.T) {
	var payload PushWebhookPayload
	err := json.Unmarshal([]byte(largeBlob), &payload)
	if err != nil {
		t.Fatal(err)
	}

	if payload.Repository.Name != "some-repo" {
		t.Fatalf("Expected payload.Repository.Name to be 'some-repo', got '%s'", payload.Repository.Name)
	}

}

var largeBlob = `{
  "zen": "Anything added dilutes everything else.",
  "hook_id": 423975653,
  "hook": {
    "type": "Repository",
    "id": 423975653,
    "name": "web",
    "active": true,
    "events": [
      "push"
    ],
    "config": {
      "content_type": "json",
      "insecure_ssl": "0",
      "secret": "********",
      "url": "https://blaha.dev/webhook"
    },
    "updated_at": "2023-07-13T22:26:00Z",
    "created_at": "2023-07-13T22:26:00Z",
    "url": "https://api.github.com/repos/SomethingSomething/some-repo/hooks/423975653",
    "test_url": "https://api.github.com/repos/SomethingSomething/some-repo/hooks/423975653/test",
    "ping_url": "https://api.github.com/repos/SomethingSomething/some-repo/hooks/423975653/pings",
    "deliveries_url": "https://api.github.com/repos/SomethingSomething/some-repo/hooks/423975653/deliveries",
    "last_response": {
      "code": null,
      "status": "unused",
      "message": null
    }
  },
  "repository": {
    "id": 661405395,
    "node_id": "R_kgDOJ2w-0w",
    "name": "some-repo",
    "full_name": "SomethingSomething/some-repo",
    "private": true,
    "owner": {
      "login": "SomethingSomething",
      "id": 1761299,
      "node_id": "MDQ6VXNlcjE3NjEyOTk=",
      "avatar_url": "https://avatars.githubusercontent.com/u/1761299?v=4",
      "gravatar_id": "",
      "url": "https://api.github.com/users/SomethingSomething",
      "html_url": "https://github.com/SomethingSomething",
      "followers_url": "https://api.github.com/users/SomethingSomething/followers",
      "following_url": "https://api.github.com/users/SomethingSomething/following{/other_user}",
      "gists_url": "https://api.github.com/users/SomethingSomething/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/SomethingSomething/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/SomethingSomething/subscriptions",
      "organizations_url": "https://api.github.com/users/SomethingSomething/orgs",
      "repos_url": "https://api.github.com/users/SomethingSomething/repos",
      "events_url": "https://api.github.com/users/SomethingSomething/events{/privacy}",
      "received_events_url": "https://api.github.com/users/SomethingSomething/received_events",
      "type": "User",
      "site_admin": false
    },
    "html_url": "https://github.com/SomethingSomething/some-repo",
    "description": null,
    "fork": false,
    "url": "https://api.github.com/repos/SomethingSomething/some-repo",
    "forks_url": "https://api.github.com/repos/SomethingSomething/some-repo/forks",
    "keys_url": "https://api.github.com/repos/SomethingSomething/some-repo/keys{/key_id}",
    "collaborators_url": "https://api.github.com/repos/SomethingSomething/some-repo/collaborators{/collaborator}",
    "teams_url": "https://api.github.com/repos/SomethingSomething/some-repo/teams",
    "hooks_url": "https://api.github.com/repos/SomethingSomething/some-repo/hooks",
    "issue_events_url": "https://api.github.com/repos/SomethingSomething/some-repo/issues/events{/number}",
    "events_url": "https://api.github.com/repos/SomethingSomething/some-repo/events",
    "assignees_url": "https://api.github.com/repos/SomethingSomething/some-repo/assignees{/user}",
    "branches_url": "https://api.github.com/repos/SomethingSomething/some-repo/branches{/branch}",
    "tags_url": "https://api.github.com/repos/SomethingSomething/some-repo/tags",
    "blobs_url": "https://api.github.com/repos/SomethingSomething/some-repo/git/blobs{/sha}",
    "git_tags_url": "https://api.github.com/repos/SomethingSomething/some-repo/git/tags{/sha}",
    "git_refs_url": "https://api.github.com/repos/SomethingSomething/some-repo/git/refs{/sha}",
    "trees_url": "https://api.github.com/repos/SomethingSomething/some-repo/git/trees{/sha}",
    "statuses_url": "https://api.github.com/repos/SomethingSomething/some-repo/statuses/{sha}",
    "languages_url": "https://api.github.com/repos/SomethingSomething/some-repo/languages",
    "stargazers_url": "https://api.github.com/repos/SomethingSomething/some-repo/stargazers",
    "contributors_url": "https://api.github.com/repos/SomethingSomething/some-repo/contributors",
    "subscribers_url": "https://api.github.com/repos/SomethingSomething/some-repo/subscribers",
    "subscription_url": "https://api.github.com/repos/SomethingSomething/some-repo/subscription",
    "commits_url": "https://api.github.com/repos/SomethingSomething/some-repo/commits{/sha}",
    "git_commits_url": "https://api.github.com/repos/SomethingSomething/some-repo/git/commits{/sha}",
    "comments_url": "https://api.github.com/repos/SomethingSomething/some-repo/comments{/number}",
    "issue_comment_url": "https://api.github.com/repos/SomethingSomething/some-repo/issues/comments{/number}",
    "contents_url": "https://api.github.com/repos/SomethingSomething/some-repo/contents/{+path}",
    "compare_url": "https://api.github.com/repos/SomethingSomething/some-repo/compare/{base}...{head}",
    "merges_url": "https://api.github.com/repos/SomethingSomething/some-repo/merges",
    "archive_url": "https://api.github.com/repos/SomethingSomething/some-repo/{archive_format}{/ref}",
    "downloads_url": "https://api.github.com/repos/SomethingSomething/some-repo/downloads",
    "issues_url": "https://api.github.com/repos/SomethingSomething/some-repo/issues{/number}",
    "pulls_url": "https://api.github.com/repos/SomethingSomething/some-repo/pulls{/number}",
    "milestones_url": "https://api.github.com/repos/SomethingSomething/some-repo/milestones{/number}",
    "notifications_url": "https://api.github.com/repos/SomethingSomething/some-repo/notifications{?since,all,participating}",
    "labels_url": "https://api.github.com/repos/SomethingSomething/some-repo/labels{/name}",
    "releases_url": "https://api.github.com/repos/SomethingSomething/some-repo/releases{/id}",
    "deployments_url": "https://api.github.com/repos/SomethingSomething/some-repo/deployments",
    "created_at": "2023-07-02T18:33:04Z",
    "updated_at": "2023-07-02T18:42:56Z",
    "pushed_at": "2023-07-13T21:58:37Z",
    "git_url": "git://github.com/SomethingSomething/some-repo.git",
    "ssh_url": "git@github.com:SomethingSomething/some-repo.git",
    "clone_url": "https://github.com/SomethingSomething/some-repo.git",
    "svn_url": "https://github.com/SomethingSomething/some-repo",
    "homepage": null,
    "size": 16,
    "stargazers_count": 0,
    "watchers_count": 0,
    "language": "Dockerfile",
    "has_issues": true,
    "has_projects": true,
    "has_downloads": true,
    "has_wiki": false,
    "has_pages": false,
    "has_discussions": false,
    "forks_count": 0,
    "mirror_url": null,
    "archived": false,
    "disabled": false,
    "open_issues_count": 0,
    "license": null,
    "allow_forking": true,
    "is_template": false,
    "web_commit_signoff_required": false,
    "topics": [

    ],
    "visibility": "private",
    "forks": 0,
    "open_issues": 0,
    "watchers": 0,
    "default_branch": "main"
  },
  "sender": {
    "login": "SomethingSomething",
    "id": 1761299,
    "node_id": "MDQ6VXNlcjE3NjEyOTk=",
    "avatar_url": "https://avatars.githubusercontent.com/u/1761299?v=4",
    "gravatar_id": "",
    "url": "https://api.github.com/users/SomethingSomething",
    "html_url": "https://github.com/SomethingSomething",
    "followers_url": "https://api.github.com/users/SomethingSomething/followers",
    "following_url": "https://api.github.com/users/SomethingSomething/following{/other_user}",
    "gists_url": "https://api.github.com/users/SomethingSomething/gists{/gist_id}",
    "starred_url": "https://api.github.com/users/SomethingSomething/starred{/owner}{/repo}",
    "subscriptions_url": "https://api.github.com/users/SomethingSomething/subscriptions",
    "organizations_url": "https://api.github.com/users/SomethingSomething/orgs",
    "repos_url": "https://api.github.com/users/SomethingSomething/repos",
    "events_url": "https://api.github.com/users/SomethingSomething/events{/privacy}",
    "received_events_url": "https://api.github.com/users/SomethingSomething/received_events",
    "type": "User",
    "site_admin": false
  }
}`
