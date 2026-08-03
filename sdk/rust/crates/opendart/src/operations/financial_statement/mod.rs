//! Handwritten DS003 operations.

use crate::protocol::{Query, items, source_status};
use crate::request::RequestParts;
use crate::values::{BusinessYear, CompanyCode, ReportCode};
use crate::wire::decode::{ObjectDecoder, decode_array, decode_source_status, decode_xml_array};
use crate::{
    OperationIdentity, PrepareError, PreparedBinaryRequest, PreparedRequest, ResponseDecodeError,
    SourceStatus, SourceValue,
};

mod company_key_accounts;
pub use company_key_accounts::*;
mod companies_key_accounts;
pub use companies_key_accounts::*;
mod xbrl_financial_statements;
pub use xbrl_financial_statements::*;
mod company_financial_statements;
pub use company_financial_statements::*;
mod xbrl_taxonomy;
pub use xbrl_taxonomy::*;
mod company_financial_indicators;
pub use company_financial_indicators::*;
mod companies_financial_indicators;
pub use companies_financial_indicators::*;
