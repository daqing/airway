package user_api

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
		[]repo.KVPair{
			repo.KV("nickname", nickname),
			repo.KV("username", username),
			repo.KV("phone", ""),
			repo.KV("email", ""),
			repo.KV("avatar", ""),
			repo.KV("role", role),
			repo.KV("encrypted_password", enc),
			repo.KV("api_token", utils.RandomHex(20)),
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
		[]repo.KVPair{
			repo.KV("username", username),
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
		[]repo.KVPair{
			repo.KV("username", username),
			repo.KV("role", AdminRole),
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

func CurrentUser(authToken string) *User {
	return userFromToken(authToken, BasicRole)
}

func CurrentAdmin(authToken string) *User {
	return userFromToken(authToken, AdminRole)
}

func userFromToken(authToken string, role UserRole) *User {
	user, err := repo.FindRow[User]([]string{"id"}, []repo.KVPair{
		repo.KV("api_token", authToken),
		repo.KV("role", role),
	})

	if err != nil {
		return nil
	}

	return user
}

func CheckAdmin(authToken string) bool {
	ok, err := repo.Exists[User]([]repo.KVPair{
		repo.KV("api_token", authToken),
		repo.KV("role", AdminRole),
	})

	if err != nil {
		log.Println("Error checking admin: ", err)
		return false
	}

	return ok
}
