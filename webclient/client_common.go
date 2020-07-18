package webclient

import "github.com/yudeguang/distributedhttpproxy/agentcomm"

func createSwitchData() *agentcomm.TagSwitchData {
	switchData := &agentcomm.TagSwitchData{}
	switchData.ProcId = globalProcessId
	switchData.OnlyId = globalOnlyId
	switchData.ProxyAddr = globalProxyAddr
	switchData.StartTime = globalStartTime
	return switchData
}
