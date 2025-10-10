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

func TestJobSubmitterFactory_Varied(t *testing.T) {
	submitter := JobSubmitterFactory(VariedSubmitter)
	if _, ok := submitter.(*VariedJobSubmitter); !ok {
		t.Errorf("Expected *VariedJobSubmitter, got %T", submitter)
	}
}

func TestJobSubmitterFactory_Unknown(t *testing.T) {
	submitter := JobSubmitterFactory("unknown")
	if _, ok := submitter.(*DefaultJobSubmitter); !ok {
		t.Errorf("Expected *DefaultJobSubmitter for unknown type, got %T", submitter)
	}
}

func TestJobSubmitterFactory_EmptyString(t *testing.T) {
	submitter := JobSubmitterFactory("")
	if _, ok := submitter.(*DefaultJobSubmitter); !ok {
		t.Errorf("Expected *DefaultJobSubmitter for empty string, got %T", submitter)
	}
}
