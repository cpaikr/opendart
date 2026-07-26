use std::{
    borrow::Cow,
    collections::{BTreeMap, BTreeSet},
};

use quick_xml::{Reader, events::Event};
use roxmltree::{Document, ParsingOptions};

use super::{SourceReply, SourceStatus, SourceValue, StatusEnvelope};

const MAX_NESTING_DEPTH: usize = 64;
pub(crate) const MAX_XML_ATTRIBUTES_PER_ELEMENT: usize = 128;

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
        if !json_members_are_unique(body) {
            return Err(envelope_error(EnvelopeFormat::Json).into());
        }
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
    ///
    /// Elements with non-whitespace text interleaved with children retain
    /// concatenated text in `$text` and their complete normalized order in
    /// `$content`. Whitespace-only text between children is treated as document
    /// formatting. Child values live only in one-field objects within `$content`
    /// so nested evidence is never duplicated.
    pub fn inspect_xml(&self, body: &[u8]) -> Result<SourceReply<SourceValue>, WireInspectError> {
        self.inspect_xml_with_root(body).map(|(_, reply)| reply)
    }

    pub(crate) fn inspect_xml_with_root(
        &self,
        body: &[u8],
    ) -> Result<(String, SourceReply<SourceValue>), WireInspectError> {
        self.check_size(body)?;
        validate_xml_document(body).map_err(|()| envelope_error(EnvelopeFormat::Xml))?;
        let (root, value) = xml_value(body).map_err(|()| envelope_error(EnvelopeFormat::Xml))?;
        Ok((root, classify_status(value)))
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

fn validate_xml_document(body: &[u8]) -> Result<(), ()> {
    // Preflight limits structural work before roxmltree's recursive tokenizer
    // and owns declaration/PI rules on which the two parsers otherwise differ.
    validate_xml_preflight(body)?;
    let text = std::str::from_utf8(body).map_err(|_| ())?;
    let options = ParsingOptions {
        allow_dtd: false,
        nodes_limit: u32::try_from(body.len()).unwrap_or(u32::MAX).max(1),
        entity_resolver: None,
    };
    Document::parse_with_options(text, options).map_err(|_| ())?;
    Ok(())
}

fn validate_xml_preflight(body: &[u8]) -> Result<(), ()> {
    let mut reader = Reader::from_reader(body);
    let config = reader.config_mut();
    config.check_comments = true;
    let mut depth = 0usize;
    let mut seen_event = false;

    loop {
        match reader.read_event().map_err(|_| ())? {
            Event::Start(start) => {
                seen_event = true;
                validate_xml_attribute_count(&start)?;
                depth = depth.checked_add(1).ok_or(())?;
                if depth > MAX_NESTING_DEPTH {
                    return Err(());
                }
            }
            Event::Empty(empty) => {
                seen_event = true;
                validate_xml_attribute_count(&empty)?;
                if depth >= MAX_NESTING_DEPTH {
                    return Err(());
                }
            }
            Event::End(_) => {
                seen_event = true;
                depth = depth.checked_sub(1).ok_or(())?;
            }
            Event::Decl(declaration) => {
                if seen_event {
                    return Err(());
                }
                validate_xml_declaration(&declaration)?;
                seen_event = true;
            }
            Event::PI(instruction) => {
                seen_event = true;
                validate_xml_processing_instruction(&instruction)?;
            }
            Event::DocType(_) => return Err(()),
            Event::Eof => return Ok(()),
            Event::Text(_) | Event::CData(_) | Event::Comment(_) | Event::GeneralRef(_) => {
                seen_event = true;
            }
        }
    }
}

fn validate_xml_attribute_count(element: &quick_xml::events::BytesStart<'_>) -> Result<(), ()> {
    // Bound roxmltree's per-element duplicate and namespace resolution before
    // its full-document authority pass.
    let mut attributes = element.attributes();
    attributes.with_checks(false);
    for (index, attribute) in attributes.enumerate() {
        attribute.map_err(|_| ())?;
        if index >= MAX_XML_ATTRIBUTES_PER_ELEMENT {
            return Err(());
        }
    }
    Ok(())
}

fn validate_xml_processing_instruction(
    instruction: &quick_xml::events::BytesPI<'_>,
) -> Result<(), ()> {
    let content = std::str::from_utf8(instruction.as_ref()).map_err(|_| ())?;
    let mut chars = content.char_indices();
    let (_, first) = chars.next().ok_or(())?;
    if !is_xml_name_start(first) {
        return Err(());
    }

    let mut target_end = first.len_utf8();
    for (index, value) in chars {
        if !is_xml_name_char(value) {
            target_end = index;
            break;
        }
        target_end = index + value.len_utf8();
    }

    let target = &content[..target_end];
    let data = &content[target_end..];
    if !data.is_empty() && !data.chars().next().is_some_and(is_xml_whitespace) {
        return Err(());
    }
    if target.eq_ignore_ascii_case("xml") {
        return Err(());
    }
    Ok(())
}

fn validate_xml_declaration(declaration: &quick_xml::events::BytesDecl<'_>) -> Result<(), ()> {
    let mut input = declaration.as_ref();
    input = input.strip_prefix(b"xml").ok_or(())?;
    if !consume_xml_space(&mut input) {
        return Err(());
    }

    if take_xml_declaration_attribute(&mut input, b"version")? != b"1.0" {
        return Err(());
    }
    let mut separated = consume_xml_space(&mut input);
    if input.is_empty() {
        return Ok(());
    }
    if !separated {
        return Err(());
    }

    if input.starts_with(b"encoding") {
        let encoding = take_xml_declaration_attribute(&mut input, b"encoding")?;
        if !encoding.eq_ignore_ascii_case(b"UTF-8") {
            return Err(());
        }
        separated = consume_xml_space(&mut input);
        if input.is_empty() {
            return Ok(());
        }
        if !separated {
            return Err(());
        }
    }

    let standalone = take_xml_declaration_attribute(&mut input, b"standalone")?;
    if !matches!(standalone, b"yes" | b"no") {
        return Err(());
    }
    consume_xml_space(&mut input);
    input.is_empty().then_some(()).ok_or(())
}

fn consume_xml_space(input: &mut &[u8]) -> bool {
    let start_len = input.len();
    while input
        .first()
        .is_some_and(|byte| matches!(byte, b' ' | b'\t' | b'\r' | b'\n'))
    {
        *input = &input[1..];
    }
    input.len() != start_len
}

fn take_xml_declaration_attribute<'a>(input: &mut &'a [u8], name: &[u8]) -> Result<&'a [u8], ()> {
    *input = input.strip_prefix(name).ok_or(())?;
    consume_xml_space(input);
    *input = input.strip_prefix(b"=").ok_or(())?;
    consume_xml_space(input);

    let quote = *input.first().ok_or(())?;
    if !matches!(quote, b'\'' | b'"') {
        return Err(());
    }
    *input = &input[1..];
    let end = input.iter().position(|byte| *byte == quote).ok_or(())?;
    let value = &input[..end];
    *input = &input[end + 1..];
    Ok(value)
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
        serde_json::Value::Number(_) => Some(SourceValue::number_from_valid_json(numbers.next()?)),
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

