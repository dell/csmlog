/*
 *
 * Copyright © 2025-2026 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 */
package csmlog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/metadata"
)

// ============================================================
// Core Logger Tests (Zap Backend)
// ============================================================

// U-ZAP-001
func TestZap_GetLogger_ReturnsNonNil(t *testing.T) {
	l := GetLogger()
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
}

// U-ZAP-002
func TestZap_GetLogger_Singleton(t *testing.T) {
	l1 := GetLogger()
	l2 := GetLogger()
	if l1 != l2 {
		t.Fatal("expected same instance from GetLogger()")
	}
}

// U-ZAP-003
func TestZap_Info_WritesToOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)
	log.SetLevel(TraceLevel)

	log.Info("hello info")
	if !strings.Contains(buf.String(), "hello info") {
		t.Errorf("expected output to contain 'hello info', got: %s", buf.String())
	}
}

// U-ZAP-004
func TestZap_Infof_FormatsMessage(t *testing.T) {
	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)
	log.SetLevel(TraceLevel)

	log.Infof("hello %s", "world")
	if !strings.Contains(buf.String(), "hello world") {
		t.Errorf("expected 'hello world' in output, got: %s", buf.String())
	}
}

// U-ZAP-005
func TestZap_Error_WritesToOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)
	log.SetLevel(TraceLevel)

	log.Error("error msg")
	out := buf.String()
	if !strings.Contains(out, "error msg") {
		t.Errorf("expected 'error msg' in output, got: %s", out)
	}
}

// U-ZAP-006
func TestZap_Warn_WritesToOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)
	log.SetLevel(TraceLevel)

	log.Warn("warn msg")
	if !strings.Contains(buf.String(), "warn msg") {
		t.Errorf("expected 'warn msg' in output, got: %s", buf.String())
	}
}

// U-ZAP-007
func TestZap_Debug_WritesToOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)
	log.SetLevel(DebugLevel)

	log.Debug("debug msg")
	if !strings.Contains(buf.String(), "debug msg") {
		t.Errorf("expected 'debug msg' in output, got: %s", buf.String())
	}
}

// U-ZAP-008
func TestZap_Trace_WritesToOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)
	log.SetLevel(TraceLevel)

	log.Trace("trace msg")
	if !strings.Contains(buf.String(), "trace msg") {
		t.Errorf("expected 'trace msg' in output, got: %s", buf.String())
	}
}

// U-ZAP-009
func TestZap_Panic_Panics(t *testing.T) {
	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)
	log.SetLevel(TraceLevel)

	recovered := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				recovered = true
			}
		}()
		log.Panic("panic msg")
	}()
	if !recovered {
		t.Error("expected Panic to cause a panic")
	}
	if !strings.Contains(buf.String(), "panic msg") {
		t.Errorf("expected 'panic msg' in output, got: %s", buf.String())
	}
}

// U-ZAP-010
func TestZap_WithFields_AddsFields(t *testing.T) {
	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)
	log.SetLevel(TraceLevel)
	log.SetFormat("json")

	log.WithFields(Fields{"volume_id": "vol-abc", "array_id": "PS001"}).Info("with fields")
	out := buf.String()
	if !strings.Contains(out, "vol-abc") {
		t.Errorf("expected 'vol-abc' in output, got: %s", out)
	}
	if !strings.Contains(out, "PS001") {
		t.Errorf("expected 'PS001' in output, got: %s", out)
	}
}

// U-ZAP-011
func TestZap_WithContext_ExtractsRequestID(t *testing.T) {
	md := metadata.Pairs(RequestIDKey, "req-999")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)
	log.SetLevel(TraceLevel)

	log.WithContext(ctx).WithFields(Fields{"testField": "testValue"}).Info("with context")
	out := buf.String()
	if !strings.Contains(out, "req-999") {
		t.Errorf("expected 'req-999' in output, got: %s", out)
	}
}

