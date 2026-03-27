package models

type User struct {
	ID   int64   `db:"id"`
	Name *string `db:"name"`
}

func (User) TableName() string {
	return "Users"
}

func init() {
	registerREPLModel("User", User{})
}
