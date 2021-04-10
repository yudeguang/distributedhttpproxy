package webservices

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/yudeguang/distributedhttpproxy/agentcomm"
	"github.com/yudeguang/distributedhttpproxy/common"
	"log"
	"os"
	"strings"
	"time"
)

var pDBHelper = &clsDBHelper{}

func replaceSQLChar(sour string) string {
	sour = strings.Replace(sour, "'", "''", -1)
	sour = strings.Replace(sour, "\\", "\\\\", -1)
	return sour
}

//数据库辅助类
type clsDBHelper struct {
	pSqliteDB *sql.DB
}

//定义需要创建表的sql语句
var lstSqlText = []string{
	`CREATE TABLE IF NOT EXISTS AgentList(
		"Id"  		 INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
		"OnlyId" 	 VARCHAR(300) NOT NULL,
		"OnlyName" 	 VARCHAR(300) DEFAULT '',
		"ProxyAddr"  VARCHAR(200) DEFAULT '',
		"Priority" 	 INT DEFAULT  0,
		"ProcessId"  VARCHAR(20)  DEFAULT '',
		"IsBusy"	 INT DEFAULT 0,
		"IsActive"	 INT DEFAULT 0,
		"Disabled"   INT DEFAULT 0,
		"GroupName"  VARCHAR(50) DEFAULT '',
		"ReportTime" VARCHAR(50)  DEFAULT '',
		"LastUseTime" VARCHAR(50) DEFAULT '')`,
}

//定义AgentList表结构
type tagAgentInfoRecord struct {
	Id          int
	OnlyId      string
	ProxyAddr   string
	Priority    int
	ProcessId   string
	IsBusy      int
	IsActive    int
	Disabled    int
	GroupName   string
	ReportTime  string
	LastUseTime string
	Resv1       interface{} //用来向页面传递参数
}

//打开数据库连接
func (this *clsDBHelper) DBOpen() error {
	var err error
	//ybs20210410,加入一个配置文件，如果有这个配置文件才使用文件数据库，否则使用内存数据库(这样每次启动就不会保留上次活跃的客户端)
	useFileDB := false;
	if _,err = os.Stat("./use_file_db.flag");err == nil{
		useFileDB = true
	}
	if !useFileDB {
		//用内存数据库
		this.pSqliteDB, err = sql.Open("sqlite3", ":memory:")
	} else {
		this.pSqliteDB, err = sql.Open("sqlite3", "database.db3")
	}
	if err != nil {
		return err
	}
	for _, text := range lstSqlText {
		_, err = this.pSqliteDB.Exec(text)
		if err != nil {
			return err
		}
	}
	go this.loopCheckThread()
	return nil
}

//关闭数据库连接
func (this *clsDBHelper) DBClose() {
	if this.pSqliteDB != nil {
		this.pSqliteDB.Close()
		this.pSqliteDB = nil
	}
}

//间隔检查数据库，设置长时间没心跳的设置为不活跃
func (this *clsDBHelper) loopCheckThread() {
	for {
		//超过一定秒数的都设置为不活跃了
		minTime := time.Now().Add(time.Second * -15).Format("2006-01-02 15:04:05")
		this.Exec("UPDATE AgentList SET IsActive=0 WHERE IsActive=1 AND ReportTime<=?", minTime)
		time.Sleep(5 * time.Second)
	}
}

//获得连接
func (this *clsDBHelper) GetDB() *sql.DB {
	return this.pSqliteDB
}

//记录sql语句错误
func (this *clsDBHelper) logSQL(sqlText string, err error) {
	if err != nil {
		pLogger.Log("执行SQL语句错误:\r\n" + sqlText + "\r\n错误:" + err.Error())
	} else {
		pLogger.Log("执行SQL语句成功:\r\n" + sqlText)
	}
}

//直接执行某个SQL语句
func (this *clsDBHelper) Exec(sqlText string, args ...interface{}) error {
	_, err := this.pSqliteDB.Exec(sqlText, args...)
	if err != nil {
		this.logSQL(sqlText, err)
	}
	return err
}
func (this *clsDBHelper) QueryRow(query string, args ...interface{}) *sql.Row {
	return this.pSqliteDB.QueryRow(query, args...)
}

