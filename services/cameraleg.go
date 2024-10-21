package services

import (
	"context"
	"depudados/cameraleg"
	"depudados/metadata"
	"depudados/repository"
	"depudados/shared"
	"github.com/sourcegraph/conc/pool"
	"log"
	"runtime"
	"strconv"
	"sync"
)

type CameraLeg struct {
	persistence *repository.Persistence
}

func NewCameraLeg(persistence *repository.Persistence) *CameraLeg {
	return &CameraLeg{persistence: persistence}
}

func (c *CameraLeg) ExtractAndPersist(ctx context.Context, id string, plpType shared.PlpType) (*cameraleg.PLP, error) {
	plp, err := cameraleg.ExtractPLP(id, plpType)
	if err != nil {
		return nil, err
	}

	err = c.persistence.InsertPlp(ctx, plp)
	if err != nil {
		return nil, err
	}

	return plp, err
}

func (c *CameraLeg) GetPLP(ctx context.Context, id string, plpType shared.PlpType) (*cameraleg.PLP, error) {
	plp, err := c.persistence.GetPlp(ctx, id)
	if err != nil {
		return nil, err
	}

	if plp == nil {
		return c.ExtractAndPersist(ctx, id, plpType)
	}

	return plp, nil
}

func (c *CameraLeg) GetAllToReport(ctx context.Context) (cameraleg.PLPList, error) {
	return c.persistence.GetAllToReport(ctx)
}

func (c *CameraLeg) ProcessManyCamara(ctx context.Context, files []string, extractMetadata bool) (cameraleg.PLPList, error) {
	workerCount := runtime.NumCPU() * 2

	et, err := metadata.NewExtractorPool(workerCount)
	if err != nil {
		return nil, err
	}

	proposicoes, err := cameraleg.LoadProposicao(files)
	if err != nil {
		return nil, err
	}

	log.Printf("total: %d", len(proposicoes))

	// Start workers
	plpWorkerPool := pool.New().WithMaxGoroutines(workerCount)

	resultLock := sync.Mutex{}
	result := make([]*cameraleg.PLP, 0)

	for _, prop := range proposicoes {
		prop := prop

		plpWorkerPool.Go(func() {
			plp, err := c.GetPLP(ctx, strconv.Itoa(prop.Id), shared.CAMARA)
			if err != nil {
				log.Printf("[err] for %d err: %s", prop.Id, err.Error())
				return
			}

			resultLock.Lock()
			result = append(result, plp)
			resultLock.Unlock()

			plp.Process(ctx, et, extractMetadata)

			err = c.persistence.UpdatePlp(ctx, plp)
			if err != nil {
				log.Printf("err for %s err: %s", plp.Id, err.Error())
			}

			log.Printf("%d of %d\n", len(result), len(proposicoes))

			err = plp.AnyError()
			if err != nil {
				log.Printf("err for %s err: %s", plp.Id, err.Error())
			}
		})
	}

	plpWorkerPool.Wait()

	return result, ctx.Err()
}
