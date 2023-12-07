package webservices

// 查看当前都有哪些活跃的客户端
func GetActiveAgentList() (ret []string) {
	list, err := pDBHelper.GetAgentRecordList(" 1=1 ")
	if err != nil {
		return nil
	}
	for _, v := range list {
		if v.IsActive == 1 {
			ret = append(ret, v.OnlyId)
		}
	}
	return ret
}
