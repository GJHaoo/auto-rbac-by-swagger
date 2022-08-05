# 基于gin、casbin、swag自动生成鉴权路由的rbac demo
## 操作步骤
  git clone git@github.com:GJHaoo/auto-rbac-by-swagger.git  
  go mod tidy  
  swag init  

  此条命令自动生成rbac_permission_auto表（所有权限路由表） 
  go run main.go initApi docs/swagger.json

  go run main.go initData  
  go run main.go  

## 权限验证
1.验证用户id为1的接口权限 
  命令行 \
    curl -X POST \
    http://localhost:8080/test/auth \
    -H 'content-type: application/json' \
    -d '{"userId":1}'

  返回值：{"value":"auth pass"}

2.验证用户id为2的接口权限 
  命令行 \
    curl -X POST \
    http://localhost:8080/test/auth \
    -H 'content-type: application/json' \
    -d '{"userId":2}'

  返回值：您没有权限访问该路径

3.验证非中间件鉴权的接口 
  命令行 \
  curl -X POST \
    http://localhost:8080/test/pass \
    -H 'content-type: application/json' \
    -d '{"userId":2}'

  返回值：{"value":"pass"}
  



## 实现方法和使用说明如下：
  1.借助swagger生成的swagger.json获取需要做权限判断的路由，使用cmd的方法执行FetchUrlBySwagger()函数，此函数会将需要做权限判断的路由写入数据库中，并自动生成一张数据库表（rbac_permission_auto）
  2.在角色表或者用户表中引入该表的关联表，例如：

    Role:
      type RbacRole struct {
        Id          *uint
        Name        *string                  
        Permissions []rbac.RbacPermissionAuto `gorm:"many2many:rbac_role_2_permission;"`
      }

    User:
      type User struct{
        Id *uint
        NickName *string
        ....
        Permissions []rbac.RbacPermissionAuto `gorm:"many2many:user_2_permission;"`
      }

  3.我们使用casbin来判断权限是否通过，因此需要在项目初始化的时候初始化casbin，引入当前包中了InitCasbin()函数
  
    // 初始化文件
    import	"codeup.aliyun.com/sevenfifteen/qk-library/link/qrbac"
    ...
    rbac.InitCasbin()
    

  4.使用casbin中间件，放在您需要判断的路由前即可

    Role
			group.Middleware(qrbac.CasbinMiddlewareRole)

    User
			group.Middleware(qrbac.CasbinMiddlewareUser)
    
  5.GetPermissionGroupByModule()函数可以获取根据module字段分数的数据结构，供前端展示使用

