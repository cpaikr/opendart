import assert from "node:assert/strict";
import test from "node:test";

import {
  endpointIdentityFromLink,
  responseFieldSourceDiagnostics,
  trustedGuideUrl,
} from "./sync-opendart.mjs";

test("accepts one exact same-origin endpoint identity", () => {
  const identity = endpointIdentityFromLink(
    "/guide/detail.do?apiGrpCd=DS002&apiId=2019011",
    "DS002",
  );
  assert.equal(
    identity.sourceUrl.toString(),
    "https://opendart.fss.or.kr/guide/detail.do?apiGrpCd=DS002&apiId=2019011",
  );
  assert.equal(identity.apiGroupCode, "DS002");
  assert.equal(identity.apiId, "2019011");
});

test("rejects off-origin and non-guide fetch URLs", () => {
  assert.throws(
    () => trustedGuideUrl("https://example.com/guide/detail.do?apiGrpCd=DS002&apiId=1"),
    /outside the trusted guide surface/,
  );
  assert.throws(
    () => trustedGuideUrl("https://opendart.fss.or.kr/api/list.json"),
    /outside the trusted guide surface/,
  );
});

test("rejects path-like, duplicated, and cross-group endpoint identities", () => {
  assert.throws(
    () => endpointIdentityFromLink(
      "/guide/detail.do?apiGrpCd=DS002&apiId=..%2F..%2Fowned",
      "DS002",
    ),
    /does not match its group/,
  );
  assert.throws(
    () => endpointIdentityFromLink(
      "/guide/detail.do?apiGrpCd=DS002&apiGrpCd=DS003&apiId=2019011",
      "DS002",
    ),
    /does not match its group/,
  );
  assert.throws(
    () => endpointIdentityFromLink(
      "/guide/detail.do?apiGrpCd=DS002&apiId=2019011&apiId=2019012",
      "DS002",
    ),
    /does not match its group/,
  );
  assert.throws(
    () => endpointIdentityFromLink(
      "/guide/detail.do?apiGrpCd=DS003&apiId=2019011",
      "DS002",
    ),
    /does not match its group/,
  );
});

test("records every curated response contradiction and ignores near misses", () => {
  const cases = [
    {
      logicalOperationId: "DS001-2019001",
      row: { key: "total_count", name: "총 건수", description: "총 페이지 수" },
    },
    {
      logicalOperationId: "DS002-2019011",
      row: { key: "rgllbr_co", name: "정규직 수", description: "상근, 비상근" },
    },
    {
      logicalOperationId: "DS002-2019011",
      row: {
        key: "rgllbr_abacpt_labrr_co",
        name: "정규직 단시간 근로자 수",
        description: "대표이사, 이사, 사외이사 등",
      },
    },
  ];

  for (const { logicalOperationId, row } of cases) {
    assert.deepEqual(
      responseFieldSourceDiagnostics({ logicalOperationId }, [row]),
      [{
        code: "field-name-description-conflict",
        severity: "warning",
        message: "공식 가이드의 필드 명칭과 출력 설명이 서로 다른 의미를 가리킵니다.",
        evidence: { name: row.name, description: row.description },
      }],
    );
  }

  assert.deepEqual(
    responseFieldSourceDiagnostics(
      { logicalOperationId: "DS002-2019011" },
      [{
        key: "rgllbr_co",
        name: "정규직 수",
        description: "9,999,999,999",
      }],
    ),
    [],
  );
});
