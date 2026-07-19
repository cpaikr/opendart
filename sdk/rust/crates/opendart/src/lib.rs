#![doc = include_str!("../README.md")]
#![forbid(unsafe_code)]

mod error;
#[rustfmt::skip]
mod generated;
mod request;
mod wire;

pub use generated::{mapping, operations, wire_shapes};

pub use error::{AuthorizationError, PrepareError};
pub use request::{
    ApiKey, Authentication, AuthorizedRequest, OperationIdentity, PreparedRequest, Representation,
    RequestMethod,
};
pub use wire::{
    HttpVersion, ResponseHeader, ResponseMetadata, SourceReply, SourceResponse, SourceStatus,
    SourceValue, SourceValueKind, StatusEnvelope,
};
