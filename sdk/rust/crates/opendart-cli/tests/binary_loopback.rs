//! Repository-only process tests for bounded binary artifact execution.

#![cfg(opendart_compat)]

use std::fs;
use std::io::{self, Read, Write};
use std::net::{Shutdown, TcpListener, TcpStream};
use std::path::Path;
use std::process::{Command, Output};
use std::thread;
use std::time::{Duration, Instant};

use serde_json::Value;

const SYNTHETIC_KEY: &str = "work5-synthetic-sentinel";

fn binary_arguments(destination: &Path) -> Vec<String> {
    vec![
        "call".to_owned(),
        "corp-code".to_owned(),
        "--output".to_owned(),
        destination.to_string_lossy().into_owned(),
    ]
}

fn command(origin: &str, arguments: &[String]) -> Command {
    let mut command = Command::new(env!("CARGO_BIN_EXE_opendart"));
    command
        .args(arguments)
        .env("OPENDART_API_KEY", SYNTHETIC_KEY)
        .env("OPENDART_COMPAT_ORIGIN", origin);
    command
}

fn invoke(origin: &str, arguments: &[String]) -> Output {
    let output = command(origin, arguments)
        .output()
        .expect("CLI process should start");
    assert_clean_channels(&output);
    output
}

fn invoke_keyless(arguments: &[String]) -> Output {
    let output = Command::new(env!("CARGO_BIN_EXE_opendart"))
        .args(arguments)
        .env_remove("OPENDART_API_KEY")
        .env_remove("OPENDART_COMPAT_ORIGIN")
        .output()
        .expect("CLI process should start");
    assert_clean_channels(&output);
    output
}

fn assert_clean_channels(output: &Output) {
    assert!(output.stderr.is_empty(), "binary calls must not use stderr");
    assert!(
        !output
            .stdout
            .windows(SYNTHETIC_KEY.len())
            .any(|window| { window == SYNTHETIC_KEY.as_bytes() })
    );
    assert!(
        !output
            .stderr
            .windows(SYNTHETIC_KEY.len())
            .any(|window| { window == SYNTHETIC_KEY.as_bytes() })
    );
}

fn with_server(
    arguments: &[String],
    responder: impl FnOnce(&mut TcpStream) + Send + 'static,
) -> Output {
    let listener = TcpListener::bind("127.0.0.1:0").expect("loopback listener");
    listener
        .set_nonblocking(true)
        .expect("configure loopback listener");
    let origin = format!("http://{}", listener.local_addr().unwrap());
    let server = thread::spawn(move || {
        let deadline = Instant::now() + Duration::from_secs(5);
        let (mut stream, _) = loop {
            match listener.accept() {
                Ok(connection) => break connection,
                Err(error) if error.kind() == io::ErrorKind::WouldBlock => {
                    assert!(Instant::now() < deadline, "CLI never connected");
                    thread::sleep(Duration::from_millis(5));
                }
                Err(error) => panic!("accept CLI connection: {error}"),
            }
        };
        stream
            .set_nonblocking(false)
            .expect("restore blocking fixture stream");
        assert_valid_request(&mut stream);
        responder(&mut stream);
    });
    let output = invoke(&origin, arguments);
    server.join().expect("fixture server should finish");
    output
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
        assert!(request.len() < 32 * 1024, "request headers too large");
    }
    assert!(request.starts_with(b"GET /api/corpCode.xml?"));
    assert_eq!(count_bytes(&request, b"crtfc_key="), 1);
}

fn count_bytes(haystack: &[u8], needle: &[u8]) -> usize {
    haystack
        .windows(needle.len())
        .filter(|window| *window == needle)
        .count()
}

fn write_fixed(stream: &mut TcpStream, content_type: &str, body: &[u8]) {
    write_fixed_with_headers(stream, content_type, body, &[]);
}

fn write_fixed_with_headers(
    stream: &mut TcpStream,
    content_type: &str,
    body: &[u8],
    extra_headers: &[(&str, &str)],
) {
    write!(
        stream,
        "HTTP/1.1 200 OK\r\nContent-Type: {content_type}\r\nContent-Length: {}\r\n",
        body.len()
    )
    .unwrap();
    for (name, value) in extra_headers {
        write!(stream, "{name}: {value}\r\n").unwrap();
    }
    write!(stream, "Connection: close\r\n\r\n").unwrap();
    stream.write_all(body).unwrap();
}

fn write_chunked(stream: &mut TcpStream, content_type: &str, chunks: &[&[u8]]) {
    write!(
        stream,
        "HTTP/1.1 200 OK\r\nContent-Type: {content_type}\r\nTransfer-Encoding: chunked\r\nConnection: close\r\n\r\n"
    )
    .unwrap();
    for chunk in chunks {
        write!(stream, "{:x}\r\n", chunk.len()).unwrap();
        stream.write_all(chunk).unwrap();
        stream.write_all(b"\r\n").unwrap();
        stream.flush().unwrap();
        thread::sleep(Duration::from_millis(5));
    }
    stream.write_all(b"0\r\n\r\n").unwrap();
}

