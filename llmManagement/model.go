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
// <https://objectweaver.dev/licensing/server-side-public-license>.
package llmManagement

import "github.com/objectweaver/go-sdk/jsonSchema"

type Inputs struct {
	Def          *jsonSchema.Definition
	Prompt       string
	SystemPrompt string
	OutStream    chan interface{}
	Index		int // The index of the item in the heap
	Priority   int32 // Higher value means higher priority // values under 0 or lower will be considered eventually ie will be processed in a batching system
}
