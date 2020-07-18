package webservices

import (
	"fmt"
	"github.com/astaxie/beego"
)

func registBeegoFuncMap() {
	beego.AddFuncMap("IsEqual", isEqual)
	beego.AddFuncMap("NotEqual", notEqual)
}

//比较两个类型是否相等，全部转换为字符串比较
func isEqual(s1 interface{}, s2 interface{}) bool {
	return fmt.Sprint(s1) == fmt.Sprint(s2)
}
func notEqual(s1 interface{}, s2 interface{}) bool {
	return fmt.Sprint(s1) != fmt.Sprint(s2)
}
