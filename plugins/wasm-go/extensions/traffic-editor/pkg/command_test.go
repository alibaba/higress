package pkg

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestNewSetCommand_Success(t *testing.T) {
	jsonStr := `{"type":"set","target":{"type":"request_header","name":"foo"},"value":"bar"}`
	json := gjson.Parse(jsonStr)
	cmd, err := newSetCommand(json)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cmd.GetType() != "set" {
		t.Errorf("expected type 'set', got %s", cmd.GetType())
	}
	refs := cmd.GetRefs()
	if len(refs) != 1 {
		t.Errorf("expected 1 ref, got %d", len(refs))
	}
}

func TestNewSetCommand_MissingTarget(t *testing.T) {
	jsonStr := `{"type":"set","value":"bar"}`
	json := gjson.Parse(jsonStr)
	_, err := newSetCommand(json)
	if err == nil || err.Error() != "setCommand: target field is required" {
		t.Errorf("expected target field error, got %v", err)
	}
}

func TestNewSetCommand_MissingValue(t *testing.T) {
	jsonStr := `{"type":"set","target":{"type":"request_header","name":"foo"}}`
	json := gjson.Parse(jsonStr)
	_, err := newSetCommand(json)
	if err == nil || err.Error() != "setCommand: value field is required" {
		t.Errorf("expected value field error, got %v", err)
	}
}

func TestNewConcatCommand_Success(t *testing.T) {
	jsonStr := `{"type":"concat","target":{"type":"request_header","name":"foo"},"values":["a","b"]}`
	json := gjson.Parse(jsonStr)
	cmd, err := newConcatCommand(json)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cmd.GetType() != "concat" {
		t.Errorf("expected type 'concat', got %s", cmd.GetType())
	}
	refs := cmd.GetRefs()
	if len(refs) < 1 {
		t.Errorf("expected at least 1 ref, got %d", len(refs))
	}
}

func TestNewConcatCommand_MissingTarget(t *testing.T) {
	jsonStr := `{"type":"concat","values":["a","b"]}`
	json := gjson.Parse(jsonStr)
	_, err := newConcatCommand(json)
	if err == nil || err.Error() != "concatCommand: target field is required" {
		t.Errorf("expected target field error, got %v", err)
	}
}

func TestNewConcatCommand_MissingValues(t *testing.T) {
	jsonStr := `{"type":"concat","target":{"type":"request_header","name":"foo"}}`
	json := gjson.Parse(jsonStr)
	_, err := newConcatCommand(json)
	if err == nil || err.Error() != "concatCommand: values field is required and must be an array" {
		t.Errorf("expected values field error, got %v", err)
	}
}

func TestNewCopyCommand_Success(t *testing.T) {
	jsonStr := `{"type":"copy","source":{"type":"request_header","name":"foo"},"target":{"type":"request_header","name":"bar"}}`
	json := gjson.Parse(jsonStr)
	cmd, err := newCopyCommand(json)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cmd.GetType() != "copy" {
		t.Errorf("expected type 'copy', got %s", cmd.GetType())
	}
	refs := cmd.GetRefs()
	if len(refs) != 2 {
		t.Errorf("expected 2 refs, got %d", len(refs))
	}
}

func TestNewCopyCommand_MissingSource(t *testing.T) {
	jsonStr := `{"type":"copy","target":{"type":"request_header","name":"bar"}}`
	json := gjson.Parse(jsonStr)
	_, err := newCopyCommand(json)
	if err == nil || err.Error() != "copyCommand: source field is required" {
		t.Errorf("expected source field error, got %v", err)
	}
}

func TestNewCopyCommand_MissingTarget(t *testing.T) {
	jsonStr := `{"type":"copy","source":{"type":"request_header","name":"foo"}}`
	json := gjson.Parse(jsonStr)
	_, err := newCopyCommand(json)
	if err == nil || err.Error() != "copyCommand: target field is required" {
		t.Errorf("expected target field error, got %v", err)
	}
}

func TestNewCopyCommand_SourceStageAfterTarget(t *testing.T) {
	jsonStr := `{"type":"copy","source":{"type":"response_header","name":"foo"},"target":{"type":"request_header","name":"bar"}}`
	json := gjson.Parse(jsonStr)
	_, err := newCopyCommand(json)
	if err == nil || err.Error() != "copyCommand: the processing stage of source [response_headers] cannot be after the stage of target [request_headers]" {
		t.Errorf("expected source stage field error, got %v", err)
	}
}

func TestNewDeleteCommand_Success(t *testing.T) {
	jsonStr := `{"type":"delete","target":{"type":"request_header","name":"foo"}}`
	json := gjson.Parse(jsonStr)
	cmd, err := newDeleteCommand(json)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cmd.GetType() != "delete" {
		t.Errorf("expected type 'delete', got %s", cmd.GetType())
	}
	refs := cmd.GetRefs()
	if len(refs) != 1 {
		t.Errorf("expected 1 ref, got %d", len(refs))
	}
}

func TestNewDeleteCommand_MissingTarget(t *testing.T) {
	jsonStr := `{"type":"delete"}`
	json := gjson.Parse(jsonStr)
	_, err := newDeleteCommand(json)
	if err == nil || err.Error() != "deleteCommand: target field is required" {
		t.Errorf("expected target field error, got %v", err)
	}
}

func TestNewRenameCommand_Success(t *testing.T) {
	jsonStr := `{"type":"rename","target":{"type":"request_header","name":"foo"},"newName":"bar"}`
	json := gjson.Parse(jsonStr)
	cmd, err := newRenameCommand(json)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cmd.GetType() != "rename" {
		t.Errorf("expected type 'rename', got %s", cmd.GetType())
	}
	refs := cmd.GetRefs()
	if len(refs) != 1 {
		t.Errorf("expected 1 ref, got %d", len(refs))
	}
}

