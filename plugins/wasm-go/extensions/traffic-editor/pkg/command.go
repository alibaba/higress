package pkg

import (
	"errors"
	"fmt"

	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/tidwall/gjson"
)

const (
	commandTypeSet    = "set"
	commandTypeConcat = "concat"
	commandTypeCopy   = "copy"
	commandTypeDelete = "delete"
	commandTypeRename = "rename"
)

var (
	commandFactories = map[string]func(gjson.Result) (Command, error){
		"set":    newSetCommand,
		"concat": newConcatCommand,
		"copy":   newCopyCommand,
		"delete": newDeleteCommand,
		"rename": newRenameCommand,
	}
)

type CommandSet struct {
	DisableReroute bool           `json:"disableReroute"`
	Commands       []Command      `json:"commands,omitempty"`
	RelatedStages  map[Stage]bool `json:"-"`
}

func (s *CommandSet) FromJson(json gjson.Result) error {
	relatedStages := map[Stage]bool{}
	if commandsJson := json.Get("commands"); commandsJson.Exists() && commandsJson.IsArray() {
		for _, item := range commandsJson.Array() {
			if command, err := NewCommand(item); err != nil {
				return fmt.Errorf("failed to create command from json: %v\n  %v", err, item)
			} else {
				s.Commands = append(s.Commands, command)
				for _, ref := range command.GetRefs() {
					relatedStages[ref.GetStage()] = true
				}
			}
		}
	}
	s.RelatedStages = relatedStages
	if disableReroute := json.Get("disableReroute"); disableReroute.Exists() {
		s.DisableReroute = disableReroute.Bool()
	} else {
		s.DisableReroute = false
	}
	return nil
}

func (s *CommandSet) CreatExecutors() []Executor {
	executors := make([]Executor, 0, len(s.Commands))
	for _, command := range s.Commands {
		executor := command.CreateExecutor()
		executors = append(executors, executor)
	}
	return executors
}

type ConditionalCommandSet struct {
	ConditionSet
	CommandSet
}

func (s *ConditionalCommandSet) FromJson(json gjson.Result) error {
	if err := s.ConditionSet.FromJson(json); err != nil {
		return err
	}
	if err := s.CommandSet.FromJson(json); err != nil {
		return err
	}
	return nil
}

type Command interface {
	GetType() string
	GetRefs() []*Ref
	CreateExecutor() Executor
}

type Executor interface {
	GetCommand() Command
	Run(editorContext EditorContext, stage Stage) error
}

func NewCommand(json gjson.Result) (Command, error) {
	t := json.Get("type").String()
	if t == "" {
		return nil, errors.New("command type is required")
	}
	if constructor, ok := commandFactories[t]; ok && constructor != nil {
		return constructor(json)
	} else {
		return nil, errors.New("unknown command type: " + t)
	}
}

type baseExecutor struct {
	finished bool
}

// setCommand
func newSetCommand(json gjson.Result) (Command, error) {
	var targetRef *Ref
	var err error
	if t := json.Get("target"); !t.Exists() {
		return nil, errors.New("setCommand: target field is required")
	} else {
		targetRef, err = NewRef(t)
		if err != nil {
			return nil, fmt.Errorf("setCommand: failed to create ref from target field: %v\n  %v", err, t.Raw)
		}
	}
	var value string
	if v := json.Get("value"); !v.Exists() {
		return nil, errors.New("setCommand: value field is required")
	} else {
		value = v.String()
		if value == "" {
			return nil, errors.New("setCommand: value cannot be empty")
		}
	}
	return &setCommand{
		targetRef: targetRef,
		value:     value,
	}, nil
}

type setCommand struct {
	targetRef *Ref
	value     string
}

func (c *setCommand) GetType() string {
	return commandTypeSet
}

func (c *setCommand) GetRefs() []*Ref {
	return []*Ref{c.targetRef}
}

func (c *setCommand) CreateExecutor() Executor {
	return &setExecutor{command: c}
}

type setExecutor struct {
	baseExecutor
	command *setCommand
}

func (e *setExecutor) GetCommand() Command {
	return e.command
}

func (e *setExecutor) Run(editorContext EditorContext, stage Stage) error {
	if e.finished {
		return nil
	}

	command := e.command
	log.Debugf("setCommand: checking stage %s for target %s", Stage2String[stage], command.targetRef)
	if command.targetRef.GetStage() == stage {
		log.Debugf("setCommand: set %s to %s", command.targetRef, command.value)
		editorContext.SetRefValue(command.targetRef, command.value)
		e.finished = true
	}

	return nil
}

