//! Black-box checks for the transport-independent public contract.

use std::fmt::Display;

use opendart::{
    ApiKey, Authentication, AuthorizedRequest, OperationIdentity, PreparedRequest, Representation,
    RequestMethod,
    operations::{AccnutAdtorNmNdAdtOpinion, Company, CorpCode, FnlttMultiAcnt, List},
};
use static_assertions::assert_not_impl_any;

assert_not_impl_any!(ApiKey: Clone, Display);
assert_not_impl_any!(AuthorizedRequest<'static>: Clone, Display);
assert_not_impl_any!(PreparedRequest: Clone);

#[test]
fn representative_json_request_is_deterministic_and_credential_free() {
    let prepared = AccnutAdtorNmNdAdtOpinion::new("00126380", "2025", "11011")
        .prepare(Representation::Json)
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
    let json = operation
        .prepare(Representation::Json)
        .expect("JSON should be supported");
    let xml = operation
        .prepare(Representation::Xml)
        .expect("XML should be supported");

    assert_ne!(json.identity().physical(), xml.identity().physical());
    assert_eq!(json.identity().logical(), xml.identity().logical());
    assert_eq!(json.expected_representations(), &[Representation::Json]);
    assert_eq!(xml.expected_representations(), &[Representation::Xml]);
}

#[test]
fn fixed_binary_operation_routes_zip_and_xml_source_error() {
    let prepared = CorpCode::new()
        .prepare(Representation::Zip)
        .expect("ZIP should be supported");
    assert_eq!(prepared.relative_path(), "/api/corpCode.xml");
    assert_eq!(
        prepared.expected_representations(),
        &[Representation::Zip, Representation::Xml]
    );
    assert!(CorpCode::new().prepare(Representation::Xml).is_err());
}

#[test]
fn multi_company_request_enforces_cardinality_and_comma_serialization() {
    let prepared = FnlttMultiAcnt::new(["00334624", "00126380"], "2025", "11011")
        .prepare(Representation::Json)
        .expect("documented multi-company input should prepare");
    assert_eq!(
        prepared.encoded_query(),
        "corp_code=00334624,00126380&bsns_year=2025&reprt_code=11011"
    );

    assert!(
        FnlttMultiAcnt::new(Vec::<String>::new(), "2025", "11011")
            .prepare(Representation::Json)
            .is_err()
    );
    assert!(
        FnlttMultiAcnt::new(vec!["00126380"; 101], "2025", "11011")
            .prepare(Representation::Json)
            .is_err()
    );
    assert!(
        FnlttMultiAcnt::new(vec!["00126380"; 100], "2025", "11011")
            .prepare(Representation::Json)
            .is_ok()
    );

    let escaped = FnlttMultiAcnt::new(["a,b", "회사 /+"], "2025", "11011")
        .prepare(Representation::Json)
        .expect("reserved list data should remain distinct from delimiters");
    assert_eq!(
        escaped.encoded_query(),
        "corp_code=a%2Cb,%ED%9A%8C%EC%82%AC+%2F%2B&bsns_year=2025&reprt_code=11011"
    );
}

#[test]
fn authorization_is_explicit_and_redacted() {
    let sentinel = "secret /+ credential";
    let encoded = "secret+%2F%2B+credential";
    let key = ApiKey::new(sentinel).expect("non-empty key should be accepted");
    let prepared = Company::new("00126380")
        .prepare(Representation::Json)
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
fn empty_inputs_fail_without_echoing_values() {
    let error = Company::new("")
        .prepare(Representation::Json)
        .expect_err("empty required input must fail");
    assert!(error.to_string().contains("corp_code"));
    assert!(ApiKey::new("").is_err());

    let error = List::new()
        .with_page_no("")
        .prepare(Representation::Json)
        .expect_err("a supplied optional query value must not be empty");
    assert!(error.to_string().contains("page_no"));
}

#[test]
fn operation_identity_debug_contains_only_stable_identifiers() {
    fn assert_identity(identity: OperationIdentity) {
        let diagnostic = format!("{identity:?}");
        assert!(diagnostic.contains(identity.physical()));
        assert!(diagnostic.contains(identity.logical()));
    }

    let prepared = Company::new("00126380")
        .prepare(Representation::Json)
        .expect("request should prepare");
    assert_identity(prepared.identity());
}
