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
package clientManager

import (
	"objectweaver/llmManagement"
	"objectweaver/llmManagement/domain"

	"github.com/sashabaranov/go-openai"
)

type ClientAdapter interface {
	Process(inputs *llmManagement.Inputs) (*domain.JobResult, error)
	//will need some proper structure etc
	ProcessBatch(jobs []any) (*openai.ChatCompletionResponse, error)
}
