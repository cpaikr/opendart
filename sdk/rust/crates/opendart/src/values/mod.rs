//! Private validated borrows for OpenAPI invariants shared by many operations.

use crate::{OperationIdentity, PrepareError};

#[derive(Clone, Copy)]
pub(crate) struct CompanyCode<'a>(&'a str);

impl<'a> CompanyCode<'a> {
    pub(crate) fn new(
        operation: OperationIdentity,
        parameter: &'static str,
        value: &'a str,
    ) -> Result<Self, PrepareError> {
        require_present(operation, parameter, value)?;
        crate::validation::require_length(operation, parameter, value, 8, 8)?;
        crate::validation::require_format(operation, parameter, value, "opendart-corp-code")?;
        Ok(Self(value))
    }

    pub(crate) const fn as_str(self) -> &'a str {
        self.0
    }
}

#[derive(Clone, Copy)]
pub(crate) struct CompactDate<'a>(&'a str);

impl<'a> CompactDate<'a> {
    pub(crate) fn new(
        operation: OperationIdentity,
        parameter: &'static str,
        value: &'a str,
    ) -> Result<Self, PrepareError> {
        require_present(operation, parameter, value)?;
        crate::validation::require_length(operation, parameter, value, 8, 8)?;
        crate::validation::require_format(operation, parameter, value, "opendart-date")?;
        Ok(Self(value))
    }

    pub(crate) const fn as_str(self) -> &'a str {
        self.0
    }
}

#[derive(Clone, Copy)]
pub(crate) struct BusinessYear<'a>(&'a str);

impl<'a> BusinessYear<'a> {
    pub(crate) fn new(
        operation: OperationIdentity,
        parameter: &'static str,
        value: &'a str,
    ) -> Result<Self, PrepareError> {
        require_present(operation, parameter, value)?;
        crate::validation::require_length(operation, parameter, value, 4, 4)?;
        crate::validation::require_format(operation, parameter, value, "opendart-year")?;
        Ok(Self(value))
    }

    pub(crate) const fn as_str(self) -> &'a str {
        self.0
    }
}

#[derive(Clone, Copy)]
pub(crate) struct ReportCode<'a>(&'a str);

impl<'a> ReportCode<'a> {
    pub(crate) fn new(
        operation: OperationIdentity,
        parameter: &'static str,
        value: &'a str,
    ) -> Result<Self, PrepareError> {
        require_present(operation, parameter, value)?;
        crate::validation::require_allowed(
            operation,
            parameter,
            value,
            &["11013", "11012", "11014", "11011"],
        )?;
        Ok(Self(value))
    }

    pub(crate) const fn as_str(self) -> &'a str {
        self.0
    }
}

fn require_present(
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

#[cfg(test)]
mod tests {
    use super::*;

    const OPERATION: OperationIdentity = OperationIdentity::new("physical", "logical");

    #[test]
    fn recurring_values_retain_source_borrows_after_validation() {
        assert_eq!(
            CompanyCode::new(OPERATION, "corp_code", "00126380")
                .unwrap()
                .as_str(),
            "00126380"
        );
        assert_eq!(
            CompactDate::new(OPERATION, "bgn_de", "20260803")
                .unwrap()
                .as_str(),
            "20260803"
        );
        assert_eq!(
            BusinessYear::new(OPERATION, "bsns_year", "2026")
                .unwrap()
                .as_str(),
            "2026"
        );
        assert_eq!(
            ReportCode::new(OPERATION, "reprt_code", "11011")
                .unwrap()
                .as_str(),
            "11011"
        );
    }

    #[test]
    fn recurring_values_preserve_stable_error_categories() {
        for result in [
            CompanyCode::new(OPERATION, "corp_code", "").map(CompanyCode::as_str),
            CompactDate::new(OPERATION, "bgn_de", "").map(CompactDate::as_str),
            BusinessYear::new(OPERATION, "bsns_year", "").map(BusinessYear::as_str),
            ReportCode::new(OPERATION, "reprt_code", "").map(ReportCode::as_str),
        ] {
            assert!(matches!(result, Err(PrepareError::MissingInput { .. })));
        }
        assert!(matches!(
            CompanyCode::new(OPERATION, "corp_code", "123"),
            Err(PrepareError::InvalidLength { .. })
        ));
        assert!(matches!(
            CompactDate::new(OPERATION, "bgn_de", "20230229"),
            Err(PrepareError::InvalidFormat { .. })
        ));
        assert!(matches!(
            BusinessYear::new(OPERATION, "bsns_year", "20x6"),
            Err(PrepareError::InvalidFormat { .. })
        ));
        assert!(matches!(
            ReportCode::new(OPERATION, "reprt_code", "future"),
            Err(PrepareError::InvalidAllowedValue { .. })
        ));
    }
}
