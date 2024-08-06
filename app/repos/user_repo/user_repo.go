package user_repo

import (
	"fmt"
	"log"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/sql_orm"
	"github.com/daqing/airway/lib/utils"
)

func CreateRootUser(username, password string) (*models.User, error) {
	return createUser(username, username, password, models.RootRole)
}

func CreateAdminUser(nickname, username, password string) (*models.User, error) {
	return createUser(nickname, username, password, models.AdminRole)
}

func CreateBasicUser(nickname, username, password string) (*models.User, error) {
	return createUser(nickname, username, password, models.BasicRole)
}

// repo function will skip validations
func createUser(nickname, username, password string, role models.UserRole) (*models.User, error) {
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
