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
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "dktrace-data-benchmark",
	Aliases: []string{"dktrace"},
	Short:   "benchmark widget written for Datakit trace test",
}

// enableLogCmd represents the enableLog command
var enableLogCmd = &cobra.Command{
	Use:   "enable-log",
	Short: "enable log or not",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("enableLog called")
	},
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "tracers config JSON file path",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("config called")
	},
}

// tracerCmd represents the tracer command
var tracerCmd = &cobra.Command{
	Use:   "tracer",
	Short: "single trace configuration command",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("tracer called")
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

	// add enable log command
	rootCmd.AddCommand(enableLogCmd)

	// add config command
	rootCmd.AddCommand(configCmd)

	// add tracer command
	rootCmd.AddCommand(tracerCmd)
	tracerCmd.PersistentFlags().String("name", "", "tracer name")
	tracerCmd.PersistentFlags().String("task", "", "task config JSON file path")
	tracerCmd.PersistentFlags().Int("threads", 10, "value used by amplifier to start `threads` number of threads")
	tracerCmd.PersistentFlags().Int("repeat", 100, "value used by amplifier to repeatedly send `repeat` times per thread")
	tracerCmd.PersistentFlags().String("collector_proto", "http", "collector schema or proto")
	tracerCmd.PersistentFlags().String("collector_ip", "127.0.0.1", "collector IP")
	tracerCmd.PersistentFlags().Int("collector_port", 9529, "collector port")
	tracerCmd.PersistentFlags().String("collector_path", "", "collector path")
}
