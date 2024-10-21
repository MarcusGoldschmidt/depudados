package cameraleg

import (
	"bytes"
	"context"
	"depudados/metadata"
	"depudados/shared"
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
	Id           string         `bson:"id"`
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
	Id string `bson:"id"`

	Type             string            `bson:"type"`
	Data             map[string]string `bson:"data"`
	Ementa           string            `bson:"ementa"`
	DataApresentacao string            `bson:"dataApresentacao"`
	Autor            string            `bson:"autor"`

	// File
	Link      string       `bson:"link"`
	Metadados FileMetadata `bson:"metadados"`
	Err       string       `bson:"error"`
}

func (plp *PLPFileData) SetData(key, value string) {
	plp.Data[key] = value
}

func (plp *PLPFileData) SetLinkCamara(link string) {
	plp.Link = "https://www.camara.leg.br/proposicoesWeb/" + link
}

func (plp *PLPFileData) getOutputPath() string {
	return filepath.Join(
		".",
		"out",
		plp.Id,
		strings.ReplaceAll(plp.Ementa, "/", "-")+"-"+strings.ReplaceAll(plp.DataApresentacao, "/", "-"),
	)
}

func ExtractPLP(proposicaoId string, plpType shared.PlpType) (*PLP, error) {
	if plpType == shared.SENADO {
		return ExtractSenadoMateria(proposicaoId)
	}

	if plpType == shared.CAMARA {
		return ExtractCamaraPLP(proposicaoId)
	}

	return nil, errors.New("invalid plp type")
}

// ExtractSenadoMateria extracts the PLP from the given id 164914
func ExtractSenadoMateria(materiaId string) (*PLP, error) {
	plp := &PLP{
		Id: materiaId,
	}

	err := plp.extractPLPSenadoFileData(
		materiaId,
		"Destaques",
		"https://www25.senado.leg.br/web/atividade/materias/-/materia/"+materiaId,
		func(data *PLPFileData, element *colly.HTMLElement) {

			data.SetData("identificacao", element.ChildText(".sf-texto-materia--coluna-dados dl dd:nth-child(2) span"))
			data.Autor = element.ChildText(".sf-texto-materia--coluna-dados dl dd:nth-child(4)")
			data.DataApresentacao = element.ChildText(".sf-texto-materia--coluna-dados dl dd:nth-child(6)")

			data.Ementa = element.ChildText(".sf-texto-materia--coluna-dados dl dd:nth-child(8)")

			data.Link = element.ChildAttr(".sf-texto-materia--coluna-link > span > a", "href")
		},
	)

	if err != nil {
		return nil, err
	}

	return plp, nil
}

func ExtractCamaraPLP(proposicaoId string) (*PLP, error) {
	plp := &PLP{
		Id: proposicaoId,
	}

	var err error

	err = plp.extractPLPCamaraFileData(
		proposicaoId,
		"Destaques",
		"https://www.camara.leg.br/proposicoesWeb/prop_destaques?idProposicao="+proposicaoId+"&subst=0",
		func(data *PLPFileData, element *colly.HTMLElement) {
			data.Data["ementa"] = element.ChildText("td:nth-child(1)")
			data.Data["dataApresentacao"] = element.ChildText("td:nth-child(2)")
			data.Data["autor"] = element.ChildText("td:nth-child(3)")
			data.SetLinkCamara(element.ChildAttr("td:nth-child(4) a", "href"))
		},
	)
	if err != nil {
		return nil, err
	}

	err = plp.extractPLPCamaraFileData(
		proposicaoId,
		"EmentaProjeto",
		"https://www.camara.leg.br/proposicoesWeb/prop_emendas?idProposicao="+proposicaoId+"&subst=0",
		func(data *PLPFileData, element *colly.HTMLElement) {
			data.Ementa = element.ChildText("td:nth-child(1)")
			data.DataApresentacao = element.ChildText("td:nth-child(3)")
			data.Autor = element.ChildText("td:nth-child(4)")
			data.SetLinkCamara(element.ChildAttr("td:nth-child(5) a", "href"))
		},
	)
	if err != nil {
		return nil, err
	}

	err = plp.extractPLPCamaraFileData(
		proposicaoId,
		"HistoricoDePareceres",
		"https://www.camara.leg.br/proposicoesWeb/prop_pareceres_substitutivos_votos?idProposicao="+proposicaoId+"&subst=0",
		func(data *PLPFileData, element *colly.HTMLElement) {
			data.Ementa = element.ChildText("td:nth-child(1)")
			data.DataApresentacao = element.ChildText("td:nth-child(3)")
			data.Autor = element.ChildText("td:nth-child(4)")
			data.SetLinkCamara(element.ChildAttr("td:nth-child(5) a", "href"))
		},
	)
	if err != nil {
		return nil, err
	}

	return plp, nil
}

func (plp *PLP) extractPLPSenadoFileData(
	proposicaoId string,
	plpType string,
	url string,
	extractDataFunc func(*PLPFileData, *colly.HTMLElement),
) error {
	plps := make([]*PLPFileData, 0)

	c := colly.NewCollector()
	c.SetRequestTimeout(time.Second * 20)

	c.OnHTML("#materia_documentos_emendas", func(e *colly.HTMLElement) {
		e.ForEach(".div-zebra > .sf-texto-materia", func(i int, e *colly.HTMLElement) {

			plp := &PLPFileData{
				Id:        proposicaoId,
				Type:      plpType,
				Metadados: EmptyFileMetadata(),
				Data:      map[string]string{},
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

func (plp *PLP) extractPLPCamaraFileData(
	proposicaoId string,
	plpType string,
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
				Type:      plpType,
				Metadados: EmptyFileMetadata(),
				Data:      map[string]string{},
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, plp.Link, nil)

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
