package LLM

import (
	"testing"
)

func TestJobSubmitterFactory_Default(t *testing.T) {
	submitter := JobSubmitterFactory(DefaultSubmitter)
	if _, ok := submitter.(*DefaultJobSubmitter); !ok {
		t.Errorf("Expected *DefaultJobSubmitter, got %T", submitter)
	}
}

func TestJobSubmitterFactory_Unknown(t *testing.T) {
	submitter := JobSubmitterFactory("unknown")
	if _, ok := submitter.(*DirectJobSubmitter); !ok {
		t.Errorf("Expected *DirectJobSubmitter for unknown type, got %T", submitter)
	}
}

func TestJobSubmitterFactory_EmptyString(t *testing.T) {
	submitter := JobSubmitterFactory("")
	if _, ok := submitter.(*DirectJobSubmitter); !ok {
		t.Errorf("Expected *DirectJobSubmitter for empty string, got %T", submitter)
	}
}
