package model

type User struct {
	ID       int     `json:"id"`
	Login    string  `json:"name"`
	Password string  `json:"password"`
	Balance  Balance `json:"balance"`
}

type Balance struct {
	Current   int `json:"current"`
	Withdrawn int `json:"withdraw"`
}

func (u *User) ScanFields() []any {
	return []any{
		&u.ID,
		&u.Login,
		&u.Password,
		&u.Balance.Current,
		&u.Balance.Withdrawn,
	}
}
