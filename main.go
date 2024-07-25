package main

import (
	"context"
	"depudados/metadata"
	"depudados/repository"
	"depudados/services"
	"flag"
	"fmt"
	"log"
	"runtime"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	dbConnection := flag.String("db", "mongodb://user:pass@localhost:27017", "connection string do banco de dados")

	plp := flag.String("plp", "", "download de plp")

	report := flag.Bool("report", false, "generate csv report")

	var files arrayFlags

	flag.Var(&files, "allAno", "arquivo de download por ano")

	flag.Parse()

	persistence, err := repository.NewPersistence(*dbConnection, "depudados")
	if err != nil {
		log.Fatal(err)
	}

	cameraLeg := services.NewCameraLeg(persistence)

	if *report {
		ctx := context.Background()

		toReport, err := cameraLeg.GetAllToReport(ctx)
		if err != nil {
			return
		}

		csv := toReport.ToCsv()

		fmt.Println(csv)
		return
	}

	if *plp != "" {
		err := runPlpMetadata(cameraLeg, *plp)
		if err != nil {
			log.Fatal(err)
		}

		return
	}

	if len(files) != 0 {
		err := runGetAllPorAno(cameraLeg, files)
		if err != nil {
			log.Fatal(err)
		}

		return
	}
}

func runGetAllPorAno(camera *services.CameraLeg, files []string) error {
	ctx := context.Background()

	proposicao, err := camera.ProcessMany(ctx, files, true)
	if err != nil {
		return err
	}

	csv := proposicao.ToCsv()

	fmt.Println(csv)

	return nil
}

func runPlpMetadata(camera *services.CameraLeg, plpId string) error {
	ctx := context.Background()

	et, err := metadata.NewExtractorPool(runtime.NumCPU())
	if err != nil {
		return err
	}

	plp, err := camera.GetPLP(ctx, plpId)
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
}
