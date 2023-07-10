package github

type Repository struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Private       bool   `json:"private"`
	HtmlUrl       string `json:"html_url"`
	Url           string `json:"url"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	PushedAt      int64  `json:"pushed_at"`
	GitUrl        string `json:"git_url"`
	SshUrl        string `json:"ssh_url"`
	CloneUrl      string `json:"clone_url"`
	SvnUrl        string `json:"svn_url"`
	Visibility    string `json:"visibility"`
	DefaultBranch string `json:"default_branch"`
	MasterBranch  string `json:"master_branch"`
}

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Commit struct {
	ID        string `json:"id"`
	TreeID    string `json:"tree_id"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	URL       string `json:"url"`
	Author    User   `json:"author"`
}

type PushWebhookPayload struct {
	Ref        string     `json:"ref"`
	Repository Repository `json:"repository"`
	Pusher     User       `json:"pusher"`
	HeadCommit Commit     `json:"head_commit"`
}
