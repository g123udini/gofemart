package model

type User struct {
	ID       int    `json:"id"`
	Login    string `json:"name"`
	Password string `json:"password"`
}

func (u *User) ScanFields() []any {
	return []any{
		&u.ID,
		&u.Login,
		&u.Password,
	}
}
