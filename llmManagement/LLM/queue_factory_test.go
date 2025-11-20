package LLM

import (
	"testing"
)

func TestNewJobQueueByType(t *testing.T) {
	tests := []struct {
		name      string
		queueType QueueType
		wantType  string
	}{
		{
			name:      "FIFO queue",
			queueType: QueueTypeFIFO,
			wantType:  "*LLM.FIFOQueueManager",
		},
		{
			name:      "Priority queue",
			queueType: QueueTypePriority,
			wantType:  "*LLM.RequestQueueManager",
		},
		{
			name:      "Default (unknown type)",
			queueType: QueueType("unknown"),
			wantType:  "*LLM.RequestQueueManager",
		},
		{
			name:      "Empty string defaults to priority",
			queueType: QueueType(""),
			wantType:  "*LLM.RequestQueueManager",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queue := NewJobQueueByType(tt.queueType)
			if queue == nil {
				t.Fatal("Expected non-nil queue")
			}

			// Verify it implements the interface
			var _ IJobQueue = queue
		})
	}
}

func TestQueueTypeConstants(t *testing.T) {
	if QueueTypeFIFO != "fifo" {
		t.Errorf("QueueTypeFIFO should be 'fifo', got %s", QueueTypeFIFO)
	}

	if QueueTypePriority != "priority" {
		t.Errorf("QueueTypePriority should be 'priority', got %s", QueueTypePriority)
	}
}

func TestNewJobQueueByType_InterfaceCompliance(t *testing.T) {
	queueTypes := []QueueType{QueueTypeFIFO, QueueTypePriority}

	for _, queueType := range queueTypes {
		t.Run(string(queueType), func(t *testing.T) {
			queue := NewJobQueueByType(queueType)

			// Test that basic interface methods are available
			size := queue.Size()
			if size != 0 {
				t.Errorf("Expected new queue to have size 0, got %d", size)
			}
		})
	}
}
