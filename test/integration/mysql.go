package main

import (
	"github.com/queryplan-ai/queryplan-proxy/test/integration/pkg/mysql"
	"github.com/queryplan-ai/queryplan-proxy/test/integration/pkg/upstream"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func MysqlCmd() *cobra.Command {
	var upstreamProcesses []*upstream.UpsreamProcess
	cmd := &cobra.Command{
		Use:          "mysql",
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ups, err := mysql.Execute()
			if err != nil {
				return err
			}

			upstreamProcesses = ups
			return nil
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			for _, up := range upstreamProcesses {
				up.Stop()
			}

			return nil
		},
	}

	return cmd
}
