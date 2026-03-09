## Context

The current trend endpoint returns raw samples only. That is enough for simple sparklines, but not
enough for an operator-facing dashboard that needs to call out regressions quickly. Symphony already
computes integral live metrics; the historical path should expose the same kind of compressed signal.

## Goals / Non-Goals

**Goals:**
- Derive a few stable rollups from existing trend points without introducing a reporting engine.
- Evaluate alert thresholds in the backend so the frontend remains simple and DB-agnostic.
- Keep the payload compact and tied to bounded trend windows.

**Non-Goals:**
- Arbitrary user-defined formulas.
- Full anomaly detection or ML-based forecasting.
- PagerDuty-style notification delivery.

## Decisions

- Compute rollups from the bounded trend window already requested by the client.
- Start with deterministic metrics: average score, latest delta, short-window slope, and warning
  pressure.
- Keep alert evaluation threshold-based and explicit in the response rather than deriving UI-only
  color rules in the frontend.
- Reuse the existing delivery trend endpoint and extend its response shape instead of adding a third
  endpoint.

## Risks / Trade-offs

- [Too many alert rules] → start with a very small set tied to blocked work, failing checks, and
  delivery-health regression.
- [Noisy trends with few points] → degrade gracefully when the window has insufficient samples.
- [Frontend duplication] → centralize alert semantics in the backend response contract.
