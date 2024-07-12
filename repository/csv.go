package repository

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
)

func (plp *PLP) ToCsv() string {
	fieldIndexMapping := map[string]int{}

	buffer := bytes.NewBufferString("")

	for _, data := range plp.Destaques {
		buffer.WriteString(data.ToCsv(&fieldIndexMapping))
	}

	for _, data := range plp.EmentaProjeto {
		buffer.WriteString(data.ToCsv(&fieldIndexMapping))
	}

	for _, data := range plp.HistoricoDePareceres {
		buffer.WriteString(data.ToCsv(&fieldIndexMapping))
	}

	header := "Ementa;DataApresentacao;Autor;LinkInteiroTeor;"

	headerPosition := make([]struct {
		v   string
		idx int
	}, 0)

	for k, v := range fieldIndexMapping {
		headerPosition = append(headerPosition, struct {
			v   string
			idx int
		}{v: k, idx: v})
	}

	slices.SortFunc(headerPosition, func(a, b struct {
		v   string
		idx int
	}) int {
		return a.idx - b.idx
	})

	for _, v := range headerPosition {
		header += v.v + ";"
	}

	return header + "\n" + buffer.String()
}

func (plp *PLPFileData) ToCsv(fieldIndexMapping *map[string]int) string {
	buffer := bytes.NewBufferString("")

	buffer.WriteString(plp.Ementa + ";")
	buffer.WriteString(plp.DataApresentacao + ";")
	buffer.WriteString(plp.Autor + ";")
	buffer.WriteString(plp.LinkInteiroTeor + ";")

	list := make([]string, len(*fieldIndexMapping))

	for _, metadata := range plp.Metadados {
		for k, v := range metadata.Fields {

			value := strings.ReplaceAll(fmt.Sprintf("%v", v), ";", ",")

			if index, ok := (*fieldIndexMapping)[k]; ok {
				list[index] = value
			} else {
				(*fieldIndexMapping)[k] = len(*fieldIndexMapping)
				list = append(list, value)
			}
		}
	}

	for _, v := range list {
		buffer.WriteString(v + ";")
	}

	buffer.WriteString("\n")

	return buffer.String()
}
