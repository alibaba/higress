# Implementation Plan: Sensitive Data Mask Threshold

## Overview

Add threshold checking to the mask decision path in `evaluateRiskMultiModal()`. The core change is ~3 lines in `config/config.go`: gate the `hasMask = true` assignment on the existing `exceeds` variable and add an `else` branch that logs the skip event. Then update existing tests whose expected results change and add new boundary tests.

## Tasks

- [x] 1. Modify `evaluateRiskMultiModal()` mask branch to check threshold
  - [x] 1.1 Add `exceeds` guard and log-skip branch in `config/config.go`
    - In `evaluateRiskMultiModal()`, change the mask branch from:
      ```go
      if dimAction == "mask" && detail.Suggestion == "mask" {
          hasMask = true
      }
      ```
      to:
      ```go
      if dimAction == "mask" && detail.Suggestion == "mask" {
          if exceeds {
              hasMask = true
          } else {
              proxywasm.LogInfof("safecheck_mask_skipped: type=%s, suggestion=%s, level=%s, threshold=%s",
                  detail.Type, detail.Suggestion, detail.Level, config.GetSensitiveDataLevelBar(consumer))
          }
      }
      ```
    - _Requirements: 1.1, 1.2, 2.1_

- [x] 2. Update existing tests to reflect new threshold behavior
  - [x] 2.1 Update tests where mask detail level is below `SensitiveDataLevelBar`
    - In `evaluate_risk_test.go`, for each test where `sensitiveDataAction=mask`, `Suggestion=mask`, and `Level < SensitiveDataLevelBar` (default S4), either lower the threshold to match the level or change the expected result to `RiskPass`:
      - **TC_EVAL_001**: `Level=S2`, threshold=S4 → set `SensitiveDataLevelBar` to `"S2"` to preserve `RiskMask` expectation
      - **TC_EVAL_005**: `Level=S1`, threshold=S4 → set `SensitiveDataLevelBar` to `"S1"` to preserve `RiskMask` expectation
      - **TC_EVAL_013**: `Level=S1`, threshold=S4 → set `SensitiveDataLevelBar` to `"S1"` to preserve `RiskMask` expectation
      - **TC_EVAL_018**: `Level=S2`, threshold=S4 → set `SensitiveDataLevelBar` to `"S2"` to preserve `RiskMask` expectation
      - **TC_EVAL_022**: `Level=S2`, threshold=S4 → set `SensitiveDataLevelBar` to `"S2"` to preserve `RiskMask` expectation
      - **TC_EVAL_027**: `Level=S2`, threshold=S4 (consumer config) → add `sensitiveDataLevelBar: "S2"` to consumer config or global config to preserve `RiskMask` expectation
      - **TC_EVAL_029**: `Level=S1`, threshold=S4 → set `SensitiveDataLevelBar` to `"S1"` to preserve `RiskMask` expectation
      - **TC_EVAL_035**: `Level=S1`, threshold=S4 → set `SensitiveDataLevelBar` to `"S1"` to preserve `RiskMask` expectation
    - For TC_EVAL_028: `Level=S1`, threshold=S4, but `Data.Suggestion=block` → result stays `RiskBlock` regardless; mask candidate no longer set but block fallback still triggers. No change needed.
    - _Requirements: 5.1_

  - [x] 2.2 Add new threshold-boundary test cases
    - Add the following new test functions to `evaluate_risk_test.go`:
    - **TC_EVAL_036**: Below-threshold mask → `RiskPass`. Config: `SensitiveDataAction=mask`, `SensitiveDataLevelBar=S3`. Detail: `Type=sensitiveData`, `Suggestion=mask`, `Level=S1`. Expected: `RiskPass`.
    - **TC_EVAL_037**: At-threshold mask → `RiskMask`. Config: `SensitiveDataAction=mask`, `SensitiveDataLevelBar=S2`. Detail: `Type=sensitiveData`, `Suggestion=mask`, `Level=S2`. Expected: `RiskMask`.
    - **TC_EVAL_038**: Mixed above/below threshold details → `RiskMask`. Config: `SensitiveDataAction=mask`, `SensitiveDataLevelBar=S3`. Details: one with `Level=S1` (below), one with `Level=S3` (at threshold). Expected: `RiskMask` (only the at-threshold detail contributes).
    - **TC_EVAL_039**: All details below threshold → `RiskPass`. Config: `SensitiveDataAction=mask`, `SensitiveDataLevelBar=S4`. Details: two sensitiveData with `Level=S1` and `Level=S2`. Expected: `RiskPass`.
    - _Requirements: 5.2, 5.3, 5.4_

- [x] 3. Checkpoint
  - Run `go test ./plugins/wasm-go/extensions/ai-security-guard/config/...` and ensure all tests pass. Ask the user if questions arise.

- [x] 4. Property-based tests for threshold behavior
  - [x] 4.1 Write property test: above-threshold mask produces RiskMask
    - **Property 1: Above-threshold mask produces RiskMask**
    - Generate random (level, threshold) pairs from valid sensitive levels where `LevelToInt(level) >= LevelToInt(threshold)`, construct a single-detail Data with `Type=sensitiveData`, `Suggestion=mask`, config `SensitiveDataAction=mask`, `SensitiveDataLevelBar=threshold`, and verify result is `RiskMask`.
    - **Validates: Requirements 1.1, 4.1**

  - [x] 4.2 Write property test: below-threshold mask produces RiskPass
    - **Property 2: Below-threshold mask produces RiskPass**
    - Generate random (level, threshold) pairs from valid sensitive levels where `LevelToInt(level) < LevelToInt(threshold)`, construct a single-detail Data, and verify result is `RiskPass`.
    - **Validates: Requirements 1.2, 1.3**

  - [x] 4.3 Write property test: per-detail threshold independence
    - **Property 3: Per-detail threshold independence for multiple sensitiveData details**
    - Generate random lists of sensitiveData details with varying levels and a random threshold. Verify result is `RiskMask` iff at least one detail has `LevelToInt(Level) >= LevelToInt(threshold)`.
    - **Validates: Requirements 1.4**

  - [x] 4.4 Write property test: block triggers always produce RiskBlock
    - **Property 4: Block triggers always produce RiskBlock**
    - Generate random details with `Suggestion=block` and verify `RiskBlock`. Also generate details where `dimAction=block` and `exceeds=true` and verify `RiskBlock`.
    - **Validates: Requirements 3.1, 3.2**

  - [x] 4.5 Write property test: top-level gates produce RiskBlock
    - **Property 5: Top-level gates produce RiskBlock**
    - Generate random `Data.RiskLevel` and `contentModerationLevelBar` where level >= threshold, verify `RiskBlock`. Same for `AttackLevel` and `promptAttackLevelBar`.
    - **Validates: Requirements 3.3, 3.4**

  - [x] 4.6 Write property test: Data.Suggestion=block fallback
    - **Property 6: Data.Suggestion=block fallback produces RiskBlock**
    - Generate random non-blocking details with `Data.Suggestion=block`, verify `RiskBlock`.
    - **Validates: Requirements 3.5**

- [x] 5. Final checkpoint
  - Run full test suite: `go test ./plugins/wasm-go/extensions/ai-security-guard/config/...`. Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- The core code change (task 1.1) is ~3 lines; the bulk of work is test updates
- Property tests use the `testing/quick` stdlib package or `rapid` library
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
