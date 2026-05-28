package service

import (
	"github.com/CheetahExchange/CheetahExchange/conf"
	"github.com/CheetahExchange/CheetahExchange/models"
	"github.com/CheetahExchange/CheetahExchange/models/mysql"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"time"
)

func CreateUser(email, password string) (*models.User, error) {
	if len(password) < 6 {
		return nil, errors.New("password must be of minimum 6 characters length")
	}
	user, err := GetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return nil, errors.New("email address is already registered")
	}

	user = &models.User{
		Email:        email,
		PasswordHash: hashPassword(password),
		UserLevel:    "v1",
	}
	return user, mysql.SharedStore().AddUser(user)
}

func RefreshAccessToken(email, password string) (string, error) {
	user, err := GetUserByEmail(email)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", errors.New("email not found or password error")
	}
	if !checkPassword(password, user.PasswordHash) {
		return "", errors.New("email not found or password error")
	}

	claim := jwt.MapClaims{
		"userId": user.Id,
		"exp":    time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	return token.SignedString([]byte(conf.GetConfig().JwtSecret))
}

func CheckToken(tokenStr string) (*models.User, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(conf.GetConfig().JwtSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claim, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("token is invalid")
	}

	userIdVal, found := claim["userId"]
	if !found {
		return nil, errors.New("bad token")
	}

	switch v := userIdVal.(type) {
	case float64:
		return GetUserById(int64(v))
	default:
		return nil, errors.New("bad token")
	}
}

func ChangePassword(email, newPassword string) error {
	user, err := GetUserByEmail(email)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	user.PasswordHash = hashPassword(newPassword)
	return mysql.SharedStore().UpdateUser(user)
}

func GetUserByEmail(email string) (*models.User, error) {
	return mysql.SharedStore().GetUserByEmail(email)
}

func GetUserById(id int64) (*models.User, error) {
	return mysql.SharedStore().GetUserById(id)
}

func GetUserByPassword(email, password string) (*models.User, error) {
	user, err := GetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	if user == nil || !checkPassword(password, user.PasswordHash) {
		return nil, errors.New("user not found or password incorrect")
	}
	return user, nil
}

func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return string(hash)
}

func checkPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
