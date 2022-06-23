package env

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Env struct {
	SubmissionFolderPrefix string `env:"SUBMISSION_FOLDER_PREFIX" env-default:"transpilations"`

	S3Bucket       string `env:"S3_BUCKET" env-required:"true"`
	S3AccessKey    string `env:"S3_ACCESS_KEY" env-required:"true"`
	S3SecretKey    string `env:"S3_SECRET_KEY" env-required:"true"`
	S3Endpoint     string `env:"S3_ENDPOINT" env-required:"true"`
	S3HTTPSEnabled bool   `env:"S3_HTTPS_ENABLED" env-default:"false"`

	NSQEndpoint        string `env:"NSQ_ENDPOINT" env-required:"true"`
	NSQLookupdEndpoint string `env:"NSQ_LOOKUPD" env-required:"true"`

	LogFormat string `env:"LOG_FORMAT" env-default:"json"`
	LogLevel  string `env:"LOG_LEVEL" env-default:"info"`
	SentryDSN string `env:"SENTRY_DSN"`
	Env       string `env:"ENV" env-required:"true"`

	MetricsPort string `env:"METRICS_PORT" env-default:"8080"`
}

var env Env

func LoadEnv() error {
	loadErr := godotenv.Load()
	readErr := cleanenv.ReadEnv(&env)

	if loadErr != nil {
		return loadErr
	}

	return readErr
}

func GetEnv() *Env {
	return &env
}
