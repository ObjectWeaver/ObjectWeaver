// Copyright (C) 2025-present ObjectWeaver.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the Server Side Public License, version 1,
// as published by ObjectWeaver.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// Server Side Public License for more details.
//
// You should have received a copy of the Server Side Public License
// along with this program. If not, see
// <https://github.com/ObjectWeaver/ObjectWeaver/blob/main/LICENSE.txt>.
package domain

import (
	"reflect"
	"testing"
)

func TestNewStreamChunk(t *testing.T) {
	key := "testKey"
	value := "testValue"
	chunk := NewStreamChunk(key, value)

	if chunk.Key != key {
		t.Errorf("Expected Key %s, got %s", key, chunk.Key)
	}
	if chunk.Value != value {
		t.Errorf("Expected Value %v, got %v", value, chunk.Value)
	}
	if chunk.IsFinal {
		t.Error("Expected IsFinal to be false")
	}
}

func TestStreamChunkWithAccumulatedData(t *testing.T) {
	chunk := NewStreamChunk("key", "value")
	data := map[string]interface{}{"field1": "value1"}
	chunk.WithAccumulatedData(data)

	if !reflect.DeepEqual(chunk.AccumulatedData, data) {
		t.Errorf("Expected AccumulatedData %v, got %v", data, chunk.AccumulatedData)
	}
}

func TestStreamChunkWithProgress(t *testing.T) {
	chunk := NewStreamChunk("key", "value")
	progress := 0.5
	chunk.WithProgress(progress)

	if chunk.Progress != progress {
		t.Errorf("Expected Progress %f, got %f", progress, chunk.Progress)
	}
}

func TestStreamChunkMarkFinal(t *testing.T) {
	chunk := NewStreamChunk("key", "value")
	chunk.MarkFinal()

	if !chunk.IsFinal {
		t.Error("Expected IsFinal to be true")
	}
}

func TestNewTokenStreamChunk(t *testing.T) {
	key := "testKey"
	token := "testToken"
	chunk := NewTokenStreamChunk(key, token)

	if chunk.Key != key {
		t.Errorf("Expected Key %s, got %s", key, chunk.Key)
	}
	if chunk.Token != token {
		t.Errorf("Expected Token %s, got %s", token, chunk.Token)
	}
	if chunk.Partial != token {
		t.Errorf("Expected Partial %s, got %s", token, chunk.Partial)
	}
	if chunk.Complete {
		t.Error("Expected Complete to be false")
	}
}

func TestTokenStreamChunkAppendToken(t *testing.T) {
	chunk := NewTokenStreamChunk("key", "first")
	chunk.AppendToken("second")

	expected := "firstsecond"
	if chunk.Partial != expected {
		t.Errorf("Expected Partial %s, got %s", expected, chunk.Partial)
	}
}

func TestTokenStreamChunkMarkComplete(t *testing.T) {
	chunk := NewTokenStreamChunk("key", "token")
	chunk.MarkComplete()

	if !chunk.Complete {
		t.Error("Expected Complete to be true")
	}
}

func TestNewProgressiveValue(t *testing.T) {
	key := "testKey"
	path := []string{"root", "field"}
	pv := NewProgressiveValue(key, path)

	if pv.Key() != key {
		t.Errorf("Expected Key %s, got %s", key, pv.Key())
	}
	if !reflect.DeepEqual(pv.Path(), path) {
		t.Errorf("Expected Path %v, got %v", path, pv.Path())
	}
	if pv.CurrentValue() != "" {
		t.Errorf("Expected CurrentValue to be empty, got %s", pv.CurrentValue())
	}
	if pv.IsComplete() {
		t.Error("Expected IsComplete to be false")
	}
	if len(pv.Tokens()) != 0 {
		t.Errorf("Expected Tokens to be empty, got %v", pv.Tokens())
	}
}

func TestProgressiveValueAppend(t *testing.T) {
	pv := NewProgressiveValue("key", []string{"path"})
	token1 := "hello"
	token2 := " world"

	pv.Append(token1)
	if pv.CurrentValue() != token1 {
		t.Errorf("Expected CurrentValue %s, got %s", token1, pv.CurrentValue())
	}
	if len(pv.Tokens()) != 1 || pv.Tokens()[0] != token1 {
		t.Errorf("Expected Tokens [%s], got %v", token1, pv.Tokens())
	}

	pv.Append(token2)
	expectedValue := "hello world"
	if pv.CurrentValue() != expectedValue {
		t.Errorf("Expected CurrentValue %s, got %s", expectedValue, pv.CurrentValue())
	}
	if len(pv.Tokens()) != 2 || pv.Tokens()[1] != token2 {
		t.Errorf("Expected Tokens [%s, %s], got %v", token1, token2, pv.Tokens())
	}
}

func TestProgressiveValueMarkComplete(t *testing.T) {
	pv := NewProgressiveValue("key", []string{"path"})
	pv.MarkComplete()

	if !pv.IsComplete() {
		t.Error("Expected IsComplete to be true")
	}
}

func TestNewTokenChunk(t *testing.T) {
	token := "testToken"
	chunk := NewTokenChunk(token)

	if chunk.Token != token {
		t.Errorf("Expected Token %s, got %s", token, chunk.Token)
	}
	if chunk.IsFinal {
		t.Error("Expected IsFinal to be false")
	}
	if chunk.Delta != "" {
		t.Errorf("Expected Delta to be empty, got %s", chunk.Delta)
	}
	if chunk.FinishReason != "" {
		t.Errorf("Expected FinishReason to be empty, got %s", chunk.FinishReason)
	}
}
