package model

type User struct {
	Id       int    `json:"id"`
	Login    string `json:"name"`
	Password string `json:"password"`
}

func (u *User) ScanFields() []any {
	return []any{
		&u.Id,
		&u.Login,
		&u.Password,
	}
}
