package user_plugin

import "time"

type User struct {
	Id                int64
	Nickname          string
	Username          string
	Phone             string
	Email             string
	Avatar            string
	Role              UserRole
	ApiToken          string
	EncryptedPassword string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (u User) TableName() string { return "users" }

type UserRole int

const (
	Basic UserRole = iota
	Admin
)