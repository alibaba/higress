package pkg

import (
	"errors"
	"fmt"

	"github.com/tidwall/gjson"
)

const (
	RefTypeRequestHeader  = "request_header"
	RefTypeRequestQuery   = "request_query"
	RefTypeResponseHeader = "response_header"
)

var (
	refType2Stage = map[string]Stage{
		RefTypeRequestHeader:  StageRequestHeaders,
		RefTypeRequestQuery:   StageRequestHeaders,
		RefTypeResponseHeader: StageResponseHeaders,
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

	if _, ok := refType2Stage[ref.Type]; !ok {
		return nil, fmt.Errorf("unknown ref type: %s", ref.Type)
	}

	if name := json.Get("name").String(); name != "" {
		ref.Name = name
	} else {
		return nil, errors.New("missing name field")
	}

	return ref, nil
}

func (r *Ref) GetStage() Stage {
	if r.stage == 0 {
		if stage, ok := refType2Stage[r.Type]; ok {
			r.stage = stage
		}
	}
	return r.stage
}

func (r *Ref) String() string {
	return fmt.Sprintf("%s/%s", r.Type, r.Name)
}
