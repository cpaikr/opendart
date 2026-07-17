import { execFile } from "node:child_process";
import {
  lstat,
  mkdir,
  mkdtemp,
  readFile,
  realpath,
  rename,
  rm,
  writeFile,
} from "node:fs/promises";
import { homedir, tmpdir } from "node:os";
import { basename, dirname, join, parse as parsePath, relative, resolve, sep } from "node:path";
import { setTimeout as delay } from "node:timers/promises";
import { fileURLToPath } from "node:url";
import { parseArgs, promisify } from "node:util";

import { load } from "cheerio";
import { stringify } from "yaml";

const GUIDE_ORIGIN = "https://opendart.fss.or.kr";
const API_SERVER = `${GUIDE_ORIGIN}/api`;
const OPENAPI_VERSION = "3.2.0";
const DEFAULT_OUTPUT = fileURLToPath(
  new URL("../../docs/opendart", import.meta.url),
);
const REPOSITORY_ROOT = fileURLToPath(new URL("../../", import.meta.url));
const CHECK_SCRIPT = fileURLToPath(new URL("./check-opendart.mjs", import.meta.url));
const REDOCLY_CLI = fileURLToPath(
  new URL("./node_modules/@redocly/cli/bin/cli.js", import.meta.url),
);
const REDOCLY_CONFIG = fileURLToPath(
  new URL("../../docs/opendart/redocly.yaml", import.meta.url),
);
const OUTPUT_MARKER = ".opendart-spec-output";
const OUTPUT_MARKER_CONTENT = "dartdb-opendart-spec-v1\n";
const MANAGED_OUTPUTS = [
  "paths",
  "schemas",
  "components",
  "openapi.yaml",
  OUTPUT_MARKER,
];
const INVALIDATED_OUTPUTS = ["generated/openapi.bundle.yaml"];
const MULTI_COMPANY_OPERATIONS = new Set(["DS003-2019017", "DS003-2022002"]);
const ZIP_ERROR_OBSERVATION = {
  observedAt: "2026-07-17",
  requestCondition: "invalid-40-character-api-key",
  httpStatus: 200,
  contentTypeHeader: "application/xml;charset=UTF-8",
  apiStatus: "010",
};
const execFileAsync = promisify(execFile);

const GROUPS = [
  { code: "DS001", name: "공시정보", expectedCount: 4 },
  { code: "DS002", name: "정기보고서 주요정보", expectedCount: 30 },
  { code: "DS003", name: "정기보고서 재무정보", expectedCount: 7 },
  { code: "DS004", name: "지분공시 종합정보", expectedCount: 2 },
  { code: "DS005", name: "주요사항보고서 주요정보", expectedCount: 36 },
  { code: "DS006", name: "증권신고서 주요정보", expectedCount: 6 },
];

const EXPECTED_FULL_TOTALS = {
  logicalEndpoints: 85,
  physicalPaths: 167,
  requestArguments: 337,
  responseFields: 2383,
  messageCodes: 13,
};

const STANDARD_CAPTIONS = new Set([
  "기본 정보",
  "요청 인자",
  "응답 결과",
  "OpenAPI 테스트",
  "메시지 설명",
]);

class SourceError extends Error {
  constructor(message, context = {}) {
    super(message);
    this.name = "SourceError";
    this.context = context;
  }
}

function checkedAtInSeoul() {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Seoul",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date());
}

function parseOptions() {
  const { values } = parseArgs({
    options: {
      output: { type: "string", default: DEFAULT_OUTPUT },
      only: { type: "string", multiple: true, default: [] },
      "checked-at": { type: "string", default: checkedAtInSeoul() },
    },
    strict: true,
  });

  if (!validCalendarDate(values["checked-at"])) {
    throw new SourceError("--checked-at must use YYYY-MM-DD", {
      value: values["checked-at"],
    });
  }

  const output = resolve(values.output);
  const only = new Set(values.only);
  if (only.size && output === resolve(DEFAULT_OUTPUT)) {
    throw new SourceError("--only requires a non-canonical --output directory");
  }

  return {
    output,
    checkedAt: values["checked-at"],
    only,
  };
}

function validCalendarDate(value) {
  const match = value.match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (!match) return false;
  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  const leapYear = year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
  const daysByMonth = [
    31,
    leapYear ? 29 : 28,
    31,
    30,
    31,
    30,
    31,
    31,
    30,
    31,
    30,
    31,
  ];
  return month >= 1 && month <= 12 && day >= 1 && day <= daysByMonth[month - 1];
}

async function fetchHtml(url) {
  let lastError;

  for (let attempt = 1; attempt <= 3; attempt += 1) {
    try {
      const response = await fetch(url, {
        headers: {
          Accept: "text/html,application/xhtml+xml",
          "User-Agent": "dartdb-opendart-spec/1.0",
        },
        redirect: "follow",
        signal: AbortSignal.timeout(30_000),
      });

      if (response.ok) {
        return await response.text();
      }

      const retryable = response.status === 429 || response.status >= 500;
      const error = new SourceError("OpenDART guide request failed", {
        url,
        status: response.status,
        attempt,
      });
      if (!retryable) throw error;
      lastError = error;
    } catch (error) {
      if (error instanceof SourceError && error.context.status < 500) throw error;
      lastError = error;
    }

    if (attempt < 3) await delay(attempt * 500);
  }

  throw new SourceError("OpenDART guide request failed after retries", {
    url,
    cause: lastError?.message,
  });
}

