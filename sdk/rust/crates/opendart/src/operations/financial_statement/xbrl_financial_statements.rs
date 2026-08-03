use super::*;

/// Inputs for logical operation `DS003-2019019`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct XbrlFinancialStatementsInput {
    receipt_number: String,
    report_code: String,
}

impl XbrlFinancialStatementsInput {
    /// Creates inputs from all required source-shaped values.
    #[must_use]
    pub fn new(receipt_number: String, report_code: String) -> Self {
        Self {
            receipt_number,
            report_code,
        }
    }

    fn parameters(
        &self,
        operation: OperationIdentity,
    ) -> Result<Vec<crate::request::QueryParameter<'_>>, PrepareError> {
        let mut query = Query::new(operation);
        query.required("rcept_no", &self.receipt_number)?;
        let value = ReportCode::new(operation, "reprt_code", &self.report_code)?;
        query.required("reprt_code", value.as_str())?;
        Ok(query.finish())
    }

    /// Prepares physical operation `get_fnlttXbrl_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_archive(&self) -> Result<PreparedBinaryRequest, PrepareError> {
        let operation = OperationIdentity::new("get_fnlttXbrl_xml", "DS003-2019019");
        let parameters = self.parameters(operation)?;
        Ok(PreparedBinaryRequest::new(RequestParts::binary_zip(
            "/api/fnlttXbrl.xml",
            operation,
            &parameters,
            "result",
        )))
    }
}
