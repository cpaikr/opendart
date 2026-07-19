use std::{collections::BTreeMap, fmt};

mod inspect;

pub use inspect::{BodyLimitError, EnvelopeError, EnvelopeFormat, WireInspectError, WireInspector};

/// Repository-owned HTTP protocol versions retained as response evidence.
#[derive(Clone, Debug, Eq, Hash, PartialEq)]
#[non_exhaustive]
pub enum HttpVersion {
    /// HTTP/0.9.
    Http09,
    /// HTTP/1.0.
    Http10,
    /// HTTP/1.1.
    Http11,
    /// HTTP/2.
    Http2,
    /// HTTP/3.
    Http3,
    /// A future or otherwise unrecognized version.
    Other(String),
}

/// One sanitized response header retained without requiring UTF-8.
///
/// Values are constructed only by crate-owned sanitization and never by callers.
#[derive(Clone, Eq, PartialEq)]
pub struct ResponseHeader {
    name: String,
    value: Vec<u8>,
}

impl ResponseHeader {
    #[cfg(any(test, all(feature = "client-reqwest", not(target_family = "wasm"))))]
    pub(crate) fn new(name: impl Into<String>, value: impl Into<Vec<u8>>) -> Self {
        let name = name.into();
        let value = value.into();
        Self { name, value }
    }

    /// Returns the normalized header name.
    #[must_use]
    pub fn name(&self) -> &str {
        &self.name
    }

    /// Returns the sanitized raw header value.
    #[must_use]
    pub fn value(&self) -> &[u8] {
        &self.value
    }
}

impl fmt::Debug for ResponseHeader {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter
            .debug_struct("ResponseHeader")
            .field("name", &self.name)
            .field("value_length", &self.value.len())
            .finish()
    }
}

/// Sanitized source response metadata independent of any HTTP client library.
///
/// The SDK constructs this value only after removing credential-bearing metadata.
#[derive(Clone, Debug, Eq, PartialEq)]
#[non_exhaustive]
pub struct ResponseMetadata {
    status: u16,
    version: HttpVersion,
    headers: Vec<ResponseHeader>,
}

impl ResponseMetadata {
    #[cfg(any(test, all(feature = "client-reqwest", not(target_family = "wasm"))))]
    pub(crate) fn new(status: u16, version: HttpVersion, headers: Vec<ResponseHeader>) -> Self {
        Self {
            status,
            version,
            headers,
        }
    }

    /// Returns the numeric HTTP status code.
    #[must_use]
    pub const fn status(&self) -> u16 {
        self.status
    }

    /// Returns the HTTP protocol version.
    #[must_use]
    pub const fn version(&self) -> &HttpVersion {
        &self.version
    }

    /// Returns the sanitized response headers in observed order.
    #[must_use]
    pub fn headers(&self) -> &[ResponseHeader] {
        &self.headers
    }
}

/// An open OpenDART source-status value that preserves unknown future strings.
#[derive(Clone, Eq, Hash, Ord, PartialEq, PartialOrd)]
pub struct SourceStatus(String);

impl SourceStatus {
    /// The documented success status.
    pub const SUCCESS: &'static str = "000";
    /// The documented no-data status, retained as evidence rather than policy.
    pub const NO_DATA: &'static str = "013";

    /// Creates an open status value without imposing known-code policy.
    #[must_use]
    pub fn new(value: impl Into<String>) -> Self {
        Self(value.into())
    }

    /// Returns the exact source status string.
    #[must_use]
    pub fn as_str(&self) -> &str {
        &self.0
    }

    /// Reports whether this is the documented source success status.
    #[must_use]
    pub fn is_success(&self) -> bool {
        self.0 == Self::SUCCESS
    }
}

impl fmt::Debug for SourceStatus {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter
            .debug_tuple("SourceStatus")
            .field(&self.0)
            .finish()
    }
}

impl fmt::Display for SourceStatus {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.0.fmt(formatter)
    }
}

/// The observable kind of an opaque source value.
#[derive(Clone, Copy, Debug, Eq, Hash, PartialEq)]
#[non_exhaustive]
pub enum SourceValueKind {
    /// A null value.
    Null,
    /// A Boolean value.
    Boolean,
    /// A number retained in source spelling.
    Number,
    /// A string value.
    String,
    /// An ordered array.
    Array,
    /// An object whose unknown fields remain available by name.
    Object,
}

/// A conservative source value that does not expose JSON or XML parser types.
#[derive(Clone, Debug, PartialEq)]
pub struct SourceValue(SourceValueRepr);

#[derive(Clone, Debug, PartialEq)]
enum SourceValueRepr {
    Null,
    Boolean(bool),
    Number(String),
    String(String),
    Array(Vec<SourceValue>),
    Object(BTreeMap<String, SourceValue>),
}

impl SourceValue {
    /// Creates a null source value, primarily for caller-owned fixtures.
    #[must_use]
    pub const fn null() -> Self {
        Self(SourceValueRepr::Null)
    }

    /// Creates a Boolean source value, primarily for caller-owned fixtures.
    #[must_use]
    pub const fn boolean(value: bool) -> Self {
        Self(SourceValueRepr::Boolean(value))
    }

    /// Creates a number from its exact source spelling.
    ///
    /// Bounded wire inspectors validate parser syntax before using this constructor.
    #[must_use]
    pub fn number(value: impl Into<String>) -> Self {
        Self(SourceValueRepr::Number(value.into()))
    }

