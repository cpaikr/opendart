//! Process-level contract tests for keyless generated CLI discovery.

use std::path::Path;
use std::process::{Command, Output};

use serde_json::Value;

const MISSING_API_KEY_FIXTURE: &[u8] = include_bytes!("fixtures/missing-api-key.json");
const INVALID_INVOCATION_FIXTURE: &[u8] = include_bytes!("fixtures/invalid-invocation.json");

fn invoke(arguments: &[String], api_key: Option<&str>) -> Output {
    let mut command = Command::new(env!("CARGO_BIN_EXE_opendart"));
    command.args(arguments);
    match api_key {
        Some(value) => {
            command.env("OPENDART_API_KEY", value);
        }
        None => {
            command.env_remove("OPENDART_API_KEY");
        }
    }
    command.output().expect("CLI process should start")
}

fn json_output(arguments: &[&str], expected_code: i32) -> Value {
    let owned: Vec<String> = arguments.iter().map(|value| (*value).to_owned()).collect();
    let output = invoke(&owned, None);
    assert_eq!(
        output.status.code(),
        Some(expected_code),
        "stderr: {}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert!(
        output.stderr.is_empty(),
        "structured commands must not use stderr"
    );
    serde_json::from_slice(&output.stdout).expect("stdout should contain one JSON document")
}

#[test]
fn keyless_home_and_inventory_are_self_describing_and_deterministic() {
    let home = json_output(&[], 0);
    assert_eq!(home["kind"], "home");
    let executable = home["executable"]["path"]
        .as_str()
        .expect("executable path");
    assert!(Path::new(executable).is_absolute());
    assert_eq!(
        home["commands"]["list"]["argv"],
        serde_json::json!(["operations", "list"])
    );
    assert_eq!(home["authentication"]["environment"], "OPENDART_API_KEY");
    assert!(home.get("global_flags").is_none());
    let call_flags = home["call_flags"].as_array().expect("call flags");
    assert_eq!(call_flags.len(), 4);
    assert!(
        call_flags
            .iter()
            .all(|flag| flag["name"] != "--artifact-limit-bytes")
    );
    for flag in call_flags {
        let arguments = vec![
            "call".to_owned(),
            "company".to_owned(),
            "--corp-code".to_owned(),
            "00126380".to_owned(),
            "--representation".to_owned(),
            "json".to_owned(),
            flag["name"].as_str().expect("call flag name").to_owned(),
            "1".to_owned(),
        ];
        let output = invoke(&arguments, None);
        assert_eq!(output.status.code(), Some(1));
        let error: Value = serde_json::from_slice(&output.stdout).expect("error JSON");
        assert_eq!(error["error"]["code"], "missing_api_key");
    }

    let first = json_output(&["operations", "list"], 0);
    let second = json_output(&["operations", "list"], 0);
    assert_eq!(first, second);
    let operations = first["operations"].as_array().expect("operation inventory");
    assert!(!operations.is_empty());
    let names: Vec<_> = operations
        .iter()
        .map(|operation| {
            (
                operation["name"].as_str().expect("name"),
                operation["logical_id"].as_str().expect("logical ID"),
            )
        })
        .collect();
    assert!(names.windows(2).all(|pair| pair[0] <= pair[1]));
}

#[test]
fn every_generated_operation_is_described_equally_by_name_and_logical_id() {
    let inventory = json_output(&["operations", "list"], 0);
    for compact in inventory["operations"]
        .as_array()
        .expect("operation inventory")
    {
        let name = compact["name"].as_str().expect("name");
        let logical_id = compact["logical_id"].as_str().expect("logical ID");
        let by_name = json_output(&["operations", "describe", name], 0);
        let by_id = json_output(&["operations", "describe", logical_id], 0);
        assert_eq!(by_name, by_id, "discovery alias mismatch for {name}");
        assert_eq!(by_name["operation"]["name"], name);
        assert_eq!(by_name["operation"]["logical_id"], logical_id);
        assert_eq!(compact["description"], by_name["operation"]["description"]);
        assert_eq!(
            by_name["operation"]["invocation"]["argv_prefix"],
            serde_json::json!(["call", name])
        );
        assert!(by_name["operation"].get("global_flags").is_none());
        assert_eq!(
            by_name["operation"]["execution_flags"],
            serde_json::json!([
                {"name": "--connect-timeout-ms", "required": false, "shape": "positive_integer"},
                {"name": "--read-timeout-ms", "required": false, "shape": "positive_integer"},
                {"name": "--total-timeout-ms", "required": false, "shape": "positive_integer"},
                {"name": "--envelope-limit-bytes", "required": false, "shape": "positive_integer"}
            ])
        );
    }
}

