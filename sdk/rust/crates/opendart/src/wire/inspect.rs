use std::collections::BTreeMap;

use quick_xml::{Reader, events::Event};

use super::{SourceReply, SourceStatus, SourceValue, StatusEnvelope};

const MAX_NESTING_DEPTH: usize = 64;

/// The source representation inspected by a bounded wire parser.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
#[non_exhaustive]
pub enum EnvelopeFormat {
    /// A JSON response body.
    Json,
    /// An XML response body.
    Xml,
}

/// A response body exceeded the configured buffered-envelope limit.
#[derive(Debug, thiserror::Error)]
#[error("the response body exceeds the configured {limit}-byte envelope limit")]
pub struct BodyLimitError {
    limit: usize,
}

impl BodyLimitError {
    #[cfg(all(feature = "client-reqwest", not(target_family = "wasm")))]
    pub(crate) const fn new(limit: usize) -> Self {
        Self { limit }
    }

    /// Returns the inclusive buffered-envelope limit.
    #[must_use]
    pub const fn limit(&self) -> usize {
        self.limit
    }
}

/// A bounded response body was not a valid source envelope.
#[derive(Debug, thiserror::Error)]
#[error("the bounded {format:?} response is not a valid source envelope")]
pub struct EnvelopeError {
    format: EnvelopeFormat,
}

impl EnvelopeError {
    /// Returns the representation that could not be inspected.
    #[must_use]
    pub const fn format(&self) -> EnvelopeFormat {
        self.format
    }
}

/// A bounded wire inspection failure.
#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum WireInspectError {
    /// The input exceeded the configured byte limit.
    #[error(transparent)]
    BodyLimit(#[from] BodyLimitError),
    /// The input was malformed or violated the safe envelope grammar.
    #[error(transparent)]
    Envelope(#[from] EnvelopeError),
}

/// A transport-independent, bounded JSON and XML envelope inspector.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub struct WireInspector {
    max_envelope_bytes: usize,
}

impl WireInspector {
    /// Creates an inspector with a nonzero inclusive body-size limit.
    ///
    /// Returns `None` for zero because the safe inspector has no unbounded mode.
    #[must_use]
    pub const fn new(max_envelope_bytes: usize) -> Option<Self> {
        if max_envelope_bytes == 0 {
            None
        } else {
            Some(Self { max_envelope_bytes })
        }
    }

    /// Returns the inclusive maximum buffered envelope size.
    #[must_use]
    pub const fn max_envelope_bytes(&self) -> usize {
        self.max_envelope_bytes
    }

    /// Inspects a bounded JSON body without imposing source-status policy.
    pub fn inspect_json(&self, body: &[u8]) -> Result<SourceReply<SourceValue>, WireInspectError> {
        self.check_size(body)?;
        let parsed: serde_json::Value =
            serde_json::from_slice(body).map_err(|_| envelope_error(EnvelopeFormat::Json))?;
        let numbers =
            json_number_tokens(body).ok_or_else(|| envelope_error(EnvelopeFormat::Json))?;
        let mut numbers = numbers.into_iter();
        let mut nodes = 0;
        let value = json_value(parsed, 1, &mut nodes, &mut numbers)
            .ok_or_else(|| envelope_error(EnvelopeFormat::Json))?;
        if numbers.next().is_some() {
            return Err(envelope_error(EnvelopeFormat::Json).into());
        }
        Ok(classify_status(value))
    }

    /// Inspects a bounded XML body without resolving external entities.
    pub fn inspect_xml(&self, body: &[u8]) -> Result<SourceReply<SourceValue>, WireInspectError> {
        self.check_size(body)?;
        let value = xml_value(body).map_err(|()| envelope_error(EnvelopeFormat::Xml))?;
        Ok(classify_status(value))
    }

    fn check_size(&self, body: &[u8]) -> Result<(), BodyLimitError> {
        if body.len() > self.max_envelope_bytes {
            Err(BodyLimitError {
                limit: self.max_envelope_bytes,
            })
        } else {
            Ok(())
        }
    }
}

