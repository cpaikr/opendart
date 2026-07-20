//! Explicitly authorized, read-only smoke coverage against OpenDART.

use std::ffi::{OsStr, OsString};
use std::fs;
use std::path::Path;
use std::process::{Command, Output};

use serde_json::Value;

fn gated_key(
    live_gate: Option<OsString>,
    read_key: impl FnOnce() -> Option<OsString>,
) -> Option<OsString> {
    if live_gate.as_deref() != Some(OsStr::new("1")) {
        return None;
    }
    read_key().filter(|key| !key.is_empty())
}

fn invoke(key: &OsStr, arguments: &[String]) -> Output {
    Command::new(env!("CARGO_BIN_EXE_opendart"))
        .args(arguments)
        .env("OPENDART_API_KEY", key)
        .env_remove("OPENDART_COMPAT_ORIGIN")
        .output()
        .expect("CLI process should start")
}

fn parse(output: &Output, key: &OsStr) -> Value {
    assert!(output.stderr.is_empty(), "live calls must not use stderr");
    assert_secret_absent(&output.stdout, key);
    assert_secret_absent(&output.stderr, key);
    serde_json::from_slice(&output.stdout).expect("live stdout should be one JSON document")
}

fn assert_secret_absent(bytes: &[u8], key: &OsStr) {
    let key = key.to_string_lossy();
    assert!(!key.is_empty());
    assert!(
        !bytes
            .windows(key.len())
            .any(|window| window == key.as_bytes()),
        "credential appeared in a captured channel"
    );
}

#[test]
fn disabled_live_gate_does_not_read_the_credential() {
    assert!(gated_key(None, || panic!("credential was read")).is_none());
    assert!(gated_key(Some(OsString::from("0")), || panic!("credential was read")).is_none());
    assert!(
        gated_key(Some(OsString::from("true")), || panic!(
            "credential was read"
        ))
        .is_none()
    );
    assert!(gated_key(Some(OsString::from("1")), || None).is_none());
}

#[test]
fn structured_and_binary_live_paths_are_read_only_and_sanitized() {
    let Some(key) = gated_key(std::env::var_os("OPENDART_LIVE_TESTS"), || {
        std::env::var_os("OPENDART_API_KEY")
    }) else {
        return;
    };

    let structured = invoke(
        &key,
        &[
            "call".to_owned(),
            "company".to_owned(),
            "--corp-code".to_owned(),
            "00126380".to_owned(),
            "--representation".to_owned(),
            "json".to_owned(),
        ],
    );
    assert_eq!(structured.status.code(), Some(0));
    let structured = parse(&structured, &key);
    assert_eq!(structured["kind"], "response");
    assert_eq!(structured["operation"]["name"], "company");
    assert_eq!(structured["operation"]["logical_id"], "DS001-2019002");
    assert_eq!(structured["operation"]["representation"], "json");
    assert_eq!(structured["response"]["reply"]["kind"], "success");

    let directory = tempfile::tempdir().expect("live artifact directory");
    let destination = directory.path().join("corp-code.zip");
    let binary = invoke(
        &key,
        &[
            "call".to_owned(),
            "corp-code".to_owned(),
            "--output".to_owned(),
            path_text(&destination),
        ],
    );
    assert_eq!(binary.status.code(), Some(0));
    let binary = parse(&binary, &key);
    assert_eq!(binary["kind"], "response");
    assert_eq!(binary["operation"]["name"], "corp-code");
    assert_eq!(binary["operation"]["logical_id"], "DS001-2019001");
    assert_eq!(binary["operation"]["representation"], "zip");
    assert_eq!(binary["response"]["reply"]["kind"], "archive");
    assert_eq!(binary["response"]["reply"]["path"], path_text(&destination));
    let artifact = fs::read(&destination).expect("live artifact should be published");
    assert!(artifact.starts_with(b"PK"));
    assert_eq!(
        binary["response"]["reply"]["bytes"].as_u64(),
        Some(u64::try_from(artifact.len()).unwrap())
    );
    assert_secret_absent(&artifact, &key);
}

fn path_text(path: &Path) -> String {
    path.to_str()
        .expect("test-owned temporary paths are Unicode")
        .to_owned()
}
