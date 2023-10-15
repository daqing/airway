package user_plugin

import (
	"fmt"
	"log"

	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
)

func CreateUser(nickname, username string, role UserRole, password string) (*User, error) {
	if len(nickname) == 0 {
		return nil, fmt.Errorf("nickname can't be empty")
	}

	if len(username) == 0 {
		return nil, fmt.Errorf("username can't be empty")
	}

	if len(password) == 0 {
		return nil, fmt.Errorf("password can't be empty")
	}

	enc, err := utils.EncryptPassword(password)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	user, err := repo.Insert[User](
		[]repo.KeyValueField{
			repo.NewKV("nickname", nickname),
			repo.NewKV("username", username),
			repo.NewKV("phone", ""),
			repo.NewKV("email", ""),
			repo.NewKV("avatar", ""),
			repo.NewKV("role", role),
			repo.NewKV("encrypted_password", enc),
			repo.NewKV("api_token", utils.RandomHex(20)),
		},
	)

	if user != nil {
		user.EncryptedPassword = ""
	}

	return user, err
}

func LoginUser(username string, password string) (*User, error) {
	if len(username) == 0 {
		return nil, fmt.Errorf("username can't be empty")
	}

	if len(password) == 0 {
		return nil, fmt.Errorf("password can't be empty")
	}

	users, err := repo.Find[User](
		[]string{
			"id", "username", "nickname", "phone", "email", "avatar",
			"encrypted_password", "api_token",
		},
		[]repo.KeyValueField{
			repo.NewKV("username", username),
		},
	)

	if err != nil {
		return nil, err
	}

	if len(users) != 1 {
		return nil, fmt.Errorf("users should have only one record")
	}

	user := users[0]

	if utils.ComparePassword(user.EncryptedPassword, password) {
		user.EncryptedPassword = ""
		return user, nil
	}

	return nil, fmt.Errorf("password is not correct")
}

func LoginAdmin(username string, password string) (*User, error) {
	if len(username) == 0 {
		return nil, fmt.Errorf("username can't be empty")
	}

	if len(password) == 0 {
		return nil, fmt.Errorf("password can't be empty")
	}

	users, err := repo.Find[User](
		[]string{
			"id", "username", "nickname", "phone", "email", "avatar",
			"encrypted_password", "api_token",
		},
		[]repo.KeyValueField{
			repo.NewKV("username", username),
			repo.NewKV("role", Admin),
		},
	)

	if err != nil {
		return nil, err
	}

	if len(users) != 1 {
		return nil, fmt.Errorf("users should have only one record")
	}

	user := users[0]

	if utils.ComparePassword(user.EncryptedPassword, password) {
		user.EncryptedPassword = ""

		return user, nil
	}

	return nil, fmt.Errorf("password is not correct")
}