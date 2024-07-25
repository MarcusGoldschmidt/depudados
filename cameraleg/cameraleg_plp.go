package cameraleg

import (
	"bytes"
	"context"
	"depudados/metadata"
	"errors"
	"github.com/barasher/go-exiftool"
	"github.com/gocolly/colly/v2"
	"github.com/sourcegraph/conc/pool"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type PLP struct {
	Id           string
	Files        []*PLPFileData `bson:"files"`
	ProcessadoEm *time.Time     `bson:"processadoEm"`
}

func (plp *PLP) Len() int {
	return len(plp.Files)
}

type FileMetadata struct {
	File   string                 `bson:"file"`
	Fields map[string]interface{} `bson:"fields"`
}

func EmptyFileMetadata() FileMetadata {
	return FileMetadata{
		File:   "",
		Fields: map[string]interface{}{},
	}
}

func NewFileMetadata(meta exiftool.FileMetadata) FileMetadata {
	return FileMetadata{
		Fields: meta.Fields,
		File:   meta.File,
	}
}

type PLPFileData struct {
	Id               string       `bson:"id"`
	Type             string       `bson:"type"`
	Ementa           string       `bson:"ementa"`
	DataApresentacao string       `bson:"dataApresentacao"`
	Autor            string       `bson:"autor"`
	LinkInteiroTeor  string       `bson:"linkInteiroTeor"`
	Metadados        FileMetadata `bson:"metadados"`
	Err              string       `bson:"error"`
}

func (plp *PLPFileData) SetLink(link string) {
	plp.LinkInteiroTeor = "https://www.camara.leg.br/proposicoesWeb/" + link
}

func (plp *PLPFileData) getOutputPath() string {
	return filepath.Join(
		".",
		"out",
		plp.Id,
		strings.ReplaceAll(plp.Ementa, "/", "-")+"-"+strings.ReplaceAll(plp.DataApresentacao, "/", "-"),
	)
}

func ExtractPLP(proposicaoId string) (*PLP, error) {
	plp := &PLP{
		Id: proposicaoId,
	}

	var err error

	err = plp.extractPLPFileData(
		proposicaoId,
		"Destaques",
		"https://www.camara.leg.br/proposicoesWeb/prop_destaques?idProposicao="+proposicaoId+"&subst=0",
		func(data *PLPFileData, element *colly.HTMLElement) {
			data.Ementa = element.ChildText("td:nth-child(1)")
			data.DataApresentacao = element.ChildText("td:nth-child(2)")
			data.Autor = element.ChildText("td:nth-child(3)")
			data.SetLink(element.ChildAttr("td:nth-child(4) a", "href"))
		},
	)
	if err != nil {
		return nil, err
	}

	err = plp.extractPLPFileData(
		proposicaoId,
		"EmentaProjeto",
		"https://www.camara.leg.br/proposicoesWeb/prop_emendas?idProposicao="+proposicaoId+"&subst=0",
		func(data *PLPFileData, element *colly.HTMLElement) {
			data.Ementa = element.ChildText("td:nth-child(1)")
			data.DataApresentacao = element.ChildText("td:nth-child(3)")
			data.Autor = element.ChildText("td:nth-child(4)")
			data.SetLink(element.ChildAttr("td:nth-child(5) a", "href"))
		},
	)
	if err != nil {
		return nil, err
	}

	err = plp.extractPLPFileData(
		proposicaoId,
		"HistoricoDePareceres",
		"https://www.camara.leg.br/proposicoesWeb/prop_pareceres_substitutivos_votos?idProposicao="+proposicaoId+"&subst=0",
		func(data *PLPFileData, element *colly.HTMLElement) {
			data.Ementa = element.ChildText("td:nth-child(1)")
			data.DataApresentacao = element.ChildText("td:nth-child(3)")
			data.Autor = element.ChildText("td:nth-child(4)")
			data.SetLink(element.ChildAttr("td:nth-child(5) a", "href"))
		},
	)
	if err != nil {
		return nil, err
	}

	return plp, nil
}

func (plp *PLP) extractPLPFileData(
	proposicaoId string,
	fileType string,
	url string,
	extractDataFunc func(*PLPFileData, *colly.HTMLElement),
) error {
	plps := make([]*PLPFileData, 0)

	c := colly.NewCollector()
	c.SetRequestTimeout(time.Second * 20)

	c.OnHTML(".coresAlternadas", func(e *colly.HTMLElement) {
		e.ForEach("tr", func(i int, e *colly.HTMLElement) {

			plp := &PLPFileData{
				Id:        proposicaoId,
				Type:      fileType,
				Metadados: EmptyFileMetadata(),
			}

			extractDataFunc(plp, e)

			plps = append(plps, plp)
		})
	})

	err := c.Visit(url)
	if err != nil {
		return err
	}

	plp.Files = append(plp.Files, plps...)
	return nil
}

func (plp *PLP) getOutputPath() string {
	return filepath.Join(".", "out", plp.Id)
}

func (plp *PLP) Process(ctx context.Context, et *metadata.ExtractorPool, extractMetadata bool) {
	if plp.Processed() {
		log.Printf("Skipping PLP %s already processed\n", plp.Id)
		return
	}

	if plp.Len() == 0 {
		log.Printf("Skipping PLP %s has no files to process\n", plp.Id)
		now := time.Now().UTC()
		plp.ProcessadoEm = &now
		return
	}

	log.Printf("Processing PLP %s len: %d\n", plp.Id, plp.Len())

	_ = os.MkdirAll(plp.getOutputPath(), os.ModePerm)

	workerCount := runtime.NumCPU() * 2
	if plp.Len() < workerCount {
		workerCount = plp.Len()
	}

	plpFileWorkerPool := pool.New().WithMaxGoroutines(workerCount)

	for _, plp := range plp.Files {
		plp := plp
		plpFileWorkerPool.Go(func() {
			plpWorker(ctx, et, plp, extractMetadata)
		})
	}

	plpFileWorkerPool.Wait()
	now := time.Now().UTC()
	plp.ProcessadoEm = &now
}

func (plp *PLP) Processed() bool {
	return plp.ProcessadoEm != nil
}

func (plp *PLP) AnyError() error {
	for _, plpFileData := range plp.Files {
		if plpFileData.Err != "" {
			return errors.New(plpFileData.Err)
		}
	}

	return nil
}

func (plp *PLPFileData) Download(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, plp.LinkInteiroTeor, nil)

	if err != nil {
		return nil, err
	}
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func plpWorker(ctx context.Context, et *metadata.ExtractorPool, plp *PLPFileData, extractMetadata bool) {
	fileName := plp.getOutputPath()

	file, err := os.ReadFile(fileName)
	if err != nil {
		file, err = plp.Download(ctx)
		if err != nil {
			plp.Err = err.Error()
			log.Printf("Error downloading file: %v\n", err)
			return
		}
	}

	create, err := os.Create(fileName)
	if err != nil {
		plp.Err = err.Error()
		return
	}

	_, err = io.Copy(create, bytes.NewBuffer(file))
	if err != nil {
		plp.Err = err.Error()
		return
	}

	if extractMetadata == true {
		fileInfos := et.ExtractMetadata(fileName)

		if len(fileInfos) == 0 {
			plp.Err = "no metadata found"
			return
		}

		fileInfo := fileInfos[0]

		if fileInfo.Err != nil {
			plp.Err = fileInfo.Err.Error()
			return
		}

		plp.Metadados = NewFileMetadata(fileInfo)
	}
}