fn envelope_error(format: EnvelopeFormat) -> EnvelopeError {
    EnvelopeError { format }
}

fn json_value(
    value: serde_json::Value,
    depth: usize,
    nodes: &mut usize,
    numbers: &mut impl Iterator<Item = String>,
) -> Option<SourceValue> {
    if depth > MAX_NESTING_DEPTH {
        return None;
    }
    *nodes = nodes.checked_add(1)?;
    match value {
        serde_json::Value::Null => Some(SourceValue::null()),
        serde_json::Value::Bool(value) => Some(SourceValue::boolean(value)),
        serde_json::Value::Number(_) => Some(SourceValue::number(numbers.next()?)),
        serde_json::Value::String(value) => Some(SourceValue::string(value)),
        serde_json::Value::Array(values) => values
            .into_iter()
            .map(|value| json_value(value, depth + 1, nodes, numbers))
            .collect::<Option<Vec<_>>>()
            .map(SourceValue::array),
        serde_json::Value::Object(values) => values
            .into_iter()
            .map(|(name, value)| Some((name, json_value(value, depth + 1, nodes, numbers)?)))
            .collect::<Option<BTreeMap<_, _>>>()
            .map(SourceValue::object),
    }
}

fn json_number_tokens(body: &[u8]) -> Option<Vec<String>> {
    let mut tokens = Vec::new();
    let mut index = 0;
    while index < body.len() {
        match body[index] {
            b'"' => {
                index += 1;
                while index < body.len() {
                    match body[index] {
                        b'\\' => index = index.checked_add(2)?,
                        b'"' => {
                            index += 1;
                            break;
                        }
                        _ => index += 1,
                    }
                }
            }
            b'-' | b'0'..=b'9' => {
                let start = index;
                index += 1;
                while index < body.len()
                    && matches!(body[index], b'0'..=b'9' | b'.' | b'e' | b'E' | b'+' | b'-')
                {
                    index += 1;
                }
                tokens.push(std::str::from_utf8(&body[start..index]).ok()?.to_owned());
            }
            _ => index += 1,
        }
    }
    Some(tokens)
}

#[derive(Debug)]
struct XmlFrame {
    name: String,
    fields: BTreeMap<String, SourceValue>,
    text: String,
}

impl XmlFrame {
    fn finish(mut self) -> SourceValue {
        if self.fields.is_empty() {
            return SourceValue::string(self.text);
        }
        if !self.text.is_empty() {
            self.fields
                .insert("$text".to_owned(), SourceValue::string(self.text));
        }
        SourceValue::object(self.fields)
    }
}

