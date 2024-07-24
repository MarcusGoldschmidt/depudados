package cameraleg

import (
	"bytes"
	"context"
	"depudados/metadata"
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
)

type PLP struct {
	Id                   string
	Destaques            []*PLPFileData
	EmentaProjeto        []*PLPFileData
	HistoricoDePareceres []*PLPFileData
}

func (plp *PLP) Len() int {
	return len(plp.Destaques) + len(plp.EmentaProjeto) + len(plp.HistoricoDePareceres)
}

type PLPFileData struct {
	Id               string
	Ementa           string
	DataApresentacao string
	Autor            string
	LinkInteiroTeor  string

	Metadados []exiftool.FileMetadata

	Err error
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

func GetPLP(proposicaoId string) (*PLP, error) {
	plp := &PLP{
		Id: proposicaoId,
	}

	var err error

	plp.Destaques, err = extractPLPFileData(
		proposicaoId,
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

	plp.EmentaProjeto, err = extractPLPFileData(
		proposicaoId,
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

	plp.HistoricoDePareceres, err = extractPLPFileData(
		proposicaoId,
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

func extractPLPFileData(proposicaoId string, url string, extractDataFunc func(*PLPFileData, *colly.HTMLElement)) ([]*PLPFileData, error) {
	plps := make([]*PLPFileData, 0)

	c := colly.NewCollector()

	c.OnHTML(".coresAlternadas", func(e *colly.HTMLElement) {
		e.ForEach("tr", func(i int, e *colly.HTMLElement) {

			plp := &PLPFileData{
				Id:        proposicaoId,
				Metadados: make([]exiftool.FileMetadata, 0),
			}

			extractDataFunc(plp, e)

			plps = append(plps, plp)
		})
	})

	err := c.Visit(url)
	if err != nil {
		return nil, err
	}
	return plps, nil
}

func (plp *PLP) getOutputPath() string {
	return filepath.Join(".", "out", plp.Id)
}

func (plp *PLP) Process(ctx context.Context, et *metadata.ExtractorPool, extractMetadata bool) {
	if plp.Len() == 0 {
		log.Printf("Skipping PLP %s has no files to process\n", plp.Id)
		return
	}

	log.Printf("Processing PLP %s len: %d\n", plp.Id, plp.Len())

	_ = os.MkdirAll(plp.getOutputPath(), os.ModePerm)

	workerCount := runtime.NumCPU() * 2
	if plp.Len() < workerCount {
		workerCount = plp.Len()
	}

	plpFileWorkerPool := pool.New().WithMaxGoroutines(workerCount)

	for _, plp := range plp.Destaques {
		plp := plp
		plpFileWorkerPool.Go(func() {
			plpWorker(ctx, et, plp, extractMetadata)
		})
	}

	for _, plp := range plp.EmentaProjeto {
		plp := plp
		plpFileWorkerPool.Go(func() {
			plpWorker(ctx, et, plp, extractMetadata)
		})
	}

	for _, plp := range plp.HistoricoDePareceres {
		plp := plp
		plpFileWorkerPool.Go(func() {
			plpWorker(ctx, et, plp, extractMetadata)
		})
	}

	plpFileWorkerPool.Wait()
}

func (plp *PLP) AnyError() error {
	for _, plpFileData := range plp.Destaques {
		if plpFileData.Err != nil {
			return plpFileData.Err
		}
	}

	for _, plpFileData := range plp.EmentaProjeto {
		if plpFileData.Err != nil {
			return plpFileData.Err
		}
	}

	for _, plpFileData := range plp.HistoricoDePareceres {
		if plpFileData.Err != nil {
			return plpFileData.Err
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
			plp.Err = err
			log.Printf("Error downloading file: %v\n", err)
			return
		}
	}

	create, err := os.Create(fileName)
	if err != nil {
		plp.Err = err
		return
	}

	_, err = io.Copy(create, bytes.NewBuffer(file))
	if err != nil {
		plp.Err = err
		return
	}

	if extractMetadata == true {
		fileInfos := et.ExtractMetadata(fileName)

		for _, fileInfo := range fileInfos {
			if fileInfo.Err != nil {
				return
			}

			plp.Metadados = append(plp.Metadados, fileInfo)
		}
	}
}
