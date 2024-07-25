package repository

import (
	"gorm.io/gorm"
)

type PLPModel struct {
	gorm.Model
	IdPlp string
}

type PLPFileDataModel struct {
	Id    string
	IdPlp string

	Type string

	Ementa           string
	DataApresentacao string
	Autor            string
	LinkInteiroTeor  string
}
