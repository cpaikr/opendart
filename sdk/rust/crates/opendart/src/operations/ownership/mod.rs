//! Handwritten DS004 operations.

use crate::protocol::{Query, items, source_status};
use crate::request::RequestParts;
use crate::values::CompanyCode;
use crate::wire::decode::{ObjectDecoder, decode_array, decode_source_status, decode_xml_array};
use crate::{
    OperationIdentity, PrepareError, PreparedRequest, ResponseDecodeError, SourceStatus,
    SourceValue,
};

mod major_shareholdings;
pub use major_shareholdings::*;
mod insider_shareholdings;
pub use insider_shareholdings::*;
