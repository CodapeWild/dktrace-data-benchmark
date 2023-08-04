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
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "dktrace-data-benchmark",
	Aliases: []string{"dkbench"},
	Short:   "benchmark widget written for Datakit trace test",
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "benchmark configuration file path in JSON format",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("config called")

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
		fmt.Println("disableLog called")

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
	Short: "tasks configuration command, the input arguments are taskConfig objects in JSON string format",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("tracer called")

		for _, arg := range args {
			task := &taskConfig{}
			if err := json.Unmarshal([]byte(arg), task); err != nil {
				log.Println(err.Error())
			} else {
				gtasks = append(gtasks, task)
			}
		}
	},
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run a task by name",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("run called")

	},
}

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "show all the saved tasks configuration",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("show called")

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
	// add run command
	rootCmd.AddCommand(runCmd)
	// add show command
	rootCmd.AddCommand(showCmd)
}
