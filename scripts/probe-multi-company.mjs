import { setTimeout as delay } from "node:timers/promises";
import { pathToFileURL } from "node:url";

import { load } from "cheerio";
import { SaxesParser } from "saxes";

const API_ORIGIN = "https://opendart.fss.or.kr/api";
const API_KEY_ENV = "OPENDART_API_KEY";
const REQUEST_TIMEOUT_MS = 30_000;

const CASES = [
  {
    logicalOperationId: "DS003-2019017",
    endpoint: "fnlttMultiAcnt",
    corpCodes: ["00334624", "00126380"],
    responseIdentityField: "stock_code",
    arguments: {
      bsns_year: "2018",
      reprt_code: "11011",
    },
  },
  {
    logicalOperationId: "DS003-2022002",
    endpoint: "fnlttCmpnyIndx",
    corpCodes: ["00164742", "00159023"],
    responseIdentityField: "corp_code",
    arguments: {
      bsns_year: "2023",
      reprt_code: "11014",
      idx_cl_code: "M210000",
    },
  },
];

class ProbeError extends Error {
  constructor(message, context = {}) {
    super(message);
    this.name = "ProbeError";
    this.context = context;
  }
}

function checkedAtInSeoul(date = new Date()) {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Seoul",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(date);
}

function apiKey() {
  const value = process.env[API_KEY_ENV];
  if (typeof value !== "string" || value.length !== 40) {
    throw new ProbeError(`${API_KEY_ENV} must contain the 40-character OpenDART API key`);
  }
  return value;
}

function encodedQuery(testCase, format, serialization, key) {
  const pairs = [["crtfc_key", encodeURIComponent(key)]];
  if (serialization === "comma-separated") {
    pairs.push(["corp_code", testCase.corpCodes.join(",")]);
  } else {
    pairs.push(...testCase.corpCodes.map((value) => ["corp_code", value]));
  }
  pairs.push(
    ...Object.entries(testCase.arguments).map(([name, value]) => [name, encodeURIComponent(value)]),
  );
  return `${API_ORIGIN}/${testCase.endpoint}.${format}?${pairs
    .map(([name, value]) => `${name}=${value}`)
    .join("&")}`;
}

function jsonObservation(text, identityField, context) {
  let value;
  try {
    value = JSON.parse(text);
  } catch {
    throw new ProbeError("OpenDART returned invalid JSON", context);
  }

  const corpCodes = new Set(
    (Array.isArray(value.list) ? value.list : [])
      .map((row) => row?.[identityField])
      .filter((child) => typeof child === "string" && child),
  );

  return {
    apiStatus: typeof value.status === "string" ? value.status : null,
    returnedCorpCodes: [...corpCodes].sort(),
  };
}

function xmlObservation(text, identityField) {
  try {
    new SaxesParser().write(text).close();
  } catch {
    throw new ProbeError("OpenDART returned malformed XML");
  }
  const $ = load(text, { xml: true });
  const corpCodes = new Set(
    $("list")
      .find(identityField)
      .toArray()
      .map((element) => $(element).text().trim())
      .filter(Boolean),
  );
  return {
    apiStatus: $("status").first().text().trim() || null,
    returnedCorpCodes: [...corpCodes].sort(),
  };
}

function mediaType(header) {
  return header?.split(";", 1)[0].trim().toLowerCase() || null;
}

async function observe(testCase, format, serialization, key, expectedIdentityValues = null) {
  const context = {
    logicalOperationId: testCase.logicalOperationId,
    endpoint: testCase.endpoint,
    format,
    serialization,
    responseIdentityField: testCase.responseIdentityField,
  };
  let response;
  try {
    response = await fetch(encodedQuery(testCase, format, serialization, key), {
      headers: {
        Accept: format === "json" ? "application/json" : "application/xml",
        "User-Agent": "opendart-serialization-probe/1.0",
      },
      redirect: "error",
      signal: AbortSignal.timeout(REQUEST_TIMEOUT_MS),
    });
  } catch {
    throw new ProbeError("OpenDART request failed before a response was received", context);
  }

  let text;
  try {
    text = await response.text();
  } catch {
    throw new ProbeError("OpenDART response body could not be read", context);
  }

  const parsed =
    format === "json"
      ? jsonObservation(text, testCase.responseIdentityField, context)
      : xmlObservation(text, testCase.responseIdentityField);
  const missingExpectedIdentityValues = expectedIdentityValues
    ? expectedIdentityValues.filter((value) => !parsed.returnedCorpCodes.includes(value))
    : [];
  const unexpectedIdentityValues = expectedIdentityValues
    ? parsed.returnedCorpCodes.filter((value) => !expectedIdentityValues.includes(value))
    : [];

  return {
    request: {
      ...context,
      corpCodes: testCase.corpCodes,
      arguments: testCase.arguments,
    },
    response: {
      httpStatus: response.status,
      mediaType: mediaType(response.headers.get("content-type")),
      apiStatus: parsed.apiStatus,
      returnedIdentityValues: parsed.returnedCorpCodes,
      missingExpectedIdentityValues,
      unexpectedIdentityValues,
    },
  };
}

