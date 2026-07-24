use crate::OperationIdentity;

/// A caller supplied a source-number spelling outside the SDK-owned JSON grammar.
///
/// The rejected spelling is deliberately not retained so partially valid source
/// evidence cannot escape through diagnostics.
#[derive(Clone, Copy, Debug, Eq, PartialEq, thiserror::Error)]
#[error("source number is not valid JSON")]
#[non_exhaustive]
pub struct InvalidSourceNumberError;

/// A deterministic request could not be prepared from the supplied operation input.
#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum PrepareError {
    /// A required input was empty.
    #[error("{operation} requires a non-empty {parameter} input")]
    MissingInput {
        /// The operation whose request could not be prepared.
        operation: OperationIdentity,
        /// The canonical wire parameter name.
        parameter: &'static str,
    },
    /// An explicit collection cardinality constraint was violated.
    #[error("{operation} parameter {parameter} requires between {minimum} and {maximum} items")]
    InvalidCardinality {
        /// The operation whose request could not be prepared.
        operation: OperationIdentity,
        /// The canonical wire parameter name.
        parameter: &'static str,
        /// Inclusive minimum item count.
        minimum: usize,
        /// Inclusive maximum item count.
        maximum: usize,
    },
    /// A string length constraint was violated.
    #[error(
        "{operation} parameter {parameter} requires between {minimum} and {maximum} characters"
    )]
    InvalidLength {
        /// The operation whose request could not be prepared.
        operation: OperationIdentity,
        /// The canonical wire parameter name.
        parameter: &'static str,
        /// Inclusive minimum character count.
        minimum: usize,
        /// Inclusive maximum character count.
        maximum: usize,
    },
    /// A documented string format was violated.
    #[error("{operation} parameter {parameter} does not match format {format}")]
    InvalidFormat {
        /// The operation whose request could not be prepared.
        operation: OperationIdentity,
        /// The canonical wire parameter name.
        parameter: &'static str,
        /// The stable canonical format identifier.
        format: &'static str,
    },
    /// A value was outside a documented closed set.
    #[error("{operation} parameter {parameter} is not one of its allowed values")]
    InvalidAllowedValue {
        /// The operation whose request could not be prepared.
        operation: OperationIdentity,
        /// The canonical wire parameter name.
        parameter: &'static str,
    },
    /// A decimal string was malformed or outside its inclusive range.
    #[error("{operation} parameter {parameter} is outside its allowed decimal range")]
    InvalidDecimalRange {
        /// The operation whose request could not be prepared.
        operation: OperationIdentity,
        /// The canonical wire parameter name.
        parameter: &'static str,
        /// Inclusive minimum value.
        minimum: u64,
        /// Inclusive maximum value, or no upper bound.
        maximum: Option<u64>,
    },
}

/// An API credential could not cross the explicit authorization boundary.
#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum AuthorizationError {
    /// The supplied API key was empty or contained only whitespace.
    #[error("the OpenDART API key must not be empty or whitespace-only")]
    EmptyApiKey,
    /// The supplied API key contained a control character.
    #[error("the OpenDART API key must not contain control characters")]
    ControlCharacterApiKey,
}
