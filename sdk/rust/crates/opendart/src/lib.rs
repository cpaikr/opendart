#![doc = include_str!("../README.md")]
#![forbid(unsafe_code)]

mod error;
#[rustfmt::skip]
mod generated;
mod provenance;
mod request;
mod wire;

#[cfg(all(feature = "client-reqwest", not(target_family = "wasm")))]
mod client;
#[cfg(all(
    opendart_compat,
    feature = "client-reqwest",
    not(target_family = "wasm")
))]
#[path = "../../../compat/reqwest-feature-unification/opendart_bridge.rs"]
pub mod compatibility;

pub use generated::operations;
pub use generated::responses;
pub use provenance::{SourceProvenance, source_provenance};

#[cfg(all(feature = "client-reqwest", not(target_family = "wasm")))]
pub use client::{
    BinaryReply, BodyChunk, BodyStream, BodyStreamError, Client, ClientBuildError, ClientBuilder,
    ClientError, TransportError, TransportFailureKind,
};
pub use error::{AuthorizationError, InvalidSourceNumberError, PrepareError};
pub use request::{
    ApiKey, Authentication, AuthorizedRequest, OperationIdentity, PreparedBinaryRequest,
    PreparedRequest, Representation, RequestMethod,
};
pub use wire::{
    BodyLimitError, EnvelopeError, EnvelopeFormat, HttpVersion, ResponseDecodeError,
    ResponseHeader, ResponseMetadata, SourceReply, SourceResponse, SourceStatus, SourceValue,
    SourceValueKind, StatusEnvelope, WireInspectError, WireInspector,
};
