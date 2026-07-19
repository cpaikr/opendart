use std::{
    collections::VecDeque,
    fmt,
    future::{Future, poll_fn},
    pin::Pin,
    task::{Context, Poll},
    time::Duration,
};

use bytes::Bytes;
use futures_core::Stream;
use zeroize::Zeroizing;

use crate::{
    ApiKey, BodyLimitError, EnvelopeError, EnvelopeFormat, HttpVersion, OperationIdentity,
    PreparedRequest, Representation, RequestMethod, ResponseHeader, ResponseMetadata, SourceReply,
    SourceResponse, SourceValue, StatusEnvelope, WireInspectError, WireInspector,
};

const DEFAULT_CONNECT_TIMEOUT: Duration = Duration::from_secs(10);
const DEFAULT_READ_TIMEOUT: Duration = Duration::from_secs(30);
const DEFAULT_TOTAL_TIMEOUT: Duration = Duration::from_secs(60);
const DEFAULT_ENVELOPE_LIMIT: usize = 1024 * 1024;
const PRODUCTION_ORIGIN: &str = "https://opendart.fss.or.kr";

/// A safe-default OpenDART HTTP client.
pub struct Client {
    http: reqwest::Client,
    api_key: ApiKey,
    inspector: WireInspector,
    origin: String,
    total_timeout: Duration,
}

impl Client {
    /// Starts configuring a client that owns the supplied API credential.
    #[must_use]
    pub fn builder(api_key: ApiKey) -> ClientBuilder {
        ClientBuilder::new(api_key)
    }

    /// Executes a prepared JSON or XML request and inspects its bounded envelope.
    pub async fn execute(
        &self,
        prepared: &PreparedRequest,
    ) -> Result<SourceResponse<SourceReply<SourceValue>>, ClientError> {
        let format = structured_format(prepared)?;
        let operation = prepared.identity();
        let sent = self.send(prepared).await?;
        let deadline = sent.deadline;
        let mut response = sent.response;
        let metadata = response_metadata(&response, &self.api_key);
        let mut body = Vec::new();

        loop {
            let chunk = tokio::time::timeout_at(deadline, response.chunk())
                .await
                .map_err(|_| timeout_error(operation, Some(metadata.clone())))?
                .map_err(|error| transport_error(operation, Some(metadata.clone()), &error))?;
            let Some(chunk) = chunk else { break };
            if body.len().saturating_add(chunk.len()) > self.inspector.max_envelope_bytes() {
                return Err(ClientError::BodyLimit {
                    operation,
                    metadata,
                    source: BodyLimitError::new(self.inspector.max_envelope_bytes()),
                });
            }
            body.extend_from_slice(&chunk);
        }

        let reply = match format {
            EnvelopeFormat::Json => self.inspector.inspect_json(&body),
            EnvelopeFormat::Xml => self.inspector.inspect_xml(&body),
        }
        .map_err(|error| map_inspect_error(operation, metadata.clone(), error))?;
        Ok(SourceResponse { metadata, reply })
    }

    /// Executes a prepared ZIP-with-XML-error request without losing consumed bytes.
    pub async fn execute_binary(
        &self,
        prepared: &PreparedRequest,
    ) -> Result<SourceResponse<BinaryReply<BodyStream>>, ClientError> {
        if !prepared
            .expected_representations()
            .contains(&Representation::Zip)
        {
            return Err(ClientError::Representation {
                operation: prepared.identity(),
                expected: "a ZIP response with an alternate XML status envelope",
            });
        }

        let sent = self.send(prepared).await?;
        let metadata = response_metadata(&sent.response, &self.api_key);
        let operation = prepared.identity();
        let stream = ReqwestBodyStream {
            inner: Box::pin(sent.response.bytes_stream()),
            deadline: Box::pin(tokio::time::sleep_until(sent.deadline)),
            terminal: false,
        };
        let raw: RawStream = Box::pin(stream);
        let reply = classify_binary(raw, self.inspector).await;
        Ok(SourceResponse {
            metadata,
            reply: reply.map_err_operation(operation),
        })
    }

    async fn send(&self, prepared: &PreparedRequest) -> Result<SentResponse, ClientError> {
        let operation = prepared.identity();
        let deadline = tokio::time::Instant::now()
            .checked_add(self.total_timeout)
            .ok_or_else(|| timeout_configuration_error(operation))?;
        let authorized = prepared.authorize(&self.api_key);
        let request = authorized.with_exposed_relative_uri(|relative_uri| {
            let target = Zeroizing::new(format!("{}{relative_uri}", self.origin));
            debug_assert_eq!(prepared.method(), RequestMethod::Get);
            self.http.get(target.as_str())
        });
        let response = request
            .send()
            .await
            .map_err(|error| transport_error(operation, None, &error))?;
        Ok(SentResponse { response, deadline })
    }

    #[cfg(opendart_compat)]
    pub(crate) fn compatibility_http_client(&self) -> reqwest::Client {
        self.http.clone()
    }
}

impl fmt::Debug for Client {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter
            .debug_struct("Client")
            .field("api_key", &self.api_key)
            .field("envelope_limit", &self.inspector.max_envelope_bytes())
            .finish_non_exhaustive()
    }
}

/// Configuration for the safe-default HTTP client.
pub struct ClientBuilder {
    api_key: ApiKey,
    connect_timeout: Duration,
    read_timeout: Duration,
    total_timeout: Duration,
    envelope_limit: usize,
    user_agent_suffix: Option<String>,
    origin: String,
    https_only: bool,
    #[cfg(any(test, opendart_compat))]
    http2_prior_knowledge: bool,
}

