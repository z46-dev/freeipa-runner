package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type RunNowOptions struct {
	Groups    []string
	Hostnames []string
	TaskType  string
	File      string
}

var (
	rootCommand, installCommand, uninstallCommand, statusCommand *cobra.Command
	runNowCommand, scheduleCommand, listCommand, removeCommand   *cobra.Command
	runNowOpts                                                   RunNowOptions
)

func init() {
	rootCommand = &cobra.Command{
		Use:   "freeipa-runner",
		Short: "FreeIPA Runner is a service to manage FreeIPA hosts",
	}

	installCommand = &cobra.Command{
		Use:   "install",
		Short: "Install the FreeIPA Daemon service",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Installing FreeIPA Daemon...")
		},
	}

	uninstallCommand = &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the FreeIPA Daemon service",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Uninstalling FreeIPA Daemon...")
		},
	}

	statusCommand = &cobra.Command{
		Use:   "status",
		Short: "Check the status of the FreeIPA Daemon service",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("FreeIPA Daemon status: running")
		},
	}

	runNowCommand = &cobra.Command{
		Use:   "run-now",
		Short: "Run a task immediately",
		Long: `Run a playbook or script on specific hosts or groups immediately.
Example:
  freeipa-daemon run-now --group=desktops --group=infra --hostname=silver --type=ansible --file=/opt/freeipa-runner/data/test.yaml`,
		Run: func(cmd *cobra.Command, args []string) {
			runNow(runNowOpts)
		},
	}

	runNowCommand.Flags().StringSliceVarP(&runNowOpts.Groups, "group", "g", nil, "Target host group(s)")
	runNowCommand.Flags().StringSliceVarP(&runNowOpts.Hostnames, "hostname", "H", nil, "Target hostname(s)")
	runNowCommand.Flags().StringVarP(&runNowOpts.TaskType, "type", "t", "", "Task type (e.g., ansible, bash)")
	runNowCommand.Flags().StringVarP(&runNowOpts.File, "file", "f", "", "File path of the playbook or script")

	_ = runNowCommand.MarkFlagRequired("type")
	_ = runNowCommand.MarkFlagRequired("file")

	rootCommand.AddCommand(
		installCommand,
		uninstallCommand,
		statusCommand,
		runNowCommand,
	)
}

func runNow(opts RunNowOptions) {
	fmt.Println("Running task now with:")
	fmt.Printf("  Groups:    %s\n", strings.Join(opts.Groups, ", "))
	fmt.Printf("  Hostnames: %s\n", strings.Join(opts.Hostnames, ", "))
	fmt.Printf("  Type:      %s\n", opts.TaskType)
	fmt.Printf("  File:      %s\n", opts.File)
}

func main() {
	if err := rootCommand.Execute(); err != nil {
		panic(err)
	}
}