async function mapLimit(items, limit, task) {
  const results = new Array(items.length);
  let nextIndex = 0;

  async function worker() {
    while (nextIndex < items.length) {
      const index = nextIndex;
      nextIndex += 1;
      results[index] = await task(items[index], index);
    }
  }

  await Promise.all(
    Array.from({ length: Math.min(limit, items.length) }, () => worker()),
  );
  return results;
}

function normalizedText(value) {
  const lines = value
    .replaceAll("\u00a0", " ")
    .replaceAll("\r", "")
    .split("\n")
    .map((line) => line.replace(/[\t ]+/g, " ").trim());

  const compact = [];
  for (const line of lines) {
    if (line || compact.at(-1)) compact.push(line);
  }
  return compact.join("\n").trim();
}

function nodeText($, node) {
  if (!node) return "";
  const clone = $(node).clone();
  clone.find("br").replaceWith("\n");
  clone.find("p, li").each((_, child) => $(child).append("\n"));
  return normalizedText(clone.text());
}

function expandedBodyRows($, table) {
  const rowSpans = [];
  return $(table)
    .find("tbody > tr")
    .toArray()
    .map((row) => {
      const values = [];
      for (let column = 0; column < rowSpans.length; column += 1) {
        const span = rowSpans[column];
        if (!span) continue;
        values[column] = span.value;
        span.remaining -= 1;
        if (span.remaining === 0) rowSpans[column] = null;
      }

      let column = 0;
      for (const cell of $(row).children("th, td").toArray()) {
        while (values[column] !== undefined) column += 1;
        const value = nodeText($, cell);
        const colspan = Number($(cell).attr("colspan") || 1);
        const rowspan = Number($(cell).attr("rowspan") || 1);
        for (let offset = 0; offset < colspan; offset += 1) {
          values[column + offset] = value;
          if (rowspan > 1) {
            rowSpans[column + offset] = { value, remaining: rowspan - 1 };
          }
        }
        column += colspan;
      }
      return values;
    });
}

function tableData($, table) {
  return {
    caption: nodeText($, $(table).find("caption").first()[0]),
    headers: $(table)
      .find("thead tr")
      .first()
      .children("th, td")
      .toArray()
      .map((cell) => nodeText($, cell)),
    rows: expandedBodyRows($, table),
    sourceHasSpans: $(table).find("tbody [rowspan], tbody [colspan]").length > 0,
  };
}

function requiredTable(tables, caption, endpoint) {
  const table = tables.find((candidate) => candidate.caption === caption);
  if (!table) {
    throw new SourceError("Required guide table is missing", {
      logicalOperationId: endpoint.logicalOperationId,
      caption,
      sourceUrl: endpoint.sourceUrl,
    });
  }
  return table;
}

function validateTableShape(table, expectedHeaders, expectedWidth, endpoint) {
  if (
    expectedHeaders &&
    JSON.stringify(table.headers) !== JSON.stringify(expectedHeaders)
  ) {
    throw new SourceError("Guide table headers changed", {
      logicalOperationId: endpoint.logicalOperationId,
      caption: table.caption,
      expectedHeaders,
      actualHeaders: table.headers,
      sourceUrl: endpoint.sourceUrl,
    });
  }
  table.rows.forEach((row, rowIndex) => {
    if (row.length !== expectedWidth) {
      throw new SourceError("Guide table row width changed", {
        logicalOperationId: endpoint.logicalOperationId,
        caption: table.caption,
        rowIndex,
        expectedWidth,
        actualWidth: row.length,
        sourceUrl: endpoint.sourceUrl,
      });
    }
  });
}

function responseFields($, responseTable) {
  return $(responseTable)
    .find("tbody > tr")
    .toArray()
    .map((row, sourceIndex) => {
      const cells = $(row).children("th, td").toArray();
      const keyCell = cells[0];
      const indentClass = $(keyCell)
        .find('span[class*="mgl"]')
        .first()
        .attr("class");
      const indentValue = Number(indentClass?.match(/mgl(\d+)/)?.[1]);
      const depth = indentValue === 5 ? 0 : Number.isFinite(indentValue) ? indentValue / 20 : null;
      const icon = $(keyCell).find("i").first().attr("class") || null;

      return {
        sourceIndex,
        key: nodeText($, keyCell),
        name: nodeText($, cells[1]),
        description: nodeText($, cells[2]),
        depth,
        sourceIndentClass: indentClass || null,
        sourceIconClass: icon,
        sourceKind: icon === "iconArrow" ? "container" : "field",
      };
    });
}

function messageCodes($, messageTable) {
  return $(messageTable)
    .find("tbody > tr")
    .toArray()
    .map((row) => {
      const cells = $(row).children("th, td").toArray();
      const label = nodeText($, cells[0]);
      const code = label.match(/\d{3}/)?.[0];
      if (!code) throw new SourceError("Message-code row has no three-digit code", { label });
      return { code, description: nodeText($, cells[1]) };
    });
}

function guideTestRequestArguments($) {
  return $("#testTable input[type='hidden'][name]")
    .toArray()
    .map((input) => ({
      key: $(input).attr("name"),
      value: $(input).attr("value") || "",
    }))
    .filter((argument) => argument.key && argument.key !== "crtfc_key");
}

function sectionNotes($) {
  return $(".DGCont")
    .toArray()
    .map((section) => {
      const heading = nodeText($, $(section).find(".titleWrapToggle h5").first()[0]);
      const content = $(section).find(".contWrapToggle").first().clone();
      content.find("table, form, script, style, button, input, select, textarea").remove();
      return { section: heading, text: nodeText($, content[0]) };
    })
    .filter((note) => note.section && note.text);
}

