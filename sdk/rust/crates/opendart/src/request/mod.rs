use std::{borrow::Cow, fmt};

use form_urlencoded::{Serializer, byte_serialize};
use secrecy::{ExposeSecret, SecretString};
use zeroize::Zeroizing;

use crate::{
    AuthorizationError, BodyLimitError, EnvelopeError, ResponseDecodeError, SourceReply,
    SourceValue, WireInspectError, WireInspector,
};

/// Stable physical and logical identities for one callable OpenDART operation.
#[derive(Clone, Copy, Debug, Eq, Hash, PartialEq)]
pub struct OperationIdentity {
    physical: &'static str,
    logical: &'static str,
}

impl OperationIdentity {
    pub(crate) const fn new(physical: &'static str, logical: &'static str) -> Self {
        Self { physical, logical }
    }

    /// Returns the canonical physical OpenAPI `operationId`.
    #[must_use]
    pub const fn physical(&self) -> &'static str {
        self.physical
    }

    /// Returns the stable logical operation identity.
    #[must_use]
    pub const fn logical(&self) -> &'static str {
        self.logical
    }
}

impl fmt::Display for OperationIdentity {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(formatter, "{} ({})", self.physical, self.logical)
    }
}

/// HTTP methods emitted by the trusted operation inventory.
#[derive(Clone, Copy, Debug, Eq, Hash, PartialEq)]
#[non_exhaustive]
pub enum RequestMethod {
    /// An HTTP GET request.
    Get,
}

/// Source representations supported by physical OpenDART operations.
#[derive(Clone, Copy, Debug, Eq, Hash, PartialEq)]
#[non_exhaustive]
pub enum Representation {
    /// JSON source output.
    Json,
    /// XML source output or source-error envelope.
    Xml,
    /// ZIP entity bytes.
    Zip,
}

/// The credential placement required by a prepared operation.
#[derive(Clone, Copy, Debug, Eq, Hash, PartialEq)]
#[non_exhaustive]
pub enum Authentication {
    /// The OpenDART `crtfc_key` query credential.
    ApiKeyQuery,
}

pub(crate) type ResponseDecoder<T> = fn(SourceValue) -> Result<T, ResponseDecodeError>;

#[derive(Clone, Copy)]
enum StructuredResponseContract {
    Json,
    Xml { expected_root: &'static str },
}

/// Shared immutable request facts hidden behind typed structured and binary plans.
pub(crate) struct RequestParts {
    method: RequestMethod,
    relative_path: &'static str,
    encoded_query: String,
    authentication: Authentication,
    identity: OperationIdentity,
    expected_representations: &'static [Representation],
    expected_xml_root: Option<&'static str>,
    generator_schema: u32,
    projection_identity: &'static str,
}

/// An immutable structured request bound to its generated success payload.
pub struct PreparedRequest<T> {
    parts: RequestParts,
    contract: StructuredResponseContract,
    decoder: ResponseDecoder<T>,
}

/// A transport-independent failure while interpreting one prepared HTTP response.
#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum ResponseInterpretError {
    /// The HTTP response status was outside the successful 2xx range.
    #[error("{operation} received a non-success HTTP status {status}")]
    HttpStatus {
        /// The prepared operation.
        operation: OperationIdentity,
        /// The numeric HTTP response status.
        status: u16,
        /// Normalized bounded body evidence, when recognizable and free of the supplied key.
        evidence: Option<SourceReply<SourceValue>>,
    },
    /// The response body exceeded the configured bound.
    #[error("{operation}: {source}")]
    BodyLimit {
        /// The prepared operation.
        operation: OperationIdentity,
        /// The bounded-body failure.
        source: BodyLimitError,
    },
    /// The bounded response body was malformed.
    #[error("{operation}: {source}")]
    Envelope {
        /// The prepared operation.
        operation: OperationIdentity,
        /// The sanitized envelope failure.
        source: EnvelopeError,
    },
    /// The response violated the selected generated response shape.
    #[error("{operation}: {source}")]
    Decode {
        /// The prepared operation.
        operation: OperationIdentity,
        /// The sanitized generated-shape failure.
        source: ResponseDecodeError,
    },
}

/// An immutable ZIP request whose successful body remains a replaying stream.
pub struct PreparedBinaryRequest {
    parts: RequestParts,
}

