package db

import (
	"fmt"
	"wayne/src/backend/controllers/auth"
	"wayne/src/backend/models"
	"wayne/src/backend/util/encode"
	"github.com/astaxie/beego/orm"
)

type DBAuth struct{}

func init() {
	auth.Register(models.AuthTypeDB, &DBAuth{})
}

type CurrentInfo struct {
	User   *models.User      `json:"user"`
	Config map[string]string `json:"config"`
}

func (*DBAuth) Authenticate(m models.AuthModel) (*models.User, error) {
	username := m.Username
	password := m.Password
	user, err := models.UserModel.GetUserByName(username)
	if err != nil {
		if err == orm.ErrNoRows {
			return nil, fmt.Errorf("username or password error!")
		}
		return nil, err
	}

	if user.Password == "" || user.Salt == "" {
		return nil, fmt.Errorf("user dons't support db login!")
	}

	passwordHashed := encode.EncodePassword(password, user.Salt)

	if passwordHashed != user.Password {
		return nil, fmt.Errorf("username or password error!")
	}
	return user, nil
}