async function groupInventory(group) {
  const mainUrl = `${GUIDE_ORIGIN}/guide/main.do?apiGrpCd=${group.code}`;
  const $ = load(await fetchHtml(mainUrl));
  const endpoints = [];

  $("table tbody > tr").each((_, row) => {
    const link = $(row).find('a[href*="/guide/detail.do"]').first();
    if (!link.length) return;
    const cells = $(row).children("td").toArray();
    if (cells.length < 3) return;

    const sourceUrl = new URL(link.attr("href"), GUIDE_ORIGIN);
    const apiGroupCode = sourceUrl.searchParams.get("apiGrpCd");
    const apiId = sourceUrl.searchParams.get("apiId");
    if (apiGroupCode !== group.code || !apiId) {
      throw new SourceError("Endpoint link identity does not match its group", {
        group: group.code,
        sourceUrl: sourceUrl.toString(),
      });
    }

    endpoints.push({
      apiGroupCode,
      apiGroupName: group.name,
      apiId,
      logicalOperationId: `${apiGroupCode}-${apiId}`,
      name: nodeText($, cells[1]),
      description: nodeText($, cells[2]),
      sourceUrl: sourceUrl.toString(),
      groupSourceUrl: mainUrl,
    });
  });

  if (endpoints.length !== group.expectedCount) {
    throw new SourceError("Endpoint group count changed", {
      group: group.code,
      expected: group.expectedCount,
      actual: endpoints.length,
      sourceUrl: mainUrl,
    });
  }

  return endpoints;
}

async function extractEndpoint(endpoint) {
  const $ = load(await fetchHtml(endpoint.sourceUrl));
  const hiddenApiId = $('input[name="apiId"]').first().attr("value");
  const hiddenGroupCode = $('input[name="apiGrpCd"]').first().attr("value");
  if (hiddenApiId && hiddenApiId !== endpoint.apiId) {
    throw new SourceError("Detail page apiId does not match its link", {
      expected: endpoint.apiId,
      actual: hiddenApiId,
      sourceUrl: endpoint.sourceUrl,
    });
  }
  if (hiddenGroupCode && hiddenGroupCode !== endpoint.apiGroupCode) {
    throw new SourceError("Detail page apiGrpCd does not match its link", {
      expected: endpoint.apiGroupCode,
      actual: hiddenGroupCode,
      sourceUrl: endpoint.sourceUrl,
    });
  }

  const tableNodes = $(".DGCont table").toArray();
  const tables = tableNodes.map((table) => tableData($, table));
  const basic = requiredTable(tables, "기본 정보", endpoint);
  const requests = requiredTable(tables, "요청 인자", endpoint);
  const response = requiredTable(tables, "응답 결과", endpoint);
  const messages = requiredTable(tables, "메시지 설명", endpoint);
  validateTableShape(
    basic,
    ["메서드", "요청URL", "인코딩", "출력포멧"],
    4,
    endpoint,
  );
  validateTableShape(
    requests,
    ["요청키", "명칭", "타입", "필수여부", "값설명"],
    5,
    endpoint,
  );
  validateTableShape(
    response,
    ["응답키", "명칭", "출력설명"],
    3,
    endpoint,
  );
  validateTableShape(messages, null, 2, endpoint);
  const responseNode = tableNodes.find(
    (table) => nodeText($, $(table).find("caption").first()[0]) === "응답 결과",
  );
  const messageNode = tableNodes.find(
    (table) => nodeText($, $(table).find("caption").first()[0]) === "메시지 설명",
  );

  const basicInfo = basic.rows.map((row) => ({
    method: row[0],
    requestUrl: row[1],
    encoding: row[2],
    outputFormat: row[3],
  }));
  const requestArguments = requests.rows.map((row) => ({
    key: row[0],
    name: row[1],
    documentedType: row[2],
    required: row[3],
    description: row[4],
  }));
  const referenceTables = tables
    .filter((table) => table.caption && !STANDARD_CAPTIONS.has(table.caption))
    .map((table) => {
      validateTableShape(table, null, table.headers.length, endpoint);
      return {
        title: table.caption,
        headers: table.headers,
        rows: table.rows,
        normalization: table.sourceHasSpans ? "rowspan-and-colspan-expanded" : "none",
      };
    });

  if (!basicInfo.length || basicInfo.some((row) => !row.requestUrl)) {
    throw new SourceError("Endpoint has no documented request URL", {
      logicalOperationId: endpoint.logicalOperationId,
      sourceUrl: endpoint.sourceUrl,
    });
  }

  return {
    ...endpoint,
    pageHeading: nodeText($, $(".DGTopTitle p").first()[0]),
    basicInfo,
    requestArguments,
    responseFields: responseFields($, responseNode),
    referenceTables,
    sectionNotes: sectionNotes($),
    sourceTableHeaders: {
      basicInfo: basic.headers,
      requestArguments: requests.headers,
      responseFields: response.headers,
    },
    guideTestRequestArguments: guideTestRequestArguments($),
    messageCodes: messageCodes($, messageNode),
  };
}

function sameMessageCodes(left, right) {
  return JSON.stringify(left) === JSON.stringify(right);
}

function sourceDescription(row) {
  const parts = [];
  if (row.name) parts.push(row.name);
  if (row.description && row.description !== row.name) parts.push(row.description);
  return parts.join("\n\n") || undefined;
}

