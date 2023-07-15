package github

import (
	"encoding/json"
	"fmt"
	"time"
)

type GhTime struct {
	Underlying time.Time
}

// UnmarshalJSON custom unmarshaler for time.Time
func (t *GhTime) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}

	// Try parsing as time.GhTime first
	goTime := time.Time{}
	errTime := goTime.UnmarshalJSON(b)
	if errTime == nil {
		t.Underlying = goTime
		return nil
	}

	// Tru parsing as unix timestamp
	var unixTime int64
	errUnix := json.Unmarshal(b, &unixTime)
	if errUnix != nil {
		return fmt.Errorf("neither time.GhTime nor unix timestamp could be parsed from %s: %w %v", string(b), errTime, errUnix)
	}

	t.Underlying = time.Unix(unixTime, 0)

	return nil
}

type Repository struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Private       bool   `json:"private"`
	HtmlUrl       string `json:"html_url"`
	Url           string `json:"url"`
	CreatedAt     GhTime `json:"created_at"`
	UpdatedAt     GhTime `json:"updated_at"`
	PushedAt      GhTime `json:"pushed_at"`
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
	Timestamp GhTime `json:"timestamp"`
	URL       string `json:"url"`
	Author    User   `json:"author"`
}

type PushWebhookPayload struct {
	Ref        string     `json:"ref"`
	HookId     int64      `json:"hook_id"`
	Repository Repository `json:"repository"`
	Pusher     User       `json:"pusher"`
	HeadCommit Commit     `json:"head_commit"`
}
