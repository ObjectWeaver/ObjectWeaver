# Recipe: Context Control (`processingOrder`, `selectFields`, `systemPrompt`, `narrowFocus`)

## Use When

Prompt context is too broad, noisy, or order-dependent.

## Workflow

1. Use `processingOrder` to guarantee producer fields run first.
2. Use `selectFields` to inject only required prior outputs.
3. Use `systemPrompt` for role/behavior controls.
4. Use `narrowFocus` when you need minimal context plus a targeted prompt.

## `selectFields` Patterns

- Simple: `"title"`
- Nested: `"product.specs.weight"`
- Array extraction: `"reviews.rating"`

## High-Impact Rule

If field B depends on field A, define both:

- generation order via `processingOrder`
- context passing via `selectFields`

Order alone does not inject context.
