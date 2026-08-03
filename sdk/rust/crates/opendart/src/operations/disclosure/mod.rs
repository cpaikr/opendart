//! Handwritten DS001 operations.

use crate::protocol::{Query, items, source_status};
use crate::request::RequestParts;
use crate::values::{CompactDate, CompanyCode};
use crate::wire::decode::{ObjectDecoder, decode_array, decode_source_status, decode_xml_array};
use crate::{
    OperationIdentity, PrepareError, PreparedBinaryRequest, PreparedRequest, ResponseDecodeError,
    SourceStatus, SourceValue,
};

mod disclosure_search;
pub use disclosure_search::*;
mod company_overview;
pub use company_overview::*;
mod disclosure_document;
pub use disclosure_document::*;
mod company_codes;
pub use company_codes::*;

#[cfg(test)]
mod tests;
