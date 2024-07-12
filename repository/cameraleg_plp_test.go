package repository

import "testing"

func TestGetPLP(t *testing.T) {
	plp, err := GetPLP("2430143")
	if err != nil {
		t.Error(err)
		return
	}

	if plp == nil {
		t.Error("PLP is nil")
	}
}

func TestExtractPLPFileData(t *testing.T) {
	plp, err := extractPLPFileData("https://www.camara.leg.br/proposicoesWeb/prop_emendas?idProposicao=2430143&subst=0")
	if err != nil {
		t.Error(err)
		return
	}

	if plp == nil {
		t.Error("PLP is nil")
	}
}
