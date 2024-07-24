package main

import (
	"context"
	"depudados/cameraleg"
	"depudados/metadata"
	"depudados/models"
	"depudados/repository"
	"flag"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"os"
	"runtime"
)

func main() {
	loadDeputados := flag.Bool("load-deputados", false, "deve carregar todos os deputados")
	csvFile := flag.String("generate-csv", "", "criar arquivo csv")

	plp := flag.String("plp", "", "download de plp")

	allAno := flag.String("allAno", "", "arquivo de download por ano")

	flag.Parse()

	if *plp != "" {
		err := runPlpMetadata(*plp)
		if err != nil {
			log.Fatal(err)
		}

		return
	}

	if *allAno != "" {
		err := runGetAllPorAno(*allAno)
		if err != nil {
			log.Fatal(err)
		}

		return
	}

	db, err := bolt.Open("my1.db", 0600, nil)

	if err != nil {
		log.Fatal(err)
	}

	err = createBuckets(db)
	if err != nil {
		log.Fatal(err)
	}

	persistence := repository.NewPersistence(db)

	if *csvFile != "" {
		proposicao, err := persistence.GetProposicao()
		if err != nil {
			log.Fatal(err)
		}
		file, err := os.Create(*csvFile)
		if err != nil {
			log.Fatal(err)
		}

		_, err = file.WriteString("DEPUTADO;URL;AUTOR\n")
		if err != nil {
			log.Fatal(err)
		}

		for _, m := range proposicao {
			row := fmt.Sprintf("%s;%s;%s\n", m.Deputado, m.Url, m.Autor)
			_, err := file.WriteString(row)
			if err != nil {
				log.Fatal(err)
			}
		}

		err = file.Close()
		if err != nil {
			log.Fatal(err)
		}

		return
	}

	err = loadProposicoes(persistence, *loadDeputados)
	if err != nil {
		log.Fatal(err)
	}

}

func loadProposicoes(persistence *repository.Persistence, loadDeputados bool) error {
	var deputados []*models.Deputado
	var err error

	if loadDeputados {
		deputados, err = repository.GetDeputados()
		err = persistence.LoadDeputados(deputados)

		if err != nil {
			log.Fatal(err)
		}
	} else {
		deputados, err = persistence.GetDeputados()
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("%d deputados\n", len(deputados))

	_, err = repository.GetProposicoes(persistence, deputados)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func createBuckets(db *bolt.DB) error {
	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Use the transaction...
	_, err = tx.CreateBucketIfNotExists([]byte("DEPUTADOS"))
	if err != nil {
		return err
	}

	_, err = tx.CreateBucketIfNotExists([]byte("PROPOSICAO"))
	if err != nil {
		return err
	}

	_, err = tx.CreateBucketIfNotExists([]byte("WORK"))
	if err != nil {
		return err
	}

	// Commit the transaction and check for error.
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func runGetAllPorAno(file string) error {
	ctx := context.Background()

	proposicao, err := cameraleg.GetAllProposicao(ctx, file, true)
	if err != nil {
		return err
	}

	csv := proposicao.ToCsv()

	fmt.Println(csv)

	return nil
}

func runPlpMetadata(plpId string) error {
	ctx := context.Background()

	et, err := metadata.NewExtractorPool(runtime.NumCPU())
	if err != nil {
		return err
	}

	plp, err := cameraleg.GetPLP(plpId)
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
