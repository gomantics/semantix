package gittokens

import "github.com/gomantics/semantix/internal/libs/gitrepo"

type GitToken struct {
	ID       int64            `json:"id"`
	Name     string           `json:"name"`
	Provider gitrepo.Provider `json:"provider"`
	Token    string           `json:"-"`
	Hint     string           `json:"hint,omitempty"`
	Created  int64            `json:"created"`
}

type CreateParams struct {
	Name     string
	Provider gitrepo.Provider
	Token    string
}
