use serde::{Serialize, Serializer, ser::Error as _};
use serde_json::value::RawValue;

use super::{HttpVersion, SourceStatus, SourceValue, SourceValueRepr};

impl Serialize for HttpVersion {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_str(match self {
            Self::Http09 => "http/0.9",
            Self::Http10 => "http/1.0",
            Self::Http11 => "http/1.1",
            Self::Http2 => "http/2",
            Self::Http3 => "http/3",
            Self::Other(value) => value,
        })
    }
}

impl Serialize for SourceStatus {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_str(self.as_str())
    }
}

impl Serialize for SourceValue {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        match &self.0 {
            SourceValueRepr::Null => serializer.serialize_unit(),
            SourceValueRepr::Boolean(value) => serializer.serialize_bool(*value),
            SourceValueRepr::Number(value) => RawValue::from_string(value.clone())
                .map_err(S::Error::custom)?
                .serialize(serializer),
            SourceValueRepr::String(value) => serializer.serialize_str(value),
            SourceValueRepr::Array(value) => value.serialize(serializer),
            SourceValueRepr::Object(value) => value.serialize(serializer),
        }
    }
}

#[cfg(test)]
mod tests {
    use std::collections::BTreeMap;

    use super::*;
    use crate::{
        ResponseHeader, ResponseMetadata, SourceReply, SourceResponse, StatusEnvelope,
        generated::responses::ds001::{decode_company_json_response, decode_list_json_response},
    };

    #[test]
    fn exact_json_numbers_bypass_fixed_width_numeric_types() {
        for lexeme in [
            "-0",
            "1.2300",
            "1E+999999999999999999999999999999999999999999999999999999",
            "123456789012345678901234567890123456789012345678901234567890",
        ] {
            let value = SourceValue::number(lexeme).expect("fixture must satisfy SDK grammar");
            assert_eq!(serde_json::to_string(&value).unwrap(), lexeme);
        }
    }

    #[test]
    fn source_value_snapshot_covers_every_natural_json_kind() {
        let value = SourceValue::object(BTreeMap::from([
            (
                "array".to_owned(),
                SourceValue::array(vec![SourceValue::string("first"), SourceValue::null()]),
            ),
            ("boolean".to_owned(), SourceValue::boolean(true)),
            ("null".to_owned(), SourceValue::null()),
            (
                "number".to_owned(),
                SourceValue::number("-0.500E+20").expect("valid source number"),
            ),
            (
                "object".to_owned(),
                SourceValue::object(BTreeMap::from([(
                    "nested".to_owned(),
                    SourceValue::boolean(false),
                )])),
            ),
            ("string".to_owned(), SourceValue::string("한글\ncontrol")),
        ]));

        assert_eq!(
            serde_json::to_string(&value).unwrap(),
            r#"{"array":["first",null],"boolean":true,"null":null,"number":-0.500E+20,"object":{"nested":false},"string":"한글\ncontrol"}"#
        );
    }

    #[test]
    fn shared_response_snapshot_preserves_tags_headers_and_absence() {
        let response = SourceResponse {
            metadata: ResponseMetadata::new(
                200,
                HttpVersion::Http11,
                vec![
                    ResponseHeader::new("x-first", vec![0x66, 0x80]),
                    ResponseHeader::new("x-second", vec![0x32]),
                ],
            ),
            reply: SourceReply::<SourceValue>::Status(StatusEnvelope {
                code: SourceStatus::new("future-status"),
                message: None,
                evidence: SourceValue::object(BTreeMap::from([
                    ("explicit_null".to_owned(), SourceValue::null()),
                    (
                        "number".to_owned(),
                        SourceValue::number("1.20e+30").expect("valid source number"),
                    ),
                ])),
            }),
        };

        assert_eq!(
            serde_json::to_string(&response).unwrap(),
            r#"{"metadata":{"status":200,"version":"http/1.1","headers":[{"name":"x-first","value":[102,128]},{"name":"x-second","value":[50]}]},"reply":{"kind":"status","value":{"code":"future-status","evidence":{"explicit_null":null,"number":1.20e+30}}}}"#
        );
    }

    #[test]
    fn generated_response_flattens_additive_fields_in_source_order() {
        let response = decode_company_json_response(SourceValue::object(BTreeMap::from([
            ("corp_name".to_owned(), SourceValue::string("Example")),
            (
                "future_number".to_owned(),
                SourceValue::number("9.90E-7").expect("valid source number"),
            ),
        ])))
        .expect("fixture must decode through the generated public shape");

        assert_eq!(
            serde_json::to_string(&SourceReply::Success(response)).unwrap(),
            r#"{"kind":"success","value":{"corp_name":"Example","future_number":9.90E-7}}"#
        );
    }

    #[test]
    fn generated_response_serializes_nested_lists_and_objects() {
        let response = decode_list_json_response(SourceValue::object(BTreeMap::from([
            (
                "future_root".to_owned(),
                SourceValue::array(vec![SourceValue::string("source")]),
            ),
            (
                "list".to_owned(),
                SourceValue::array(vec![SourceValue::object(BTreeMap::from([
                    ("corp_name".to_owned(), SourceValue::string("Example")),
                    (
                        "future_nested".to_owned(),
                        SourceValue::object(BTreeMap::from([(
                            "flag".to_owned(),
                            SourceValue::boolean(false),
                        )])),
                    ),
                ]))]),
            ),
            (
                "page_no".to_owned(),
                SourceValue::number("1").expect("valid source number"),
            ),
        ])))
        .expect("nested fixture must decode through the generated public shape");

        assert_eq!(
            serde_json::to_string(&response).unwrap(),
            r#"{"list":[{"corp_name":"Example","future_nested":{"flag":false}}],"page_no":1,"future_root":["source"]}"#
        );
    }

    #[test]
    fn unknown_http_version_preserves_the_sdk_string() {
        assert_eq!(
            serde_json::to_string(&HttpVersion::Other("HTTP/FUTURE".to_owned())).unwrap(),
            r#""HTTP/FUTURE""#
        );
    }
}