// U-ZAP-012
func TestZap_SetLevel_ChangesLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)

	log.SetLevel(ErrorLevel)
	log.Info("should not appear")
	if strings.Contains(buf.String(), "should not appear") {
		t.Error("Info message appeared at ErrorLevel")
	}

	log.Error("should appear")
	if !strings.Contains(buf.String(), "should appear") {
		t.Errorf("Error message did not appear, got: %s", buf.String())
	}
}

// U-ZAP-013
func TestZap_GetLevel_ReturnsCurrentLevel(t *testing.T) {
	SetLevel(DebugLevel)
	if GetLevel() != DebugLevel {
		t.Errorf("expected DebugLevel, got %v", GetLevel())
	}
	SetLevel(ErrorLevel)
	if GetLevel() != ErrorLevel {
		t.Errorf("expected ErrorLevel, got %v", GetLevel())
	}
}

// U-ZAP-014
func TestZap_ParseLevel_ValidLevels(t *testing.T) {
	tests := []struct {
		input string
		want  Level
	}{
		{"info", InfoLevel},
		{"debug", DebugLevel},
		{"error", ErrorLevel},
		{"warn", WarnLevel},
		{"trace", TraceLevel},
		{"fatal", FatalLevel},
		{"panic", PanicLevel},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseLevel(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// U-ZAP-015
func TestZap_ParseLevel_InvalidLevel(t *testing.T) {
	_, err := ParseLevel("invalidlevel")
	if err == nil {
		t.Error("expected error for invalid level, got nil")
	}
}

// U-ZAP-016
func TestZap_SetOutput_RedirectsOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)
	log.SetLevel(InfoLevel)

	log.Info("redirected")
	if !strings.Contains(buf.String(), "redirected") {
		t.Errorf("output not redirected, got: %s", buf.String())
	}
}

// U-ZAP-017
func TestZap_SetFormat_JSON(t *testing.T) {
	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)
	log.SetLevel(InfoLevel)
	log.SetFormat("json")

	log.WithFields(Fields{"key": "val"}).Info("json test")
	var m map[string]interface{}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, buf.String())
	}
	if m["msg"] == nil && m["message"] == nil {
		t.Errorf("expected 'msg' or 'message' key in JSON, got: %v", m)
	}
}

// U-ZAP-018
func TestZap_SetFormat_Text(t *testing.T) {
	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)
	log.SetLevel(InfoLevel)
	log.SetFormat("text")

	log.WithFields(Fields{"volume_id": "vol-abc", "array_id": "PS001", "arbitrary_info": "testdata"}).Info("with fields")
	out := buf.String()
	if !strings.Contains(out, "[INFO]") {
		t.Errorf("expected [INFO] in text output, got: %s", out)
	}
}

// U-ZAP-019
func TestZap_LevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{InfoLevel, "info"},
		{ErrorLevel, "error"},
		{WarnLevel, "warn"},
		{DebugLevel, "debug"},
		{TraceLevel, "trace"},
		{FatalLevel, "fatal"},
		{PanicLevel, "panic"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}

// U-ZAP-020
func TestZap_CloneFields(t *testing.T) {
	orig := Fields{"a": "b"}
	cloned := cloneFields(orig)
	cloned["a"] = "c"
	if orig["a"] != "b" {
		t.Fatalf("expected original to remain 'b', got %v", orig["a"])
	}
}

// U-ZAP-021
func TestZap_PackageLevelConvenienceFunctions(t *testing.T) {
	buf := &bytes.Buffer{}
	SetOutput(buf)
	SetLevel(TraceLevel)
	SetFormat("text")

	Info("package info")
	Warn("package warn")
	Error("package error")
	Debug("package debug")
	Trace("package trace")

	Infof("package %s", "infof")
	Warnf("package %s", "warnf")
	Errorf("package %s", "errorf")
	Debugf("package %s", "debugf")
	Tracef("package %s", "tracef")

	out := buf.String()
	if !strings.Contains(out, "package info") {
		t.Error("expected 'package info' in output")
	}
	if !strings.Contains(out, "package warn") {
		t.Error("expected 'package warn' in output")
	}
	if !strings.Contains(out, "package error") {
		t.Error("expected 'package error' in output")
	}
	if !strings.Contains(out, "package debug") {
		t.Error("expected 'package debug' in output")
	}
	if !strings.Contains(out, "package trace") {
		t.Error("expected 'package trace' in output")
	}
	if !strings.Contains(out, "package infof") {
		t.Error("expected 'package infof' in output")
	}
	if !strings.Contains(out, "package warnf") {
		t.Error("expected 'package warnf' in output")
	}
	if !strings.Contains(out, "package errorf") {
		t.Error("expected 'package errorf' in output")
	}
	if !strings.Contains(out, "package debugf") {
		t.Error("expected 'package debugf' in output")
	}
	if !strings.Contains(out, "package tracef") {
		t.Error("expected 'package tracef' in output")
	}
}

