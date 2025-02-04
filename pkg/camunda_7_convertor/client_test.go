package camunda7convertor

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wanderer69/flow_processor/pkg/entity"
)

func TestConvert1(t *testing.T) {
	cc := NewConverterClient()
	result, err := cc.Convert(process2)
	require.NoError(t, err)
	require.Len(t, result, 4865)
	var process []*entity.Process
	require.NoError(t, json.Unmarshal([]byte(result), &process))
}

func TestConvert2(t *testing.T) {
	cc := NewConverterClient()
	result, err := cc.Check(process2)
	require.NoError(t, err)
	require.Len(t, result, 4865)
}
