package commands

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
)

func reportCommand() *cobra.Command {
	return dependenciesCommandWrapper(
		"report",
		func(cmd *cobra.Command, dep *Dependencies, args []string) error {
			ctx := context.Background()

			toReport, err := dep.CameraLeg.GetAllToReport(ctx)
			if err != nil {
				return err
			}

			csv := toReport.ToCsv()

			fmt.Println(csv)

			return nil
		},
	)
}