pub(crate) enum QueryValue<'a> {
    Scalar(&'a str),
    CommaSeparated(&'a [String]),
}

pub(crate) struct QueryParameter<'a> {
    pub(crate) name: &'static str,
    pub(crate) value: QueryValue<'a>,
}

impl RequestParts {
    pub(crate) fn new(
        relative_path: &'static str,
        identity: OperationIdentity,
        parameters: &[QueryParameter<'_>],
        expected_representations: &'static [Representation],
        expected_xml_root: Option<&'static str>,
        generator_schema: u32,
        projection_identity: &'static str,
    ) -> Self {
        debug_assert!(relative_path.starts_with("/api/"));
        debug_assert!(!relative_path.contains(['?', '#']));
        let encoded_query = parameters
            .iter()
            .map(|parameter| {
                debug_assert!(
                    parameter
                        .name
                        .bytes()
                        .all(|byte| byte.is_ascii_alphanumeric() || byte == b'_')
                );
                let encoded_value = match &parameter.value {
                    QueryValue::Scalar(value) => encode_query_value(value),
                    QueryValue::CommaSeparated(values) => values
                        .iter()
                        .map(|value| encode_query_value(value))
                        .collect::<Vec<_>>()
                        .join(","),
                };
                format!("{}={encoded_value}", parameter.name)
            })
            .collect::<Vec<_>>()
            .join("&");
        Self {
            method: RequestMethod::Get,
            relative_path,
            encoded_query,
            authentication: Authentication::ApiKeyQuery,
            identity,
            expected_representations,
            expected_xml_root,
            generator_schema,
            projection_identity,
        }
    }

    fn debug(&self, name: &str, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter
            .debug_struct(name)
            .field("method", &self.method)
            .field("relative_path", &self.relative_path)
            .field(
                "query_parameter_count",
                &if self.encoded_query.is_empty() {
                    0
                } else {
                    self.encoded_query.split('&').count()
                },
            )
            .field("authentication", &self.authentication)
            .field("identity", &self.identity)
            .field("expected_representations", &self.expected_representations)
            .field("generator_schema", &self.generator_schema)
            .field("projection_identity", &self.projection_identity)
            .finish()
    }

    #[cfg(all(feature = "client-reqwest", not(target_family = "wasm")))]
    pub(crate) const fn identity(&self) -> OperationIdentity {
        self.identity
    }

    #[cfg(all(feature = "client-reqwest", not(target_family = "wasm")))]
    pub(crate) const fn method(&self) -> RequestMethod {
        self.method
    }

    #[cfg(all(feature = "client-reqwest", not(target_family = "wasm")))]
    pub(crate) const fn expected_xml_root(&self) -> Option<&'static str> {
        self.expected_xml_root
    }

    pub(crate) fn authorize<'a>(&'a self, api_key: &'a ApiKey) -> AuthorizedRequest<'a> {
        AuthorizedRequest {
            prepared: self,
            api_key,
        }
    }
}

impl<T> PreparedRequest<T> {
    pub(crate) fn new(parts: RequestParts, decoder: ResponseDecoder<T>) -> Self {
        let contract = if parts.expected_representations == [Representation::Json] {
            StructuredResponseContract::Json
        } else if parts.expected_representations == [Representation::Xml] {
            StructuredResponseContract::Xml {
                expected_root: parts
                    .expected_xml_root
                    .expect("generated XML requests carry their expected root"),
            }
        } else {
            unreachable!("generated typed requests are JSON or XML")
        };
        Self {
            parts,
            contract,
            decoder,
        }
    }

    pub(crate) fn decode(&self, value: SourceValue) -> Result<T, ResponseDecodeError> {
        (self.decoder)(value)
    }

    #[cfg(all(feature = "client-reqwest", not(target_family = "wasm")))]
    pub(crate) const fn parts(&self) -> &RequestParts {
        &self.parts
    }

