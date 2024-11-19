package store

import "github.com/wanderer69/flow_processor/pkg/entity"

type Loader interface {
	Save(process *entity.Process) (string, error)
	Load(processRaw string) (*entity.Process, error)
}
