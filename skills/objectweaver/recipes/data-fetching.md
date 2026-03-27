# Recipe: External Data Fetching (`req`)

## Use When

A field should be generated with external API/DB context.

## Steps

1. Add `req` to target field (`url`, `method`, `headers`, `body`).
2. Include auth headers.
3. Add `requireFields` for prerequisite generated fields.
4. Use `processingOrder` to ensure prerequisites exist.
5. Optionally combine with `narrowFocus.fields` for minimal prompt context.

## Skeleton

```json
{
  "type": "string",
  "instruction": "Generate final content using external context",
  "req": {
    "url": "https://example.internal/search",
    "method": "POST",
    "headers": { "Authorization": "Bearer TOKEN" },
    "body": { "projectId": "abc" },
    "requireFields": ["outline"]
  },
  "narrowFocus": {
    "fields": ["outline"],
    "prompt": "Use outline and fetched context"
  }
}
```