// concatCommand
func newConcatCommand(json gjson.Result) (Command, error) {
	var targetRef *Ref
	var err error
	if t := json.Get("target"); !t.Exists() {
		return nil, errors.New("concatCommand: target field is required")
	} else {
		targetRef, err = NewRef(t)
		if err != nil {
			return nil, fmt.Errorf("concatCommand: failed to create ref from target field: %v\n  %v", err, t.Raw)
		}
	}

	valuesJson := json.Get("values")
	if !valuesJson.Exists() || !valuesJson.IsArray() {
		return nil, errors.New("concatCommand: values field is required and must be an array")
	}

	values := make([]interface{}, 0, len(valuesJson.Array()))
	for _, item := range valuesJson.Array() {
		var value interface{}
		if item.IsObject() {
			valueRef, err := NewRef(item)
			if err != nil {
				return nil, fmt.Errorf("concatCommand: failed to create ref from values field: %v\n  %v", err, item.Raw)
			}
			if valueRef.GetStage() > targetRef.GetStage() {
				return nil, fmt.Errorf("concatCommand: the processing stage of value [%s] cannot be after the stage of target [%s]", Stage2String[valueRef.GetStage()], Stage2String[targetRef.GetStage()])
			}
			value = valueRef
		} else {
			value = item.String()
		}
		values = append(values, value)
	}

	return &concatCommand{
		targetRef: targetRef,
		values:    values,
	}, nil
}

type concatCommand struct {
	targetRef *Ref
	values    []interface{}
}

func (c *concatCommand) GetType() string {
	return commandTypeConcat
}

func (c *concatCommand) GetRefs() []*Ref {
	refs := []*Ref{c.targetRef}
	if c.values != nil && len(c.values) != 0 {
		for _, value := range c.values {
			if ref, ok := value.(*Ref); ok {
				refs = append(refs, ref)
			}
		}
	}
	return refs
}

func (c *concatCommand) CreateExecutor() Executor {
	return &concatExecutor{command: c}
}

type concatExecutor struct {
	baseExecutor
	command *concatCommand
	values  []string
}

func (e *concatExecutor) GetCommand() Command {
	return e.command
}

func (e *concatExecutor) Run(editorContext EditorContext, stage Stage) error {
	if e.finished {
		return nil
	}

	command := e.command

	if e.values == nil {
		e.values = make([]string, len(command.values))
	}

	for i, value := range command.values {
		if value == nil || e.values[i] != "" {
			continue
		}
		v := ""
		if s, ok := value.(string); ok {
			v = s
		} else if ref, ok := value.(*Ref); ok && ref.GetStage() == stage {
			v = editorContext.GetRefValue(ref)
		}
		e.values[i] = v
	}

	if command.targetRef.GetStage() == stage {
		result := ""
		for _, v := range e.values {
			if v == "" {
				continue
			}
			result += v
		}
		log.Debugf("concatCommand: set %s to %s", command.targetRef, result)
		editorContext.SetRefValue(command.targetRef, result)
		e.finished = true
	}
	return nil
}

// copyCommand
func newCopyCommand(json gjson.Result) (Command, error) {
	var sourceRef *Ref
	var targetRef *Ref
	var err error
	if t := json.Get("source"); !t.Exists() {
		return nil, errors.New("copyCommand: source field is required")
	} else {
		sourceRef, err = NewRef(t)
		if err != nil {
			return nil, fmt.Errorf("copyCommand: failed to create ref from source field: %v\n  %v", err, t.Raw)
		}
	}
	if t := json.Get("target"); !t.Exists() {
		return nil, errors.New("copyCommand: target field is required")
	} else {
		targetRef, err = NewRef(t)
		if err != nil {
			return nil, fmt.Errorf("copyCommand: failed to create ref from target field: %v\n  %v", err, t.Raw)
		}
	}
	if sourceRef.GetStage() > targetRef.GetStage() {
		return nil, fmt.Errorf("copyCommand: the processing stage of source [%s] cannot be after the stage of target [%s]", Stage2String[sourceRef.GetStage()], Stage2String[targetRef.GetStage()])
	}
	return &copyCommand{
		sourceRef: sourceRef,
		targetRef: targetRef,
	}, nil
}

type copyCommand struct {
	sourceRef *Ref
	targetRef *Ref
}

func (c *copyCommand) GetType() string {
	return commandTypeCopy
}

func (c *copyCommand) GetRefs() []*Ref {
	return []*Ref{c.sourceRef, c.targetRef}
}

func (c *copyCommand) CreateExecutor() Executor {
	return &copyExecutor{command: c}
}

type copyExecutor struct {
	baseExecutor
	command     *copyCommand
	valueToCopy string
}

func (e *copyExecutor) GetCommand() Command {
	return e.command
}

