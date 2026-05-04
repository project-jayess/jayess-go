# Runtime Architecture

This document describes how Jayess runtime work should evolve. QuickJS is a
useful reference for subsystem boundaries and dynamic-language runtime problems,
but Jayess is building its own runtime model:

- Jayess is targeting **no GC, scope-based cleanup**
- QuickJS uses **reference counting plus cycle collection**
- Jayess can borrow implementation ideas, testing strategy, and subsystem split
  ideas from QuickJS, but it must keep **Jayess-specific ownership contracts**

## Goals

- keep code generation simple by lowering dynamic behavior into a stable C
  runtime surface
- make value ownership explicit enough that section `9.5` can eventually be
  checked honestly
- reduce the amount of ad hoc behavior living in monolithic `runtime.c`
- keep host-resource wrappers consistent with the same ownership model used for
  plain values

## QuickJS Comparison

QuickJS is helpful as a comparison point because it already separates several
runtime concerns clearly:

- value representation and call dispatch
- object/property behavior
- exceptions and stack reporting
- regexp / unicode / typed-array subsystems
- promise/module/runtime host hooks
- memory accounting and stress testing

Jayess should copy the **questions** QuickJS answers, not the implementation
strategy:

- What does a helper return: owned, borrowed, immortal, or transferred?
- Which code path owns callback results?
- Where do host handles get closed?
- Which operations are generic runtime semantics versus stdlib host glue?
- How do we stress those paths under ASAN/LSAN/UBSAN?

For Jayess, that comparison also has a compiler-side half. The useful
compiler-side questions are still Jayess-owned questions, not imported answers:

- Which semantics should be normalized in lowering versus deferred to runtime
  helpers?
- Which ownership facts should lowering preserve explicitly so codegen does not
  have to rediscover them heuristically?
- Which helper families need one stable call/cleanup contract across
  lowering/codegen/runtime boundaries?
- Which dynamic behaviors deserve one canonical emitted shape instead of many
  special-case codegen paths?
- Which proof lanes should live at compiler/lowering level versus executable
  runtime/sanitizer level?

The practical Jayess-owned version of that comparison is:

- for value helpers:
  - which helpers allocate fresh boxes versus return immortal/static values
  - which helpers can alias existing storage and therefore must never be put on
    generic statement-exit cleanup paths
- for object/array/property behavior:
  - which writes store aliased values versus clone-or-materialize fresh storage
  - which reads return borrowed aliases versus fresh wrappers or snapshots
- for callback and call dispatch:
  - which path owns callback results on direct, bound, and generic apply calls
  - where bound-arg materialization stops borrowing temporary wrappers
- for queues and async runtime state:
  - which queued payloads are aliased, cloned, detached, or shared on purpose
  - where settlement, enqueue, dequeue, and shutdown transfer ownership
- for host wrappers:
  - where the authoritative live host handle lives
  - what after-close, duplicate-close, and forgotten-close behavior is supposed
    to be in Jayess terms
- for accounting:
  - which runtime entities deserve live counts versus richer structural
    summaries
  - where those counters update in Jayess allocation/free paths

And the expected Jayess-owned answer shape for those areas is:

- value helpers:
  - a helper-by-helper ownership classification in Jayess vocabulary
- object/array/property behavior:
  - explicit aliasing, snapshot, and fresh-materialization rules for reads and
    writes
- callback and call dispatch:
  - one ownership contract describing direct, bound, and fallback invocation
    paths
- queues and async runtime state:
  - explicit enqueue/dequeue/settlement/shutdown transfer rules for queued
    payloads
- host wrappers:
  - one authoritative lifecycle contract for live handles, after-close state,
    and duplicate-close behavior
- accounting:
  - a Jayess-facing summary surface backed by concrete runtime counters at known
    hook points

For the compiler-side half, the expected Jayess-owned answer shape is:

- lowering boundaries:
  - explicit Jayess rules for what gets normalized before runtime dispatch
- ownership propagation:
  - lowering/codegen artifacts that preserve fresh/alias/borrowed distinctions
- helper call shapes:
  - one emitted contract per helper family instead of many ad hoc variants
- dynamic behavior lowering:
  - canonical emitted forms for equivalent language/runtime operations
- proof placement:
  - explicit separation between compile-time proof artifacts and executable
    runtime proof artifacts

And each of those Jayess-owned answer shapes should eventually connect to an
appropriate proof boundary:

- value helpers:
  - ownership classification that codegen can consume mechanically
- object/array/property behavior:
  - aliasing rules backed by executable coverage and lifetime-stress coverage
- callback and call dispatch:
  - ownership contracts backed by cleanup probes and sanitizer lanes
- queues and async runtime state:
  - transfer rules backed by executable queue semantics and async stress lanes
- host wrappers:
  - lifecycle contracts backed by close/after-close/double-close regression
    coverage
- accounting:
  - summary surfaces backed by implemented counters plus usefulness validation on
    Jayess workloads

For the compiler-side half, the proof boundary should eventually be:

- lowering boundaries:
  - lowering tests plus backend semantic coverage for the normalized forms
- ownership propagation:
  - compile-time cleanup/proven-fresh artifacts that shrink heuristic codegen
- helper call shapes:
  - compiler/backend coverage showing one helper family no longer depends on
    many emitted special cases
- dynamic behavior lowering:
  - equivalent emitted/runtime behavior across the canonicalized shapes
- proof placement:
  - a documented split showing which claims are compiler artifacts versus which
    require executable/sanitizer evidence

At the current checklist boundary, the active compiler-side Jayess-owned
QuickJS-inspired threads are:

- lowering boundaries:
  - blocked by places where equivalent dynamic behavior still reaches runtime
    through more than one lowering/codegen shape
- ownership propagation:
  - blocked by the still-missing mechanical propagation of helper ownership into
    codegen cleanup decisions
- helper call shapes:
  - blocked by helper families whose lowering/codegen/runtime contract is still
    spread across multiple emitted special cases
- dynamic behavior lowering:
  - blocked by canonicalization gaps where equivalent behaviors still depend on
    helper-specific or arity-specific emitted forms
- proof placement:
  - blocked by checklist areas where compiler-proof versus executable/sanitizer
    proof is still documented by slice rather than enforced by one explicit
    discipline

So, as on the runtime side, each active compiler-side thread already names the
current Jayess compiler bottleneck that is keeping that thread alive rather
than only naming a topic area in the abstract.

As on the runtime side, these compiler-side threads are useful only when they
are framed in terms of Jayess compiler bottlenecks and compiler/lowering/codegen
gates, not in terms of matching external subsystem labels.
They should also stay active only while they point to a concrete next Jayess
compiler artifact, proof artifact, or checklist movement opportunity rather
than remaining open as comparison categories alone.
There is also no separate "QuickJS compiler compatibility" success track here:
compiler-side QuickJS-inspired progress only counts when it advances real
Jayess compiler/lowering/codegen gates or shrinks real Jayess compiler-side
proof surfaces.
That also means compiler-side prioritization should flow from Jayess's still-red
compiler/lowering/codegen gates and compiler-side proof surfaces first, not
from whichever external compiler comparison seems most attractive.

The next concrete Jayess artifact expected from each active compiler-side
thread is also clear at this boundary:

- lowering boundaries:
  - a lowering rule or backend coverage artifact that collapses one equivalent
    dynamic behavior into one canonical emitted shape
- ownership propagation:
  - a lowering/codegen ownership artifact that preserves one more fresh/alias
    distinction mechanically into cleanup decisions
- helper call shapes:
  - a compiler/backend artifact that unifies one helper family behind a smaller
    emitted call/cleanup contract
- dynamic behavior lowering:
  - a canonical emitted-form artifact proving one currently split behavior no
    longer depends on helper-specific or arity-specific lowering
- proof placement:
  - a documented or enforced Jayess proof-boundary artifact that makes one
    compiler-proof versus executable-proof split more explicit

And when those compiler-side artifacts land, the Jayess checklist credit they
are expected to move is also concrete:

- lowering boundaries:
  - the compiler-side half of `dynamic behavior lowering`
  - and secondarily `helper call shapes`
- ownership propagation:
  - the compiler-side half of `ownership propagation`
  - and secondarily the still-red `codegen uses that ownership classification
    directly instead of heuristic cleanup whitelists` gate
- helper call shapes:
  - the compiler-side half of `helper call shapes`
  - and secondarily the still-red callback/helper-family unification surfaces
- dynamic behavior lowering:
  - the compiler-side half of `dynamic behavior lowering`
  - and secondarily the canonicalization pressure behind several runtime helper
    families
- proof placement:
  - the compiler-side half of `proof placement`
  - and secondarily the explicit split between compiler artifacts and
    executable/sanitizer evidence throughout the broad `9.5` gate map

So the still-red Jayess compiler/lowering/codegen gates currently line up with
the active compiler-side threads like this:

- heuristic cleanup/codegen-classification gates:
  - primarily `ownership propagation`
  - and secondarily `helper call shapes`
- helper-family contract-sprawl gates:
  - primarily `helper call shapes`
  - and secondarily `lowering boundaries`
- canonical emitted-form / split-lowering gates:
  - primarily `dynamic behavior lowering`
  - and secondarily `lowering boundaries`
- blurry compiler-proof versus executable-proof responsibility:
  - primarily `proof placement`

And the compiler-side proof surfaces currently line up with those same active
threads like this:

- heuristic cleanup / non-mechanical ownership propagation:
  - primarily `ownership propagation`
  - and secondarily `helper call shapes`
- helper-family emitted contract sprawl:
  - primarily `helper call shapes`
  - and secondarily `lowering boundaries`
- split canonical emitted forms for equivalent behavior:
  - primarily `dynamic behavior lowering`
  - and secondarily `lowering boundaries`
- blurry compiler-artifact versus executable/sanitizer proof responsibility:
  - primarily `proof placement`

The active compiler-side and runtime-side Jayess-owned threads also intersect in
concrete ways:

- ownership propagation:
  - feeds the runtime-side `value helpers` thread directly
- helper call shapes:
  - feeds the runtime-side `callback and call dispatch` thread directly
- dynamic behavior lowering:
  - feeds the runtime-side `object/array/property behavior` and
    `queues and async runtime state` threads where multiple emitted forms still
    reach the same runtime helper families
- proof placement:
  - feeds the runtime-side broad `9.5` gate discipline by making compiler-proof
    versus executable/sanitizer-proof boundaries more explicit
- lowering boundaries:
  - feeds several runtime-side threads indirectly by reducing how many emitted
    shapes reach the same runtime ownership/lifecycle surfaces

The reverse dependency also matters: runtime-side progress feeds compiler-side
progress when it reduces ambiguity at the helper boundary:

- value helpers:
  - feeds `ownership propagation` by making helper return classes easier to
    preserve mechanically through lowering/codegen
- callback and call dispatch:
  - feeds `helper call shapes` and `dynamic behavior lowering` by stabilizing
    what one helper-family contract is supposed to mean at runtime
- object/array/property behavior:
  - feeds `dynamic behavior lowering` by making canonical emitted forms easier
    to justify against one runtime ownership/aliasing model
- queues and async runtime state:
  - feeds `proof placement` by making it clearer which claims must remain
    executable/sanitizer proofs rather than compiler artifacts
- host wrappers:
  - feeds `proof placement` and secondarily `helper call shapes` where host
    wrapper method exposure still crosses compiler/runtime boundaries

Because those dependencies are bidirectional, a meaningful change in one active
compiler-side or runtime-side thread should trigger a revisit of the mapped
counterpart threads so the coupling map stays aligned with the real Jayess
state.
If that revisit changes the real dependency picture, the mapped thread's
bottleneck, next artifact, target gate, expected evidence, or leverage ordering
should be refreshed too rather than leaving the coupling entry stale.
Its non-inherited assumption and retirement condition should be refreshed too,
so dependency-driven updates keep the full Jayess-owned coupling record aligned
instead of only the narrower operational subset.
And if the refreshed dependency changes which thread now has the strongest
Jayess checklist leverage, the affected compiler-side or runtime-side priority
ordering should be recalculated as well.
If a revisit shows that a previously named compiler/runtime dependency is no
longer real, that coupling entry should be removed rather than preserved as
historical comparison scaffolding.
If a lower-leverage exception had depended on that removed coupling, that
exception should also be re-justified through some other still-real Jayess
dependency or dropped with the stale coupling rather than preserved by inertia.
Conversely, if a new real compiler/runtime dependency appears in Jayess work,
that coupling should be added explicitly so the roadmap keeps reflecting the
current system instead of an outdated one.
And when such a coupling is added, it should immediately inherit the same
Jayess-owned metadata discipline as the existing ones: what bottleneck it
connects, what artifact it should move next, what gate or proof surface it is
meant to shrink, what evidence would count as progress, what non-inherited
assumption it rejects, how it can retire, and where it sits in leverage
ordering.
If the new coupling is active rather than purely informational, it should also
enter the current Jayess leverage ordering immediately instead of lingering as
an unprioritized side note.
And if that newly active coupling creates a new case where compiler-side and
runtime-side work are both aimed at the same still-red Jayess gate, the leading
side for that shared gate should be computed immediately under the same
shared-gate rule rather than left implicit until some later revisit.
That newly active shared-gate case should also receive the same full
Jayess-owned metadata discipline immediately, so the coupling does not enter the
shared-gate inventory as a partially specified placeholder.
That means the chosen leading side should also point immediately to a concrete
next Jayess artifact and expected evidence, rather than only winning the
shared-gate tiebreak abstractly.
It should also name the concrete Jayess bottleneck it is expected to shrink, so
the lead choice is tied to a real reduction target rather than just a selected
side.
And it should name the concrete Jayess gate that artifact is expected to move,
so the shared-gate choice is anchored to actual checklist movement rather than
only to a pressure-reduction rationale.
It should also name its Jayess-owned retirement condition immediately, so the
new lead enters the shared-gate inventory with an explicit stop condition
instead of only an initial forward direction.
It should also name its non-inherited comparison assumption immediately, so the
new lead enters the roadmap with both a Jayess-owned direction and an explicit
boundary on what it is not inheriting from the comparison.
At the same time, the non-leading side should remain named explicitly too, so a
later shared-gate revisit is still comparing against a live alternative rather
than reconstructing one from stale memory.
If that newly computed shared-gate lead changes the footing of any
lower-leverage exception, that exception should also be re-justified or dropped
immediately rather than being left attached to the pre-coupling state.
If adding that active coupling changes the leverage order, any existing
lower-leverage exception should be re-justified or dropped against the updated
Jayess ordering.
When an active compiler-side thread and an active runtime-side thread are both
aimed at the same still-red Jayess gate, the preferred next step should be the
side that currently has the narrower named artifact and clearer expected
evidence, because that is the side most likely to turn comparison pressure into
real Jayess checklist movement next.
If both sides are equally concrete by that measure, the preferred next step
should fall back to whichever side currently has the higher Jayess leverage
ordering overall.
If leverage is also tied, the preferred next step should be the side whose
named artifact would remove, narrow, or reclassify the larger currently active
Jayess bottleneck inventory entry.
If that bottleneck size is tied too, the final fallback should be the side that
would make the proof boundary clearer sooner in Jayess terms, because that most
directly reduces future ambiguity in checklist claims.
Whichever side is chosen by that shared-gate rule should be revisited if its
artifact concreteness, expected evidence, leverage ordering, bottleneck size,
or proof-boundary clarity changes, so the choice keeps matching the current
Jayess state instead of an earlier snapshot.
If that revisit changes which side should lead, the corresponding coupling
metadata, priority ordering, and any lower-leverage exception justification
should be refreshed together so the roadmap does not preserve the old choice by
inertia.
Its non-inherited assumption and retirement condition should be refreshed too,
so a shared-gate lead change updates the full Jayess-owned coupling record
rather than only the narrower operational subset.
If a revisit shows that neither side now has concrete Jayess movement available
for that shared gate, the active shared-gate coupling should be retired until a
named Jayess artifact, proof artifact, or checklist movement opportunity
reappears on one side.
A retired shared-gate coupling should only re-enter the active set when one or
both sides again have named Jayess movement available and the chosen leading
side can be recomputed under the same shared-gate rule.
When that re-entry happens, the shared-gate coupling should also refresh its
current bottleneck, artifact, gate, evidence, and ordering metadata instead of
reusing the pre-retirement snapshot unchanged.
Its non-inherited assumption and retirement-condition metadata should be
refreshed too, so the coupling re-enters with the same full Jayess-owned
metadata discipline required of newly added active couplings.
And if a lower-leverage exception had been justified against the old shared-gate
choice, that exception should also be re-justified or dropped against the
recomputed state rather than carried forward automatically.

