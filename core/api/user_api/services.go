package user_api

import (
	"fmt"
	"log"

	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
)

func CreateBasicUser(nickname, username string, password string) (*User, error) {
	return createUser(nickname, username, basicRole, password)
}

func CreateAdminUser(nickname, username string, password string) (*User, error) {
	return createUser(nickname, username, adminRole, password)
}

func createUser(nickname, username string, role userRole, password string) (*User, error) {
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
			repo.KV("role", adminRole),
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

func UserFromAPIToken(token string) *User {
	user, err := repo.FindRow[User](
		[]string{
			"id", "username", "nickname",
			"phone", "email", "avatar",
			"role", "api_token",
		},
		[]repo.KVPair{
			repo.KV("api_token", token),
		},
	)

	if err != nil {
		return nil
	}

	return user
}

func CurrentUser(authToken string) *User {
	return userFromToken(authToken, basicRole)
}

func CurrentAdmin(authToken string) *User {
	return userFromToken(authToken, adminRole)
}

func userFromToken(apiToken string, role userRole) *User {
	user := UserFromAPIToken(apiToken)
	if user == nil {
		return nil
	}

	if user.Role == role {
		return user
	}

	return nil
}

func CheckAdmin(authToken string) bool {
	ok, err := repo.Exists[User]([]repo.KVPair{
		repo.KV("api_token", authToken),
		repo.KV("role", adminRole),
	})

	if err != nil {
		log.Println("Error checking admin: ", err)
		return false
	}

	return ok
}

func Users(fields []string, order string, page, limit int) ([]*User, error) {
	if page == 0 {
		page = 1
	}

	return repo.FindLimit[User](
		fields,
		[]repo.KVPair{},
		order,
		(page-1)*limit,
		limit,
	)
}

func Nickname(id int64) string {
	user, err := repo.FindRow[User](
		[]string{"id", "nickname"},
		[]repo.KVPair{
			repo.KV("id", id),
		},
	)

	if err != nil {
		return ""
	}

	return user.Nickname
}
