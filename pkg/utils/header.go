package utils

import (
	"encoding/json"
	"net/http"

	"jusha/mcp/pkg/model"
)

// 从 Header 解析完整 user
func GetUserFromHeader(r *http.Request) (*model.User, error) {
	userJson := r.Header.Get("X-User")
	if userJson == "" {
		return nil, http.ErrNoCookie // 用作未认证标志
	}
	var user model.User
	if err := json.Unmarshal([]byte(userJson), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// 将user信息写入到Header中
func SetUserToHeader(r *http.Request, user *model.User) error {
	userJson, err := json.Marshal(user)
	if err != nil {
		return err
	}
	r.Header.Set("X-User", string(userJson))
	return nil
}