impl ClientBuilder {
    fn new(api_key: ApiKey) -> Self {
        Self {
            api_key,
            connect_timeout: DEFAULT_CONNECT_TIMEOUT,
            read_timeout: DEFAULT_READ_TIMEOUT,
            total_timeout: DEFAULT_TOTAL_TIMEOUT,
            envelope_limit: DEFAULT_ENVELOPE_LIMIT,
            user_agent_suffix: None,
            origin: PRODUCTION_ORIGIN.to_owned(),
            https_only: true,
            #[cfg(any(test, opendart_compat))]
            http2_prior_knowledge: false,
        }
    }

    /// Sets the connection-establishment timeout.
    #[must_use]
    pub fn connect_timeout(mut self, timeout: Duration) -> Self {
        self.connect_timeout = timeout;
        self
    }

    /// Sets the maximum duration between successful body reads.
    #[must_use]
    pub fn read_timeout(mut self, timeout: Duration) -> Self {
        self.read_timeout = timeout;
        self
    }

    /// Sets the total request and response-body deadline.
    #[must_use]
    pub fn total_timeout(mut self, timeout: Duration) -> Self {
        self.total_timeout = timeout;
        self
    }

    /// Sets the inclusive JSON, XML, and alternate-envelope byte limit.
    #[must_use]
    pub fn envelope_limit(mut self, bytes: usize) -> Self {
        self.envelope_limit = bytes;
        self
    }

    /// Appends a visible ASCII application token to the SDK user agent.
    #[must_use]
    pub fn user_agent_suffix(mut self, suffix: impl Into<String>) -> Self {
        self.user_agent_suffix = Some(suffix.into());
        self
    }

    /// Validates the configuration and constructs the only official HTTP adapter.
    pub fn build(self) -> Result<Client, ClientBuildError> {
        validate_timeout(self.connect_timeout, "connect timeout")?;
        validate_timeout(self.read_timeout, "read timeout")?;
        validate_timeout(self.total_timeout, "total timeout")?;
        if self.envelope_limit == 0 {
            return Err(ClientBuildError::InvalidConfiguration {
                setting: "envelope limit",
            });
        }
        let user_agent = user_agent(self.user_agent_suffix.as_deref())?;
        let inspector = WireInspector::new(self.envelope_limit).expect("validated nonzero limit");
        let factory = FactoryConfig {
            connect_timeout: self.connect_timeout,
            read_timeout: self.read_timeout,
            total_timeout: self.total_timeout,
            user_agent,
            https_only: self.https_only,
            #[cfg(any(test, opendart_compat))]
            http2_prior_knowledge: self.http2_prior_knowledge,
        };
        let http = build_http_client(&factory).map_err(|_| ClientBuildError::TransportSetup)?;
        Ok(Client {
            http,
            api_key: self.api_key,
            inspector,
            origin: self.origin,
            total_timeout: self.total_timeout,
        })
    }

    #[cfg(test)]
    fn test_origin(mut self, origin: String) -> Self {
        self.origin = origin;
        self.https_only = false;
        self
    }

    #[cfg(test)]
    fn test_h2_origin(mut self, origin: String) -> Self {
        self.origin = origin;
        self.https_only = false;
        self.http2_prior_knowledge = true;
        self
    }

    #[cfg(opendart_compat)]
    pub(crate) fn compatibility_origin(mut self, origin: String) -> Self {
        self.origin = origin;
        self.https_only = false;
        self
    }
}

fn validate_timeout(value: Duration, setting: &'static str) -> Result<(), ClientBuildError> {
    if value.is_zero() || std::time::Instant::now().checked_add(value).is_none() {
        Err(ClientBuildError::InvalidConfiguration { setting })
    } else {
        Ok(())
    }
}

fn user_agent(suffix: Option<&str>) -> Result<String, ClientBuildError> {
    let base = concat!("opendart-rust/", env!("CARGO_PKG_VERSION"));
    match suffix {
        None => Ok(base.to_owned()),
        Some(suffix)
            if !suffix.is_empty() && suffix.bytes().all(|byte| byte.is_ascii_graphic()) =>
        {
            Ok(format!("{base} {suffix}"))
        }
        Some(_) => Err(ClientBuildError::InvalidConfiguration {
            setting: "user-agent suffix",
        }),
    }
}

struct FactoryConfig {
    connect_timeout: Duration,
    read_timeout: Duration,
    total_timeout: Duration,
    user_agent: String,
    https_only: bool,
    #[cfg(any(test, opendart_compat))]
    http2_prior_knowledge: bool,
}

struct SentResponse {
    response: reqwest::Response,
    deadline: tokio::time::Instant,
}

fn build_http_client(config: &FactoryConfig) -> Result<reqwest::Client, reqwest::Error> {
    let builder = reqwest::Client::builder()
        .retry(reqwest::retry::never())
        .redirect(reqwest::redirect::Policy::none())
        .no_proxy()
        .tls_backend_rustls()
        .no_hickory_dns()
        .no_gzip()
        .no_brotli()
        .no_zstd()
        .no_deflate()
        .referer(false)
        .https_only(config.https_only)
        .connect_timeout(config.connect_timeout)
        .read_timeout(config.read_timeout)
        .timeout(config.total_timeout)
        .user_agent(&config.user_agent);
    #[cfg(any(test, opendart_compat))]
    let builder = if config.http2_prior_knowledge {
        builder.http2_prior_knowledge()
    } else {
        builder
    };
    builder.build()
}

/// A client configuration could not preserve the safe-default contract.
#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum ClientBuildError {
    /// A configurable bound was zero or otherwise invalid.
    #[error("invalid safe-client configuration for {setting}")]
    InvalidConfiguration {
        /// The rejected setting, without its potentially sensitive value.
        setting: &'static str,
    },
    /// The private HTTP adapter could not be initialized.
    #[error("the safe HTTP adapter could not be initialized")]
    TransportSetup,
}

