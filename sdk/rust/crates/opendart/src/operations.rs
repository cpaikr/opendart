//! Representative handwritten operations used to review the public contract.
//!
//! The deterministic generator replaces this representative inventory with the complete
//! canonical operation surface in the next implementation slice.

use crate::request::{QueryParameter, QueryValue};
use crate::{OperationIdentity, PrepareError, PreparedRequest, Representation};

const JSON_ONLY: &[Representation] = &[Representation::Json];
const XML_ONLY: &[Representation] = &[Representation::Xml];
const ZIP_OR_XML: &[Representation] = &[Representation::Zip, Representation::Xml];

/// The company-overview logical operation with selectable JSON or XML output.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct CompanyOverview {
    corp_code: String,
}

impl CompanyOverview {
    /// Creates a company-overview input. Contract validation occurs during preparation.
    pub fn new(corp_code: impl Into<String>) -> Self {
        Self {
            corp_code: corp_code.into(),
        }
    }

    /// Returns the supplied company code.
    #[must_use]
    pub fn corp_code(&self) -> &str {
        &self.corp_code
    }

    /// Prepares one physical representation without performing I/O.
    pub fn prepare(&self, representation: Representation) -> Result<PreparedRequest, PrepareError> {
        let (path, identity, expected) = match representation {
            Representation::Json => (
                "/api/company.json",
                OperationIdentity::new("get_company_json", "DS001-2019002"),
                JSON_ONLY,
            ),
            Representation::Xml => (
                "/api/company.xml",
                OperationIdentity::new("get_company_xml", "DS001-2019002"),
                XML_ONLY,
            ),
            _ => {
                return Err(PrepareError::UnsupportedRepresentation {
                    logical_operation: "DS001-2019002",
                    representation,
                });
            }
        };
        require_nonempty(identity, "corp_code", &self.corp_code)?;
        Ok(PreparedRequest::new(
            path,
            identity,
            &[QueryParameter {
                name: "corp_code",
                value: QueryValue::Scalar(&self.corp_code),
            }],
            expected,
        ))
    }
}

/// The auditor-opinion logical operation with selectable JSON or XML output.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct AuditorOpinion {
    corp_code: String,
    business_year: String,
    report_code: String,
}

impl AuditorOpinion {
    /// Creates an auditor-opinion input. Contract validation occurs during preparation.
    pub fn new(
        corp_code: impl Into<String>,
        business_year: impl Into<String>,
        report_code: impl Into<String>,
    ) -> Self {
        Self {
            corp_code: corp_code.into(),
            business_year: business_year.into(),
            report_code: report_code.into(),
        }
    }

    /// Prepares one physical representation without performing I/O.
    pub fn prepare(&self, representation: Representation) -> Result<PreparedRequest, PrepareError> {
        let (path, identity, expected) = match representation {
            Representation::Json => (
                "/api/accnutAdtorNmNdAdtOpinion.json",
                OperationIdentity::new("get_accnutAdtorNmNdAdtOpinion_json", "DS002-2020009"),
                JSON_ONLY,
            ),
            Representation::Xml => (
                "/api/accnutAdtorNmNdAdtOpinion.xml",
                OperationIdentity::new("get_accnutAdtorNmNdAdtOpinion_xml", "DS002-2020009"),
                XML_ONLY,
            ),
            _ => {
                return Err(PrepareError::UnsupportedRepresentation {
                    logical_operation: "DS002-2020009",
                    representation,
                });
            }
        };
        require_nonempty(identity, "corp_code", &self.corp_code)?;
        require_nonempty(identity, "bsns_year", &self.business_year)?;
        require_nonempty(identity, "reprt_code", &self.report_code)?;
        Ok(PreparedRequest::new(
            path,
            identity,
            &[
                QueryParameter {
                    name: "corp_code",
                    value: QueryValue::Scalar(&self.corp_code),
                },
                QueryParameter {
                    name: "bsns_year",
                    value: QueryValue::Scalar(&self.business_year),
                },
                QueryParameter {
                    name: "reprt_code",
                    value: QueryValue::Scalar(&self.report_code),
                },
            ],
            expected,
        ))
    }
}