// U-ZAP-022
func TestZap_ParseLevel_ErrorCase(t *testing.T) {
	_, err := parseZapLevel("invalid")
	if err == nil {
		t.Error("expected error for invalid level")
	}
}

// U-ZAP-023
func TestZap_SetFormat_PackageLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	SetOutput(buf)
	SetFormat("json")
	Info("test")

	var m map[string]interface{}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) > 0 {
		last := lines[len(lines)-1]
		if err := json.Unmarshal([]byte(last), &m); err != nil {
			t.Errorf("expected JSON output after SetFormat, got: %s", buf.String())
		}
	}
}

// U-ZAP-024
func TestZap_InstanceFormattedMethods(t *testing.T) {
	buf := &bytes.Buffer{}
	log := New("text", TraceLevel, buf)
	log.SetOutput(buf)

	log.Warnf("formatted %s", "warn")
	log.Errorf("formatted %s", "error")
	log.Debugf("formatted %s", "debug")
	log.Tracef("formatted %s", "trace")

	// Test Panicf separately since it panics
	recovered := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				recovered = true
			}
		}()
		log.Panicf("formatted %s", "panic")
	}()
	if !recovered {
		t.Error("expected Panicf to panic")
	}

	log.Debugf("formatted %s", "debugf")

	out := buf.String()
	if !strings.Contains(out, "formatted warn") {
		t.Error("expected 'formatted warn' in output")
	}
	if !strings.Contains(out, "formatted error") {
		t.Error("expected 'formatted error' in output")
	}
	if !strings.Contains(out, "formatted debug") {
		t.Error("expected 'formatted debug' in output")
	}
	if !strings.Contains(out, "formatted trace") {
		t.Error("expected 'formatted trace' in output")
	}
	if !strings.Contains(out, "formatted panic") {
		t.Error("expected 'formatted panic' in output")
	}
}

// U-ZAP-025
func TestZap_New_WithNilOutput(t *testing.T) {
	log := New("text", InfoLevel, nil)
	if log == nil {
		t.Fatal("expected non-nil logger")
	}
	log.Info("test with nil output")
}

// U-ZAP-027
func TestZap_New_WithEmptyFormat(t *testing.T) {
	log := New("", InfoLevel, nil)
	if log == nil {
		t.Fatal("expected non-nil logger")
	}
	log.Info("test with empty format")
}

