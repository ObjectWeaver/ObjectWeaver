# Skill: ObjectWeaver Core

## What ObjectWeaver Is

ObjectWeaver is a schema-first orchestration layer for LLM generation. You provide:

- `prompt`
- `definition` (typed schema)

ObjectWeaver then:

1. Splits work by fields.
2. Routes fields to configured models.
3. Applies dependency order (`processingOrder`) where needed.
4. Validates generated types.
5. Returns assembled structured output.

## Core Endpoints

- `POST /api/objectGen` — generate structured object/list.
- `POST /api/textToWeaver` — generate schema from natural language (development mode).
- `GET /health` — health status.
- gRPC stream method: `StreamGeneratedObjects(RequestBody) -> stream StreamingResponse`.

## Core Mental Model for Agents

- Structure is primary: design schema first, prompts second.
- Isolate context: only pass required context via `selectFields` / `narrowFocus`.
- Separate concerns per field: routing, scoring, refinement can be field-local.
- Prefer explicit dependencies: use `processingOrder` whenever field B depends on field A.

## Default Request Shape

```json
{
  "prompt": "...",
  "stream": false,
  "definition": {
    "type": "object",
    "instruction": "...",
    "properties": {}
  }
}
```

## Response Parsing Guidance

Docs show more than one response shape:

- direct object payloads
- wrapped/protobuf-style fields (`data.fields...`)

Agent clients should parse both safely.