fn json(output: &Output, expected_exit: i32) -> Value {
    assert_eq!(output.status.code(), Some(expected_exit));
    serde_json::from_slice(&output.stdout).expect("stdout should be one JSON document")
}

fn assert_artifact_reply(
    output: &Output,
    expected_exit: i32,
    expected_kind: &str,
    destination: &Path,
    body: &[u8],
) {
    let value = json(output, expected_exit);
    assert_eq!(value["kind"], "response");
    assert_eq!(value["operation"]["name"], "corp-code");
    assert_eq!(value["operation"]["logical_id"], "DS001-2019018");
    assert_eq!(value["operation"]["physical_id"], "get_corpCode_xml");
    assert_eq!(value["operation"]["representation"], "zip");
    assert_eq!(value["response"]["reply"]["kind"], expected_kind);
    assert_eq!(
        value["response"]["reply"]["value"]["path"],
        destination.to_string_lossy().as_ref()
    );
    assert_eq!(
        value["response"]["reply"]["value"]["bytes"],
        u64::try_from(body.len()).unwrap()
    );
    assert_eq!(fs::read(destination).unwrap(), body);
}

fn assert_directory_empty(directory: &Path) {
    assert_eq!(fs::read_dir(directory).unwrap().count(), 0);
}

#[test]
fn split_archives_and_unrecognized_bodies_publish_exact_bytes() {
    let directory = tempfile::tempdir().unwrap();
    let archive = directory.path().join("결과 파일.zip");
    let archive_body = b"PK\x03\x04ABCD\x00\xfftail";
    let output = with_server(&binary_arguments(&archive), |stream| {
        write_chunked(
            stream,
            "application/xml",
            &[b"P", b"K\x03", b"\x04A", b"BCD\x00", b"\xfftail"],
        );
    });
    assert_artifact_reply(&output, 0, "archive", &archive, archive_body);

    let empty = directory.path().join("empty.zip");
    let mut empty_body = b"PK\x05\x06".to_vec();
    empty_body.extend([0_u8; 18]);
    let empty_fixture = empty_body.clone();
    let output = with_server(&binary_arguments(&empty), move |stream| {
        write_chunked(
            stream,
            "application/octet-stream",
            &[
                &empty_fixture[..2],
                &empty_fixture[2..4],
                &empty_fixture[4..],
            ],
        );
    });
    assert_artifact_reply(&output, 0, "archive", &empty, &empty_body);

    let unknown = directory.path().join("unknown.bin");
    let unknown_body = b"PK\x07\x08split-descriptor\x00\xff";
    let output = with_server(&binary_arguments(&unknown), |stream| {
        write_fixed(stream, "application/zip", unknown_body);
    });
    assert_artifact_reply(&output, 1, "unrecognized", &unknown, unknown_body);
}

#[test]
fn alternate_status_preserves_evidence_without_publishing() {
    let directory = tempfile::tempdir().unwrap();
    let destination = directory.path().join("status.zip");
    let body = b" \n<result><status>future</status><message>none</message><request_id>binary-123</request_id></result>";
    let output = with_server(&binary_arguments(&destination), |stream| {
        write_fixed(stream, "application/zip", body);
    });
    let value = json(&output, 1);
    assert_eq!(value["response"]["reply"]["kind"], "status");
    assert_eq!(value["response"]["reply"]["value"]["code"], "future");
    assert_eq!(
        value["response"]["reply"]["value"]["evidence"]["request_id"],
        "binary-123"
    );
    assert_eq!(value["response"]["metadata"]["status"], 200);
    assert_directory_empty(directory.path());
}

#[test]
fn limits_and_incomplete_streams_never_publish_partial_artifacts() {
    let body = b"PK\x03\x04ABCD";

    let exact_dir = tempfile::tempdir().unwrap();
    let exact = exact_dir.path().join("exact.zip");
    let mut arguments = binary_arguments(&exact);
    arguments.extend(["--artifact-limit-bytes".to_owned(), "8".to_owned()]);
    let output = with_server(&arguments, |stream| {
        write_chunked(
            stream,
            "application/zip",
            &[b"P", b"K\x03", b"\x04A", b"BCD"],
        );
    });
    assert_artifact_reply(&output, 0, "archive", &exact, body);

    let overflow_dir = tempfile::tempdir().unwrap();
    let overflow = overflow_dir.path().join("overflow.zip");
    let mut arguments = binary_arguments(&overflow);
    arguments.extend(["--artifact-limit-bytes".to_owned(), "7".to_owned()]);
    let output = with_server(&arguments, |stream| {
        write_chunked(
            stream,
            "application/zip",
            &[b"P", b"K\x03", b"\x04A", b"BCD"],
        );
    });
    let error = json(&output, 1);
    assert_eq!(error["error"]["code"], "artifact_limit");
    assert_eq!(error["metadata"]["status"], 200);
    assert_directory_empty(overflow_dir.path());

    let incomplete_dir = tempfile::tempdir().unwrap();
    let incomplete = incomplete_dir.path().join("incomplete.zip");
    let partial = b"PK\x03\x04partial";
    let output = with_server(&binary_arguments(&incomplete), |stream| {
        write!(
            stream,
            "HTTP/1.1 200 OK\r\nContent-Type: application/zip\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
            partial.len() + 9
        )
        .unwrap();
        stream.write_all(partial).unwrap();
        stream.flush().unwrap();
        stream.shutdown(Shutdown::Write).unwrap();
    });
    let error = json(&output, 1);
    assert_eq!(error["error"]["code"], "transport_body");
    assert_eq!(error["metadata"]["status"], 200);
    assert_directory_empty(incomplete_dir.path());
}