function assertCanonicalObservation(observation) {
  if (
    observation.response.httpStatus !== 200 ||
    observation.response.apiStatus !== "000" ||
    observation.response.mediaType !==
      (observation.request.format === "json" ? "application/json" : "application/xml") ||
    observation.response.missingExpectedIdentityValues.length ||
    observation.response.unexpectedIdentityValues.length
  ) {
    throw new ProbeError("Comma-separated request did not return both guide-example companies", {
      logicalOperationId: observation.request.logicalOperationId,
      endpoint: observation.request.endpoint,
      format: observation.request.format,
      httpStatus: observation.response.httpStatus,
      mediaType: observation.response.mediaType,
      apiStatus: observation.response.apiStatus,
      missingExpectedIdentityValues: observation.response.missingExpectedIdentityValues,
      unexpectedIdentityValues: observation.response.unexpectedIdentityValues,
    });
  }
}

function repeatedKeyConclusion(observations) {
  const accepted = observations.filter(
    (observation) =>
      observation.response.httpStatus === 200 &&
      observation.response.apiStatus === "000" &&
      observation.response.missingExpectedIdentityValues.length === 0 &&
      observation.response.unexpectedIdentityValues.length === 0,
  ).length;
  if (accepted === observations.length) return "accepted-in-all-controls";
  if (accepted === 0) return "not-accepted-in-controls";
  return "inconsistent-across-controls";
}

function distinctBaselineIdentityValues(testCase, observations) {
  const identities = observations.map((baseline, index) => {
    if (
      baseline.response.httpStatus !== 200 ||
      baseline.response.apiStatus !== "000" ||
      baseline.response.mediaType !== "application/json" ||
      baseline.response.returnedIdentityValues.length !== 1
    ) {
      throw new ProbeError("Single-company baseline did not expose one response identity", {
        logicalOperationId: testCase.logicalOperationId,
        endpoint: testCase.endpoint,
        corpCode: testCase.corpCodes[index],
        httpStatus: baseline.response.httpStatus,
        mediaType: baseline.response.mediaType,
        apiStatus: baseline.response.apiStatus,
        responseIdentityCount: baseline.response.returnedIdentityValues.length,
      });
    }
    return baseline.response.returnedIdentityValues[0];
  });
  const distinct = [...new Set(identities)].sort();
  if (distinct.length !== testCase.corpCodes.length) {
    throw new ProbeError("Single-company baselines did not produce distinct identities", {
      logicalOperationId: testCase.logicalOperationId,
      endpoint: testCase.endpoint,
      requestedCompanyCount: testCase.corpCodes.length,
      responseIdentityCount: distinct.length,
    });
  }
  return distinct;
}

async function main() {
  const key = apiKey();
  const baselines = [];
  const canonical = [];
  const repeatedKeyControls = [];

  for (const testCase of CASES) {
    let expectedIdentityValues = testCase.corpCodes;
    if (testCase.responseIdentityField !== "corp_code") {
      const caseBaselines = [];
      for (const corpCode of testCase.corpCodes) {
        const baseline = await observe(
          { ...testCase, corpCodes: [corpCode] },
          "json",
          "single-value-baseline",
          key,
          null,
        );
        caseBaselines.push(baseline);
        await delay(100);
      }
      expectedIdentityValues = distinctBaselineIdentityValues(testCase, caseBaselines);
      baselines.push(...caseBaselines);
    }

    for (const format of ["json", "xml"]) {
      const commaObservation = await observe(
        testCase,
        format,
        "comma-separated",
        key,
        expectedIdentityValues,
      );
      assertCanonicalObservation(commaObservation);
      canonical.push(commaObservation);
      await delay(100);

      repeatedKeyControls.push(
        await observe(
          testCase,
          format,
          "repeated-query-key",
          key,
          expectedIdentityValues,
        ),
      );
      await delay(100);
    }
  }

  const observedAt = new Date();
  const report = JSON.stringify(
    {
      schemaVersion: 1,
      observedAt: observedAt.toISOString(),
      observedDate: checkedAtInSeoul(observedAt),
      credentialSource: API_KEY_ENV,
      credentialPersisted: false,
      requestCount: baselines.length + canonical.length + repeatedKeyControls.length,
      baselines,
      canonical,
      repeatedKeyControls,
      conclusion: {
        commaSeparated: "verified",
        repeatedQueryKey: repeatedKeyConclusion(repeatedKeyControls),
        repeatedQueryKeyContract: "non-canonical-control-only",
      },
    },
    null,
    2,
  );
  if (report.includes(key)) {
    throw new ProbeError("Sanitized probe report unexpectedly contains the API key");
  }
  process.stdout.write(`${report}\n`);
}

export {
  assertCanonicalObservation,
  distinctBaselineIdentityValues,
  encodedQuery,
  jsonObservation,
  xmlObservation,
};

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main().catch((error) => {
    const report =
      error instanceof ProbeError
        ? { error: error.name, message: error.message, context: error.context }
        : { error: "ProbeError", message: "Unexpected serialization probe failure" };
    process.stderr.write(`${JSON.stringify(report, null, 2)}\n`);
    process.exitCode = 1;
  });
}
