package webservices

import (
	"log"
	"strings"
)

type agentController struct {
	baseController
}

func (this *agentController) ListAgent() {
	searchonlyid := strings.TrimSpace(this.GetString("searchonlyid"))
	searchgroupname := strings.TrimSpace(this.GetString("searchgroupname"))
	lstSearch := []string{}
	if searchonlyid != "" {
		lstSearch = append(lstSearch, "OnlyId like '%"+searchonlyid+"%'")
	}
	if searchgroupname != "" {
		lstSearch = append(lstSearch, "GroupName like '%"+searchgroupname+"%'")
	}
	lstAgent, err := pDBHelper.GetAgentRecordList(strings.Join(lstSearch, " AND "))
	if err != nil {
		this.Ctx.WriteString("查询Agent列表失败:" + err.Error())
		return
	}
	this.Data["searchonlyid"] = searchonlyid
	this.Data["searchgroupname"] = searchgroupname
	this.Data["AgentList"] = lstAgent
	this.TplName = "page_agent_list.html"
}

//测试提交请求
func (this *agentController) CheckProxyUrl() {
	log.Println(this.GetString("proxyurl"))
	proxyurl := this.GetString("proxyurl")
	proxygroup := this.GetString("proxygroup")
	if proxyurl == "" {
		this.replyERRJson("参数错误")
		return
	}
	proxygroup = strings.Replace(proxygroup, " ", "", -1)
	proxygroup = strings.Replace(proxygroup, ",", ";", -1)
	var lstGroups = []string{}
	if proxygroup != "" {
		lstGroups = strings.Split(proxygroup, ";")
	}
	html, onlyid, err := ProxyGet(proxyurl, lstGroups...)
	if err != nil {
		this.replyERRJson(err.Error())
		return
	}
	var v struct {
		Err    error
		OnlyId string
		Html   string
	}
	v.Err = err
	v.OnlyId = onlyid
	v.Html = html
	this.replyOKJson(v)
}

//删除一个agent信息,删除后会重新再上来
func (this *agentController) DeleteAgent() {
	id, err := this.GetInt("id")
	if err != nil {
		this.replyERRJson("参数错误:" + err.Error())
		return
	}
	err = pDBHelper.DeleteAgent(id)
	if err != nil {
		this.replyERRJson("删除失败:" + err.Error())
	} else {
		this.replyOKJson("删除成功")
	}
}

//设置启用或禁用
func (this *agentController) DisableAgent() {
	id, err := this.GetInt("id")
	if err != nil {
		this.replyERRJson("参数错误:" + err.Error())
		return
	}
	disable, err := this.GetInt("disable")
	if err != nil {
		this.replyERRJson("参数错误:" + err.Error())
		return
	}
	err = pDBHelper.Exec("UPDATE AgentList SET Disabled=? WHERE Id=?", disable, id)
	if err != nil {
		this.replyERRJson("删除失败:" + err.Error())
	} else {
		this.replyOKJson("删除成功")
	}
}

func (this *agentController) EditAgent() {
	id, err := this.GetInt("id")
	if err != nil {
		this.Ctx.WriteString("参数错误:" + err.Error())
		return
	}
	var onlyId, groupName string
	err = pDBHelper.QueryRow("SELECT OnlyId,GroupName FROM AgentList Where Id=?", id).Scan(&onlyId, &groupName)
	if err != nil {
		this.Ctx.WriteString("查询错误:" + err.Error())
		return
	}
	this.Data["id"] = id
	this.Data["onlyid"] = onlyId
	this.Data["groupname"] = groupName
	this.TplName = "page_edit.html"
}

func (this *agentController) SaveAgentConfig() {
	id, err := this.GetInt("id")
	if err != nil {
		this.replyERRJson("参数错误:" + err.Error())
		return
	}
	groupname := strings.TrimSpace(this.GetString("groupname"))
	err = pDBHelper.Exec("UPDATE AgentList SET GroupName=? WHERE Id=?", groupname, id)
	if err != nil {
		this.replyERRJson("保存失败:" + err.Error())
	} else {
		this.replyOKJson("保存成功")
	}
}