//处理主连接的信息入库,这里是设备存活信息
func (this *clsDBHelper) UpdateAgentRecord(switchData *agentcomm.TagSwitchData) error {
	agentRecord, err := pDBHelper.GetAgentRecordByOnlyId(switchData.OnlyId)
	isInsert := false
	if err != nil {
		if err == sql.ErrNoRows {
			isInsert = true
		} else {
			return err
		}
	}
	var nowTime = common.GetNowTime()
	var sqlText = ""
	if isInsert { //没有记录,插入
		sqlText = fmt.Sprintf("INSERT INTO AgentList(OnlyId,ProxyAddr,ProcessId,ReportTime,IsActive,LastUseTime) VALUES ('%s','%s','%s','%s',1,'%s')",
			replaceSQLChar(switchData.OnlyId),
			switchData.ProxyAddr,
			switchData.ProcId,
			nowTime,
			"2000-01-01 00:00:00.000")
	} else {
		sqlText = "UPDATE AgentList SET ReportTime='" + nowTime + "'"
		if agentRecord.IsActive != 1 {
			sqlText += ",IsActive=1"
		}
		if agentRecord.ProxyAddr != switchData.ProxyAddr {
			sqlText += ",ProxyAddr='" + switchData.ProxyAddr + "'"
		}
		if agentRecord.ProcessId != switchData.ProcId {
			sqlText += ",ProcessId='" + switchData.ProcId + "'"
		}
		sqlText += " WHERE OnlyId='" + replaceSQLChar(switchData.OnlyId) + "'"
	}
	if _, err = this.pSqliteDB.Exec(sqlText); err != nil {
		this.logSQL(sqlText, err)
		return err
	}
	return nil
}

//查询已经存在的Agent列表
func (this *clsDBHelper) GetAgentRecordList(wheresql string) ([]*tagAgentInfoRecord, error) {
	var lstAgent = []*tagAgentInfoRecord{}
	var sqlText = `SELECT Id,OnlyId,ProxyAddr,Priority,ProcessId,IsBusy,IsActive,Disabled,GroupName,ReportTime,LastUseTime FROM AgentList `
	if wheresql != "" {
		sqlText += " WHERE " + wheresql
	}
	sqlText += " ORDER BY OnlyId"
	log.Println(sqlText)
	rows, err := this.pSqliteDB.Query(sqlText)
	if err != nil {
		return lstAgent, err
	}
	defer rows.Close()
	for rows.Next() {
		item := &tagAgentInfoRecord{}
		err = rows.Scan(&item.Id,
			&item.OnlyId,
			&item.ProxyAddr,
			&item.Priority,
			&item.ProcessId,
			&item.IsBusy,
			&item.IsActive,
			&item.Disabled,
			&item.GroupName,
			&item.ReportTime,
			&item.LastUseTime,
		)
		if err != nil {
			return lstAgent, err
		}
		lstAgent = append(lstAgent, item)
	}
	return lstAgent, err
}

//查询一条记录
func (this *clsDBHelper) getOneAgentRecordWithSQL(wheresql string) (*tagAgentInfoRecord, error) {
	var sqlText = "SELECT Id,OnlyId,ProxyAddr,Priority,ProcessId,IsBusy,IsActive,ReportTime,LastUseTime FROM AgentList"
	sqlText += " WHERE " + wheresql + " LIMIT 1"
	var item = &tagAgentInfoRecord{}
	var err = this.pSqliteDB.QueryRow(sqlText).Scan(
		&item.Id,
		&item.OnlyId,
		&item.ProxyAddr,
		&item.Priority,
		&item.ProcessId,
		&item.IsBusy,
		&item.IsActive,
		&item.ReportTime,
		&item.LastUseTime,
	)
	return item, err
}

//根据uniqueid条件查询一条记录
func (this *clsDBHelper) GetAgentRecordByOnlyId(onlyId string) (*tagAgentInfoRecord, error) {
	var whereSql = fmt.Sprintf("OnlyId='%s'", replaceSQLChar(onlyId))
	return this.getOneAgentRecordWithSQL(whereSql)
}

//根据Id查询一条Agent记录
func (this *clsDBHelper) GetAgentRecordById(Id int) (*tagAgentInfoRecord, error) {
	var whereSql = fmt.Sprintf("Id=%d", Id)
	return this.getOneAgentRecordWithSQL(whereSql)
}

//设置某个onlyid的全为不存活状态,一般是因为agent断开了
func (this *clsDBHelper) SetIsActiveByOnlyId(onlyId string, isActive int) {
	sqlText := fmt.Sprintf("UPDATE AgentList SET IsActive=%d WHERE OnlyId='%s'", isActive, replaceSQLChar(onlyId))
	this.Exec(sqlText)
}

//删除一个条目
func (this *clsDBHelper) DeleteAgent(Id int) error {
	err := this.Exec("DELETE FROM AgentList WHERE Id=?", Id)
	return err
}
