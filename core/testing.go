package core

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

type AssertTranspilationConfig struct {
	SourceFilename    string
	SourceCode        string
	TargetCode        string
	TranspileFunction TranspileFunction
}

func AssertTranspilation(t *testing.T, config *AssertTranspilationConfig) {
	sourceCode := strings.TrimSpace(config.SourceCode)
	targetCode := strings.TrimSpace(config.TargetCode)

	dir := t.TempDir()
	sourceFile := fmt.Sprintf("%s/%s", dir, config.SourceFilename)
	err := os.WriteFile(sourceFile, []byte(sourceCode), 0644)
	if err != nil {
		t.Error(err)
	}

	output, err := config.TranspileFunction(sourceFile)
	if err != nil {
		t.Error(err)
	}

	output = strings.TrimSpace(output)
	if output != targetCode {
		t.Errorf("Expected %s, got %s", targetCode, output)
	}
}
