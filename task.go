package main

import (
	"encoding/json"
	"errors"
	"os"
)

type task struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Action  string  `json:"action"`
	Status  string  `json:"status"`
	Message string  `json:"message"`
	Call    []*task `json:"call"`
}

func parseTaskJSON(path string) ([]*task, error) {
	bts, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var task []*task
	err = json.Unmarshal(bts, &task)

	return task, err
}

func findTaskByID(id int, tasks []*task) *task {
	for _, t := range tasks {
		if t.ID == id {
			return t
		}
	}

	return nil
}

func buildNodeByTask(task *task, n *node) {
	if task == nil || n == nil {
		return
	}

	n.id = task.ID
	n.name = task.Name
	n.action = task.Action
	n.status = task.Status
	n.message = task.Message
	for i := range task.Call {
		n.children = append(n.children, &node{id: task.Call[i].ID})
	}
}

type node struct {
	id       int
	name     string
	action   string
	status   string
	message  string
	children []*node
}

func newTree(tasks []*task) (*tree, error) {
	if len(tasks) == 0 {
		return nil, errors.New("input tasks data is empty")
	}

	var root = &node{}
	buildNodeByTask(tasks[0], root)

	var buildQueue []*node
	buildQueue = append(buildQueue, root.children...)
	for i := 0; i < len(buildQueue); i++ {
		if task := findTaskByID(buildQueue[i].id, tasks); task != nil {
			buildNodeByTask(task, buildQueue[i])
			buildQueue = append(buildQueue, buildQueue[i].children...)
		} else {
			return nil, errors.New("malformed tasks data")
		}
	}

	return &tree{root: root}, nil
}

type tree struct {
	root *node
}

func traverse(node *node, f func(node *node)) {
	if node == nil || f == nil {
		return
	}

	f(node)
	for _, next := range node.children {
		traverse(next, f)
	}
}