#[test]
fn discovery_exposes_canonical_string_constraints() {
    let company = json_output(&["operations", "describe", "company"], 0);
    let corp_code = company["operation"]["flags"]
        .as_array()
        .expect("company flags")
        .iter()
        .find(|flag| flag["name"] == "--corp-code")
        .expect("corp-code flag");
    assert_eq!(
        corp_code["constraints"],
        serde_json::json!({
            "format": "opendart-corp-code",
            "min_length": 8,
            "max_length": 8
        })
    );

    let list = json_output(&["operations", "describe", "list"], 0);
    let flags = list["operation"]["flags"].as_array().expect("list flags");
    let constraint = |name: &str| {
        &flags
            .iter()
            .find(|flag| flag["name"] == name)
            .unwrap_or_else(|| panic!("missing {name}"))["constraints"]
    };
    assert_eq!(
        constraint("--last-reprt-at")["allowed_values"],
        serde_json::json!(["Y", "N"])
    );
    assert_eq!(constraint("--bgn-de")["format"], "opendart-date");
    assert_eq!(constraint("--page-no")["decimal_minimum"], 1);
    assert_eq!(constraint("--page-count")["decimal_maximum"], 100);
}

#[test]
fn request_constraint_errors_are_actionable_and_do_not_echo_values() {
    let format = json_output(
        &[
            "call",
            "company",
            "--corp-code",
            "１２３４５６７８",
            "--representation",
            "json",
        ],
        2,
    );
    assert_eq!(format["error"]["reason"], "invalid_format");
    assert_eq!(format["error"]["argument"], "--corp-code");
    assert_eq!(format["error"]["format"], "opendart-corp-code");
    assert_eq!(format["operation"]["name"], "company");
    assert!(!format.to_string().contains("１２３４５６７８"));

    let allowed = json_output(
        &[
            "call",
            "list",
            "--last-reprt-at",
            "private-value",
            "--representation",
            "json",
        ],
        2,
    );
    assert_eq!(allowed["error"]["reason"], "invalid_allowed_value");
    assert_eq!(allowed["error"]["argument"], "--last-reprt-at");
    assert_eq!(allowed["error"]["allowed"], serde_json::json!(["Y", "N"]));
    assert!(!allowed.to_string().contains("private-value"));

    let range = json_output(
        &[
            "call",
            "list",
            "--page-count",
            "101",
            "--representation",
            "json",
        ],
        2,
    );
    assert_eq!(range["error"]["reason"], "invalid_decimal_range");
    assert_eq!(range["error"]["argument"], "--page-count");
    assert_eq!(range["error"]["minimum"], 1);
    assert_eq!(range["error"]["maximum"], 100);
    assert!(!range.to_string().contains("101"));
}

