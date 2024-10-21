package commands

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCommand() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "depudados",
		Short: "depudados ",
		Long:  "depudados scrapper for brazilian congress",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	err := viper.BindPFlags(cmd.Flags())
	if err != nil {
		return nil, err
	}

	cmd.AddCommand(reportCommand())
	cmd.AddCommand(camaraCommand())
	cmd.AddCommand(senadoCommand())

	return cmd, nil
}
