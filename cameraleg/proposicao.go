package cameraleg

import (
	"encoding/json"
	"github.com/sourcegraph/conc/iter"
	"log"
	"os"
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

func LoadProposicao(files []string) ([]*Proposicao, error) {
	if len(files) == 0 {
		return make([]*Proposicao, 0), nil
	}

	tempResponse, err := iter.MapErr(files, func(fileName *string) ([]*Proposicao, error) {
		var decodeTemp struct {
			Dados []*Proposicao `json:"dados"`
		}

		file, err := os.Open(*fileName)
		if err != nil {
			log.Printf("error opening file %s: %s", fileName, err.Error())
			return nil, err
		}

		err = json.NewDecoder(file).Decode(&decodeTemp)
		if err != nil {
			log.Printf("error decoding file %s: %s", fileName, err.Error())
			return nil, err
		}

		err = file.Close()
		if err != nil {
			return nil, err
		}

		return decodeTemp.Dados, nil
	})
	if err != nil {
		return nil, err
	}

	response := make([]*Proposicao, 0)
	for i := range tempResponse {
		response = append(response, tempResponse[i]...)
	}

	return response, nil
}