#[test]
fn operation_inventory_filters_are_keyless_deterministic_and_composable() {
    let company_by_name = json_output(&["operations", "list", "--query", "COMPANY"], 0);
    let company_by_name = company_by_name["operations"]
        .as_array()
        .expect("filtered operations");
    assert_eq!(company_by_name.len(), 1);
    assert_eq!(company_by_name[0]["name"], "company");

    for query in ["ds001-2019002", "2019002", "개황정보"] {
        let filtered = json_output(&["operations", "list", "--query", query], 0);
        let operations = filtered["operations"]
            .as_array()
            .expect("filtered operations");
        assert_eq!(
            operations.len(),
            1,
            "query did not narrow to company: {query}"
        );
        assert_eq!(operations[0]["name"], "company");
    }

    let group = json_output(&["operations", "list", "--group", "ds001"], 0);
    let grouped = group["operations"].as_array().expect("grouped operations");
    assert!(!grouped.is_empty());
    assert!(
        grouped
            .iter()
            .all(|operation| operation["group"] == "DS001")
    );

    let zip = json_output(&["operations", "list", "--representation", "zip"], 0);
    assert_eq!(
        zip,
        json_output(&["operations", "list", "--representation", "zip"], 0)
    );
    let zip_operations = zip["operations"].as_array().expect("ZIP operations");
    assert!(!zip_operations.is_empty());
    assert!(zip_operations.iter().all(|operation| {
        operation["representations"]
            .as_array()
            .expect("representations")
            .iter()
            .any(|representation| representation == "zip")
    }));

    let combined = json_output(
        &[
            "operations",
            "list",
            "--query",
            "code",
            "--group",
            "DS001",
            "--representation",
            "zip",
        ],
        0,
    );
    let combined = combined["operations"]
        .as_array()
        .expect("combined operations");
    assert_eq!(combined.len(), 1);
    assert_eq!(combined[0]["name"], "corp-code");
    assert_eq!(combined[0]["representations"], serde_json::json!(["zip"]));
    assert!(
        combined[0]["description"]
            .as_str()
            .is_some_and(|description| !description.is_empty())
    );

    let empty = json_output(
        &[
            "operations",
            "list",
            "--query",
            "no-such-opendart-operation",
        ],
        0,
    );
    assert_eq!(
        empty,
        serde_json::json!({"kind": "operations", "operations": []})
    );
}

#[test]
fn discovery_json_alone_constructs_an_sdk_prepared_call() {
    let detail = json_output(&["operations", "describe", "company"], 0);
    let operation = &detail["operation"];
    let mut arguments: Vec<String> = operation["invocation"]["argv_prefix"]
        .as_array()
        .expect("argv prefix")
        .iter()
        .map(|value| value.as_str().expect("argument").to_owned())
        .collect();
    for flag in operation["flags"].as_array().expect("flags") {
        if flag["required"] == true {
            arguments.push(flag["name"].as_str().expect("flag name").to_owned());
            arguments.push("00126380".to_owned());
        }
    }
    let representation = &operation["representations"][0];
    arguments.extend(
        representation["selector_argv"]
            .as_array()
            .expect("selector argv")
            .iter()
            .map(|value| value.as_str().expect("argument").to_owned()),
    );

    let output = invoke(&arguments, None);
    assert_eq!(output.status.code(), Some(1));
    let error: Value = serde_json::from_slice(&output.stdout).expect("error JSON");
    assert_eq!(error["error"]["code"], "missing_api_key");
    assert_eq!(error["operation"]["name"], "company");
    assert_eq!(error["operation"]["logical_id"], "DS001-2019002");
    assert_eq!(error["operation"]["physical_id"], "get_company_json");
    assert_eq!(error["operation"]["representation"], "json");

    arguments[1] = operation["logical_id"]
        .as_str()
        .expect("logical ID")
        .to_owned();
    let alias_output = invoke(&arguments, None);
    assert_eq!(alias_output.status.code(), Some(1));
    assert_eq!(alias_output.stdout, output.stdout);
}

#[test]
fn empty_api_key_remains_missing() {
    let arguments = vec![
        "call".to_owned(),
        "company".to_owned(),
        "--corp-code".to_owned(),
        "00126380".to_owned(),
        "--representation".to_owned(),
        "json".to_owned(),
    ];
    let output = invoke(&arguments, Some(""));
    assert_eq!(output.status.code(), Some(1));
    let error: Value = serde_json::from_slice(&output.stdout).expect("error JSON");
    assert_eq!(error["error"]["code"], "missing_api_key");
}

