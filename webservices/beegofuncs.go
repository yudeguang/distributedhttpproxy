package webservices

import (
	"fmt"

	"github.com/yudeguang/oldbeego"
)

func registBeegoFuncMap() {
	oldbeego.AddFuncMap("IsEqual", isEqual)
	oldbeego.AddFuncMap("NotEqual", notEqual)
}

//比较两个类型是否相等，全部转换为字符串比较
func isEqual(s1 interface{}, s2 interface{}) bool {
	return fmt.Sprint(s1) == fmt.Sprint(s2)
}
func notEqual(s1 interface{}, s2 interface{}) bool {
	return fmt.Sprint(s1) != fmt.Sprint(s2)
}
