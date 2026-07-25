//! Black-box checks for the transport-independent public contract.

use std::fmt::Display;

use opendart::{
    ApiKey, Authentication, AuthorizedRequest, OperationIdentity, PreparedBinaryRequest,
    PreparedRequest, Representation, RequestMethod, ResponseMetadata, SourceReply, SourceValue,
    SourceValueKind, WireInspector,
    operations::{AccnutAdtorNmNdAdtOpinion, Company, CorpCode, FnlttMultiAcnt, List},
    responses::CompanyJsonResponse,
    source_provenance,
};
use static_assertions::assert_not_impl_any;

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

#[test]
fn multi_company_request_enforces_cardinality_and_comma_serialization() {
    let prepared = FnlttMultiAcnt::new(["00334624", "00126380"], "2025", "11011")
        .prepare_json()
        .expect("documented multi-company input should prepare");
    assert_eq!(
        prepared.encoded_query(),
        "corp_code=00334624,00126380&bsns_year=2025&reprt_code=11011"
    );

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
        FnlttMultiAcnt::new(vec!["00126380"; 101], "2025", "11011")
            .prepare_json()
            .is_err()
    );
    assert!(
        FnlttMultiAcnt::new(vec!["00126380"; 100], "2025", "11011")
            .prepare_json()
            .is_ok()
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