#[cfg(unix)]
#[test]
fn non_text_api_key_is_invalid_client_configuration() {
    use std::ffi::OsString;
    use std::os::unix::ffi::OsStringExt;

    let output = Command::new(env!("CARGO_BIN_EXE_opendart"))
        .args([
            "call",
            "company",
            "--corp-code",
            "00126380",
            "--representation",
            "json",
        ])
        .env("OPENDART_API_KEY", OsString::from_vec(vec![0xff]))
        .output()
        .expect("CLI process should start");
    assert_eq!(output.status.code(), Some(1));
    assert!(output.stderr.is_empty());
    let error: Value = serde_json::from_slice(&output.stdout).expect("error JSON");
    assert_eq!(error["error"]["code"], "invalid_client_configuration");
    assert_eq!(error["error"]["reason"], "non_text_api_key");
    assert_eq!(error["operation"]["name"], "company");
}

#[test]
fn whitespace_and_control_api_keys_are_rejected_without_disclosure() {
    let arguments = vec![
        "call".to_owned(),
        "company".to_owned(),
        "--corp-code".to_owned(),
        "00126380".to_owned(),
        "--representation".to_owned(),
        "json".to_owned(),
    ];
    for (key, reason) in [
        (" \t\u{2003}", "whitespace_only_api_key"),
        ("sentinel\nsecret", "control_character_api_key"),
    ] {
        let output = invoke(&arguments, Some(key));
        assert_eq!(output.status.code(), Some(1));
        assert!(output.stderr.is_empty());
        let text = String::from_utf8(output.stdout).expect("error JSON should be UTF-8");
        assert!(!text.contains(key));
        assert!(!text.contains("sentinel"));
        assert!(!text.contains("secret"));
        let error: Value = serde_json::from_str(&text).expect("error JSON");
        assert_eq!(error["error"]["code"], "invalid_client_configuration");
        assert_eq!(error["error"]["reason"], reason);
        assert_eq!(error["operation"]["name"], "company");
    }
}

#[test]
fn invalid_invocations_are_strict_json_usage_errors_before_credentials() {
    let cases: &[&[&str]] = &[
        &["unknown"],
        &["operations", "describe", "unknown"],
        &[
            "call",
            "company",
            "--unknown",
            "value",
            "--corp-code",
            "00126380",
            "--representation",
            "json",
        ],
        &[
            "call",
            "company",
            "--corp-code",
            "00126380",
            "--corp-code",
            "00126381",
            "--representation",
            "json",
        ],
        &[
            "call",
            "company",
            "--corp-code",
            "00126380",
            "spillover",
            "--representation",
            "json",
        ],
        &["call", "company", "--corp-code", "00126380"],
        &[
            "call",
            "company",
            "--corp-code",
            "00126380",
            "--representation",
            "zip",
        ],
        &["call", "corp-code"],
        &["operations", "list", "--representation", "yaml"],
    ];
    for arguments in cases {
        let owned: Vec<String> = arguments.iter().map(|value| (*value).to_owned()).collect();
        let output = invoke(&owned, Some("must-not-be-read"));
        assert_eq!(output.status.code(), Some(2), "arguments: {arguments:?}");
        assert!(output.stderr.is_empty());
        let error: Value = serde_json::from_slice(&output.stdout).expect("usage error JSON");
        assert_eq!(error["kind"], "error");
        assert_eq!(error["error"]["code"], "invalid_invocation");
        assert!(!String::from_utf8_lossy(&output.stdout).contains("must-not-be-read"));
    }

    let unknown_flag = json_output(
        &[
            "call",
            "company",
            "--unknown",
            "value",
            "--corp-code",
            "00126380",
            "--representation",
            "json",
        ],
        2,
    );
    let correction = unknown_flag["error"]["help"][0]
        .as_str()
        .expect("safe correction hint");
    assert!(correction.contains("--corp-code"));
    assert!(correction.contains("--representation"));
    assert!(!correction.contains("--unknown"));
    assert_eq!(unknown_flag["error"]["reason"], "unknown_argument");
    assert!(unknown_flag["error"].get("argument").is_none());
    assert!(!unknown_flag.to_string().contains("--unknown"));

    let missing = json_output(&["call", "company", "--corp-code", "00126380"], 2);
    assert_eq!(missing["error"]["reason"], "missing_required_argument");
    assert_eq!(missing["error"]["argument"], "--representation");

    let invalid = json_output(
        &[
            "call",
            "company",
            "--corp-code",
            "00126380",
            "--representation",
            "private-value",
        ],
        2,
    );
    assert_eq!(invalid["error"]["reason"], "invalid_value");
    assert_eq!(invalid["error"]["argument"], "--representation");
    assert_eq!(
        invalid["error"]["allowed"],
        serde_json::json!(["json", "xml"])
    );
    assert!(!invalid.to_string().contains("private-value"));

    let conflict = json_output(
        &[
            "call",
            "company",
            "--corp-code",
            "00126380",
            "--corp-code",
            "00126381",
            "--representation",
            "json",
        ],
        2,
    );
    assert_eq!(conflict["error"]["reason"], "argument_conflict");
    assert_eq!(conflict["error"]["argument"], "--corp-code");

    let invalid_filter = json_output(
        &["operations", "list", "--representation", "private-value"],
        2,
    );
    assert_eq!(invalid_filter["error"]["reason"], "invalid_value");
    assert_eq!(invalid_filter["error"]["argument"], "--representation");
    assert_eq!(
        invalid_filter["error"]["allowed"],
        serde_json::json!(["json", "xml", "zip"])
    );
    assert!(
        invalid_filter["error"]["help"][0]
            .as_str()
            .is_some_and(|help| help.contains("--query") && help.contains("--representation"))
    );
    assert!(!invalid_filter.to_string().contains("private-value"));
}

