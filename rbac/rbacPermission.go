package rbac

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

var RbacPermission = rbacPermission{}

type rbacPermission struct {
}

var swaggerUrl = "./swagger/swagger.json"
var swaggerPathsFormat = "$.M|$.BasePath$.Path" // permission format
// TODO： 可以根据读取的配置文件自由设置rbacPermission表的名字和字段，比如module，description，summary等
var rbacPermissionTable = "rbac_permission_auto"

// 数据库权限模型
type RbacPermissionAuto struct {
	Id          uint   `json:"id" gorm:"primarykey"`
	Path        string `gorm:"type:VARCHAR(64);uniqueIndex"`
	Summary     string
	Description string
	Module      string
}

type swaggerJson struct {
	BasePath *string
	Paths    map[string]map[string]struct {
		Description *string
		Summary     *string
		XModule     map[string]interface{} `json:"x-module"`
	}
}

type rbacPermissionPath struct {
	Path        string
	Description string
	Summary     string
	Module      string
}

// fetch all path by swagger.json which have x-module attribute
// accept http or file path
func (i *rbacPermission) FetchUrlBySwagger(DB *gorm.DB, url string) (*[]rbacPermissionPath, error) {
	swaggerJson := &swaggerJson{}
	err := RbacPermission.LoadJson(url, swaggerJson)
	if err != nil {
		panic(err)
	}
	fmt.Println(swaggerJson)
	var swaggerPaths = &[]rbacPermissionPath{}
pathFor:
	for k, v := range swaggerJson.Paths {
		var swaggerPath = rbacPermissionPath{}
		path := ""
		for method, vv := range v {
			if vv.XModule["ignore"] != nil && vv.XModule["ignore"].(bool) {
				continue pathFor
			}
			path = strings.Replace(swaggerPathsFormat, "$.M", strings.ToUpper(method), -1)
			path = strings.Replace(path, "$.m", method, -1)
			swaggerPath.Description = *v[method].Description
			swaggerPath.Summary = *v[method].Summary
			if vv.XModule["module"] != nil {
				module := vv.XModule["module"].(string)
				swaggerPath.Module = module
			}
		}
		basePath := ""
		if swaggerJson.BasePath != nil {
			basePath =
				*swaggerJson.BasePath
			path = strings.Replace(path, "$.BasePath", basePath[:len(basePath)-1], -1)
		} else {
			path = strings.Replace(path, "$.BasePath", "", -1)
		}
		path = strings.Replace(path, "$.Path", k, -1)
		swaggerPath.Path = path
		*swaggerPaths = append(*swaggerPaths, swaggerPath)
	}
	fmt.Println(swaggerPaths)
	i.ReloadRbacPermission2Database(DB, swaggerPaths)
	return swaggerPaths, nil
}

func (i *rbacPermission) ReloadRbacPermission2Database(DB *gorm.DB, permssion *[]rbacPermissionPath) error {
	i.CheckTable(DB)
	i.DealNotInSwaggerApi(DB, permssion)
	// 如果module、summary、description都相同则更新path
	for _, v := range *permssion {
		var rbacPermissionAuto = RbacPermissionAuto{}
		DB.Where("description = ? and summary =? and module = ?", v.Description, v.Summary, v.Module).First(&rbacPermissionAuto)
		rbacPermissionAuto.Path = v.Path
		rbacPermissionAuto.Description = v.Description
		rbacPermissionAuto.Summary = v.Summary
		rbacPermissionAuto.Module = v.Module
		if rbacPermissionAuto.Id == 0 {
			DB.Create(&rbacPermissionAuto)
		} else {
			DB.Model(&rbacPermissionAuto).Where("id = ?", rbacPermissionAuto.Id).Updates(&rbacPermissionAuto)
		}
	}
	return nil
}

func (i *rbacPermission) DealNotInSwaggerApi(DB *gorm.DB, swaggerApis *[]rbacPermissionPath) {
	var permissions = &[]RbacPermissionAuto{}
	DB.Find(&permissions)
	for _, pv := range *permissions {
		if !i.IsInSwaggerApis(&pv, swaggerApis) {
			DB.Delete("id = ?", pv.Id)
		}
	}
}

