package sentry

import (
	"github.com/onrik/logrus/sentry"
	"github.com/tereus-project/tereus-go-std/logging"
	"github.com/tereus-project/tereus-transpiler-std/env"
)

type SentryService struct {
	sentryHook *sentry.Hook
}

func NewSentryService() (*SentryService, error) {
	config := env.GetEnv()

	sentryHook, err := logging.SetupLog(logging.LogConfig{
		Format:       config.LogFormat,
		LogLevel:     config.LogLevel,
		ShowFilename: true,
		ReportCaller: true,
		SentryDSN:    config.SentryDSN,
		Env:          config.Env,
	})
	if err != nil {
		return nil, err
	}

	return &SentryService{
		sentryHook: sentryHook,
	}, nil
}

func (s *SentryService) StopSentry() {
	s.sentryHook.Flush()
	logging.RecoverAndLogPanic()
}
