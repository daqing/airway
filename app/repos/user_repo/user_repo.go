package user_repo

import (
	"fmt"
	"log"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/orm"
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

	user, err := orm.Insert[models.User](
		orm.DB(),
		orm.MultiFields(
			orm.Eq("nickname", nickname),
			orm.Eq("username", username),
			orm.Eq("phone", ""),
			orm.Eq("email", ""),
			orm.Eq("avatar", ""),
			orm.Eq("role", role),
			orm.Eq("encrypted_password", enc),
			orm.Eq("api_token", utils.RandomHex(20)),
		),
	)

	if user != nil {
		user.EncryptedPassword = ""
	}

	return user, err
}

func LoginUser(cond orm.CondBuilder, password string) (*models.User, error) {
	users, err := orm.Find[models.User](
		orm.DB(),
		[]string{
			"id", "username", "nickname", "phone", "email", "avatar",
			"encrypted_password", "api_token",
		},
		cond,
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
	user, err := orm.FindOne[models.User](
		orm.DB(),
		[]string{
			"id", "username", "nickname",
			"phone", "email", "avatar",
			"role", "api_token",
		},

		orm.Eq("api_token", token),
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

	return orm.FindLimit[models.User](
		orm.DB(),
		fields,
		orm.EmptyCond{},
		order,
		(page-1)*limit,
		limit,
	)
}

func Nickname(id models.IdType) string {
	user, err := orm.FindOne[models.User](
		orm.DB(),
		[]string{"id", "nickname"},

		orm.Eq("id", id),
	)

	if err != nil {
		return ""
	}

	return user.Nickname
}
