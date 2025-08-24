package main

import (
	"errors"
	"fmt"

	"github.com/tidwall/gjson"
)

const (
	commandTypeSet    = "set"
	commandTypeCopy   = "copy"
	commandTypeDelete = "delete"
	commandTypeRename = "rename"
)

var (
	commandFactories = map[string]func(gjson.Result) (Command, error){
		"set":    newSetCommand,
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

type ConditionSet struct {
	Conditions    []Condition    `json:"conditions,omitempty"`
	RelatedStages map[Stage]bool `json:"-"`
}

func (s *ConditionSet) FromJson(json gjson.Result) error {
	relatedStages := map[Stage]bool{}
	s.Conditions = nil
	if conditionsJson := json.Get("conditions"); conditionsJson.Exists() && conditionsJson.IsArray() {
		for _, item := range conditionsJson.Array() {
			if condition, err := CreateCondition(item); err != nil {
				return fmt.Errorf("failed to create condition from json: %v\n  %v", err, item)
			} else {
				s.Conditions = append(s.Conditions, condition)
				for _, ref := range condition.GetRefs() {
					relatedStages[ref.GetStage()] = true
				}
			}
		}
	}
	s.RelatedStages = relatedStages

	return nil
}

func (s *ConditionSet) Matches(editorContext *EditorContext) bool {
	if len(s.Conditions) == 0 {
		return true
	}
	for _, condition := range s.Conditions {
		if !condition.Evaluate(editorContext) {
			return false
		}
	}
	return true
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
	Run(editorContext *EditorContext, stage Stage) error
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

func (e *setExecutor) Run(editorContext *EditorContext, stage Stage) error {
	if e.finished {
		return nil
	}

	command := e.command
	if command.targetRef.GetStage() == stage {
		editorContext.SetRefValue(command.targetRef, command.value)
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
		return nil, fmt.Errorf("setCommand: the processing stage of source [%s] cannot be after the stage of target [%s]", Stage2String[sourceRef.GetStage()], Stage2String[targetRef.GetStage()])
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
	command    *copyCommand
	valueToSet string
}

func (e *copyExecutor) GetCommand() Command {
	return e.command
}

func (e *copyExecutor) Run(editorContext *EditorContext, stage Stage) error {
	if e.finished {
		return nil
	}

	command := e.command

	if command.sourceRef.GetStage() == stage {
		e.valueToSet = editorContext.GetRefValue(command.sourceRef)
	}

	if e.valueToSet == "" {
		return nil
	}

	if command.targetRef.GetStage() == stage {
		editorContext.SetRefValue(command.targetRef, e.valueToSet)
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

func (e *deleteExecutor) Run(editorContext *EditorContext, stage Stage) error {
	// TODO: 实现 delete 操作逻辑
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

func (e *renameExecutor) Run(editorContext *EditorContext, stage Stage) error {
	if e.finished {
		return nil
	}

	command := e.command

	if command.targetRef.GetStage() == stage && command.newName != command.targetRef.Name {
		values := editorContext.GetRefValues(command.targetRef)
		editorContext.SetRefValues(command.targetRef, values)
		editorContext.DeleteRefValues(command.targetRef)
		e.finished = true
	}

	return nil
}
