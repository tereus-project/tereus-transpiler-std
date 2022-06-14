package submission

type SubmissionMessage struct {
	ID string `json:"id"`
}

type SubmissionStatus string

const (
	StatusPending    SubmissionStatus = "pending"
	StatusProcessing SubmissionStatus = "processing"
	StatusDone       SubmissionStatus = "done"
	StatusFailed     SubmissionStatus = "failed"
)

type SubmissionStatusMessage struct {
	ID     string           `json:"id"`
	Status SubmissionStatus `json:"status"`
	Reason string           `json:"reason"`
}