function parameterSourceDiagnostics(endpoint, argument) {
  const diagnostics = [];
  if (
    argument.key === "bsns_year" &&
    argument.documentedType === "STRING(1)" &&
    argument.description.includes("4자리")
  ) {
    diagnostics.push({
      code: "documented-length-conflict",
      severity: "warning",
      message: "공식 가이드의 타입 길이와 값 설명의 자리수가 서로 다릅니다.",
      evidence: {
        documentedType: argument.documentedType,
        description: argument.description,
      },
    });
  }
  if (endpoint.logicalOperationId === "DS003-2019019" && argument.key === "rcept_no") {
    diagnostics.push({
      code: "inconsistent-length-across-endpoints",
      severity: "warning",
      message: "동일한 접수번호 요청키의 공식 가이드 타입 길이가 엔드포인트마다 다릅니다.",
      evidence: [
        {
          logicalOperationId: endpoint.logicalOperationId,
          documentedType: argument.documentedType,
        },
        {
          logicalOperationId: "DS001-2019003",
          documentedType: "STRING(14)",
        },
      ],
    });
  }
  if (MULTI_COMPANY_OPERATIONS.has(endpoint.logicalOperationId) && argument.key === "corp_code") {
    diagnostics.push({
      code: "request-cardinality-conflict",
      severity: "warning",
      message: "공식 가이드는 복수회사 조회를 설명하지만 요청 인자는 단일 STRING(8)로 표기합니다.",
      evidence: {
        documentedType: argument.documentedType,
        endpointDescription: endpoint.description,
      },
      handling: "modeled-from-guide-test-example",
    });
  }
  return diagnostics;
}

function responseFieldSourceDiagnostics(endpoint, rows) {
  if (endpoint.logicalOperationId !== "DS001-2019001") return [];
  const row = rows.find(
    (candidate) =>
      candidate.key === "total_count" &&
      candidate.name === "총 건수" &&
      candidate.description === "총 페이지 수",
  );
  if (!row) return [];
  return [
    {
      code: "field-name-description-conflict",
      severity: "warning",
      message: "공식 가이드의 필드 명칭과 출력 설명이 서로 다른 의미를 가리킵니다.",
      evidence: {
        name: row.name,
        description: row.description,
      },
    },
  ];
}

function parameterSerialization(endpoint, argument) {
  if (!MULTI_COMPANY_OPERATIONS.has(endpoint.logicalOperationId) || argument.key !== "corp_code") {
    return undefined;
  }

  const example = endpoint.guideTestRequestArguments.find(
    (candidate) => candidate.key === "corp_code",
  );
  const values = example?.value.split(",").filter(Boolean) || [];
  if (
    values.length < 2 ||
    values.some((value) => !/^\d{8}$/.test(value)) ||
    example.value !== values.join(",")
  ) {
    throw new SourceError("Multi-company guide example is missing or malformed", {
      logicalOperationId: endpoint.logicalOperationId,
      sourceUrl: endpoint.sourceUrl,
      value: example?.value,
    });
  }

  const maximumMessage = endpoint.messageCodes.find((row) => row.code === "021");
  const maximumItems = Number(maximumMessage?.description.match(/최대\s*(\d+)건/)?.[1]);
  if (!Number.isInteger(maximumItems) || maximumItems < 1) {
    throw new SourceError("Multi-company maximum is missing from message 021", {
      logicalOperationId: endpoint.logicalOperationId,
      sourceUrl: endpoint.sourceUrl,
      message: maximumMessage?.description,
    });
  }

  return {
    status: "guide-example-supported",
    wireFormat: "comma-separated",
    delimiter: ",",
    guideEvidence: {
      source: "official-guide-test-form",
      serializedValue: example.value,
      values,
      maximumItems: {
        value: maximumItems,
        source: "official-guide-message-code",
        messageCode: maximumMessage.code,
        description: maximumMessage.description,
      },
    },
    authenticatedVerification: {
      status: "pending",
    },
  };
}

