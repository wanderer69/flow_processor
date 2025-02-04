package loader

import (
	"encoding/json"
	"fmt"

	"github.com/wanderer69/flow_processor/pkg/entity"
)

type Loader struct {
	camunda7Convertor Camunda7Convertor
	internalFormat    InternalFormat
}

func NewLoader(
	camunda7Convertor Camunda7Convertor,
	internalFormat InternalFormat,
) *Loader {
	return &Loader{
		camunda7Convertor: camunda7Convertor,
		internalFormat:    internalFormat,
	}
}

func (l *Loader) Load(processRaw string) (*entity.Process, error) {
	if l.camunda7Convertor != nil {
		ok, _ := l.camunda7Convertor.Check(processRaw)
		if ok {
			loadedProcess, err := l.camunda7Convertor.Convert(processRaw)
			if err == nil {
				var process []*entity.Process
				err := json.Unmarshal([]byte(loadedProcess), &process)
				if err == nil {
					return process[0], nil
				}
			}
		}
	}
	if l.internalFormat != nil {
		ok, _ := l.internalFormat.Check(processRaw)
		if ok {
			loadedProcess, err := l.internalFormat.Convert(processRaw)
			if err == nil {
				var process entity.Process
				err := json.Unmarshal([]byte(loadedProcess), &process)
				if err == nil {
					return &process, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("format unsupported")
}

func (l *Loader) Save(process *entity.Process) (string, error) {
	dataRaw, err := json.Marshal(process)
	if err != nil {
		return "", err
	}
	return l.internalFormat.Store(string(dataRaw))
}
