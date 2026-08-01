//! Black-box checks for the transport-independent public contract.

use std::{cell::Cell, fmt::Display};

#[cfg(opendart_compat)]
use std::path::Path;

use opendart::{
    ApiKey, Authentication, AuthorizedRequest, EnvelopeFormat, OperationIdentity, PrepareError,
    PreparedBinaryRequest, PreparedRequest, Representation, RequestMethod, ResponseInterpretError,
    ResponseMetadata, SourceReply, SourceValue, SourceValueKind, WireInspectError, WireInspector,
    operations::{
        AccnutAdtorNmNdAdtOpinion, Company, CorpCode, FnlttCmpnyIndx, FnlttMultiAcnt, List,
    },
    responses::CompanyJsonResponse,
    source_provenance,
};
use static_assertions::assert_not_impl_any;

#[cfg(opendart_compat)]
use serde_json::Value as JsonValue;

#[cfg(feature = "serde-json")]
use opendart::SourceResponse;
#[cfg(feature = "serde-json")]
use static_assertions::assert_impl_all;

assert_not_impl_any!(ApiKey: Clone, Display);
assert_not_impl_any!(AuthorizedRequest<'static>: Clone, Display);
assert_not_impl_any!(PreparedRequest<CompanyJsonResponse>: Clone);
assert_not_impl_any!(PreparedBinaryRequest: Clone);
assert_not_impl_any!(ApiKey: serde::Serialize);
assert_not_impl_any!(AuthorizedRequest<'static>: serde::Serialize);
assert_not_impl_any!(PreparedRequest<CompanyJsonResponse>: serde::Serialize);
assert_not_impl_any!(PreparedBinaryRequest: serde::Serialize);

#[cfg(feature = "serde-json")]
assert_impl_all!(CompanyJsonResponse: serde::Serialize);
#[cfg(feature = "serde-json")]
assert_impl_all!(SourceValue: serde::Serialize);
#[cfg(feature = "serde-json")]
assert_impl_all!(ResponseMetadata: serde::Serialize);
#[cfg(feature = "serde-json")]
assert_impl_all!(SourceResponse<SourceReply<CompanyJsonResponse>>: serde::Serialize);

#[cfg(not(feature = "serde-json"))]
assert_not_impl_any!(CompanyJsonResponse: serde::Serialize);
#[cfg(not(feature = "serde-json"))]
assert_not_impl_any!(SourceValue: serde::Serialize);
#[cfg(not(feature = "serde-json"))]
assert_not_impl_any!(ResponseMetadata: serde::Serialize);

#[cfg(all(
    feature = "serde-json",
    feature = "client-reqwest",
    not(target_family = "wasm")
))]
assert_not_impl_any!(opendart::Client: serde::Serialize);
#[cfg(all(
    feature = "serde-json",
    feature = "client-reqwest",
    not(target_family = "wasm")
))]
assert_not_impl_any!(opendart::ClientBuilder: serde::Serialize);
#[cfg(all(
    feature = "serde-json",
    feature = "client-reqwest",
    not(target_family = "wasm")
))]
assert_not_impl_any!(opendart::BodyStream: serde::Serialize);

#[test]
fn representative_json_request_is_deterministic_and_credential_free() {
    let prepared = AccnutAdtorNmNdAdtOpinion::new("00126380", "2025", "11011")
        .prepare_json()
        .expect("representative input should prepare");

    assert_eq!(prepared.method(), RequestMethod::Get);
    assert_eq!(
        prepared.identity().physical(),
        "get_accnutAdtorNmNdAdtOpinion_json"
    );
    assert_eq!(prepared.identity().logical(), "DS002-2020009");
    assert_eq!(
        prepared.encoded_query(),
        "corp_code=00126380&bsns_year=2025&reprt_code=11011"
    );
    assert_eq!(prepared.authentication(), Authentication::ApiKeyQuery);
    assert!(!format!("{prepared:?}").contains("00126380"));
}

#[test]
fn representation_selection_changes_only_the_physical_contract() {
    let operation = Company::new("00126380");
    let json = operation.prepare_json().expect("JSON should be supported");
    let xml = operation.prepare_xml().expect("XML should be supported");

    assert_ne!(json.identity().physical(), xml.identity().physical());
    assert_eq!(json.identity().logical(), xml.identity().logical());
    assert_eq!(json.expected_representations(), &[Representation::Json]);
    assert_eq!(xml.expected_representations(), &[Representation::Xml]);
}

