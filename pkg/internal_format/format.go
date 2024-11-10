package internalformat

import "github.com/wanderer69/flow_processor/pkg/entity"

type ProcessFormat struct {
	Version      string
	IsCompressed bool
	Process      *entity.Process
	Sign         string
}