struct JsonMemberScanner<'a> {
    body: &'a [u8],
    index: usize,
}

impl JsonMemberScanner<'_> {
    fn scan_value(&mut self, depth: usize) -> Option<()> {
        if depth > MAX_NESTING_DEPTH {
            return None;
        }
        self.skip_whitespace();
        match self.body.get(self.index)? {
            b'{' => self.scan_object(depth),
            b'[' => self.scan_array(depth),
            b'"' => self.scan_string().map(|_| ()),
            _ => self.scan_scalar(),
        }
    }

    fn scan_object(&mut self, depth: usize) -> Option<()> {
        self.index += 1;
        self.skip_whitespace();
        if self.consume(b'}') {
            return Some(());
        }

        let mut names = BTreeSet::new();
        loop {
            self.skip_whitespace();
            let name = self.scan_string()?;
            let name: String = serde_json::from_slice(name).ok()?;
            if !names.insert(name) {
                return None;
            }
            self.skip_whitespace();
            if !self.consume(b':') {
                return None;
            }
            self.scan_value(depth + 1)?;
            self.skip_whitespace();
            if self.consume(b'}') {
                return Some(());
            }
            if !self.consume(b',') {
                return None;
            }
        }
    }

    fn scan_array(&mut self, depth: usize) -> Option<()> {
        self.index += 1;
        self.skip_whitespace();
        if self.consume(b']') {
            return Some(());
        }

        loop {
            self.scan_value(depth + 1)?;
            self.skip_whitespace();
            if self.consume(b']') {
                return Some(());
            }
            if !self.consume(b',') {
                return None;
            }
        }
    }

    fn scan_string(&mut self) -> Option<&[u8]> {
        let start = self.index;
        if !self.consume(b'"') {
            return None;
        }
        while let Some(byte) = self.body.get(self.index) {
            match byte {
                b'\\' => self.index = self.index.checked_add(2)?,
                b'"' => {
                    self.index += 1;
                    return self.body.get(start..self.index);
                }
                _ => self.index += 1,
            }
        }
        None
    }

    fn scan_scalar(&mut self) -> Option<()> {
        let start = self.index;
        while let Some(byte) = self.body.get(self.index) {
            if byte.is_ascii_whitespace() || matches!(byte, b',' | b']' | b'}') {
                break;
            }
            self.index += 1;
        }
        (self.index > start).then_some(())
    }

    fn skip_whitespace(&mut self) {
        while self
            .body
            .get(self.index)
            .is_some_and(u8::is_ascii_whitespace)
        {
            self.index += 1;
        }
    }

    fn consume(&mut self, expected: u8) -> bool {
        if self.body.get(self.index) == Some(&expected) {
            self.index += 1;
            true
        } else {
            false
        }
    }
}

