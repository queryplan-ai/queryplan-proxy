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
			v := viper.GetViper()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

			opts := daemontypes.DaemonOpts{
				APIURL:      v.GetString("api-url"),
				Token:       v.GetString("token"),
				Environment: v.GetString("env"),

				LiveConnectionURI: v.GetString("live-connection-uri"),
				DatabaseName:      v.GetString("database-name"),

				DBMS:        daemontypes.DBMS(v.GetString("dbms")),
				BindAddress: v.GetString("bind-address"),
				BindPort:    v.GetInt("bind-port"),

				UpstreamAddress: v.GetString("upstream-address"),
				UpstreamPort:    v.GetInt("upstream-port"),
			}
			go daemon.Run(ctx, opts)

			<-sigs
			cancel()

			fmt.Println("queryplan-proxy stopped gracefully.")
			return nil
		},
	}

	cmd.Flags().String("token", "", "API token for QueryPlan")
	cmd.Flags().String("env", "", "Environment name for QueryPlan")

	cmd.Flags().String("api-url", "https://api.queryplan.ai", "URL for QueryPlan API")
	cmd.Flags().MarkHidden("api-url")

	cmd.Flags().String("live-connection-uri", "", "Live connection URI for the database")
	cmd.Flags().String("database-name", "", "Name of the database")

	cmd.Flags().String("dbms", "", "DBMS type")
	cmd.Flags().String("bind-address", "0.0.0.0", "Address to bind the proxy to")
	cmd.Flags().Int("bind-port", 0, "Port to bind the proxy to")

	cmd.Flags().String("upstream-address", "", "Address of the upstream database")
	cmd.Flags().Int("upstream-port", 0, "Port of the upstream database")

	return cmd
}
