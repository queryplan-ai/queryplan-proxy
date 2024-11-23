package main

import (
	"fmt"

	"github.com/queryplan-ai/queryplan-proxy/test/integration/pkg/upstream"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func MysqlCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "mysql",
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Starting mysql\n")
			upstreamProcess, err := upstream.StartMysql(false, false)
			if err != nil {
				return err
			}
			defer upstreamProcess.Stop()

			return nil
		},
	}

	return cmd
}