function normalizedResponseSchema(endpoint) {
  const diagnostics = [];
  const sourceRows = endpoint.responseFields;
  const root = { key: "$root", depth: -1, container: true, rows: [], children: [] };
  const stack = [root];

  function findOrAdd(parent, row, container) {
    let node = parent.children.find((candidate) => candidate.key === row.key);
    if (!node) {
      node = { key: row.key, depth: row.depth, container, rows: [], children: [] };
      parent.children.push(node);
    } else if (node.container !== container) {
      diagnostics.push({
        code: "conflicting-source-kind",
        key: row.key,
        sourceIndex: row.sourceIndex,
      });
      node.container ||= container;
    }
    node.rows.push(row);
    return node;
  }

  for (const row of sourceRows) {
    const normalizedContainer = row.key === "result" || row.key === "list" || row.key === "group";
    if (row.key === "result" && row.sourceKind !== "container") {
      diagnostics.push({
        code: "result-source-icon-is-not-container",
        sourceIndex: row.sourceIndex,
        sourceIconClass: row.sourceIconClass,
      });
    }

    if (!normalizedContainer && stack.at(-1)?.key === "list" && row.depth <= stack.at(-1).depth) {
      diagnostics.push({
        code: "list-child-shares-container-depth",
        key: row.key,
        sourceIndex: row.sourceIndex,
      });
      findOrAdd(stack.at(-1), row, false);
      continue;
    }

    while (stack.length > 1 && stack.at(-1).depth >= row.depth) stack.pop();
    const parent = stack.at(-1);
    const node = findOrAdd(parent, row, normalizedContainer || row.sourceKind === "container");
    if (node.container) stack.push(node);
  }

  const resultNode = root.children.find((child) => child.key === "result");
  const effectiveRoot = resultNode || root;

  function objectSchema(node) {
    const properties = {};
    for (const child of node.children) {
      if (child.container) {
        const collection = child.key === "list" || child.key === "group";
        const nested = objectSchema(child);
        properties[child.key] = collection
          ? {
              type: "array",
              items: nested,
              description: "공식 가이드의 계층 표시를 바탕으로 배열 컨테이너로 정규화했습니다.",
              "x-opendart-normalization": "source-derived-unverified",
            }
          : nested;
      } else if (child.key === "status") {
        properties[child.key] = {
          "$ref": "../../components/schemas.yaml#/OpenDartStatus",
        };
      } else {
        const descriptions = [...new Set(child.rows.map(sourceDescription).filter(Boolean))];
        const sourceDiagnostics = responseFieldSourceDiagnostics(endpoint, child.rows);
        properties[child.key] = {
          ...(descriptions.length ? { description: descriptions.join("\n\n") } : {}),
          "x-opendart-documented-type": "not-specified",
          ...(child.rows[0]?.name
            ? { "x-opendart-korean-name": child.rows[0].name }
            : {}),
          ...(sourceDiagnostics.length
            ? { "x-opendart-source-diagnostics": sourceDiagnostics }
            : {}),
        };
      }
    }
    return {
      type: "object",
      properties,
      additionalProperties: true,
    };
  }

  return {
    ...objectSchema(effectiveRoot),
    ...(resultNode ? { xml: { nodeType: "element", name: resultNode.key } } : {}),
    description:
      "공식 OpenDART 가이드의 응답 결과 표를 정규화한 보수적 스키마입니다. 가이드가 필드 타입을 제공하지 않으므로 타입을 추정하지 않았습니다.",
    "x-opendart": {
      schemaStatus: "source-derived-unverified",
      sourceRootKey: resultNode ? "result" : null,
      diagnostics,
      responseFields: sourceRows,
    },
  };
}

function parameterObjects(endpoint) {
  return endpoint.requestArguments
    .filter((argument) => argument.key !== "crtfc_key")
    .map((argument) => {
      const sourceDiagnostics = parameterSourceDiagnostics(endpoint, argument);
      const serialization = parameterSerialization(endpoint, argument);
      return {
        name: argument.key,
        in: "query",
        required: argument.required === "Y",
        ...(serialization ? { style: "form", explode: false } : {}),
        description: argument.description || argument.name,
        schema: serialization
          ? {
              type: "array",
              minItems: 1,
              maxItems: serialization.guideEvidence.maximumItems.value,
              items: { type: "string" },
            }
          : { type: "string" },
        ...(serialization
          ? {
              examples: {
                officialGuide: {
                  summary: "공식 개발가이드 테스트 예시",
                  dataValue: [...serialization.guideEvidence.values],
                  serializedValue: `corp_code=${serialization.guideEvidence.serializedValue}`,
                },
              },
            }
          : {}),
        "x-opendart-korean-name": argument.name,
        "x-opendart-documented-type": argument.documentedType,
        "x-opendart-documented-required": argument.required,
        ...(sourceDiagnostics.length
          ? { "x-opendart-source-diagnostics": sourceDiagnostics }
          : {}),
        ...(serialization ? { "x-opendart-serialization": serialization } : {}),
      };
    });
}

function outputMediaType(outputFormat) {
  if (outputFormat === "JSON") return "application/json";
  if (outputFormat === "XML") return "application/xml";
  if (outputFormat === "Zip FILE (binary)") return "application/zip";
  throw new SourceError("Unknown documented output format", { outputFormat });
}

function relativeRef(fromFile, toFile) {
  const value = relative(dirname(fromFile), toFile).split(sep).join("/");
  return value.startsWith(".") ? value : `./${value}`;
}

function operationIdFor(url) {
  return `get_${basename(new URL(url).pathname).replace(/[^A-Za-z0-9]+/g, "_")}`;
}

