# Recipe: Arrays and Recursive Refinement

## Use When

- You need lists of objects.
- You need iterative quality improvement with nested checks.

## Array Generation Steps

1. Use `type: "array"` for list fields.
2. Define `items` schema.
3. Put count/uniqueness rules in array `instruction`.
4. Use `processingOrder` when later fields depend on arrays.

## Recursive Refinement Steps

1. Generate draft field.
2. Attach `scoringCriteria`.
3. Add `decisionPoint` branch for low-score path.
4. In low-score branch, regenerate improved content.
5. Add `default` fallback.
6. Optional: add `recursiveLoop.maxIterations` for bounded retries.

## Design Rule

Always bound recursion and always define fallback behavior.
