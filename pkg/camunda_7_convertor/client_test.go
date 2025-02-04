package camunda7convertor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvert1(t *testing.T) {
	cc := NewConverterClient()
	result, err := cc.Convert(process2)
	require.NoError(t, err)
	require.Len(t, result, 4865)
}

func TestConvert2(t *testing.T) {
	cc := NewConverterClient()
	result, err := cc.Check(process2)
	require.NoError(t, err)
	require.Len(t, result, 4865)
}