fn json_members_are_unique(body: &[u8]) -> bool {
    let mut scanner = JsonMemberScanner { body, index: 0 };
    if scanner.scan_value(1).is_none() {
        return false;
    }
    scanner.skip_whitespace();
    scanner.index == body.len()
}

#[derive(Debug)]
struct XmlFrame {
    name: String,
    fields: BTreeMap<String, SourceValue>,
    content: Vec<XmlContent>,
}

impl XmlFrame {
    fn push_text(&mut self, text: &str) {
        if text.is_empty() {
            return;
        }
        if let Some(XmlContent::Text(existing)) = self.content.last_mut() {
            existing.push_str(text);
        } else {
            self.content.push(XmlContent::Text(text.to_owned()));
        }
    }

    fn finish(mut self) -> SourceValue {
        let has_children = self
            .content
            .iter()
            .any(|content| matches!(content, XmlContent::Child { .. }));
        let has_text = self.content.iter().any(|content| {
            matches!(content, XmlContent::Text(value) if !value.chars().all(is_xml_whitespace))
        });

        if !has_children {
            let text = self
                .content
                .into_iter()
                .filter_map(XmlContent::into_text)
                .collect::<String>();
            if self.fields.is_empty() {
                return SourceValue::string(text);
            }
            if !text.is_empty() {
                self.fields
                    .insert("$text".to_owned(), SourceValue::string(text));
            }
            return SourceValue::object(self.fields);
        }

        if !has_text {
            for content in self.content {
                if let XmlContent::Child { name, value } = content {
                    attach_xml_field(&mut self.fields, name, value);
                }
            }
            return SourceValue::object(self.fields);
        }

        let mut text = String::new();
        for content in &self.content {
            if let XmlContent::Text(value) = content {
                text.push_str(value);
            }
        }
        if !text.is_empty() {
            self.fields
                .insert("$text".to_owned(), SourceValue::string(text));
        }
        let ordered = self
            .content
            .into_iter()
            .map(XmlContent::into_source_value)
            .collect();
        self.fields
            .insert("$content".to_owned(), SourceValue::array(ordered));
        SourceValue::object(self.fields)
    }
}

#[derive(Debug)]
enum XmlContent {
    Text(String),
    Child { name: String, value: SourceValue },
}

impl XmlContent {
    fn into_text(self) -> Option<String> {
        match self {
            Self::Text(value) => Some(value),
            Self::Child { .. } => None,
        }
    }

    fn into_source_value(self) -> SourceValue {
        match self {
            Self::Text(value) => SourceValue::string(value),
            Self::Child { name, value } => SourceValue::object(BTreeMap::from([(name, value)])),
        }
    }
}

