import assert from "node:assert/strict";
import test from "node:test";

import {
  assertCanonicalObservation,
  distinctBaselineIdentityValues,
  encodedQuery,
  jsonObservation,
  xmlObservation,
} from "./probe-multi-company.mjs";

const SERIALIZATION_CASE = {
  endpoint: "fnlttCmpnyIndx",
  corpCodes: ["00164742", "00159023"],
  arguments: {
    bsns_year: "2023",
    reprt_code: "11014",
    idx_cl_code: "M210000",
  },
};
const TEST_KEY = "k".repeat(40);

test("builds the canonical comma-separated query", () => {
  assert.equal(
    encodedQuery(SERIALIZATION_CASE, "json", "comma-separated", TEST_KEY),
    `https://opendart.fss.or.kr/api/fnlttCmpnyIndx.json?crtfc_key=${TEST_KEY}&corp_code=00164742,00159023&bsns_year=2023&reprt_code=11014&idx_cl_code=M210000`,
  );
});

test("builds the repeated-key control query", () => {
  assert.equal(
    encodedQuery(SERIALIZATION_CASE, "xml", "repeated-query-key", TEST_KEY),
    `https://opendart.fss.or.kr/api/fnlttCmpnyIndx.xml?crtfc_key=${TEST_KEY}&corp_code=00164742&corp_code=00159023&bsns_year=2023&reprt_code=11014&idx_cl_code=M210000`,
  );
});

test("builds a single-value baseline query", () => {
  assert.equal(
    encodedQuery(
      { ...SERIALIZATION_CASE, corpCodes: [SERIALIZATION_CASE.corpCodes[0]] },
      "json",
      "single-value-baseline",
      TEST_KEY,
    ),
    `https://opendart.fss.or.kr/api/fnlttCmpnyIndx.json?crtfc_key=${TEST_KEY}&corp_code=00164742&bsns_year=2023&reprt_code=11014&idx_cl_code=M210000`,
  );
});

test("extracts stock-code identities from multi-account JSON", () => {
  assert.deepEqual(
    jsonObservation(
      JSON.stringify({
        status: "000",
        metadata: { stock_code: "999999" },
        list: [
          { stock_code: "005930", account_nm: "자산총계" },
          { stock_code: "000660", account_nm: "자산총계" },
          { stock_code: "005930", account_nm: "부채총계" },
        ],
      }),
      "stock_code",
      {},
    ),
    {
      apiStatus: "000",
      returnedCorpCodes: ["000660", "005930"],
    },
  );
});

test("extracts company-code identities from multi-index XML", () => {
  assert.deepEqual(
    xmlObservation(
      "<result><status>000</status><list><corp_code>00164742</corp_code></list><list><corp_code>00159023</corp_code></list></result>",
      "corp_code",
    ),
    {
      apiStatus: "000",
      returnedCorpCodes: ["00159023", "00164742"],
    },
  );
});

test("rejects truncated XML instead of accepting parser recovery", () => {
  assert.throws(
    () =>
      xmlObservation(
        "<result><status>000</status><list><corp_code>00164742</corp_code></list>",
        "corp_code",
      ),
    /malformed XML/,
  );
});

test("rejects extra identities in canonical responses", () => {
  assert.throws(
    () =>
      assertCanonicalObservation({
        request: {
          logicalOperationId: "DS003-2022002",
          endpoint: "fnlttCmpnyIndx",
          format: "json",
        },
        response: {
          httpStatus: 200,
          mediaType: "application/json",
          apiStatus: "000",
          missingExpectedIdentityValues: [],
          unexpectedIdentityValues: ["99999999"],
        },
      }),
    /did not return both guide-example companies/,
  );
});

test("rejects shared identities across single-company baselines", () => {
  const testCase = {
    logicalOperationId: "DS003-2019017",
    endpoint: "fnlttMultiAcnt",
    corpCodes: ["00334624", "00126380"],
  };
  const response = {
    httpStatus: 200,
    mediaType: "application/json",
    apiStatus: "000",
    returnedIdentityValues: ["005930"],
  };
  assert.throws(
    () =>
      distinctBaselineIdentityValues(testCase, [
        { response },
        { response: { ...response } },
      ]),
    /did not produce distinct identities/,
  );
});