function pathFragment(endpoint, basicRow, pathFile, schemaFile, componentsFile, checkedAt) {
  const binary = basicRow.outputFormat === "Zip FILE (binary)";
  const mediaType = outputMediaType(basicRow.outputFormat);
  const schemaComponent = `${endpoint.apiGroupCode}_${endpoint.apiId}_Response`;
  const schemaRef = relativeRef(pathFile, schemaFile);
  const xmlErrorRef = `${relativeRef(pathFile, componentsFile)}#/OpenDartXmlError`;
  const responseContent = binary
    ? {
        [mediaType]: {
          schema: {},
          "x-opendart-content-type-status": "inferred-from-documented-output-format",
        },
        "application/xml": {
          schema: { "$ref": xmlErrorRef },
          "x-opendart-content-type-status": "empirically-observed-error-response",
          "x-opendart-observation": ZIP_ERROR_OBSERVATION,
        },
      }
    : {
        [mediaType]: {
          schema: { "$ref": schemaRef },
          "x-opendart-content-type-status": "inferred-from-documented-output-format",
        },
      };

  return {
    get: {
      operationId: operationIdFor(basicRow.requestUrl),
      summary: `${endpoint.name} (${basicRow.outputFormat})`,
      description: endpoint.description,
      tags: [endpoint.apiGroupCode],
      externalDocs: {
        description: "OpenDART 공식 개발가이드",
        url: endpoint.sourceUrl,
      },
      security: [{ crtfcKey: [] }],
      parameters: parameterObjects(endpoint),
      responses: {
        default: {
          description: binary
            ? "공식 가이드는 성공 시 ZIP binary 출력을 설명합니다. 무효한 40자리 키를 사용한 실측에서는 동일 응답이 HTTP 200의 XML API 오류를 반환했습니다."
            : "공식 가이드는 API 수준 status/message를 설명하지만 HTTP 상태 코드는 별도로 규정하지 않습니다.",
          content: responseContent,
          "x-opendart-http-status": "not-documented",
          ...(binary
            ? {
                "x-opendart-documented-response-schema": {
                  component: schemaComponent,
                  note: "응답 결과 표 또는 ZIP 내부 XML 필드 설명을 보존합니다. 성공 ZIP 자체의 구조를 뜻하지 않습니다.",
                },
              }
            : {}),
        },
      },
      "x-opendart": {
        logicalOperationId: endpoint.logicalOperationId,
        apiGroupCode: endpoint.apiGroupCode,
        apiGroupName: endpoint.apiGroupName,
        apiId: endpoint.apiId,
        apiName: endpoint.name,
        documentedPageHeading: endpoint.pageHeading,
        source: {
          guideUrl: endpoint.sourceUrl,
          groupUrl: endpoint.groupSourceUrl,
          checkedAt,
        },
        documentedBasicInfo: basicRow,
        documentedRequestArguments: endpoint.requestArguments,
        sourceTableHeaders: endpoint.sourceTableHeaders,
        referenceTables: endpoint.referenceTables,
        sectionNotes: endpoint.sectionNotes,
        coverage: {
          status: "probe-required",
          classification: "not-assessed",
          acquisitionIdentity: "not-documented",
          successfulEmptyCoverage: "not-documented",
          partitionClosure: "not-documented",
          historicalAvailability: "not-documented",
        },
      },
    },
  };
}

function commonSchemas(messageCodeRows) {
  return {
    OpenDartStatus: {
      type: "string",
      enum: messageCodeRows.map((row) => row.code),
      description: "OpenDART API 수준 상태 코드입니다.",
      "x-opendart-code-descriptions": Object.fromEntries(
        messageCodeRows.map((row) => [row.code, row.description]),
      ),
    },
    OpenDartXmlError: {
      type: "object",
      properties: {
        status: { "$ref": "#/OpenDartStatus" },
        message: { type: "string" },
      },
      required: ["status", "message"],
      additionalProperties: true,
      xml: { nodeType: "element", name: "result" },
      description: "ZIP 다운로드 API에서 실측된 XML API 오류 응답입니다.",
      "x-opendart": {
        schemaStatus: "empirically-observed",
        observation: ZIP_ERROR_OBSERVATION,
      },
    },
  };
}

async function writeYaml(file, value) {
  await mkdir(dirname(file), { recursive: true });
  await writeFile(file, stringify(value, { lineWidth: 0 }), "utf8");
}

async function exists(path) {
  try {
    await lstat(path);
    return true;
  } catch (error) {
    if (error.code === "ENOENT") return false;
    throw error;
  }
}

function assertSafeOutput(output) {
  const blocked = new Set([
    parsePath(output).root,
    resolve(homedir()),
    resolve(tmpdir()),
    resolve(REPOSITORY_ROOT),
  ]);
  if (blocked.has(resolve(output))) {
    throw new SourceError("Refusing to publish into a broad directory", { output });
  }
}

async function assertSafePhysicalOutput(output) {
  const physicalOutput = await realpath(output);
  const blocked = new Set(
    await Promise.all(
      [parsePath(output).root, homedir(), tmpdir(), REPOSITORY_ROOT].map((path) =>
        realpath(path),
      ),
    ),
  );
  if (blocked.has(physicalOutput)) {
    throw new SourceError("Refusing to publish into a broad physical directory", {
      output,
      physicalOutput,
    });
  }
}

