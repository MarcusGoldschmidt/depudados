package commands

import (
	"context"
	"depudados/metadata"
	"depudados/shared"
	"fmt"
	"github.com/spf13/cobra"
	"runtime"
)

func camaraCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "camara",
		Short: "camara ",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(downloadCamaraPlp())
	cmd.AddCommand(downloadManyCamaraPlp())

	return cmd
}
func downloadCamaraPlp() *cobra.Command {
	plpId := ""

	return dependenciesCommandWrapper(
		"download",
		func(cmd *cobra.Command, dep *Dependencies, args []string) error {
			ctx := cmd.Context()

			et, err := metadata.NewExtractorPool(runtime.NumCPU())
			if err != nil {
				return err
			}

			plp, err := dep.CameraLeg.GetPLP(ctx, plpId, shared.CAMARA)
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
		},
	)
}

func downloadManyCamaraPlp() *cobra.Command {
	files := []string{}

	return dependenciesCommandWrapper(
		"process-many",
		func(cmd *cobra.Command, dep *Dependencies, args []string) error {
			ctx := context.Background()

			proposicao, err := dep.CameraLeg.ProcessManyCamara(ctx, files, true)
			if err != nil {
				return err
			}

			csv := proposicao.ToCsv()

			fmt.Println(csv)

			return nil
		},
		func(cmd *cobra.Command) {
			cmd.Flags().StringArrayVar(&files, "files", []string{}, "Files")
		},
	)
}
