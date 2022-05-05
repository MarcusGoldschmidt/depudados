package models

type Deputado struct {
	Id   string
	Nome string
}

type Proposicao struct {
	Nome      string
	Metadados string
	Autor     string
	Url       string
	Deputado  string
}

func NewProposicao(nome string, metadados string, autor string, url string, deputado string) *Proposicao {
	return &Proposicao{Nome: nome, Metadados: metadados, Autor: autor, Url: url, Deputado: deputado}
}

func NewDeputado(id, nome string) *Deputado {
	return &Deputado{id, nome}
}
