package commands

import (
	"depudados/metadata"
	"depudados/shared"
	"fmt"
	"github.com/spf13/cobra"
	"runtime"
)

func senadoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "senado",
		Short: "senado ",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(downloadSenadoPlp())

	return cmd
}
func downloadSenadoPlp() *cobra.Command {
	plpId := ""

	return dependenciesCommandWrapper(
		"download",
		func(cmd *cobra.Command, dep *Dependencies, args []string) error {
			ctx := cmd.Context()

			et, err := metadata.NewExtractorPool(runtime.NumCPU())
			if err != nil {
				return err
			}

			plp, err := dep.CameraLeg.GetPLP(ctx, plpId, shared.SENADO)
			if err != nil {
				return err
			}

			plp.Process(ctx, et, true)

			err = plp.AnyError()
			if err != nil {
				return err
			}

			csv := plp.ToCsv(map[string]int{}, true)

			fmt.Println(csv)

			return nil
		},
		func(cmd *cobra.Command) {
			cmd.Flags().StringVar(&plpId, "plp", "", "PLP ID")

			_ = cmd.MarkFlagRequired("plp")
		},
	)
}
