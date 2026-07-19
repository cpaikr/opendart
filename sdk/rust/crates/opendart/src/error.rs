use crate::OperationIdentity;

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
}

/// An API credential could not cross the explicit authorization boundary.
#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum AuthorizationError {
    /// The supplied API key was empty.
    #[error("the OpenDART API key must not be empty")]
    EmptyApiKey,
}
