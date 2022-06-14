package core

import (
	"fmt"

	"github.com/nsqio/go-nsq"
	"github.com/sirupsen/logrus"
	"github.com/tereus-project/tereus-go-std/queue"
	"github.com/tereus-project/tereus-transpiler-std/env"
	"github.com/tereus-project/tereus-transpiler-std/metrics"
	"github.com/tereus-project/tereus-transpiler-std/sentry"
	"github.com/tereus-project/tereus-transpiler-std/storage"
)

type TranspilerContext struct {
	SentryService  *sentry.SentryService
	MetricsService *metrics.MetricsService
	StorageService *storage.StorageService
	QueueService   *queue.QueueService
}

type TranspilerSubmissionHandler func(transpilerContext *TranspilerContext, msg *nsq.Message) error

func InitTranspiler(sourceLanguage string, targetLanguage string, onSubmission TranspilerSubmissionHandler) (*TranspilerContext, error) {
	err := env.LoadEnv()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load environment variables")
	}

	config := env.GetEnv()

	sentryService, err := sentry.NewSentryService()
	if err != nil {
		return nil, err
	}

	prometheusNamespace := fmt.Sprintf("transpiler_%s_%s", sourceLanguage, targetLanguage)
	metricsService, err := metrics.NewMetricsService(prometheusNamespace)
	if err != nil {
		return nil, err
	}
	go metricsService.Listen()

	storageService, err := storage.NewStorageService()
	if err != nil {
		return nil, err
	}

	queueService, err := queue.NewQueueService(config.NSQEndpoint, config.NSQLookupdEndpoint)
	if err != nil {
		return nil, err
	}

	transpilerContext := &TranspilerContext{
		SentryService:  sentryService,
		MetricsService: metricsService,
		StorageService: storageService,
		QueueService:   queueService,
	}

	queueTopic := fmt.Sprintf("transpilation_jobs_%s_to_%s", sourceLanguage, targetLanguage)
	queueService.AddHandler(queueTopic, "transpiler", func(msg *nsq.Message) error {
		return onSubmission(transpilerContext, msg)
	})

	return transpilerContext, nil
}

func (t *TranspilerContext) StopTranspiler() {
	t.SentryService.StopSentry()
	t.MetricsService.Close()
}
