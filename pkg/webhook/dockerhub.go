package webhook

import (
	"encoding/json"
)

// DHHookBody is the body received from a dockerhub wehbook request
// thank you https://mholt.github.io/json-to-go/
type DHHookBody struct {
	CallbackURL string `json:"callback_url"`
	PushData    struct {
		Images   []string `json:"images"`
		PushedAt int      `json:"pushed_at"`
		Pusher   string   `json:"pusher"`
		Tag      string   `json:"tag"`
	} `json:"push_data"`
	Repository struct {
		CommentCount    int    `json:"comment_count"`
		DateCreated     int    `json:"date_created"`
		Description     string `json:"description"`
		Dockerfile      string `json:"dockerfile"`
		FullDescription string `json:"full_description"`
		IsOfficial      bool   `json:"is_official"`
		IsPrivate       bool   `json:"is_private"`
		IsTrusted       bool   `json:"is_trusted"`
		Name            string `json:"name"`
		Namespace       string `json:"namespace"`
		Owner           string `json:"owner"`
		RepoName        string `json:"repo_name"`
		RepoURL         string `json:"repo_url"`
		StarCount       int    `json:"star_count"`
		Status          string `json:"status"`
	} `json:"repository"`
}

// UnmarshalDHHook unmarshals a request body into a DHHookBody
func UnmarshalDHHook(body []byte) (resp *DHHookBody, err error) {
	err = json.Unmarshal(body, &resp)
	return
}