fn xml_value(body: &[u8]) -> Result<(String, SourceValue), ()> {
    let mut reader = Reader::from_reader(body);
    reader.config_mut().trim_text(false);
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
                validate_xml_text(&name)?;
                let mut fields = BTreeMap::new();
                for attribute in start.attributes() {
                    let attribute = attribute.map_err(|_| ())?;
                    let key = reader
                        .decoder()
                        .decode(attribute.key.as_ref())
                        .map_err(|_| ())?;
                    validate_xml_text(&key)?;
                    let value = attribute
                        .decoded_and_normalized_value(
                            quick_xml::XmlVersion::Implicit1_0,
                            reader.decoder(),
                        )
                        .map_err(|_| ())?;
                    validate_xml_text(&value)?;
                    fields.insert(format!("@{key}"), SourceValue::string(value.into_owned()));
                }
                stack.push(XmlFrame {
                    name,
                    fields,
                    content: Vec::new(),
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
                validate_xml_text(&name)?;
                let mut fields = BTreeMap::new();
                for attribute in empty.attributes() {
                    let attribute = attribute.map_err(|_| ())?;
                    let key = reader
                        .decoder()
                        .decode(attribute.key.as_ref())
                        .map_err(|_| ())?;
                    validate_xml_text(&key)?;
                    let value = attribute
                        .decoded_and_normalized_value(
                            quick_xml::XmlVersion::Implicit1_0,
                            reader.decoder(),
                        )
                        .map_err(|_| ())?;
                    validate_xml_text(&value)?;
                    fields.insert(format!("@{key}"), SourceValue::string(value.into_owned()));
                }
                let value = XmlFrame {
                    name: name.clone(),
                    fields,
                    content: Vec::new(),
                }
                .finish();
                attach_xml_value(&mut stack, &mut root, name, value)?;
            }
            Event::End(end) => {
                let frame = stack.pop().ok_or(())?;
                let end_name = reader
                    .decoder()
                    .decode(end.name().as_ref())
                    .map_err(|_| ())?
                    .into_owned();
                validate_xml_text(&end_name)?;
                if end_name != frame.name {
                    return Err(());
                }
                let name = frame.name.clone();
                attach_xml_value(&mut stack, &mut root, name, frame.finish())?;
            }
            Event::Text(text) => {
                let text = text.decode().map_err(|_| ())?;
                let text = normalize_xml_line_endings(&text);
                validate_xml_text(&text)?;
                if let Some(frame) = stack.last_mut() {
                    frame.push_text(&text);
                } else if !text.chars().all(is_xml_whitespace) {
                    return Err(());
                }
            }
            Event::CData(text) => {
                let frame = stack.last_mut().ok_or(())?;
                let text = text.decode().map_err(|_| ())?;
                let text = normalize_xml_line_endings(&text);
                validate_xml_text(&text)?;
                frame.push_text(&text);
            }
            Event::GeneralRef(reference) => {
                let frame = stack.last_mut().ok_or(())?;
                let value = decode_reference(&reference.decode().map_err(|_| ())?)?;
                frame.push_text(&value.to_string());
            }
            Event::DocType(_) => return Err(()),
            Event::Decl(declaration) => {
                let declaration = reader
                    .decoder()
                    .decode(declaration.as_ref())
                    .map_err(|_| ())?;
                validate_xml_text(&declaration)?;
            }
            Event::Comment(comment) => {
                let comment = comment.decode().map_err(|_| ())?;
                validate_xml_text(&comment)?;
            }
            Event::PI(instruction) => {
                let instruction = reader
                    .decoder()
                    .decode(instruction.as_ref())
                    .map_err(|_| ())?;
                validate_xml_text(&instruction)?;
            }
            Event::Eof => break,
        }
    }

    if !stack.is_empty() {
        return Err(());
    }
    root.ok_or(())
}

fn normalize_xml_line_endings(value: &str) -> Cow<'_, str> {
    if !value.as_bytes().contains(&b'\r') {
        return Cow::Borrowed(value);
    }

    let mut normalized = String::with_capacity(value.len());
    let mut remaining = value;
    while let Some(index) = remaining.find('\r') {
        normalized.push_str(&remaining[..index]);
        remaining = &remaining[index + 1..];
        if let Some(without_lf) = remaining.strip_prefix('\n') {
            remaining = without_lf;
        }
        normalized.push('\n');
    }
    normalized.push_str(remaining);
    Cow::Owned(normalized)
}

