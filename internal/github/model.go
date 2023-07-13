package github

import "time"

type Repository struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"`
	Private       bool      `json:"private"`
	HtmlUrl       string    `json:"html_url"`
	Url           string    `json:"url"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	PushedAt      time.Time `json:"pushed_at"`
	GitUrl        string    `json:"git_url"`
	SshUrl        string    `json:"ssh_url"`
	CloneUrl      string    `json:"clone_url"`
	SvnUrl        string    `json:"svn_url"`
	Visibility    string    `json:"visibility"`
	DefaultBranch string    `json:"default_branch"`
	MasterBranch  string    `json:"master_branch"`
}

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Commit struct {
	ID        string    `json:"id"`
	TreeID    string    `json:"tree_id"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	URL       string    `json:"url"`
	Author    User      `json:"author"`
}

type PushWebhookPayload struct {
	Ref        string     `json:"ref"`
	HookId     int64      `json:"hook_id"`
	Repository Repository `json:"repository"`
	Pusher     User       `json:"pusher"`
	HeadCommit Commit     `json:"head_commit"`
}
