package cmd

import (
	"github.com/labib0x9/ffgif/config"
	"github.com/labib0x9/ffgif/internal/infra/rabbitmq"
	"github.com/spf13/cobra"
)

var rmqCmd = &cobra.Command{
	Use:     "rabbitmq",
	Aliases: []string{"rmq"},
	Short:   "initializes queue",
	RunE:    rmqSetupfunc,
}

func rmqSetupfunc(cmd *cobra.Command, args []string) error {
	return setupMessageQueue()
}

func setupMessageQueue() error {
	cnf := config.GetConfig()
	conn := rabbitmq.NewRabbitMQ(cnf.RabbitMq)
	return rabbitmq.Setup(conn, cnf.Minio)
}

func init() {
	rootCmd.AddCommand(rmqCmd)
}
