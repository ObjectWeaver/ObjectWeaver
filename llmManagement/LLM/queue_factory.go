package LLM

// QueueType represents the type of queue to create
type QueueType string

const (
	QueueTypeFIFO     QueueType = "fifo"
	QueueTypePriority QueueType = "priority"
)

// NewJobQueueByType creates a new JobQueueInterface based on the specified type
func NewJobQueueByType(queueType QueueType) IJobQueue {
	switch queueType {
	case QueueTypePriority:
		return NewRequestQueueManager()
	case QueueTypeFIFO:
		return NewFIFOQueueManager()
	default:
		return NewRequestQueueManager() // Default to Priority Queue (heap system)
	}
}