    /// Interprets one bounded HTTP response using this generated operation contract.
    ///
    /// The caller owns HTTP execution and must bound body collection while reading.
    /// This method defensively rechecks the supplied bytes, selects JSON or XML from
    /// generated facts, validates XML roots, removes evidence reflecting `api_key`,
    /// preserves other provider-status evidence, and decodes successful payloads
    /// without depending on an HTTP client or async runtime.
    ///
    /// # Errors
    ///
    /// Returns [`ResponseInterpretError::HttpStatus`] for every non-2xx status,
    /// retaining normalized body evidence only when it does not expose `api_key`.
    /// Successful HTTP responses can instead fail body bounds, envelope validation,
    /// XML-root validation, or typed generated decoding.
    pub fn interpret_response(
        &self,
        inspector: &WireInspector,
        api_key: &ApiKey,
        http_status: u16,
        body: &[u8],
    ) -> Result<SourceReply<T>, ResponseInterpretError> {
        let raw = self.inspect_response(inspector, api_key, http_status, body)?;
        match raw {
            SourceReply::Success(value) => {
                self.decode(value)
                    .map(SourceReply::Success)
                    .map_err(|source| ResponseInterpretError::Decode {
                        operation: self.identity(),
                        source,
                    })
            }
            SourceReply::Status(status) => Ok(SourceReply::Status(status)),
        }
    }

    pub(crate) fn inspect_response(
        &self,
        inspector: &WireInspector,
        api_key: &ApiKey,
        http_status: u16,
        body: &[u8],
    ) -> Result<SourceReply<SourceValue>, ResponseInterpretError> {
        let inspected = self.inspect_body(inspector, body);
        if !(200..=299).contains(&http_status) {
            return Err(ResponseInterpretError::HttpStatus {
                operation: self.identity(),
                status: http_status,
                evidence: inspected
                    .ok()
                    .filter(|evidence| api_key.response_evidence_is_safe(evidence)),
            });
        }
        inspected
    }

    fn inspect_body(
        &self,
        inspector: &WireInspector,
        body: &[u8],
    ) -> Result<SourceReply<SourceValue>, ResponseInterpretError> {
        let operation = self.identity();
        let inspected = match self.contract {
            StructuredResponseContract::Json => inspector
                .inspect_json(body)
                .map_err(|error| map_wire_error(operation, error))?,
            StructuredResponseContract::Xml { expected_root } => {
                let (root, reply) = inspector
                    .inspect_xml_with_root(body)
                    .map_err(|error| map_wire_error(operation, error))?;
                if root != expected_root {
                    return Err(ResponseInterpretError::Decode {
                        operation,
                        source: ResponseDecodeError::UnexpectedXmlRoot {
                            expected: expected_root,
                        },
                    });
                }
                reply
            }
        };
        Ok(inspected)
    }

    /// Returns the HTTP method.
    #[must_use]
    pub const fn method(&self) -> RequestMethod {
        self.parts.method
    }

    /// Returns the trusted credential-free relative path.
    #[must_use]
    pub const fn relative_path(&self) -> &'static str {
        self.parts.relative_path
    }

    /// Returns the deterministically encoded, credential-free query string.
    #[must_use]
    pub fn encoded_query(&self) -> &str {
        &self.parts.encoded_query
    }

    /// Returns the required credential placement.
    #[must_use]
    pub const fn authentication(&self) -> Authentication {
        self.parts.authentication
    }

    /// Returns the stable operation identity.
    #[must_use]
    pub const fn identity(&self) -> OperationIdentity {
        self.parts.identity
    }

    /// Returns the representations expected from this physical operation.
    #[must_use]
    pub const fn expected_representations(&self) -> &'static [Representation] {
        self.parts.expected_representations
    }

    /// Returns the SDK generator schema version used to prepare this request.
    #[must_use]
    pub const fn generator_schema(&self) -> u32 {
        self.parts.generator_schema
    }

    /// Returns the SDK projection identity used for safe diagnostics.
    #[must_use]
    pub const fn projection_identity(&self) -> &'static str {
        self.parts.projection_identity
    }

    /// Adds the API credential at the explicit adapter boundary.
    #[must_use]
    pub fn authorize<'a>(&'a self, api_key: &'a ApiKey) -> AuthorizedRequest<'a> {
        self.parts.authorize(api_key)
    }
}

fn map_wire_error(operation: OperationIdentity, error: WireInspectError) -> ResponseInterpretError {
    match error {
        WireInspectError::BodyLimit(source) => {
            ResponseInterpretError::BodyLimit { operation, source }
        }
        WireInspectError::Envelope(source) => {
            ResponseInterpretError::Envelope { operation, source }
        }
    }
}