func TestNewRenameCommand_MissingTarget(t *testing.T) {
	jsonStr := `{"type":"rename","newName":"bar"}`
	json := gjson.Parse(jsonStr)
	_, err := newRenameCommand(json)
	if err == nil || err.Error() != "renameCommand: target field is required" {
		t.Errorf("expected target field error, got %v", err)
	}
}

func TestNewRenameCommand_MissingNewName(t *testing.T) {
	jsonStr := `{"type":"rename","target":{"type":"request_header","name":"foo"}}`
	json := gjson.Parse(jsonStr)
	_, err := newRenameCommand(json)
	if err == nil || err.Error() != "renameCommand: newName field is required" {
		t.Errorf("expected newName field error, got %v", err)
	}
}

func TestSetExecutor_Run_SingleStage(t *testing.T) {
	ref := &Ref{Type: RefTypeRequestHeader, Name: "foo"}
	cmd := &setCommand{targetRef: ref, value: "bar"}
	executor := cmd.CreateExecutor()
	ctx := NewEditorContext()
	stage := StageRequestHeaders

	err := executor.Run(ctx, stage)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ctx.GetRefValue(ref) != "bar" {
		t.Errorf("expected value 'bar', got %s", ctx.GetRefValue(ref))
	}
}

func TestConcatExecutor_Run_SingleStage(t *testing.T) {
	ref := &Ref{Type: RefTypeRequestHeader, Name: "foo"}
	srcRef := &Ref{Type: RefTypeRequestHeader, Name: "test"}
	cmd := &concatCommand{targetRef: ref, values: []interface{}{"a", srcRef, "b"}}
	executor := cmd.CreateExecutor()
	ctx := NewEditorContext()
	ctx.SetRefValue(srcRef, "-")
	stage := StageRequestHeaders

	err := executor.Run(ctx, stage)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ctx.GetRefValue(ref) != "a-b" {
		t.Errorf("expected value 'a-b', got %s", ctx.GetRefValue(ref))
	}
}

func TestConcatExecutor_Run_MultiStages(t *testing.T) {
	ref := &Ref{Type: RefTypeResponseHeader, Name: "foo"}
	srcRef := &Ref{Type: RefTypeRequestHeader, Name: "test"}
	cmd := &concatCommand{targetRef: ref, values: []interface{}{"a", srcRef, "b"}}
	executor := cmd.CreateExecutor()
	ctx := NewEditorContext()
	ctx.SetRefValue(srcRef, "-")

	err := executor.Run(ctx, StageRequestHeaders)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	err = executor.Run(ctx, StageResponseHeaders)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ctx.GetRefValue(ref) != "a-b" {
		t.Errorf("expected value 'a-b', got %s", ctx.GetRefValue(ref))
	}
}

func TestCopyExecutor_Run_SingleStage(t *testing.T) {
	source := &Ref{Type: RefTypeRequestHeader, Name: "foo"}
	target := &Ref{Type: RefTypeRequestHeader, Name: "bar"}
	ctx := NewEditorContext()
	ctx.SetRefValue(source, "baz")
	cmd := &copyCommand{sourceRef: source, targetRef: target}
	executor := cmd.CreateExecutor()
	stage := StageRequestHeaders

	err := executor.Run(ctx, stage)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ctx.GetRefValue(target) != "baz" {
		t.Errorf("expected value 'baz' for target, got %s", ctx.GetRefValue(target))
	}
}

func TestCopyExecutor_Run_MultiStages(t *testing.T) {
	source := &Ref{Type: RefTypeRequestHeader, Name: "foo"}
	target := &Ref{Type: RefTypeResponseHeader, Name: "bar"}
	ctx := NewEditorContext()
	ctx.SetRefValue(source, "baz")
	cmd := &copyCommand{sourceRef: source, targetRef: target}
	executor := cmd.CreateExecutor()

	err := executor.Run(ctx, StageRequestHeaders)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	err = executor.Run(ctx, StageResponseHeaders)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ctx.GetRefValue(target) != "baz" {
		t.Errorf("expected value 'baz' for target, got %s", ctx.GetRefValue(target))
	}
}

func TestDeleteExecutor_Run(t *testing.T) {
	ref := &Ref{Type: RefTypeRequestHeader, Name: "foo"}
	ctx := NewEditorContext()
	ctx.SetRefValue(ref, "bar")
	cmd := &deleteCommand{targetRef: ref}
	executor := cmd.CreateExecutor()
	stage := StageRequestHeaders

	err := executor.Run(ctx, stage)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ctx.GetRefValue(ref) != "" {
		t.Errorf("expected value to be deleted, got %s", ctx.GetRefValue(ref))
	}
}

func TestRenameExecutor_Run(t *testing.T) {
	ref := &Ref{Type: RefTypeRequestHeader, Name: "foo"}
	ctx := NewEditorContext()
	ctx.SetRefValue(ref, "bar")
	cmd := &renameCommand{targetRef: ref, newName: "baz"}
	executor := cmd.CreateExecutor()
	stage := StageRequestHeaders

	err := executor.Run(ctx, stage)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	newRef := &Ref{Type: ref.Type, Name: "baz"}
	if ctx.GetRefValue(newRef) != "bar" {
		t.Errorf("expected value 'bar' for new name, got %s", ctx.GetRefValue(newRef))
	}
	if ctx.GetRefValue(ref) != "" {
		t.Errorf("expected old name to be deleted, got %s", ctx.GetRefValue(ref))
	}
}
