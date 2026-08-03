use super::*;

/// Inputs for logical operation `DS003-2022001`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct CompanyFinancialIndicatorsInput {
    company_code: String,
    business_year: String,
    report_code: String,
    indicator_category: String,
}

impl CompanyFinancialIndicatorsInput {
    /// Creates inputs from all required source-shaped values.
    #[must_use]
    pub fn new(
        company_code: String,
        business_year: String,
        report_code: String,
        indicator_category: String,
    ) -> Self {
        Self {
            company_code,
            business_year,
            report_code,
            indicator_category,
        }
    }

    fn parameters(
        &self,
        operation: OperationIdentity,
    ) -> Result<Vec<crate::request::QueryParameter<'_>>, PrepareError> {
        let mut query = Query::new(operation);
        let value = CompanyCode::new(operation, "corp_code", &self.company_code)?;
        query.required("corp_code", value.as_str())?;
        let value = BusinessYear::new(operation, "bsns_year", &self.business_year)?;
        query.required("bsns_year", value.as_str())?;
        let value = ReportCode::new(operation, "reprt_code", &self.report_code)?;
        query.required("reprt_code", value.as_str())?;
        query.required("idx_cl_code", &self.indicator_category)?;
        let value = self.indicator_category.as_str();
        crate::validation::require_allowed(
            operation,
            "idx_cl_code",
            value,
            &["M210000", "M220000", "M230000", "M240000"],
        )?;
        Ok(query.finish())
    }

    /// Prepares physical operation `get_fnlttSinglIndx_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<CompanyFinancialIndicatorsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_fnlttSinglIndx_json", "DS003-2022001");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/fnlttSinglIndx.json", operation, &parameters),
            decode_company_financial_indicators_json_response,
        ))
    }

    /// Prepares physical operation `get_fnlttSinglIndx_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<CompanyFinancialIndicatorsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_fnlttSinglIndx_xml", "DS003-2022001");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/fnlttSinglIndx.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_company_financial_indicators_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct CompanyFinancialIndicator<'a> {
    source: &'a SourceValue,
}

impl<'a> CompanyFinancialIndicator<'a> {
    fn new(source: &'a SourceValue) -> Self {
        Self { source }
    }

    /// Returns the complete normalized source object for this view.
    #[must_use]
    pub const fn source(&self) -> &'a SourceValue {
        self.source
    }

    /// Returns a field by its exact source name.
    #[must_use]
    pub fn field(&self, name: &str) -> Option<&'a SourceValue> {
        self.source.get(name)
    }

    /// Returns source field `bsns_year` when present.
    #[must_use]
    pub fn business_year(&self) -> Option<&SourceValue> {
        self.source.get("bsns_year")
    }

    /// Returns source field `corp_code` when present.
    #[must_use]
    pub fn company_code(&self) -> Option<&SourceValue> {
        self.source.get("corp_code")
    }

    /// Returns source field `idx_cl_code` when present.
    #[must_use]
    pub fn indicator_category_code(&self) -> Option<&SourceValue> {
        self.source.get("idx_cl_code")
    }

    /// Returns source field `idx_cl_nm` when present.
    #[must_use]
    pub fn indicator_category_name(&self) -> Option<&SourceValue> {
        self.source.get("idx_cl_nm")
    }

    /// Returns source field `idx_code` when present.
    #[must_use]
    pub fn indicator_code(&self) -> Option<&SourceValue> {
        self.source.get("idx_code")
    }

    /// Returns source field `idx_nm` when present.
    #[must_use]
    pub fn indicator_name(&self) -> Option<&SourceValue> {
        self.source.get("idx_nm")
    }

    /// Returns source field `idx_val` when present.
    #[must_use]
    pub fn indicator_value(&self) -> Option<&SourceValue> {
        self.source.get("idx_val")
    }

    /// Returns source field `reprt_code` when present.
    #[must_use]
    pub fn report_code(&self) -> Option<&SourceValue> {
        self.source.get("reprt_code")
    }

    /// Returns source field `stlm_dt` when present.
    #[must_use]
    pub fn fiscal_year_end(&self) -> Option<&SourceValue> {
        self.source.get("stlm_dt")
    }

    /// Returns source field `stock_code` when present.
    #[must_use]
    pub fn stock_code(&self) -> Option<&SourceValue> {
        self.source.get("stock_code")
    }
}

/// Opaque response for physical operation `get_fnlttSinglIndx_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct CompanyFinancialIndicatorsJsonResponse {
    source: SourceValue,
}

impl CompanyFinancialIndicatorsJsonResponse {
    /// Returns the complete normalized source object.
    #[must_use]
    pub const fn source(&self) -> &SourceValue {
        &self.source
    }

    /// Returns the open source status when present and string-shaped.
    #[must_use]
    pub fn status(&self) -> Option<SourceStatus> {
        source_status(&self.source)
    }

    /// Returns the source message when present.
    #[must_use]
    pub fn message(&self) -> Option<&SourceValue> {
        self.source.get("message")
    }

    /// Returns a field by its exact source name.
    #[must_use]
    pub fn field(&self, name: &str) -> Option<&SourceValue> {
        self.source.get(name)
    }

    /// Iterates the reviewed `$.list[]` item views.
    pub fn items(&self) -> impl Iterator<Item = CompanyFinancialIndicator<'_>> + '_ {
        items(&self.source, "list", false).map(CompanyFinancialIndicator::new)
    }
}

fn decode_company_financial_indicators_json_response(
    source: SourceValue,
) -> Result<CompanyFinancialIndicatorsJsonResponse, ResponseDecodeError> {
    decode_company_financial_indicators_json_response_root(&source, "$".to_owned())?;
    Ok(CompanyFinancialIndicatorsJsonResponse { source })
}

fn decode_company_financial_indicators_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_company_financial_indicators_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_company_financial_indicators_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_company_financial_indicators_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_company_financial_indicators_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_fnlttSinglIndx_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct CompanyFinancialIndicatorsXmlResponse {
    source: SourceValue,
}

impl CompanyFinancialIndicatorsXmlResponse {
    /// Returns the complete normalized source object.
    #[must_use]
    pub const fn source(&self) -> &SourceValue {
        &self.source
    }

    /// Returns the open source status when present and string-shaped.
    #[must_use]
    pub fn status(&self) -> Option<SourceStatus> {
        source_status(&self.source)
    }

    /// Returns the source message when present.
    #[must_use]
    pub fn message(&self) -> Option<&SourceValue> {
        self.source.get("message")
    }

    /// Returns a field by its exact source name.
    #[must_use]
    pub fn field(&self, name: &str) -> Option<&SourceValue> {
        self.source.get(name)
    }

    /// Iterates the reviewed `$.list[]` item views.
    pub fn items(&self) -> impl Iterator<Item = CompanyFinancialIndicator<'_>> + '_ {
        items(&self.source, "list", true).map(CompanyFinancialIndicator::new)
    }
}

fn decode_company_financial_indicators_xml_response(
    source: SourceValue,
) -> Result<CompanyFinancialIndicatorsXmlResponse, ResponseDecodeError> {
    decode_company_financial_indicators_xml_response_root(&source, "$".to_owned())?;
    Ok(CompanyFinancialIndicatorsXmlResponse { source })
}

fn decode_company_financial_indicators_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_company_financial_indicators_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_company_financial_indicators_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_company_financial_indicators_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_company_financial_indicators_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
