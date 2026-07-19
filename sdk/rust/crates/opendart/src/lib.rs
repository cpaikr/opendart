#![doc = include_str!("../README.md")]
#![forbid(unsafe_code)]

mod error;
mod request;
mod wire;

pub mod operations;

pub use error::{AuthorizationError, PrepareError};
pub use request::{
    ApiKey, Authentication, AuthorizedRequest, OperationIdentity, PreparedRequest, Representation,
    RequestMethod,
};
pub use wire::{
    HttpVersion, ResponseHeader, ResponseMetadata, SourceReply, SourceResponse, SourceStatus,
    SourceValue, SourceValueKind, StatusEnvelope,
};
