package camunda7convertor

import (
	"context"
	_ "embed"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

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

type mapTag struct {
	root map[string]interface{}
}

/*
func (c *mapTag) UnmarshalXML1(d *xml.Decoder, start xml.StartElement) error {
	c.root = map[string]interface{}

	key := ""
	val := ""

	for {
		t, _ := d.Token()
		switch tt := t.(type) {
		case xml.StartElement:
			fmt.Println(">", tt)

		case xml.EndElement:
			fmt.Println("<", tt)
			if tt.Name == start.Name {
				return nil
			}

			if tt.Name.Local == "enabled" {
				c.m[key] = val
			}
		}
	}
}
*/

func (x *mapTag) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	x.root = map[string]any{"_": start.Name.Local}
	path := []map[string]any{x.root}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		switch elem := token.(type) {
		case xml.StartElement:
			newMap := map[string]any{"_": elem.Name.Local}
			path[len(path)-1][elem.Name.Local] = newMap
			path = append(path, newMap)
		case xml.EndElement:
			path = path[:len(path)-1]
		case xml.CharData:
			val := strings.TrimSpace(string(elem))
			if val == "" {
				break
			}
			curName := path[len(path)-1]["_"].(string)
			path[len(path)-2][curName] = typeConvert(val)
		}
	}
}

func typeConvert(s string) any {
	f, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return f
	}
	return s
}

func (cc *ConverterClient) Check(processRaw string) (bool, error) {
	var data mapTag // map[string]interface{}
	err := xml.Unmarshal([]byte(processRaw), &data)
	if err != nil {
		return false, nil
	}
	//fmt.Printf("`%#v`\r\n", data)
	/*
		verI, ok := data["version"]
		if !ok {
			return false, nil
		}
		ver, ok := verI.(string)
		if !ok {
			return false, nil
		}
		if ver != "0.5.0" {
			return false, nil
		}

		signI, ok := data["sign"]
		if !ok {
			return false, nil
		}
		sign, ok := signI.(string)
		if !ok {
			return false, nil
		}

		processI, ok := data["process"]
		if !ok {
			return false, nil
		}
		process, ok := processI.(*entity.Process)
		if !ok {
			return false, nil
		}

		dataRaw, err := json.Marshal(process)
		if err != nil {
			return false, nil
		}
		md := md5.New()
		calcSign := string(md.Sum(dataRaw))
		if calcSign != sign {
			return false, nil
		}
	*/
	return true, nil
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