func (i *rbacPermission) IsInSwaggerApis(rbacPermission *RbacPermissionAuto, swaggerApis *[]rbacPermissionPath) bool {
	for _, sv := range *swaggerApis {
		if RString(rbacPermission.Description) == RString(sv.Description) && RString(rbacPermission.Summary) == RString(sv.Summary) && RString(rbacPermission.Module) == RString(sv.Module) && RString(rbacPermission.Path) == RString(sv.Path) {
			return true
		}
	}
	return false
}

// String converts `any` to string.
// It's most common used converting function.
func RString(any interface{}) string {
	if any == nil {
		return ""
	}
	switch value := any.(type) {
	case int:
		return strconv.Itoa(value)
	case int8:
		return strconv.Itoa(int(value))
	case int16:
		return strconv.Itoa(int(value))
	case int32:
		return strconv.Itoa(int(value))
	case int64:
		return strconv.FormatInt(value, 10)
	case uint:
		return strconv.FormatUint(uint64(value), 10)
	case uint8:
		return strconv.FormatUint(uint64(value), 10)
	case uint16:
		return strconv.FormatUint(uint64(value), 10)
	case uint32:
		return strconv.FormatUint(uint64(value), 10)
	case uint64:
		return strconv.FormatUint(value, 10)
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(value)
	case string:
		return value
	case []byte:
		return string(value)
	case time.Time:
		if value.IsZero() {
			return ""
		}
		return value.String()
	case *time.Time:
		if value == nil {
			return ""
		}
		return value.String()
	default:
		// Empty checks.
		if value == nil {
			return ""
		}
		// Reflect checks.
		var (
			rv   = reflect.ValueOf(value)
			kind = rv.Kind()
		)
		switch kind {
		case reflect.Chan,
			reflect.Map,
			reflect.Slice,
			reflect.Func,
			reflect.Ptr,
			reflect.Interface,
			reflect.UnsafePointer:
			if rv.IsNil() {
				return ""
			}
		case reflect.String:
			return rv.String()
		}
		if kind == reflect.Ptr {
			return RString(rv.Elem().Interface())
		}
		// Finally we use json.Marshal to convert.
		if jsonContent, err := json.Marshal(value); err != nil {
			return fmt.Sprint(value)
		} else {
			return string(jsonContent)
		}
	}
}

func (i *rbacPermission) CheckTable(DB *gorm.DB) {
	if !DB.Migrator().HasTable(rbacPermissionTable) {
		DB.Migrator().CreateTable(&RbacPermissionAuto{})
		DB.Migrator().RenameTable("rbac_permission_auto", rbacPermissionTable)
	}
}

func (i *rbacPermission) LoadJson(url string, v interface{}) (err error) {
	if strings.HasPrefix(url, "http") {
		err = getJSONByHttp(url, v)
		if err != nil {
			return
		}
	} else {
		if data, err := ioutil.ReadFile(url); err != nil {
			return err
		} else {
			data = []byte(string(data))
			//读取的数据为json格式，需要进行解码
			err = json.Unmarshal(data, v)
			return err
		}
	}
	return nil
}

// getJSON fetches the contents of the given URL
// and decodes it as JSON into the given result,
// which should be a pointer to the expected data.
func getJSONByHttp(url string, result interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("cannot fetch URL %q: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected http GET status: %s", resp.Status)
	}
	fmt.Println(result)
	fmt.Println(resp.Body)
	// We could check the resulting content type
	// here if desired.
	err = json.NewDecoder(resp.Body).Decode(result)
	if err != nil {

		return fmt.Errorf("cannot decode JSON: %v", err)
	}
	return nil
}

type ResStruct struct {
	Name     string
	Children []RbacPermissionAuto
}

func GetPermissionGroupByModule(DB *gorm.DB) []ResStruct {
	var rbacPermissions = &[]RbacPermissionAuto{}
	DB.Model(&RbacPermissionAuto{}).Find(rbacPermissions)
	var res = []ResStruct{}
	for _, permission := range *rbacPermissions {
		isIn, key := inArr(permission.Module, res)
		if isIn {
			res[key].Children = append(res[key].Children, permission)
		} else {
			r := ResStruct{}
			r.Name = permission.Module
			r.Children = append(r.Children, permission)
			res = append(res, r)
		}
	}
	return res
}

func inArr(str string, res []ResStruct) (bool, int) {
	for k, v := range res {
		if v.Name == str {
			return true, k
		}
	}
	return false, 0
}
