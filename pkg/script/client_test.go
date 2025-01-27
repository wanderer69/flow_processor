package script

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wanderer69/flow_processor/pkg/entity"
)

func TestLexema1(t *testing.T) {
	test1 := "${isTest}"
	ll, err := ParserLexema(test1)
	require.NoError(t, err)
	for i := range ll {
		fmt.Printf("%#v\r\n", *ll[i])
	}
	//require.Equal()
}

func TestLexema2(t *testing.T) {
	test1 := `${isTest == "value"}`
	ll, err := ParserLexema(test1)
	require.NoError(t, err)
	for i := range ll {
		fmt.Printf("%#v\r\n", *ll[i])
	}
	//require.Equal()
}

func TestPattern1(t *testing.T) {
	test1 := "${isTest}"
	ll, err := ParserLexema(test1)
	require.NoError(t, err)
	for i := range ll {
		fmt.Printf("%#v\r\n", *ll[i])
	}
	//require.Equal()

	context := entity.Context{
		VariablesByName: make(map[string]*entity.Variable),
		MessagesByName:  make(map[string]*entity.Message),
	}

	vars, err := TranslateLexemaList(ll, &context)
	require.NoError(t, err)
	require.Len(t, vars, 1)

	context.VariablesByName["isTest"] = &entity.Variable{
		Name:  "isTest",
		Type:  "boolean",
		Value: "true",
	}
	vars, err = TranslateLexemaList(ll, &context)
	require.NoError(t, err)
	require.Len(t, vars, 1)
}

func TestPattern2(t *testing.T) {
	test1 := `${isTest == "value"}`
	ll, err := ParserLexema(test1)
	require.NoError(t, err)
	for i := range ll {
		fmt.Printf("%#v\r\n", *ll[i])
	}
	//require.Equal()

	context := entity.Context{
		VariablesByName: make(map[string]*entity.Variable),
		MessagesByName:  make(map[string]*entity.Message),
	}
	/*
		vars, err := TranslateLexemaList(ll, &context)
		require.NoError(t, err)
		require.Len(t, vars, 1)
	*/
	context.VariablesByName["isTest"] = &entity.Variable{
		Name:  "isTest",
		Type:  "string",
		Value: "not_value",
	}
	vars, err := TranslateLexemaList(ll, &context)
	require.NoError(t, err)
	require.Len(t, vars, 1)
	context.VariablesByName["isTest"] = &entity.Variable{
		Name:  "isTest",
		Type:  "string",
		Value: "value",
	}
	vars, err = TranslateLexemaList(ll, &context)
	require.NoError(t, err)
	require.Len(t, vars, 1)
}
