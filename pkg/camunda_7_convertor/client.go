package camunda7convertor

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/wanderer69/flow_processor/pkg/entity"
)

//go:embed module/convertor.wasm
var convertorWasm []byte

type ConverterClient struct {
}

func NewConverterClient() *ConverterClient {
	return &ConverterClient{}
}

func (cc *ConverterClient) Convert(process string) (string, error) {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx) // This closes everything this Runtime created.
	_, err := r.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(logString).Export("log").
		NewFunctionBuilder().WithFunc(getResult).Export("getResult").
		Instantiate(ctx)
	if err != nil {
		return "", err
	}

	wasi_snapshot_preview1.MustInstantiate(ctx, r)
	mod, err := r.Instantiate(ctx, convertorWasm)
	if err != nil {
		return "", err
	}

	malloc := mod.ExportedFunction("malloc")
	free := mod.ExportedFunction("free")

	convertFromCamunda7 := mod.ExportedFunction("convertFromCamunda7")
	results, err := malloc.Call(ctx, uint64(len(process)))
	if err != nil {
		return "", err
	}
	processPtr := results[0]
	defer free.Call(ctx, processPtr)

	if !mod.Memory().Write(uint32(processPtr), []byte(process)) {
		return "", fmt.Errorf("Memory.Write(%d, %d) out of range of memory size %d", processPtr, len(process), mod.Memory().Size())
	}

	ptrSize, err := convertFromCamunda7.Call(ctx, processPtr, uint64(len(process)))
	if err != nil {
		return "", err
	}

	resultPtr := uint32(ptrSize[0] >> 32)
	resultSize := uint32(ptrSize[0])
	if resultPtr != 0 {
		defer func() {
			_, err := free.Call(ctx, uint64(resultPtr))
			if err != nil {
				fmt.Printf("%v\r\n", err)
			}
		}()
	}

	bytes, ok := mod.Memory().Read(resultPtr, resultSize)
	if !ok {
		return "", fmt.Errorf("Memory.Read(%d, %d) out of range of memory size %d", resultPtr, resultSize, mod.Memory().Size())
	}
	// fmt.Println("go >>", string(bytes))

	var resultFunc entity.Result
	err = json.Unmarshal(bytes, &resultFunc)
	if err != nil {
		return "", err
	}
	//fmt.Printf(">> %#v\r\n", resultFunc)
	if resultFunc.Err != nil {
		return "", fmt.Errorf("error convert %w", resultFunc.Err)
	}
	return resultFunc.Result, nil
}