#[test]
fn prepared_request_interprets_typed_responses_without_an_http_client() {
    let prepared = Company::new("00126380").prepare_json().unwrap();
    let inspector = WireInspector::new(1024).unwrap();
    let reply = prepared
        .interpret_response(
            &inspector,
            200,
            br#"{"status":"000","corp_name":"Example Corp"}"#,
        )
        .unwrap();

    let SourceReply::Success(company) = reply else {
        panic!("a success envelope should use the generated response decoder");
    };
    assert_eq!(
        company.corp_name.as_ref().and_then(SourceValue::as_str),
        Some("Example Corp")
    );
}

#[test]
fn prepared_request_makes_non_success_http_status_explicit() {
    let prepared = Company::new("00126380").prepare_json().unwrap();
    let inspector = WireInspector::new(1024).unwrap();
    let error = prepared
        .interpret_response(
            &inspector,
            500,
            br#"{"status":"000","corp_name":"contradictory success"}"#,
        )
        .unwrap_err();

    let ResponseInterpretError::HttpStatus {
        status,
        evidence: Some(SourceReply::Success(value)),
        ..
    } = error
    else {
        panic!("valid bounded evidence should survive the HTTP failure");
    };
    assert_eq!(status, 500);
    assert_eq!(
        value.get("corp_name").and_then(SourceValue::as_str),
        Some("contradictory success")
    );
}

#[cfg(opendart_compat)]
#[test]
fn repository_contract_corpus_crosses_the_public_interpreter() {
    let inspector = WireInspector::new(64 * 1024).unwrap();
    let json = Company::new("00126380").prepare_json().unwrap();
    let xml = Company::new("00126380").prepare_xml().unwrap();
    let binary = CorpCode::new().prepare_zip().unwrap();

    let manifest: JsonValue = serde_json::from_slice(&contract_fixture("manifest.json"))
        .expect("contract fixture manifest is valid JSON");
    for case in manifest["requestCases"]
        .as_array()
        .expect("requestCases is an array")
    {
        let physical = case["physicalOperation"]
            .as_str()
            .expect("physical operation is a string");
        let logical = case["logicalOperation"]
            .as_str()
            .expect("logical operation is a string");
        let path = case["path"].as_str().expect("request path is a string");
        let representation = case["representation"]
            .as_str()
            .expect("representation is a string");
        let (identity, method, actual_path, authentication, representations) = match physical {
            "get_company_json" => (
                json.identity(),
                json.method(),
                json.relative_path(),
                json.authentication(),
                json.expected_representations(),
            ),
            "get_corpCode_xml" => (
                binary.identity(),
                binary.method(),
                binary.relative_path(),
                binary.authentication(),
                binary.expected_representations(),
            ),
            other => panic!("unhandled request fixture operation {other}"),
        };
        assert_eq!(identity.physical(), physical);
        assert_eq!(identity.logical(), logical);
        assert_eq!(method, RequestMethod::Get);
        assert_eq!(case["method"].as_str(), Some("GET"));
        assert_eq!(actual_path, path);
        assert_eq!(authentication, Authentication::ApiKeyQuery);
        let expected = match representation {
            "json" => Representation::Json,
            "zip" => Representation::Zip,
            other => panic!("unhandled request fixture representation {other}"),
        };
        assert!(representations.contains(&expected));
    }
    for case in manifest["responseCases"]
        .as_array()
        .expect("responseCases is an array")
    {
        let physical = case["physicalOperation"]
            .as_str()
            .expect("physical operation is a string");
        let body = contract_fixture(case["file"].as_str().expect("fixture file is a string"));
        match physical {
            "get_company_json" => assert_structured_case(&json, &inspector, case, &body),
            "get_company_xml" => assert_structured_case(&xml, &inspector, case, &body),
            "get_corpCode_xml" => {
                assert_eq!(case["httpStatus"].as_u64(), Some(200));
                assert_eq!(case["outcome"].as_str(), Some("source-status"));
                assert!(matches!(
                    inspector.inspect_xml(&body),
                    Ok(SourceReply::Status(_))
                ));
            }
            other => panic!("unhandled fixture operation {other}"),
        }
    }
}

