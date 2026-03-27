# Recipe: Simple Object Generation

## Use When

You need a single structured JSON object from a prompt.

## Steps

1. Define root `type: "object"`.
2. Add root `instruction` with scope.
3. Add `properties` with explicit field types and instructions.
4. Send to `POST /api/objectGen`.
5. Parse direct or wrapped response shape.

## Minimal Payload

```json
{
  "prompt": "Create a product card",
  "definition": {
    "type": "object",
    "instruction": "Generate a concise product card",
    "properties": {
      "name": { "type": "string", "instruction": "Product name" },
      "price": { "type": "number", "instruction": "Price in USD" },
      "summary": { "type": "string", "instruction": "Two-sentence value summary" }
    }
  }
}
```

## Agent Checklist

- Keep field names semantic.
- Keep instructions measurable (format, range, length).
- Add model overrides only when needed.
