package cli

import (
	"fmt"

	"github.com/queryplan-ai/queryplan-proxy/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func VersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the current version and exit",
		Long:  `Print the current version and exit`,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("queryplan-proxy %s\n", version.Version())

			return nil
		},
	}

	return cmd
}
