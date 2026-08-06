# Human Review Projection Generation

Use this guide to generate the ordinary Markdown comment passed to `issue-spec projection upsert`. A projection helps a person review authoritative workflow data; it is not a typed artifact, gate, status, transition, or source of truth.

## Contents

- [Authority and inputs](#authority-and-inputs)
- [Decision integrity and comprehension](#decision-integrity-and-comprehension)
- [Generation procedure](#generation-procedure)
- [Shared information design](#shared-information-design)
- [Phase recipes](#phase-recipes)
- [Markdown and HTML skeleton](#markdown-and-html-skeleton)
- [Sandbox and accessibility](#sandbox-and-accessibility)
- [Update and acceptance checklist](#update-and-acceptance-checklist)

## Authority and inputs

Read only the authoritative records needed for the phase. Do not expand or reuse an older projection's HTML as input.

| Phase | Required authoritative inputs | Optional inputs |
|---|---|---|
| `proposal-choice-brief` | Proposal body | Confirmed SPEC facts already available; linked evidence |
| `design-explainer` | Design body; confirmed SPEC facts | Existing TASK facts when regenerating; linked evidence |
| `implement-execution-brief` | Implement body; Design invariants and decisions; TASK; current PROCESS records, dependencies, links, statuses, and handoffs | Exact-current code-change, review, verification, or check evidence |

Apply these authority rules:

- Treat issue bodies and typed artifacts as authoritative.
- Treat the projection as a coverage-complete review surface over the current phase inputs, not as a delta, changelog, executive summary, or component inventory. Lead from a human or operator scene and a concrete case into the technical model. Synthesis may compress wording and move detail behind progressive disclosure, but it must not omit an applicable review dimension; source links support verification and drill-down, not discovery of omitted concerns.
- Label every recommendation, comparison, candidate PROCESS, estimate, confidence value, and inferred relationship as synthesis. Never let synthesis override an authoritative record.
- Do not infer workflow readiness, PROCESS boundaries, gates, or status from an estimate, visual state, HTML control, or projection text. In Implement, invariant cohesion and typed dependencies define semantics; file count, line count, complexity, Agent count, and parallelism are planning aids only.
- Link claims to their source issue or typed comment. If evidence is absent, say that it is absent instead of filling the gap.
- Keep the projection self-explanatory as ordinary Markdown because GitHub displays the fenced HTML source and does not execute the preview.

## Decision integrity and comprehension

Treat decision integrity as the non-negotiable constraint. Never make a projection easier to consume by omitting a material fact, hiding uncertainty or tradeoffs, weakening a boundary, or framing credible options unevenly.

Within that constraint, optimize every choice of content, wording, order, interaction, and visual form for the least cognitive effort needed to build the correct mental model and make the right decision. Prefer the form that removes unstated premises, acronym decoding, context switching, and unnecessary inference: scene before mechanism, concrete case before abstraction, comparison table for repeated criteria, flow for sequence, and progressive disclosure for detail.

Use this precedence when goals conflict:

1. Preserve fidelity to authoritative facts, uncertainty, risks, alternatives, and decision consequences.
2. Preserve decision sufficiency: the reviewer can identify what matters, compare credible options, and understand what would make a choice wrong.
3. Then maximize comprehension by simplifying language and presentation. If complexity is decision-relevant, explain it through the case and move supporting detail behind drill-down instead of deleting it.

## Generation procedure

1. Select the phase inputs above with bounded issue-spec reads. Do not request `--expand-preview` or `--expand-all-previews`.
2. Build a coverage ledger before writing UI:
   - enumerate the phase-specific review dimensions below and map every applicable authoritative input to top-level attention, progressive drill-down, or an explicit not-applicable rationale;
   - identify the affected person or operator, their goal, the situation that triggers the change, and one representative case that can be traced end to end;
   - confirmed facts and constraints, each with a source link;
   - unresolved evidence gaps;
   - open decisions a human must make, each with the credible options;
   - phase-specific derived synthesis, clearly labeled.
3. Resolve contradictions in favor of authoritative data. Stop generation if two authoritative inputs conflict or a required record cannot be identified uniquely.
4. Write a compact Markdown fallback first. Open with the scene, why it matters, and a concrete before/after or request-to-outcome case; then expose every applicable review dimension, required human decisions, critical constraints, and source links without running HTML.
5. Add one valid `html-preview` fence containing a complete, standalone document. Prefer one preview per projection so the intended review surface is the first preview.
6. Serialize a deterministic source manifest containing the selected source identities, body digests or exact revisions, and typed statuses and links. Hash that manifest for `--source-digest`; do not hash only `projection.md`, and exclude the projection itself.
7. Audit coverage before validating presentation: every applicable ledger entry must be discoverable, and a phase with no open decision must still expose settled premises, evidence gaps, alternatives, boundaries, risks, and verification obligations. Then validate the Markdown, preview metadata, keyboard flow, narrow layout, and GitHub fallback.
8. Upsert the one logical phase comment. The CLI appends the projection marker; do not add a typed marker or projection marker inside the body.

Example write:

```sh
issue-spec projection upsert \
  --repo owner/repo \
  --issue 123 \
  --phase implement-execution-brief \
  --source-digest "$SOURCE_DIGEST" \
  --body-file projection.md \
  --json
```

## Shared information design

Optimize for the reviewer reaching a correct understanding and decision with the least avoidable cognitive effort, not for decorative animation, novelty, or information density.

Use this hierarchy:

1. **Scene and outcome:** identify who is affected, what they are trying to accomplish, what happens today, and what becomes observably better.
2. **Concrete case walkthrough:** trace one realistic case from trigger to outcome, including one meaningful failure or boundary. Pair each step with `What the person sees`, `What the system does`, and `What the reviewer should verify`.
3. **Review request and attention queue:** state the exact request, then unresolved decisions, evidence gaps, and settled choices needing constraint verification. When there is no open decision, say so without collapsing the rest of the review surface.
4. **Recommendation and technical model:** connect the case to premises, benefits, costs, alternatives, invariants, data flow, dependency graph, or execution plan.
5. **Drill-down and sources:** place component-level detail, history, estimates, and source links behind tabs, accordions, or `details`.

The first viewport must make sense without repository-specific acronyms or prior design context. Translate IDs and technical nouns into their role in the case before using them as navigation labels. When authoritative inputs do not contain safe example values, use clearly labeled illustrative values without inventing facts or presenting them as evidence.

Use a calm, consistent visual language:

| Meaning | Treatment |
|---|---|
| Settled / clearly better choice | Neutral or green-tinted card, `Confirmed` label, supporting premise, and alternative cost; ask the reviewer to verify the premise rather than decide again |
| Needs evidence | Amber-tinted card, `Evidence needed` label, known facts, missing evidence, and the next way to obtain it |
| Needs human decision | Strong blue or indigo accent, `Decision needed` label, recommendation, comparable options, tradeoffs, and an exact question |
| Actual workflow blocker | Red reserved for a typed blocker or failed invariant; include the affected work and source |
| Synthesis / estimate | Muted or dashed treatment with `Planning aid` or `Estimate` and confidence; never resemble authoritative status |

Do not use color alone. Pair it with text and, when useful, a simple icon. Avoid pulsing, autoplay tours, animated diagrams, artificial urgency, and dense walls of cards. Prefer:

- comparison tables for two or more choices with repeated criteria;
- a compact flow or DAG when dependencies affect review;
- a small metric strip only for decision-relevant counts;
- progressive disclosure for detailed evidence and history;
- direct source links beside the claim they support.

## Phase recipes

### Proposal choice brief

Help a reviewer turn scene, goal, and proposed scope into decisions before complete SPEC authoring:

1. Lead with one representative person or operator, their current friction, and a concrete before/after journey.
2. State the desired outcome, success signal, and proposed boundary in terms visible in that journey.
3. Separate settled choices, evidence-dependent items, and genuine decisions; show how each option changes the case.
4. Cover risks, assumptions, alternatives, reversibility, expected SPEC coverage, non-goals, and what remains intentionally out of scope.

### Design explainer

Help a reviewer understand correctness and alternatives before complete TASK planning:

1. Lead with a concrete request, event, or operator action and its expected observable outcome; then name the selected architecture and every invariant it must preserve.
2. Trace that case through the end-to-end data or control flow as numbered steps, with a meaningful failure path and trust boundaries visible.
3. Cover interfaces, shared state, cache or persistence behavior, state transitions, and downstream consumers where applicable.
4. Compare rejected or conditional alternatives and the premises that made the selected design preferable.
5. Cover compatibility, migration, rollout, rollback, risks, verification strategy, and traceability to every active SPEC.

Use interaction to explore layers or branches; do not add animation merely to make the page feel active.

### Implement execution brief

Help a reviewer validate the execution strategy before complete PROCESS planning and monitor it after PROCESS records exist.

Open with one concrete acceptance case and show how the candidate or current PROCESS sequence carries it from trigger to verified outcome. Keep the DAG as the technical explanation of that case, not the first concept a reviewer must decode.

The top level must show:

- the representative acceptance case and its observable outcome;
- the invariant-based work packages or current typed PROCESS DAG;
- counts for planned/ready/active/blocked/completed work, without inventing statuses;
- the critical path and safe parallel groups;
- suggested Agent allocation and the reason for each distinct role;
- actual typed blockers;
- shared touchpoints and independent review/verify obligations.

Before PROCESS records exist, label work packages and dependencies `Candidate planning`. After they exist, replace candidates with the current typed PROCESS records and links; never leave a conflicting candidate DAG looking authoritative.

For each work package or PROCESS drill-down, show:

- owned Design invariant and acceptance outcome;
- parent TASK and covered SPEC/scenarios;
- dependencies, predecessor handoff, and downstream consumers;
- major entry points, owned areas, and shared touchpoints;
- role recommendation and whether parallel execution is safe;
- focused tests, generators, review, and verification obligations;
- code-volume range with confidence and stated basis;
- correctness complexity separately from change-surface, verification, rollout, and coordination complexity;
- human review focus and authoritative links.

Explain correctness complexity as the reasoning difficulty of preserving the invariant across states, failures, concurrency, compatibility, or trust boundaries. It is not a synonym for lines changed.

## Markdown and HTML skeleton

Generate `projection.md` in this shape:

````markdown
# Implement execution review

> Human-review projection. The phase issue bodies and typed artifacts remain authoritative.

## Review summary

**Recommendation:** ...

**Decision requested:** ...

## Decisions and evidence

- **Decision needed:** ... ([source](...))
- **Confirmed:** ... ([source](...))
- **Planning estimate:** ...; confidence: medium; basis: ...

```html-preview id=implement-execution-review version=1 title="Implement execution review" height=720
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Implement execution review</title>
  <style>
    :root {
      color-scheme: light;
      --ink: #172033; --muted: #607086; --line: #d9e0ea;
      --surface: #fff; --soft: #f5f7fb; --decision: #4056b4;
      --settled: #187252; --evidence: #9a6500; --blocked: #a93636;
    }
    * { box-sizing: border-box; }
    body { margin: 0; padding: 20px; color: var(--ink); background: var(--surface);
      font: 15px/1.5 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    main { width: min(100%, 1080px); margin: auto; }
    .summary, .card { border: 1px solid var(--line); border-radius: 14px; padding: 16px; }
    .grid { display: grid; grid-template-columns: repeat(12, 1fr); gap: 12px; }
    .card { grid-column: span 6; background: var(--soft); }
    .decision { border-left: 5px solid var(--decision); }
    .settled { border-left: 5px solid var(--settled); }
    .estimate { border-style: dashed; color: var(--muted); }
    button, input, textarea { font: inherit; }
    button:focus-visible, input:focus-visible, textarea:focus-visible,
    summary:focus-visible, a:focus-visible { outline: 3px solid #8da2ff; outline-offset: 2px; }
    @media (max-width: 700px) { body { padding: 12px; } .card { grid-column: 1 / -1; } }
    @media (prefers-reduced-motion: reduce) { *, *::before, *::after {
      animation-duration: .01ms !important; transition-duration: .01ms !important; } }
  </style>
</head>
<body>
  <main>
    <section class="summary" aria-labelledby="summary-title">
      <h1 id="summary-title">Implement execution review</h1>
      <p>Who is affected, the situation they are in, and the observable outcome this plan must deliver.</p>
    </section>
    <section aria-labelledby="case-title">
      <h2 id="case-title">Concrete case walkthrough</h2>
      <div class="grid">
        <article class="card"><h3>What the person sees</h3><p>...</p></article>
        <article class="card"><h3>What the system does</h3><p>...</p></article>
      </div>
      <p><strong>Reviewer verifies:</strong> ...</p>
    </section>
    <section aria-labelledby="attention-title">
      <h2 id="attention-title">Needs your attention</h2>
      <div class="grid">
        <article class="card decision"><h3>Decision needed</h3><p>...</p></article>
        <article class="card settled"><h3>Confirmed constraint</h3><p>...</p></article>
        <article class="card estimate"><h3>Planning estimate</h3><p>...</p></article>
      </div>
    </section>
    <!-- Add the phase model, drill-down, and source links. -->
  </main>
  <script>
    // Add only local presentation state (filters, tabs, accordions, DAG focus).
  </script>
</body>
</html>
```
````

Use a stable preview ID for the logical phase view. Metadata accepts only `id`, `version`, `title`, and `height`; IDs use lowercase letters, digits, and hyphens, `version` is `1`, title is at most 120 Unicode scalar values, and height is clamped to 240–720. Keep a body to at most eight previews and each preview source below 256 KiB.

## Sandbox and accessibility

- Produce a complete inline document. Inline CSS and JavaScript; do not depend on CDNs, remote fonts, images, APIs, modules, storage, cookies, popups, navigation, forms, downloads, media, or same-origin access.
- Assume an opaque origin and `sandbox="allow-scripts"`. Never request credentials, CSRF values, repository tokens, or issue data from the host.
- Use JavaScript only for review-relevant filtering, tabs, accordions, DAG focus, and comparisons. Keep core content available without animation.
- Use semantic landmarks, heading order, native buttons/inputs, explicit labels, fieldsets and legends for choices, status text, table headers, and meaningful link text.
- Support keyboard-only operation, visible focus, 200% zoom, narrow/mobile layouts, long localized strings, and `prefers-reduced-motion`.
- Avoid horizontal page scrolling. Wrap long IDs and URLs, make wide tables scroll within a labeled region, and never fix heights for content cards.
- Escape all authoritative and custom text before inserting it into HTML or JavaScript. Treat it as untrusted display data, never as markup, script, CSS, or command input.

## Update and acceptance checklist

Before upsert:

- [ ] No simplification hides a material fact, uncertainty, tradeoff, boundary, risk, alternative, or decision consequence.
- [ ] Content order and presentation minimize acronym decoding, context switching, and unstated inference while preserving everything needed for a correct decision.
- [ ] The first viewport identifies an affected person or operator, their goal, a concrete trigger-to-outcome case, and why the change matters before introducing component names or artifact IDs.
- [ ] The case walkthrough covers the normal path and one meaningful failure or boundary, mapping human-visible effects to system behavior and review obligations.
- [ ] Repository-specific terms and illustrative values are translated or labeled; the projection does not invent facts to make the story concrete.
- [ ] A coverage ledger maps every applicable authoritative input and phase review dimension to top-level attention, progressive drill-down, or an explicit not-applicable rationale.
- [ ] The projection is understandable as a complete current review surface rather than a delta from design information the reviewer is presumed to know.
- [ ] If there is no open decision, settled premises, evidence gaps, alternatives, boundaries, risks, and verification obligations remain discoverable.
- [ ] Every displayed fact has an authoritative source or is labeled synthesis.
- [ ] The Markdown fallback communicates recommendations, decisions, constraints, and links without HTML execution.
- [ ] Settled, evidence-needed, decision-needed, blocker, and estimate states are visually and textually distinct.
- [ ] Implement estimates and Agent/parallelism suggestions do not define PROCESS semantics, readiness, or gates.
- [ ] The preview uses one stable ID, valid metadata, inline assets, no network dependencies, and no manually authored projection/typed marker.
- [ ] The page works with keyboard, visible focus, narrow width, 200% zoom, reduced motion, and long text.
- [ ] Source size and preview-count limits are respected.
- [ ] `--source-digest` covers the authoritative input manifest, not the generated projection.

After upsert:

- [ ] Re-read the unique phase projection descriptor without expanding its source and confirm phase, owner, source digest, and one logical comment.
- [ ] On self-host, run the first preview and verify layout, console, and interaction.
- [ ] On GitHub, verify the ordinary Markdown remains sufficient and make no claim that the fenced HTML executes.
- [ ] When authoritative inputs change, regenerate from them and update the same logical projection; never edit the projection as a substitute for updating typed data.
