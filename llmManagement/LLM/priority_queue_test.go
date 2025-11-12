package LLM

import (
	"objectweaver/llmManagement"
	"testing"
)

func TestPriorityQueue_Len(t *testing.T) {
	tests := []struct {
		name     string
		queue    PriorityQueue
		expected int
	}{
		{
			name:     "empty queue",
			queue:    PriorityQueue{},
			expected: 0,
		},
		{
			name: "single item",
			queue: PriorityQueue{
				&Job{Inputs: &llmManagement.Inputs{Priority: 1}},
			},
			expected: 1,
		},
		{
			name: "multiple items",
			queue: PriorityQueue{
				&Job{Inputs: &llmManagement.Inputs{Priority: 1}},
				&Job{Inputs: &llmManagement.Inputs{Priority: 2}},
				&Job{Inputs: &llmManagement.Inputs{Priority: 3}},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.queue.Len(); got != tt.expected {
				t.Errorf("Len() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPriorityQueue_Less(t *testing.T) {
	pq := PriorityQueue{
		&Job{Inputs: &llmManagement.Inputs{Priority: 1}},
		&Job{Inputs: &llmManagement.Inputs{Priority: 5}},
		&Job{Inputs: &llmManagement.Inputs{Priority: 3}},
	}

	tests := []struct {
		name     string
		i, j     int
		expected bool
	}{
		{
			name:     "i has lower priority than j",
			i:        0,
			j:        1,
			expected: false,
		},
		{
			name:     "i has higher priority than j",
			i:        1,
			j:        0,
			expected: true,
		},
		{
			name:     "equal priority",
			i:        0,
			j:        0,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pq.Less(tt.i, tt.j); got != tt.expected {
				t.Errorf("Less(%d, %d) = %v, want %v (priorities: %d, %d)",
					tt.i, tt.j, got, tt.expected, pq[tt.i].Inputs.Priority, pq[tt.j].Inputs.Priority)
			}
		})
	}
}

func TestPriorityQueue_Swap(t *testing.T) {
	pq := PriorityQueue{
		&Job{Inputs: &llmManagement.Inputs{Priority: 1, Index: 0}},
		&Job{Inputs: &llmManagement.Inputs{Priority: 5, Index: 1}},
	}

	originalFirst := pq[0]
	originalSecond := pq[1]

	pq.Swap(0, 1)

	if pq[0] != originalSecond {
		t.Error("Expected first element to be swapped with second")
	}
	if pq[1] != originalFirst {
		t.Error("Expected second element to be swapped with first")
	}
	if pq[0].Inputs.Index != 0 {
		t.Errorf("Expected index of first element to be 0, got %d", pq[0].Inputs.Index)
	}
	if pq[1].Inputs.Index != 1 {
		t.Errorf("Expected index of second element to be 1, got %d", pq[1].Inputs.Index)
	}
}

func TestPriorityQueue_Push(t *testing.T) {
	pq := &PriorityQueue{}

	job1 := &Job{Inputs: &llmManagement.Inputs{Priority: 1}}
	job2 := &Job{Inputs: &llmManagement.Inputs{Priority: 2}}

	pq.Push(job1)
	if pq.Len() != 1 {
		t.Errorf("Expected length 1 after first push, got %d", pq.Len())
	}
	if job1.Inputs.Index != 0 {
		t.Errorf("Expected index 0 for first job, got %d", job1.Inputs.Index)
	}

	pq.Push(job2)
	if pq.Len() != 2 {
		t.Errorf("Expected length 2 after second push, got %d", pq.Len())
	}
	if job2.Inputs.Index != 1 {
		t.Errorf("Expected index 1 for second job, got %d", job2.Inputs.Index)
	}
}

func TestPriorityQueue_Pop(t *testing.T) {
	pq := &PriorityQueue{
		&Job{Inputs: &llmManagement.Inputs{Priority: 1, Index: 0}},
		&Job{Inputs: &llmManagement.Inputs{Priority: 2, Index: 1}},
		&Job{Inputs: &llmManagement.Inputs{Priority: 3, Index: 2}},
	}

	originalLen := pq.Len()

	popped := pq.Pop()
	if popped == nil {
		t.Fatal("Expected non-nil popped item")
	}

	job := popped.(*Job)
	if job.Inputs.Priority != 3 {
		t.Errorf("Expected to pop last item with priority 3, got %d", job.Inputs.Priority)
	}
	if job.Inputs.Index != -1 {
		t.Errorf("Expected popped item index to be -1, got %d", job.Inputs.Index)
	}
	if pq.Len() != originalLen-1 {
		t.Errorf("Expected length to decrease by 1, got %d", pq.Len())
	}
}

func TestPriorityQueue_PopEmptyHandling(t *testing.T) {
	pq := &PriorityQueue{
		&Job{Inputs: &llmManagement.Inputs{Priority: 1, Index: 0}},
	}

	// Pop the only item
	popped := pq.Pop()
	if popped == nil {
		t.Fatal("Expected non-nil popped item")
	}

	if pq.Len() != 0 {
		t.Errorf("Expected empty queue after popping only item, got length %d", pq.Len())
	}
}

func TestPriorityQueue_MultipleOperations(t *testing.T) {
	pq := &PriorityQueue{}

	// Push multiple items
	jobs := []*Job{
		{Inputs: &llmManagement.Inputs{Priority: 3}},
		{Inputs: &llmManagement.Inputs{Priority: 1}},
		{Inputs: &llmManagement.Inputs{Priority: 5}},
		{Inputs: &llmManagement.Inputs{Priority: 2}},
	}

	for _, job := range jobs {
		pq.Push(job)
	}

	if pq.Len() != 4 {
		t.Errorf("Expected length 4 after pushes, got %d", pq.Len())
	}

	// Verify indices are set correctly
	for i := 0; i < pq.Len(); i++ {
		if (*pq)[i].Inputs.Index != i {
			t.Errorf("Expected index %d for item at position %d, got %d", i, i, (*pq)[i].Inputs.Index)
		}
	}

	// Test swap
	pq.Swap(0, 1)
	if (*pq)[0].Inputs.Index != 0 || (*pq)[1].Inputs.Index != 1 {
		t.Error("Swap didn't update indices correctly")
	}

	// Pop an item
	popped := pq.Pop()
	if popped == nil {
		t.Fatal("Expected non-nil popped item")
	}
	if pq.Len() != 3 {
		t.Errorf("Expected length 3 after pop, got %d", pq.Len())
	}
}
