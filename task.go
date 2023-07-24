package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
)

func newTaskFromJSONFile(path string) (task, error) {
	bts, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tk task
	err = json.Unmarshal(bts, &tk)

	return tk, err
}

type task []*option

func (tk task) findOptionkByID(id int) (*option, bool) {
	for _, op := range tk {
		if op.ID == id {
			return op, true
		}
	}

	return nil, false
}

func (tk task) createTree(tracer Tracer) *tree {
	if len(tk) == 0 || tracer == nil {
		log.Printf("create tree with empty task or nil tracer")

		return nil
	}

	root := tk[0].createNode("")
	var buildQue = root.children
	for i := 0; i < len(buildQue); i++ {
		node := buildQue[i]
		tk.setNode(node)
		buildQue = append(buildQue, node.children...)
	}

	return &tree{root: root, tracer: tracer}
}

func (tk task) setNode(uncomplete *node) {
	op, ok := tk.findOptionkByID(uncomplete.id)
	if !ok {
		return
	}

	if uncomplete.service == "" {
		uncomplete.service = op.Name
	}
	uncomplete.name = op.Name
	uncomplete.action = op.Action
	uncomplete.status = op.Status
	uncomplete.message = op.Message
	for _, c := range op.Calls {
		if c.Outgoing {
			uncomplete.children = append(uncomplete.children, &node{id: c.ID})
		} else {
			uncomplete.children = append(uncomplete.children, &node{id: c.ID, service: op.Name})
		}
	}
}

type option struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Action  string  `json:"action"`
	Status  string  `json:"status"`
	Message string  `json:"message"`
	Calls   []*call `json:"calls"`
}

func (op *option) createNode(service string) *node {
	if service == "" {
		service = op.Name
	}

	n := &node{
		id:      op.ID,
		service: service,
		name:    op.Name,
		action:  op.Action,
		status:  op.Status,
		message: op.Message,
	}
	for _, c := range op.Calls {
		if c.Outgoing {
			n.children = append(n.children, &node{id: c.ID})
		} else {
			n.children = append(n.children, &node{id: c.ID, service: service})
		}
	}

	return n
}

type call struct {
	ID       int  `json:"id"`
	Outgoing bool `json:"outgoing"`
	service  string
}

type node struct {
	id       int
	service  string
	name     string
	action   string
	status   string
	message  string
	children []*node
}

type tree struct {
	root   *node
	tracer Tracer
}

func (tr *tree) count() int {
	var (
		nodes = []*node{tr.root}
		c     = 0
	)
	for i := 0; i < len(nodes); i++ {
		c++
		nodes = append(nodes, nodes[i].children...)
	}

	return c
}

func (tr *tree) spawn(ctx context.Context, agentAddress string) {
	if tr.tracer == nil || tr.root == nil {
		log.Printf("got nil tracer: %v or nil span tree: %v", tr.tracer, tr.root)

		return
	}

	tr.tracer.Start(agentAddress, tr.root.service)
	defer tr.tracer.Stop()

	tr.root.spawn(ctx, tr.tracer)
}

// func traverse(root *node, p func(n *node) bool) {
// 	if root == nil || p == nil {
// 		return
// 	}

// 	if p(root) {
// 		return
// 	}
// 	for _, c := range root.children {
// 		traverse(c, p)
// 	}
// }