#[cfg(opendart_compat)]
fn assert_structured_case<T>(
    prepared: &PreparedRequest<T>,
    inspector: &WireInspector,
    case: &JsonValue,
    body: &[u8],
) {
    let id = case["id"].as_str().expect("fixture ID is a string");
    let status = u16::try_from(
        case["httpStatus"]
            .as_u64()
            .expect("HTTP status is an integer"),
    )
    .expect("HTTP status fits u16");
    let outcome = case["outcome"].as_str().expect("outcome is a string");
    let result = prepared.interpret_response(inspector, status, body);
    let matches = match (outcome, result) {
        ("typed-success", Ok(SourceReply::Success(_)))
        | ("source-status", Ok(SourceReply::Status(_)))
        | ("decode-failure", Err(ResponseInterpretError::Decode { .. }))
        | ("envelope-failure", Err(ResponseInterpretError::Envelope { .. }))
        | (
            "http-status-failure",
            Err(ResponseInterpretError::HttpStatus {
                evidence: Some(_), ..
            }),
        ) => true,
        _ => false,
    };
    assert!(matches, "fixture {id} did not produce {outcome}");
}

#[cfg(opendart_compat)]
fn contract_fixture(name: &str) -> Vec<u8> {
    std::fs::read(
        Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../../../openapi/fixtures/v1")
            .join(name),
    )
    .expect("repository contract fixture is readable")
}

#[test]
fn fixed_binary_operation_routes_zip_and_xml_source_error() {
    let prepared = CorpCode::new()
        .prepare_zip()
        .expect("ZIP should be supported");
    assert_eq!(prepared.relative_path(), "/api/corpCode.xml");
    assert_eq!(
        prepared.expected_representations(),
        &[Representation::Zip, Representation::Xml]
    );
}

fn assert_invalid_cardinality(error: PrepareError, physical: &'static str, logical: &'static str) {
    let PrepareError::InvalidCardinality {
        operation,
        parameter,
        minimum,
        maximum,
    } = error
    else {
        panic!("unexpected preparation error: {error:?}");
    };
    assert_eq!(operation.physical(), physical);
    assert_eq!(operation.logical(), logical);
    assert_eq!(parameter, "corp_code");
    assert_eq!(minimum, 1);
    assert_eq!(maximum, 100);
}

#[test]
fn bounded_array_consumes_each_available_valid_item_once() {
    let yielded = Cell::new(0);
    let values = ["00334624", "00126380"].into_iter().inspect(|_| {
        yielded.set(yielded.get() + 1);
    });
    let operation = FnlttMultiAcnt::new(values, "2025", "11011");

    assert_eq!(yielded.get(), 2);
    assert_eq!(operation.corp_code(), ["00334624", "00126380"]);
}

#[test]
fn bounded_array_preserves_exact_maximum_serialization() {
    let values = vec!["00126380"; 100];
    let prepared = FnlttMultiAcnt::new(values.clone(), "2025", "11011")
        .prepare_json()
        .expect("the documented maximum should prepare");
    assert_eq!(
        prepared.encoded_query(),
        format!(
            "corp_code={}&bsns_year=2025&reprt_code=11011",
            values.join(",")
        )
    );
}

#[test]
fn bounded_array_stops_at_overflow_sentinel_and_reports_every_representation() {
    let yielded = Cell::new(0);
    let operation = FnlttMultiAcnt::new(
        std::iter::repeat_with(|| {
            yielded.set(yielded.get() + 1);
            "00126380"
        }),
        "2025",
        "11011",
    );
    assert_eq!(yielded.get(), 101);
    assert_eq!(operation.corp_code().len(), 101);
    assert_invalid_cardinality(
        operation
            .prepare_json()
            .expect_err("the overflow sentinel must fail JSON preparation"),
        "get_fnlttMultiAcnt_json",
        "DS003-2019017",
    );
    assert_invalid_cardinality(
        operation
            .prepare_xml()
            .expect_err("the overflow sentinel must fail XML preparation"),
        "get_fnlttMultiAcnt_xml",
        "DS003-2019017",
    );
}

#[test]
fn every_generated_bounded_parameter_uses_the_same_overflow_contract() {
    let operation = FnlttCmpnyIndx::new(std::iter::repeat("00126380"), "2025", "11011", "M210000");
    assert_eq!(operation.corp_code().len(), 101);
    assert_invalid_cardinality(
        operation
            .prepare_json()
            .expect_err("the overflow sentinel must fail JSON preparation"),
        "get_fnlttCmpnyIndx_json",
        "DS003-2022002",
    );
    assert_invalid_cardinality(
        operation
            .prepare_xml()
            .expect_err("the overflow sentinel must fail XML preparation"),
        "get_fnlttCmpnyIndx_xml",
        "DS003-2022002",
    );
}