// U-ZAP-028
func TestZap_csmLevelToZap_AllLevels(t *testing.T) {
	tests := []struct {
		level Level
		want  zapcore.Level
	}{
		{PanicLevel, zapcore.DPanicLevel},
		{FatalLevel, zapcore.FatalLevel},
		{ErrorLevel, zapcore.ErrorLevel},
		{WarnLevel, zapcore.WarnLevel},
		{InfoLevel, zapcore.InfoLevel},
		{DebugLevel, zapcore.DebugLevel},
		{TraceLevel, zapcore.DebugLevel},
	}
	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if got := csmLevelToZap(tt.level); got != tt.want {
				t.Errorf("csmLevelToZap(%v) = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}

// U-ZAP-029
func TestZap_zapLevelToCsm_AllLevels(t *testing.T) {
	tests := []struct {
		level zapcore.Level
		want  Level
	}{
		{zapcore.DPanicLevel, PanicLevel},
		{zapcore.FatalLevel, FatalLevel},
		{zapcore.ErrorLevel, ErrorLevel},
		{zapcore.WarnLevel, WarnLevel},
		{zapcore.InfoLevel, InfoLevel},
		{zapcore.DebugLevel, DebugLevel},
	}
	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if got := zapLevelToCsm(tt.level); got != tt.want {
				t.Errorf("zapLevelToCsm(%v) = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}

// U-ZAP-030
func TestZap_PackageLevelPanic(t *testing.T) {
	buf := &bytes.Buffer{}
	SetOutput(buf)
	SetLevel(TraceLevel)

	recovered := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				recovered = true
			}
		}()
		Panic("package panic")
	}()
	if !recovered {
		t.Error("expected Panic to cause a panic")
	}
	if !strings.Contains(buf.String(), "package panic") {
		t.Errorf("expected 'package panic' in output, got: %s", buf.String())
	}
}

// U-ZAP-031
func TestZap_PackageLevelPanicf(t *testing.T) {
	buf := &bytes.Buffer{}
	SetOutput(buf)
	SetLevel(TraceLevel)

	recovered := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				recovered = true
			}
		}()
		Panicf("package %s", "panicf")
	}()
	if !recovered {
		t.Error("expected Panicf to cause a panic")
	}
	if !strings.Contains(buf.String(), "package panicf") {
		t.Errorf("expected 'package panicf' in output, got: %s", buf.String())
	}
}

// U-ZAP-032
func TestZap_LevelString_Unknown(t *testing.T) {
	// Test the default case in String() for unknown levels
	unknownLevel := Level(999)
	str := unknownLevel.String()
	if !strings.Contains(str, "unknown level") {
		t.Errorf("expected 'unknown level' in string, got: %s", str)
	}
}

// U-ZAP-033
func TestZap_csmLevelToZap_Default(t *testing.T) {
	// Test the default case in csmLevelToZap
	unknownLevel := Level(999)
	got := csmLevelToZap(unknownLevel)
	if got != zapcore.InfoLevel {
		t.Errorf("csmLevelToZap(999) = %v, want InfoLevel", got)
	}
}

// U-ZAP-034
func TestZap_zapLevelToCsm_Default(t *testing.T) {
	// Test the default case in zapLevelToCsm
	got := zapLevelToCsm(zapcore.Level(127))
	if got != InfoLevel {
		t.Errorf("zapLevelToCsm(127) = %v, want InfoLevel", got)
	}
}

// U-ZAP-035
func TestZap_csmLevelBracket_Default(t *testing.T) {
	// Test the default case in csmLevelBracket
	got := csmLevelBracket(zapcore.Level(127))
	if !strings.Contains(got, "[") || !strings.Contains(got, "]") {
		t.Errorf("csmLevelBracket(127) should contain brackets, got: %s", got)
	}
}

// U-ZAP-036
func TestZap_GetLogger_SingletonPath(t *testing.T) {
	// Test the path where globalLogInstance already exists
	// This is already tested in TestZap_GetLogger_Singleton, but let's verify
	// the init() function already created it
	l := GetLogger()
	if l == nil {
		t.Fatal("expected non-nil logger from GetLogger()")
	}
	l2 := GetLogger()
	if l != l2 {
		t.Fatal("expected same instance on second call")
	}
}

// U-ZAP-037
func TestZap_rebuildLogger_WithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	log := New("json", InfoLevel, buf)
	log.SetOutput(buf)

	// Add some fields
	log = log.WithFields(Fields{"test": "value"})
	log.Info("test")

	// Change format to trigger rebuild
	log.SetFormat("text")
	log.Info("test2")

	out := buf.String()
	if !strings.Contains(out, "test") {
		t.Error("expected 'test' in output")
	}
}