/// The fixed ZIP company-code inventory operation.
#[derive(Clone, Copy, Debug, Default, Eq, PartialEq)]
pub struct CorpCodeFile;

impl CorpCodeFile {
    /// Creates the parameter-free company-code inventory input.
    #[must_use]
    pub const fn new() -> Self {
        Self
    }

    /// Prepares the fixed ZIP operation without performing I/O.
    pub fn prepare(&self, representation: Representation) -> Result<PreparedRequest, PrepareError> {
        let identity = OperationIdentity::new("get_corpCode_xml", "DS001-2019018");
        if representation != Representation::Zip {
            return Err(PrepareError::UnsupportedRepresentation {
                logical_operation: identity.logical(),
                representation,
            });
        }
        Ok(PreparedRequest::new(
            "/api/corpCode.xml",
            identity,
            &[],
            ZIP_OR_XML,
        ))
    }
}

/// The multi-company major-accounts operation with explicit canonical cardinality.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct MultiCompanyAccounts {
    corp_codes: Vec<String>,
    business_year: String,
    report_code: String,
}

impl MultiCompanyAccounts {
    /// Creates a multi-company input. Contract validation occurs during preparation.
    pub fn new<I, S>(
        corp_codes: I,
        business_year: impl Into<String>,
        report_code: impl Into<String>,
    ) -> Self
    where
        I: IntoIterator<Item = S>,
        S: Into<String>,
    {
        Self {
            corp_codes: corp_codes.into_iter().map(Into::into).collect(),
            business_year: business_year.into(),
            report_code: report_code.into(),
        }
    }

    /// Returns the supplied company codes in serialization order.
    #[must_use]
    pub fn corp_codes(&self) -> &[String] {
        &self.corp_codes
    }

    /// Prepares one physical representation without performing I/O.
    pub fn prepare(&self, representation: Representation) -> Result<PreparedRequest, PrepareError> {
        let (path, identity, expected) = match representation {
            Representation::Json => (
                "/api/fnlttMultiAcnt.json",
                OperationIdentity::new("get_fnlttMultiAcnt_json", "DS003-2019017"),
                JSON_ONLY,
            ),
            Representation::Xml => (
                "/api/fnlttMultiAcnt.xml",
                OperationIdentity::new("get_fnlttMultiAcnt_xml", "DS003-2019017"),
                XML_ONLY,
            ),
            _ => {
                return Err(PrepareError::UnsupportedRepresentation {
                    logical_operation: "DS003-2019017",
                    representation,
                });
            }
        };
        if !(1..=100).contains(&self.corp_codes.len()) {
            return Err(PrepareError::InvalidCardinality {
                operation: identity,
                parameter: "corp_code",
                minimum: 1,
                maximum: 100,
            });
        }
        for corp_code in &self.corp_codes {
            require_nonempty(identity, "corp_code", corp_code)?;
        }
        require_nonempty(identity, "bsns_year", &self.business_year)?;
        require_nonempty(identity, "reprt_code", &self.report_code)?;
        Ok(PreparedRequest::new(
            path,
            identity,
            &[
                QueryParameter {
                    name: "corp_code",
                    value: QueryValue::CommaSeparated(&self.corp_codes),
                },
                QueryParameter {
                    name: "bsns_year",
                    value: QueryValue::Scalar(&self.business_year),
                },
                QueryParameter {
                    name: "reprt_code",
                    value: QueryValue::Scalar(&self.report_code),
                },
            ],
            expected,
        ))
    }
}

fn require_nonempty(
    operation: OperationIdentity,
    parameter: &'static str,
    value: &str,
) -> Result<(), PrepareError> {
    if value.is_empty() {
        return Err(PrepareError::MissingInput {
            operation,
            parameter,
        });
    }
    Ok(())
}