#[test]
fn multi_company_request_rejects_empty_and_invalid_elements() {
    assert!(
        FnlttMultiAcnt::new(Vec::<String>::new(), "2025", "11011")
            .prepare_json()
            .is_err()
    );
    assert!(
        FnlttMultiAcnt::new([""], "2025", "11011")
            .prepare_json()
            .is_err()
    );
    assert!(
        FnlttMultiAcnt::new(["a,b", "회사 /+"], "2025", "11011")
            .prepare_json()
            .is_err(),
        "company-code format validation must run before comma serialization"
    );
}

#[test]
fn authorization_is_explicit_and_redacted() {
    let sentinel = "secret /+ credential";
    let encoded = "secret+%2F%2B+credential";
    let key = ApiKey::new(sentinel).expect("non-empty key should be accepted");
    let prepared = Company::new("00126380")
        .prepare_json()
        .expect("request should prepare");
    let authorized = prepared.authorize(&key);

    let authorized_diagnostic = format!("{authorized:?}");
    authorized.with_exposed_relative_uri(|relative_uri| {
        assert_eq!(
            relative_uri,
            "/api/company.json?corp_code=00126380&crtfc_key=secret+%2F%2B+credential"
        );
    });
    for diagnostic in [
        format!("{key:?}"),
        format!("{prepared:?}"),
        authorized_diagnostic,
    ] {
        assert!(!diagnostic.contains(sentinel));
        assert!(!diagnostic.contains(encoded));
    }
}

#[test]
fn api_key_validation_rejects_unsafe_values_without_retaining_them() {
    for value in ["", " ", "\t\n", "\u{a0}"] {
        assert!(
            matches!(
                ApiKey::new(value),
                Err(opendart::AuthorizationError::EmptyApiKey)
            ),
            "empty and whitespace-only keys must be rejected"
        );
    }

    for value in [
        "key\0suffix",
        "key\nsuffix",
        "key\u{7f}suffix",
        "key\u{85}suffix",
    ] {
        let error = ApiKey::new(value).expect_err("control characters must be rejected");
        assert!(matches!(
            &error,
            opendart::AuthorizationError::ControlCharacterApiKey
        ));
        assert_eq!(
            error.to_string(),
            "the OpenDART API key must not contain control characters"
        );
        assert_eq!(format!("{error:?}"), "ControlCharacterApiKey");
    }
}

#[test]
fn api_key_validation_does_not_impose_length_or_character_set_rules() {
    assert!(ApiKey::new("x").is_ok());
    assert!(ApiKey::new("x".repeat(512)).is_ok());
    assert!(ApiKey::new(" key with spaces and 한글 ").is_ok());
}

#[test]
fn empty_inputs_fail_without_echoing_values() {
    let error = Company::new("")
        .prepare_json()
        .expect_err("empty required input must fail");
    assert!(error.to_string().contains("corp_code"));
    assert!(ApiKey::new("").is_err());

    let error = List::new()
        .with_page_no("")
        .prepare_json()
        .expect_err("a supplied optional query value must not be empty");
    assert!(error.to_string().contains("page_no"));
}

#[test]
fn canonical_input_constraints_fail_during_preparation_without_echoing_values() {
    let cases = [
        Company::new("１２３４５６７８")
            .prepare_json()
            .expect_err("company codes require ASCII digits"),
        List::new()
            .with_bgn_de("20230229")
            .prepare_json()
            .expect_err("compact dates require a valid calendar day"),
        List::new()
            .with_last_reprt_at("maybe")
            .prepare_json()
            .expect_err("closed values must be enforced"),
        List::new()
            .with_page_count("101")
            .prepare_json()
            .expect_err("page count must remain within its bound"),
        AccnutAdtorNmNdAdtOpinion::new("00126380", "２０２５", "11011")
            .prepare_json()
            .expect_err("business years require ASCII digits"),
        AccnutAdtorNmNdAdtOpinion::new("00126380", "2025", "99999")
            .prepare_json()
            .expect_err("report codes require documented values"),
    ];

    for error in cases {
        let diagnostic = error.to_string();
        for rejected in [
            "１２３４５６７８",
            "20230229",
            "maybe",
            "101",
            "２０２５",
            "99999",
        ] {
            assert!(!diagnostic.contains(rejected));
        }
    }
}

#[test]
fn operation_identity_debug_contains_only_stable_identifiers() {
    fn assert_identity(identity: OperationIdentity) {
        let diagnostic = format!("{identity:?}");
        assert!(diagnostic.contains(identity.physical()));
        assert!(diagnostic.contains(identity.logical()));
    }

    let prepared = Company::new("00126380")
        .prepare_json()
        .expect("request should prepare");
    assert_identity(prepared.identity());
}