/// A sanitized class of HTTP transport failure.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
#[non_exhaustive]
pub enum TransportFailureKind {
    /// A configured deadline expired.
    Timeout,
    /// A connection could not be established or retained.
    Connection,
    /// Entity-body delivery failed after response headers.
    Body,
    /// TLS or HTTP protocol processing failed.
    Protocol,
    /// A future transport failure class.
    Other,
}

/// A sanitized transport failure with optional retained response metadata.
#[derive(Debug, thiserror::Error)]
#[error("{operation} failed during HTTP transport ({kind:?})")]
pub struct TransportError {
    operation: OperationIdentity,
    kind: TransportFailureKind,
    metadata: Option<ResponseMetadata>,
}

impl TransportError {
    /// Returns the operation whose single attempt failed.
    #[must_use]
    pub const fn operation(&self) -> OperationIdentity {
        self.operation
    }

    /// Returns the sanitized failure class without retry advice.
    #[must_use]
    pub const fn kind(&self) -> TransportFailureKind {
        self.kind
    }

    /// Returns response metadata when headers arrived before the failure.
    #[must_use]
    pub const fn metadata(&self) -> Option<&ResponseMetadata> {
        self.metadata.as_ref()
    }
}

/// A safe-client execution failure.
#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum ClientError {
    /// The prepared representation belongs to the other execution method.
    #[error("{operation} requires {expected}")]
    Representation {
        /// The prepared operation.
        operation: OperationIdentity,
        /// The expected execution shape.
        expected: &'static str,
    },
    /// The HTTP adapter failed without exposing its credential-bearing URL.
    #[error(transparent)]
    Transport(#[from] TransportError),
    /// A structured body exceeded the configured bound.
    #[error("{operation}: {source}")]
    BodyLimit {
        /// The prepared operation.
        operation: OperationIdentity,
        /// Metadata retained before bounded buffering.
        metadata: ResponseMetadata,
        /// The bounded-body failure.
        source: BodyLimitError,
    },
    /// A bounded structured body was malformed.
    #[error("{operation}: {source}")]
    Envelope {
        /// The prepared operation.
        operation: OperationIdentity,
        /// Metadata retained before envelope inspection.
        metadata: ResponseMetadata,
        /// The sanitized envelope failure.
        source: EnvelopeError,
    },
}

impl ClientError {
    /// Returns response metadata when it was observed before the failure.
    #[must_use]
    pub const fn metadata(&self) -> Option<&ResponseMetadata> {
        match self {
            Self::Representation { .. } => None,
            Self::Transport(error) => error.metadata(),
            Self::BodyLimit { metadata, .. } | Self::Envelope { metadata, .. } => Some(metadata),
        }
    }
}

fn transport_error(
    operation: OperationIdentity,
    metadata: Option<ResponseMetadata>,
    error: &reqwest::Error,
) -> ClientError {
    let kind = if error.is_timeout() {
        TransportFailureKind::Timeout
    } else if error.is_connect() {
        TransportFailureKind::Connection
    } else if error.is_body() || error.is_decode() {
        TransportFailureKind::Body
    } else if error.is_builder() || error.is_request() {
        TransportFailureKind::Protocol
    } else {
        TransportFailureKind::Other
    };
    TransportError {
        operation,
        kind,
        metadata,
    }
    .into()
}

fn timeout_error(operation: OperationIdentity, metadata: Option<ResponseMetadata>) -> ClientError {
    TransportError {
        operation,
        kind: TransportFailureKind::Timeout,
        metadata,
    }
    .into()
}

fn timeout_configuration_error(operation: OperationIdentity) -> ClientError {
    TransportError {
        operation,
        kind: TransportFailureKind::Other,
        metadata: None,
    }
    .into()
}

fn map_inspect_error(
    operation: OperationIdentity,
    metadata: ResponseMetadata,
    error: WireInspectError,
) -> ClientError {
    match error {
        WireInspectError::BodyLimit(source) => ClientError::BodyLimit {
            operation,
            metadata,
            source,
        },
        WireInspectError::Envelope(source) => ClientError::Envelope {
            operation,
            metadata,
            source,
        },
    }
}

fn structured_format(prepared: &PreparedRequest) -> Result<EnvelopeFormat, ClientError> {
    let representations = prepared.expected_representations();
    if representations.contains(&Representation::Json) {
        Ok(EnvelopeFormat::Json)
    } else if representations == [Representation::Xml] {
        Ok(EnvelopeFormat::Xml)
    } else {
        Err(ClientError::Representation {
            operation: prepared.identity(),
            expected: "a bounded JSON or XML execution",
        })
    }
}