async function publishGenerated(staging, output) {
  assertSafeOutput(output);
  await mkdir(output, { recursive: true });
  await assertSafePhysicalOutput(output);

  const outputStat = await lstat(output);
  if (!outputStat.isDirectory() || outputStat.isSymbolicLink()) {
    throw new SourceError("Refusing to publish through a non-directory or symlink", {
      output,
    });
  }

  const generatedDirectory = join(output, "generated");
  if (await exists(generatedDirectory)) {
    const generatedStat = await lstat(generatedDirectory);
    if (!generatedStat.isDirectory() || generatedStat.isSymbolicLink()) {
      throw new SourceError("Refusing to invalidate a bundle through an unsafe path", {
        output,
        path: generatedDirectory,
      });
    }
  }

  const existingManaged = [];
  for (const name of [...MANAGED_OUTPUTS, ...INVALIDATED_OUTPUTS]) {
    if (await exists(join(output, name))) existingManaged.push(name);
  }
  if (existingManaged.length) {
    const markerPath = join(output, OUTPUT_MARKER);
    let markerStat;
    try {
      markerStat = await lstat(markerPath);
    } catch (error) {
      if (error.code !== "ENOENT") throw error;
    }
    const markerValid =
      markerStat?.isFile() &&
      !markerStat.isSymbolicLink() &&
      (await readFile(markerPath, "utf8")) === OUTPUT_MARKER_CONTENT;
    if (!markerValid) {
      throw new SourceError("Refusing to replace an unowned output directory", {
        output,
        existingManaged,
        requiredMarker: OUTPUT_MARKER,
      });
    }
  }

  const backup = await mkdtemp(join(dirname(output), ".opendart-backup-"));
  const movedOld = [];
  const movedNew = [];
  let publishSucceeded = false;
  let rollbackSucceeded = false;
  try {
    for (const name of MANAGED_OUTPUTS) {
      const target = join(output, name);
      if (!(await exists(target))) continue;
      await rename(target, join(backup, name));
      movedOld.push(name);
    }
    for (const name of MANAGED_OUTPUTS) {
      const source = join(staging, name);
      if (!(await exists(source))) continue;
      await rename(source, join(output, name));
      movedNew.push(name);
    }
    await rm(join(output, "generated", "openapi.bundle.yaml"), { force: true });
    publishSucceeded = true;
  } catch (error) {
    try {
      for (const name of movedNew.reverse()) {
        await rm(join(output, name), { recursive: true, force: true });
      }
      for (const name of movedOld.reverse()) {
        await rename(join(backup, name), join(output, name));
      }
      rollbackSucceeded = true;
    } catch (rollbackError) {
      throw new SourceError("Publishing failed and rollback is incomplete", {
        output,
        backup,
        publishError: error.message,
        rollbackError: rollbackError.message,
      });
    }
    throw error;
  } finally {
    if (publishSucceeded || rollbackSucceeded) {
      await rm(backup, { recursive: true, force: true });
    }
  }
}

async function generate(endpoints, options) {
  const output = options.output;
  const paths = {};
  const seenPaths = new Set();
  const seenOperationIds = new Set();
  const schemaFiles = new Map();
  const rootFile = join(output, "openapi.yaml");
  const componentsFile = join(output, "components", "schemas.yaml");

  for (const endpoint of endpoints) {
    const groupDir = endpoint.apiGroupCode.toLowerCase();
    const schemaFile = join(output, "schemas", groupDir, `${endpoint.apiId}.yaml`);
    schemaFiles.set(endpoint.logicalOperationId, schemaFile);
    await writeYaml(schemaFile, normalizedResponseSchema(endpoint));

    for (const basicRow of endpoint.basicInfo) {
      if (basicRow.method !== "GET" || basicRow.encoding !== "UTF-8") {
        throw new SourceError("Unexpected method or encoding", {
          logicalOperationId: endpoint.logicalOperationId,
          basicRow,
        });
      }

      const url = new URL(basicRow.requestUrl);
      if (`${url.origin}/api` !== API_SERVER || !url.pathname.startsWith("/api/")) {
        throw new SourceError("Request URL is outside the documented API server", {
          logicalOperationId: endpoint.logicalOperationId,
          requestUrl: basicRow.requestUrl,
        });
      }
      const pathKey = url.pathname.slice("/api".length);
      if (seenPaths.has(pathKey)) {
        throw new SourceError("Duplicate physical API path", { pathKey });
      }
      seenPaths.add(pathKey);

      const operationId = operationIdFor(basicRow.requestUrl);
      if (seenOperationIds.has(operationId)) {
        throw new SourceError("Duplicate operationId", { operationId });
      }
      seenOperationIds.add(operationId);

      const pathFile = join(output, "paths", groupDir, `${basename(url.pathname)}.yaml`);
      await writeYaml(
        pathFile,
        pathFragment(
          endpoint,
          basicRow,
          pathFile,
          schemaFile,
          componentsFile,
          options.checkedAt,
        ),
      );
      paths[pathKey] = {
        "$ref": relativeRef(rootFile, pathFile),
      };
    }
  }

  const codes = endpoints[0]?.messageCodes || [];
  await writeYaml(componentsFile, commonSchemas(codes));

  const groupCounts = Object.fromEntries(
    GROUPS.map((group) => [
      group.code,
      new Set(
        endpoints
          .filter((endpoint) => endpoint.apiGroupCode === group.code)
          .map((endpoint) => endpoint.logicalOperationId),
      ).size,
    ]),
  );

  const root = {
    openapi: OPENAPI_VERSION,
    info: {
      title: "OpenDART API",
      version: options.checkedAt,
      summary: "금융감독원 OpenDART 공식 개발가이드 기반 API 명세",
      description:
        "공식 OpenDART 개발가이드에서 추출한 소스 기반 명세입니다. HTTP 상태, 응답 필드 타입, 완전성 및 수집 의미처럼 가이드가 규정하지 않는 동작은 추정하지 않고 x-opendart에 명시합니다.",
    },
    externalDocs: {
      description: "OpenDART 개발가이드",
      url: `${GUIDE_ORIGIN}/guide/main.do?apiGrpCd=DS001`,
    },
    servers: [{ url: API_SERVER, description: "OpenDART production API" }],
    tags: GROUPS.map((group) => ({
      name: group.code,
      description: group.name,
      externalDocs: {
        description: `${group.name} 개발가이드`,
        url: `${GUIDE_ORIGIN}/guide/main.do?apiGrpCd=${group.code}`,
      },
    })),
    security: [{ crtfcKey: [] }],
    paths: Object.fromEntries(Object.entries(paths).sort(([left], [right]) => left.localeCompare(right))),
    components: {
      securitySchemes: {
        crtfcKey: {
          type: "apiKey",
          in: "query",
          name: "crtfc_key",
          description: "OpenDART에서 발급받은 40자리 API 인증키",
          "x-opendart-documented-type": "STRING(40)",
        },
      },
      schemas: {
        OpenDartStatus: {
          "$ref": `${relativeRef(rootFile, componentsFile)}#/OpenDartStatus`,
        },
        OpenDartXmlError: {
          "$ref": `${relativeRef(rootFile, componentsFile)}#/OpenDartXmlError`,
        },
        ...Object.fromEntries(
          endpoints
            .filter((endpoint) =>
              endpoint.basicInfo.some(
                (row) => row.outputFormat === "Zip FILE (binary)",
              ),
            )
            .map((endpoint) => [
              `${endpoint.apiGroupCode}_${endpoint.apiId}_Response`,
              {
                "$ref": relativeRef(
                  rootFile,
                  schemaFiles.get(endpoint.logicalOperationId),
                ),
              },
            ]),
        ),
      },
    },
    "x-opendart": {
      source: {
        origin: GUIDE_ORIGIN,
        checkedAt: options.checkedAt,
      },
      inventory: {
        logicalEndpointCount: endpoints.length,
        physicalPathCount: seenPaths.size,
        groupCounts,
      },
      extractionPolicy: {
        sourceLanguage: "ko",
        excluded: [
          "site chrome",
          "interactive test controls and transient results",
          "commented-out API sample sections",
        ],
        responseSchema:
          "Source hierarchy is normalized conservatively; raw rows, indentation, icons, and diagnostics are retained on every schema.",
      },
    },
  };

  await writeYaml(rootFile, root);
  await writeFile(join(output, OUTPUT_MARKER), OUTPUT_MARKER_CONTENT, "utf8");
  return { physicalPaths: seenPaths.size, schemaFiles: schemaFiles.size };
}

