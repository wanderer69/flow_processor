package internalformat

import (
	"crypto/sha256"
	"encoding/json"

	"github.com/wanderer69/flow_processor/pkg/entity"
)

type InternalFormat struct {
}

func NewInternalFormat() *InternalFormat {
	return &InternalFormat{}
}

func (nl *InternalFormat) Convert(processRaw string) (string, error) {
	var pf ProcessFormat
	err := json.Unmarshal([]byte(processRaw), &pf)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(pf.Process)
	return string(data), err
}

func (nl *InternalFormat) Check(processRaw string) (bool, error) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(processRaw), &data)
	if err != nil {
		return false, nil
	}
	verI, ok := data["Version"]
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

	signI, ok := data["Sign"]
	if !ok {
		return false, nil
	}
	sign, ok := signI.(string)
	if !ok {
		return false, nil
	}

	processI, ok := data["Process"]
	if !ok {
		return false, nil
	}

	process, ok := processI.(map[string]interface{})
	if !ok {
		return false, nil
	}

	nameI, ok := process["Name"]
	if !ok {
		return false, nil
	}
	name, ok := nameI.(string)
	if !ok {
		return false, nil
	}
	if len(name) == 0 {
		return false, nil
	}

	if len(sign) == 0 {
		return false, nil
	}

	/*
		dataRaw, err := json.Marshal(processI)
		if err != nil {
			return false, nil
		}
	*/
	/*
		md := sha256.New()
		calcSign := string(md.Sum(dataRaw))
		if calcSign != sign {
			return false, nil
		}
	*/
	return true, nil
}

func (nl *InternalFormat) Store(processRaw string) (string, error) {
	var process entity.Process
	err := json.Unmarshal([]byte(processRaw), &process)
	if err != nil {
		return "", err
	}

	md := sha256.New()

	pf := ProcessFormat{
		Version: "0.5.0",
		Sign:    string(md.Sum([]byte(processRaw))),
		Process: &process,
	}
	data, err := json.Marshal(pf)
	return string(data), err
}