fn response_metadata(response: &reqwest::Response, api_key: &ApiKey) -> ResponseMetadata {
    let status = response.status().as_u16();
    let version = match response.version() {
        reqwest::Version::HTTP_09 => HttpVersion::Http09,
        reqwest::Version::HTTP_10 => HttpVersion::Http10,
        reqwest::Version::HTTP_11 => HttpVersion::Http11,
        reqwest::Version::HTTP_2 => HttpVersion::Http2,
        reqwest::Version::HTTP_3 => HttpVersion::Http3,
        other => HttpVersion::Other(format!("{other:?}")),
    };
    let headers = api_key.with_exposed_secret(|secret| {
        let form_encoded = form_urlencoded::byte_serialize(secret.as_bytes()).collect::<String>();
        let percent_encoded = percent_encode(secret.as_bytes());
        response
            .headers()
            .iter()
            .filter_map(|(name, value)| {
                let bytes = value.as_bytes();
                if contains_bytes(bytes, secret.as_bytes())
                    || contains_ascii_case_insensitive(bytes, form_encoded.as_bytes())
                    || contains_ascii_case_insensitive(bytes, percent_encoded.as_bytes())
                    || contains_ascii_case_insensitive(bytes, b"crtfc_key")
                {
                    None
                } else {
                    Some(ResponseHeader::new(name.as_str(), bytes))
                }
            })
            .collect()
    });
    ResponseMetadata::new(status, version, headers)
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

/// The classified evidence from a ZIP-with-XML-error endpoint.
#[derive(Debug)]
#[non_exhaustive]
pub enum BinaryReply<S> {
    /// Positive evidence of a normal or empty ZIP archive signature.
    Archive(S),
    /// A bounded status-only XML envelope.
    Status(StatusEnvelope),
    /// A body that did not satisfy either positive discriminator.
    Unrecognized(S),
}

impl BinaryReply<BodyStream> {
    fn map_err_operation(self, operation: OperationIdentity) -> Self {
        match self {
            Self::Archive(mut stream) => {
                stream.operation = operation;
                Self::Archive(stream)
            }
            Self::Status(status) => Self::Status(status),
            Self::Unrecognized(mut stream) => {
                stream.operation = operation;
                Self::Unrecognized(stream)
            }
        }
    }
}

/// One exact entity-body chunk from a binary response.
#[derive(Clone, Eq, PartialEq)]
pub struct BodyChunk(Bytes);

impl BodyChunk {
    /// Returns the chunk bytes without conversion.
    #[must_use]
    pub fn as_bytes(&self) -> &[u8] {
        &self.0
    }

    /// Returns the chunk length.
    #[must_use]
    pub fn len(&self) -> usize {
        self.0.len()
    }

    /// Reports whether this chunk contains no bytes.
    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.0.is_empty()
    }
}

impl fmt::Debug for BodyChunk {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter
            .debug_struct("BodyChunk")
            .field("length", &self.len())
            .finish()
    }
}

/// A sanitized terminal binary-body stream failure.
#[derive(Clone, Debug, thiserror::Error)]
#[error("{operation} binary body stream failed ({kind:?})")]
pub struct BodyStreamError {
    operation: OperationIdentity,
    kind: TransportFailureKind,
}

impl BodyStreamError {
    /// Returns the operation whose body failed.
    #[must_use]
    pub const fn operation(&self) -> OperationIdentity {
        self.operation
    }

    /// Returns the failure class without retry advice.
    #[must_use]
    pub const fn kind(&self) -> TransportFailureKind {
        self.kind
    }
}

/// A fallible exact-byte stream that replays bytes consumed during classification.
pub struct BodyStream {
    operation: OperationIdentity,
    replay: VecDeque<Bytes>,
    remaining: RawStream,
}

impl fmt::Debug for BodyStream {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter
            .debug_struct("BodyStream")
            .field("operation", &self.operation)
            .field("retained_chunks", &self.replay.len())
            .finish_non_exhaustive()
    }
}

impl Stream for BodyStream {
    type Item = Result<BodyChunk, BodyStreamError>;

    fn poll_next(mut self: Pin<&mut Self>, context: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        if let Some(bytes) = self.replay.pop_front() {
            return Poll::Ready(Some(Ok(BodyChunk(bytes))));
        }
        match self.remaining.as_mut().poll_next(context) {
            Poll::Ready(Some(Ok(bytes))) => Poll::Ready(Some(Ok(BodyChunk(bytes)))),
            Poll::Ready(Some(Err(kind))) => Poll::Ready(Some(Err(BodyStreamError {
                operation: self.operation,
                kind,
            }))),
            Poll::Ready(None) => Poll::Ready(None),
            Poll::Pending => Poll::Pending,
        }
    }
}

type RawStream = Pin<Box<dyn Stream<Item = Result<Bytes, TransportFailureKind>> + Send>>;

struct ReqwestBodyStream {
    inner: Pin<Box<dyn Stream<Item = Result<Bytes, reqwest::Error>> + Send>>,
    deadline: Pin<Box<tokio::time::Sleep>>,
    terminal: bool,
}

impl Stream for ReqwestBodyStream {
    type Item = Result<Bytes, TransportFailureKind>;

    fn poll_next(mut self: Pin<&mut Self>, context: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        if self.terminal {
            return Poll::Ready(None);
        }
        if self.deadline.as_mut().poll(context).is_ready() {
            self.terminal = true;
            return Poll::Ready(Some(Err(TransportFailureKind::Timeout)));
        }
        match self.inner.as_mut().poll_next(context) {
            Poll::Ready(Some(Ok(bytes))) => Poll::Ready(Some(Ok(bytes))),
            Poll::Ready(Some(Err(error))) => {
                self.terminal = true;
                let kind = if error.is_timeout() {
                    TransportFailureKind::Timeout
                } else if error.is_body() || error.is_decode() {
                    TransportFailureKind::Body
                } else if error.is_connect() {
                    TransportFailureKind::Connection
                } else {
                    TransportFailureKind::Protocol
                };
                Poll::Ready(Some(Err(kind)))
            }
            Poll::Ready(None) => {
                self.terminal = true;
                Poll::Ready(None)
            }
            Poll::Pending => Poll::Pending,
        }
    }
}

