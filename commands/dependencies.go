package commands

import (
	"depudados/repository"
	"depudados/services"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	MongoUrl string

	Persistence *repository.Persistence
	CameraLeg   *services.CameraLeg
}

func (dep *Dependencies) Init() error {
	persistence, err := repository.NewPersistence(dep.MongoUrl, "depudados")
	if err != nil {
		return err
	}

	dep.Persistence = persistence
	dep.CameraLeg = services.NewCameraLeg(persistence)

	return nil
}

func (dep *Dependencies) ConfigureFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&dep.MongoUrl, "mongo-url", "mongodb://user:pass@localhost:27017", "MongoDB URL")
}

func dependenciesCommandWrapper(
	use string,
	f func(cmd *cobra.Command, deps *Dependencies, args []string) error,
	options ...func(cmd *cobra.Command),
) *cobra.Command {
	dep := &Dependencies{}

	cmd := &cobra.Command{
		Use: use,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := dep.Init()
			if err != nil {
				return err
			}

			return f(cmd, dep, args)
		},
	}

	dep.ConfigureFlags(cmd)
	if len(options) > 0 {
		for _, f := range options {
			f(cmd)
		}
	}

	return cmd
}
