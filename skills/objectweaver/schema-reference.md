# Skill: Schema Reference (High-Value Fields)

Use this as a fast schema checklist when constructing `definition` payloads.

## Core Fields

- `type`: `object | array | string | number | integer | boolean | null | map | vector | byte`
- `instruction`: generation guidance
- `properties`: object field map
- `items`: array item definition

## Control & Routing Fields

- `processingOrder`: dependency ordering for sibling fields
- `model`: per-field model override
- `modelConfig`: `temperature`, `topP`, `maxCompletionTokens`, etc.
- `priority`: `<0` batch, `>=0` real-time
- `systemPrompt`: role/behavior guidance (child overrides parent)
- `overridePrompt`: custom prompt override

## Context Management

- `selectFields`: copy selected prior outputs into current prompt context
  - supports nested dot paths, e.g. `product.specs.weight`
  - supports array extraction, e.g. `reviews.rating`
- `narrowFocus`: focused prompt + selected fields (`fields`, `prompt`, `keepOriginal`)

## Quality & Branching

- `scoringCriteria`
  - `dimensions` with `description`, `type`, optional range/weight
  - `aggregationMethod`
- `decisionPoint`
  - `strategy`: `score | field | hybrid`
  - ordered `branches` with condition operators: `eq`, `neq`, `gt`, `lt`, `gte`, `lte`, `in`, `nin`, `contains`
  - `default` fallback branch
- `recursiveLoop`
  - `maxIterations`, `selection`, optional termination rules

## External Integration

- `req`: external request data (`url`, `method`, `headers`, `body`, `authorization`, `requireFields`)

## Byte / Media

- `type: "byte"` required for media operations
- `image`: image generation
- `textToSpeech`: text-to-audio
- `speechToText`: audio transcription
- `sendImage`: multimodal image input

## Practical Constraints

- Streaming requests should keep priorities `>= 0`.
- Add `processingOrder` before referencing fields with `selectFields`.
- Always include a default/fallback branch in complex routing.
