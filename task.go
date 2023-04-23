package main

type Task struct {
	id          int64
	duration    int64
	concurrency int
	children    []*Task
}

type TaskTree struct {
	root *Task
}

func (tt *TaskTree) Find(id int64) *Task {

}
