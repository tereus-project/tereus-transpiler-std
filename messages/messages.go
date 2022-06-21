package messages

import (
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tereus-project/tereus-go-std/queue"
)

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
	ID        string           `json:"id"`
	Status    SubmissionStatus `json:"status"`
	Reason    string           `json:"reason"`
	Timestamp int64            `json:"timestamp"`
}

type MessagesService struct {
	queueService *queue.QueueService
}

func NewMessagesService(queueService *queue.QueueService) *MessagesService {
	return &MessagesService{
		queueService: queueService,
	}
}

func (m *MessagesService) SendSubmissionStatus(submissionId string, status SubmissionStatus, err error) error {
	logrus.Debugf("Sending %s submission status for %s", status, submissionId)

	reason := ""
	if err != nil {
		reason = err.Error()
	}

	message, err := json.Marshal(SubmissionStatusMessage{
		ID:        submissionId,
		Status:    status,
		Reason:    reason,
		Timestamp: time.Now().UnixMilli(),
	})
	if err != nil {
		logrus.Fatal(err)
	}

	err = m.queueService.Publish("transpilation_submission_status", message)
	if err != nil {
		logrus.WithError(err).WithField("job_id", submissionId).Errorf("Error publishing status message for job")
		return err
	}

	return nil
}
