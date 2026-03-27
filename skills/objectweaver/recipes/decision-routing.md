# Recipe: Decision Routing (Quality / Field / Hybrid)

## Use When

A generated result must route to different follow-up logic.

## Build Pattern

1. Add `scoringCriteria.dimensions` (if score-based).
2. Add `decisionPoint.strategy`:
   - `score`
   - `field`
   - `hybrid`
3. Define ordered `branches` with `conditions`.
4. Add `then` definition for each branch.
5. Add `default` definition.

## Condition Operators

`eq`, `neq`, `gt`, `lt`, `gte`, `lte`, `in`, `nin`, `contains`

## Practical Notes

- Use `fieldPath` for nested field checks.
- Keep branch instructions single-purpose.
- In branch `then`, use `selectFields` to constrain context.
- Docs warn that decision branch `selectFields` may be first-layer constrained in some flows.