#[test]
fn invalid_and_existing_destinations_fail_before_credentials_or_network() {
    for spelling in ["", "-"] {
        let arguments = vec![
            "call".to_owned(),
            "corp-code".to_owned(),
            "--output".to_owned(),
            spelling.to_owned(),
        ];
        let error = json(&invoke_keyless(&arguments), 2);
        assert_eq!(error["error"]["code"], "invalid_invocation");
    }

    let directory = tempfile::tempdir().unwrap();
    let existing_file = directory.path().join("existing.zip");
    fs::write(&existing_file, b"keep").unwrap();
    let error = json(&invoke_keyless(&binary_arguments(&existing_file)), 1);
    assert_eq!(error["error"]["code"], "destination_exists");
    assert_eq!(fs::read(&existing_file).unwrap(), b"keep");

    let existing_directory = directory.path().join("existing-dir");
    fs::create_dir(&existing_directory).unwrap();
    let error = json(&invoke_keyless(&binary_arguments(&existing_directory)), 1);
    assert_eq!(error["error"]["code"], "destination_exists");

    let invalid_parent = existing_file.join("child.zip");
    let error = json(&invoke_keyless(&binary_arguments(&invalid_parent)), 1);
    assert_eq!(error["error"]["code"], "artifact_io");

    let mut zero_limit = binary_arguments(&directory.path().join("zero.zip"));
    zero_limit.extend(["--artifact-limit-bytes".to_owned(), "0".to_owned()]);
    let error = json(&invoke_keyless(&zero_limit), 2);
    assert_eq!(error["error"]["code"], "invalid_invocation");
}

#[cfg(unix)]
#[test]
fn dangling_symlink_is_an_existing_destination() {
    use std::os::unix::fs::symlink;

    let directory = tempfile::tempdir().unwrap();
    let destination = directory.path().join("dangling.zip");
    symlink("missing-target", &destination).unwrap();
    let error = json(&invoke_keyless(&binary_arguments(&destination)), 1);
    assert_eq!(error["error"]["code"], "destination_exists");
    assert!(
        fs::symlink_metadata(&destination)
            .unwrap()
            .file_type()
            .is_symlink()
    );
}

#[test]
fn publication_race_preserves_the_rival_destination_and_cleans_the_tempfile() {
    let directory = tempfile::tempdir().unwrap();
    let destination = directory.path().join("raced.zip");
    let parent = directory.path().to_owned();
    let raced = destination.clone();
    let body = b"PK\x03\x04complete";
    let output = with_server(&binary_arguments(&destination), move |stream| {
        let entries: Vec<_> = fs::read_dir(&parent).unwrap().collect();
        assert_eq!(entries.len(), 1, "one same-directory tempfile is staged");
        assert!(!raced.exists());
        fs::write(&raced, b"rival").unwrap();
        write_fixed_with_headers(
            stream,
            "application/zip",
            body,
            &[("Content-Language", SYNTHETIC_KEY)],
        );
    });
    let error = json(&output, 1);
    assert_eq!(error["error"]["code"], "destination_exists");
    assert_eq!(error["metadata"]["status"], 200);
    assert_eq!(fs::read(&destination).unwrap(), b"rival");
    assert_eq!(fs::read_dir(directory.path()).unwrap().count(), 1);
}

#[cfg(target_os = "linux")]
#[test]
fn stdout_failure_after_publication_leaves_the_complete_artifact() {
    use std::fs::OpenOptions;
    use std::process::Stdio;

    let directory = tempfile::tempdir().unwrap();
    let destination = directory.path().join("published.zip");
    let body = b"PK\x03\x04complete";
    let listener = TcpListener::bind("127.0.0.1:0").unwrap();
    let origin = format!("http://{}", listener.local_addr().unwrap());
    let server = thread::spawn(move || {
        let (mut stream, _) = listener.accept().unwrap();
        assert_valid_request(&mut stream);
        write_fixed(&mut stream, "application/zip", body);
    });
    let full = OpenOptions::new().write(true).open("/dev/full").unwrap();
    let output = command(&origin, &binary_arguments(&destination))
        .stdout(Stdio::from(full))
        .stderr(Stdio::piped())
        .output()
        .unwrap();
    server.join().unwrap();
    assert_eq!(output.status.code(), Some(1));
    assert_clean_channels(&output);
    assert_eq!(fs::read(&destination).unwrap(), body);
    assert_eq!(fs::read_dir(directory.path()).unwrap().count(), 1);
}