fn xml_value(body: &[u8]) -> Result<SourceValue, ()> {
    let mut reader = Reader::from_reader(body);
    reader.config_mut().trim_text(true);
    let mut stack: Vec<XmlFrame> = Vec::new();
    let mut root = None;
    let mut nodes = 0usize;

    loop {
        match reader.read_event().map_err(|_| ())? {
            Event::Start(start) => {
                if stack.len() >= MAX_NESTING_DEPTH {
                    return Err(());
                }
                nodes = nodes.checked_add(1).ok_or(())?;
                let name = reader
                    .decoder()
                    .decode(start.name().as_ref())
                    .map_err(|_| ())?
                    .into_owned();
                let mut fields = BTreeMap::new();
                for attribute in start.attributes() {
                    let attribute = attribute.map_err(|_| ())?;
                    let key = reader
                        .decoder()
                        .decode(attribute.key.as_ref())
                        .map_err(|_| ())?;
                    let value = attribute
                        .decoded_and_normalized_value(
                            quick_xml::XmlVersion::Implicit1_0,
                            reader.decoder(),
                        )
                        .map_err(|_| ())?;
                    fields.insert(format!("@{key}"), SourceValue::string(value.into_owned()));
                }
                stack.push(XmlFrame {
                    name,
                    fields,
                    text: String::new(),
                });
            }
            Event::Empty(empty) => {
                if stack.len() >= MAX_NESTING_DEPTH {
                    return Err(());
                }
                nodes = nodes.checked_add(1).ok_or(())?;
                let name = reader
                    .decoder()
                    .decode(empty.name().as_ref())
                    .map_err(|_| ())?
                    .into_owned();
                let mut fields = BTreeMap::new();
                for attribute in empty.attributes() {
                    let attribute = attribute.map_err(|_| ())?;
                    let key = reader
                        .decoder()
                        .decode(attribute.key.as_ref())
                        .map_err(|_| ())?;
                    let value = attribute
                        .decoded_and_normalized_value(
                            quick_xml::XmlVersion::Implicit1_0,
                            reader.decoder(),
                        )
                        .map_err(|_| ())?;
                    fields.insert(format!("@{key}"), SourceValue::string(value.into_owned()));
                }
                let value = XmlFrame {
                    name: name.clone(),
                    fields,
                    text: String::new(),
                }
                .finish();
                attach_xml_value(&mut stack, &mut root, name, value)?;
            }
            Event::End(_) => {
                let frame = stack.pop().ok_or(())?;
                let name = frame.name.clone();
                attach_xml_value(&mut stack, &mut root, name, frame.finish())?;
            }
            Event::Text(text) => {
                let frame = stack.last_mut().ok_or(())?;
                frame.text.push_str(&text.decode().map_err(|_| ())?);
            }
            Event::CData(text) => {
                let frame = stack.last_mut().ok_or(())?;
                frame.text.push_str(&text.decode().map_err(|_| ())?);
            }
            Event::GeneralRef(reference) => {
                let frame = stack.last_mut().ok_or(())?;
                frame
                    .text
                    .push(decode_reference(&reference.decode().map_err(|_| ())?)?);
            }
            Event::DocType(_) => return Err(()),
            Event::Decl(_) | Event::Comment(_) | Event::PI(_) => {}
            Event::Eof => break,
        }
    }

    if !stack.is_empty() {
        return Err(());
    }
    root.map(|(_, value)| value).ok_or(())
}

fn attach_xml_value(
    stack: &mut [XmlFrame],
    root: &mut Option<(String, SourceValue)>,
    name: String,
    value: SourceValue,
) -> Result<(), ()> {
    if let Some(parent) = stack.last_mut() {
        if let Some(existing) = parent.fields.get_mut(&name) {
            existing.append_repeated(value);
        } else {
            parent.fields.insert(name, value);
        }
        Ok(())
    } else if root.is_none() {
        *root = Some((name, value));
        Ok(())
    } else {
        Err(())
    }
}

fn decode_reference(reference: &str) -> Result<char, ()> {
    match reference {
        "amp" => Ok('&'),
        "lt" => Ok('<'),
        "gt" => Ok('>'),
        "apos" => Ok('\''),
        "quot" => Ok('"'),
        value if value.starts_with("#x") => u32::from_str_radix(&value[2..], 16)
            .ok()
            .and_then(char::from_u32)
            .ok_or(()),
        value if value.starts_with('#') => value[1..]
            .parse::<u32>()
            .ok()
            .and_then(char::from_u32)
            .ok_or(()),
        _ => Err(()),
    }
}

