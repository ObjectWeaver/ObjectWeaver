# Recipe: Text-to-Weaver

## Use When

You need a first draft schema from plain language.

## Constraints

`/api/textToWeaver` is documented as development-mode only.

## Steps

1. Send `{ "prompt": "..." }` to `POST /api/textToWeaver`.
2. Receive generated `definition`.
3. Review/adjust field names, types, and instructions.
4. Reuse definition in `POST /api/objectGen`.

## Example

```json
{
  "prompt": "Create a user profile with name, email, and nested address with street and city"
}
```

## Agent Rule

Never blindly trust generated schemas for production. Validate and tighten constraints first.