impl PreparedBinaryRequest {
    pub(crate) const fn new(parts: RequestParts) -> Self {
        Self { parts }
    }

    #[cfg(all(feature = "client-reqwest", not(target_family = "wasm")))]
    pub(crate) const fn parts(&self) -> &RequestParts {
        &self.parts
    }

    /// Returns the HTTP method.
    #[must_use]
    pub const fn method(&self) -> RequestMethod {
        self.parts.method
    }

    /// Returns the trusted credential-free relative path.
    #[must_use]
    pub const fn relative_path(&self) -> &'static str {
        self.parts.relative_path
    }

    /// Returns the deterministically encoded, credential-free query string.
    #[must_use]
    pub fn encoded_query(&self) -> &str {
        &self.parts.encoded_query
    }

    /// Returns the required credential placement.
    #[must_use]
    pub const fn authentication(&self) -> Authentication {
        self.parts.authentication
    }

    /// Returns the stable operation identity.
    #[must_use]
    pub const fn identity(&self) -> OperationIdentity {
        self.parts.identity
    }

    /// Returns the representations expected from this physical operation.
    #[must_use]
    pub const fn expected_representations(&self) -> &'static [Representation] {
        self.parts.expected_representations
    }

    /// Returns the SDK generator schema version used to prepare this request.
    #[must_use]
    pub const fn generator_schema(&self) -> u32 {
        self.parts.generator_schema
    }

    /// Returns the SDK projection identity used for safe diagnostics.
    #[must_use]
    pub const fn projection_identity(&self) -> &'static str {
        self.parts.projection_identity
    }

    /// Adds the API credential at the explicit adapter boundary.
    #[must_use]
    pub fn authorize<'a>(&'a self, api_key: &'a ApiKey) -> AuthorizedRequest<'a> {
        self.parts.authorize(api_key)
    }
}

impl<T> fmt::Debug for PreparedRequest<T> {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.parts.debug("PreparedRequest", formatter)
    }
}

impl fmt::Debug for PreparedBinaryRequest {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.parts.debug("PreparedBinaryRequest", formatter)
    }
}

/// An owned OpenDART API credential with redacted diagnostics and zeroizing drop.
pub struct ApiKey {
    secret: SecretString,
}

impl ApiKey {
    /// Validates and owns an API key without exposing it through formatting.
    ///
    /// # Errors
    ///
    /// Returns [`AuthorizationError::EmptyApiKey`] for an empty or whitespace-only
    /// value. Returns [`AuthorizationError::ControlCharacterApiKey`] when a value
    /// that is not whitespace-only contains any control character.
    pub fn new(value: impl Into<String>) -> Result<Self, AuthorizationError> {
        let value = value.into();
        if value.chars().all(char::is_whitespace) {
            return Err(AuthorizationError::EmptyApiKey);
        }
        if value.chars().any(char::is_control) {
            return Err(AuthorizationError::ControlCharacterApiKey);
        }
        Ok(Self {
            secret: value.into(),
        })
    }

    pub(crate) fn with_exposed_secret<T>(&self, adapter: impl FnOnce(&str) -> T) -> T {
        adapter(self.secret.expose_secret())
    }

    pub(crate) fn response_evidence_is_safe(&self, evidence: &SourceReply<SourceValue>) -> bool {
        let value = match evidence {
            SourceReply::Success(value) => value,
            SourceReply::Status(status) => &status.evidence,
        };
        self.with_exposed_secret(|secret| {
            let form_encoded = byte_serialize(secret.as_bytes()).collect::<String>();
            let percent_encoded = percent_encode(secret.as_bytes());
            source_value_is_safe(
                value,
                secret.as_bytes(),
                form_encoded.as_bytes(),
                percent_encoded.as_bytes(),
            )
        })
    }

    #[cfg(all(feature = "client-reqwest", not(target_family = "wasm")))]
    pub(crate) fn response_bytes_are_safe(&self, value: &[u8]) -> bool {
        self.with_exposed_secret(|secret| {
            let form_encoded = byte_serialize(secret.as_bytes()).collect::<String>();
            let percent_encoded = percent_encode(secret.as_bytes());
            response_bytes_are_safe(
                value,
                secret.as_bytes(),
                form_encoded.as_bytes(),
                percent_encoded.as_bytes(),
            )
        })
    }
}