async function runStagingValidator(name, script, arguments_) {
  try {
    await execFileAsync(process.execPath, [script, ...arguments_], {
      encoding: "utf8",
      maxBuffer: 10 * 1024 * 1024,
    });
  } catch (error) {
    throw new SourceError(`Staged ${name} validation failed`, {
      exitCode: error.code,
      signal: error.signal,
      stdout: error.stdout,
      stderr: error.stderr,
    });
  }
}

async function validateStaging(staging, { requireCompleteCatalog }) {
  const rootFile = join(staging, "openapi.yaml");
  await runStagingValidator("catalog", CHECK_SCRIPT, [
    "--root",
    rootFile,
    ...(!requireCompleteCatalog ? ["--structural-only"] : []),
  ]);
  await runStagingValidator("OpenAPI", REDOCLY_CLI, [
    "lint",
    rootFile,
    "--config",
    REDOCLY_CONFIG,
    "--lint-config",
    "error",
    "--max-problems",
    "1000",
  ]);
}

async function main() {
  const options = parseOptions();
  const inventory = (
    await Promise.all(GROUPS.map((group) => groupInventory(group)))
  ).flat();
  const selected = options.only.size
    ? inventory.filter((endpoint) => options.only.has(endpoint.logicalOperationId))
    : inventory;

  if (options.only.size && selected.length !== options.only.size) {
    const found = new Set(selected.map((endpoint) => endpoint.logicalOperationId));
    throw new SourceError("One or more --only identities were not found", {
      missing: [...options.only].filter((identity) => !found.has(identity)),
    });
  }

  const endpoints = await mapLimit(selected, 6, extractEndpoint);
  const baselineMessages = endpoints[0]?.messageCodes || [];
  for (const endpoint of endpoints) {
    if (!sameMessageCodes(baselineMessages, endpoint.messageCodes)) {
      throw new SourceError("Endpoint message-code tables differ", {
        logicalOperationId: endpoint.logicalOperationId,
        sourceUrl: endpoint.sourceUrl,
      });
    }
  }

  if (!options.only.size) {
    const totals = {
      logicalEndpoints: endpoints.length,
      physicalPaths: endpoints.reduce((sum, endpoint) => sum + endpoint.basicInfo.length, 0),
      requestArguments: endpoints.reduce(
        (sum, endpoint) => sum + endpoint.requestArguments.length,
        0,
      ),
      responseFields: endpoints.reduce(
        (sum, endpoint) => sum + endpoint.responseFields.length,
        0,
      ),
      messageCodes: baselineMessages.length,
    };
    if (JSON.stringify(totals) !== JSON.stringify(EXPECTED_FULL_TOTALS)) {
      throw new SourceError("Official guide inventory changed", {
        expected: EXPECTED_FULL_TOTALS,
        actual: totals,
      });
    }
  }

  await mkdir(dirname(options.output), { recursive: true });
  const staging = await mkdtemp(join(dirname(options.output), ".opendart-stage-"));
  let generated;
  try {
    generated = await generate(endpoints, { ...options, output: staging });
    await validateStaging(staging, { requireCompleteCatalog: !options.only.size });
    await publishGenerated(staging, options.output);
  } finally {
    await rm(staging, { recursive: true, force: true });
  }
  process.stdout.write(
    `${JSON.stringify(
      {
        output: options.output,
        checkedAt: options.checkedAt,
        logicalEndpoints: endpoints.length,
        physicalPaths: generated.physicalPaths,
        schemas: generated.schemaFiles,
      },
      null,
      2,
    )}\n`,
  );
}

main().catch((error) => {
  const detail = error instanceof SourceError ? { message: error.message, ...error.context } : {
    message: error.message,
    stack: error.stack,
  };
  process.stderr.write(`${JSON.stringify(detail, null, 2)}\n`);
  process.exitCode = 1;
});
