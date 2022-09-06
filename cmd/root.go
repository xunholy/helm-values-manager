package cmd

import (
	"errors"
	"io"
	"os"

	"github.com/spf13/cobra"
)

const rootCmdUsage = `
The value-manager plugin helps you to manage detection of unused and/or outdated values being used in helm charts
`

var settings *EnvSettings

//NewRootCmd creates a root cmd
func NewRootCmd(out io.Writer, args []string) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "value-manager",
		Short:        "detect unused and/or outdated values being used",
		Long:         rootCmdUsage,
		SilenceUsage: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("no argument accepted")
			}
			return nil
		},
	}
	flags := cmd.PersistentFlags()
	flags.Parse(args)

	settings = new(EnvSettings)
	if ctx := os.Getenv("HELM_KUBECONTEXT"); ctx != "" {
		settings.KubeContext = ctx
	}

	cmd.AddCommand(NewFetchCmd(out))

	return cmd
}
