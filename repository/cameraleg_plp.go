package repository

import (
	"bytes"
	"context"
	"fmt"
	"github.com/barasher/go-exiftool"
	"github.com/gocolly/colly/v2"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

type PLP struct {
	Destaques            []*PLPFileData
	EmentaProjeto        []*PLPFileData
	HistoricoDePareceres []*PLPFileData
}

type PLPFileData struct {
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

func GetPLP(proposicaoId string) (*PLP, error) {
	plp := &PLP{}

	var err error

	plp.Destaques, err = extractPLPFileData(
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

func extractPLPFileData(url string, extractDataFunc func(*PLPFileData, *colly.HTMLElement)) ([]*PLPFileData, error) {
	plps := make([]*PLPFileData, 0)

	c := colly.NewCollector()

	c.OnHTML(".coresAlternadas", func(e *colly.HTMLElement) {
		e.ForEach("tr", func(i int, e *colly.HTMLElement) {

			plp := &PLPFileData{
				Metadados: make([]exiftool.FileMetadata, 0),
			}

			extractDataFunc(plp, e)

			plps = append(plps, plp)
		})
	})

	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting: ", r.URL.String())
	})

	err := c.Visit(url)
	if err != nil {
		return nil, err
	}
	return plps, nil
}

func (plp *PLP) Process(ctx context.Context) {
	wg := &sync.WaitGroup{}
	receiver := make(chan *PLPFileData)

	for i := 0; i < 20; i++ {
		go plpWorker(ctx, wg, receiver)
	}

	for _, plp := range plp.Destaques {
		wg.Add(1)
		receiver <- plp
	}

	for _, plp := range plp.EmentaProjeto {
		wg.Add(1)
		receiver <- plp
	}

	for _, plp := range plp.HistoricoDePareceres {
		wg.Add(1)
		receiver <- plp
	}

	wg.Wait()
	close(receiver)
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

	log.Printf("Downloading file: %s\n", plp.LinkInteiroTeor)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func plpWorker(ctx context.Context, wg *sync.WaitGroup, receiver <-chan *PLPFileData) {
	et, err1 := exiftool.NewExiftool()
	if err1 != nil {
		log.Printf("Error when intializing: %v\n", err1)
		return
	}

	defer et.Close()

	for plp := range receiver {
		func(plp *PLPFileData) {
			defer wg.Done()

			fileName := fmt.Sprintf("./out/%s-%s.pdf", strings.ReplaceAll(plp.Ementa, "/", "-"), strings.ReplaceAll(plp.DataApresentacao, "/", "-"))

			file, err := os.ReadFile(fileName)
			if err != nil {
				file, err = plp.Download(ctx)
				if err != nil {
					plp.Err = err
					log.Printf("Error downloading file: %v\n", err)
					return
				}
			} else {
				log.Printf("cache hit for %s\n", fileName)
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

			fileInfos := et.ExtractMetadata(fileName)

			for _, fileInfo := range fileInfos {
				if fileInfo.Err != nil {
					return
				}

				plp.Metadados = append(plp.Metadados, fileInfo)
			}
		}(plp)
	}
}
