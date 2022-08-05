package rbac

import (
	"errors"
	"fmt"

	"net/http"
)

type userStruct struct {
	Id    *uint
	Roles []struct {
		Id   *uint
		Name *string
	}
}

//Casbin 允许接口跨域请求
func CasbinMiddlewareRole(r http.Request, roleIds []uint) error {
	isOk := false
	for _, v := range roleIds {
		// 根据角色获取权限
		fmt.Println(RString(v), r.URL.Path, r.Method)
		ok, err := Enforcer.Enforce(RString(v), r.URL.Path, r.Method)
		if err != nil {
			fmt.Printf("%v", err)
			panic("1##权限验证出错")
		}
		if ok {
			isOk = true
			return nil
		}
	}
	if !isOk {
		return errors.New("您没有权限访问该路径")
	}
	return nil
}

// IsRoot 是否是超级管理员，根据自己的系统管理员用户进行修改
func IsRoot(user *userStruct) bool {
	for i := range user.Roles {
		if *user.Roles[i].Name == "root" || *user.Roles[i].Name == "admin" { // TODO: 配置管理员角色
			return true
		}
	}
	return false
}

func CasbinMiddlewareUser(r *http.Request, userId uint) error {
	// 根据角色获取权限
	ok, err := Enforcer.Enforce(RString(userId), r.URL.Path, r.Method)
	if err != nil {
		panic("1##权限验证出错")
	}
	if ok {
		return nil
	}
	return errors.New("您没有权限访问该路径")
}
