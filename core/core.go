package core

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nsqio/go-nsq"
	"github.com/sirupsen/logrus"
	"github.com/tereus-project/tereus-go-std/queue"
	"github.com/tereus-project/tereus-transpiler-std/env"
	"github.com/tereus-project/tereus-transpiler-std/messages"
	"github.com/tereus-project/tereus-transpiler-std/metrics"
	"github.com/tereus-project/tereus-transpiler-std/sentry"
	"github.com/tereus-project/tereus-transpiler-std/storage"
)

type TranspilerContext struct {
	SentryService   *sentry.SentryService
	MetricsService  *metrics.MetricsService
	StorageService  *storage.StorageService
	QueueService    *queue.QueueService
	MessagesService *messages.MessagesService

	sourceLanguageFileExtension string
	targetLanguageFileExtension string

	transpileFunction TranspileFunction
}

type TranspileFunction func(localPath string) (string, error)

type TranspilerContextConfig struct {
	SourceLanguage              string
	SourceLanguageFileExtension string
	TargetLanguage              string
	TargetLanguageFileExtension string

	TranspileFunction TranspileFunction
}

func InitTranspiler(contextConfig *TranspilerContextConfig) {
	if len(os.Args) >= 2 {
		executeHeadless(os.Args[1], contextConfig.TranspileFunction)
		return
	}

	logrus.Infoln("Initializing transpiler...")

	err := env.LoadEnv()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load environment variables")
	}

	config := env.GetEnv()

	sentryService, err := sentry.NewSentryService()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize sentry service")
	}

	prometheusNamespace := fmt.Sprintf("transpiler_%s_%s", contextConfig.SourceLanguage, contextConfig.TargetLanguage)
	metricsService, err := metrics.NewMetricsService(prometheusNamespace)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize metrics service")
	}
	go metricsService.Listen()

	storageService, err := storage.NewStorageService()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize storage service")
	}

	queueService, err := queue.NewQueueService(config.NSQEndpoint, config.NSQLookupdEndpoint)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize queue service")
	}

	messagesService := messages.NewMessagesService(queueService)

	transpilerContext := &TranspilerContext{
		SentryService:   sentryService,
		MetricsService:  metricsService,
		StorageService:  storageService,
		QueueService:    queueService,
		MessagesService: messagesService,

		sourceLanguageFileExtension: contextConfig.SourceLanguageFileExtension,
		targetLanguageFileExtension: contextConfig.TargetLanguageFileExtension,

		transpileFunction: contextConfig.TranspileFunction,
	}
	defer transpilerContext.StopTranspiler()

	queueTopic := fmt.Sprintf("transpilation_jobs_%s_to_%s", contextConfig.SourceLanguage, contextConfig.TargetLanguage)
	err = queueService.AddHandler(queueTopic, "transpiler", func(msg *nsq.Message) error {
		logrus.Infoln("Received message")
		return transpilerContext.onSubmission(msg)
	})
	if err != nil {
		logrus.WithError(err).Fatal("Failed to add queue message handler")
	}

	logrus.Infoln("Transpiler successfully initialized!")

	// wait for signal to exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logrus.Infoln("Shutting down...")
}

func (t *TranspilerContext) StopTranspiler() {
	t.SentryService.StopSentry()
	t.MetricsService.Close()
	t.QueueService.Close()
}

// Returning a non-nil error will automatically send a REQ command to NSQ to re-queue the message.
func (t *TranspilerContext) onSubmission(m *nsq.Message) error {
	var startTime = time.Now()

	logrus.WithField("message", string(m.Body)).Info("Received message")

	var job messages.SubmissionMessage
	err := json.Unmarshal(m.Body, &job)
	if err != nil {
		logrus.WithError(err).Error("Error unmarshaling message")
		return nil
	}

	err = t.MessagesService.SendSubmissionStatus(job.ID, messages.StatusProcessing, nil)
	if err != nil {
		t.MetricsService.ObserveTranspilingDuration(messages.StatusFailed, startTime)
		return err
	}

	err = t.transpileSubmission(job.ID)
	if err != nil {
		logrus.WithError(err).WithField("job_id", job.ID).Errorf("Failed to transpile and upload job")
		t.MetricsService.ObserveTranspilingDuration(messages.StatusFailed, startTime)

		err := t.MessagesService.SendSubmissionStatus(job.ID, messages.StatusFailed, err)
		if err != nil {
			return err
		}
	} else {
		err := t.MessagesService.SendSubmissionStatus(job.ID, messages.StatusDone, nil)
		if err != nil {
			t.MetricsService.ObserveTranspilingDuration(messages.StatusFailed, startTime)
			return err
		}

		t.MetricsService.ObserveTranspilingDuration(messages.StatusDone, startTime)
		logrus.Debugf("Job '%s' completed", job.ID)
	}

	return nil
}

func (t *TranspilerContext) transpileSubmission(submissionId string) error {
	logrus.Debugln("Downloading submission files...")
	files, err := t.StorageService.DownloadSourceObjects(submissionId)
	if err != nil {
		return err
	}

	for _, file := range files {
		logrus.Debugf("Transpiling file '%s'", file.SourceFilePath)
		if !strings.HasSuffix(file.SourceFilePath, t.sourceLanguageFileExtension) {
			data, err := os.ReadFile(file.LocalPath)
			if err != nil {
				return err
			}

			err = t.StorageService.UploadTranspiledObject(submissionId, file.SourceFilePath, data)
			if err != nil {
				return err
			}

			continue
		}

		logrus.Debugf("Remixing file '%s'", file.SourceFilePath)

		output, err := t.transpileFunction(file.LocalPath)
		if err != nil {
			return err
		}

		outputPath := fmt.Sprintf("%s%s", strings.TrimSuffix(file.SourceFilePath, t.sourceLanguageFileExtension), t.targetLanguageFileExtension)

		logrus.Debugf("Uploading file '%s'", outputPath)
		err = t.StorageService.UploadTranspiledObject(submissionId, outputPath, []byte(output))
		if err != nil {
			return err
		}
	}

	return nil
}
