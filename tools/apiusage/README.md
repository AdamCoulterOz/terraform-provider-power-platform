# API usage mapping

Works out which provider resources and data sources call which API operations,
and writes it out as a mapping file the OAS browser renders as a coverage view.

The mapping answers questions the code cannot be read for at a glance: which
operations nothing calls, which parts of the provider still depend on the legacy
BAPI, and which calls the provider makes that no published spec describes.

## Ownership

The **format** is owned by the OAS browser, not by this repository. It defines
the contract, publishes the validator, and every content repository authors into
it and runs that validator in CI. This directory owns the **content** only, which
is why there is no conformance checker here: if each producer wrote its own, the
next invariant added to the contract would have to be independently rediscovered
by all of them, and the moment one lagged the browser would be consuming a file
checked against an older contract.

## Running it

Needs a local checkout of the spec corpus. Point `POWERPLATFORM_APIS` at it, or
pass `--corpus`.

```sh
cd tools/apiusage
go run ./callsites > sites.json
go run ./callgraph > artifacts.json
python3 serialise.py sites.json artifacts.json -o ../../api-usage-mapping.json
```

Both Go tools take the repository root as an optional argument and default to
`../..`. This is a module of its own so the provider's dependency graph does not
grow a static-analysis library it never links; it is excluded from the parent
module's `./...`.

## How it works

**`callsites`** resolves every `api.Client.Execute` call to a method, host, path
template and api-version. It has no type information, so it works by following
the codebase's own URL-builder helpers, string constants and local variables.
It resolves 170 call sites.

**`callgraph`** type-checks the whole of `internal/` and walks from each
registered resource and data source down to those call sites, recording the
route taken and whether it passes through a conditional. The component list
comes from the provider's own `Resources()` and `DataSources()` rather than from
filenames, because a filename heuristic is a remembered list rather than a
derived one, and its failure mode is quiet: an earlier version of this tool
matched `resource_*.go` and silently dropped the one component whose file is
named `resources_*.go`.

**`serialise.py`** joins the two, resolves each call to an `operationId` in the
corpus, and emits the mapping. `resolve.py` holds the matching rule, which comes
from the contract: a templated segment in the spec matches any single segment in
the call, everything else must match exactly including case, and where several
operations match, the one matching on the most literal segments wins. That last
clause matters because the Dataverse spec documents a generic OData surface, and
without it `records_query` shadows every specific operation on that host.

## What it deliberately does not do

**It does not normalise away defects.** Two billing-policy calls spell a path
`/licensing/BillingPolicies` where the API and the provider's own other calls
spell it `billingPolicies`. Those rows fail the catalogue check, correctly, and
will keep failing until the provider is fixed. A mapping that quietly corrected
them would be hiding a real defect inside the file whose job is describing what
the code does.

**It does not assert that two operations are the same.** Saying a BAPI operation
and a Power Platform API operation are equivalent is a claim about what the specs
describe, not about what this code calls. It belongs to the corpus, and encoding
it here would turn a mapping into an equivalence table that nothing validates.

**It does not guess.** Two call sites in `data_record` build their path by
conditional reassignment, and this analysis does not track branches. They carry
`approximate: true` rather than a plausible path, because a row that says it is
uncertain is worth more than a row that is quietly wrong.

## Reading the output

Every call carries a `coverage` and a `grade`.

`coverage` is `full` when the entrypoint always reaches the operation and
`partial` when it reaches it only on some code path, with a `note` naming the
route. It is derived from whether the call sits inside a conditional, which is a
property of the code rather than of any practitioner's configuration.

`grade` is `derived` throughout: every row is read out of the source by static
analysis, not seen executing. The vocabulary has room for `observed`, for rows
that are one day backed by recorded traffic.

`uncatalogued` holds calls no operation can name: two async continuations that
follow a `Location` header, and the practitioner-supplied URL the `rest`
components exist to issue.
