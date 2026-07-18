# Retrieving External Auditor Information

OpenDART exposes auditor information through two different filing populations.
Periodic-report extraction provides convenient structured fields, while
standalone external-audit filings cover companies that do not publish the
periodic reports expected by those structured operations. A consumer that needs
broad company coverage should use the layered strategy below rather than treat
one endpoint as complete.

Guide-sourced facts and live observations are kept separate here. The canonical
request and response contracts remain the generated OpenAPI files linked from
each section.

## Retrieval strategy

### 1. Prefer structured auditor and opinion data

Use `GET /accnutAdtorNmNdAdtOpinion.json` when the company, business year, and
periodic report code are known. Its rows include `adtor` (auditor),
`adt_opinion` (opinion), filing identity, and related audit-matter fields.

The operation extracts the auditor section from periodic reports: annual,
semiannual, and quarterly reports. Status `013` therefore does not prove that
OpenDART has no audit report for the company; the company may instead have filed
a standalone external-audit report.

See the canonical
[path contract](../../openapi/paths/ds002/accnutAdtorNmNdAdtOpinion.json.yaml) and
[response schema](../../openapi/schemas/ds002/2020009.yaml).

### 2. Fall back to standalone audit-report discovery

Use `GET /list.json` with a company code and explicit filing-date range. Query
the external-audit detail types separately:

- `F001`: 감사보고서 (audit report)
- `F002`: 연결감사보고서 (consolidated audit report)

For each detail type, do not stop after the default first page. Set a bounded
`page_count`, advance `page_no` through the response's `total_page`, and retain
the pagination metadata with the result set so a truncated search is detectable.

Use `last_reprt_at=Y` when only the final corrected filing is wanted. Use
`last_reprt_at=N` when preserving the complete submission and correction
history. Retain at least `rcept_no`, `report_nm`, `flr_nm`, `rcept_dt`, and `rm`.

For an `F001` or `F002` filing, `flr_nm` is a strong auditor candidate because
the submitter is ordinarily the accounting firm or audit group. The contract,
however, defines it only as 공시 제출인명 (disclosure submitter name). It is not
a general auditor field and must not be interpreted as one on arbitrary filing
types.

See the canonical [disclosure-search path](../../openapi/paths/ds001/list.json.yaml)
and [response schema](../../openapi/schemas/ds001/2019001.yaml).

### 3. Verify or extract from the original filing

Use `GET /document.xml` with the selected `rcept_no`. A successful response is a
ZIP containing the original disclosure document. The signed independent-
auditor section is the authoritative fallback for the auditor identity and
opinion when:

- strict verification of `flr_nm` is required;
- the structured operation returns `013`;
- the structured response contains placeholders such as `adtor=-`; or
- the opinion is needed but exists only in the standalone report body.

The guide does not define a stable ZIP-member schema for auditor extraction.
Preserve the source archive, bound extraction, and treat document parsing as an
empirical adapter rather than as a documented OpenAPI field contract. Older
documents may also require a CP949 decoding fallback even when their declaration
suggests UTF-8.

See the canonical [original-document path](../../openapi/paths/ds001/document.xml.yaml).

```text
structured auditor/opinion result is substantive
    -> use adtor and adt_opinion

status 013 or missing/placeholder auditor
    -> search F001 and F002 over an explicit date range
    -> paginate each search through total_page
    -> retain flr_nm as an auditor candidate and retain rcept_no
    -> fetch document.xml when authoritative identity or opinion is required
```

## Related operations

`GET /adtServcCnclsSttus.json` also contains an `adtor` field, but it extracts
audit-engagement information from the same periodic-report population. It is a
useful corroborating structured source, not a general fallback for standalone
audit-report filers. See its [path](../../openapi/paths/ds002/adtServcCnclsSttus.json.yaml)
and [schema](../../openapi/schemas/ds002/2020010.yaml).

Despite its name, `GET /accnutAdtorNonAdtServcCnclsSttus.json` documents
non-audit service contract fields and no auditor-name field. `GET /fnlttXbrl.xml`
is limited to a narrower filing population and documents no auditor field.

## Live observations from 2026-07-18

These are bounded observations of the live service, not completeness guarantees
or additions to the guide-derived contract. The request coordinates, response
hashes, allowlisted filing rows, and bounded document-member assertions are in
the dated [sanitized evidence manifest](evidence/auditor-2026-07-18.json).

### 누가의료기

For corporation code `00571818`, the structured auditor-opinion operation
returned status `013` for sampled annual-report coordinates in 2015, 2020, and
2024. The audit-engagement and non-audit-service operations also returned `013`
for the sampled 2024 annual-report coordinate.

Disclosure search instead found standalone audit filings across report periods
from 2013 through 2025. The submitter history included 다산회계법인,
삼정회계법인, and 한울회계법인.

In this observation, the separate `F001` and `F002` requests returned
byte-identical complete `F`-population responses for 누가의료기,
삼성바이오에피스, and 삼성디스플레이. The rows included both 감사보고서 and
연결감사보고서 names. Preserve the requested filter and response hash as
provenance, but classify the returned filing from `report_nm`; do not label a
row as `F001` or `F002` merely because it arrived under that request.

Original documents were sampled from the latest selected 2015, 2020, and 2025
filings. In every sample, the expected `flr_nm` value and an independent-auditor
section marker occurred in the same bounded archive member: 다산회계법인,
삼정회계법인, and 한울회계법인, respectively. The 2015 and 2020 members
required the CP949-compatible decoding fallback; the 2025 member decoded as
UTF-8.

### Cross-company behavior

The same two populations appeared across a varied listed and unlisted sample:

| Observed case | Structured periodic-report result | Standalone or document fallback |
| --- | --- | --- |
| 삼성바이오에피스 | `013` | `F001`/`F002` identified 삼정회계법인 |
| 삼성디스플레이 | `013` | The typed searches identified recent 삼정회계법인, earlier 안진회계법인, and older 삼일회계법인 filings |
| 삼성전자 | Substantive auditor rows | The sampled `F001` search returned only 2013–2015 report periods |
| 교보생명보험 | Substantive auditor rows | The sampled `F001` search returned only 2013–2014 report periods |
| 롯데건설 | Successful rows with `adtor=-` | The selected periodic-report archive contained 안진회계법인 and auditor-section markers in the same members |
| 카카오모빌리티 | Substantive auditor rows | `F001` covered 2018–2021 report periods, including later corrections |

Corrections were present in the external-audit corpus. Consumers collecting
history should preserve every receipt and model correction relationships rather
than overwrite by company and report period. Consumers selecting only the
currently effective filing should request final reports while retaining the
selected receipt as provenance.

The sampled document responses used `application/x-msdownload`, despite the
guide-derived ZIP representation. ZIP signatures and archive structure were
therefore checked positively before bounded in-memory inspection. One historical
member also used an absolute-looking name. Consumers must never extract source
member paths directly; normalize or replace names inside an extraction boundary.

## What this evidence does not establish

- `flr_nm` is not contractually guaranteed to equal the legal auditor, even for
  `F001` and `F002`; document verification remains the strict path.
- The sampled companies and periods do not prove completeness or stable
  historical availability for either filing population.
- A status `013` observation is specific to its request coordinate and does not
  imply absence from the other population.
- The sampled `F001`/`F002` responses do not establish that the detail filter
  reliably partitions the live result set.
- Opinion text in original documents has no documented stable extraction
  schema and requires normalization rules backed by additional fixtures.
- Co-occurrence of a filer name and auditor-section marker in one member is
  evidence for the fallback, not a general parser for the signed legal identity.
