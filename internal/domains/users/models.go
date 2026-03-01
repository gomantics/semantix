package users

type User struct {
	ID      int64
	Email   string
	Created int64
	Updated int64
}

type CreateParams struct {
	Email    string
	Password string
}
