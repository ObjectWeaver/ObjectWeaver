# Guardrails: Common Pitfalls and Fixes

## 1) Endpoint mismatch

Issue: examples use both `/api/objectGen` and `/objectGen`.

Fix: default to `/api/objectGen` unless deployment confirms otherwise.

## 2) Missing dependency order

Issue: field uses prior output but no `processingOrder`.

Fix: define producer-before-consumer order.

## 3) Missing context wiring

Issue: relying on order alone for context passing.

Fix: add `selectFields` (or `narrowFocus`) explicitly.

## 4) Over-broad context

Issue: too many selected fields degrade quality/cost.

Fix: pass the minimum required fields.

## 5) Unbounded recursive logic

Issue: no stop/fallback in iterative refinement.

Fix: set `maxIterations` and define `default` branch behavior.

## 6) Streaming with negative priorities

Issue: expecting streamed updates from batched fields.

Fix: keep stream workloads at `priority >= 0`.

## 7) Weak instructions

Issue: vague field instructions produce unstable outputs.

Fix: make instructions explicit (format, limits, quality criteria).

## 8) No parser tolerance

Issue: client expects one exact response envelope.

Fix: support direct JSON and wrapped/protobuf-like structures.
