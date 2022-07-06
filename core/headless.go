package core

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

func executeHeadless(filepath string, transpileFunction TranspileFunction) {
	output, err := transpileFunction(&TranspileFunctionConfig{
		LocalPathPrefix: "",
		LocalPath:       filepath,
	})
	if err != nil {
		logrus.WithError(err).Fatalf("Failed to transpile '%s'", filepath)
	}

	fmt.Println(output)
}
