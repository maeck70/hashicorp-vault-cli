# Project Guidelines

## 1. Build & Output Constraints
* **Always build executables into `./bin`** (e.g., `./bin/rmq-stream-app`).
* Do not place compiled binaries in the repository root directory.

### 2. Go Coding Standards & Modern Idioms
* **Prefer `switch` Over `if` Ladders**: Avoid multi-branch `if ... else if ... else` ladders and repetitive sequential conditional assignments. Use idiomatic Go `switch` (expressionless `switch` or value/type `switch`) for multi-branch control flow.
* **Prefer Built-in `min` and `max`**: Always use Go's built-in `min()` and `max()` functions for clamping, bounding, and threshold checks instead of manual `if x < min { x = min }` or `if x > max { x = max }` blocks.
* **Prefer `any` Over `interface{}`**: Use Go's `any` type alias instead of `interface{}` for empty interface definitions and generic value placeholders.
* **Prefer Range Over Integer**: Use Go's `for i := range n` (or `for range n`) syntax instead of traditional three-clause counting loops `for i := 0; i < n; i++`.
* **Avoid Deprecated APIs**: Do not use deprecated functions or methods. Keep third-party library calls updated to modern equivalents (e.g., Bubbles Viewport `ScrollUp` / `ScrollDown` and `PageUp` / `PageDown` instead of deprecated `LineUp`, `LineDown`, `ViewUp`, `ViewDown`).

### 3. Go Modules
* Use standard go modules as much as possible
