package cmd

import (
	"github.com/labib0x9/ffgif/config"
	"github.com/labib0x9/ffgif/internal/infra/minio"
	"github.com/spf13/cobra"
)

var minioCmd = &cobra.Command{
	Use:     "minio",
	Aliases: []string{"ms3"},
	Short:   "initializes queue",
	RunE:    minioSetupfunc,
}

func minioSetupfunc(cmd *cobra.Command, args []string) error {
	return setupStorage()
}

func setupStorage() error {
	cnf := config.GetConfig()
	client := minio.NewMinio(cnf.Minio)

	return minio.Setup(client, cnf.Minio)
}

func init() {
	rootCmd.AddCommand(minioCmd)
}