func (e *copyExecutor) Run(editorContext EditorContext, stage Stage) error {
	if e.finished {
		return nil
	}

	command := e.command

	if command.sourceRef.GetStage() == stage {
		e.valueToCopy = editorContext.GetRefValue(command.sourceRef)
		log.Debugf("copyCommand: valueToCopy=%s", e.valueToCopy)
	}

	if e.valueToCopy == "" {
		log.Debug("copyCommand: valueToCopy is empty. skip.")
		e.finished = true
		return nil
	}

	if command.targetRef.GetStage() == stage {
		editorContext.SetRefValue(command.targetRef, e.valueToCopy)
		log.Debugf("copyCommand: set %s to %s", e.valueToCopy, command.targetRef)
		e.finished = true
	}

	return nil
}

// deleteCommand
func newDeleteCommand(json gjson.Result) (Command, error) {
	var targetRef *Ref
	var err error
	if t := json.Get("target"); !t.Exists() {
		return nil, errors.New("deleteCommand: target field is required")
	} else {
		targetRef, err = NewRef(t)
		if err != nil {
			return nil, fmt.Errorf("deleteCommand: failed to create ref from target field: %v\n  %v", err, t.Raw)
		}
	}
	return &deleteCommand{
		targetRef: targetRef,
	}, nil
}

type deleteCommand struct {
	targetRef *Ref
}

func (c *deleteCommand) GetType() string {
	return commandTypeDelete
}

func (c *deleteCommand) GetRefs() []*Ref {
	return []*Ref{c.targetRef}
}

func (c *deleteCommand) CreateExecutor() Executor {
	return &deleteExecutor{command: c}
}

type deleteExecutor struct {
	baseExecutor
	command *deleteCommand
}

func (e *deleteExecutor) GetCommand() Command {
	return e.command
}

func (e *deleteExecutor) Run(editorContext EditorContext, stage Stage) error {
	if e.finished {
		return nil
	}

	command := e.command
	log.Debugf("deleteCommand: checking stage %s for target %s", Stage2String[stage], command.targetRef)

	if command.targetRef.GetStage() == stage {
		log.Debugf("deleteCommand: delete %s", command.targetRef)
		editorContext.DeleteRefValues(command.targetRef)
		e.finished = true
		log.Debugf("deleteCommand: finished deleting %s", command.targetRef)
	} else {
		log.Debugf("deleteCommand: stage %s does not match targetRef stage %s, skipping.", Stage2String[stage], Stage2String[command.targetRef.GetStage()])
	}

	return nil
}

// renameCommand
func newRenameCommand(json gjson.Result) (Command, error) {
	var targetRef *Ref
	var err error
	if t := json.Get("target"); !t.Exists() {
		return nil, errors.New("renameCommand: target field is required")
	} else {
		targetRef, err = NewRef(t)
		if err != nil {
			return nil, fmt.Errorf("renameCommand: failed to create ref from target field: %v\n  %v", err, t.Raw)
		}
	}
	newName := json.Get("newName").String()
	if newName == "" {
		return nil, errors.New("renameCommand: newName field is required")
	}
	return &renameCommand{
		targetRef: targetRef,
		newName:   newName,
	}, nil
}

type renameCommand struct {
	targetRef *Ref
	newName   string
}

func (c *renameCommand) GetType() string {
	return commandTypeRename
}

func (c *renameCommand) GetRefs() []*Ref {
	return []*Ref{c.targetRef}
}

func (c *renameCommand) CreateExecutor() Executor {
	return &renameExecutor{command: c}
}

type renameExecutor struct {
	baseExecutor
	command *renameCommand
}

func (e *renameExecutor) GetCommand() Command {
	return e.command
}

func (e *renameExecutor) Run(editorContext EditorContext, stage Stage) error {
	if e.finished {
		return nil
	}

	command := e.command
	log.Debugf("renameCommand: checking stage %s for target %s", Stage2String[stage], command.targetRef)

	if command.targetRef.GetStage() == stage {
		if command.newName == command.targetRef.Name {
			log.Debugf("renameCommand: skip renaming %s to itself", command.targetRef)
		} else {
			values := editorContext.GetRefValues(command.targetRef)
			log.Debugf("renameCommand: rename %s to %s value=%v", command.targetRef, command.newName, values)
			editorContext.SetRefValues(&Ref{
				Type: command.targetRef.Type,
				Name: command.newName,
			}, values)
			editorContext.DeleteRefValues(command.targetRef)
			log.Debugf("renameCommand: finished renaming %s to %s", command.targetRef, command.newName)
		}
		e.finished = true
	} else {
		log.Debugf("renameCommand: stage %s does not match targetRef stage %s, skipping.", Stage2String[stage], Stage2String[command.targetRef.GetStage()])
	}

	return nil
}
