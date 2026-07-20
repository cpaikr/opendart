//! Repository-only process tests for typed structured execution over loopback.

#![cfg(opendart_compat)]

mod common;

use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};
use std::process::{Command, Output};
use std::thread;
use std::time::Duration;

use serde_json::Value;

const SYNTHETIC_KEY: &str = "work4-synthetic-sentinel";
const JSON_SUCCESS_FIXTURE: &[u8] = include_bytes!("fixtures/company-json-success.json");
const XML_SUCCESS_FIXTURE: &[u8] = include_bytes!("fixtures/company-xml-success.json");

fn company_arguments(representation: &str) -> Vec<String> {
    vec![
        "call".to_owned(),
        "company".to_owned(),
        "--corp-code".to_owned(),
        "00126380".to_owned(),
        "--representation".to_owned(),
        representation.to_owned(),
    ]
}

fn invoke(origin: &str, arguments: &[String]) -> Output {
    let output = Command::new(env!("CARGO_BIN_EXE_opendart"))
        .args(arguments)
        .env("OPENDART_API_KEY", SYNTHETIC_KEY)
        .env("OPENDART_COMPAT_ORIGIN", origin)
        .output()
        .expect("CLI process should start");
    assert!(
        output.stderr.is_empty(),
        "structured calls must not use stderr"
    );
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(!stdout.contains(SYNTHETIC_KEY));
    output
}

fn with_response(response: Vec<u8>, arguments: &[String]) -> Output {
    let listener = TcpListener::bind("127.0.0.1:0").expect("loopback listener");
    let origin = format!("http://{}", listener.local_addr().unwrap());
    let server = thread::spawn(move || {
        let mut stream = common::accept_within(&listener);
        assert_valid_request(&mut stream);
        stream.write_all(&response).expect("write fixture response");
    });
    let output = invoke(&origin, arguments);
    server.join().expect("fixture server should finish");
    output
}

fn http_response(content_type: &str, body: &[u8], extra_headers: &[(&str, &str)]) -> Vec<u8> {
    let mut response = format!(
        "HTTP/1.1 200 OK\r\nContent-Type: {content_type}\r\nContent-Length: {}\r\n",
        body.len()
    );
    for (name, value) in extra_headers {
        response.push_str(name);
        response.push_str(": ");
        response.push_str(value);
        response.push_str("\r\n");
    }
    response.push_str("Connection: close\r\n\r\n");
    let mut bytes = response.into_bytes();
    bytes.extend_from_slice(body);
    bytes
}

fn assert_valid_request(stream: &mut TcpStream) {
    stream
        .set_read_timeout(Some(Duration::from_secs(2)))
        .unwrap();
    let mut request = Vec::new();
    let mut chunk = [0_u8; 1024];
    while !request.windows(4).any(|window| window == b"\r\n\r\n") {
        let count = stream.read(&mut chunk).expect("read request headers");
        assert_ne!(count, 0, "request headers ended early");
        request.extend_from_slice(&chunk[..count]);
        assert!(
            request.len() < 32 * 1024,
            "request headers are unexpectedly large"
        );
    }
    assert!(request.starts_with(b"GET /api/company."));
    assert_eq!(count_bytes(&request, b"crtfc_key="), 1);
}

fn count_bytes(haystack: &[u8], needle: &[u8]) -> usize {
    haystack
        .windows(needle.len())
        .filter(|window| *window == needle)
        .count()
}

fn json(output: &Output, expected_exit: i32) -> Value {
    assert_eq!(output.status.code(), Some(expected_exit));
    serde_json::from_slice(&output.stdout).expect("stdout should be one JSON document")
}

#[test]
fn json_success_preserves_typed_additive_values_and_exact_numbers() {
    let body = r#"{"status":"000","corp_name":"한글\ncontrol","future":{"mixed":[null,true,"text",-0,1.2300,1E+999999999999999999999999]}}"#.as_bytes();
    let response = http_response(
        "application/json",
        body,
        &[("Content-Language", SYNTHETIC_KEY)],
    );
    let output = with_response(response, &company_arguments("json"));
    assert_eq!(output.stdout, JSON_SUCCESS_FIXTURE);
    let value = json(&output, 0);

    assert_eq!(value["kind"], "response");
    assert_eq!(value["operation"]["name"], "company");
    assert_eq!(value["operation"]["logical_id"], "DS001-2019002");
    assert_eq!(value["operation"]["physical_id"], "get_company_json");
    assert_eq!(value["operation"]["representation"], "json");
    assert_eq!(value["response"]["reply"]["kind"], "success");
    assert_eq!(
        value["response"]["reply"]["value"]["corp_name"],
        "한글\ncontrol"
    );
    let text = String::from_utf8(output.stdout).unwrap();
    assert!(text.contains("-0,1.2300,1E+999999999999999999999999"));
    let headers = value["response"]["metadata"]["headers"]
        .as_array()
        .expect("headers");
    assert!(
        headers
            .iter()
            .any(|header| header["name"] == "content-type")
    );
    assert!(
        !headers
            .iter()
            .any(|header| header["name"] == "content-language")
    );
}

