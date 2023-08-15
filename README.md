# dktrace-data-benchmark

under progression still

## how to use this repo

run the command below in your shell to check the full help document.

```shell
./dktrace-data-benchmark -h
```

```doc
benchmark widget written for Datakit testing of trace modules

Usage:
  dktrace-data-benchmark [command]

Aliases:
  dktrace-data-benchmark, dkb, dkbench

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  config      benchmark configuration file path in JSON format
  disable-log disable log output
  help        Help about any command
  run         run task by name, task name required, multiple arguments supported but normally do not input more
	than 10 tasks at once which will take too long to complete
  show        show all the saved tasks configuration if no task name offered, otherwise show as arguments provided
  tasks       tasks configuration command, JSON object string required, multiple arguments supported

Flags:
  -h, --help     help for dktrace-data-benchmark
  -t, --toggle   Help message for toggle

Use "dktrace-data-benchmark [command] --help" for more information about a command.
```