// U-ZAP-038
func TestZap_EncodeEntry_Fallback(t *testing.T) {
	// Test the JSON parse fallback in EncodeEntry
	// This is hard to test directly since it's internal to the encoder,
	// but we can verify text encoding works with fields
	buf := &bytes.Buffer{}
	log := New("text", InfoLevel, buf)
	log.WithFields(Fields{"key": "value"}).Info("test")

	out := buf.String()
	if !strings.Contains(out, "[key=value]") {
		t.Errorf("expected '[key=value]' in text output, got: %s", out)
	}
}

// U-ZAP-039
func TestZap_parseZapLevel_AllValidLevels(t *testing.T) {
	// Test all valid level strings parse correctly
	tests := []string{"info", "debug", "error", "warn", "trace", "fatal", "panic", "INFO", "DEBUG", "ERROR", "WARN", "TRACE", "FATAL", "PANIC"}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			level, err := parseZapLevel(tt)
			if err != nil {
				t.Errorf("parseZapLevel(%q) returned error: %v", tt, err)
			}
			if level == zapcore.InfoLevel && tt != "info" && tt != "INFO" {
				// This is the default when parsing fails, but we expect success
				t.Errorf("parseZapLevel(%q) returned InfoLevel for non-info input", tt)
			}
		})
	}
}

// U-ZAP-040
func TestZap_PackageLevelFatal(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		buf := &bytes.Buffer{}
		SetOutput(buf)
		SetLevel(TraceLevel)
		Fatal("package fatal")
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to get executable: %v", err)
	}
	exe = filepath.Clean(exe)
	cmd := exec.Command(exe, "-test.run=TestZap_PackageLevelFatal")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err = cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("expected exit status 1, got %v", err)
}

// U-ZAP-041
func TestZap_PackageLevelFatalf(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		buf := &bytes.Buffer{}
		SetOutput(buf)
		SetLevel(TraceLevel)
		Fatalf("package %s", "fatalf")
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to get executable: %v", err)
	}
	exe = filepath.Clean(exe)
	cmd := exec.Command(exe, "-test.run=TestZap_PackageLevelFatalf")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err = cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("expected exit status 1, got %v", err)
}

// ============================================================
// New Helpers & Constants
// ============================================================

// U-ZAP-030
func TestZap_WithOperation(t *testing.T) {
	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)
	log.SetLevel(InfoLevel)
	log.SetFormat("json")

	log.WithOperation("CreateVolume").Info("op test")
	if !strings.Contains(buf.String(), "CreateVolume") {
		t.Errorf("expected 'CreateVolume' in output, got: %s", buf.String())
	}
}

// U-ZAP-031
func TestZap_WithComponent(t *testing.T) {
	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)
	log.SetLevel(InfoLevel)
	log.SetFormat("json")

	log.WithComponent("csi-powerstore").Info("comp test")
	if !strings.Contains(buf.String(), "csi-powerstore") {
		t.Errorf("expected 'csi-powerstore' in output, got: %s", buf.String())
	}
}

// U-ZAP-032
func TestZap_TrackDuration(t *testing.T) {
	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)
	log.SetLevel(InfoLevel)
	log.SetFormat("json")

	start := time.Now().Add(-100 * time.Millisecond)
	log.TrackDuration(start).Info("dur test")
	if !strings.Contains(buf.String(), FieldDurationMs) {
		t.Errorf("expected '%s' in output, got: %s", FieldDurationMs, buf.String())
	}
}

// U-ZAP-033
func TestZap_FieldConstants_NotEmpty(t *testing.T) {
	constants := []string{
		FieldComponent, FieldOperation, FieldProtocol, FieldArrayID,
		FieldVolumeID, FieldVolumeName, FieldNodeID, FieldDurationMs,
		FieldError, FieldStagingPath, FieldTargetPath, FieldDevicePath,
		EnvLogFormat, EnvLogLevel,
	}
	for _, c := range constants {
		if c == "" {
			t.Errorf("expected non-empty constant")
		}
	}
}

// ============================================================
// Context Propagation
// ============================================================