fn attach_xml_value(
    stack: &mut [XmlFrame],
    root: &mut Option<(String, SourceValue)>,
    name: String,
    value: SourceValue,
) -> Result<(), ()> {
    if let Some(parent) = stack.last_mut() {
        parent.content.push(XmlContent::Child { name, value });
        Ok(())
    } else if root.is_none() {
        *root = Some((name, value));
        Ok(())
    } else {
        Err(())
    }
}

fn attach_xml_field(fields: &mut BTreeMap<String, SourceValue>, name: String, value: SourceValue) {
    if let Some(existing) = fields.get_mut(&name) {
        existing.append_repeated(value);
    } else {
        fields.insert(name, value);
    }
}

fn validate_xml_text(value: &str) -> Result<(), ()> {
    value.chars().all(is_xml_1_0_char).then_some(()).ok_or(())
}

fn is_xml_1_0_char(value: char) -> bool {
    matches!(value, '\u{9}' | '\u{A}' | '\u{D}')
        || matches!(value, '\u{20}'..='\u{D7FF}' | '\u{E000}'..='\u{FFFD}' | '\u{10000}'..='\u{10FFFF}')
}

fn is_xml_name_start(value: char) -> bool {
    matches!(
        value,
        ':' | 'A'..='Z'
            | '_'
            | 'a'..='z'
            | '\u{C0}'..='\u{D6}'
            | '\u{D8}'..='\u{F6}'
            | '\u{F8}'..='\u{2FF}'
            | '\u{370}'..='\u{37D}'
            | '\u{37F}'..='\u{1FFF}'
            | '\u{200C}'..='\u{200D}'
            | '\u{2070}'..='\u{218F}'
            | '\u{2C00}'..='\u{2FEF}'
            | '\u{3001}'..='\u{D7FF}'
            | '\u{F900}'..='\u{FDCF}'
            | '\u{FDF0}'..='\u{FFFD}'
            | '\u{10000}'..='\u{EFFFF}'
    )
}

fn is_xml_name_char(value: char) -> bool {
    is_xml_name_start(value)
        || matches!(
            value,
            '-' | '.' | '0'..='9' | '\u{B7}' | '\u{300}'..='\u{36F}' | '\u{203F}'..='\u{2040}'
        )
}

fn is_xml_whitespace(value: char) -> bool {
    matches!(value, '\u{9}' | '\u{A}' | '\u{D}' | ' ')
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
            .filter(|value| is_xml_1_0_char(*value))
            .ok_or(()),
        value if value.starts_with('#') => value[1..]
            .parse::<u32>()
            .ok()
            .and_then(char::from_u32)
            .filter(|value| is_xml_1_0_char(*value))
            .ok_or(()),
        _ => Err(()),
    }
}