The Jayess-owned retirement condition for each currently active compiler-side
thread is also concrete:

- lowering boundaries:
  - one more equivalent dynamic behavior has a sufficiently canonical emitted
    shape that the current lowering-boundary bottleneck is materially narrowed
- ownership propagation:
  - one more fresh/alias distinction reaches codegen mechanically enough to
    reduce the current cleanup/classification bottleneck
- helper call shapes:
  - one helper family has a materially smaller emitted call/cleanup surface than
    before
- dynamic behavior lowering:
  - one currently split dynamic behavior no longer depends on multiple
    helper-specific or arity-specific emitted forms
- proof placement:
  - one current slice has a clearer Jayess proof-boundary split between
    compiler artifacts and executable/sanitizer proof

And the "not inherited from the comparison" side is concrete for the current
active compiler-side threads too:

- lowering boundaries:
  - Jayess is not inheriting any assumption that multiple emitted forms are
    acceptable if another compiler/runtime happens to tolerate them
- ownership propagation:
  - Jayess is not inheriting any assumption that ownership facts may remain
    informal or prose-only once they must affect cleanup/codegen behavior
- helper call shapes:
  - Jayess is not inheriting any assumption that helper-family contract sprawl
    is acceptable until unified by Jayess compiler/runtime work
- dynamic behavior lowering:
  - Jayess is not inheriting any assumption that equivalent language behavior
    may remain permanently split across many emitted patterns
- proof placement:
  - Jayess is not inheriting any assumption that compiler-proof and
    executable-proof responsibility can stay implicit or blurry

And the Jayess-owned evidence expected next from each active compiler-side
thread is concrete too:

- lowering boundaries:
  - lowering-test and backend-semantic evidence for one more canonical emitted
    form
- ownership propagation:
  - compile-time cleanup/proven-fresh evidence for one more mechanically
    propagated ownership distinction
- helper call shapes:
  - compiler/backend evidence that one helper family now uses a smaller emitted
    contract surface
- dynamic behavior lowering:
  - equivalent emitted/runtime behavior evidence across one newly canonicalized
    dynamic behavior
- proof placement:
  - an explicit checklist/docs/compiler-boundary artifact showing one clearer
    split between compiler proof and executable/sanitizer proof

In terms of present compiler-side checklist leverage, those threads are not
equally urgent:

- highest leverage:
  - ownership propagation
  - helper call shapes
  - lowering boundaries
- medium leverage:
  - dynamic behavior lowering
- lower immediate leverage, but still important for compiler clarity:
  - proof placement

So, unless a lower-leverage compiler-side thread is the only one with concrete
new Jayess checklist movement available, compiler-side QuickJS-inspired effort
should preferentially stay on the highest-leverage compiler threads first.
When effort does move to a lower-leverage compiler-side thread, the Jayess
reason should be explicit in the same way: it is the only thread with concrete
new movement available, or it unblocks a higher-leverage compiler-side thread
indirectly.
In that second case, the higher-leverage compiler-side Jayess gate or proof
surface being unblocked should be named explicitly.
If that higher-leverage compiler-side Jayess target cannot be named, the
lower-leverage compiler-side thread should not outrank the higher-leverage
threads.
Likewise, if the claim is that a lower-leverage compiler-side thread is the
only one with concrete new movement available, that concrete Jayess compiler
checklist movement should be named explicitly rather than left implicit.
If neither kind of named Jayess compiler justification exists, the
lower-leverage compiler-side thread should drop out of the active set instead
of competing for priority.
A dropped compiler-side QuickJS-inspired thread should only re-enter the active
set when it again has a named Jayess compiler bottleneck, named checklist
movement, or named higher-leverage compiler gate that it can now concretely
unblock.
When that happens, the Jayess-side compiler change that justified re-entry
should also be named explicitly: a new emitted-form gap, a new ownership
propagation gap, a new helper-family split, a new proof-boundary gap, or a new
concrete compiler checklist movement opportunity.
If the next concrete compiler-side Jayess artifact stops moving, the active
thread should be reprioritized or retired under the same compiler-side
active-set rules instead of staying alive on comparison momentum alone.
Because those compiler-side threads are ordered by current Jayess leverage, the
compiler-side ordering should also be recalculated whenever one of the named
compiler artifacts lands, one of the compiler bottlenecks narrows, or a new
Jayess-side compiler repro/proof gap appears.
When that compiler-side ordering changes, any existing exception that had
allowed a lower-leverage compiler-side thread to outrank a higher-leverage one
should be re-justified against the new Jayess compiler ordering or dropped.
At the same time, the named compiler-side bottleneck, next artifact, target
gate, retirement condition, non-inherited assumption, and expected evidence for
each still-active compiler-side thread should be refreshed together so the
compiler-side active-thread map stays coherent.
When a compiler-side Jayess artifact does land, the resulting checklist credit
should flow back into the relevant Jayess-owned compiler/lowering/codegen gate
rather than being tracked as a separate QuickJS-inspired compiler win. At the
same time, the corresponding active compiler-side bottleneck entry should be
removed, narrowed, or reclassified so the compiler-side comparison inventory
keeps matching the real Jayess compiler state.

Those Jayess-owned QuickJS-inspired areas are not a separate roadmap. They map
directly onto the existing runtime-hardening sections in this checklist:

- value helpers:
  - `Value ownership model`
- object/array/property behavior:
  - `Objects, arrays, and property storage`
- callback and call dispatch:
  - `Function call and callback dispatch`
- queues and async runtime state:
  - `Exceptions, promises, and async runtime state`
- host wrappers:
  - `Native handles and host resources`
- accounting:
  - `Memory accounting and torture testing`

Progress in those areas should only count when it advances the real Jayess
checklist gates attached to those sections: ownership classification, cleanup
rules, lifecycle contracts, sanitizer matrices, reproducer discipline, and the
broad `9.5` exit criteria. Improving the comparison note by itself is not
runtime progress.

The same rule applies to runtime decomposition work inspired by QuickJS:
splitting or reshaping Jayess runtime files only counts as progress when it
reduces an actual Jayess ownership-reasoning bottleneck or proof bottleneck,
not when it merely makes the file layout look more like another runtime.
In practice, that means a Jayess runtime split should earn credit only when it
produces a narrower documented ownership surface and a smaller proof obligation
for tests, probes, or sanitizer lanes.
It should also shrink the set of helper families whose ownership behavior still
has to be reasoned about as one broad mixed cluster.
And it should make helper-by-helper ownership classification easier to express
mechanically, not merely easier to describe in prose.
And it should reduce the number of places where Jayess codegen still has to rely
on heuristic cleanup reasoning or per-slice exception lists.
Ideally it should also move Jayess closer to one machine-readable ownership and
cleanup-exclusion mechanism instead of scattered family-by-family special cases.
And it should shrink one of the still-open broad proof surfaces called out by
the `9.5` gate map, not merely relocate the same uncertainty into different
files.

In practice, the current open-surface map lines up with the Jayess-owned
QuickJS-inspired areas like this:

- host-wrapper close coverage:
  - primarily blocks `host wrappers`
- async/host stress:
  - primarily blocks `queues and async runtime state`
  - and secondarily `host wrappers`
- exception-heavy flow:
  - primarily blocks `queues and async runtime state`
- property/array churn:
  - primarily blocks `object/array/property behavior`
- missing runtime-wide mechanical ownership classification:
  - primarily blocks `value helpers`
  - and secondarily `callback and call dispatch`

There is intentionally no separate "QuickJS compatibility" success track in
this roadmap. A QuickJS-inspired idea only counts when it closes a Jayess-owned
section gate or shrinks a Jayess-owned proof surface.
That also means prioritization should flow from Jayess's still-red gates and
open proof surfaces first, not from whichever runtime subsystem happens to have
the closest comparison point elsewhere.
And a QuickJS-inspired thread should only stay active when it can be traced to
one concrete next Jayess task: a helper classification task, a cleanup/lifetime
proof task, a host-wrapper lifecycle task, an async queue proof task, or an
accounting implementation task.
If a QuickJS-inspired thread no longer produces narrower Jayess proof surfaces
or concrete checklist movement, it should be retired rather than kept alive as
comparison work.
Likewise, if Jayess-native ownership evidence, executable behavior, or
sanitizer evidence contradicts a QuickJS-inspired assumption, the Jayess
evidence wins and the comparison thread should be rewritten or dropped.
Each surviving QuickJS-inspired thread should therefore name both sides
explicitly:

- what Jayess will define, prove, or implement in its own terms
- what Jayess is deliberately not inheriting from the comparison
- which concrete Jayess checklist section or gate that thread is expected to
  move next
- what Jayess-owned condition will retire that thread once it is satisfied
- what current Jayess bottleneck is keeping that thread alive right now

That means the useful unit is not "QuickJS has X subsystem". The useful unit is
"Jayess still has Y bottleneck in section Z, and this comparison only stays
alive while it helps close that bottleneck."

At the current checklist boundary, the active Jayess-owned QuickJS-inspired
threads are:

- value helpers:
  - blocked by the missing runtime-wide helper-by-helper ownership
    classification and machine-readable cleanup/exclusion mechanism
- object/array/property behavior:
  - blocked by missing lifetime-stress proof for property/array churn and
    alias-heavy container helpers
- callback and call dispatch:
  - blocked by the still-non-mechanical cleanup/classification path and the
    still-unproven ownership equivalence between fast and slow callback paths
- queues and async runtime state:
  - blocked by missing async/host sanitizer stress coverage and still-broad
    queue/exception proof surfaces
- host wrappers:
  - blocked by incomplete close-after-alias / double-close / forgotten-close
    regression coverage across wrapper families
- accounting:
  - blocked by missing counters and missing Jayess-facing accounting API
    surface, even though the target shape and hook points are now specified

The next concrete Jayess artifact expected from each active thread is also
clear at this boundary:

- value helpers:
  - a helper-by-helper ownership classification artifact or machine-readable
    ownership/cleanup-exclusion mechanism
- object/array/property behavior:
  - a narrower alias-heavy lifetime probe or sanitizer lane for property/array
    churn
- callback and call dispatch:
  - a cleanup/classification artifact or sanitizer-backed proof slice that
    narrows the fast/slow ownership gap
- queues and async runtime state:
  - an async/host stress lane or a narrower queue/exception lifetime proof
    artifact
- host wrappers:
  - a wrapper-family regression set for close-after-alias, double-close, or
    forgotten-close behavior
- accounting:
  - implemented counters, or a Jayess-facing accounting API surface built on
    those counters

And when those artifacts land, the checklist credit they are expected to move
is also concrete:

- value helpers:
  - `Value ownership model`
  - or the broad `9.5 no memory leaks for non-escaping values` /
    `9.5 no use-after-free is possible` gates if the classification becomes
    mechanically consumable
- object/array/property behavior:
  - `Objects, arrays, and property storage`
  - and secondarily the broad `9.5 pointer/reference validity is always
    preserved` gate
- callback and call dispatch:
  - `Function call and callback dispatch`
  - and secondarily the broad `9.5 no use-after-free is possible` /
    `9.5 no memory leaks for non-escaping values` gates
- queues and async runtime state:
  - `Exceptions, promises, and async runtime state`
  - and secondarily the broad async/host proof surfaces beneath `9.5`
- host wrappers:
  - `Native handles and host resources`
  - and secondarily the broad `9.5 no double-free is possible` /
    `9.5 pointer/reference validity is always preserved` gates
- accounting:
  - `Memory accounting and torture testing`

The Jayess-owned retirement condition for each currently active thread is also
concrete:

- value helpers:
  - helper ownership becomes mechanical enough that the remaining broad
    classification/cleanup-exclusion bottleneck is materially narrowed
- object/array/property behavior:
  - the alias-heavy property/array churn surface gains the missing lifetime
    stress proof slice
- callback and call dispatch:
  - the fast/slow ownership gap is materially narrowed by a real cleanup/proof
    artifact rather than only by documentation
- queues and async runtime state:
  - an async/host stress lane or narrower queue/exception proof artifact closes
    part of the currently broad async proof surface
- host wrappers:
  - wrapper-family close/after-close/double-close coverage materially reduces
    the still-open host-wrapper proof surface
- accounting:
  - real counters or a Jayess-facing accounting API surface exist, so the
    thread is no longer only target-shape planning

And the "not inherited from the comparison" side is concrete for the current
active threads too:

- value helpers:
  - Jayess is not inheriting any assumption that helper ownership can remain an
    implicit convention instead of a Jayess-classified runtime surface
- object/array/property behavior:
  - Jayess is not inheriting any assumption that a more elaborate external
    object model automatically supplies Jayess aliasing or snapshot rules
- callback and call dispatch:
  - Jayess is not inheriting any assumption that callback fast paths and
    fallback paths are ownership-equivalent until Jayess proves that itself
- queues and async runtime state:
  - Jayess is not inheriting any assumption that queue transfer or shutdown
    semantics are already safe without Jayess async proof lanes
- host wrappers:
  - Jayess is not inheriting any assumption that wrapper lifecycle behavior is
    already acceptable without Jayess close/after-close/double-close coverage
- accounting:
  - Jayess is not inheriting any assumption that an external runtime-style
    accounting story is useful until Jayess has its own counters and workload
    validation

And the Jayess-owned evidence expected next from each active thread is concrete
too:

- value helpers:
  - mechanical helper classification or cleanup-exclusion evidence consumable by
    codegen