    /// Creates a string source value, primarily for caller-owned fixtures.
    #[must_use]
    pub fn string(value: impl Into<String>) -> Self {
        Self(SourceValueRepr::String(value.into()))
    }

    /// Creates an ordered array source value, primarily for caller-owned fixtures.
    #[must_use]
    pub fn array(value: Vec<Self>) -> Self {
        Self(SourceValueRepr::Array(value))
    }

    /// Creates an object whose unknown fields remain available by name.
    #[must_use]
    pub fn object(value: BTreeMap<String, Self>) -> Self {
        Self(SourceValueRepr::Object(value))
    }

    /// Returns the observable source value kind.
    #[must_use]
    pub const fn kind(&self) -> SourceValueKind {
        match self.0 {
            SourceValueRepr::Null => SourceValueKind::Null,
            SourceValueRepr::Boolean(_) => SourceValueKind::Boolean,
            SourceValueRepr::Number(_) => SourceValueKind::Number,
            SourceValueRepr::String(_) => SourceValueKind::String,
            SourceValueRepr::Array(_) => SourceValueKind::Array,
            SourceValueRepr::Object(_) => SourceValueKind::Object,
        }
    }

    /// Returns the Boolean value when present.
    #[must_use]
    pub const fn as_bool(&self) -> Option<bool> {
        match self.0 {
            SourceValueRepr::Boolean(value) => Some(value),
            _ => None,
        }
    }

    /// Returns the exact number spelling when present.
    #[must_use]
    pub fn as_number_str(&self) -> Option<&str> {
        match &self.0 {
            SourceValueRepr::Number(value) => Some(value),
            _ => None,
        }
    }

    /// Returns the string value when present.
    #[must_use]
    pub fn as_str(&self) -> Option<&str> {
        match &self.0 {
            SourceValueRepr::String(value) => Some(value),
            _ => None,
        }
    }

    /// Returns the array items when present.
    #[must_use]
    pub fn as_array(&self) -> Option<&[Self]> {
        match &self.0 {
            SourceValueRepr::Array(value) => Some(value),
            _ => None,
        }
    }

    /// Returns an object field while retaining all unknown siblings.
    #[must_use]
    pub fn get(&self, name: &str) -> Option<&Self> {
        match &self.0 {
            SourceValueRepr::Object(value) => value.get(name),
            _ => None,
        }
    }

    /// Iterates object fields in deterministic name order.
    pub fn fields(&self) -> impl Iterator<Item = (&str, &Self)> {
        let value = match &self.0 {
            SourceValueRepr::Object(value) => Some(value),
            _ => None,
        };
        value
            .into_iter()
            .flat_map(|fields| fields.iter().map(|(name, value)| (name.as_str(), value)))
    }

    fn append_repeated(&mut self, value: Self) {
        match &mut self.0 {
            SourceValueRepr::Array(values) => values.push(value),
            _ => {
                let first = std::mem::replace(self, Self::null());
                *self = Self::array(vec![first, value]);
            }
        }
    }
}

/// A source status-only envelope retained without application interpretation.
#[derive(Clone, Debug, PartialEq)]
#[non_exhaustive]
pub struct StatusEnvelope {
    /// The exact known or unknown source status.
    pub code: SourceStatus,
    /// The optional opaque source message.
    pub message: Option<SourceValue>,
}

/// A recognized source reply that separates success payload from status evidence.
#[derive(Clone, Debug, PartialEq)]
#[non_exhaustive]
pub enum SourceReply<T> {
    /// A representation-specific success payload.
    Success(T),
    /// A status-only source envelope, including success-like and unknown codes.
    Status(StatusEnvelope),
}

/// A source reply paired with sanitized HTTP metadata.
#[derive(Clone, Debug, PartialEq)]
#[non_exhaustive]
pub struct SourceResponse<T> {
    /// Sanitized source response metadata.
    pub metadata: ResponseMetadata,
    /// Parsed source evidence or success payload.
    pub reply: T,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn opaque_values_preserve_unknown_fields_and_scalar_kinds() {
        let value = SourceValue::object(BTreeMap::from([
            ("future".to_owned(), SourceValue::boolean(true)),
            (
                "number".to_owned(),
                SourceValue::number("1.20e3".to_owned()),
            ),
            ("nothing".to_owned(), SourceValue::null()),
            (
                "nested".to_owned(),
                SourceValue::array(vec![SourceValue::string("value".to_owned())]),
            ),
        ]));

        assert_eq!(value.kind(), SourceValueKind::Object);
        assert_eq!(
            value.get("future").and_then(SourceValue::as_bool),
            Some(true)
        );
        assert_eq!(
            value.get("number").and_then(SourceValue::as_number_str),
            Some("1.20e3")
        );
        assert_eq!(
            value.get("nothing").map(SourceValue::kind),
            Some(SourceValueKind::Null)
        );
        assert_eq!(value.fields().count(), 4);
    }

    #[test]
    fn metadata_retains_non_utf8_safe_headers() {
        let metadata = ResponseMetadata::new(
            200,
            HttpVersion::Http11,
            vec![ResponseHeader::new("x-source".to_owned(), vec![0x66, 0x80])],
        );
        assert_eq!(metadata.status(), 200);
        assert_eq!(metadata.headers()[0].value(), &[0x66, 0x80]);
    }

    #[test]
    fn unknown_source_status_remains_open() {
        let status = SourceStatus::new("future-status".to_owned());
        assert_eq!(status.as_str(), "future-status");
        assert!(!status.is_success());
    }
}
