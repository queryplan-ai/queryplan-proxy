package main

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		panic(err)
	}
}

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "integration-test",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("no command specified")
		},
	}

	cobra.OnInitialize(initConfig)

	cmd.AddCommand(PostgresCmd())
	cmd.AddCommand(MysqlCmd())

	cmd.PersistentFlags().String("log-level", "info", "log level")

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	return cmd
}

func initConfig() {
	viper.SetEnvPrefix("QUERYPLAN")
	viper.AutomaticEnv()
}
