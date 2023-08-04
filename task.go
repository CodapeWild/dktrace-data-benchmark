/*
 *   Copyright (c) 2023 CodapeWild
 *   All rights reserved.

 *   Licensed under the Apache License, Version 2.0 (the "License");
 *   you may not use this file except in compliance with the License.
 *   You may obtain a copy of the License at

 *   http://www.apache.org/licenses/LICENSE-2.0

 *   Unless required by applicable law or agreed to in writing, software
 *   distributed under the License is distributed on an "AS IS" BASIS,
 *   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *   See the License for the specific language governing permissions and
 *   limitations under the License.
 */

package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
)

type hop struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Action  string  `json:"action"`
	Status  string  `json:"status"`
	Message string  `json:"message"`
	Calls   []*call `json:"calls"`
}

func (op *hop) createNode(service string) *node {
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

type route []*hop

func (h route) findOptionkByID(id int) (*hop, bool) {
	for _, op := range h {
		if op.ID == id {
			return op, true
		}
	}

	return nil, false
}

func (h route) createTree(tracer Tracer) *tree {
	if len(h) == 0 || tracer == nil {
		log.Printf("create tree with empty task or nil tracer")

		return nil
	}

	root := h[0].createNode("")
	var buildQue = root.children
	for i := 0; i < len(buildQue); i++ {
		node := buildQue[i]
		h.setNode(node)
		buildQue = append(buildQue, node.children...)
	}

	return &tree{root: root, tracer: tracer}
}

func (h route) setNode(uncomplete *node) {
	op, ok := h.findOptionkByID(uncomplete.id)
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

func newRouteFromJSONFile(path string) (route, error) {
	bts, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var h route
	err = json.Unmarshal(bts, &h)

	return h, err
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
