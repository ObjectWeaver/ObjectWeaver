package LLM

type PriorityQueue []*Job

func (pq PriorityQueue) Len() int {
	return len(pq)
}

func (pq PriorityQueue) Less(i, j int) bool {
	// Higher priority comes first
	return pq[i].Inputs.Priority > pq[j].Inputs.Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Inputs.Index = i
	pq[j].Inputs.Index = j
}

func (pq *PriorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*Job)
	item.Inputs.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil         // avoid memory leak
	item.Inputs.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}
