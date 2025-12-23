package model

import "testing"

func TestUser_ScanFieldsPointers(t *testing.T) {
	var u User

	fields := u.ScanFields()
	if len(fields) != 5 {
		t.Fatalf("len=%d want=5", len(fields))
	}

	idPtr, ok := fields[0].(*int)
	if !ok {
		t.Fatalf("field[0] type=%T want *int", fields[0])
	}
	loginPtr, ok := fields[1].(*string)
	if !ok {
		t.Fatalf("field[1] type=%T want *string", fields[1])
	}
	passPtr, ok := fields[2].(*string)
	if !ok {
		t.Fatalf("field[2] type=%T want *string", fields[2])
	}
	curPtr, ok := fields[3].(*int)
	if !ok {
		t.Fatalf("field[3] type=%T want *int", fields[3])
	}
	withPtr, ok := fields[4].(*int)
	if !ok {
		t.Fatalf("field[4] type=%T want *int", fields[4])
	}

	*idPtr = 10
	*loginPtr = "alice"
	*passPtr = "hash"
	*curPtr = 123
	*withPtr = 7

	if u.ID != 10 {
		t.Fatalf("ID=%d want=10", u.ID)
	}
	if u.Login != "alice" {
		t.Fatalf("Login=%q want=%q", u.Login, "alice")
	}
	if u.Password != "hash" {
		t.Fatalf("Password=%q want=%q", u.Password, "hash")
	}
	if u.Balance.Current != 123 {
		t.Fatalf("Balance.Current=%d want=123", u.Balance.Current)
	}
	if u.Balance.Withdrawn != 7 {
		t.Fatalf("Balance.Withdrawn=%d want=7", u.Balance.Withdrawn)
	}
}
