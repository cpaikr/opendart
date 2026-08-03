use super::*;

/// Inputs for logical operation `DS001-2019003`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct DisclosureDocumentInput {
    receipt_number: String,
}

impl DisclosureDocumentInput {
    /// Creates inputs from all required source-shaped values.
    #[must_use]
    pub fn new(receipt_number: String) -> Self {
        Self { receipt_number }
    }

    fn parameters(
        &self,
        operation: OperationIdentity,
    ) -> Result<Vec<crate::request::QueryParameter<'_>>, PrepareError> {
        let mut query = Query::new(operation);
        query.required("rcept_no", &self.receipt_number)?;
        Ok(query.finish())
    }

    /// Prepares physical operation `get_document_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_archive(&self) -> Result<PreparedBinaryRequest, PrepareError> {
        let operation = OperationIdentity::new("get_document_xml", "DS001-2019003");
        let parameters = self.parameters(operation)?;
        Ok(PreparedBinaryRequest::new(RequestParts::binary_zip(
            "/api/document.xml",
            operation,
            &parameters,
            "result",
        )))
    }
}