#[test]
fn bounded_inspection_retains_unknown_json_and_xml_evidence() {
    let inspector = WireInspector::new(512).expect("the public inspector requires a bound");
    let SourceReply::Success(json) = inspector
        .inspect_json(br#"{"status":"000","future":1.20e3,"flag":true,"none":null}"#)
        .expect("valid JSON should be inspectable")
    else {
        panic!("payload fields must prevent status-only classification");
    };
    assert_eq!(
        json.get("future")
            .and_then(opendart::SourceValue::as_number_str),
        Some("1.20e3")
    );
    assert_eq!(
        json.get("none").map(opendart::SourceValue::kind),
        Some(SourceValueKind::Null)
    );

    let SourceReply::Success(xml) = inspector
        .inspect_xml(br#"<result future="yes"><item>A</item><item>B</item></result>"#)
        .expect("valid XML should be inspectable")
    else {
        panic!("unknown XML fields must remain success evidence");
    };
    assert_eq!(
        xml.get("@future").and_then(opendart::SourceValue::as_str),
        Some("yes")
    );
    assert_eq!(
        xml.get("item")
            .and_then(opendart::SourceValue::as_array)
            .map(<[_]>::len),
        Some(2)
    );
}

#[test]
fn malformed_xml_never_becomes_authoritative_public_status_evidence() {
    for (case, body) in [
        (
            "illegal element name",
            b"<1result><status>013</status></1result>".as_slice(),
        ),
        (
            "illegal attribute name",
            b"<result 1value=\"x\"><status>013</status></result>".as_slice(),
        ),
        (
            "literal less-than in attribute",
            b"<result value=\"<\"><status>013</status></result>".as_slice(),
        ),
        (
            "duplicate attribute",
            b"<result value=\"a\" value=\"b\"><status>013</status></result>".as_slice(),
        ),
        (
            "unbound namespace prefix",
            b"<result><x:status>013</x:status></result>".as_slice(),
        ),
        (
            "double hyphen in comment",
            b"<result><!--bad--comment--><status>013</status></result>".as_slice(),
        ),
        (
            "trailing hyphen in comment",
            b"<result><!--bad---><status>013</status></result>".as_slice(),
        ),
        (
            "CDATA terminator in text",
            b"<result><status>013]]></status></result>".as_slice(),
        ),
        (
            "declaration inside root",
            b"<result><?xml version=\"1.0\"?><status>013</status></result>".as_slice(),
        ),
        (
            "repeated declaration",
            b"<?xml version=\"1.0\"?><?xml version=\"1.0\"?><result><status>013</status></result>"
                .as_slice(),
        ),
        (
            "declaration after comment",
            b"<!--before--><?xml version=\"1.0\"?><result><status>013</status></result>"
                .as_slice(),
        ),
        (
            "declaration without version",
            b"<?xml encoding=\"UTF-8\"?><result><status>013</status></result>".as_slice(),
        ),
        (
            "declaration pseudo-attributes out of order",
            b"<?xml standalone=\"no\" version=\"1.0\"?><result><status>013</status></result>"
                .as_slice(),
        ),
        (
            "unsupported declaration version",
            b"<?xml version=\"1.1\"?><result><status>013</status></result>".as_slice(),
        ),
        (
            "unsupported declared encoding",
            b"<?xml version=\"1.0\" encoding=\"EUC-JP\"?><result><status>013</status></result>"
                .as_slice(),
        ),
        (
            "invalid standalone value",
            b"<?xml version=\"1.0\" standalone=\"maybe\"?><result><status>013</status></result>"
                .as_slice(),
        ),
        (
            "unknown declaration pseudo-attribute",
            b"<?xml version=\"1.0\" bogus=\"value\"?><result><status>013</status></result>"
                .as_slice(),
        ),
        (
            "tab-separated declaration inside root",
            b"<result><?xml\tversion=\"1.0\"?><status>013</status></result>".as_slice(),
        ),
        (
            "newline-separated repeated declaration",
            b"<?xml version=\"1.0\"?><?xml\nversion=\"1.0\"?><result><status>013</status></result>"
                .as_slice(),
        ),
        (
            "processing instruction without target",
            b"<? ?><result><status>013</status></result>".as_slice(),
        ),
        (
            "processing instruction with illegal target",
            b"<?1bad?><result><status>013</status></result>".as_slice(),
        ),
        (
            "processing instruction data without whitespace",
            b"<?pi?x?><result><status>013</status></result>".as_slice(),
        ),
        (
            "processing instruction target with slash",
            b"<?pi/data?><result><status>013</status></result>".as_slice(),
        ),
        (
            "reserved target followed by question mark",
            b"<?xml?x?><result><status>013</status></result>".as_slice(),
        ),
        (
            "reserved target followed by slash",
            b"<?xml/data?><result><status>013</status></result>".as_slice(),
        ),
        (
            "reserved processing instruction target",
            b"<?XmL note?><result><status>013</status></result>".as_slice(),
        ),
        (
            "empty DTD",
            b"<!DOCTYPE result><result><status>013</status></result>".as_slice(),
        ),
        (
            "internal DTD",
            b"<!DOCTYPE result [<!ENTITY code \"013\">]><result><status>&code;</status></result>"
                .as_slice(),
        ),
        (
            "external DTD",
            b"<!DOCTYPE result SYSTEM \"https://example.invalid/source.dtd\"><result><status>013</status></result>"
                .as_slice(),
        ),
        (
            "unknown entity",
            b"<result><status>&unknown;</status></result>".as_slice(),
        ),
        (
            "CDATA before root",
            b"<![CDATA[before]]><result><status>013</status></result>".as_slice(),
        ),
        (
            "non-whitespace after root",
            b"<result><status>013</status></result>after".as_slice(),
        ),
        (
            "forbidden literal character",
            b"<result><status>013\0</status></result>".as_slice(),
        ),
        (
            "forbidden character reference",
            b"<result><status>013&#0;</status></result>".as_slice(),
        ),
        (
            "mismatched root",
            b"<result><status>013</result></status>".as_slice(),
        ),
        (
            "unclosed root",
            b"<result><status>013</status>".as_slice(),
        ),
        (
            "multiple roots",
            b"<result><status>013</status></result><result/>".as_slice(),
        ),
    ] {
        let result = WireInspector::new(body.len())
            .expect("nonempty adversarial fixture")
            .inspect_xml(body);
        let Err(WireInspectError::Envelope(error)) = result else {
            panic!("{case} became authoritative: {result:?}");
        };
        assert_eq!(error.format(), EnvelopeFormat::Xml, "{case}");
    }
}

#[test]
fn valid_xml_document_misc_and_line_endings_remain_public_evidence() {
    let body = b"<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\
        <!--before--><?before ok?>\
        <result xmlns:x=\"urn:example\" x:future=\"yes\">\
        <status>000</status><x:item>A\r\nB\rC\nD&#13;E</x:item>\
        <data><![CDATA[tail\r\nvalue]]></data><?inside ok?>\
        </result><?after ok?><!--after-->";
    let SourceReply::Success(value) = WireInspector::new(body.len())
        .expect("nonempty valid fixture")
        .inspect_xml(body)
        .expect("supported XML 1.0 constructs must remain valid")
    else {
        panic!("payload-bearing XML must remain success evidence");
    };
    assert_eq!(
        value.get("@x:future").and_then(SourceValue::as_str),
        Some("yes")
    );
    assert_eq!(
        value.get("x:item").and_then(SourceValue::as_str),
        Some("A\nB\nC\nD\rE")
    );
    assert_eq!(
        value.get("data").and_then(SourceValue::as_str),
        Some("tail\nvalue")
    );
}

#[test]
fn additive_non_success_status_fields_remain_public_evidence() {
    let inspector = WireInspector::new(256).expect("the public inspector requires a bound");
    let SourceReply::Status(status) = inspector
        .inspect_json(br#"{"status":"013","message":"none","request_id":"abc"}"#)
        .expect("valid JSON should be inspectable")
    else {
        panic!("non-success status must not become a success payload");
    };

    assert_eq!(status.code.as_str(), "013");
    assert_eq!(
        status
            .evidence
            .get("request_id")
            .and_then(opendart::SourceValue::as_str),
        Some("abc")
    );
}

#[test]
fn source_provenance_identifies_the_reviewed_contract_snapshot() {
    let provenance = source_provenance();
    assert_eq!(provenance.crate_version(), env!("CARGO_PKG_VERSION"));
    assert_eq!(provenance.specification_source_release(), Some("v0.1.0"));
    assert_eq!(provenance.canonical_bundle_sha256().len(), 64);
    assert_eq!(provenance.sdk_projection_sha256().len(), 64);
    assert!(provenance.generator_schema() > 0);
}