fn classify_status(value: SourceValue) -> SourceReply<SourceValue> {
    let field_count = value.fields().count();
    let status = value.get("status").and_then(SourceValue::as_str);
    let status_only = status.is_some()
        && field_count <= 2
        && value
            .fields()
            .all(|(name, _)| matches!(name, "status" | "message"));

    if status_only {
        SourceReply::Status(StatusEnvelope {
            code: SourceStatus::new(status.expect("checked above")),
            message: value.get("message").cloned(),
        })
    } else {
        SourceReply::Success(value)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn inspector(limit: usize) -> WireInspector {
        WireInspector::new(limit).expect("nonzero test limit")
    }

    #[test]
    fn body_limit_is_inclusive() {
        assert!(inspector(2).inspect_json(b"[]").is_ok());
        let error = inspector(1).inspect_json(b"[]").unwrap_err();
        assert_eq!(
            error.to_string(),
            "the response body exceeds the configured 1-byte envelope limit"
        );
    }

    #[test]
    fn json_retains_unknown_values_and_exact_number_spelling() {
        let reply = inspector(256)
            .inspect_json(br#"{"status":"000","future":1.20e3,"flag":true,"none":null}"#)
            .unwrap();
        let SourceReply::Success(value) = reply else {
            panic!("payload fields must prevent status-only classification");
        };
        assert_eq!(
            value.get("future").and_then(SourceValue::as_number_str),
            Some("1.20e3")
        );
        assert_eq!(value.get("flag").and_then(SourceValue::as_bool), Some(true));
        assert_eq!(
            value.get("none").map(SourceValue::kind),
            Some(super::super::SourceValueKind::Null)
        );
    }

    #[test]
    fn every_status_string_remains_evidence_without_policy() {
        for code in ["000", "013", "999", "future"] {
            let body = format!(r#"{{"status":"{code}","message":"source"}}"#);
            let SourceReply::Status(status) = inspector(128).inspect_json(body.as_bytes()).unwrap()
            else {
                panic!("status-only envelope was not recognized");
            };
            assert_eq!(status.code.as_str(), code);
        }
    }

    #[test]
    fn xml_retains_attributes_repeated_elements_and_character_references() {
        let body = br#"<?xml version="1.0"?><result future="yes"><item>A&amp;B</item><item><![CDATA[C]]></item></result>"#;
        let SourceReply::Success(value) = inspector(256).inspect_xml(body).unwrap() else {
            panic!("expected a success value");
        };
        assert_eq!(
            value.get("@future").and_then(SourceValue::as_str),
            Some("yes")
        );
        let items = value.get("item").and_then(SourceValue::as_array).unwrap();
        assert_eq!(items[0].as_str(), Some("A&B"));
        assert_eq!(items[1].as_str(), Some("C"));
    }

    #[test]
    fn equivalent_empty_xml_elements_have_the_same_value() {
        let empty = inspector(64).inspect_xml(b"<result><item/></result>");
        let explicit = inspector(64).inspect_xml(b"<result><item></item></result>");
        assert_eq!(empty.unwrap(), explicit.unwrap());
    }

    #[test]
    fn many_repeated_xml_siblings_append_in_place() {
        let body = format!("<result>{}</result>", "<item/>".repeat(10_000));
        let SourceReply::Success(value) =
            inspector(body.len()).inspect_xml(body.as_bytes()).unwrap()
        else {
            panic!("expected repeated success evidence");
        };
        assert_eq!(
            value
                .get("item")
                .and_then(SourceValue::as_array)
                .map(<[_]>::len),
            Some(10_000)
        );
    }

    #[test]
    fn xml_rejects_doctype_external_entities_and_excessive_depth() {
        assert!(inspector(256)
            .inspect_xml(br#"<!DOCTYPE result [<!ENTITY x SYSTEM "file:///etc/passwd">]><result>&x;</result>"#)
            .is_err());
        let nested = format!(
            "{}x{}",
            "<a>".repeat(MAX_NESTING_DEPTH + 1),
            "</a>".repeat(MAX_NESTING_DEPTH + 1)
        );
        assert!(
            inspector(nested.len())
                .inspect_xml(nested.as_bytes())
                .is_err()
        );
    }

    #[test]
    fn malformed_and_multiple_root_documents_are_rejected() {
        for body in [
            b"{".as_slice(),
            b"<result>".as_slice(),
            b"<a/><b/>".as_slice(),
        ] {
            let result = if body.starts_with(b"{") {
                inspector(64).inspect_json(body)
            } else {
                inspector(64).inspect_xml(body)
            };
            assert!(result.is_err());
        }
    }
}
