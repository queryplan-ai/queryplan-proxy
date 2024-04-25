package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/queryplan-ai/queryplan-proxy/pkg/daemon"
	daemontypes "github.com/queryplan-ai/queryplan-proxy/pkg/daemon/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func StartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "start",
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

			opts := daemontypes.DaemonOpts{
				DBMS:        daemontypes.Mysql,
				BindAddress: "0.0.0.0",
				BindPort:    3306,

				UpstreamAddress: "127.0.0.1",
				UpstreamPort:    3307,
			}
			go daemon.Run(ctx, opts)

			<-sigs
			cancel()

			fmt.Println("queryplan-proxy stopped gracefully.")
			return nil
		},
	}

	return cmd
}
