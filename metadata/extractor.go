package metadata

import (
	"github.com/barasher/go-exiftool"
)

type ExtractorPool struct {
	workerCount int

	send chan struct {
		file      string
		valueBack chan []exiftool.FileMetadata
	}
}

func (m *ExtractorPool) Close() error {
	close(m.send)

	return nil
}

func NewExtractorPool(workerCount int) (*ExtractorPool, error) {
	if workerCount <= 0 {
		workerCount = 1
	}

	meta := &ExtractorPool{workerCount: workerCount,
		send: make(chan struct {
			file      string
			valueBack chan []exiftool.FileMetadata
		}, workerCount),
	}

	err := meta.startPool()
	if err != nil {
		return nil, err
	}

	return meta, nil
}

func (m *ExtractorPool) ExtractMetadata(file string) []exiftool.FileMetadata {
	valueBack := make(chan []exiftool.FileMetadata)

	m.send <- struct {
		file      string
		valueBack chan []exiftool.FileMetadata
	}{
		file:      file,
		valueBack: valueBack,
	}

	return <-valueBack
}

func (m *ExtractorPool) startPool() error {
	for i := 0; i < m.workerCount; i++ {
		et, err := exiftool.NewExiftool()
		if err != nil {
			return err
		}

		go func() {
			for value := range m.send {
				response := et.ExtractMetadata(value.file)
				value.valueBack <- response
			}
		}()
	}

	return nil
}
