import { readFile, readdir } from "node:fs/promises";
import { dirname, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { parseArgs } from "node:util";

import { parseDocument } from "yaml";

const DEFAULT_ROOT = fileURLToPath(
  new URL("../../docs/opendart/openapi.yaml", import.meta.url),
);

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
    options: { root: { type: "string", default: DEFAULT_ROOT } },
    strict: true,
  });
  return { root: resolve(values.root) };
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

function refFile(fromFile, ref) {
  assert(typeof ref === "string", "$ref must be a string", { fromFile, ref });
  assert(!/^https?:/i.test(ref), "Remote $ref is forbidden", { fromFile, ref });
  const [filePart] = ref.split("#", 1);
  return filePart ? resolve(dirname(fromFile), filePart) : fromFile;
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

function normalizedPath(path) {
  return path.split(sep).join("/");
}

function sameValue(left, right) {
  return JSON.stringify(left) === JSON.stringify(right);
}

function normalizedParameters(documentedArguments) {
  return documentedArguments
    .filter((argument) => argument.key !== "crtfc_key")
    .map((argument) => ({
      name: argument.key,
      in: "query",
      required: argument.required === "Y",
      description: argument.description || argument.name,
      schema: { type: "string" },
      "x-opendart-korean-name": argument.name,
      "x-opendart-documented-type": argument.documentedType,
      "x-opendart-documented-required": argument.required,
    }));
}

function mediaTypeFor(outputFormat) {
  if (outputFormat === "JSON") return "application/json";
  if (outputFormat === "XML") return "application/xml";
  if (outputFormat === "Zip FILE (binary)") return "application/zip";
  throw new CatalogError("Unknown documented output format", { outputFormat });
}

async function main() {
  const { root: rootFile } = options();
  const rootDir = dirname(rootFile);
  const { value: root } = await yamlFile(rootFile);

  assert(root.openapi === "3.1.2", "Unexpected OpenAPI version", {
    actual: root.openapi,
  });
  assert(root.paths && typeof root.paths === "object", "Root paths object is missing");
  assert(
    Object.keys(root.paths).length === EXPECTED.physicalPaths,
    "Physical path count is incomplete",
    { expected: EXPECTED.physicalPaths, actual: Object.keys(root.paths).length },
  );

  const logical = new Map();
  const operationIds = new Set();
  const referencedPathFiles = new Set();
  const referencedSchemaFiles = new Set();
  const parsedFiles = new Map([[rootFile, root]]);

  for (const [pathKey, pathReference] of Object.entries(root.paths)) {
    assert(Object.keys(pathReference).length === 1 && pathReference.$ref, "Path entry must contain one local $ref", {
      pathKey,
      pathReference,
    });
    const pathFile = refFile(rootFile, pathReference.$ref);
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
      sameValue(operation.parameters || [], normalizedParameters(source.documentedRequestArguments || [])),
      "OpenAPI parameters differ from the documented request arguments",
      { pathKey, pathFile },
    );
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
    assert(
      response && sameValue(Object.keys(response.content || {}), [mediaType]),
      "Response media type differs from the documented output format",
      { pathKey, expected: mediaType, actual: Object.keys(response?.content || {}) },
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
        sameValue(responseSchema, { type: "string", format: "binary" }),
        "ZIP response is missing its raw-binary schema",
        { pathKey, responseSchema },
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
        refFile(pathFile, responseSchema.$ref) === expectedSchemaFile,
        "Operation points to another endpoint's response schema",
        {
          pathKey,
          expected: normalizedPath(relative(rootDir, expectedSchemaFile)),
          actual: normalizedPath(relative(rootDir, refFile(pathFile, responseSchema.$ref))),
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
      const target = refFile(pathFile, ref.value);
      if (target.startsWith(`${join(rootDir, "schemas")}${sep}`)) {
        referencedSchemaFiles.add(target);
      }
    }
  }

  assert(logical.size === EXPECTED.logicalEndpoints, "Logical endpoint count is incomplete", {
    expected: EXPECTED.logicalEndpoints,
    actual: logical.size,
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
  assert(sameValue(groupCounts, EXPECTED.groupCounts), "API group counts changed", {
    expected: EXPECTED.groupCounts,
    actual: groupCounts,
  });
  assert(requestArgumentCount === EXPECTED.requestArguments, "Request argument total changed", {
    expected: EXPECTED.requestArguments,
    actual: requestArgumentCount,
  });

  const schemas = root.components?.schemas;
  assert(schemas?.OpenDartStatus?.$ref, "Shared status schema is missing");
  const endpointSchemaEntries = Object.entries(schemas).filter(([name]) => name !== "OpenDartStatus");
  assert(endpointSchemaEntries.length === 3, "Binary endpoint schema component count changed", {
    expected: 3,
    actual: endpointSchemaEntries.length,
  });

  for (const [name, schemaReference] of endpointSchemaEntries) {
    assert(schemaReference.$ref, "Endpoint schema component must use a local $ref", { name });
    const schemaFile = refFile(rootFile, schemaReference.$ref);
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
  assert(referencedSchemaFiles.size === EXPECTED.logicalEndpoints, "Endpoint schema reference coverage is incomplete", {
    expected: EXPECTED.logicalEndpoints,
    actual: referencedSchemaFiles.size,
  });

  let responseFieldCount = 0;
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
  }
  assert(responseFieldCount === EXPECTED.responseFields, "Response field total changed", {
    expected: EXPECTED.responseFields,
    actual: responseFieldCount,
  });

  const statusFile = refFile(rootFile, schemas.OpenDartStatus.$ref);
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
      assert(!/^https?:/i.test(ref.value), "Remote $ref is forbidden", { file, ...ref });
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
