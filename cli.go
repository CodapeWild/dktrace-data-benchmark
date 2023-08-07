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
	"log"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "dktrace-data-benchmark",
	Aliases: []string{"dkb", "dkbench"},
	Short:   "benchmark widget written for Datakit trace test",
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "benchmark configuration file path in JSON format",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			defBenchConf = args[0]
		}
	},
}

// disableLogCmd represents the disableLog command
var disableLogCmd = &cobra.Command{
	Use:   "disable-log",
	Short: "disable log output",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			if ok, err := strconv.ParseBool(args[0]); err == nil {
				defDisableLog = ok
			}
		}
	},
}

// tasksCmd represents the tracer command
var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "tasks configuration command, JSON object string required, multiple arguments supported",
	Run: func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			task := &taskConfig{}
			if err := json.Unmarshal([]byte(arg), task); err != nil {
				log.Println(err.Error())
			} else {
				gTasks = append(gTasks, task)
			}
		}

		if len(gTasks) != 0 {
			mergeTasks(&gBenchConf.Tasks, gTasks)
		}
		if err := dumpBenchConfigFile(defBenchConf, gBenchConf); err != nil {
			log.Println(err.Error())
		}
	},
}

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "show all the saved tasks configuration if no task name offered, otherwise show as arguments provided",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			for _, task := range gBenchConf.Tasks {
				task.Print()
			}
		} else {
			for _, arg := range args {
				for _, task := range gBenchConf.Tasks {
					if task.Name == arg {
						task.Print()
					}
				}
			}
		}
	},
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use: "run",
	Short: `run task by name, task name required, multiple arguments supported but normally do not input more
	than 10 tasks at once which will take too long to complete`,
	Run: func(cmd *cobra.Command, args []string) {
		go runTaskThread()

		var c = 0
		for _, arg := range args {
			found := false
			for _, task := range gBenchConf.Tasks {
				if task.Name == arg {
					gTaskChan <- task
					c++
					found = true
				}
			}
			if !found {
				log.Printf("task: %s not found", arg)
			}
		}
		for range gFinish {
			if c--; c == 0 {
				log.Println("all tasks finished")

				return
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobratest.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	// add config command
	rootCmd.AddCommand(configCmd)
	// add enable log command
	rootCmd.AddCommand(disableLogCmd)
	// add tasks command
	rootCmd.AddCommand(tasksCmd)
	// add show command
	rootCmd.AddCommand(showCmd)
	// add run command
	rootCmd.AddCommand(runCmd)
}
