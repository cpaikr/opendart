import { realpathSync } from "node:fs";
import { lstat, readFile, readdir } from "node:fs/promises";
import { dirname, isAbsolute, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { parseArgs } from "node:util";

import { parseDocument } from "yaml";

const DEFAULT_ROOT = fileURLToPath(
  new URL("../openapi/openapi.yaml", import.meta.url),
);
const OPENAPI_VERSION = "3.2.0";
const MULTI_COMPANY_OPERATIONS = new Set(["DS003-2019017", "DS003-2022002"]);
const MULTI_COMPANY_GUIDE_EVIDENCE = {
  "DS003-2019017": {
    serializedValue: "00334624,00126380",
    values: ["00334624", "00126380"],
  },
  "DS003-2022002": {
    serializedValue: "00164742,00159023",
    values: ["00164742", "00159023"],
  },
};
const MULTI_COMPANY_MAXIMUM = {
  value: 100,
  source: "official-guide-message-code",
  messageCode: "021",
  description: "조회 가능한 회사 개수가 초과하였습니다.(최대 100건)",
};
const ZIP_ERROR_OBSERVATION = {
  observedAt: "2026-07-17",
  requestCondition: "invalid-40-character-api-key",
  httpStatus: 200,
  contentTypeHeader: "application/xml;charset=UTF-8",
  apiStatus: "010",
};
const TOTAL_COUNT_SOURCE_DIAGNOSTICS = [
  {
    code: "field-name-description-conflict",
    severity: "warning",
    message: "공식 가이드의 필드 명칭과 출력 설명이 서로 다른 의미를 가리킵니다.",
    evidence: { name: "총 건수", description: "총 페이지 수" },
  },
];
const EMPLOYEE_COUNT_SOURCE_DIAGNOSTICS = [
  [
    {
      code: "field-name-description-conflict",
      severity: "warning",
      message: "공식 가이드의 필드 명칭과 출력 설명이 서로 다른 의미를 가리킵니다.",
      evidence: { name: "정규직 수", description: "상근, 비상근" },
    },
  ],
  [
    {
      code: "field-name-description-conflict",
      severity: "warning",
      message: "공식 가이드의 필드 명칭과 출력 설명이 서로 다른 의미를 가리킵니다.",
      evidence: {
        name: "정규직 단시간 근로자 수",
        description: "대표이사, 이사, 사외이사 등",
      },
    },
  ],
];
const EXPECTED_RESPONSE_SOURCE_DIAGNOSTICS = {
  "schemas/ds001/2019001.yaml": [TOTAL_COUNT_SOURCE_DIAGNOSTICS],
  "schemas/ds002/2019011.yaml": EMPLOYEE_COUNT_SOURCE_DIAGNOSTICS,
};

const EXPECTED = {
  logicalEndpoints: 85,
  physicalPaths: 167,
  groupCounts: {
    DS001: 4,
    DS002: 30,
    DS003: 7,
    DS004: 2,
    DS005: 36,
    DS006: 6,
  },
  requestArguments: 337,
  responseFields: 2383,
  messageCodes: 13,
  referenceTables: {
    "DS001-2019001": { title: "상세 유형", rows: 61 },
    "DS003-2020001": { title: "재무제표구분", rows: 26 },
  },
};

class CatalogError extends Error {
  constructor(message, context = {}) {
    super(message);
    this.name = "CatalogError";
    this.context = context;
  }
}

function assert(condition, message, context = {}) {
  if (!condition) throw new CatalogError(message, context);
}

function options() {
  const { values } = parseArgs({
    options: {
      root: { type: "string", default: DEFAULT_ROOT },
      "structural-only": { type: "boolean", default: false },
    },
    strict: true,
  });
  return {
    root: resolve(values.root),
    structuralOnly: values["structural-only"],
  };
}

async function yamlFile(file) {
  const text = await readFile(file, "utf8");
  const document = parseDocument(text, { strict: true, uniqueKeys: true });
  if (document.errors.length) {
    throw new CatalogError("YAML parsing failed", {
      file,
      errors: document.errors.map((error) => error.message),
    });
  }
  return { value: document.toJS(), text };
}

async function filesBelow(directory) {
  const result = [];
  async function visit(current) {
    for (const entry of await readdir(current, { withFileTypes: true })) {
      const path = join(current, entry.name);
      if (entry.isDirectory()) await visit(path);
      else result.push(path);
    }
  }
  await visit(directory);
  return result.sort();
}

function refFile(fromFile, ref, rootDir) {
  assert(typeof ref === "string", "$ref must be a string", { fromFile, ref });
  assert(!/^[A-Za-z][A-Za-z0-9+.-]*:/.test(ref), "URI-scheme $ref is forbidden", {
    fromFile,
    ref,
  });
  const [filePart] = ref.split("#", 1);
  assert(
    !filePart?.startsWith("//") && !isAbsolute(filePart || ""),
    "Absolute $ref is forbidden",
    { fromFile, ref },
  );
  const target = filePart ? resolve(dirname(fromFile), filePart) : fromFile;
  const fromRoot = relative(rootDir, target);
  assert(
    fromRoot !== ".." && !fromRoot.startsWith(`..${sep}`) && !isAbsolute(fromRoot),
    "$ref escapes the OpenDART specification directory",
    { fromFile, ref, rootDir, target },
  );
  const physicalRoot = realpathSync(rootDir);
  const physicalTarget = realpathSync(target);
  const physicalFromRoot = relative(physicalRoot, physicalTarget);
  assert(
    physicalFromRoot !== ".." &&
      !physicalFromRoot.startsWith(`..${sep}`) &&
      !isAbsolute(physicalFromRoot),
    "$ref resolves outside the OpenDART specification directory",
    { fromFile, ref, rootDir, target, physicalRoot, physicalTarget },
  );
  return target;
}

function allRefs(value, pointer = "#", refs = []) {
  if (Array.isArray(value)) {
    value.forEach((item, index) => allRefs(item, `${pointer}/${index}`, refs));
  } else if (value && typeof value === "object") {
    for (const [key, child] of Object.entries(value)) {
      const childPointer = `${pointer}/${key}`;
      if (key === "$ref") refs.push({ pointer: childPointer, value: child });
      allRefs(child, childPointer, refs);
    }
  }
  return refs;
}

function extensionValues(value, key, values = []) {
  if (Array.isArray(value)) {
    value.forEach((item) => extensionValues(item, key, values));
  } else if (value && typeof value === "object") {
    for (const [childKey, child] of Object.entries(value)) {
      if (childKey === key) values.push(child);
      extensionValues(child, key, values);
    }
  }
  return values;
}

function normalizedPath(path) {
  return path.split(sep).join("/");
}

function sameValue(left, right) {
  return JSON.stringify(left) === JSON.stringify(right);
}

function parameterSourceDiagnostics(logicalOperationId, endpointDescription, argument) {
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
  if (logicalOperationId === "DS003-2019019" && argument.key === "rcept_no") {
    diagnostics.push({
      code: "inconsistent-length-across-endpoints",
      severity: "warning",
      message: "동일한 접수번호 요청키의 공식 가이드 타입 길이가 엔드포인트마다 다릅니다.",
      evidence: [
        { logicalOperationId, documentedType: argument.documentedType },
        { logicalOperationId: "DS001-2019003", documentedType: "STRING(14)" },
      ],
    });
  }
  if (MULTI_COMPANY_OPERATIONS.has(logicalOperationId) && argument.key === "corp_code") {
    diagnostics.push({
      code: "request-cardinality-conflict",
      severity: "warning",
      message: "공식 가이드는 복수회사 조회를 설명하지만 요청 인자는 단일 STRING(8)로 표기합니다.",
      evidence: {
        documentedType: argument.documentedType,
        endpointDescription,
      },
      handling: "modeled-from-guide-test-example",
    });
  }
  return diagnostics;
}

function parameterSerialization(logicalOperationId, argument) {
  if (!MULTI_COMPANY_OPERATIONS.has(logicalOperationId) || argument.key !== "corp_code") {
    return undefined;
  }
  const evidence = MULTI_COMPANY_GUIDE_EVIDENCE[logicalOperationId];
  return {
    status: "guide-example-supported",
    wireFormat: "comma-separated",
    delimiter: ",",
    guideEvidence: {
      source: "official-guide-test-form",
      ...evidence,
      maximumItems: MULTI_COMPANY_MAXIMUM,
    },
    authenticatedVerification: {
      status: "pending",
    },
  };
}

function normalizedParameters(logicalOperationId, endpointDescription, documentedArguments) {
  return documentedArguments
    .filter((argument) => argument.key !== "crtfc_key")
    .map((argument) => {
      const sourceDiagnostics = parameterSourceDiagnostics(
        logicalOperationId,
        endpointDescription,
        argument,
      );
      const serialization = parameterSerialization(logicalOperationId, argument);
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
                  dataValue: serialization.guideEvidence.values,
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

function mediaTypeFor(outputFormat) {
  if (outputFormat === "JSON") return "application/json";
  if (outputFormat === "XML") return "application/xml";
  if (outputFormat === "Zip FILE (binary)") return "application/zip";
  throw new CatalogError("Unknown documented output format", { outputFormat });
}

async function main() {
  const { root: rootFile, structuralOnly } = options();
  const rootDir = dirname(rootFile);
  const markerFile = join(rootDir, ".opendart-spec-output");
  let markerStat;
  let markerContent;
  try {
    markerStat = await lstat(markerFile);
    markerContent = await readFile(markerFile, "utf8");
  } catch (error) {
    throw new CatalogError("Generated-output ownership marker is missing", {
      markerFile,
      cause: error.message,
    });
  }
  assert(
    markerStat.isFile() && !markerStat.isSymbolicLink(),
    "Generated-output marker must be a regular non-symlink file",
    { markerFile },
  );
  assert(markerContent === "opendart-spec-v1\n", "Generated-output marker changed", {
    markerFile,
  });
  const { value: root } = await yamlFile(rootFile);

  assert(root.openapi === OPENAPI_VERSION, "Unexpected OpenAPI version", {
    actual: root.openapi,
  });
  assert(root.paths && typeof root.paths === "object", "Root paths object is missing");
  const catalogMetadata = root["x-opendart"];
  const catalogCheckedAt = catalogMetadata?.source?.checkedAt;
  assert(
    typeof catalogCheckedAt === "string" && root.info?.version === catalogCheckedAt,
    "Catalog version and source check date differ",
    { infoVersion: root.info?.version, catalogCheckedAt },
  );
  assert(
    catalogMetadata?.inventory?.physicalPathCount === Object.keys(root.paths).length,
    "Root inventory path count differs from the paths object",
    {
      inventory: catalogMetadata?.inventory?.physicalPathCount,
      actual: Object.keys(root.paths).length,
    },
  );
  assert(
    structuralOnly || Object.keys(root.paths).length === EXPECTED.physicalPaths,
    "Physical path count is incomplete",
    { expected: EXPECTED.physicalPaths, actual: Object.keys(root.paths).length },
  );

  const logical = new Map();
  const operationIds = new Set();
  const referencedPathFiles = new Set();
  const referencedSchemaFiles = new Set();
  const parsedFiles = new Map([[rootFile, root]]);
  let parameterSourceDiagnosticCount = 0;
  let multiCompanySerializationCount = 0;
  let pendingSerializationVerificationCount = 0;

  for (const [pathKey, pathReference] of Object.entries(root.paths)) {
    assert(Object.keys(pathReference).length === 1 && pathReference.$ref, "Path entry must contain one local $ref", {
      pathKey,
      pathReference,
    });
    const pathFile = refFile(rootFile, pathReference.$ref, rootDir);
    referencedPathFiles.add(pathFile);
    const { value: pathItem, text } = await yamlFile(pathFile);
    parsedFiles.set(pathFile, pathItem);
    assert(pathItem.get && Object.keys(pathItem).length === 1, "Path fragment must contain one GET operation", {
      pathKey,
      pathFile,
    });

    const operation = pathItem.get;
    const source = operation["x-opendart"];
    assert(source, "Operation provenance is missing", { pathKey, pathFile });
    assert(source.logicalOperationId, "Logical operation identity is missing", { pathKey, pathFile });
    assert(source.documentedPageHeading, "Documented page heading is missing", {
      pathKey,
      pathFile,
    });
    assert(source.source?.guideUrl && source.source?.checkedAt, "Source URL or checked date is missing", {
      pathKey,
      pathFile,
    });
    assert(
      source.source.checkedAt === catalogCheckedAt,
      "Operation source check date differs from the catalog",
      { pathKey, expected: catalogCheckedAt, actual: source.source.checkedAt },
    );
    assert(
      source.logicalOperationId === `${source.apiGroupCode}-${source.apiId}`,
      "Logical operation identity differs from its group and API ID",
      {
        pathKey,
        logicalOperationId: source.logicalOperationId,
        apiGroupCode: source.apiGroupCode,
        apiId: source.apiId,
      },
    );
    assert(!operationIds.has(operation.operationId), "operationId is duplicated", {
      operationId: operation.operationId,
      pathKey,
    });
    operationIds.add(operation.operationId);

    const documented = source.documentedBasicInfo;
    assert(
      sameValue(source.sourceTableHeaders, {
        basicInfo: ["메서드", "요청URL", "인코딩", "출력포멧"],
        requestArguments: ["요청키", "명칭", "타입", "필수여부", "값설명"],
        responseFields: ["응답키", "명칭", "출력설명"],
      }),
      "Documented source-table headers changed",
      { pathKey, sourceTableHeaders: source.sourceTableHeaders },
    );
    assert(documented?.method === "GET", "Documented method changed", { pathKey, documented });
    assert(documented?.encoding === "UTF-8", "Documented encoding changed", { pathKey, documented });
    assert(
      sameValue(
        operation.parameters || [],
        normalizedParameters(
          source.logicalOperationId,
          operation.description,
          source.documentedRequestArguments || [],
        ),
      ),
      "OpenAPI parameters differ from the documented request arguments",
      { pathKey, pathFile },
    );
    parameterSourceDiagnosticCount += (operation.parameters || []).reduce(
      (count, parameter) =>
        count + (parameter["x-opendart-source-diagnostics"]?.length || 0),
      0,
    );
    multiCompanySerializationCount += (operation.parameters || []).filter(
      (parameter) =>
        parameter["x-opendart-serialization"]?.status === "guide-example-supported",
    ).length;
    pendingSerializationVerificationCount += (operation.parameters || []).filter(
      (parameter) =>
        parameter["x-opendart-serialization"]?.authenticatedVerification?.status ===
        "pending",
    ).length;
    assert(
      sameValue(operation.security, [{ crtfcKey: [] }]),
      "Operation security does not use the documented query authentication key",
      { pathKey, pathFile, security: operation.security },
    );
    const expectedPath = new URL(documented.requestUrl).pathname.slice("/api".length);
    assert(expectedPath === pathKey, "OpenAPI path does not match the documented URL", {
      pathKey,
      documentedUrl: documented.requestUrl,
    });
    assert(
      (pathKey.endsWith(".json") && documented.outputFormat === "JSON") ||
        (pathKey.endsWith(".xml") && ["XML", "Zip FILE (binary)"].includes(documented.outputFormat)),
      "Path suffix and documented output format disagree",
      { pathKey, outputFormat: documented.outputFormat },
    );

    const mediaType = mediaTypeFor(documented.outputFormat);
    const response = operation.responses?.default;
    const expectedMediaTypes =
      documented.outputFormat === "Zip FILE (binary)"
        ? [mediaType, "application/xml"]
        : [mediaType];
    assert(
      response && sameValue(Object.keys(response.content || {}), expectedMediaTypes),
      "Response media type differs from the documented output format",
      { pathKey, expected: expectedMediaTypes, actual: Object.keys(response?.content || {}) },
    );
    const responseSchema = response.content[mediaType].schema;
    const expectedSchemaFile = resolve(
      rootDir,
      "schemas",
      source.apiGroupCode.toLowerCase(),
      `${source.apiId}.yaml`,
    );
    if (documented.outputFormat === "Zip FILE (binary)") {
      assert(
        sameValue(responseSchema, {}),
        "ZIP response does not use the canonical raw-binary schema",
        { pathKey, responseSchema },
      );
      const xmlError = response.content["application/xml"];
      assert(
        xmlError.schema?.$ref?.endsWith("#/OpenDartXmlError") &&
          refFile(pathFile, xmlError.schema.$ref, rootDir) ===
            resolve(rootDir, "components", "schemas.yaml"),
        "ZIP XML error response does not use the shared empirical schema",
        { pathKey, schema: xmlError.schema },
      );
      assert(
        xmlError["x-opendart-content-type-status"] ===
          "empirically-observed-error-response" &&
          sameValue(xmlError["x-opendart-observation"], ZIP_ERROR_OBSERVATION),
        "ZIP XML error observation is missing or changed",
        { pathKey, xmlError },
      );
      assert(
        response["x-opendart-documented-response-schema"]?.component ===
          `${source.apiGroupCode}_${source.apiId}_Response`,
        "ZIP operation points to the wrong documented response-table component",
        { pathKey },
      );
    } else {
      assert(responseSchema?.$ref, "Structured response schema reference is missing", {
        pathKey,
      });
      assert(
        refFile(pathFile, responseSchema.$ref, rootDir) === expectedSchemaFile,
        "Operation points to another endpoint's response schema",
        {
          pathKey,
          expected: normalizedPath(relative(rootDir, expectedSchemaFile)),
          actual: normalizedPath(
            relative(rootDir, refFile(pathFile, responseSchema.$ref, rootDir)),
          ),
        },
      );
    }
    assert(!text.includes("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"), "Test-form placeholder leaked into a path file", {
      pathFile,
    });

    const identity = source.logicalOperationId;
    const current = logical.get(identity);
    if (!current) {
      logical.set(identity, {
        apiGroupCode: source.apiGroupCode,
        apiId: source.apiId,
        requestArguments: source.documentedRequestArguments,
        referenceTables: source.referenceTables,
        formats: new Set([documented.outputFormat]),
        paths: [pathKey],
      });
    } else {
      assert(current.apiGroupCode === source.apiGroupCode && current.apiId === source.apiId, "Logical identity metadata differs by format", {
        identity,
      });
      assert(sameValue(current.requestArguments, source.documentedRequestArguments), "Request arguments differ by format", {
        identity,
      });
      assert(sameValue(current.referenceTables, source.referenceTables), "Reference tables differ by format", {
        identity,
      });
      current.formats.add(documented.outputFormat);
      current.paths.push(pathKey);
    }

    for (const ref of allRefs(pathItem)) {
      assert(!/^https?:/i.test(ref.value), "Remote $ref is forbidden", {
        file: pathFile,
        ...ref,
      });
      const target = refFile(pathFile, ref.value, rootDir);
      if (target.startsWith(`${join(rootDir, "schemas")}${sep}`)) {
        referencedSchemaFiles.add(target);
      }
    }
  }

  assert(
    catalogMetadata.inventory.logicalEndpointCount === logical.size,
    "Root inventory logical count differs from operations",
    {
      inventory: catalogMetadata.inventory.logicalEndpointCount,
      actual: logical.size,
    },
  );
  assert(structuralOnly || logical.size === EXPECTED.logicalEndpoints, "Logical endpoint count is incomplete", {
    expected: EXPECTED.logicalEndpoints,
    actual: logical.size,
  });
  assert(structuralOnly || parameterSourceDiagnosticCount === 37, "Request source-diagnostic count changed", {
    expected: 37,
    actual: parameterSourceDiagnosticCount,
  });
  assert(structuralOnly || multiCompanySerializationCount === 4, "Multi-company serialization count changed", {
    expected: 4,
    actual: multiCompanySerializationCount,
  });
  assert(structuralOnly || pendingSerializationVerificationCount === 4, "Pending multi-company verification count changed", {
    expected: 4,
    actual: pendingSerializationVerificationCount,
  });

  const groupCounts = Object.fromEntries(Object.keys(EXPECTED.groupCounts).map((group) => [group, 0]));
  let requestArgumentCount = 0;
  for (const [identity, operation] of logical) {
    groupCounts[operation.apiGroupCode] += 1;
    requestArgumentCount += operation.requestArguments.length;
    const formats = [...operation.formats].sort();
    const expectedFormats = formats.includes("Zip FILE (binary)")
      ? ["Zip FILE (binary)"]
      : ["JSON", "XML"];
    assert(sameValue(formats, expectedFormats), "Logical endpoint representation set is incomplete", {
      identity,
      expected: expectedFormats,
      actual: formats,
    });

    const expectedReference = EXPECTED.referenceTables[identity];
    if (expectedReference) {
      assert(operation.referenceTables.length === 1, "Expected endpoint reference table is missing", {
        identity,
      });
      assert(
        operation.referenceTables[0].title === expectedReference.title &&
          operation.referenceTables[0].rows.length === expectedReference.rows,
        "Reference table changed",
        { identity, expected: expectedReference, actual: operation.referenceTables[0] },
      );
    } else {
      assert(operation.referenceTables.length === 0, "Unexpected endpoint reference table appeared", {
        identity,
        referenceTables: operation.referenceTables.map((table) => table.title),
      });
    }
  }
  assert(
    sameValue(groupCounts, catalogMetadata.inventory.groupCounts),
    "Root inventory group counts differ from operations",
    { inventory: catalogMetadata.inventory.groupCounts, actual: groupCounts },
  );
  assert(structuralOnly || sameValue(groupCounts, EXPECTED.groupCounts), "API group counts changed", {
    expected: EXPECTED.groupCounts,
    actual: groupCounts,
  });
  assert(structuralOnly || requestArgumentCount === EXPECTED.requestArguments, "Request argument total changed", {
    expected: EXPECTED.requestArguments,
    actual: requestArgumentCount,
  });

  const schemas = root.components?.schemas;
  assert(schemas?.OpenDartStatus?.$ref, "Shared status schema is missing");
  assert(schemas?.OpenDartXmlError?.$ref, "Shared XML error schema is missing");
  const sharedSchemaNames = new Set(["OpenDartStatus", "OpenDartXmlError"]);
  const endpointSchemaEntries = Object.entries(schemas).filter(
    ([name]) => !sharedSchemaNames.has(name),
  );
  const binaryOperationCount = [...logical.values()].filter((operation) =>
    operation.formats.has("Zip FILE (binary)"),
  ).length;
  assert(endpointSchemaEntries.length === binaryOperationCount, "Binary endpoint schema component count changed", {
    expected: binaryOperationCount,
    actual: endpointSchemaEntries.length,
  });

  for (const [name, schemaReference] of endpointSchemaEntries) {
    assert(schemaReference.$ref, "Endpoint schema component must use a local $ref", { name });
    const schemaFile = refFile(rootFile, schemaReference.$ref, rootDir);
    const identity = name.match(/^(DS\d{3})_(\d+)_Response$/);
    assert(identity, "Binary endpoint schema component has an unexpected name", { name });
    const expectedSchemaFile = resolve(
      rootDir,
      "schemas",
      identity[1].toLowerCase(),
      `${identity[2]}.yaml`,
    );
    assert(schemaFile === expectedSchemaFile, "Binary component points to another endpoint's schema", {
      name,
      expected: normalizedPath(relative(rootDir, expectedSchemaFile)),
      actual: normalizedPath(relative(rootDir, schemaFile)),
    });
    referencedSchemaFiles.add(schemaFile);
  }
  assert(referencedSchemaFiles.size === logical.size, "Endpoint schema reference coverage is incomplete", {
    expected: logical.size,
    actual: referencedSchemaFiles.size,
  });

  let responseFieldCount = 0;
  let responseSourceDiagnosticCount = 0;
  let expectedResponseSourceDiagnosticCount = 0;
  for (const schemaFile of [...referencedSchemaFiles].sort()) {
    const { value: schema } = await yamlFile(schemaFile);
    parsedFiles.set(schemaFile, schema);
    const fields = schema["x-opendart"]?.responseFields;
    assert(Array.isArray(fields) && fields.length, "Raw response-field rows are missing", {
      schemaFile,
    });
    fields.forEach((field, index) => {
      assert(field.sourceIndex === index, "Response-field source order is not preserved", {
        schemaFile,
        expected: index,
        actual: field.sourceIndex,
      });
    });
    responseFieldCount += fields.length;
    const sourceRoots = fields.filter((field) => field.depth === 0);
    assert(
      sourceRoots.length === 1 && sourceRoots[0].key === "result",
      "Documented response root changed",
      { schemaFile, sourceRoots },
    );
    assert(
      schema["x-opendart"].sourceRootKey === sourceRoots[0].key &&
        schema.xml?.name === sourceRoots[0].key &&
        schema.xml?.nodeType === "element",
      "Response schema XML root does not match its documented source root",
      {
        schemaFile,
        expected: sourceRoots[0].key,
        sourceRootKey: schema["x-opendart"].sourceRootKey,
        xmlName: schema.xml?.name,
        xmlNodeType: schema.xml?.nodeType,
      },
    );
    const sourceDiagnostics = extensionValues(schema, "x-opendart-source-diagnostics");
    const relativeSchemaFile = normalizedPath(relative(rootDir, schemaFile));
    const expectedSourceDiagnostics =
      EXPECTED_RESPONSE_SOURCE_DIAGNOSTICS[relativeSchemaFile] || [];
    assert(
      sameValue(sourceDiagnostics, expectedSourceDiagnostics),
      "Response source diagnostics differ from the curated contradictions",
      { schemaFile, expectedSourceDiagnostics, sourceDiagnostics },
    );
    expectedResponseSourceDiagnosticCount += expectedSourceDiagnostics.reduce(
      (count, diagnostics) => count + diagnostics.length,
      0,
    );
    responseSourceDiagnosticCount += sourceDiagnostics.reduce(
      (count, diagnostics) => count + diagnostics.length,
      0,
    );
  }
  assert(structuralOnly || responseFieldCount === EXPECTED.responseFields, "Response field total changed", {
    expected: EXPECTED.responseFields,
    actual: responseFieldCount,
  });
  assert(responseSourceDiagnosticCount === expectedResponseSourceDiagnosticCount, "Response source-diagnostic count changed", {
    expected: expectedResponseSourceDiagnosticCount,
    actual: responseSourceDiagnosticCount,
  });

  const statusFile = refFile(rootFile, schemas.OpenDartStatus.$ref, rootDir);
  assert(
    schemas.OpenDartXmlError.$ref.endsWith("#/OpenDartXmlError") &&
      refFile(rootFile, schemas.OpenDartXmlError.$ref, rootDir) === statusFile,
    "Shared XML error component points to an unexpected schema",
  );
  const { value: sharedSchemas } = await yamlFile(statusFile);
  parsedFiles.set(statusFile, sharedSchemas);
  const status = sharedSchemas.OpenDartStatus;
  assert(status.enum?.length === EXPECTED.messageCodes, "Message-code inventory changed", {
    expected: EXPECTED.messageCodes,
    actual: status.enum?.length,
  });
  assert(
    Object.keys(status["x-opendart-code-descriptions"] || {}).length === EXPECTED.messageCodes,
    "Message-code descriptions are incomplete",
  );
  assert(
    sameValue(sharedSchemas.OpenDartXmlError, {
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
    }),
    "Shared XML error schema changed",
    { actual: sharedSchemas.OpenDartXmlError },
  );

  const actualPathFiles = new Set(
    (await filesBelow(join(rootDir, "paths"))).filter((file) => file.endsWith(".yaml")),
  );
  const actualSchemaFiles = new Set(
    (await filesBelow(join(rootDir, "schemas"))).filter((file) => file.endsWith(".yaml")),
  );
  assert(sameValue([...actualPathFiles].sort(), [...referencedPathFiles].sort()), "Orphaned or missing path fragments detected", {
    actual: [...actualPathFiles].map((file) => normalizedPath(relative(rootDir, file))),
    referenced: [...referencedPathFiles].map((file) => normalizedPath(relative(rootDir, file))),
  });
  assert(sameValue([...actualSchemaFiles].sort(), [...referencedSchemaFiles].sort()), "Orphaned or missing schema fragments detected", {
    actual: [...actualSchemaFiles].map((file) => normalizedPath(relative(rootDir, file))),
    referenced: [...referencedSchemaFiles].map((file) => normalizedPath(relative(rootDir, file))),
  });

  for (const [file, value] of parsedFiles) {
    for (const ref of allRefs(value)) {
      refFile(file, ref.value, rootDir);
    }
  }

  process.stdout.write(
    `${JSON.stringify(
      {
        root: rootFile,
        openapi: root.openapi,
        logicalEndpoints: logical.size,
        physicalPaths: Object.keys(root.paths).length,
        requestArguments: requestArgumentCount,
        responseFields: responseFieldCount,
        messageCodes: status.enum.length,
        groupCounts,
      },
      null,
      2,
    )}\n`,
  );
}

main().catch((error) => {
  const detail =
    error instanceof CatalogError
      ? { message: error.message, ...error.context }
      : { message: error.message, stack: error.stack };
  process.stderr.write(`${JSON.stringify(detail, null, 2)}\n`);
  process.exitCode = 1;
});
