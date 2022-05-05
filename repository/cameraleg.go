package repository

import (
	"bytes"
	"context"
	"depudados/models"
	"encoding/json"
	"fmt"
	"github.com/barasher/go-exiftool"
	"github.com/gocolly/colly/v2"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

type govResponse struct {
	Dados []struct {
		Id        int    `json:"id"`
		Uri       string `json:"uri"`
		SiglaTipo string `json:"siglaTipo"`
		CodTipo   int    `json:"codTipo"`
		Numero    int    `json:"numero"`
		Ano       int    `json:"ano"`
		Ementa    string `json:"ementa"`
	} `xml:"dados"`
	Links []struct {
		Rel  string `json:"rel"`
		Href string `json:"href"`
	} `json:"links"`
}

type proposicaoResponse struct {
	Dados struct {
		Id                int    `json:"id"`
		Uri               string `json:"uri"`
		SiglaTipo         string `json:"siglaTipo"`
		CodTipo           int    `json:"codTipo"`
		Numero            int    `json:"numero"`
		Ano               int    `json:"ano"`
		Ementa            string `json:"ementa"`
		DataApresentacao  string `json:"dataApresentacao"`
		UriOrgaoNumerador string `json:"uriOrgaoNumerador"`
		StatusProposicao  struct {
			DataHora            string `json:"dataHora"`
			Sequencia           int    `json:"sequencia"`
			SiglaOrgao          string `json:"siglaOrgao"`
			UriOrgao            string `json:"uriOrgao"`
			Regime              string `json:"regime"`
			DescricaoTramitacao string `json:"descricaoTramitacao"`
			DescricaoSituacao   string `json:"descricaoSituacao"`
			CodSituacao         int    `json:"codSituacao"`
			Despacho            string `json:"despacho"`
			Url                 string `json:"url"`
			Ambito              string `json:"ambito"`
		} `json:"statusProposicao"`
		UriAutores       string        `json:"uriAutores"`
		DescricaoTipo    string        `json:"descricaoTipo"`
		EmentaDetalhada  string        `json:"ementaDetalhada"`
		Keywords         string        `json:"keywords"`
		UriPropPrincipal string        `json:"uriPropPrincipal"`
		UriPropAnterior  []interface{} `json:"uriPropAnterior"`
		UriPropPosterior []interface{} `json:"uriPropPosterior"`
		UrlInteiroTeor   string        `json:"urlInteiroTeor"`
		UrnFinal         []interface{} `json:"urnFinal"`
		Texto            []interface{} `json:"texto"`
		Justificativa    []interface{} `json:"justificativa"`
	} `json:"dados"`
}

const deputadosUrl = "https://www.camara.leg.br/deputados/quem-sao/resultado?search=&partido=&uf=&legislatura=56&sexo=&pagina="

const arquivosUrl = "https://dadosabertos.camara.leg.br/api/v2/proposicoes?autor=%s&pagina=%d&ordem=ASC&ordenarPor=id&ano=2022&ano=2021&ano=2020&ano=2019"

func GetDeputados() ([]*models.Deputado, error) {
	c := colly.NewCollector()

	deputados := make([]*models.Deputado, 0)

	c.OnHTML(".lista-resultados__cabecalho > a", func(e *colly.HTMLElement) {

		link := e.Attr("href")
		split := strings.Split(link, "/")
		deputados = append(deputados, models.NewDeputado(split[len(split)-1], e.Text))
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting: ", r.URL.String())
	})

	for i := 1; i < 25; i++ {
		err := c.Visit(deputadosUrl + strconv.Itoa(i))
		if err != nil {
			return nil, err
		}
	}

	return deputados, nil
}

func GetProposicoes(p *Persistence, deputados []*models.Deputado) ([]*models.Proposicao, error) {

	proposicoes := make([]*models.Proposicao, 0)

	for _, deputado := range deputados {
		if p.GetWorkDeputado(deputado.Nome) {
			fmt.Println("Pulando deputado " + deputado.Nome)
			continue
		}

		numeroPagina := 1

		sem := make(chan int, 4)
		propChan := make(chan *models.Proposicao)

		var wg sync.WaitGroup

		ctx, cancel := context.WithCancel(context.Background())

		// Init consumer
		go func() {
			for proposicao := range propChan {
				fmt.Printf("Inserindo autor %s Url: %s\n", proposicao.Autor, proposicao.Url)
				_ = p.LoadProposicoes([]*models.Proposicao{proposicao})
				proposicoes = append(proposicoes, proposicao)
			}
		}()

	Loop:
		for {
			select {
			case <-ctx.Done():
				break Loop
			default:
				wg.Add(1)
				sem <- 1
				go worker(p, deputado.Nome, numeroPagina, sem, &wg, propChan, cancel)
				numeroPagina++
			}
		}
		wg.Wait()
		close(propChan)
		err := p.SetWorkDoneDeputado(deputado.Nome)
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println(len(proposicoes))

	return proposicoes, nil
}

func worker(p *Persistence, nome string, numeroPagina int, sem chan int, wg *sync.WaitGroup, propChan chan *models.Proposicao, cancel context.CancelFunc) {
	defer func() {
		<-sem
		wg.Done()
	}()

	fmt.Printf("Init %s page %d\n", nome, numeroPagina)

	nome = nome[:strings.Index(nome, "(")-1]

	url := fmt.Sprintf(arquivosUrl, nome, numeroPagina)

	url = strings.ReplaceAll(url, " ", "%20")

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	responseBody, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	var response govResponse

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		fmt.Println("error: " + url)
		resp.Body.Close()
		cancel()
		return
	}
	resp.Body.Close()

	if len(response.Dados) == 0 {
		cancel()
	}

	et, err := exiftool.NewExiftool()
	if err != nil {
		fmt.Printf("Error when intializing: %v\n", err)
		return
	}
	defer et.Close()

	for _, dado := range response.Dados {
		req, err := http.NewRequest(http.MethodGet, dado.Uri, nil)

		if err != nil {
			continue
		}

		resp, err := client.Do(req)
		responseBody, err := ioutil.ReadAll(resp.Body)

		var response proposicaoResponse

		err = json.Unmarshal(responseBody, &response)
		if err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		url := response.Dados.StatusProposicao.Url

		if url == "" {
			url = response.Dados.UrlInteiroTeor
		}

		if p.ExistProposicao(url) {
			fmt.Println("Pulando ", url)
			continue
		}

		metadados, err := obtemMetadados(response, et, url, nome)
		if err != nil {
			continue
		}

		propChan <- metadados

		resp.Body.Close()
	}

	fmt.Println("Finalizando ", numeroPagina)

}

func obtemMetadados(response proposicaoResponse, et *exiftool.Exiftool, url string, deputado string) (*models.Proposicao, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {
		return nil, err
	}
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	file, err := os.CreateTemp("", "depudados_")
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return nil, err
	}

	fileInfos := et.ExtractMetadata(file.Name())

	author := ""

	bufer := bytes.NewBufferString("")

	for _, fileInfo := range fileInfos {
		if fileInfo.Err != nil {
			continue
		}

		if v, ok := fileInfo.Fields["Author"].(string); ok {
			author = v
		}

		for k, v := range fileInfo.Fields {
			_, err := fmt.Fprintf(bufer, "[%v] %v\n", k, v)
			if err != nil {
				continue
			}
		}
	}

	prop := models.NewProposicao(strconv.Itoa(response.Dados.Id), bufer.String(), author, url, deputado)

	file.Close()
	os.Remove(file.Name())

	return prop, nil
}
