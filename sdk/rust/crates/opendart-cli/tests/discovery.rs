//! Process-level contract tests for keyless generated CLI discovery.

use std::collections::HashSet;
use std::path::Path;
use std::process::{Command, Output};

use serde::Deserialize;
use serde_json::Value;

const MISSING_API_KEY_FIXTURE: &[u8] = include_bytes!("fixtures/missing-api-key.json");
const INVALID_INVOCATION_FIXTURE: &[u8] = include_bytes!("fixtures/invalid-invocation.json");
const DISPATCH_CASES: &[u8] = include_bytes!("../src/generated/dispatch_cases.json");

#[derive(Deserialize)]
struct DispatchCase {
    name: String,
    logical_id: String,
    physical_id: String,
    representation: String,
    argv: Vec<String>,
}

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
fn every_generated_dispatch_reaches_its_exact_sdk_identity_by_name_and_alias() {
    let cases: Vec<DispatchCase> =
        serde_json::from_slice(DISPATCH_CASES).expect("generated dispatch cases");
    let output_root = tempfile::tempdir().expect("temporary output directory");
    let mut identities = HashSet::new();
    let mut saw_zip = false;
    let mut saw_multiple_representations = false;
    let mut previous_name = None;

    for case in cases {
        assert_eq!(case.argv.first().map(String::as_str), Some("call"));
        assert_eq!(case.argv.get(1), Some(&case.name));
        assert!(
            identities.insert((case.name.clone(), case.representation.clone())),
            "duplicate dispatch case for {}/{}",
            case.name,
            case.representation
        );
        if previous_name.as_deref() == Some(case.name.as_str()) {
            saw_multiple_representations = true;
        }
        previous_name = Some(case.name.clone());

        let destination = output_root
            .path()
            .join(format!("{}-{}.zip", case.name, case.representation));
        let destination = destination.to_string_lossy().into_owned();
        let mut arguments = case.argv;
        for argument in &mut arguments {
            if argument == "<generated-test-output>" {
                *argument = destination.clone();
                saw_zip = true;
            }
        }

        let output = invoke(&arguments, None);
        assert_eq!(
            output.status.code(),
            Some(1),
            "canonical invocation failed for {}/{}: {}",
            case.name,
            case.representation,
            String::from_utf8_lossy(&output.stdout)
        );
        assert!(output.stderr.is_empty());
        let error: Value = serde_json::from_slice(&output.stdout).expect("credential error JSON");
        assert_eq!(error["error"]["code"], "missing_api_key");
        assert_eq!(error["operation"]["name"], case.name);
        assert_eq!(error["operation"]["logical_id"], case.logical_id);
        assert_eq!(error["operation"]["physical_id"], case.physical_id);
        assert_eq!(error["operation"]["representation"], case.representation);
        assert!(!Path::new(&destination).exists());

        if case.logical_id != case.name {
            arguments[1] = case.logical_id;
            let alias_output = invoke(&arguments, None);
            assert_eq!(alias_output.status.code(), Some(1));
            assert!(alias_output.stderr.is_empty());
            assert_eq!(alias_output.stdout, output.stdout);
        }
    }

    assert!(saw_zip, "generated matrix omitted ZIP dispatch");
    assert!(
        saw_multiple_representations,
        "generated matrix omitted a multi-representation operation"
    );
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

    let nested_command = json_output(&["operations", "typo"], 2);
    assert_eq!(nested_command["error"]["reason"], "unknown_command");
    assert_eq!(
        nested_command["error"]["allowed"],
        serde_json::json!(["list", "describe"])
    );
    assert!(
        nested_command["error"]["help"][0]
            .as_str()
            .is_some_and(|help| help.contains("list") && help.contains("describe"))
    );
    assert!(!nested_command.to_string().contains("typo"));

    let repeated_ancestor = json_output(&["operations", "operations"], 2);
    assert_eq!(repeated_ancestor["error"]["reason"], "unknown_command");
    assert_eq!(
        repeated_ancestor["error"]["allowed"],
        serde_json::json!(["list", "describe"])
    );

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
fn nested_invocation_errors_use_the_deepest_valid_safe_context() {
    let bare_call = json_output(&["call"], 2);
    assert_eq!(bare_call["error"]["reason"], "missing_subcommand");
    let operations = bare_call["error"]["allowed"]
        .as_array()
        .expect("generated operation choices");
    assert!(operations.iter().any(|name| name == "company"));
    assert!(!bare_call.to_string().contains("Valid commands: operations"));

    let unknown_call = json_output(&["call", "private-operation-value"], 2);
    assert_eq!(unknown_call["error"]["reason"], "unknown_command");
    assert_eq!(
        unknown_call["error"]["allowed"],
        bare_call["error"]["allowed"]
    );
    assert!(
        unknown_call["error"]["help"][0]
            .as_str()
            .is_some_and(|help| help.contains("opendart call <OPERATION>"))
    );
    assert!(!unknown_call.to_string().contains("private-operation-value"));

    let bare_operations = json_output(&["operations"], 2);
    assert_eq!(bare_operations["error"]["reason"], "missing_subcommand");
    assert_eq!(
        bare_operations["error"]["allowed"],
        serde_json::json!(["list", "describe"])
    );

    let bare_describe = json_output(&["operations", "describe"], 2);
    assert_eq!(
        bare_describe["error"]["reason"],
        "missing_required_argument"
    );
    assert!(
        bare_describe["error"]["help"][0]
            .as_str()
            .is_some_and(|help| help.contains("operations describe <OPERATION>"))
    );
    assert!(
        !bare_describe
            .to_string()
            .contains("Valid commands: operations")
    );

    let unknown_description =
        json_output(&["operations", "describe", "private-description-value"], 2);
    assert!(
        unknown_description["error"]["help"][0]
            .as_str()
            .is_some_and(|help| help.contains("operations list"))
    );
    assert!(
        !unknown_description
            .to_string()
            .contains("private-description-value")
    );

    let missing_operation_flag = json_output(&["call", "company"], 2);
    assert_eq!(
        missing_operation_flag["error"]["reason"],
        "missing_required_argument"
    );
    assert_eq!(
        missing_operation_flag["error"]["argument"],
        "--representation"
    );
    let help = missing_operation_flag["error"]["help"][0]
        .as_str()
        .expect("operation help");
    assert!(help.contains("--corp-code") && help.contains("--representation"));
    assert!(!help.contains("Valid commands: operations"));
}

#[test]
fn hyphen_leading_query_and_output_values_do_not_swallow_real_options() {
    for arguments in [
        ["operations", "list", "--query", "-company"].as_slice(),
        ["operations", "list", "--query=-company"].as_slice(),
        ["operations", "list", "--query", "--private-query-value"].as_slice(),
    ] {
        let value = json_output(arguments, 0);
        assert_eq!(value["kind"], "operations");
    }

    let output_root = tempfile::tempdir().expect("temporary output directory");
    let hyphen_output = output_root.path().join("-artifact.zip");
    for arguments in [
        ["call", "corp-code", "--output", "-artifact.zip"].as_slice(),
        ["call", "corp-code", "--output=-artifact.zip"].as_slice(),
    ] {
        let output = Command::new(env!("CARGO_BIN_EXE_opendart"))
            .args(arguments)
            .current_dir(output_root.path())
            .env_remove("OPENDART_API_KEY")
            .output()
            .expect("CLI process should start");
        assert_eq!(output.status.code(), Some(1));
        assert!(output.stderr.is_empty());
        let value: Value = serde_json::from_slice(&output.stdout).expect("credential error JSON");
        assert_eq!(value["error"]["code"], "missing_api_key");
        assert_eq!(value["operation"]["representation"], "zip");
        assert!(!hyphen_output.exists());
    }

    for following in [
        "--query",
        "--group",
        "--representation",
        "--help",
        "--version",
        "-h",
        "-V",
        "--",
    ] {
        let value = json_output(&["operations", "list", "--query", following], 2);
        assert_eq!(value["error"]["argument"], "--query", "{following}");
    }

    for following in [
        "--output",
        "--artifact-limit-bytes",
        "--connect-timeout-ms",
        "--read-timeout-ms",
        "--total-timeout-ms",
        "--envelope-limit-bytes",
        "--help",
        "--version",
        "-h",
        "-V",
        "--",
    ] {
        let value = json_output(&["call", "corp-code", "--output", following], 2);
        assert_eq!(value["error"]["argument"], "--output", "{following}");
    }

    for arguments in [
        ["call", "corp-code", "--output", "-"].as_slice(),
        ["call", "corp-code", "--output=-"].as_slice(),
    ] {
        let value = json_output(arguments, 2);
        assert_eq!(value["error"]["reason"], "invalid_output_path");
        assert_eq!(value["error"]["argument"], "--output");
    }

    for arguments in [
        ["operations", "list", "--", "--query", "-private"].as_slice(),
        ["call", "corp-code", "--", "--output", "-private.zip"].as_slice(),
    ] {
        let value = json_output(arguments, 2);
        assert_eq!(value["error"]["code"], "invalid_invocation");
        assert_ne!(value["error"]["code"], "missing_api_key");
    }
}

#[cfg(unix)]
#[test]
fn non_utf8_command_and_hyphen_value_errors_remain_contextual_and_sanitized() {
    use std::ffi::OsString;
    use std::os::unix::ffi::OsStringExt;

    for (arguments, reason) in [
        (
            vec![OsString::from("call"), OsString::from_vec(vec![0xff])],
            "unknown_command",
        ),
        (
            vec![
                OsString::from("operations"),
                OsString::from("list"),
                OsString::from("--query"),
                OsString::from_vec(vec![b'-', 0xff]),
            ],
            "invalid_utf8",
        ),
    ] {
        let output = Command::new(env!("CARGO_BIN_EXE_opendart"))
            .args(arguments)
            .env_remove("OPENDART_API_KEY")
            .output()
            .expect("CLI process should start");
        assert_eq!(output.status.code(), Some(2));
        assert!(output.stderr.is_empty());
        assert!(!output.stdout.contains(&0xff));
        let error: Value = serde_json::from_slice(&output.stdout).expect("usage error JSON");
        assert_eq!(error["error"]["reason"], reason);
        assert!(!error.to_string().contains("Valid commands: operations"));
    }
}

#[cfg(target_os = "linux")]
#[test]
fn non_utf8_executable_path_emits_one_global_output_encode_error() {
    use std::ffi::OsString;
    use std::os::unix::ffi::OsStringExt;

    let root = tempfile::tempdir().expect("temporary executable directory");
    let executable = root
        .path()
        .join(OsString::from_vec(b"opendart-\xff".to_vec()));
    std::fs::copy(env!("CARGO_BIN_EXE_opendart"), &executable)
        .expect("copy executable to non-UTF-8 path");

    let output = Command::new(executable)
        .env_remove("OPENDART_API_KEY")
        .output()
        .expect("copied CLI process should start");
    assert_eq!(output.status.code(), Some(1));
    assert!(output.stderr.is_empty());
    assert_eq!(
        output.stdout,
        b"{\"kind\":\"error\",\"error\":{\"code\":\"output_encode\",\"message\":\"the structured result could not be encoded safely\"}}\n"
    );
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
