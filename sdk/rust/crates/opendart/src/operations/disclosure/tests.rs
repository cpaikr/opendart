use std::collections::BTreeMap;

use super::*;
use crate::SourceValueKind;

fn object(fields: impl IntoIterator<Item = (&'static str, SourceValue)>) -> SourceValue {
    SourceValue::object(
        fields
            .into_iter()
            .map(|(name, value)| (name.to_owned(), value))
            .collect::<BTreeMap<_, _>>(),
    )
}

#[test]
fn preparation_uses_reviewed_names_and_canonical_query_order() {
    let request = DisclosureSearchInput::new()
        .with_company_code("00126380".to_owned())
        .with_page_size("100".to_owned())
        .prepare_json()
        .unwrap();

    assert_eq!(request.identity().physical(), "get_list_json");
    assert_eq!(request.identity().logical(), "DS001-2019001");
    assert_eq!(request.relative_path(), "/api/list.json");
    assert_eq!(request.encoded_query(), "corp_code=00126380&page_count=100");
}

#[test]
fn decoder_rejects_nested_wrong_kinds_with_a_stable_path() {
    let request = DisclosureSearchInput::new().prepare_json().unwrap();
    let error = request
        .decode(object([("list", SourceValue::string("not-an-array"))]))
        .unwrap_err();

    assert_eq!(
        error,
        ResponseDecodeError::WrongKind {
            path: "$/list".to_owned(),
            expected: SourceValueKind::Array,
            actual: SourceValueKind::String,
        }
    );
}

#[test]
fn xml_singletons_are_exposed_through_the_reviewed_item_view() {
    let request = DisclosureSearchInput::new().prepare_xml().unwrap();
    let response = request
        .decode(object([(
            "list",
            object([("corp_name", SourceValue::string("OpenDART Corp"))]),
        )]))
        .unwrap();
    let items = response.items().collect::<Vec<_>>();

    assert_eq!(items.len(), 1);
    assert_eq!(
        items[0].company_name().and_then(SourceValue::as_str),
        Some("OpenDART Corp")
    );
    assert_eq!(
        response.source().get("list").unwrap().kind(),
        SourceValueKind::Object
    );
}

#[test]
fn binary_preparation_keeps_the_separate_zip_lifecycle() {
    let request = CompanyCodesInput::new().prepare_archive().unwrap();

    assert_eq!(request.identity().physical(), "get_corpCode_xml");
    assert_eq!(request.relative_path(), "/api/corpCode.xml");
    assert_eq!(request.encoded_query(), "");
    assert_eq!(
        request.expected_representations(),
        &[crate::Representation::Zip, crate::Representation::Xml]
    );
}