fn classify_status(value: SourceValue) -> SourceReply<SourceValue> {
    let field_count = value.fields().count();
    let status = value
        .get("status")
        .and_then(SourceValue::as_str)
        .map(str::to_owned);
    let status_only = status.is_some()
        && field_count <= 2
        && value
            .fields()
            .all(|(name, _)| matches!(name, "status" | "message"));
    let source_status = status
        .as_deref()
        .is_some_and(|status| status != SourceStatus::SUCCESS);

    if status_only || source_status {
        SourceReply::Status(StatusEnvelope {
            code: SourceStatus::new(status.expect("checked above")),
            message: value.get("message").cloned(),
            evidence: value,
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
    fn json_rejects_duplicate_object_members_after_name_decoding() {
        for body in [
            br#"{"status":"013","status":"000"}"#.as_slice(),
            br#"{"status":"013","sta\u0074us":"000"}"#.as_slice(),
            br#"{"outer":{"value":1,"value":2}}"#.as_slice(),
        ] {
            assert!(inspector(body.len()).inspect_json(body).is_err());
        }
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
    fn non_success_status_retains_additive_json_and_xml_evidence() {
        let SourceReply::Status(json) = inspector(256)
            .inspect_json(br#"{"status":"013","message":"none","request_id":"json-123"}"#)
            .unwrap()
        else {
            panic!("a non-success JSON status with additive fields must remain status evidence");
        };
        assert_eq!(json.code.as_str(), "013");
        assert_eq!(
            json.evidence
                .get("request_id")
                .and_then(SourceValue::as_str),
            Some("json-123")
        );

        let SourceReply::Status(xml) = inspector(256)
            .inspect_xml(
                br#"<result><status>013</status><message>none</message><request_id>xml-123</request_id></result>"#,
            )
            .unwrap()
        else {
            panic!("a non-success XML status with additive fields must remain status evidence");
        };
        assert_eq!(xml.code.as_str(), "013");
        assert_eq!(
            xml.evidence.get("request_id").and_then(SourceValue::as_str),
            Some("xml-123")
        );
    }

    #[test]
    fn success_status_with_payload_is_not_a_status_envelope() {
        for body in [
            br#"{"status":"000","payload":"json"}"#.as_slice(),
            br#"{"status":"000","message":"ok","future":true}"#.as_slice(),
        ] {
            assert!(matches!(
                inspector(128).inspect_json(body).unwrap(),
                SourceReply::Success(_)
            ));
        }

        assert!(matches!(
            inspector(128)
                .inspect_xml(b"<result><status>000</status><payload>xml</payload></result>")
                .unwrap(),
            SourceReply::Success(_)
        ));
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
    fn xml_accepts_supported_document_misc_namespaces_and_declarations() {
        for standalone in ["yes", "no"] {
            let body = format!(
                "\u{feff}<?xml version='1.0' encoding='utf-8' standalone='{standalone}'?>\
                 <!--before--><?before ok?>\
                 <결과 xmlns:x='urn:example' x:future='yes'>\
                 <?inside ok?><?처리 값?><x:item>값</x:item>\
                 </결과><?after ok?><!--after-->"
            );
            let SourceReply::Success(value) =
                inspector(body.len()).inspect_xml(body.as_bytes()).unwrap()
            else {
                panic!("supported XML document misc must remain valid");
            };
            assert_eq!(
                value.get("@x:future").and_then(SourceValue::as_str),
                Some("yes")
            );
            assert_eq!(
                value.get("x:item").and_then(SourceValue::as_str),
                Some("값")
            );
        }
        for separator in ["\t", "\r", "\n"] {
            let body = format!("<?xml{separator}version='1.0'?><result>ok</result>");
            assert!(
                inspector(body.len()).inspect_xml(body.as_bytes()).is_ok(),
                "XML space is valid between the declaration target and version"
            );
        }

        let long_text = "x".repeat(16 * 1024);
        let body = format!("<result><future>{long_text}</future></result>");
        let SourceReply::Success(value) =
            inspector(body.len()).inspect_xml(body.as_bytes()).unwrap()
        else {
            panic!("bounded long XML tokens must remain valid");
        };
        assert_eq!(
            value
                .get("future")
                .and_then(SourceValue::as_str)
                .map(str::len),
            Some(long_text.len())
        );

        let SourceReply::Success(value) = inspector(128)
            .inspect_xml(b"<result><item>ok</item></result >")
            .unwrap()
        else {
            panic!("closing-tag whitespace permitted by XML 1.0 must remain valid");
        };
        assert_eq!(value.get("item").and_then(SourceValue::as_str), Some("ok"));
    }

    #[test]
    fn xml_attribute_limit_is_inclusive() {
        let body_at_limit = xml_with_attributes(MAX_XML_ATTRIBUTES_PER_ELEMENT);
        assert!(
            inspector(body_at_limit.len())
                .inspect_xml(body_at_limit.as_bytes())
                .is_ok()
        );

        let body_over_limit = xml_with_attributes(MAX_XML_ATTRIBUTES_PER_ELEMENT + 1);
        assert!(
            inspector(body_over_limit.len())
                .inspect_xml(body_over_limit.as_bytes())
                .is_err()
        );
    }

    fn xml_with_attributes(count: usize) -> String {
        let mut body = String::from("<result");
        for index in 0..count {
            body.push_str(&format!(" a{index}=\"\""));
        }
        body.push_str("><status>013</status></result>");
        body
    }

    #[test]
    fn xml_normalizes_literal_line_endings_without_changing_references() {
        let body = b"<result value=\"a\tb\nc\rd\r\ne&#9;f&#10;g&#13;h\"><text>a\r\nb\rc\nd&#13;e</text><data><![CDATA[x\r\ny\rz\n]]></data></result>";
        let SourceReply::Success(value) = inspector(body.len()).inspect_xml(body).unwrap() else {
            panic!("line-ending fixture must remain a valid success value");
        };
        assert_eq!(
            value.get("@value").and_then(SourceValue::as_str),
            Some("a b c d e\tf\ng\rh")
        );
        assert_eq!(
            value.get("text").and_then(SourceValue::as_str),
            Some("a\nb\nc\nd\re")
        );
        assert_eq!(
            value.get("data").and_then(SourceValue::as_str),
            Some("x\ny\nz\n")
        );
    }

    #[test]
    fn unsupported_declaration_values_and_reserved_pi_targets_are_rejected() {
        for body in [
            br#"<?xml version="1.1"?><result/>"#.as_slice(),
            br#"<?xml version="2.0"?><result/>"#.as_slice(),
            br#"<?xml version="1.0" encoding="EUC-JP"?><result/>"#.as_slice(),
            br#"<?xml version="1.0" standalone="maybe"?><result/>"#.as_slice(),
            br#"<?xml version="1.0" bogus="value"?><result/>"#.as_slice(),
            b"<result><?xml\tversion=\"1.0\"?></result>".as_slice(),
            b"<?xml version=\"1.0\"?><?xml\nversion=\"1.0\"?><result/>".as_slice(),
            br#"<?pi?x?><result/>"#.as_slice(),
            br#"<?pi/data?><result/>"#.as_slice(),
            br#"<?xml?x?><result/>"#.as_slice(),
            br#"<?xml/data?><result/>"#.as_slice(),
            br#"<?XML note?><result/>"#.as_slice(),
            br#"<?xMl?><result/>"#.as_slice(),
        ] {
            assert!(inspector(body.len()).inspect_xml(body).is_err());
        }
    }

    #[test]
    fn xml_preserves_interleaved_mixed_content_order() {
        let SourceReply::Success(value) = inspector(128)
            .inspect_xml(b"<p>before <b>x</b> after</p>")
            .unwrap()
        else {
            panic!("expected mixed-content evidence");
        };
        assert_eq!(
            value.get("$text").and_then(SourceValue::as_str),
            Some("before  after")
        );
        let content = value
            .get("$content")
            .and_then(SourceValue::as_array)
            .unwrap();
        assert!(value.get("b").is_none());
        assert_eq!(content[0].as_str(), Some("before "));
        assert_eq!(content[1].get("b").and_then(SourceValue::as_str), Some("x"));
        assert_eq!(content[2].as_str(), Some(" after"));

        let reordered = inspector(128)
            .inspect_xml(b"<p><b>x</b>before  after</p>")
            .unwrap();
        assert_ne!(SourceReply::Success(value), reordered);
    }

    #[test]
    fn internal_xml_inspection_retains_the_document_root() {
        let (root, reply) = inspector(128)
            .inspect_xml_with_root(b"<other><status>000</status><value>x</value></other>")
            .unwrap();
        assert_eq!(root, "other");
        assert!(matches!(reply, SourceReply::Success(_)));
        assert!(
            inspector(128)
                .inspect_xml(b"<other><status>000</status><value>x</value></other>")
                .is_ok()
        );
    }

    #[test]
    fn xml_rejects_forbidden_xml_1_0_characters() {
        for body in [
            b"<result>&#0;</result>".as_slice(),
            b"<result>&#x1;</result>".as_slice(),
            b"<result>\0</result>".as_slice(),
            b"<result value=\"&#11;\"/>".as_slice(),
            b"<result><!--\0--></result>".as_slice(),
            b"<result><?note \0?></result>".as_slice(),
            b"<res\0ult/>".as_slice(),
            b"<result val\0ue=\"x\"/>".as_slice(),
        ] {
            assert!(inspector(body.len()).inspect_xml(body).is_err());
        }
    }

    #[test]
    fn xml_ignores_element_only_formatting_whitespace() {
        for body in [
            b"<result>\n  <status>013</status>\n  <message>source</message>\n</result>".as_slice(),
            b"<result><status>013</status><![CDATA[ ]]><message>source</message></result>"
                .as_slice(),
            b"<result><status>013</status>&#32;<message>source</message></result>".as_slice(),
        ] {
            let SourceReply::Status(status) = inspector(body.len()).inspect_xml(body).unwrap()
            else {
                panic!("formatting whitespace must not obscure a status envelope");
            };
            assert_eq!(status.code.as_str(), "013");
        }
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
            b"<a></b>".as_slice(),
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