- object/array/property behavior:
  - executable alias-heavy coverage plus a narrower lifetime-stress artifact
- callback and call dispatch:
  - cleanup-probe or sanitizer-backed evidence that narrows the fast/slow
    ownership gap
- queues and async runtime state:
  - executable async queue semantics plus a narrower async/host stress artifact
- host wrappers:
  - wrapper-family lifecycle regressions covering close-after-alias,
    after-close, or double-close behavior
- accounting:
  - implemented runtime counters and/or a Jayess-facing accounting API with
    workload validation

Because those active threads are ordered by current Jayess leverage, the order
should be recalculated whenever one of the named artifacts lands, one of the
bottlenecks narrows, or a new Jayess-side repro/proof gap appears.
When that ordering changes, any existing exception that had allowed a
lower-leverage thread to outrank a higher-leverage one should be re-justified
against the new Jayess ordering or dropped.
At the same time, the named bottleneck, next artifact, target gate, retirement
condition, non-inherited assumption, and expected evidence for each still-active
thread should be refreshed together so the active-thread map stays coherent.

In terms of present checklist leverage, those threads are not equally urgent:

- highest leverage on broad `9.5` gates:
  - value helpers
  - callback and call dispatch
  - host wrappers
  - queues and async runtime state
- medium leverage:
  - object/array/property behavior
- lower immediate leverage on `9.5`, but still important for runtime hardening:
  - accounting

So, unless a lower-leverage thread is the only one with concrete new checklist
movement available, QuickJS-inspired effort should preferentially stay on the
highest-leverage threads first.
When effort does move to a lower-leverage thread, the Jayess reason should be
explicit: it is the only thread with concrete new movement available, or it
unblocks a higher-leverage thread indirectly. In that second case, the higher-
leverage Jayess gate or proof surface being unblocked should be named
explicitly.
If that higher-leverage Jayess target cannot be named, the lower-leverage
thread should not outrank the higher-leverage threads.
Likewise, if the claim is that a lower-leverage thread is the only one with
concrete new movement available, that concrete Jayess checklist movement should
be named explicitly rather than left implicit.
If neither kind of named Jayess justification exists, the lower-leverage thread
should drop out of the active set instead of competing for priority.
A dropped QuickJS-inspired thread should only re-enter the active set when it
again has a named Jayess bottleneck, named checklist movement, or named
higher-leverage gate that it can now concretely unblock.
When that happens, the Jayess-side change that justified re-entry should also be
named explicitly: a new repro, a new proof gap, a new helper surface, or a new
concrete checklist movement opportunity.
For an active QuickJS-inspired thread to stay useful, it should also identify
the next concrete Jayess artifact expected to move: a helper classification
table, a cleanup/probe test, a sanitizer lane, a host-wrapper regression set,
or a runtime accounting/API change.
If that Jayess artifact stops moving, the thread should be reprioritized or
retired under the same active-set rules instead of staying alive as comparison
momentum alone.
When that artifact does land, it should be credited back to the relevant
Jayess-owned `9.6` section or `9.5` gate rather than tracked as a separate
QuickJS-inspired accomplishment.
At the same time, the corresponding active QuickJS-inspired bottleneck entry
should be removed, narrowed, or reclassified so the comparison inventory keeps
matching the real Jayess state.

## Current Runtime Split

The runtime is already moving away from a single-file implementation:

- [runtime/jayess_runtime.c](/home/remote-desktop/Desktop/it/jayess/jayess-go/runtime/jayess_runtime.c):
  remaining core dispatch, generic dynamic behavior, and some stdlib glue
- [runtime/jayess_runtime_values.c](/home/remote-desktop/Desktop/it/jayess/jayess-go/runtime/jayess_runtime_values.c):
  boxed value allocation/freeing and value cleanup rules
- [runtime/jayess_runtime_errors.c](/home/remote-desktop/Desktop/it/jayess/jayess-go/runtime/jayess_runtime_errors.c):
  exceptions, error objects, stack reporting
- [runtime/jayess_runtime_strings.c](/home/remote-desktop/Desktop/it/jayess/jayess-go/runtime/jayess_runtime_strings.c):
  string conversion and text helpers
- [runtime/jayess_runtime_collections.c](/home/remote-desktop/Desktop/it/jayess/jayess-go/runtime/jayess_runtime_collections.c):
  objects, arrays, maps/sets, collection helpers
- [runtime/jayess_runtime_bigint.c](/home/remote-desktop/Desktop/it/jayess/jayess-go/runtime/jayess_runtime_bigint.c):
  bigint operations
- [runtime/jayess_runtime_typed_arrays.c](/home/remote-desktop/Desktop/it/jayess/jayess-go/runtime/jayess_runtime_typed_arrays.c):
  `ArrayBuffer`, typed arrays, `DataView`
- [runtime/jayess_runtime_fs.c](/home/remote-desktop/Desktop/it/jayess/jayess-go/runtime/jayess_runtime_fs.c):
  filesystem helpers
- [runtime/jayess_runtime_network.c](/home/remote-desktop/Desktop/it/jayess/jayess-go/runtime/jayess_runtime_network.c):
  sockets, HTTP, TLS, networking glue
- [runtime/jayess_runtime_process.c](/home/remote-desktop/Desktop/it/jayess/jayess-go/runtime/jayess_runtime_process.c):
  process, workers, signals, scheduling-adjacent glue
- [runtime/jayess_runtime_crypto.c](/home/remote-desktop/Desktop/it/jayess/jayess-go/runtime/jayess_runtime_crypto.c):
  crypto helpers
- [runtime/jayess_runtime_streams.c](/home/remote-desktop/Desktop/it/jayess/jayess-go/runtime/jayess_runtime_streams.c):
  stream/evented wrapper behavior

Shared concrete runtime structs live in:

- [runtime/jayess_runtime_internal.h](/home/remote-desktop/Desktop/it/jayess/jayess-go/runtime/jayess_runtime_internal.h)

## Remaining `runtime.c` Classification

The remaining generic/dynamic operations in `runtime.c` can now be classified
into four buckets.

### 1. Value Core

These are runtime primitives that support general execution and are not tied to
one stdlib area:

- singleton values such as `null` / `undefined`
- argv and sleep helpers
- generic stringify / add / scalar conversion glue
- symbol singletons and registry-adjacent helpers
- scheduler globals and microtask/timer scaffolding

This bucket should eventually be the smallest stable core because it is the
hardest part to reason about for ownership.

#### Current ownership contract for core value constructors

The core value-constructor family already falls into a few clear ownership
classes.

Immortal/static constructors:

- `jayess_value_null()`
- `jayess_value_undefined()`
- `jayess_value_from_bool(...)`
  - return immortal/static singleton boxes
  - callers must never treat them as freshly owned heap values
- `jayess_value_from_number(...)`
  - returns immortal/static singleton boxes for the current small-number pool
    (`-1` through `16`)
  - otherwise returns a fresh owned boxed number
- `jayess_value_from_static_string(...)`
  - returns an interned/static string box from the runtime static-string table
  - repeated calls for equal text return the same immortal/static boxed string

Fresh owned boxed constructors:

- `jayess_value_from_string(...)`
- `jayess_value_from_bigint(...)`
- `jayess_value_from_symbol(...)`
- `jayess_value_from_object(...)`
- `jayess_value_from_array(...)`
- `jayess_value_from_args(...)`
  - return fresh owned boxed values
  - the wrapped payload is not shared through an implicit intern table
  - object/array wrappers keep the provided container pointer rather than
    cloning the container

Transferred/consumed input constructors:

- `jayess_value_from_owned_string(...)`
  - returns a fresh owned string box
  - consumes the provided heap string pointer as the box payload
  - the caller must not free or reuse that string after passing it in

Derived consequences:

- `jayess_value_free_unshared(...)` is valid only for constructors that really
  returned a fresh owned box
- the same constructor name can have mixed ownership behavior when it uses
  singleton fast paths, especially `jayess_value_from_number(...)`
- codegen and runtime helper callers must not collapse “scalar constructor” into
  one ownership class; `bool`, `null`, `undefined`, interned static strings,
  pooled small numbers, and heap-allocated strings/numbers are not equivalent

### 2. Object / Array / Property Semantics

These are the generic dynamic-language operations that define object-model
behavior:

- member/index access and deletion
- computed property keys
- object rest / keys / values / entries
- array length, slicing, and iterable bridging
- special method exposure for dynamic wrapper objects
- property-backed behavior for sockets, streams, watchers, workers, and other
  host wrappers

This bucket is the closest Jayess analogue to the “object/property semantics”
concerns that QuickJS solves with shapes and property metadata, even though
Jayess should keep its own simpler ownership rules.

#### Current ownership contract for object / array / member helpers

The object/array helper family is mostly **aliasing container storage** rather
than deep-copying storage.

- `jayess_object_new()` / `jayess_array_new()`
  - return **fresh owned containers**
  - do not pre-populate child values
- `jayess_value_from_object(...)` / `jayess_value_from_array(...)`
  - return **fresh owned boxes**
  - wrap the provided container pointer without cloning it
  - the box and wrapped container should be treated as one ownership unit for
    later unshared cleanup

Container write/update helpers:

- `jayess_object_set_value(...)` / `jayess_object_set_key_value(...)`
- `jayess_array_set_value(...)`
- `jayess_array_push_value(...)`
- `jayess_value_set_member(...)`
- `jayess_value_set_computed_member(...)`
- `jayess_value_set_index(...)`
- `jayess_value_set_dynamic_index(...)`
  - **alias** the stored `jayess_value *`
  - do not deep-clone the stored value
  - do not consume the caller's value box
  - string property names are copied as text into entry metadata
  - symbol property keys are stored by aliasing the symbol value pointer

Container read helpers:

- `jayess_object_get(...)` / `jayess_object_get_key_value(...)`
- `jayess_array_get(...)`
  - return **borrowed/aliased stored element references**
  - do not allocate fresh wrappers
- `jayess_value_get_member(...)` / `jayess_value_get_dynamic_index(...)`
  - are **mixed-result helpers**
  - for ordinary stored object/function properties, they return borrowed/aliased
    stored values
  - for virtual/dynamic-language properties such as:
    - string/typed-array lengths
    - boxed method values
    - boxed string descriptions
    - other runtime-computed wrappers
    they return **fresh owned values**
  - missing stored object/function properties generally produce the immortal
    `undefined` singleton on the property path
- `jayess_value_get_index(...)`
  - returns borrowed/aliased array elements on plain arrays
  - returns fresh boxed numeric wrappers for typed-array reads

Current codegen-side cleanup boundary for these helpers:

- when codegen has to box a temporary scalar/object/string receiver only to call
  `jayess_value_get_member(...)`, `jayess_value_get_dynamic_index(...)`, or
  `jayess_value_array_length(...)`, that temporary receiver wrapper is now part
  of the proven discarded-cleanup slice
- this already covers:
  - member access
  - index access
  - optional member/index access
  - temporary `.length` wrapper access
- so the current known member/index wrapper leak is closed at the statement-
  exit cleanup boundary, even though the underlying read helpers themselves are
  still mixed borrowed-versus-fresh result helpers

Delete/remove helpers:

- `jayess_object_delete(...)` / `jayess_object_delete_key_value(...)`
- `jayess_value_delete_member(...)` / `jayess_value_delete_dynamic_index(...)`
- `jayess_array_remove_at(...)`
- `jayess_array_pop_value(...)` / `jayess_array_shift_value(...)`
  - remove container metadata or slots
  - do **not** free aliased stored value boxes just because the container no
    longer references them
  - `pop` / `shift` return the removed element by transfer/preservation of the
    existing pointer rather than boxing a clone

Fresh container-building helpers:

- `jayess_object_keys(...)` / `jayess_value_object_keys(...)`
  - return a **fresh owned array**
  - string keys are fresh boxed strings
- `jayess_value_object_symbols(...)`
  - returns a **fresh owned array**
  - symbol entries are aliased symbol values, not cloned symbols
- `jayess_array_slice_values(...)` / `jayess_value_array_slice(...)`
  - return a **fresh owned array container**
  - elements are aliased from the source container
- `jayess_value_object_rest(...)`
  - returns a **fresh owned object container**
  - copied properties still alias the original stored values
- `jayess_value_object_values(...)`
  - returns a **fresh owned array container**
  - elements alias the values read from the source object
- `jayess_value_object_entries(...)`
  - returns a **fresh owned outer array**
  - each pair is a fresh owned inner array
  - pair contents alias the source key/value boxes or freshly boxed key strings

Current array-helper aliasing map:

- growth/mutation helpers
  - `jayess_array_push_value(...)`
  - `jayess_array_unshift_value(...)`
  - `jayess_array_set_value(...)`
  - alias inserted values; do not deep-clone stored elements
- transfer/removal helpers
  - `jayess_array_pop_value(...)`
  - `jayess_array_shift_value(...)`
  - return the preserved removed pointer; they do not clone it before return
- shallow-copy helpers
  - `jayess_array_slice_values(...)`
  - plain-array slices produce fresh array containers with aliased element
    pointers
  - typed-array slices instead allocate fresh backing storage and fresh typed
    wrappers
- concat-style helpers
  - internal helpers such as `jayess_array_concat(...)` and
    `jayess_array_concat_bound_args_owned(...)`
  - plain concat aliases ordinary element pointers into a fresh container
  - bound-arg concat is narrower: it clones-or-preserves according to the
    bound-arg ownership rules instead of generic array aliasing

What is already true versus still unproven:

- the helper-level aliasing behavior above is now documented explicitly
- object spread/property enumeration and typed-array slice/copy semantics already
  have executable semantic coverage
- what is still missing is a broad lifetime-stress proof that all of these
  growth/mutation/copy helpers preserve lifetime invariants under aliasing

What this means for codegen and `9.5` work:

- storing into objects/arrays extends lifetime by **adding another aliasing
  container reference**, not by cloning the stored value graph
- removing a property or array slot must not be treated as permission to free
  the removed value unless some other ownership rule proves it fresh and
  unaliased
- member/index helpers cannot be treated as uniformly borrowed or uniformly
  fresh; cleanup decisions must distinguish stored-property reads from virtual
  wrapper reads
- array/object copy-style helpers currently create **fresh containers with
  aliased elements**, which is the key invariant to preserve when broad `9.5`
  rows are eventually checked

#### Current internal representation in QuickJS-comparable terms

QuickJS solves object-model pressure with dedicated object metadata, property
tables, shapes, and specialized array/typed-array storage. Jayess is currently
much simpler, but the same architectural questions still apply.

Current Jayess representation:

- `jayess_value`
  - small tagged box over:
    - string pointer
    - number
    - bigint pointer
    - bool
    - object pointer
    - array pointer
    - function pointer
    - symbol pointer
- `jayess_object`
  - singly linked property list via `head` / `tail`
  - each property entry stores:
    - string key text, or
    - symbol key value pointer
    - stored value pointer
  - also carries host/runtime sidecar fields directly on the object:
    - promise dependents
    - stream file
    - socket handle
    - native handle
