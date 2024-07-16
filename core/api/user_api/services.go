package user_api

import (
	"fmt"
	"log"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/sql_orm"
	"github.com/daqing/airway/lib/utils"
)

func CreateRootUser(username string, password string) (*models.User, error) {
	return createUser(username, username, models.RootRole, password)
}

func CreateAdminUser(nickname, username string, password string) (*models.User, error) {
	return createUser(nickname, username, models.AdminRole, password)
}

func CreateBasicUser(nickname, username string, password string) (*models.User, error) {
	return createUser(nickname, username, models.BasicRole, password)
}

func createUser(nickname, username string, role models.UserRole, password string) (*models.User, error) {
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

	user, err := sql_orm.Insert[models.User](
		[]sql_orm.KVPair{
			sql_orm.KV("nickname", nickname),
			sql_orm.KV("username", username),
			sql_orm.KV("phone", ""),
			sql_orm.KV("email", ""),
			sql_orm.KV("avatar", ""),
			sql_orm.KV("role", role),
			sql_orm.KV("encrypted_password", enc),
			sql_orm.KV("api_token", utils.RandomHex(20)),
		},
	)

	if user != nil {
		user.EncryptedPassword = ""
	}

	return user, err
}

func LoginUser(where []sql_orm.KVPair, password string) (*models.User, error) {
	if len(where) == 0 {
		return nil, fmt.Errorf("where can't be empty")
	}

	if len(password) == 0 {
		return nil, fmt.Errorf("password can't be empty")
	}

	users, err := sql_orm.Find[models.User](
		[]string{
			"id", "username", "nickname", "phone", "email", "avatar",
			"encrypted_password", "api_token",
		},
		where,
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

func UserFromAPIToken(token string) *models.User {
	user, err := sql_orm.FindOne[models.User](
		[]string{
			"id", "username", "nickname",
			"phone", "email", "avatar",
			"role", "api_token",
		},
		[]sql_orm.KVPair{
			sql_orm.KV("api_token", token),
		},
	)

	if err != nil {
		return nil
	}

	return user
}

func CurrentUser(authToken string) *models.User {
	return userFromToken(authToken, models.AllRole)
}

func CurrentAdmin(authToken string) *models.User {
	user := CurrentUser(authToken)

	if user == nil {
		return nil
	}

	if user.IsAdmin() {
		return user
	}

	return nil
}

func userFromToken(apiToken string, role models.UserRole) *models.User {
	user := UserFromAPIToken(apiToken)
	if user == nil {
		return nil
	}

	if role == models.AllRole || user.Role == role {
		return user
	}

	return nil
}

func Users(fields []string, order string, page, limit int) ([]*models.User, error) {
	if page == 0 {
		page = 1
	}

	return sql_orm.FindLimit[models.User](
		fields,
		[]sql_orm.KVPair{},
		order,
		(page-1)*limit,
		limit,
	)
}

func Nickname(id models.IdType) string {
	user, err := sql_orm.FindOne[models.User](
		[]string{"id", "nickname"},
		[]sql_orm.KVPair{
			sql_orm.KV("id", id),
		},
	)

	if err != nil {
		return ""
	}

	return user.Nickname
}
