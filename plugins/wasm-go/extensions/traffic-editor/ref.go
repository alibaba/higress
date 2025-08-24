package main

import (
	"errors"

	"github.com/tidwall/gjson"
)

const (
	refTypeRequestHeader  = "requestHeader"
	refTypeRequestQuery   = "requestQuery"
	refTypeResponseHeader = "responseHeader"
)

var (
	refType2Stage = map[string]Stage{
		refTypeRequestHeader:  StageRequestHeaders,
		refTypeRequestQuery:   StageRequestHeaders,
		refTypeResponseHeader: StageResponseHeaders,
	}
)

type Ref struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`

	stage Stage
}

func NewRef(json gjson.Result) (*Ref, error) {
	ref := &Ref{}

	if t := json.Get("type").String(); t != "" {
		ref.Type = t
	} else {
		return nil, errors.New("missing type field")
	}

	if stage, ok := refType2Stage[ref.Type]; ok {
		ref.stage = stage
	} else {
		return nil, errors.New("invalid type field: " + ref.Type)
	}

	if name := json.Get("name").String(); name != "" {
		ref.Name = name
	} else {
		return nil, errors.New("missing name field")
	}

	return ref, nil
}

func (r *Ref) GetStage() Stage {
	return r.stage
}
