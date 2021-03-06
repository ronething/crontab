package common

import (
	"encoding/json"
	"fmt"
	"github.com/gorhill/cronexpr"
	"strings"
	"syscall"
	"time"
)

// etcd 任务
type Job struct {
	Name     string `json:"name"`      // 任务名
	Command  string `json:"command"`   // shell 命令
	CronExpr string `json:"cron_expr"` // cron 表达式
}

//任务调度计划
type JobSchedulePlan struct {
	Job      *Job
	Expr     *cronexpr.Expression // 解析好的 cron 表达式
	NextTime time.Time            // 下次调度时间
}

// 任务执行状态
type JobExecuteInfo struct {
	Job      *Job
	PlanTime time.Time // 理论上的调度时间
	RealTime time.Time // 实际的调度时间
	Pid      int       // 任务主进程 pid
}

// HTTP 接口应答
type Response struct {
	Errno int         `json:"errno"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}

// 变化事件
type JobEvent struct {
	EventType int // SAVE or DELETE
	Job       *Job
}

// 任务执行结果
type JobExecuteResult struct {
	ExecuteInfo *JobExecuteInfo
	Output      []byte
	Err         error
	StartTime   time.Time
	EndTime     time.Time
}

type JobLog struct {
	JobName      string `json:"job_name" bson:"jobName"`
	Command      string `json:"command" bson:"command"`
	Err          string `json:"err" bson:"err"`
	Output       string `json:"output" bson:"output"`
	PlanTime     int64  `json:"plan_time" bson:"planTime"`
	ScheduleTime int64  `json:"schedule_time" bson:"scheduleTime"`
	StartTime    int64  `json:"start_time" bson:"startTime"`
	EndTime      int64  `json:"end_time" bson:"endTime"`
}

// 日志批次
type LogBatch struct {
	Logs []interface{}
}

// 任务执行日志过滤条件
type JobLogFilter struct {
	JobName string `bson:"jobName"`
}

// 任务日志排序规则
type SortLogByStartTime struct {
	SortOrder int `bson:"startTime"` // -1
}

func BuildResponse(errno int, msg string, data interface{}) (resp []byte, err error) {
	var (
		response Response
	)
	response.Errno = errno
	response.Msg = msg
	response.Data = data

	if resp, err = json.Marshal(response); err != nil {
		return
	}

	return
}

// 反序列化 job
func UnpackJob(value []byte) (ret *Job, err error) {
	var (
		job *Job
	)

	job = &Job{}
	if err = json.Unmarshal(value, job); err != nil {
		return
	}

	ret = job

	return

}

// 提取任务名
func ExtractJobName(jobKey string) string {
	return strings.TrimPrefix(jobKey, JobSaveDir)
}

// 提取任务名
func ExtractKillerName(killerKey string) string {
	return strings.TrimPrefix(killerKey, JobKillerDir)
}

// 提取 worker ip
func ExtractWorkerIP(regKey string) string {
	return strings.TrimPrefix(regKey, JobWorkerDir)
}

func BuildJobEvent(eventType int, job *Job) (jobEvent *JobEvent) {
	return &JobEvent{
		EventType: eventType,
		Job:       job,
	}
}

// 构造任务执行计划
func BuildJobSchedulePlan(job *Job) (jobSchedulePlan *JobSchedulePlan, err error) {
	var (
		expr *cronexpr.Expression
	)

	// 解析 job cron 表达式
	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {
		fmt.Println("cron 解析出错", err, job)
		return
	}

	jobSchedulePlan = &JobSchedulePlan{
		Job:      job,
		Expr:     expr,
		NextTime: expr.Next(time.Now()),
	}

	return

}

// 构造执行状态信息
func BuildJobExecuteInfo(jobSchedulePlan *JobSchedulePlan) (jobExecuteInfo *JobExecuteInfo) {
	jobExecuteInfo = &JobExecuteInfo{
		Job:      jobSchedulePlan.Job,
		PlanTime: jobSchedulePlan.NextTime, // 计算得来的调度时间
		RealTime: time.Now(),               // 真实调度时间
		Pid:      0,                        // 默认为 0，说明任务没有执行
	}

	return

}

// 强杀任务
func (j *JobExecuteInfo) KillJob() (err error) {
	if j.Pid == 0 {
		return
	}
	if err = syscall.Kill(-j.Pid, syscall.SIGKILL); err != nil {
		return
	}
	return
}