#[test]
fn xml_success_and_json_xml_statuses_keep_complete_source_evidence() {
    let xml = r#"<result><status>000</status><corp_name>한글 &amp; XML</corp_name><future><item>A</item><item>B</item></future></result>"#.as_bytes();
    let output = with_response(
        http_response("application/xml", xml, &[]),
        &company_arguments("xml"),
    );
    assert_eq!(output.stdout, XML_SUCCESS_FIXTURE);
    let value = json(&output, 0);
    assert_eq!(value["operation"]["physical_id"], "get_company_xml");
    assert_eq!(value["operation"]["representation"], "xml");
    assert_eq!(
        value["response"]["reply"]["value"]["corp_name"],
        "한글 & XML"
    );

    for (representation, content_type, body, request_id) in [
        (
            "json",
            "application/json",
            br#"{"status":"013","message":"none","request_id":"json-123"}"#.as_slice(),
            "json-123",
        ),
        (
            "xml",
            "application/xml",
            br#"<result><status>future</status><message>none</message><request_id>xml-123</request_id></result>"#.as_slice(),
            "xml-123",
        ),
    ] {
        let output = with_response(
            http_response(content_type, body, &[]),
            &company_arguments(representation),
        );
        let value = json(&output, 1);
        assert_eq!(value["kind"], "response");
        assert_eq!(value["response"]["reply"]["kind"], "status");
        assert_eq!(
            value["response"]["reply"]["value"]["evidence"]["request_id"],
            request_id
        );
    }
}

#[test]
fn envelope_limits_malformed_bodies_and_decode_failures_are_sanitized() {
    let body = br#"{"status":"000","corp_name":"bounded"}"#;
    let mut exact = company_arguments("json");
    exact.extend(["--envelope-limit-bytes".to_owned(), body.len().to_string()]);
    let output = with_response(http_response("application/json", body, &[]), &exact);
    assert_eq!(json(&output, 0)["response"]["reply"]["kind"], "success");

    let mut too_small = company_arguments("json");
    too_small.extend([
        "--envelope-limit-bytes".to_owned(),
        (body.len() - 1).to_string(),
    ]);
    let output = with_response(http_response("application/json", body, &[]), &too_small);
    let error = json(&output, 1);
    assert_eq!(error["error"]["code"], "body_limit");
    assert_eq!(error["operation"]["name"], "company");
    assert_eq!(error["metadata"]["status"], 200);

    let output = with_response(
        http_response("application/json", b"{\"unterminated\":", &[]),
        &company_arguments("json"),
    );
    assert_eq!(json(&output, 1)["error"]["code"], "malformed_envelope");
}

#[test]
fn unexpected_xml_root_is_a_typed_response_decode_failure() {
    let body = br#"<wrong><status>000</status><value>x</value></wrong>"#;
    let response = http_response("application/xml", body, &[]);
    assert!(
        response
            .windows(15)
            .any(|value| value == b"application/xml")
    );
    let output = with_response(response, &company_arguments("xml"));
    let error = json(&output, 1);
    assert_eq!(error["error"]["code"], "response_decode", "{error}");
}

#[test]
fn timeout_incomplete_body_and_refused_connection_have_stable_classes() {
    let listener = TcpListener::bind("127.0.0.1:0").unwrap();
    let origin = format!("http://{}", listener.local_addr().unwrap());
    let server = thread::spawn(move || {
        let mut stream = common::accept_within(&listener);
        assert_valid_request(&mut stream);
        stream
            .write_all(
                b"HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 2\r\n\r\n",
            )
            .unwrap();
        thread::sleep(Duration::from_millis(150));
    });
    let mut arguments = company_arguments("json");
    arguments.extend([
        "--read-timeout-ms".to_owned(),
        "20".to_owned(),
        "--total-timeout-ms".to_owned(),
        "100".to_owned(),
    ]);
    let output = invoke(&origin, &arguments);
    server.join().unwrap();
    let error = json(&output, 1);
    assert_eq!(error["error"]["code"], "transport_timeout");
    assert_eq!(error["metadata"]["status"], 200);

    let body = br#"{}"#;
    let mut incomplete = http_response("application/json", body, &[]);
    let needle = format!("Content-Length: {}", body.len());
    let replacement = format!("Content-Length: {}", body.len() + 5);
    let text = String::from_utf8(incomplete)
        .unwrap()
        .replace(&needle, &replacement);
    incomplete = text.into_bytes();
    let output = with_response(incomplete, &company_arguments("json"));
    let error = json(&output, 1);
    assert_eq!(error["error"]["code"], "transport_body");
    assert_eq!(error["metadata"]["status"], 200);

    let listener = TcpListener::bind("127.0.0.1:0").unwrap();
    let origin = format!("http://{}", listener.local_addr().unwrap());
    drop(listener);
    let output = invoke(&origin, &company_arguments("json"));
    let error = json(&output, 1);
    assert_eq!(error["error"]["code"], "transport_connection");
    assert!(error.get("metadata").is_none());
}

#[cfg(target_os = "linux")]
#[test]
fn stdout_failure_after_structured_encoding_exits_without_a_replacement_document() {
    use std::fs::OpenOptions;
    use std::process::Stdio;

    let listener = TcpListener::bind("127.0.0.1:0").unwrap();
    let origin = format!("http://{}", listener.local_addr().unwrap());
    let server = thread::spawn(move || {
        let mut stream = common::accept_within(&listener);
        assert_valid_request(&mut stream);
        stream
            .write_all(&http_response(
                "application/json",
                br#"{"status":"000","corp_name":"complete"}"#,
                &[],
            ))
            .unwrap();
    });
    let full = OpenOptions::new().write(true).open("/dev/full").unwrap();
    let output = Command::new(env!("CARGO_BIN_EXE_opendart"))
        .args(company_arguments("json"))
        .env("OPENDART_API_KEY", SYNTHETIC_KEY)
        .env("OPENDART_COMPAT_ORIGIN", origin)
        .stdout(Stdio::from(full))
        .stderr(Stdio::piped())
        .output()
        .unwrap();
    server.join().unwrap();
    assert_eq!(output.status.code(), Some(1));
    assert!(output.stdout.is_empty());
    assert!(output.stderr.is_empty());
}
