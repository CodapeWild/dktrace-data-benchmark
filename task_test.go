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
	"encoding/json"
	"fmt"
	"log"
	"testing"
)

func childrenPrinter(children []*node) string {
	if l := len(children); l == 0 {
		return ""
	} else {
		str := nodePrinter(children[0])
		for _, node := range children[1:] {
			str += "," + nodePrinter(node)
		}

		return str
	}
}

func nodePrinter(node *node) string {
	return fmt.Sprintf(`
{
  "id": %d,
  "name": %q,
  "action": %q,
  "status": %q,
  "message": %q,
  "children": [%s]
}`, node.id, node.name, node.action, node.status, node.message, childrenPrinter(node.children))
}

func TestBuildTree(t *testing.T) {
	tasks, err := newRouteFromJSONFile("./tasks/user-login.json")
	if err != nil {
		log.Fatalln(err.Error())
	}
	tree := tasks.createTree(&ddtracerwrapper{})
	jsonstr := nodePrinter(tree.root)
	if !json.Valid([]byte(jsonstr)) {
		log.Fatalln("invalid JSON string")
	}

	log.Println(nodePrinter(tree.root))
}