- `jayess_array`
  - flat growable `jayess_value **values` buffer plus `count`
  - no separate capacity field or hole/shape metadata
- `jayess_function`
  - direct C callee pointer
  - env pointer
  - static metadata (`name`, `class_name`, `param_count`, `has_rest`)
  - ordinary property object
  - bound `this`
  - bound-arg array
- `jayess_symbol`
  - monotonically assigned runtime id
  - optional description text

How to compare that to QuickJS without copying it:

- Jayess objects currently behave more like **linked property bags** than shape-
  based objects
- Jayess arrays are **flat aliasing pointer vectors**, not specialized dense
  array records with independent element ownership rules
- Jayess functions bundle callable metadata, properties, environment, and bound
  args in one runtime record instead of splitting those concerns across a richer
  object engine
- Jayess host wrappers currently reuse ordinary object storage and embed host
  sidecar state directly on `jayess_object`, whereas QuickJS tends to separate
  opaque class payloads and generic property storage more sharply

Why this matters for `9.5`:

- there is no separate shape/property metadata layer currently absorbing aliasing
  complexity; the linked entries themselves are the storage contract
- object properties and array elements are pointer aliases, so lifetime bugs are
  about **who owns the pointed-to box**, not about copy-on-write metadata
- host-resource sidecar fields on `jayess_object` mean object-lifetime and
  native-handle lifetime are still coupled more tightly than they would be in a
  more segmented runtime design

### 3. Function / Call Dispatch

These are the helpers that make compiled code able to call Jayess functions
through a stable ABI:

- `jayess_value_call_with_this(...)`
- fixed-arity fast-path helpers such as `jayess_call_function2(...)` through
  `jayess_call_function13(...)`
- bind/apply/merge-bound-args helpers
- method boxing via `jayess_value_from_function(...)`
- constructor-return glue and callback helper dispatch

This area is directly relevant to section `9.5`, because callback results,
bound-arg ownership, and aliasing bugs have all shown up here.

#### Current ownership contract for function helpers

The function-helper family now has a documented ownership boundary. This is not
the final full-runtime classification yet, but it is explicit enough to guide
codegen and future `9.5` work.

- `jayess_value_from_function(...)`
  - returns a **fresh owned function box**
  - allocates a fresh `jayess_function`
  - allocates a fresh `properties` object
  - allocates a fresh empty `bound_args` array
  - borrows `env`
  - uses immortal/static metadata pointers such as `name` / `class_name`
- `jayess_call_function(...)` through `jayess_call_function13(...)`
  - return **whatever the callee returns**
  - do not wrap the result in an extra ownership layer
  - borrow the callback and argument references for the duration of the call
  - are ABI-specialized shims over the same dynamic call semantics
- `jayess_value_call_with_this(...)`
  - is the generic call path
  - returns **whatever the callee returns**
  - borrows callback, `this`, and argument references for the duration of the call
  - should match the same ownership result as the fixed-arity fast paths
- `jayess_value_bind(...)`
  - returns a **fresh owned bound-function box**
  - allocates a fresh `jayess_function`
  - allocates a fresh `properties` object
  - allocates a fresh `bound_args` container
  - keeps `bound_this` as a borrowed/aliased reference
  - stores bound args without borrowing the temporary bind-array wrapper itself
  - currently clones primitive bound values and aliases non-primitive bound values
- `jayess_value_merge_bound_args(...)`
  - returns a **fresh owned array container**
  - does not return the incoming tail array by alias
  - currently clones primitive bound values and aliases non-primitive values
- `jayess_value_function_bound_arg(...)`
  - returns a **borrowed/aliased element reference**
  - does not allocate a fresh boxed copy
- `jayess_value_function_bound_this(...)`
  - returns a **borrowed/aliased `this` reference**
- `jayess_value_function_env(...)`
  - returns a **borrowed/aliased environment reference**
- metadata helpers such as:
  - `jayess_value_function_ptr(...)`
  - `jayess_value_function_param_count(...)`
  - `jayess_value_function_has_rest(...)`
  - `jayess_value_function_bound_arg_count(...)`
  - `jayess_value_function_class_name(...)`
  - return scalar or metadata views and do not allocate ownership-bearing value boxes

What this means for codegen:

- cleanup may only assume ownership of values returned by helpers explicitly
  documented as **fresh owned**
- call helpers must be treated as **callee-result preserving**, not inherently
  fresh
- bind/merge helpers create fresh containers, but not necessarily deep-cloned
  object graphs
- bound-arg accessors are borrowing helpers and must never drive cleanup by
  themselves

#### Current fixed-arity callback fast-path policy

The callback fast-path fan-out now has a real, documented boundary:

- direct callback helpers exist for:
  - zero pre-bound args through twelve pre-bound args
  - that means the specialized runtime shims currently cover
    `jayess_call_function(...)` through `jayess_call_function13(...)`
  - the highest specialized callback shape is therefore:
    - twelve pre-bound args
    - plus one iterated item arg
- beyond twelve pre-bound args, codegen does **not** keep extending the direct
  call ABI in-place
  - it falls back to the generic `apply` path
  - that fallback materializes a boxed arg-array wrapper and routes through
    `emitApplyFromValues(...)`

What this means for the roadmap:

- the current policy is **capped at a deliberate maximum with a required
  fallback path**
- it is **not yet generated from one source of truth**
- ownership work still has to keep the capped direct paths and the generic
  fallback path behaviorally aligned, but the existence of the cap/fallback
  boundary is no longer implicit

#### Current ownership contract for constructor-return helpers

Jayess constructors have one runtime ownership helper today:

- `jayess_value_constructor_return(self, value)`
  - preserves current constructor semantics: explicit `return value` wins over
    the synthetic `__self` object
  - if `value != NULL` and `value != self`:
    - treats `self` as a **fresh owned synthetic constructor object**
    - frees that synthetic `self`
    - returns `value` unchanged
  - if `value == self`:
    - returns the same `self` box unchanged
    - does not free and re-box it
  - if `self != NULL` and `value == NULL`:
    - returns `self`
  - if `self == NULL` and `value != NULL`:
    - returns `value`
  - if both are `NULL`:
    - returns the immortal/static `undefined` singleton

What this means for codegen:

- `__jayess_constructor_return(__self, expr)` is a **constructor-only
  ownership boundary**, not a generic freshness marker
- the helper consumes the synthetic constructor `self` only when an alternate
  explicit return value replaces it
- the returned `value` keeps its original ownership class; the helper does not
  deep-clone or fresh-box aliasing return values
- fresh-return analysis may treat
  `__jayess_constructor_return(__self, freshExpr)` as fresh because the helper
  preserves the ownership class of `freshExpr`, not because constructor returns
  are globally fresh by default

Checklist-shaped summary of that same contract:

- implicit `__self`
  - `__jayess_constructor_return(__self, NULL)` returns the synthetic `self`
    unchanged
- fresh alternate return
  - `__jayess_constructor_return(__self, freshValue)` frees the synthetic
    `self` and forwards `freshValue` unchanged
- aliased alternate return
  - `__jayess_constructor_return(__self, aliasedValue)` also frees the
    synthetic `self` and forwards `aliasedValue` unchanged

So constructor-return ownership is currently uniform in one precise sense:

- all constructor exits are normalized through the same helper
- the helper always decides between:
  - keep `self`
  - or discard `self` and forward the explicit return value
- the helper does not change the ownership class of the explicit return value;
  it only resolves what happens to the synthetic constructor object

## Closures, Bound Functions, and Environments

Jayess closures and bound functions currently share one practical ownership
contract across compiler/lowering/runtime boundaries.

### Current closure/environment contract

- `jayess_value_from_function(...)`
  - creates a fresh owned function box
  - stores `env` by **borrowed/aliased reference**
  - does not deep-clone the closure environment
- `jayess_value_function_env(...)`
  - returns that same borrowed/aliased environment reference
- `jayess_value_bind(...)`
  - preserves the original function `env` by alias
  - does not clone or replace the closure environment when creating a bound
    function wrapper
- `jayess_value_function_bound_this(...)`
  - returns borrowed/aliased `this`
- `jayess_value_function_bound_arg(...)`
  - returns borrowed/aliased bound-arg element references

Compiler/lifetime consequence:

- closure environments and captured values must already be placed in an
  escaping/extended-lifetime representation before the runtime function box
  points at them
- the runtime assumes the closure environment pointer remains valid for as long
  as the function value remains live
- this means closure safety is achieved by:
  - compiler/lifetime escape analysis deciding what must outlive the lexical
    stack slot
  - runtime function boxes borrowing that already-escaped environment

### Current bound-arg storage contract

- `jayess_value_bind(...)` accepts an optional array-like wrapper of new bound
  args
- it does **not** keep borrowing that temporary wrapper past the bind call
- instead it builds a fresh owned `bound_args` container through
  `jayess_array_concat_bound_args_owned(...)`
- `jayess_value_merge_bound_args(...)` does the same when preparing call-time
  merged args

Element-level behavior inside that owned container is still mixed:

- primitive-like bound values are cloned into fresh boxes:
  - strings
  - non-pooled numbers
  - bigints
  - symbols
- singleton-like values stay singleton:
  - booleans
  - `null`
  - `undefined`
  - pooled small numbers
- object/array/function bound values remain aliased pointers

What this means:

- the temporary wrapper array is not part of the long-term bound-function
  ownership graph
- the long-term ownership graph is:
  - fresh bound-function box
  - fresh `jayess_function`
  - fresh owned `bound_args` container
  - borrowed/aliased `env`
  - borrowed/aliased `bound_this`
  - cloned-or-aliased bound arg elements depending on value kind

This uses the same ownership vocabulary as the rest of the runtime roadmap:

- fresh owned
- borrowed/aliased
- immortal/static
- transferred/consumed

## Binary Data, Views, and Backing Storage

Jayess typed-array helpers already have a concrete storage model. It is simpler
than QuickJS, but the ownership boundary is specific enough to document now.

### Current `ArrayBuffer` / typed-array / `DataView` contract

- `jayess_std_array_buffer_new(...)`
  - returns a fresh owned object box for the buffer
  - creates a fresh owned internal byte array stored under `__jayess_bytes`
  - that byte storage is currently represented as a normal `jayess_array` of
    boxed number bytes
- `jayess_std_shared_array_buffer_new(...)`
  - returns a fresh owned object box
  - creates a fresh owned internal byte array
  - also creates a native shared-bytes sidecar state and stores it on
    `object->native_handle`
- `jayess_std_typed_array_new(kind, source)`
  - returns a fresh owned typed-array object box
  - if `source` is an existing `ArrayBuffer` or `SharedArrayBuffer`:
    - aliases that existing buffer object
    - aliases its underlying `__jayess_bytes` storage
  - if `source` is another typed array, a plain array, or a numeric length:
    - allocates a fresh buffer
    - typed-array contents are copied into that fresh backing storage as needed
  - stores both:
    - aliased/fresh `buffer`
    - aliased `__jayess_bytes`
- `jayess_std_data_view_new(buffer)`
  - returns a fresh owned `DataView` object box
  - if given an existing `ArrayBuffer`/`SharedArrayBuffer`, it aliases:
    - the buffer object
    - the underlying `__jayess_bytes` array
  - otherwise it creates a fresh zero-length buffer and aliases that
- `jayess_std_typed_array_slice_values(...)`
  - returns a fresh owned typed-array result
  - allocates fresh backing storage for the slice result
  - copies numeric contents into that fresh storage
- scalar typed-array and `DataView` reads:
  - return fresh boxed numeric wrappers
  - do not expose raw backing-storage pointers directly into codegen
- scalar typed-array and `DataView` writes:
  - mutate aliased backing storage in place
  - do not create a detached copy first

What this means for ownership:

- buffer/view relationships are currently **aliasing views over one backing byte
  array**, not copy-on-write views
- slices produce fresh result storage, but ordinary view creation does not
- `buffer` object identity and `__jayess_bytes` storage identity can outlive one
  particular typed-array or `DataView` wrapper as long as some aliasing wrapper
  still points at them
- shared-buffer state adds a native sidecar, so byte-storage lifetime and native
  handle lifetime are coupled for `SharedArrayBuffer`

QuickJS comparison:

- QuickJS has a richer distinction between object headers, array buffers, typed
  array views, and detached/shared storage states
- Jayess currently reuses ordinary object storage plus `__jayess_bytes` slots
  and, for shared buffers, a native sidecar pointer
- the ownership question is still the same: who owns backing bytes, who merely
  views them, and which operations allocate fresh storage versus alias existing
  storage

### Current generic native-handle wrapper contract

Jayess also has a generic native-handle wrapper family separate from the
higher-level stream/socket/server wrappers.

Creation:

- `jayess_value_from_native_handle(kind, handle)`
  - returns a fresh owned wrapper object box
  - stores the raw native handle pointer directly on `object->native_handle`
  - does not install a finalizer
  - does not create a `closed` property by default
- `jayess_value_from_managed_native_handle(kind, handle, finalizer)`
  - returns a fresh owned wrapper object box
  - allocates a fresh managed-handle sidecar
  - stores:
    - raw handle pointer
    - finalizer callback
    - closed flag
  - exposes `closed = false` at wrapper creation time

Lookup / borrowing:

- `jayess_value_as_native_handle(value, kind)`
  - returns the raw native handle pointer by borrowed view
  - for managed wrappers, returns `NULL` once the wrapper is closed
  - performs kind checking but does not transfer ownership of the handle
- `jayess_expect_native_handle(...)`
  - is the checked borrowing form of the same contract

Close / finalize:

- `jayess_value_clear_native_handle(value)`
  - clears the stored pointer without running a finalizer
  - marks managed wrappers closed
- `jayess_value_close_native_handle(value)`
  - unmanaged wrapper:
    - clears the stored pointer
    - returns success
    - no finalizer exists to call
  - managed wrapper:
    - runs the finalizer at most once if the handle is still live
    - nulls the handle pointer
    - marks the wrapper closed
    - exposes `closed = true`
- `jayess_object_free_unshared(...)`
  - if the object is a managed native-handle wrapper and the managed handle is
    still live, it runs the finalizer during object destruction
  - frees the managed sidecar afterward

Double-close safety:

- managed wrappers are explicitly one-shot:
  - repeated `jayess_value_close_native_handle(...)` calls after closure return
    failure and do not rerun the finalizer
  - destruction after explicit close also does not rerun the finalizer because
    the managed sidecar is already marked closed with a `NULL` handle
- unmanaged wrappers are only pointer-clearing wrappers:
  - duplicate close just clears/keeps a `NULL` pointer
  - there is no runtime finalizer protection because the runtime never owned a
    finalizer there in the first place

Aliasing through object properties and containers:

- native-handle wrappers are ordinary Jayess object boxes
- storing them in objects/arrays/functions therefore aliases the same wrapper
  object pointer; it does not clone the underlying native handle or its managed
  sidecar
- close/clear on one alias is visible through every alias that still points at
  the same wrapper object

### Current host-wrapper ownership map

