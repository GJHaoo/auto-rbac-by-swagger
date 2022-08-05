package main

import (
	"net/http"
	"strings"

	"os"

	"main/rbac"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

type User struct {
	Id    uint
	Name  string `gorm:"not null;unique"`
	Roles []Role `gorm:"many2many:user_roles;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Role struct {
	Id          uint
	Name        string                    `gorm:"unique_index"`
	Permissions []rbac.RbacPermissionAuto `gorm:"many2many:rbac_role_2_permission;"`
}

func GormInit() {
	dsn := "root:123456@tcp(localhost:3306)/test_rbac?charset=utf8&parseTime=True&loc=Local"
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	DB.AutoMigrate(&User{}, &Role{}, &rbac.RbacPermissionAuto{})
}

func CasbinInit() {
	dsn := "root:123456@tcp(127.0.0.1:3306)/test_rbac?charset=utf8&parseTime=True&loc=Local"
	rbac.InitCasbin(rbac.DATABASE_TYPE_MYSQL, dsn, "127.0.0.1:6379", "")
}

func DataInit() {
	permission := rbac.RbacPermissionAuto{}
	DB.Model(&rbac.RbacPermissionAuto{}).Where("path = ?", "POST|/test/auth").Find(&permission)
	role := Role{}
	role.Id = 1
	role.Name = "role1"
	DB.Create(&role)
	DB.Where("name = ?", "role1").First(&role)
	user := &User{}
	user.Id = 1
	user.Name = "user1"
	user.Roles = append(user.Roles, role)

	urlMsg := strings.Split(permission.Path, "|")
	if _, err := rbac.Enforcer.RemoveFilteredPolicy(0, rbac.RString(role.Id)); err != nil {
		panic(err)
	}

	if _, err := rbac.Enforcer.AddPolicy(rbac.RString(role.Id), urlMsg[1], urlMsg[0]); err != nil {
		panic(err)
	}
	DB.Create(&user)
}

// @summary 测试通过接口
// @description 测试通过接口
// @tags    test/pass
// @produce  json
// @security ApiKeyAuth
// @router  /test/pass [POST]
func TestApiPass(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"value": "pass"})
}

// @summary 测试验证接口
// @description 测试验证接口
// @tags    test/auth
// @produce  json
// @security ApiKeyAuth
// @router  /test/auth [POST]
func TestApiAuth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"value": "auth pass"})
}

func CasbinMiddleware(c *gin.Context) {
	var Body struct {
		UserId int `json:"userId"`
	}
	err := c.Bind(&Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		c.Abort()
		return
	}
	if Body.UserId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "userId is empty"})
		c.Abort()
		return
	}
	user := User{}
	DB.Model(&User{}).Where("id = ?", Body.UserId).Preload("Roles").First(&user)
	roleIds := []uint{}
	for _, role := range user.Roles {
		roleIds = append(roleIds, role.Id)
	}
	err = rbac.CasbinMiddlewareRole(*c.Request, roleIds)
	if err != nil {
		c.String(http.StatusUnauthorized, err.Error())
		c.Abort()
	} else {
		c.Next()
	}
}

func setupRouter() *gin.Engine {
	r := gin.Default()

	// Ping test
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	r.POST("/test/pass", TestApiPass)
	r.Use(CasbinMiddleware)
	r.POST("/test/auth", TestApiAuth)
	return r
}

func main() {
	GormInit()
	CasbinInit()
	if len(os.Args) > 1 {
		if os.Args[1] == "initApi" {
			rbac.RbacPermission.FetchUrlBySwagger(DB, os.Args[2])
		}
		if os.Args[1] == "initData" {
			DataInit()
		}
	} else {
		r := setupRouter()
		// Listen and Server in 0.0.0.0:8080
		r.Run(":8080")

	}
}