impl fmt::Debug for ApiKey {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter.write_str("ApiKey([REDACTED])")
    }
}

/// A non-cloneable request whose relative URI contains an API credential.
pub struct AuthorizedRequest<'a> {
    prepared: &'a RequestParts,
    api_key: &'a ApiKey,
}

impl AuthorizedRequest<'_> {
    /// Returns the HTTP method without exposing the credential-bearing target.
    #[must_use]
    pub const fn method(&self) -> RequestMethod {
        self.prepared.method
    }

    /// Returns the stable operation identity.
    #[must_use]
    pub const fn identity(&self) -> OperationIdentity {
        self.prepared.identity
    }

    /// Returns the expected response representations.
    #[must_use]
    pub const fn expected_representations(&self) -> &'static [Representation] {
        self.prepared.expected_representations
    }

    /// Exposes the credential-bearing relative URI for one consuming adapter call.
    ///
    /// The callback must treat the argument as secret and must not log, persist, or include it
    /// in an error. Callers own all execution policy after crossing this boundary. To execute a
    /// separate attempt, authorize the credential-free [`PreparedRequest`] again.
    ///
    /// ```compile_fail
    /// # use opendart::{ApiKey, operations::Company};
    /// # let prepared = Company::new("00126380").prepare_json()?;
    /// # let key = ApiKey::new("example-key")?;
    /// let authorized = prepared.authorize(&key);
    /// authorized.with_exposed_relative_uri(|_| ());
    /// authorized.with_exposed_relative_uri(|_| ()); // consumed by the first adapter call
    /// # Ok::<(), Box<dyn std::error::Error>>(())
    /// ```
    pub fn with_exposed_relative_uri<T>(self, adapter: impl FnOnce(&str) -> T) -> T {
        let exposed = self.api_key.secret.expose_secret();
        let credential_capacity = "crtfc_key="
            .len()
            .saturating_add(exposed.len().saturating_mul(3));
        let mut serializer = Serializer::new(String::with_capacity(credential_capacity));
        serializer.append_pair("crtfc_key", exposed);
        let credential_query = Zeroizing::new(serializer.finish());
        let separator_capacity = usize::from(!self.prepared.encoded_query.is_empty());
        let relative_capacity = self
            .prepared
            .relative_path
            .len()
            .saturating_add(1)
            .saturating_add(self.prepared.encoded_query.len())
            .saturating_add(separator_capacity)
            .saturating_add(credential_query.len());
        let mut relative_uri = Zeroizing::new(String::with_capacity(relative_capacity));
        relative_uri.push_str(self.prepared.relative_path);
        relative_uri.push('?');
        if !self.prepared.encoded_query.is_empty() {
            relative_uri.push_str(&self.prepared.encoded_query);
            relative_uri.push('&');
        }
        relative_uri.push_str(credential_query.as_str());
        adapter(relative_uri.as_str())
    }
}

impl fmt::Debug for AuthorizedRequest<'_> {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter
            .debug_struct("AuthorizedRequest")
            .field("method", &self.prepared.method)
            .field("identity", &self.prepared.identity)
            .field(
                "expected_representations",
                &self.prepared.expected_representations,
            )
            .field("relative_uri", &"[REDACTED]")
            .finish()
    }
}

const MAX_PERCENT_DECODING_PASSES: usize = 3;

fn source_value_is_safe(
    value: &SourceValue,
    secret: &[u8],
    form_encoded_secret: &[u8],
    percent_encoded_secret: &[u8],
) -> bool {
    let safe = |value: &[u8]| {
        response_bytes_are_safe(value, secret, form_encoded_secret, percent_encoded_secret)
    };
    if value.as_str().is_some_and(|value| !safe(value.as_bytes()))
        || value
            .as_number_str()
            .is_some_and(|value| !safe(value.as_bytes()))
    {
        return false;
    }
    if value.as_array().is_some_and(|values| {
        !values.iter().all(|value| {
            source_value_is_safe(value, secret, form_encoded_secret, percent_encoded_secret)
        })
    }) {
        return false;
    }
    value.fields().all(|(name, value)| {
        safe(name.as_bytes())
            && source_value_is_safe(value, secret, form_encoded_secret, percent_encoded_secret)
    })
}