async fn classify_binary(
    mut remaining: RawStream,
    inspector: WireInspector,
) -> BinaryReply<BodyStream> {
    let mut replay = VecDeque::new();
    let mut prefix = Vec::new();
    let mut terminal = false;

    while prefix.len() < 4 {
        match poll_fn(|context| remaining.as_mut().poll_next(context)).await {
            Some(Ok(bytes)) => {
                let inspection_limit = inspector.max_envelope_bytes().saturating_add(1).max(4);
                let retained = bytes.len().min(inspection_limit);
                prefix.extend_from_slice(&bytes[..retained]);
                replay.push_back(bytes);
            }
            Some(Err(kind)) => {
                remaining = Box::pin(OneErrorStream(Some(kind)));
                terminal = true;
                break;
            }
            None => break,
        }
    }

    if supported_zip_signature(&prefix) {
        return BinaryReply::Archive(BodyStream {
            operation: placeholder_operation(),
            replay,
            remaining,
        });
    }

    while !terminal
        && xml_prefix_state(&prefix) == XmlPrefixState::Inconclusive
        && prefix.len() <= inspector.max_envelope_bytes()
    {
        match poll_fn(|context| remaining.as_mut().poll_next(context)).await {
            Some(Ok(bytes)) => {
                let remaining_capacity = inspector
                    .max_envelope_bytes()
                    .saturating_add(1)
                    .saturating_sub(prefix.len());
                prefix.extend_from_slice(&bytes[..bytes.len().min(remaining_capacity)]);
                replay.push_back(bytes);
            }
            Some(Err(kind)) => {
                remaining = Box::pin(OneErrorStream(Some(kind)));
                terminal = true;
            }
            None => break,
        }
    }

    if !terminal && xml_prefix_state(&prefix) == XmlPrefixState::Candidate {
        while prefix.len() <= inspector.max_envelope_bytes() {
            match poll_fn(|context| remaining.as_mut().poll_next(context)).await {
                Some(Ok(bytes)) => {
                    let remaining_capacity = inspector
                        .max_envelope_bytes()
                        .saturating_add(1)
                        .saturating_sub(prefix.len());
                    prefix.extend_from_slice(&bytes[..bytes.len().min(remaining_capacity)]);
                    replay.push_back(bytes);
                }
                Some(Err(kind)) => {
                    remaining = Box::pin(OneErrorStream(Some(kind)));
                    terminal = true;
                    break;
                }
                None => break,
            }
        }
        if !terminal && prefix.len() <= inspector.max_envelope_bytes() {
            if let Ok(SourceReply::Status(status)) = inspector.inspect_xml(&prefix) {
                return BinaryReply::Status(status);
            }
        }
    }

    BinaryReply::Unrecognized(BodyStream {
        operation: placeholder_operation(),
        replay,
        remaining,
    })
}

struct OneErrorStream(Option<TransportFailureKind>);

impl Stream for OneErrorStream {
    type Item = Result<Bytes, TransportFailureKind>;

    fn poll_next(mut self: Pin<&mut Self>, _: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        Poll::Ready(self.0.take().map(Err))
    }
}

fn supported_zip_signature(prefix: &[u8]) -> bool {
    prefix.starts_with(b"PK\x03\x04") || prefix.starts_with(b"PK\x05\x06")
}

#[derive(Clone, Copy, Eq, PartialEq)]
enum XmlPrefixState {
    Candidate,
    NotXml,
    Inconclusive,
}

fn xml_prefix_state(prefix: &[u8]) -> XmlPrefixState {
    let prefix = if prefix.starts_with(b"\xEF\xBB\xBF") {
        &prefix[3..]
    } else if b"\xEF\xBB\xBF".starts_with(prefix) {
        return XmlPrefixState::Inconclusive;
    } else {
        prefix
    };
    match prefix
        .iter()
        .copied()
        .find(|byte| !byte.is_ascii_whitespace())
    {
        Some(b'<') => XmlPrefixState::Candidate,
        Some(_) => XmlPrefixState::NotXml,
        None => XmlPrefixState::Inconclusive,
    }
}