Beyond the generic native-handle helpers, Jayess has several concrete wrapper
families that store live host resources directly on runtime objects.

Common rule across these wrappers:

- creation allocates one fresh Jayess object wrapper
- the live host handle/state is stored on fields of that wrapper object
- method calls borrow the wrapper and operate on the stored handle/state in
  place
- aliasing the wrapper through objects/arrays/functions aliases the same
  underlying host resource
- close/terminate paths clear the stored handle/state on that same wrapper and
  mark `closed` (or equivalent) on the wrapper object

#### File streams

- `ReadStream` / `WriteStream`
  - live host resource lives in `object->stream_file`
  - creator:
    - `jayess_std_fs_create_read_stream(...)`
    - `jayess_std_fs_create_write_stream(...)`
  - read/write/end/close methods borrow the wrapper and use that `FILE *`
  - explicit close/end:
    - `jayess_std_read_stream_close_method(...)`
    - `jayess_std_write_stream_end_method(...)`
  - closing:
    - `fclose(...)`
    - clears `stream_file`
    - marks `closed = true`

#### Filesystem watchers

- `Watcher`
  - live watcher state lives in `object->native_handle`
  - creator:
    - `jayess_std_fs_watch(...)`
  - polling/event methods borrow the wrapper and use that watcher state
  - explicit close:
    - `jayess_std_fs_watch_close_method(...)`
  - closing:
    - frees watcher state
    - clears `native_handle`
    - marks `closed = true`

#### Workers

- `Worker`
  - live worker state lives in `object->native_handle`
  - creator:
    - `jayess_std_worker_create(...)`
  - `postMessage` / `receive` borrow the wrapper and interact with the same
    queue/thread state
  - explicit termination:
    - `jayess_std_worker_terminate_method(...)`
  - termination:
    - signals thread shutdown
    - joins/destroys synchronization state
    - frees queued messages/state
    - clears `native_handle`
    - marks `closed = true`

#### Sockets and datagram sockets

- `Socket` / `DatagramSocket`
  - transport handle lives in `object->socket_handle`
  - optional protocol/TLS sidecar state may also live in `object->native_handle`
  - creators:
    - `jayess_std_socket_value_from_handle(...)`
    - `jayess_std_datagram_socket_value_from_handle(...)`
    - TLS socket constructors also attach protocol state through
      `object->native_handle`
  - read/write/send/receive methods borrow the wrapper and use the stored live
    handle/state
  - explicit close:
    - `jayess_std_socket_close_method(...)`
  - closing:
    - shuts down and closes `socket_handle`
    - clears the socket handle field
    - closes/clears native sidecar state where applicable
    - marks `closed = true`
    - emits close events on the same wrapper

#### Servers

- `Server`
  - listening socket lives in `object->socket_handle`
  - creators:
    - server/listen constructors that eventually call
      `jayess_std_net_listen(...)`
  - `accept` / `acceptAsync` borrow the server wrapper and use that listening
    socket
  - accepted client connections create new socket wrapper objects that take over
    ownership of the accepted client handle
  - explicit close:
    - `jayess_std_server_close_method(...)`
  - closing:
    - shuts down and closes the listening handle
    - clears `socket_handle`
    - marks `closed = true`
    - marks `listening = false`

#### HTTP body streams and HTTP servers

- `HttpBodyStream`
  - may depend on:
    - an aliased socket wrapper for transport closure
    - native protocol/body state in `object->native_handle`
  - explicit close:
    - `jayess_std_http_body_stream_close_method(...)`
  - closing:
    - marks stream ended
    - closes the associated socket path if present
    - clears/frees native body state
- HTTP server wrapper
  - high-level HTTP server state lives in `object->native_handle`
  - may also own an aliased backend `Server` wrapper in that state
  - explicit close:
    - `jayess_std_http_server_close_method(...)`
  - closing:
    - marks HTTP server state closed
    - closes the backend server wrapper if present
    - marks `closed = true`

Why this matters for `9.5`:

- these wrappers are not detached handles floating outside the value model;
  they are ordinary Jayess objects with live host-resource fields attached
- any alias to the wrapper aliases the same host resource
- “who owns the handle” therefore means “which wrapper object currently stores
  the authoritative live handle/state field,” and close paths must update that
  wrapper in place so all aliases observe the transition

## Promises, Microtasks, and Worker Queues

QuickJS is a useful comparison point here because it also has to define queue
ownership precisely across promises, async I/O, and worker messaging. Jayess
should keep its own simpler ownership vocabulary, but the same discipline
applies: every queued value needs one explicit answer to "borrowed alias,
cloned payload, or fresh owned wrapper?"

### Current promise-state ownership contract

Jayess promises are ordinary object wrappers with promise-specific metadata
stored on the wrapper itself.

- `jayess_std_promise_pending(...)`
  - allocates one fresh promise object wrapper
  - initializes:
    - `__jayess_promise_state`
    - `__jayess_promise_value`
    - `object->promise_dependents`
- `jayess_std_promise_resolve(...)` / `jayess_std_promise_reject(...)`
  - allocate the same kind of fresh wrapper, then settle it immediately
- `jayess_promise_settle(promise, state, value)`
  - mutates the existing promise wrapper in place
  - stores `state` and `value` on that wrapper by ordinary Jayess object-slot
    assignment rules
  - this means the stored promise value is currently an aliased Jayess value,
    not a deep-cloned payload
  - detaches the dependent-task list from `object->promise_dependents` and
    schedules dependents from that detached list

Current consequence:

- settled promise values live exactly as long as the promise wrapper and any
  aliases to that same wrapper keep them reachable
- settlement does not currently create a deep ownership boundary around the
  stored fulfillment/rejection value

### Current scheduler queue ownership map

Jayess currently has three main scheduler queues:

- `promise_callbacks`
- `timers`
- `io_pending` / `io_completions`

The queue nodes themselves are fresh runtime records (`jayess_microtask`), but
most Jayess values stored inside those records are borrowed/aliased pointers to
existing Jayess values rather than cloned copies.

#### Promise callback queue

- `jayess_enqueue_microtask(...)` and `jayess_enqueue_promise_task(...)`
  - allocate one fresh `jayess_microtask`
  - store aliased pointers for:
    - `source`
    - `result`
    - `on_fulfilled`
    - `on_rejected`
  - append that task node to `promise_callbacks`
- `jayess_run_microtask(...)`
  - borrows those stored pointers while running the task
  - may create fresh intermediate results during callback execution
  - settles `task->result` or re-enqueues follow-up work
- queue ownership rule:
  - the queue owns the task node
  - the queue does not own cloned copies of the Jayess values inside that node
  - correctness therefore depends on those values already being in an escaped
    state suitable for deferred use

#### Timer and async I/O queues

- `jayess_enqueue_timer_task(...)`
  - allocates a fresh task node
  - stores the callback as an aliased Jayess function value
- `jayess_enqueue_sleep_async_task(...)`
  - allocates a fresh task node
  - stores aliased `result` and `value`
- `jayess_enqueue_fs_read_file_task(...)`
  - allocates a fresh task node
  - stores aliased `result`, `path`, and `encoding`
- `jayess_enqueue_fs_write_file_task(...)`
  - allocates a fresh task node
  - stores aliased `result`, `path`, and `content`
- `jayess_enqueue_socket_read_task(...)`
  - allocates a fresh task node
  - stores aliased `result`, `socket`, and `size_value`
- `jayess_enqueue_socket_write_task(...)`
  - allocates a fresh task node
  - stores aliased `result`, `socket`, and `value`
- `jayess_enqueue_server_accept_task(...)`
  - allocates a fresh task node
  - stores aliased `result` and `server`
- `jayess_enqueue_http_request_task(...)`
  - allocates a fresh task node
  - stores aliased `result` and `options`

Current queue rule for these async helpers:

- the scheduler owns the task record and any native worker-side buffers it
  allocates for execution
- the embedded Jayess values remain aliased references to pre-existing runtime
  values until the task is completed or discarded
- completion may attach a fresh `worker_result` value to the task before it is
  delivered back through `jayess_run_microtask(...)`

#### Event-stream / watcher-style queue note

Jayess does not currently have one separate generic "event stream values" queue
type parallel to the worker message queues. Instead:

- watcher, socket, HTTP, timer, and async file operations currently route
  deferred work through the scheduler task queues above
- wrapper-local native state may accumulate host-side event buffers, but queued
  Jayess values still enter the runtime through fresh task nodes plus aliased
  wrapper/result pointers unless a subsystem explicitly clones them

### Current worker queue ownership contract

Worker message queues are stricter than the promise/timer queues because they
cross thread boundaries.

#### Inbound `postMessage(...)`

- `jayess_std_worker_post_message_method(...)`
  - first clones the supplied Jayess value with `jayess_worker_clone_value(...)`
  - wraps that cloned payload in a fresh `jayess_worker_message`
  - enqueues the message onto `state->inbound_head` / `state->inbound_tail`
- queue ownership rule:
  - the worker inbound queue owns the message node
  - the message node owns the cloned payload stored in `message->value`
  - the sender does not share its original value box directly with the worker
    queue

#### Outbound worker results and messages

- worker thread completion paths clone thrown values and normal results before
  queueing outbound messages
- `jayess_worker_queue_push(...)`
  - appends owned message nodes to the outbound queue
- `jayess_std_worker_receive_method(...)`
  - pops one outbound message
  - transfers `message->value` out to the receiver
  - frees the queue node itself
  - does not deep-copy the returned payload again at receive time

Current worker rule:

- unlike scheduler microtasks, worker queues do create a clone boundary before
  cross-thread enqueue
- queue nodes own those cloned payloads until dequeue or termination

#### Worker shutdown

- `jayess_worker_queue_free(...)`
  - walks remaining inbound/outbound messages
  - frees both the queue nodes and the cloned payloads they own
- `jayess_std_worker_terminate_method(...)`
  - shuts down the worker thread
  - drains/frees queue state
  - clears the wrapper's stored native state

### What is documented versus what is still unproven

Documented now:

- promise callback queues mostly store aliased Jayess values inside fresh task
  records
- worker queues clone payloads before cross-thread enqueue
- timer and async I/O queues own task nodes but mostly borrow/alias the Jayess
  values embedded in those nodes

Current executable promise-settlement coverage:

- `TestBuildExecutableSupportsPromiseThenRejectAndAwaitCatch`
  - exercises:
    - fulfilled `then(...)`
    - rejected promise propagation
    - `await` catch handling
- `TestBuildExecutableSupportsPromiseAllAndRace`
  - exercises:
    - `Promise.all(...)`
    - `Promise.race(...)`
- `TestBuildExecutableSupportsTimerPromiseRace`
  - exercises:
    - promise settlement racing against timer-driven async work
- `TestBuildExecutableSupportsPromiseFinallyAndAllSettled`
  - exercises:
    - fulfilled `finally`
    - rejected `finally`
    - `await` over settled promises
    - `Promise.allSettled(...)` settled-record construction
- `TestBuildExecutableSupportsPromiseAnyAndAggregateError`
  - exercises:
    - `Promise.any(...)`
    - aggregate-error construction on the all-rejected path
- `TestBuildExecutableRunsPromiseCallbacksAsMicrotasks`
  - exercises:
    - promise callback scheduling as microtasks rather than synchronous calls
    - microtask ordering relative to surrounding synchronous code
    - microtask ordering relative to timer callbacks
- `TestBuildExecutableSupportsJayessLibUVSchedulerIntegration`
  - exercises:
    - promise callbacks coexisting with timers, file I/O, path watchers,
      process completion, and UDP delivery in one integrated scheduler run

Still intentionally unproven:

- that every aliased queued value is preserved for exactly the required
  lifetime and no longer
- that promise settlement and deferred callback execution are globally free of
  leaks, UAF, or dead-reference retention across every async path

### Current executable worker-queue boundary coverage

The worker-queue ownership contract also has narrower executable coverage than
the broad async/runtime stress row by itself suggests.

Current backend executable tests already cover these two distinct worker
message-boundary shapes:

- `TestBuildExecutableSupportsWorkers`
  - exercises cloned `postMessage(...)` payload delivery across threads
  - shows that worker replies can return cloned object/array data without
    being affected by later mutations to the sender's original payload
- `TestBuildExecutableSupportsSharedMemoryAndAtomics`
  - exercises the intentional non-cloned shared-memory path through
    `SharedArrayBuffer`
  - shows that worker and main-thread views can observe the same backing
    storage through atomics and typed-array reads/writes

What this already proves:

- ordinary worker message payloads cross a real clone boundary before enqueue
- shared-memory worker traffic is a deliberate aliasing exception carried by
  shared backing storage rather than by reusing ordinary boxed value payloads

What it still does **not** prove:

- that all worker queue enqueue/dequeue/termination paths are sanitizer-clean
- that every worker message shape has complete forgotten-close and shutdown
  lifetime coverage
- that worker queue ownership is fully unified with the rest of the async
  scheduler surfaces

## Current executable host-wrapper coverage map

The remaining host-resource checklist row is broader than the coverage already
in tree. The useful distinction is between:

- wrappers that already have executable close-path regression coverage
- wrappers that still lack explicit alias-close, double-close, or forgotten-
  close repro coverage

### Coverage already present in backend executable tests

Current backend coverage already exercises explicit close paths for several
host-wrapper families.

#### LibUV package wrappers

`backend/toolchain_test.go` currently has executable coverage through:

- `TestBuildExecutableSupportsJayessLibUVPackage`
- `TestBuildExecutableSupportsJayessLibUVSchedulerIntegration`
- `TestBuildExecutableSupportsJayessLibUVProcessAndSignalIntegration`

Those tests currently prove explicit close-path behavior for:

- signal watchers
- path watchers
- spawned process wrappers
- UDP wrappers
- accepted TCP client wrappers
- outbound TCP client wrappers
- TCP server wrappers
- loop wrappers

And they also prove one important post-close surface:

- using a closed loop again reports a `TypeError` instead of silently behaving
  like a live handle

#### TLS socket detached snapshot coverage

`backend/toolchain_test.go` also has executable coverage through:

- `TestBuildExecutableSupportsJayessOpenSSLTLSConnect`

That test proves a narrower but still important host-wrapper property:

- a peer-certificate snapshot returned from a live TLS socket remains readable
  after `socket.close()`

This is not the same as full alias-close coverage for every socket-side host
wrapper, but it does prove that one detached data snapshot does not borrow the
live socket wrapper too aggressively.

#### Existing explicit close-again / after-close semantic slices

The backend suite also already contains narrower close-safety slices beyond the
libuv package tests:

- managed native-handle duplicate close behavior
  - executable coverage shows one-shot close followed by a non-repeating failed
    second close result
- UI/runtime wrapper after-close behavior
  - executable coverage already checks TypeError-style after-close behavior for
    wrappers such as:
    - webview
    - GTK child wrappers
    - closed libuv loops
    - closed SQLite statement/database wrappers
- several package/native wrapper close paths already report deterministic closed
  state or false-after-close state, including:
    - TLS socket/server close state
    - UDP/socket/server close state
    - worker close state
    - audio/windowing/native package close state in existing executable tests