// U-ZAP-040
func TestZap_ExtractFieldsFromContext_RequestID(t *testing.T) {
	md := metadata.Pairs(RequestIDKey, "456")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	fields := ExtractFieldsFromContext(ctx)
	if fields[RequestIDKey] != "456" {
		t.Errorf("expected request ID '456', got %v", fields[RequestIDKey])
	}
}

// U-ZAP-042
func TestZap_ExtractFieldsFromContext_NilContext(t *testing.T) {
	fields := ExtractFieldsFromContext(context.TODO())
	if len(fields) != 0 {
		t.Errorf("expected empty fields for nil context, got %v", fields)
	}
}

// U-ZAP-043
func TestZap_ExtractFieldsFromContext_NoMutation(t *testing.T) {
	md := metadata.Pairs(RequestIDKey, "789")
	ctxWithMD := metadata.NewIncomingContext(context.Background(), md)

	wg := sync.WaitGroup{}
	errCh := make(chan error, 50)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got := ExtractFieldsFromContext(ctxWithMD)
			if got[RequestIDKey] != "789" {
				errCh <- fmt.Errorf("expected request ID 789, got %v", got[RequestIDKey])
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatal(err)
	}
}

// ============================================================
// Integration Tests
// ============================================================

// I-ZAP-001
func TestZap_JSONOutput_ValidJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)
	log.SetLevel(InfoLevel)
	log.SetFormat("json")

	log.WithFields(Fields{
		FieldComponent: "test-comp",
		FieldOperation: "TestOp",
		FieldProtocol:  "nfs",
	}).Info("json integration")

	var m map[string]interface{}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	last := lines[len(lines)-1]
	if err := json.Unmarshal([]byte(last), &m); err != nil {
		t.Fatalf("not valid JSON: %v, output: %s", err, buf.String())
	}
	for _, key := range []string{FieldComponent, FieldOperation, FieldProtocol} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing key %q in JSON: %v", key, m)
		}
	}
}

// I-ZAP-002
func TestZap_TextOutput_BracketFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	log := GetLogger()
	log.SetOutput(buf)
	log.SetLevel(InfoLevel)
	log.SetFormat("text")

	log.Info("bracket test")
	out := buf.String()
	if !strings.Contains(out, "[INFO]") {
		t.Errorf("expected [INFO] in output, got: %s", out)
	}
}

// I-ZAP-003
func TestZap_EnvLogFormat_JSON(t *testing.T) {
	buf := &bytes.Buffer{}
	log := New("json", InfoLevel, buf)

	log.Info("env json test")
	var m map[string]interface{}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	last := lines[len(lines)-1]
	if err := json.Unmarshal([]byte(last), &m); err != nil {
		t.Fatalf("JSON format did not produce valid JSON: %v, output: %s", err, buf.String())
	}
}

// I-ZAP-004
func TestZap_EnvLogLevel_Debug(t *testing.T) {
	log := New("text", DebugLevel, nil)
	if log.GetLevel() != DebugLevel {
		t.Errorf("expected DebugLevel, got %v", log.GetLevel())
	}
}

// ============================================================
// Fatal tests (subprocess strategy)
// ============================================================

func zapFatalScenario() {
	buf := &bytes.Buffer{}
	log := GetLogger()
	SetOutput(buf)
	SetLevel(TraceLevel)
	log.Fatal("fatal msg")
}

func TestZap_Fatal_Exits(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		zapFatalScenario()
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to get executable: %v", err)
	}
	exe = filepath.Clean(exe)
	cmd := exec.Command(exe, "-test.run=TestZap_Fatal_Exits") // #nosec G702
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err = cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("expected exit status 1, got %v", err)
}

func zapFatalfScenario() {
	buf := &bytes.Buffer{}
	log := GetLogger()
	SetOutput(buf)
	SetLevel(TraceLevel)
	log.Fatalf("fatal %s", "msg")
}

func TestZap_Fatalf_Exits(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		zapFatalfScenario()
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to get executable: %v", err)
	}
	exe = filepath.Clean(exe)
	cmd := exec.Command(exe, "-test.run=TestZap_Fatalf_Exits") // #nosec G702
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err = cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("expected exit status 1, got %v", err)
}
