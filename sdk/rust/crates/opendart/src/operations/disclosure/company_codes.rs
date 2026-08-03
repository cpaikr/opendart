use super::*;

/// Inputs for logical operation `DS001-2019018`.
#[derive(Clone, Debug, Default, Eq, PartialEq)]
pub struct CompanyCodesInput {}

impl CompanyCodesInput {
    /// Creates inputs from all required source-shaped values.
    #[must_use]
    pub fn new() -> Self {
        Self {}
    }

    fn parameters(
        &self,
        operation: OperationIdentity,
    ) -> Result<Vec<crate::request::QueryParameter<'_>>, PrepareError> {
        let query = Query::new(operation);
        Ok(query.finish())
    }

    /// Prepares physical operation `get_corpCode_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_archive(&self) -> Result<PreparedBinaryRequest, PrepareError> {
        let operation = OperationIdentity::new("get_corpCode_xml", "DS001-2019018");
        let parameters = self.parameters(operation)?;
        Ok(PreparedBinaryRequest::new(RequestParts::binary_zip(
            "/api/corpCode.xml",
            operation,
            &parameters,
            "result",
        )))
    }
}