- native wrapper misuse/type-mismatch rejection is already exercised across
  several package families, including:
  - generic native-wrapper object/buffer/handle mismatch checks
  - managed native-handle stale-handle rejection after close
  - SDL/OpenAL/miniaudio/PortAudio wrong-handle `TypeError` paths
  - GLFW/Raylib/windowing after-destroy stale-handle `TypeError` paths
- explicit destroy/terminate success paths are already exercised across several
  UI/audio wrapper families, including:
  - PortAudio stream close plus runtime terminate
  - GLFW window destroy plus worker terminate integration
  - Webview destroy/terminate flows, including mixed webview+GLFW cases
  - GTK explicit destroy and package destroy flows
  - Raylib/miniaudio-style context destroy slices already covered in existing
    executable tests

### What remains outside the currently proven host-wrapper coverage

Still not broadly proven today:

- close-after-alias behavior across every wrapper family
- explicit double-close behavior across every wrapper family
- forgotten-close paths across every wrapper family
- uniform executable coverage for all stream/socket/file/process/worker/server
  wrappers documented earlier

## Current async/runtime host-boundary executable coverage

The remaining async/runtime host-boundary row in the checklist is broader than
the executable integration coverage already in tree. The tree already proves
that several async subsystems can cross from Jayess values into host/runtime
state and back again successfully; it does not yet prove that those same paths
have been stress-tested under ASAN/LSAN/UBSAN like the synchronous lifetime
slices.

### Executable integration coverage already present

Current backend executable tests already cover these async/runtime host
boundaries:

- `TestBuildExecutableSupportsJayessLibUVPackage`
  - timers, callbacks, file reads, signal watchers, path watchers, process
    exits, UDP, TCP, and loop shutdown
- `TestBuildExecutableSupportsJayessLibUVSchedulerIntegration`
  - promise callbacks, timers, file I/O, path watching, process completion, UDP
    delivery, and loop shutdown in one integrated scheduler run
- `TestBuildExecutableSupportsJayessLibUVProcessAndSignalIntegration`
  - concurrent process-exit and signal-watcher interaction across the same loop
- `TestBuildExecutableSupportsWorkers`
  - worker creation, messaging, and termination across a host-thread boundary
- `TestBuildExecutableSupportsHttpCreateServer`
  - plain HTTP server callback execution over a live listening socket
- `TestBuildExecutableSupportsHttpsCreateServer`
  - HTTPS server callback execution over TLS-backed server state
- `TestBuildExecutableSupportsTlsCreateServer`
  - direct TLS server callback execution and socket lifecycle
- `TestBuildExecutableSupportsJayessOpenSSLTLSConnect`
  - TLS client connect, peer-certificate access, and close behavior
- `TestBuildExecutableSupportsJayessOpenSSLTLSServer`
  - OpenSSL-backed TLS server accept/read/write/close flow
- `TestBuildExecutableSupportsJayessWebviewWorkerIntegration`
  - worker messaging while a live webview wrapper is created, run, and
    destroyed
- `TestBuildExecutableSupportsJayessWebviewHTTPServerCoexistence`
  - webview and HTTP server host wrappers coexist in one executable process
- existing GLFW/audio worker integration slices in the backend suite
  - worker messaging and host-loop progression can coexist with live UI/audio
    wrapper state

What these tests already prove:

- async callbacks can cross host boundaries and re-enter Jayess successfully
- promise/timer/I/O/worker/server/socket wrappers can operate together in real
  compiled executables
- mixed subsystem coexistence already works across worker, UI/windowing, and
  server wrapper combinations covered by the backend suite
- several close and shutdown paths already work in integrated executable runs

### What remains intentionally unproven

Still not proven by those executable integration tests alone:

- ASAN/LSAN/UBSAN stress coverage across the same async/host boundaries
- systematic leak/UAF proof for exception-heavy async paths
- uniform ownership proof for every queue, wrapper, callback, and shutdown path
  involved in those integrations

## Exceptions, stacks, and propagation state

The exception/runtime-state path is another area where QuickJS is a useful
comparison point for discipline, but Jayess still has to describe its own
ownership rules explicitly.

### Current exception-state storage

`runtime/jayess_runtime_errors.c` currently owns three thread-local runtime
state chains:

- `jayess_this_stack`
- `jayess_call_stack`
- `jayess_current_exception`

Current ownership split:

- `jayess_this_stack`
  - owns only the stack-frame nodes
  - stores borrowed/aliased `jayess_value *` pointers as `this`
- `jayess_call_stack`
  - owns only the stack-frame nodes
  - stores borrowed function-name pointers
- `jayess_current_exception`
  - stores one current exception value pointer
  - that pointer is usually a fresh error object or another existing Jayess
    value forwarded through `jayess_throw(...)`

### Current error-construction contract

Error constructors are fresh object-wrapper constructors in the sense used
elsewhere in this document.

Examples:

- `jayess_error_value(...)`
- `jayess_type_error_value(...)`
- `jayess_std_error_new(...)`
- `jayess_std_aggregate_error_new(...)`

Current behavior:

- allocate one fresh error object wrapper
- write fresh string properties such as `name` and `message`
- in the aggregate-error case, attach an `errors` value built through normal
  iterable/container helper rules
- these helpers return fresh object wrappers, not immortal sentinels

### Current stack-attachment contract

- `jayess_capture_stack_trace_text(...)`
  - allocates a temporary C string buffer for the textual stack trace
- `jayess_attach_exception_stack(value)`
  - only acts on object exceptions
  - boxes that temporary text as a fresh Jayess string
  - stores the boxed stack string on the error object under `stack`
  - frees the temporary C buffer afterward

Current consequence:

- the `stack` property is an ordinary aliased object slot on the exception
  wrapper once attached
- stack attachment does not keep the raw temporary C buffer alive after the
  boxed Jayess string is created

### Current throw / take / report contract

- `jayess_throw(value)`
  - stores `value` directly into `jayess_current_exception`
  - substitutes immortal `undefined` when `value == NULL`
  - attaches a stack string if the value is an object
- `jayess_take_exception()`
  - transfers the current exception pointer out of thread-local storage
  - clears `jayess_current_exception`
  - returns immortal `undefined` if there was no exception
- `jayess_report_uncaught_exception()`
  - reads the current exception and optional `stack` property
  - prints them
  - clears `jayess_current_exception`
  - frees the current exception only if it is not the immortal `null` or
    `undefined` singleton
- `jayess_runtime_error_state_shutdown()`
  - frees all `this` and call-stack frame nodes
  - frees any remaining non-immortal current exception

### What is documented versus what is still unproven

Documented now:

- exception state owns frame nodes but mostly aliases Jayess values inside that
  state
- stack attachment uses a temporary C buffer but stores a fresh Jayess string
  on the error object
- `take`, `report`, and runtime shutdown are the current ownership transfer and
  cleanup boundaries for thread-local exception state

Current executable exception/error coverage:

- uncaught exception reporting
  - executable tests exercise uncaught `Error` reporting and exit behavior
- stack-bearing errors
  - executable tests exercise caught error stack output and debug-friendly stack
    traces
- promise aggregate error behavior
  - executable tests exercise `AggregateError` creation and `errors` payload
    exposure on promise failure paths
- non-function/type error reporting
  - executable tests exercise emitted `TypeError` behavior for invalid calls
  - executable tests also cover type-mismatch `TypeError` surfaces for native
    wrapper helpers and invalid value-kind expectations
- native-wrapper error propagation
  - executable tests exercise caught and uncaught native-wrapper error surfaces

Still intentionally unproven:

- that every exception propagation path is free of leaks for all temporary
  intermediate values
- that stack-bearing error objects never retain more aliased state than
  intended across every async and host boundary

## Public runtime header versus internal runtime surface

QuickJS also separates its public embedding/runtime surface from its internal
engine data structures. Jayess should do the same in Jayess terms: codegen and
host entry points belong in the public header, while cross-file implementation
helpers and concrete runtime layouts belong behind the internal header.

### What `jayess_runtime.h` currently exports

`runtime/jayess_runtime.h` is the public runtime surface that generated code
and host-facing callers are expected to use.

Current exported families:

- core value constructors and printers
- stringification / concatenation helpers
- process/console/readline helpers
- public object/array/value operations used directly by generated code
- public stdlib entry points such as:
  - promises
  - JSON
  - typed arrays / `ArrayBuffer` / `DataView`
  - process/path/url/querystring
  - DNS, crypto, compression
  - network/TLS/HTTP
  - filesystem
  - child process / worker
- public lifecycle hooks:
  - `jayess_run_microtasks()`
  - `jayess_runtime_shutdown()`
- public cleanup hooks intentionally used by codegen:
  - `jayess_value_free_unshared(...)`
  - `jayess_value_free_array_shallow(...)`
  - `jayess_object_free_unshared(...)`
  - `jayess_array_free_unshared(...)`

### What is intentionally *not* in `jayess_runtime.h`

The public header does not expose the concrete internal layouts or the
cross-file helper plumbing that split runtime modules use to implement those
APIs.

Those implementation details instead live in
`runtime/jayess_runtime_internal.h`, including:

- concrete struct layouts for:
  - `jayess_object`
  - `jayess_array`
  - `jayess_function`
  - `jayess_symbol`
  - `jayess_value`
- object-entry manipulation helpers
- stream/socket/crypto/worker helper plumbing used between runtime modules
- internal utility helpers such as:
  - `jayess_std_bytes_slot(...)`
  - `jayess_std_stream_emit(...)`
  - `jayess_std_worker_receive_method(...)`
  - `jayess_std_crypto_copy_bytes(...)`

### Current contract of the split

In the current tree, the header split means:

- generated LLVM/Jayess code is expected to call public runtime entry points
  from `jayess_runtime.h`
- embedding/host callers are expected to use that same public surface
- split runtime modules may include `jayess_runtime_internal.h` and use the
  concrete layouts plus helper plumbing behind the public API boundary
- code outside the runtime implementation is not expected to depend on concrete
  object/array/function/value layouts directly

This does **not** yet prove that the public surface is minimal or ideally
layered forever. It does mean the current tree already draws a real line
between:

- intentionally supported public codegen/host entry points
- internal cross-module runtime implementation helpers

## Current per-file ownership surface map

The remaining architecture row about "narrow ownership surfaces" is too broad
to be useful as a single yes/no claim. Some runtime files are already fairly
tight; others are still implementation clusters that mix several ownership
surfaces together.

### Runtime files that are already relatively narrow

These files mostly revolve around one primary ownership surface today:

- `runtime/jayess_runtime_values.c`
  - boxed value construction
  - singleton/static value policy
  - value destruction helpers
- `runtime/jayess_runtime_errors.c`
  - exception state
  - call/`this` frame stacks
  - error-object construction
  - stack attachment
- `runtime/jayess_runtime_strings.c`
  - text conversion
  - console/printing
  - prompt/readline helpers
- `runtime/jayess_runtime_bigint.c`
  - bigint arithmetic and bigint boxing helpers
- `runtime/jayess_runtime_typed_arrays.c`
  - typed-array/view/backing-storage behavior
  - shared-bytes state

These modules are not perfect, but each one already has a recognizable primary
ownership topic instead of acting as a general runtime grab-bag.

### Runtime files that are still broader clusters

These files still combine multiple ownership surfaces:

- `runtime/jayess_runtime_collections.c`
  - object storage
  - array storage
  - property-key lookup/update/delete
  - some higher-level collection/value helpers
- `runtime/jayess_runtime_process.c`
  - process helpers
  - signal handling
  - child-process helpers
  - worker message queues and thread state
- `runtime/jayess_runtime_network.c`
  - sockets
  - TLS
  - HTTP
  - watcher state
  - async network task helpers
- `runtime/jayess_runtime_fs.c`
  - filesystem operations
  - stream wrappers
  - watch/open-error helper families
- `runtime/jayess_runtime_streams.c`
  - generic stream/event-emitter behavior
  - compression-stream wrappers
  - byte-flow helpers
- `runtime/jayess_runtime.c`
  - still retains mixed value-core, callback-dispatch, scheduler, and stdlib
    host glue surfaces despite the split already achieved

### What this means for the checklist

So the honest current state is:

- Jayess has already moved several runtime concerns into fairly narrow files
- Jayess still has several broad implementation clusters where ownership
  reasoning crosses subsystem boundaries inside the same file
- the architecture row should only be fully checked once those remaining
  clusters stop depending on ad hoc cross-file helper use as their main glue

## Current object / array executable coverage map

The object/array storage rows are broader than the executable semantic coverage
already present in the backend suite. The current suite already exercises a
useful slice of dynamic object and array behavior, but that is not the same as
targeted ASAN/LSAN lifetime stress coverage.

### Executable semantic coverage already present

Current backend executable tests already cover these object/array-heavy paths:

- property enumeration and object spread
  - object key/value/entry enumeration
  - `for..in` ordering checks
  - object spread into fresh wrappers
- typed arrays and backing storage
  - `Int8Array`, `Uint16Array`, `Int32Array`, `Float32Array`, `Float64Array`
  - shared backing storage across typed views
  - typed-array copying and slicing
- symbol-registry object semantics
  - well-known symbols
  - symbol registry lookup and equality behavior
- native interop object/buffer/handle paths
  - object property reads/writes through native helpers
  - array/byte buffer conversion paths
  - native-handle wrapper interop

These executable slices are currently represented by tests such as:

- `TestBuildExecutableSupportsObjectSpread`
- the property enumeration executable test immediately above it in
  `backend/toolchain_test.go`
- `TestBuildExecutableSupportsTypedArrays`
- `TestBuildExecutableSupportsSymbolRegistryAndWellKnownSymbols`
- `TestBuildExecutableSupportsNativeInteropObjectsBuffersAndHandles`

### What this does and does not prove

What it does prove:

- object/array/property semantics are exercised in real compiled executables
- fresh object wrappers, aliased property reads, typed-array backing storage,
  and object-to-native interop all have executable behavioral coverage

What it does not prove yet:

- targeted ASAN/LSAN regressions for property-heavy churn
- targeted ASAN/LSAN regressions for array-heavy helper mutation/copy paths
- a full lifetime proof for every array growth/slice/concat/push/pop/shift/
  unshift helper under aliasing

## Current sanitizer coverage map

The checklist's memory/torture-testing block should reflect the distinction
between sanitizer probes that already exist and the broader runtime stress
matrix that still does not.

### Sanitizer probes already present

Current backend probes already provide targeted sanitizer coverage for two
important areas:

- parser package paths
  - `TestBuildExecutableParserPackagesAreLeakFreeUnderASAN`
  - proves a parser-oriented native executable lane can run clean under the
    opt-in ASAN/LSAN probe
- runtime lifetime cleanup slice
  - `TestBuildExecutableScopeCleanupStaysSafeUnderASAN`
  - proves the current implemented lifetime/cleanup slice can run clean under
    the opt-in ASAN/LSAN probe

Those probes are real and useful, but they are still narrower than a full
runtime-wide sanitizer matrix.

