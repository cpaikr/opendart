//! Process-level contract tests for keyless generated CLI discovery.

use std::path::Path;
use std::process::{Command, Output};

use serde_json::Value;

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
    assert_eq!(
        home["global_flags"].as_array().expect("global flags").len(),
        5
    );

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
        assert_eq!(
            by_name["operation"]["invocation"]["argv_prefix"],
            serde_json::json!(["call", name])
        );
    }
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

    arguments[1] = operation["logical_id"]
        .as_str()
        .expect("logical ID")
        .to_owned();
    let alias_output = invoke(&arguments, None);
    assert_eq!(alias_output.status.code(), Some(1));
    assert_eq!(alias_output.stdout, output.stdout);
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
}

#[test]
fn help_and_version_are_the_only_plain_text_successes() {
    for arguments in [
        ["--help"].as_slice(),
        ["operations", "--help"].as_slice(),
        ["--version"].as_slice(),
        ["operations", "--version"].as_slice(),
    ] {
        let owned: Vec<String> = arguments.iter().map(|value| (*value).to_owned()).collect();
        let output = invoke(&owned, None);
        assert_eq!(output.status.code(), Some(0));
        assert!(!output.stdout.is_empty());
        assert!(serde_json::from_slice::<Value>(&output.stdout).is_err());
    }
}
