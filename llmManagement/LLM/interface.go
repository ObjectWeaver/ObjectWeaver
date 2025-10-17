package LLM

// IJobQueue defines the contract for different queue implementations
type IJobQueue interface {
	Enqueue(request *Job)
	Dequeue() *Job
	Size() int
}