### What the current probes do not cover

The existing sanitizer probes do not yet amount to one broad matrix covering:

- dynamic property churn
- array helper callbacks as a whole ownership family
- constructor alternate returns across all ownership shapes
- exception-heavy control flow
- promise/worker/event queue ownership under sanitizer stress
- typed-array and native-handle lifetime stress

So the honest current state is:

- Jayess already has opt-in ASAN/LSAN proof lanes for parser behavior and the
  current implemented runtime cleanup slice
- Jayess does **not** yet have the dedicated per-subsystem ASAN/LSAN/UBSAN
  matrix implied by the broad remaining checklist row

### Existing semantic coverage beneath the missing sanitizer matrix

Even though the dedicated sanitizer matrix is still missing, several of its
categories already have narrower semantic or executable coverage:

- dynamic property churn
  - object spread and property enumeration executable coverage exists
  - concrete covered object/property slices already include:
    - `TestBuildExecutableSupportsObjectSpread`
    - the property enumeration executable test immediately above it in
      `backend/toolchain_test.go`
    - `TestBuildExecutableSupportsSymbolRegistryAndWellKnownSymbols`
    - `TestBuildExecutableSupportsNativeInteropObjectsBuffersAndHandles`
- array helper callbacks
  - the current discarded callback cleanup slices are already exercised through
    compile-time cleanup probes and the opt-in lifetime ASAN lane
  - concrete covered callback-helper slices already include:
    - `TestCompileEmitsDiscardedFreshExpressionCleanup`
    - `TestBuildExecutableScopeCleanupStaysSafeUnderASAN`
  - those probes currently exercise direct and bound callback forms across:
    - `forEach`
    - `map`
    - `filter`
    - `find`
    - the currently claimed scalar fixed-arity callback slices through the
      present twelve-bound cap
- constructor alternate returns
  - executable semantic coverage exists through
    `TestBuildExecutableConstructorCanReturnAlternateObject`
  - the currently explicit executable slice is the aliased-object alternate
    return shape
  - the current discarded-cleanup probe boundary also already covers:
    - plain implicit-`__self` constructor temporaries
    - fresh alternate-return constructor temporaries
  - those discarded constructor cleanup slices are exercised through:
    - `TestCompileEmitsDiscardedFreshExpressionCleanup`
    - `TestBuildExecutableScopeCleanupStaysSafeUnderASAN`
- exception-heavy control flow
  - executable semantic coverage exists for caught/uncaught errors, stack
    output, `TypeError`, and aggregate-error behavior
  - concrete covered exception/error slices already include:
    - `TestBuildExecutableThrowsTypeErrorForNonFunctionInvocation`
    - `TestBuildExecutableSupportsErrorStackTraces`
    - the uncaught-exception executable test immediately before it in
      `backend/toolchain_test.go`
    - `TestBuildExecutableSupportsPromiseAnyAndAggregateError`
    - `TestBuildExecutableSupportsNativeWrapperErrorPropagation`
- promise/worker/event queues
  - executable semantic coverage exists for promises, workers, libuv scheduler
    integration, and queue-owning async paths
  - concrete covered queue/scheduler slices already include:
    - `TestBuildExecutableSupportsPromiseThenRejectAndAwaitCatch`
    - `TestBuildExecutableSupportsPromiseAllAndRace`
    - `TestBuildExecutableSupportsTimerPromiseRace`
    - `TestBuildExecutableSupportsPromiseFinallyAndAllSettled`
    - `TestBuildExecutableSupportsPromiseAnyAndAggregateError`
    - `TestBuildExecutableRunsPromiseCallbacksAsMicrotasks`
    - `TestBuildExecutableSupportsWorkers`
    - `TestBuildExecutableSupportsSharedMemoryAndAtomics`
    - `TestBuildExecutableSupportsJayessLibUVSchedulerIntegration`
  - together these already cover:
    - promise settlement and rejection paths
    - microtask ordering
    - worker message queues and shared-memory worker aliasing
    - mixed scheduler coexistence with timers, file I/O, path watchers,
      process completion, and UDP delivery
- typed-array and native-handle paths
  - executable semantic coverage exists for typed arrays, native interop
    objects/buffers/handles, managed native-handle close behavior, and native
    wrapper lifetime-safe copies
  - concrete covered slices already include:
    - `TestBuildExecutableSupportsTypedArrays`
    - `TestBuildExecutableSupportsNativeInteropObjectsBuffersAndHandles`
    - `TestBuildExecutableSupportsManagedNativeHandleOwnership`
    - `TestBuildExecutableSupportsNativeWrapperLifetimeSafeCopies`
    - `TestBuildExecutableSupportsSharedMemoryAndAtomics`
  - within those tests, the typed-array side already covers:
    - ordinary typed-array copy construction
    - typed-array slice result materialization
    - cross-view aliasing over one `ArrayBuffer`
    - shared-backing aliasing through `SharedArrayBuffer` plus `Atomics`

That means the remaining red block is specifically about missing **sanitizer
stress coverage**, not complete absence of semantic coverage.

## Current memory introspection boundary

QuickJS-style runtime accounting would mean introspection about Jayess-managed
runtime entities themselves: boxed values, objects, arrays, functions, strings,
and host-resource wrappers.

### What Jayess already has

Jayess already exposes process-level/system-level introspection helpers such as:

- `process.memoryInfo()`
- `process.cpuInfo()`
- `process.userInfo()`
- `process.threadPoolSize()`
- `process.tmpdir()`
- `process.hostname()`

Those are already covered by executable tests such as:

- `TestBuildExecutableSupportsProcessSystemInfoSurface`
- `TestBuildExecutableSupportsExtendedProcessSystemInfoSurface`

Within those tests, the currently explicit process/system introspection slices
already cover:

- `process.tmpdir()`
- `process.hostname()`
- `process.cpuInfo()`
- `process.memoryInfo()`
- `process.userInfo()`
- `process.threadPoolSize()`

### What Jayess does not have yet

Those process helpers are **not** the same as runtime-owned accounting.

Still missing today:

- a debug/accounting view over Jayess-managed runtime entities
- counts or summaries for:
  - boxed values
  - objects
  - arrays
  - functions/closures
  - strings/bigints
  - native handles / host resources

So the honest current boundary is:

- OS/process introspection exists
- Jayess runtime accounting comparable to QuickJS usefulness does not

### Category-by-category runtime accounting gap

The remaining accounting checklist row is not blocked by one abstract missing
"debug mode." It is blocked by the absence of category-specific runtime-owned
counters.

Still missing by category:

- boxed values
  - no runtime-wide count of live `jayess_value` boxes by kind
- objects
  - no live object-wrapper count or object-entry/property count summary
- arrays
  - no live array-wrapper count or total slot-count summary
- functions/closures
  - no live function-wrapper count
  - no bound-arg / closure-env summary accounting
- strings/bigints
  - no live heap-string / bigint allocation summary
- native handles / host resources
  - no runtime-owned summary of live managed handles, sockets, streams,
    workers, watchers, or TLS/HTTP sidecars

So the remaining work here is concrete:

- add runtime-owned counters or summaries for the categories above
- expose them through a Jayess-facing debug/accounting API
- prove that the counters remain useful across host-wrapper and async-heavy
  workloads

### Minimum shape of a useful Jayess-owned accounting view

To be comparable in usefulness to QuickJS-style runtime accounting while still
matching Jayess's own ownership model, a future accounting/debug surface should
at least answer:

- how many live boxed runtime values currently exist
- how many live object wrappers and total object entries currently exist
- how many live array wrappers and total array slots currently exist
- how many live function wrappers exist, plus bound-arg/container summaries
- how many live heap strings and bigints exist
- how many live managed native handles / host-resource wrappers exist by major
  family

And it should ideally distinguish:

- immortal/static values versus heap-owned boxes
- wrapper counts versus backing-storage counts
- host-wrapper counts versus underlying live host-handle counts
- process-level/system memory information versus Jayess-owned runtime state

That is still a design target, not a current implementation.

### Where those future counters would need to hook in

In the current runtime layout, the missing accounting would have to hook into
concrete allocation and release sites such as:

- boxed values
  - `jayess_value_from_*` constructors
  - `jayess_value_free_unshared(...)`
- objects / object entries
  - `jayess_object_new()`
  - object-entry allocation/removal helpers
  - `jayess_object_free_unshared(...)`
- arrays / array slots
  - `jayess_array_new()`
  - grow/shrink helpers
  - `jayess_array_free_unshared(...)`
- functions / bound args
  - `jayess_value_from_function(...)`
  - `jayess_value_bind(...)`
  - `jayess_value_merge_bound_args(...)`
  - function-wrapper destruction in `jayess_value_free_unshared(...)`
- strings / bigints / symbols
  - heap-owning scalar constructors
  - scalar destruction paths in `jayess_value_free_unshared(...)`
- native handles / host wrappers
  - managed-handle constructors and finalizers
  - wrapper-specific close/terminate/free paths in fs/network/process/streams

That means the accounting gap is not conceptual anymore. It is an explicit lack
of counters at known allocation/free boundaries already present in the runtime.
For every missing category, the tree now has both a documented target summary
shape and a documented set of hook points where future counters would update.
Those six missing summary families also already line up with concrete runtime
subsystem boundaries: value boxes, collections, function wrappers, heap scalar
boxes, and host-resource wrappers.

## Current open-repro versus open-surface distinction

The remaining broad `9.5` gate rows should distinguish two different kinds of
blocker:

- a concrete known open repro
  - an actual reproduced UAF, double-free, or non-escaping leak
- an open surface
  - an area that is not yet stress-proven broadly enough to justify the global
    guarantee, even if no single current crashing repro is pinned to it

Current state in this tree:

- the currently claimed lifetime slices are green on their existing proof lanes
  and are not carrying a known still-open reproducer inside those claimed
  boundaries
- the remaining blockers are mostly open surfaces:
  - host-resource alias-close / double-close / forgotten-close coverage is not
    complete across wrapper families
  - async/host boundaries do not yet have a full ASAN/LSAN/UBSAN stress matrix
  - exception-heavy control flow is documented but not broadly sanitizer-proven
  - property-heavy and array-heavy churn is not yet covered by a dedicated
    lifetime stress matrix
  - helper ownership is still documented by family/slice rather than enforced
    by one runtime-wide mechanical table

So the broad `9.5` rows should stay unchecked until both conditions are true:

- no concrete known repro remains
- no major open surface remains that could still plausibly hide one

## Current gate map for the broad `9.5` rows

The remaining top-level `9.5` guarantees are now blocked more by incomplete
proof surfaces than by one single unresolved subsystem. Each row has a specific
remaining gate.

### `9.5 no use-after-free is possible`

Already true:

- the currently claimed proved slices are green on their existing ASAN/LSAN
  lanes
- several formerly failing callback and constructor slices have already been
  converted into explicit green probes

Still required before the broad row can be checked:

- no known runtime UAF repro remains anywhere in the unclaimed surface
- the broader runtime stress matrix exists, not only the current parser/lifetime
  probe lanes

### `9.5 no double-free is possible`

Already true:

- generic native-handle wrapper semantics document one-shot managed close
  behavior
- close/finalize ownership is documented across the major wrapper families
- the Jayess/native boundary already has its own checked no-double-free slice

Still required before the broad row can be checked:

- duplicate-destroy regressions across all wrapper families and close/finalize
  paths, not only the generic handle contract
- broader alias-close and forgotten-close coverage where double-destroy bugs
  could still hide

### `9.5 no memory leaks for non-escaping values`

Already true:

- the current implemented cleanup slice is green under the opt-in lifetime
  ASAN/LSAN probe
- many discarded temporary families have been individually proven and recorded

Still required before the broad row can be checked:

- a runtime-wide mechanical ownership classification for non-escaping values,
  not only enumerated proved slices
- broader sanitizer coverage outside the currently claimed slices

### `9.5 pointer/reference validity is always preserved`

Already true:

- aliasing rules are documented for:
  - containers
  - closures and bound functions
  - constructor returns
  - callback queues
  - native handles and host wrappers
- several alias-sensitive executable semantic slices already exist, including:
  - `TestBuildExecutableSupportsObjectSpread`
  - `TestBuildExecutableSupportsTypedArrays`
  - `TestBuildExecutableConstructorCanReturnAlternateObject`
  - `TestBuildExecutableSupportsWorkers`
  - `TestBuildExecutableSupportsSharedMemoryAndAtomics`
  - `TestBuildExecutableSupportsJayessOpenSSLTLSConnect` detached certificate
    snapshot behavior

Still required before the broad row can be checked:

- those aliasing rules are stress-tested broadly enough across the runtime
- the remaining open surfaces around async host boundaries, wrapper close
  behavior, and object/array churn are closed

## Current lifetime reproducer discipline

The remaining reproducer row in the checklist should not imply that Jayess has
no reproducer discipline today. The tree already has a concrete pattern for
turning ownership bugs into narrow probes; it is just not yet runtime-wide.

### Reproducer patterns already present

Current tree patterns include:

- compile-time narrow cleanup probes
  - `TestCompileEmitsDiscardedFreshExpressionCleanup`
- executable cleanup/finalizer probes built around the native cleanup-probe
  fixture
  - `@jayess/cleanupprobe`
  - `TestBuildExecutableCleansUpEligibleDynamicLocalsOnScopeExit`
  - `TestBuildExecutableScopeCleanupStaysSafeUnderASAN`
- narrow semantic executable probes for lifetime-sensitive edges such as:
  - `TestBuildExecutableConstructorCanReturnAlternateObject`
  - `TestBuildExecutableSupportsWorkers`
  - `TestBuildExecutableSupportsSharedMemoryAndAtomics`
  - `TestBuildExecutableSupportsNativeWrapperErrorPropagation`
  - promise/exception control-flow executable cases
  - typed-array/object/native-interop executable cases
  - object/property churn executable cases such as:
    - `TestBuildExecutableSupportsObjectSpread`
    - the property enumeration executable test immediately above it in
      `backend/toolchain_test.go`
    - `TestBuildExecutableSupportsSymbolRegistryAndWellKnownSymbols`
  - host-wrapper close/after-close executable cases such as:
    - webview explicit after-close checks
    - GTK child after-close checks
    - SQLite finalize/close after-close checks
    - closed libuv loop after-close checks
    - detached TLS certificate snapshot reads after socket close
- focused callback-helper cleanup probes such as:
  - `TestCompileEmitsDiscardedFreshExpressionCleanup`
  - `TestBuildExecutableScopeCleanupStaysSafeUnderASAN`
  - exercising discarded `forEach` / `map` / `filter` / `find` callback-helper
    slices across the currently claimed scalar fixed-arity boundary

What that means:

- when a lifetime bug is narrowed enough to one helper shape or one discarded
  temporary family, the project already tends to add a focused reproducer-style
  test rather than relying only on broad end-to-end behavior
- the cleanup-probe fixture gives the runtime a concrete way to observe finalizer
  behavior and statement-exit cleanup in compiled executables