fn placeholder_operation() -> OperationIdentity {
    OperationIdentity::new("binary-stream", "binary-stream")
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::operations::{Company, CorpCode};
    use std::{
        io::{Read, Write},
        net::TcpListener as StdTcpListener,
        process::Command,
        sync::{
            Arc,
            atomic::{AtomicUsize, Ordering},
        },
        thread,
    };
    use tokio::{
        io::{AsyncReadExt, AsyncWriteExt},
        net::TcpListener,
        task::JoinHandle,
    };

    struct ChunkStream(VecDeque<Result<Bytes, TransportFailureKind>>);

    impl Stream for ChunkStream {
        type Item = Result<Bytes, TransportFailureKind>;

        fn poll_next(mut self: Pin<&mut Self>, _: &mut Context<'_>) -> Poll<Option<Self::Item>> {
            Poll::Ready(self.0.pop_front())
        }
    }

    fn chunks(parts: &[&[u8]]) -> RawStream {
        Box::pin(ChunkStream(
            parts
                .iter()
                .map(|part| Ok(Bytes::copy_from_slice(part)))
                .collect(),
        ))
    }

    async fn serve_once(response: Vec<u8>) -> (String, JoinHandle<Vec<u8>>) {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let address = listener.local_addr().unwrap();
        let task = tokio::spawn(async move {
            let (mut socket, _) = listener.accept().await.unwrap();
            let mut request = Vec::new();
            let mut buffer = [0; 1024];
            while !request.windows(4).any(|window| window == b"\r\n\r\n") {
                let read = socket.read(&mut buffer).await.unwrap();
                if read == 0 {
                    break;
                }
                request.extend_from_slice(&buffer[..read]);
            }
            socket.write_all(&response).await.unwrap();
            request
        });
        (format!("http://{address}"), task)
    }

    async fn serve_refused_h2() -> (String, Arc<AtomicUsize>, JoinHandle<()>) {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let address = listener.local_addr().unwrap();
        let request_count = Arc::new(AtomicUsize::new(0));
        let observed = Arc::clone(&request_count);
        let task = tokio::spawn(async move {
            loop {
                let (socket, _) = listener.accept().await.unwrap();
                let observed = Arc::clone(&observed);
                tokio::spawn(async move {
                    let mut connection = h2::server::handshake(socket).await.unwrap();
                    while let Some(request) = connection.accept().await {
                        let (_, mut response) = request.unwrap();
                        observed.fetch_add(1, Ordering::SeqCst);
                        response.send_reset(h2::Reason::REFUSED_STREAM);
                    }
                });
            }
        });
        (format!("http://{address}"), request_count, task)
    }

    fn response(headers: &[(&str, &str)], body: &[u8]) -> Vec<u8> {
        let mut response = format!("HTTP/1.1 200 OK\r\nContent-Length: {}\r\n", body.len());
        for (name, value) in headers {
            response.push_str(name);
            response.push_str(": ");
            response.push_str(value);
            response.push_str("\r\n");
        }
        response.push_str("\r\n");
        let mut response = response.into_bytes();
        response.extend_from_slice(body);
        response
    }

    fn company_request() -> PreparedRequest {
        Company::new("00126380")
            .prepare(Representation::Json)
            .unwrap()
    }

    async fn collect(mut stream: BodyStream) -> (Vec<u8>, Option<TransportFailureKind>) {
        let mut body = Vec::new();
        let mut error = None;
        while let Some(item) = poll_fn(|context| Pin::new(&mut stream).poll_next(context)).await {
            match item {
                Ok(chunk) => body.extend_from_slice(chunk.as_bytes()),
                Err(item) => error = Some(item.kind()),
            }
        }
        (body, error)
    }

    #[tokio::test]
    async fn zip_signatures_are_positive_and_replay_every_partition() {
        for body in [
            b"PK\x03\x04archive".as_slice(),
            b"PK\x05\x06empty".as_slice(),
        ] {
            for split in 1..body.len() {
                let BinaryReply::Archive(stream) = classify_binary(
                    chunks(&[&body[..split], &body[split..]]),
                    WireInspector::new(128).unwrap(),
                )
                .await
                else {
                    panic!("supported ZIP signature was not recognized");
                };
                assert_eq!(collect(stream).await.0, body);
            }
        }

        for limit in 1..=3 {
            let body = b"PK\x03\x04archive";
            let BinaryReply::Archive(stream) =
                classify_binary(chunks(&[body]), WireInspector::new(limit).unwrap()).await
            else {
                panic!("the envelope limit must not weaken ZIP discrimination");
            };
            assert_eq!(collect(stream).await.0, body);
        }
    }

    #[tokio::test]
    async fn ambiguous_binary_and_xml_candidates_replay_exact_bytes() {
        for body in [
            b"PK\x07\x08split".as_slice(),
            b"PK\x06\x06zip64".as_slice(),
            b"MZself-extracting".as_slice(),
            b"PK\x03".as_slice(),
            b"<result><status>013".as_slice(),
            b"<result><payload>x</payload></result>".as_slice(),
        ] {
            let BinaryReply::Unrecognized(stream) = classify_binary(
                chunks(&[&body[..1], &body[1..]]),
                WireInspector::new(128).unwrap(),
            )
            .await
            else {
                panic!("ambiguous body was over-classified");
            };
            assert_eq!(collect(stream).await.0, body);
        }

        let oversized = b"   <result><status>013</status></result>";
        let BinaryReply::Unrecognized(stream) =
            classify_binary(chunks(&[oversized]), WireInspector::new(8).unwrap()).await
        else {
            panic!("oversized XML candidate was over-classified");
        };
        assert_eq!(collect(stream).await.0, oversized);
    }

    #[tokio::test]
    async fn recognized_xml_status_consumes_only_the_alternate_envelope() {
        let body = b" \n<result><status>013</status><message>none</message></result>";
        let BinaryReply::Status(status) = classify_binary(
            chunks(&[&body[..2], &body[2..13], &body[13..]]),
            WireInspector::new(128).unwrap(),
        )
        .await
        else {
            panic!("bounded status envelope was not recognized");
        };
        assert_eq!(status.code.as_str(), "013");
    }

    #[tokio::test]
    async fn stream_error_follows_every_successfully_read_byte() {
        let stream: RawStream = Box::pin(ChunkStream(VecDeque::from([
            Ok(Bytes::from_static(b"PK\x03")),
            Err(TransportFailureKind::Timeout),
        ])));
        let BinaryReply::Unrecognized(stream) =
            classify_binary(stream, WireInspector::new(128).unwrap()).await
        else {
            panic!("truncated prefix was over-classified");
        };
        let (body, error) = collect(stream).await;
        assert_eq!(body, b"PK\x03");
        assert_eq!(error, Some(TransportFailureKind::Timeout));
    }

    #[test]
    fn builder_rejects_zero_bounds_and_unsafe_user_agent_values() {
        let key = ApiKey::new("key").unwrap();
        assert!(matches!(
            Client::builder(key).envelope_limit(0).build(),
            Err(ClientBuildError::InvalidConfiguration { .. })
        ));
        let key = ApiKey::new("key").unwrap();
        assert!(matches!(
            Client::builder(key).user_agent_suffix("bad value").build(),
            Err(ClientBuildError::InvalidConfiguration { .. })
        ));

        for setting in ["connect timeout", "read timeout", "total timeout"] {
            let builder = Client::builder(ApiKey::new("key").unwrap());
            let result = match setting {
                "connect timeout" => builder.connect_timeout(Duration::MAX).build(),
                "read timeout" => builder.read_timeout(Duration::MAX).build(),
                "total timeout" => builder.total_timeout(Duration::MAX).build(),
                _ => unreachable!(),
            };
            assert!(matches!(
                result,
                Err(ClientBuildError::InvalidConfiguration {
                    setting: rejected
                }) if rejected == setting
            ));
        }
    }

    #[tokio::test]
    async fn client_sends_one_authorized_request_and_sanitizes_metadata() {
        let sentinel = "secret /+ credential";
        let encoded = "secret+%2F%2B+credential";
        let body = br#"{"status":"000","value":1.20e3}"#;
        let unsafe_header = format!("https://elsewhere.invalid/?crtfc_key={encoded}");
        let (origin, server) = serve_once(response(
            &[("x-safe", "retained"), ("location", &unsafe_header)],
            body,
        ))
        .await;
        let client = Client::builder(ApiKey::new(sentinel).unwrap())
            .test_origin(origin)
            .build()
            .unwrap();

        let result = client.execute(&company_request()).await.unwrap();
        let SourceReply::Success(value) = result.reply else {
            panic!("payload-bearing status must remain success evidence");
        };
        assert_eq!(
            value.get("value").and_then(SourceValue::as_number_str),
            Some("1.20e3")
        );
        assert!(
            result
                .metadata
                .headers()
                .iter()
                .any(|header| { header.name() == "x-safe" && header.value() == b"retained" })
        );
        assert!(
            !result
                .metadata
                .headers()
                .iter()
                .any(|header| header.name() == "location")
        );

        let request = String::from_utf8(server.await.unwrap()).unwrap();
        assert!(request.starts_with(
            "GET /api/company.json?corp_code=00126380&crtfc_key=secret+%2F%2B+credential HTTP/1.1\r\n"
        ));
        assert_eq!(request.matches("crtfc_key=").count(), 1);
        assert!(!format!("{client:?}").contains(sentinel));
    }

    #[tokio::test]
    async fn protocol_nack_is_not_retried_even_when_reqwest_features_are_unified() {
        let (origin, request_count, server) = serve_refused_h2().await;
        let default = reqwest::Client::builder()
            .no_proxy()
            .http2_prior_knowledge()
            .build()
            .unwrap();
        let _ = tokio::time::timeout(
            Duration::from_secs(2),
            default.get(format!("{origin}/default")).send(),
        )
        .await
        .unwrap();
        assert!(
            request_count.load(Ordering::SeqCst) > 1,
            "the positive control must observe reqwest's default protocol-NACK retry"
        );
        server.abort();

        let (origin, request_count, server) = serve_refused_h2().await;
        let client = Client::builder(ApiKey::new("nack-sentinel").unwrap())
            .test_h2_origin(origin)
            .build()
            .unwrap();
        let _ = tokio::time::timeout(Duration::from_secs(2), client.execute(&company_request()))
            .await
            .unwrap();
        assert_eq!(request_count.load(Ordering::SeqCst), 1);
        server.abort();
    }

    #[tokio::test]
    async fn redirects_are_returned_without_a_second_request() {
        let target = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let target_address = target.local_addr().unwrap();
        let target_observation = tokio::spawn(async move {
            tokio::time::timeout(Duration::from_millis(300), target.accept())
                .await
                .is_ok()
        });
        let redirect = format!(
            "HTTP/1.1 302 Found\r\nLocation: http://{target_address}/leak\r\nContent-Length: 0\r\n\r\n"
        );
        let (origin, first) = serve_once(redirect.into_bytes()).await;
        let client = Client::builder(ApiKey::new("redirect-sentinel").unwrap())
            .test_origin(origin)
            .build()
            .unwrap();

        let error = client.execute(&company_request()).await.unwrap_err();
        assert_eq!(error.metadata().map(ResponseMetadata::status), Some(302));
        assert_eq!(
            first
                .await
                .unwrap()
                .windows(4)
                .filter(|part| *part == b"GET ")
                .count(),
            1
        );
        assert!(!target_observation.await.unwrap());
    }

    #[tokio::test]
    async fn feature_unification_does_not_enable_content_decoding() {
        for encoding in ["gzip", "br", "zstd", "deflate"] {
            let encoded = format!("opaque-{encoding}-encoded-entity").into_bytes();
            let (origin, server) =
                serve_once(response(&[("content-encoding", encoding)], &encoded)).await;
            let client = Client::builder(ApiKey::new("key").unwrap())
                .test_origin(origin)
                .build()
                .unwrap();
            let prepared = CorpCode::new().prepare(Representation::Zip).unwrap();

            let result = client.execute_binary(&prepared).await.unwrap();
            let BinaryReply::Unrecognized(stream) = result.reply else {
                panic!("encoded bytes are not positive ZIP evidence");
            };
            assert_eq!(collect(stream).await.0, encoded);
            assert!(result.metadata.headers().iter().any(|header| {
                header.name() == "content-encoding" && header.value() == encoding.as_bytes()
            }));
            server.await.unwrap();
        }
    }

    #[tokio::test]
    async fn read_timeout_and_incomplete_body_retain_response_metadata() {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let address = listener.local_addr().unwrap();
        let stalled = tokio::spawn(async move {
            let (mut socket, _) = listener.accept().await.unwrap();
            let mut request = [0; 1024];
            let _ = socket.read(&mut request).await.unwrap();
            socket
                .write_all(b"HTTP/1.1 200 OK\r\nContent-Length: 10\r\n\r\nabc")
                .await
                .unwrap();
            tokio::time::sleep(Duration::from_millis(300)).await;
        });
        let client = Client::builder(ApiKey::new("timeout-sentinel").unwrap())
            .test_origin(format!("http://{address}"))
            .read_timeout(Duration::from_millis(50))
            .total_timeout(Duration::from_secs(1))
            .build()
            .unwrap();

        let error = client.execute(&company_request()).await.unwrap_err();
        assert_eq!(error.metadata().map(ResponseMetadata::status), Some(200));
        assert!(matches!(
            error,
            ClientError::Transport(TransportError {
                kind: TransportFailureKind::Timeout,
                ..
            })
        ));
        assert!(!format!("{error:?}").contains("timeout-sentinel"));
        stalled.await.unwrap();

        let (origin, server) =
            serve_once(b"HTTP/1.1 200 OK\r\nContent-Length: 10\r\n\r\nabc".to_vec()).await;
        let client = Client::builder(ApiKey::new("incomplete-sentinel").unwrap())
            .test_origin(origin)
            .build()
            .unwrap();
        let error = client.execute(&company_request()).await.unwrap_err();
        assert_eq!(error.metadata().map(ResponseMetadata::status), Some(200));
        assert!(matches!(
            error,
            ClientError::Transport(TransportError {
                kind: TransportFailureKind::Body,
                ..
            })
        ));
        server.await.unwrap();
    }

    #[test]
    fn ambient_proxy_variables_do_not_change_route() {
        const CHILD: &str = "OPENDART_PROXY_TEST_CHILD";
        const TARGET: &str = "OPENDART_PROXY_TEST_TARGET";
        if std::env::var_os(CHILD).is_some() {
            let origin = std::env::var(TARGET).unwrap();
            let runtime = tokio::runtime::Runtime::new().unwrap();
            runtime.block_on(async move {
                let client = Client::builder(ApiKey::new("proxy-sentinel").unwrap())
                    .test_origin(origin)
                    .build()
                    .unwrap();
                let result = client.execute(&company_request()).await.unwrap();
                assert_eq!(result.metadata.status(), 200);
            });
            return;
        }

        let target = StdTcpListener::bind("127.0.0.1:0").unwrap();
        let target_origin = format!("http://{}", target.local_addr().unwrap());
        let target_count = Arc::new(AtomicUsize::new(0));
        let target_observed = Arc::clone(&target_count);
        let target_thread = thread::spawn(move || {
            let (mut socket, _) = target.accept().unwrap();
            target_observed.fetch_add(1, Ordering::SeqCst);
            let mut request = [0; 2048];
            let _ = socket.read(&mut request).unwrap();
            socket
                .write_all(
                    b"HTTP/1.1 200 OK\r\nContent-Length: 31\r\n\r\n{\"status\":\"000\",\"payload\":true}",
                )
                .unwrap();
        });

        let proxy = StdTcpListener::bind("127.0.0.1:0").unwrap();
        let proxy_url = format!("http://{}", proxy.local_addr().unwrap());
        proxy.set_nonblocking(true).unwrap();
        let proxy_count = Arc::new(AtomicUsize::new(0));
        let proxy_observed = Arc::clone(&proxy_count);
        let proxy_thread = thread::spawn(move || {
            for _ in 0..100 {
                match proxy.accept() {
                    Ok((mut socket, _)) => {
                        proxy_observed.fetch_add(1, Ordering::SeqCst);
                        let _ = socket
                            .write_all(b"HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n");
                    }
                    Err(error) if error.kind() == std::io::ErrorKind::WouldBlock => {
                        thread::sleep(Duration::from_millis(10));
                    }
                    Err(error) => panic!("proxy fixture failed: {error}"),
                }
            }
        });

        let output = Command::new(std::env::current_exe().unwrap())
            .args([
                "--exact",
                "client::tests::ambient_proxy_variables_do_not_change_route",
                "--nocapture",
            ])
            .env(CHILD, "1")
            .env(TARGET, target_origin)
            .env("HTTP_PROXY", &proxy_url)
            .env("HTTPS_PROXY", &proxy_url)
            .env("ALL_PROXY", &proxy_url)
            .env_remove("NO_PROXY")
            .env_remove("no_proxy")
            .output()
            .unwrap();

        target_thread.join().unwrap();
        proxy_thread.join().unwrap();
        assert!(
            output.status.success(),
            "proxy child failed: {}",
            String::from_utf8_lossy(&output.stderr)
        );
        assert_eq!(target_count.load(Ordering::SeqCst), 1);
        assert_eq!(proxy_count.load(Ordering::SeqCst), 0);
    }

    #[tokio::test]
    async fn incomplete_binary_stream_replays_partial_bytes_then_fails() {
        let body = b"PK\x03\x04partial";
        let response = format!(
            "HTTP/1.1 200 OK\r\nContent-Length: {}\r\n\r\n",
            body.len() + 10
        );
        let mut response = response.into_bytes();
        response.extend_from_slice(body);
        let (origin, server) = serve_once(response).await;
        let client = Client::builder(ApiKey::new("binary-sentinel").unwrap())
            .test_origin(origin)
            .build()
            .unwrap();
        let prepared = CorpCode::new().prepare(Representation::Zip).unwrap();

        let result = client.execute_binary(&prepared).await.unwrap();
        assert_eq!(result.metadata.status(), 200);
        let BinaryReply::Archive(stream) = result.reply else {
            panic!("the supported prefix is positive archive evidence");
        };
        let (observed, error) = collect(stream).await;
        assert_eq!(observed, body);
        assert_eq!(error, Some(TransportFailureKind::Body));
        server.await.unwrap();
    }

    #[tokio::test]
    async fn total_deadline_remains_active_on_returned_binary_stream() {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let address = listener.local_addr().unwrap();
        let server = tokio::spawn(async move {
            let (mut socket, _) = listener.accept().await.unwrap();
            let mut request = [0; 1024];
            let _ = socket.read(&mut request).await.unwrap();
            socket
                .write_all(b"HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\nPK\x03\x04")
                .await
                .unwrap();
            for _ in 0..10 {
                tokio::time::sleep(Duration::from_millis(25)).await;
                if socket.write_all(b"x").await.is_err() {
                    break;
                }
            }
        });
        let client = Client::builder(ApiKey::new("deadline-sentinel").unwrap())
            .test_origin(format!("http://{address}"))
            .read_timeout(Duration::from_millis(200))
            .total_timeout(Duration::from_millis(70))
            .build()
            .unwrap();
        let prepared = CorpCode::new().prepare(Representation::Zip).unwrap();

        let started = tokio::time::Instant::now();
        let result = client.execute_binary(&prepared).await.unwrap();
        assert_eq!(result.metadata.status(), 200);
        let BinaryReply::Archive(stream) = result.reply else {
            panic!("the first entity bytes are positive ZIP evidence");
        };
        let (body, error) = collect(stream).await;
        assert!(body.starts_with(b"PK\x03\x04"));
        assert_eq!(error, Some(TransportFailureKind::Timeout));
        assert!(started.elapsed() < Duration::from_millis(200));
        server.await.unwrap();
    }
}
