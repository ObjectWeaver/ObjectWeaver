package LLM

const (
	DefaultSubmitter = "default"
	VariedSubmitter  = "varied"
)

func JobSubmitterFactory(submitterType string) JobSumitter {
	switch submitterType {
	case DefaultSubmitter:
		return NewDefaultJobSubmitter()
	case VariedSubmitter:
		return NewVariedJobSubmitter()
	default:
		return NewDefaultJobSubmitter()
	}
}
