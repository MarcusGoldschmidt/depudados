package cameraleg

import (
	"context"
	"depudados/metadata"
	"encoding/json"
	"github.com/sourcegraph/conc/pool"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"sync"
)

type Proposicao struct {
	Id                int    `json:"id"`
	Uri               string `json:"uri"`
	SiglaTipo         string `json:"siglaTipo"`
	Numero            int    `json:"numero"`
	Ano               int    `json:"ano"`
	CodTipo           int    `json:"codTipo"`
	DescricaoTipo     string `json:"descricaoTipo"`
	Ementa            string `json:"ementa"`
	EmentaDetalhada   string `json:"ementaDetalhada"`
	Keywords          string `json:"keywords"`
	DataApresentacao  string `json:"dataApresentacao"`
	UriOrgaoNumerador string `json:"uriOrgaoNumerador"`
	UriPropAnterior   string `json:"uriPropAnterior"`
	UriPropPrincipal  string `json:"uriPropPrincipal"`
	UriPropPosterior  string `json:"uriPropPosterior"`
	UrlInteiroTeor    string `json:"urlInteiroTeor"`
	UltimoStatus      struct {
		Data                string `json:"data"`
		Sequencia           string `json:"sequencia"`
		UriRelator          string `json:"uriRelator"`
		CodOrgao            string `json:"codOrgao"`
		SiglaOrgao          string `json:"siglaOrgao"`
		UriOrgao            string `json:"uriOrgao"`
		Regime              string `json:"regime"`
		DescricaoTramitacao string `json:"descricaoTramitacao"`
		IdTipoTramitacao    string `json:"idTipoTramitacao"`
		DescricaoSituacao   string `json:"descricaoSituacao"`
		IdSituacao          string `json:"idSituacao"`
		Despacho            string `json:"despacho"`
		Apreciacao          string `json:"apreciacao"`
		Url                 string `json:"url"`
	} `json:"ultimoStatus"`
}

func LoadProposicao(reader io.Reader) ([]*Proposicao, error) {
	var response struct {
		Dados []*Proposicao `json:"dados"`
	}

	err := json.NewDecoder(reader).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.Dados, nil
}

func GetAllProposicao(ctx context.Context, fileName string, extractMetadata bool) (PLPList, error) {
	et, err := metadata.NewExtractorPool(runtime.NumCPU())
	if err != nil {
		return nil, err
	}

	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	proposicoes, err := LoadProposicao(file)
	if err != nil {
		return nil, err
	}

	// Start workers
	plpWorkerPool := pool.New().WithMaxGoroutines(100)

	resultLock := sync.Mutex{}
	result := make([]*PLP, 0)

	for _, prop := range proposicoes {
		prop := prop

		plpWorkerPool.Go(func() {
			plp, err := GetPLP(strconv.Itoa(prop.Id))
			if err != nil {
				log.Printf("[ERR] for %d err: %s", prop.Id, err.Error())
				return
			}
			resultLock.Lock()
			result = append(result, plp)
			resultLock.Unlock()

			plp.Process(ctx, et, extractMetadata)

			err = plp.AnyError()

			if err != nil {
				log.Printf("err for %s err: %s", plp.Id, err.Error())
			}
		})
	}

	plpWorkerPool.Wait()

	return result, ctx.Err()
}
