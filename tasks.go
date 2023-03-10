package workertaskqueue

import (
	"sort"
	"time"
)

type Task struct {
	Id       string
	Priority int
	Result   string
	Created  time.Time

	CallbackId         string
	CallbackSignalName string
}

type tasks []*Task

// tasks can be sorted by ts and prio
func (t tasks) Len() int {
	return len(t)
}

// Less should true if prio is higher or ts is later
func (t tasks) Less(i, j int) bool {
	if t[i].Priority < t[j].Priority {
		return true
	}
	if t[i].Priority > t[j].Priority {
		return false
	}
	return t[i].Created.Before(t[j].Created)
}

func (t tasks) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]

}

// Pop removes the first element from tasks from the queue and returns it
func (t tasks) Pop() (*Task, tasks) {
	sort.Sort(t) // sort by Timestamp and prio
	return t[0], t[1:]
}