#[test]
fn stable_error_outputs_match_repository_fixtures() {
    let invalid = invoke(&["unknown".to_owned()], None);
    assert_eq!(invalid.status.code(), Some(2));
    assert!(invalid.stderr.is_empty());
    assert_eq!(invalid.stdout, INVALID_INVOCATION_FIXTURE);

    let missing = invoke(
        &[
            "call".to_owned(),
            "company".to_owned(),
            "--corp-code".to_owned(),
            "00126380".to_owned(),
            "--representation".to_owned(),
            "json".to_owned(),
        ],
        None,
    );
    assert_eq!(missing.status.code(), Some(1));
    assert!(missing.stderr.is_empty());
    assert_eq!(missing.stdout, MISSING_API_KEY_FIXTURE);
}

#[test]
fn zip_output_and_sdk_input_rules_precede_credentials() {
    let zip = json_output(&["operations", "describe", "corp-code"], 0);
    assert_eq!(zip["operation"]["representations"][0]["name"], "zip");
    assert_eq!(
        zip["operation"]["representations"][0]["output"]["kind"],
        "artifact"
    );
    let destination =
        std::env::temp_dir().join(format!("opendart-discovery-{}.zip", std::process::id()));
    let destination = destination.to_string_lossy().into_owned();
    let arguments = vec![
        "call".to_owned(),
        "corp-code".to_owned(),
        "--output".to_owned(),
        destination.clone(),
    ];
    let output = invoke(&arguments, None);
    assert_eq!(output.status.code(), Some(1));
    let error: Value = serde_json::from_slice(&output.stdout).expect("credential error JSON");
    assert_eq!(error["error"]["code"], "missing_api_key");
    assert!(!Path::new(&destination).exists());

    let empty_required = json_output(
        &[
            "call",
            "company",
            "--corp-code",
            "",
            "--representation",
            "json",
        ],
        2,
    );
    assert_eq!(empty_required["error"]["code"], "invalid_request");
    assert_eq!(empty_required["error"]["reason"], "missing_input");
    assert_eq!(empty_required["error"]["argument"], "--corp-code");
    assert_eq!(empty_required["operation"]["name"], "company");
}

