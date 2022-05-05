package repository

import (
	"bytes"
	"depudados/models"
	"encoding/gob"
	"github.com/boltdb/bolt"
)

type Persistence struct {
	db *bolt.DB
}

func NewPersistence(DB *bolt.DB) *Persistence {
	return &Persistence{DB}
}

func (p Persistence) Close() error {
	return p.db.Close()
}

func (p Persistence) SetWorkDoneDeputado(deputado string) error {
	return load[string](p, "WORK", []*string{&deputado}, func(t *string) string {
		return *t
	})
}

func (p Persistence) GetWorkDeputado(deputadoNome string) bool {
	result := false

	err := p.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("WORK"))

		get := b.Get([]byte(deputadoNome))

		result = get != nil

		return nil
	})
	if err != nil {
		return false
	}

	return result
}

func (p Persistence) LoadDeputados(deputados []*models.Deputado) error {
	return load[models.Deputado](p, "DEPUTADOS", deputados, func(t *models.Deputado) string {
		return t.Nome
	})
}

func (p Persistence) LoadProposicoes(data []*models.Proposicao) error {
	return load[models.Proposicao](p, "PROPOSICAO", data, func(t *models.Proposicao) string {
		return t.Url
	})
}

func load[T any](p Persistence, bucket string, data []*T, getKey func(*T) string) error {
	err := p.db.Batch(func(tx *bolt.Tx) error {

		bucket := tx.Bucket([]byte(bucket))

		for _, value := range data {
			buf := bytes.Buffer{}
			enc := gob.NewEncoder(&buf)
			err := enc.Encode(value)
			if err != nil {
				return err
			}

			err = bucket.Put([]byte(getKey(value)), buf.Bytes())
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func getAll[T any](p Persistence, bucket string) ([]*T, error) {
	data := make([]*T, 0)

	err := p.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))

		err := b.ForEach(func(k, v []byte) error {
			enc := gob.NewDecoder(bytes.NewBuffer(v))

			var value T

			err := enc.Decode(&value)
			if err != nil {
				return err
			}

			data = append(data, &value)
			return nil
		})

		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return data, err
}

func (p Persistence) GetDeputados() ([]*models.Deputado, error) {
	return getAll[models.Deputado](p, "DEPUTADOS")
}

func (p Persistence) GetProposicao() ([]*models.Proposicao, error) {
	return getAll[models.Proposicao](p, "PROPOSICAO")
}

func (p Persistence) ExistProposicao(url string) bool {
	result := false

	err := p.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("PROPOSICAO"))

		get := b.Get([]byte(url))

		result = get != nil

		return nil
	})
	if err != nil {
		return false
	}

	return result
}
