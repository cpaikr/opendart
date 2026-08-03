//! Handwritten DS006 operations.

use crate::protocol::{Query, items, source_status};
use crate::request::RequestParts;
use crate::values::{CompactDate, CompanyCode};
use crate::wire::decode::{ObjectDecoder, decode_array, decode_source_status, decode_xml_array};
use crate::{
    OperationIdentity, PrepareError, PreparedRequest, ResponseDecodeError, SourceStatus,
    SourceValue,
};

mod equity_securities_registration;
pub use equity_securities_registration::*;
mod debt_securities_registration;
pub use debt_securities_registration::*;
mod depositary_receipt_registration;
pub use depositary_receipt_registration::*;
mod merger_registration;
pub use merger_registration::*;
mod share_exchange_transfer_registration;
pub use share_exchange_transfer_registration::*;
mod demerger_registration;
pub use demerger_registration::*;
