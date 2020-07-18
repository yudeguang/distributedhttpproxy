package webservices

import (
	"encoding/json"
	"github.com/astaxie/beego"
)

type baseController struct {
	beego.Controller
}
type tagResultStruct struct {
	Status int
	Data   interface{}
}

func (this *baseController) replyERRJson(obj interface{}) {
	var o = &tagResultStruct{Status: 0, Data: obj}
	data, err := json.MarshalIndent(o, "\t", "\t")
	if err != nil {
		this.Ctx.WriteString("ERR:" + err.Error())
	} else {
		this.Ctx.WriteString(string(data))
	}
}
func (this *baseController) replyOKJson(obj interface{}) {
	var o = &tagResultStruct{Status: 1, Data: obj}
	data, err := json.MarshalIndent(o, "\t", "\t")
	if err != nil {
		this.Ctx.WriteString("ERR:" + err.Error())
	} else {
		this.Ctx.WriteString(string(data))
	}
}
func (this *baseController) replyJson(obj interface{}) {
	data, err := json.MarshalIndent(obj, "\t", "\t")
	if err != nil {
		this.Ctx.WriteString("ERR:" + err.Error())
	} else {
		this.Ctx.WriteString(string(data))
	}
}
