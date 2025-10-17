package db

import "github.com/z46-dev/gomysql"

const (
	JOB_TYPE_BASH = iota
	JOB_TYPE_ANSIBLE
	JOB_TYPE_PYTHON
)

type ScheduledJob struct {
	ID       int    `gomysql:"id,primary,increment"`
	Name     string `gomysql:"name"`
	Enabled  bool   `gomysql:"enabled"`
	FilePath string `gomysql:"file_path"`
	JobType  int    `gomysql:"job_type"`
	CronExpr string `gomysql:"cron_expr"`
}

var ScheduledJobs *gomysql.RegisteredStruct[ScheduledJob]

func init() {
	var err error
	if ScheduledJobs, err = gomysql.Register(ScheduledJob{}); err != nil {
		panic(err)
	}
}
