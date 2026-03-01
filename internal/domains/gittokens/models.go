package gittokens

type GitToken struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Token    string `json:"-"`
	Hint     string `json:"hint,omitempty"`
	Created  int64  `json:"created"`
}

type CreateParams struct {
	Name     string
	Provider string
	Token    string
}