#[test]
fn help_and_version_are_the_only_plain_text_successes() {
    for arguments in [
        ["--help"].as_slice(),
        ["operations", "--help"].as_slice(),
        ["operations", "list", "--help"].as_slice(),
        ["call", "--help"].as_slice(),
        ["call", "company", "--help"].as_slice(),
        ["--version"].as_slice(),
        ["operations", "--version"].as_slice(),
        ["operations", "list", "--version"].as_slice(),
        ["call", "--version"].as_slice(),
        ["call", "company", "--version"].as_slice(),
    ] {
        let owned: Vec<String> = arguments.iter().map(|value| (*value).to_owned()).collect();
        let output = invoke(&owned, None);
        assert_eq!(output.status.code(), Some(0));
        assert!(!output.stdout.is_empty());
        assert!(serde_json::from_slice::<Value>(&output.stdout).is_err());
    }
}

#[test]
fn call_help_is_concise_and_operation_help_explains_shared_controls() {
    let call = invoke(&["call".to_owned(), "--help".to_owned()], None);
    assert_eq!(call.status.code(), Some(0));
    assert!(call.stderr.is_empty());
    let call = String::from_utf8(call.stdout).expect("call help should be UTF-8");
    assert!(call.contains("opendart operations list"));
    assert!(call.contains("opendart operations describe <operation>"));
    assert!(call.contains("Usage: opendart call <OPERATION> [OPTIONS]"));
    assert!(!call.contains("accnut-adtor-nm-nd-adt-opinion"));
    assert!(!call.contains("company\n"));

    let company = invoke(
        &["call".to_owned(), "company".to_owned(), "--help".to_owned()],
        None,
    );
    assert_eq!(company.status.code(), Some(0));
    assert!(company.stderr.is_empty());
    let company = String::from_utf8(company.stdout).expect("operation help should be UTF-8");
    for expected in [
        "--representation <REPRESENTATION>",
        "Select the structured response representation",
        "--connect-timeout-ms <MILLISECONDS>",
        "Set the connection timeout in milliseconds",
        "--read-timeout-ms <MILLISECONDS>",
        "Set the response-read timeout in milliseconds",
        "--total-timeout-ms <MILLISECONDS>",
        "Set the total request timeout in milliseconds",
        "--envelope-limit-bytes <BYTES>",
        "Set the maximum buffered structured-response size in bytes",
    ] {
        assert!(
            company.contains(expected),
            "operation help omitted {expected:?}"
        );
    }

    let artifact = invoke(
        &[
            "call".to_owned(),
            "corp-code".to_owned(),
            "--help".to_owned(),
        ],
        None,
    );
    assert_eq!(artifact.status.code(), Some(0));
    assert!(artifact.stderr.is_empty());
    let artifact = String::from_utf8(artifact.stdout).expect("artifact help should be UTF-8");
    for expected in [
        "--output <PATH>",
        "Write the binary response to a new file without overwriting",
        "--artifact-limit-bytes <BYTES>",
        "Set the maximum binary artifact size in bytes",
    ] {
        assert!(
            artifact.contains(expected),
            "artifact help omitted {expected:?}"
        );
    }
}

#[test]
fn every_command_depth_reports_the_cli_package_identity() {
    let expected = format!("opendart {}\n", env!("CARGO_PKG_VERSION"));
    for arguments in [
        ["--version"].as_slice(),
        ["operations", "--version"].as_slice(),
        ["operations", "list", "--version"].as_slice(),
        ["operations", "describe", "--version"].as_slice(),
        ["call", "--version"].as_slice(),
        ["call", "company", "--version"].as_slice(),
    ] {
        let owned: Vec<String> = arguments.iter().map(|value| (*value).to_owned()).collect();
        let output = invoke(&owned, None);
        assert_eq!(output.status.code(), Some(0), "arguments: {arguments:?}");
        assert!(output.stderr.is_empty());
        assert_eq!(
            String::from_utf8(output.stdout).expect("version should be UTF-8"),
            expected,
            "arguments: {arguments:?}"
        );
    }
}
