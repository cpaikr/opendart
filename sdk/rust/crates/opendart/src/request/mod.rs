use std::fmt;

use form_urlencoded::Serializer;
use secrecy::{ExposeSecret, SecretString};
use zeroize::Zeroizing;

use crate::AuthorizationError;

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

/// An immutable, credential-free OpenDART request prepared without network I/O.
pub struct PreparedRequest {
    method: RequestMethod,
    relative_path: &'static str,
    encoded_query: String,
    authentication: Authentication,
    identity: OperationIdentity,
    expected_representations: &'static [Representation],
    generator_schema: u32,
    projection_identity: &'static str,
}

pub(crate) enum QueryValue<'a> {
    Scalar(&'a str),
    CommaSeparated(&'a [String]),
}

pub(crate) struct QueryParameter<'a> {
    pub(crate) name: &'static str,
    pub(crate) value: QueryValue<'a>,
}

impl PreparedRequest {
    pub(crate) fn new(
        relative_path: &'static str,
        identity: OperationIdentity,
        parameters: &[QueryParameter<'_>],
        expected_representations: &'static [Representation],
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
            generator_schema: 1,
            projection_identity: "handwritten-contract-v1",
        }
    }

    /// Returns the HTTP method.
    #[must_use]
    pub const fn method(&self) -> RequestMethod {
        self.method
    }

    /// Returns the trusted credential-free relative path.
    #[must_use]
    pub const fn relative_path(&self) -> &'static str {
        self.relative_path
    }

    /// Returns the deterministically encoded, credential-free query string.
    #[must_use]
    pub fn encoded_query(&self) -> &str {
        &self.encoded_query
    }

    /// Returns the required credential placement.
    #[must_use]
    pub const fn authentication(&self) -> Authentication {
        self.authentication
    }

    /// Returns the stable operation identity.
    #[must_use]
    pub const fn identity(&self) -> OperationIdentity {
        self.identity
    }

    /// Returns the representations expected from this physical operation.
    #[must_use]
    pub const fn expected_representations(&self) -> &'static [Representation] {
        self.expected_representations
    }

    /// Returns the SDK generator schema version used to prepare this request.
    #[must_use]
    pub const fn generator_schema(&self) -> u32 {
        self.generator_schema
    }

    /// Returns the SDK projection identity used for safe diagnostics.
    #[must_use]
    pub const fn projection_identity(&self) -> &'static str {
        self.projection_identity
    }

    /// Adds the API credential at the explicit adapter boundary.
    #[must_use]
    pub fn authorize<'a>(&'a self, api_key: &'a ApiKey) -> AuthorizedRequest<'a> {
        AuthorizedRequest {
            prepared: self,
            api_key,
        }
    }
}

impl fmt::Debug for PreparedRequest {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter
            .debug_struct("PreparedRequest")
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
}

/// An owned OpenDART API credential with redacted diagnostics and zeroizing drop.
pub struct ApiKey {
    secret: SecretString,
}

impl ApiKey {
    /// Validates and owns an API key without exposing it through formatting.
    pub fn new(value: impl Into<String>) -> Result<Self, AuthorizationError> {
        let value = value.into();
        if value.is_empty() {
            return Err(AuthorizationError::EmptyApiKey);
        }
        Ok(Self {
            secret: value.into(),
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
    prepared: &'a PreparedRequest,
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
    /// # use opendart::{ApiKey, Representation, operations::CompanyOverview};
    /// # let prepared = CompanyOverview::new("00126380").prepare(Representation::Json)?;
    /// # let key = ApiKey::new("example-key")?;
    /// let authorized = prepared.authorize(&key);
    /// authorized.with_exposed_relative_uri(|_| ());
    /// authorized.with_exposed_relative_uri(|_| ()); // consumed by the first adapter call
    /// # Ok::<(), Box<dyn std::error::Error>>(())
    /// ```
    pub fn with_exposed_relative_uri<T>(self, adapter: impl FnOnce(&str) -> T) -> T {
        let mut serializer = Serializer::new(String::new());
        serializer.append_pair("crtfc_key", self.api_key.secret.expose_secret());
        let credential_query = Zeroizing::new(serializer.finish());
        let relative_uri = if self.prepared.encoded_query.is_empty() {
            format!(
                "{}?{}",
                self.prepared.relative_path,
                credential_query.as_str()
            )
        } else {
            format!(
                "{}?{}&{}",
                self.prepared.relative_path,
                self.prepared.encoded_query,
                credential_query.as_str()
            )
        };
        let relative_uri = Zeroizing::new(relative_uri);
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

fn encode_query_value(value: &str) -> String {
    let mut serializer = Serializer::new(String::new());
    serializer.append_pair("value", value);
    serializer
        .finish()
        .strip_prefix("value=")
        .expect("the fixed query name is present")
        .to_owned()
}