- constructor alternate-return, worker queue, shared-memory alias, and
  native-wrapper error edges already have their own focused executable
  reproducer-style tests instead of only broad integration coverage
- callback-helper lifetime edges also already have focused compile-time and
  opt-in ASAN reproducer coverage instead of only broad array-helper behavior
- promise settlement/rejection and exception-propagation edges already have
  focused executable repro-style tests instead of only broad async integration
- object/property churn and typed-array/native-interop edges already have
  focused executable repro-style tests instead of only broad runtime-surface
  coverage
- host-wrapper close/after-close edges already have focused executable
  repro-style tests instead of only broad wrapper-family integration coverage

### What is still missing

Still not true yet:

- every known lifetime bug category is guaranteed to have a dedicated minimal
  reproducer before being considered fixed
- the reproducer pattern is not yet systematic across every remaining runtime
  surface, especially the broad open surfaces listed in the `9.5` gate map

## Current ownership contract for stdlib helper families

This section is the stdlib-side counterpart to the lower-level value/object
helper sections above. The goal is not to pretend every leaf helper is already
mechanically tagged in code; the goal is to describe the current ownership
families that stdlib helpers fall into today.

### 1. Predicate, flag, and sentinel helpers

These helpers usually return immortal/static singleton values rather than fresh
heap boxes.

Examples:

- `jayess_std_array_is_array(...)`
- `jayess_std_fs_exists(...)`
- `jayess_std_fs_write_file(...)`
- `jayess_std_fs_append_file(...)`
- failure sentinels from many stdlib helpers that return
  `jayess_value_undefined()`

Current contract:

- boolean-style success/failure helpers usually return
  `jayess_value_from_bool(...)`, which means:
  - `true` and `false` are immortal/static singleton values
- "missing/failed/not supported" sentinel paths often return
  `jayess_value_undefined()`, which is also immortal/static
- these helpers therefore usually do not transfer heap ownership to callers on
  their scalar success/failure path

### 2. Fresh stdlib object-wrapper constructors

These helpers allocate one fresh object wrapper and populate it with aliased or
fresh property values according to the ordinary object-slot rules.

Examples:

- `jayess_std_error_new(...)`
- `jayess_std_aggregate_error_new(...)`
- `jayess_std_promise_pending(...)`
- `jayess_std_promise_resolve(...)`
- `jayess_std_promise_reject(...)`
- host wrapper constructors such as:
  - `jayess_std_fs_create_read_stream(...)`
  - `jayess_std_fs_create_write_stream(...)`
  - `jayess_std_socket_value_from_handle(...)`
  - worker/server/http wrapper creators

Current contract:

- the wrapper object itself is a fresh owned runtime allocation
- property values written onto that wrapper follow the existing object helper
  rules:
  - many are fresh values created for the wrapper
  - some are aliased existing Jayess values
- promise helpers are a special case of the same pattern:
  - fresh promise wrapper
  - aliased settled value stored on the wrapper
- host-wrapper constructors are another special case:
  - fresh wrapper object
  - authoritative host handle/state stored on that wrapper

### 3. Fresh stdlib container-building helpers

These helpers create fresh arrays/objects/typed wrappers, but often populate
them with aliased element values unless they are explicitly copying raw bytes or
string data.

Examples:

- `jayess_std_array_from(...)`
- `jayess_std_array_of(...)`
- `jayess_std_object_from_entries(...)`
- `jayess_std_fs_read_dir(...)`
- `jayess_std_fs_stat(...)`
- promise combinators that build result arrays/records:
  - `Promise.all`
  - `Promise.allSettled`
  - `Promise.any`

Current contract:

- the returned container wrapper is fresh owned
- contained Jayess values are frequently aliased rather than deep-cloned
- container-building helpers that consume iterable/object input therefore do
  not automatically create a deep ownership boundary for all nested values

### 4. Fresh byte/string materialization helpers

These helpers allocate new host buffers or new Jayess boxed values from copied
host data.

Examples:

- `jayess_std_fs_read_file(...)`
- compression helpers:
  - `jayess_std_compression_gzip(...)`
  - `jayess_std_compression_gunzip(...)`
  - `jayess_std_compression_brotli(...)`
  - `jayess_std_compression_unbrotli(...)`
- stream/socket read helpers that create new text or byte buffers

Current contract:

- transient C buffers allocated during I/O or compression are owned by the
  helper itself and freed before return
- the returned Jayess value is usually a fresh owned:
  - string box
  - `ArrayBuffer` wrapper
  - typed-array wrapper
- when raw bytes are returned, the helper typically allocates fresh backing
  storage and copies produced bytes into that storage before returning

### 5. Async stdlib entry points

These helpers usually return a fresh promise wrapper immediately, then enqueue
aliased input values into scheduler task records as documented in the async
section above.

Examples:

- `jayess_std_fs_read_file_async(...)`
- `jayess_std_fs_write_file_async(...)`
- socket read/write async helpers
- watcher poll async helpers
- HTTP request async helpers
- `sleepAsync(...)`-style helpers

Current contract:

- immediate return value is a fresh owned promise wrapper
- queued task nodes are fresh owned runtime records
- most Jayess values captured into those task records are aliased pointers, not
  cloned payloads
- worker messaging remains the exception: cross-thread worker queues clone the
  message payload before enqueue

### 6. Mixed-family stdlib failure behavior

Stdlib helpers do not currently report failure through one universal ownership
shape.

Current patterns include:

- immortal `undefined`
- immortal booleans
- fresh error wrappers
- fresh rejected promises

This matters for `9.5` because statement-exit cleanup cannot infer ownership
from "stdlib helper" as one class. It still has to know which stdlib family a
particular helper belongs to.

## Current statement-exit cleanup policy in codegen

Jayess codegen does not yet consume one mechanically generated ownership table
for every runtime helper. The current model is narrower and more manual.

### What codegen currently does

Generated code currently emits direct cleanup calls such as:

- `jayess_value_free_unshared(...)`
- `jayess_value_free_array_shallow(...)`

but only for slices that have already been explicitly proven or documented as
safe fresh-returning paths.

In practice that means:

- statement-exit cleanup is still driven by conservative helper-family
  knowledge and explicit fresh-return analysis
- alias-returning paths are intentionally kept out of broad cleanup claims until
  they have their own documented/proven rule
- many runtime/library shapes are still covered by enumerated proven slices
  rather than one universal helper-classification table

### What is already true

- `jayess_value_free_unshared(...)` is only intended for values that are known
  to be fresh owned boxes or fresh wrappers on the current path
- constructor-return freshness only propagates through
  `__jayess_constructor_return(__self, freshExpr)` because that helper preserves
  the ownership class of `freshExpr`
- alias-returning helper families such as ordinary object-slot reads or promise
  settled-value storage are documented as aliasing boundaries, not generic
  statement-exit cleanup candidates

### What is not true yet

- there is not yet one mechanical source of truth that codegen consults for all
  runtime helper ownership classes
- alias-returning helpers are not yet marked in one universal machine-readable
  way
- cleanup still depends on explicit per-slice reasoning rather than a complete
  runtime-wide ownership table

## Current helper-classification completeness boundary

The runtime roadmap now has the four ownership classes written down in Jayess
terms:

- fresh owned value
- borrowed non-owning alias
- immortal/static value
- transferred/consumed input

And those classes are already documented across the main helper families:

- core value constructors
- object/array/member helpers
- function/bind/apply/call helpers
- constructor-return helpers
- stdlib helpers

What is still missing is helper-by-helper completeness.

Current honest state:

- the class vocabulary exists
- the major helper families are mapped into that vocabulary
- some concrete helpers inside those families are still described by family
  contract rather than one explicit per-helper table
- there is not yet one runtime-wide inventory proving that every exported and
  internal helper has exactly one mechanically assigned ownership class

## Current callback fast-path versus slow-path ownership boundary

Jayess currently has two callback invocation shapes in codegen:

- direct fixed-arity fast paths emitted through `emitArrayCallbackInvocation(...)`
  and `emitArrayCallbackInvocationDiscardingResult(...)`
- generic boxed-arg-array fallback emitted through `emitApplyFromValues(...)`

### What is already aligned

- both paths ultimately invoke the same runtime function values
- both paths preserve the same high-level call intent:
  - callback
  - `this`
  - bound arguments
  - current item argument
- both paths are covered by the same deliberate arity-cap rule:
  - zero through twelve pre-bound args stay on direct fixed-arity paths
  - larger arities fall back to generic apply

### What is still handled by separate cleanup rules

Today the ownership story is still not one unified callback contract.

The remaining differences are intentionally tracked as checklist blockers:

- fast paths can free discarded callback results directly in helper-specific
  places such as `forEach` and `filter`
- slow apply paths still rely on boxed arg-array wrapper creation and cleanup
  that direct paths avoid entirely
- callback-result cleanup is still reasoned about per helper shape
  (`forEach` / `map` / `filter` / `find`) rather than through one shared rule
- bound-arg storage and `this` handling are closer than before, but still not
  expressed as one mechanical ownership contract spanning direct, bound, and
  generic apply paths

### What this means for the checklist

So the honest current state is:

- the fast/slow callback boundary is documented
- the arity-cap/fallback rule is documented
- the paths are **not** yet proven identical in ownership behavior across
  callback results, arg wrappers, and `this` handling

### Current executable callback-dispatch coverage

The callback-dispatch checklist block should also distinguish the executable
coverage that already exists from the stronger ownership-unification claims that
still do not.

Current backend and compiler coverage already exercises these callback-dispatch
surfaces:

- `TestBuildExecutableSupportsFunctionBindCallApplyMethods`
  - direct function invocation
  - `.bind(...)` with pre-bound arguments
  - `.call(...)`
  - `.apply(...)`
- `TestCompileEmitsDiscardedFreshExpressionCleanup`
  - compile-time cleanup emission for discarded callback-heavy array helpers
  - direct and bound callback forms
  - current fixed-arity slices from zero through twelve pre-bound args
- `TestBuildExecutableScopeCleanupStaysSafeUnderASAN`
  - opt-in executable ASAN/LSAN probe for the currently claimed callback
    cleanup slices
  - direct and bound `forEach` / `map` / `filter` / `find` cases within the
    proved scalar boundary

What that already proves:

- direct invocation and the object-method surfaces for `bind`, `call`, and
  `apply` have executable semantic coverage
- the current fixed-arity array-callback fast paths are not only documented;
  they are exercised through the compile-time cleanup probe and the opt-in
  lifetime sanitizer lane

What it still does **not** prove:

- that direct, bound, and apply paths share one mechanically unified ownership
  contract
- that fast and slow callback paths have identical ownership behavior for
  callback results, temporary arg-array wrappers, or `this` handling
- that callback cleanup is unified across all current and future helper shapes

### 4. Stdlib Host Glue

These are runtime entry points that are still mixed into `runtime.c` even though
they are really host-API behavior:

- some path/fs/network/process/http/tls/socket helper wiring
- wrapper-object method exposure for streams, watchers, sockets, HTTP bodies,
  servers, workers, and compression streams
- platform-specific helper glue

This is the least desirable place to keep growing. It makes ownership reasoning
harder because dynamic semantics and host-resource management get mixed together.

## Why This Helps `9.5`

The unchecked `9.5` rows are still broad runtime guarantees:

- no use-after-free
- no double-free
- no non-escaping leaks
- pointer/reference validity is always preserved

Those rows stay hard because ownership bugs can hide in different layers:

- value constructors and destructors
- object/property aliasing
- callback fast paths versus slow `apply` paths
- constructor alternate returns
- native handles and host-resource wrappers

By classifying the remaining `runtime.c` work into the buckets above, Jayess can
close `9.5` more systematically:

1. Define ownership contracts per bucket.
2. Move helpers into the right runtime file instead of leaving mixed logic in
   `runtime.c`.
3. Add ASAN/LSAN regressions targeted at each bucket.
4. Stop relying on “known-safe slices” alone and move toward a mechanically
   documented ownership model.

## QuickJS reference discipline for Jayess checklist claims

QuickJS is a useful reference implementation for subsystem boundaries, dynamic
feature coverage, and runtime testing discipline. It is **not** a license to
claim a Jayess checklist row just because QuickJS has an analogous feature.

The current rule for this roadmap is:

- if a QuickJS comparison motivates a Jayess runtime feature or hardening task,
  the Jayess ownership model for that feature must be written down in Jayess
  terms before the checklist row is claimed

In this document, that translation is now explicit for the QuickJS-comparable
runtime areas already discussed:

- value constructors and boxed-value lifetime classes
- object/array/property storage and aliasing rules
- fixed-arity callback fast paths versus generic apply fallback
- constructor-return ownership
- closures, bound functions, and captured environments
- typed arrays, `ArrayBuffer`, `DataView`, and backing storage
- native handles and host-resource wrappers
- promise queues, microtasks, async tasks, and worker message queues
- public runtime header versus internal runtime implementation surface

That means QuickJS is currently being used in this project as:

- a comparison point for where subsystem boundaries should exist
- a comparison point for which dynamic/runtime surfaces deserve explicit
  ownership contracts
- a comparison point for what kind of torture-testing and accounting discipline
  Jayess still needs
- a comparison point for which runtime questions Jayess must answer explicitly
  in its own terms: ownership, aliasing, queueing, wrapper lifetime, and
  accounting boundaries
- a comparison point for how runtime code may be decomposed, while Jayess's own
  ownership vocabulary and public/runtime API boundaries remain authoritative

And QuickJS is **not** being used as:

- proof that Jayess already has the same safety properties
- a reason to skip Jayess-specific ownership documentation
- a reason to copy QuickJS memory management instead of describing Jayess's own
  aliasing, fresh-value, and host-wrapper rules
- a substitute for Jayess-native executable probes, sanitizer lanes, or
  checklist exit criteria
- a reason to mark a Jayess subsystem "done" just because it now resembles a
  QuickJS subsystem structurally
- a reason to describe accounting or torture-testing targets in generic
  QuickJS-like terms instead of naming Jayess-specific runtime entities and
  workloads
- a requirement that Jayess converge to QuickJS internal representation details
  when Jayess's own representation and lifetime tradeoffs are the real subject
  being evaluated
- a reason to treat code-structure similarity as equivalent to Jayess-observable
  runtime behavior, ownership semantics, or proof coverage
- a reason to inherit QuickJS public-surface expectations or failure behavior
  without defining the Jayess-specific contract for those same runtime edges
- a reason to assume QuickJS-style cleanup or host-wrapper lifecycle behavior is
  already correct for Jayess without Jayess-native lifetime proofs

## Next Runtime Steps

- classify every exported runtime helper as:
  - fresh owned
  - borrowed alias
  - immortal/static
  - transferred/consumed
- generate or centralize the fixed-arity callback helper fan-out instead of
  extending it manually forever
- move more host-wrapper method exposure out of `runtime.c`
- add ownership-oriented debug accounting for boxed values, containers,
  functions, strings/bigints, and host handles
- keep using QuickJS as a reference for runtime testing discipline and subsystem
  boundaries, not as a memory-management design to copy