fn response_bytes_are_safe(
    value: &[u8],
    secret: &[u8],
    form_encoded_secret: &[u8],
    percent_encoded_secret: &[u8],
) -> bool {
    let mut stage = Cow::Borrowed(value);
    if stage_contains_sensitive_value(
        stage.as_ref(),
        secret,
        form_encoded_secret,
        percent_encoded_secret,
    ) {
        return false;
    }
    for _ in 0..MAX_PERCENT_DECODING_PASSES {
        let Some(decoded) = (match percent_decode(stage.as_ref()) {
            Ok(decoded) => decoded,
            Err(()) => return false,
        }) else {
            return true;
        };
        let Ok(text) = std::str::from_utf8(&decoded) else {
            return false;
        };
        if text.chars().any(char::is_control)
            || stage_contains_sensitive_value(
                &decoded,
                secret,
                form_encoded_secret,
                percent_encoded_secret,
            )
        {
            return false;
        }
        stage = Cow::Owned(decoded);
    }
    !stage.contains(&b'%')
}

fn stage_contains_sensitive_value(
    value: &[u8],
    secret: &[u8],
    form_encoded_secret: &[u8],
    percent_encoded_secret: &[u8],
) -> bool {
    contains_bytes(value, secret)
        || contains_secret_after_partial_form_decoding(value, secret)
        || contains_ascii_case_insensitive(value, form_encoded_secret)
        || contains_ascii_case_insensitive(value, percent_encoded_secret)
        || contains_ascii_case_insensitive(value, b"crtfc_key")
}

fn contains_secret_after_partial_form_decoding(value: &[u8], secret: &[u8]) -> bool {
    !secret.is_empty()
        && value.windows(secret.len()).any(|window| {
            window
                .iter()
                .zip(secret)
                .all(|(value, secret)| value == secret || (*value == b'+' && *secret == b' '))
        })
}

fn percent_decode(value: &[u8]) -> Result<Option<Vec<u8>>, ()> {
    let Some(first_escape) = value.iter().position(|byte| *byte == b'%') else {
        return Ok(None);
    };
    let mut decoded = Vec::with_capacity(value.len());
    decoded.extend_from_slice(&value[..first_escape]);
    let mut index = first_escape;
    while index < value.len() {
        if value[index] != b'%' {
            decoded.push(value[index]);
            index += 1;
            continue;
        }
        let high = value.get(index + 1).and_then(|byte| hex_value(*byte));
        let low = value.get(index + 2).and_then(|byte| hex_value(*byte));
        let (Some(high), Some(low)) = (high, low) else {
            return Err(());
        };
        decoded.push((high << 4) | low);
        index += 3;
    }
    Ok(Some(decoded))
}

fn hex_value(value: u8) -> Option<u8> {
    match value {
        b'0'..=b'9' => Some(value - b'0'),
        b'a'..=b'f' => Some(value - b'a' + 10),
        b'A'..=b'F' => Some(value - b'A' + 10),
        _ => None,
    }
}

fn percent_encode(value: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789ABCDEF";
    let mut encoded = String::with_capacity(value.len().saturating_mul(3));
    for byte in value {
        if byte.is_ascii_alphanumeric() || matches!(byte, b'-' | b'.' | b'_' | b'~') {
            encoded.push(char::from(*byte));
        } else {
            encoded.push('%');
            encoded.push(char::from(HEX[usize::from(byte >> 4)]));
            encoded.push(char::from(HEX[usize::from(byte & 0x0f)]));
        }
    }
    encoded
}

fn contains_bytes(haystack: &[u8], needle: &[u8]) -> bool {
    !needle.is_empty()
        && haystack
            .windows(needle.len())
            .any(|window| window == needle)
}

fn contains_ascii_case_insensitive(haystack: &[u8], needle: &[u8]) -> bool {
    !needle.is_empty()
        && haystack
            .windows(needle.len())
            .any(|window| window.eq_ignore_ascii_case(needle))
}

fn encode_query_value(value: &str) -> String {
    byte_serialize(value.as_bytes()).collect()
}
