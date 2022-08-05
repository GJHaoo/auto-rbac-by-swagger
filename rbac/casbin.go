package rbac

import (
	rediswatcher "github.com/billcobbler/casbin-redis-watcher/v2"
	"github.com/casbin/casbin/v2"
	gormAdapter "github.com/casbin/gorm-adapter/v3"
	_ "github.com/go-sql-driver/mysql"
)

type DatabaseType string

var (
	DATABASE_TYPE_MYSQL      DatabaseType = "mysql"
	DATABASE_TYPE_POSTGRESQL DatabaseType = "postgresql"
	DATABASE_TYPE_SQLITE     DatabaseType = "sqlite"
)

var Enforcer *casbin.Enforcer

/*
 InitCasbin 初始化casbin
 @param databaseType 数据库类型 mysql,postgresql,sqlite
 @param databaseLink 数据库连接字符串
 @param redisAddress redis地址
 @param redisPassword redis密码
 @return
*/
func InitCasbin(databaseType DatabaseType, databaseLink string, redisAddress string, redisPassword string) {
	var err error
	adapter, err := gormAdapter.NewAdapter(string(databaseType), databaseLink, true)
	if err != nil {
		panic(err)
	}
	Enforcer, err = casbin.NewEnforcer("rbac/casbin_model.conf", adapter)
	if err != nil {
		panic(err)
	}

	watcher, err := rediswatcher.NewWatcher(redisAddress, rediswatcher.Password(redisPassword))
	if err != nil {
		panic(err)
	}
	if err = Enforcer.SetWatcher(watcher); err != nil {
		panic(err)
	}
	if err = Enforcer.LoadPolicy(); err != nil {
		panic(err)
	}
}
