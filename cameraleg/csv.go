package cameraleg

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
)

type PLPList []*PLP

func formatStringCsv(value string) string {
	value = strings.ReplaceAll(value, ";", ",")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\"", "'")

	return value
}

func (v PLPList) ToCsv() string {
	buffer := bytes.NewBufferString("")

	fieldIndexMapping := map[string]int{}

	for _, item := range v {
		buffer.WriteString(item.ToCsv(fieldIndexMapping, false))
	}

	header := getHeader(fieldIndexMapping)

	return header + "\n" + buffer.String()
}

func (plp *PLP) ToCsv(fieldIndexMapping map[string]int, withHeader bool) string {
	buffer := bytes.NewBufferString("")

	for _, data := range plp.Files {
		buffer.WriteString(data.ToCsv(fieldIndexMapping))
	}

	if withHeader == false {
		return buffer.String()
	}

	header := getHeader(fieldIndexMapping)

	return header + "\n" + buffer.String()
}

func (plp *PLPFileData) ToCsv(fieldIndexMapping map[string]int) string {
	buffer := bytes.NewBufferString("")

	buffer.WriteString(formatStringCsv(plp.Id) + ";")
	buffer.WriteString(formatStringCsv(plp.Ementa) + ";")
	buffer.WriteString(formatStringCsv(plp.DataApresentacao) + ";")
	buffer.WriteString(formatStringCsv(plp.Autor) + ";")
	buffer.WriteString(plp.LinkInteiroTeor + ";")

	list := make([]string, len(fieldIndexMapping))

	for k, v := range plp.Metadados.Fields {
		value := formatStringCsv(fmt.Sprintf("%v", v))

		if index, ok := (fieldIndexMapping)[k]; ok {
			list[index] = value
		} else {
			(fieldIndexMapping)[k] = len(fieldIndexMapping)
			list = append(list, value)
		}
	}

	for _, v := range list {
		buffer.WriteString(v + ";")
	}

	buffer.WriteString("\n")

	return buffer.String()
}

func getHeader(fieldIndexMapping map[string]int) string {
	header := "Id;Ementa;DataApresentacao;Autor;LinkInteiroTeor;"

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

	return header
}
