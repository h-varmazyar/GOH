package consts

// JobStatus is the state of a queued background job. Every job table stores
// the same values, so the job runner and the job queue repository serve them all.
type JobStatus string

const (
	JobPending  JobStatus = "PENDING"
	JobRunning  JobStatus = "RUNNING"
	JobDone     JobStatus = "DONE"
	JobFailed   JobStatus = "FAILED"
	JobCanceled JobStatus = "CANCELED"
)

// JobActive are the statuses of jobs that are not finished.
var JobActive = []JobStatus{JobPending, JobRunning}

func (s JobStatus) Valid() bool {
	switch s {
	case JobPending, JobRunning, JobDone, JobFailed, JobCanceled:
		return true
	}
	return false
}
