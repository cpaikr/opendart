use super::*;

/// Inputs for logical operation `DS003-2019020`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct CompanyFinancialStatementsInput {
    company_code: String,
    business_year: String,
    report_code: String,
    financial_statement_scope: String,
}

impl CompanyFinancialStatementsInput {
    /// Creates inputs from all required source-shaped values.
    #[must_use]
    pub fn new(
        company_code: String,
        business_year: String,
        report_code: String,
        financial_statement_scope: String,
    ) -> Self {
        Self {
            company_code,
            business_year,
            report_code,
            financial_statement_scope,
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
        query.required("fs_div", &self.financial_statement_scope)?;
        let value = self.financial_statement_scope.as_str();
        crate::validation::require_allowed(operation, "fs_div", value, &["OFS", "CFS"])?;
        Ok(query.finish())
    }

    /// Prepares physical operation `get_fnlttSinglAcntAll_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<CompanyFinancialStatementsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_fnlttSinglAcntAll_json", "DS003-2019020");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/fnlttSinglAcntAll.json", operation, &parameters),
            decode_company_financial_statements_json_response,
        ))
    }

    /// Prepares physical operation `get_fnlttSinglAcntAll_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<CompanyFinancialStatementsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_fnlttSinglAcntAll_xml", "DS003-2019020");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/fnlttSinglAcntAll.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_company_financial_statements_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct CompanyFinancialStatementLine<'a> {
    source: &'a SourceValue,
}

impl<'a> CompanyFinancialStatementLine<'a> {
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

    /// Returns source field `account_detail` when present.
    #[must_use]
    pub fn account_detail(&self) -> Option<&SourceValue> {
        self.source.get("account_detail")
    }

    /// Returns source field `account_id` when present.
    #[must_use]
    pub fn account_id(&self) -> Option<&SourceValue> {
        self.source.get("account_id")
    }

    /// Returns source field `account_nm` when present.
    #[must_use]
    pub fn account_name(&self) -> Option<&SourceValue> {
        self.source.get("account_nm")
    }

    /// Returns source field `bfefrmtrm_amount` when present.
    #[must_use]
    pub fn two_periods_prior_amount(&self) -> Option<&SourceValue> {
        self.source.get("bfefrmtrm_amount")
    }

    /// Returns source field `bfefrmtrm_nm` when present.
    #[must_use]
    pub fn two_periods_prior_name(&self) -> Option<&SourceValue> {
        self.source.get("bfefrmtrm_nm")
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

    /// Returns source field `currency` when present.
    #[must_use]
    pub fn currency(&self) -> Option<&SourceValue> {
        self.source.get("currency")
    }

    /// Returns source field `frmtrm_add_amount` when present.
    #[must_use]
    pub fn prior_period_cumulative_amount(&self) -> Option<&SourceValue> {
        self.source.get("frmtrm_add_amount")
    }

    /// Returns source field `frmtrm_amount` when present.
    #[must_use]
    pub fn prior_period_amount(&self) -> Option<&SourceValue> {
        self.source.get("frmtrm_amount")
    }

    /// Returns source field `frmtrm_nm` when present.
    #[must_use]
    pub fn prior_period_name(&self) -> Option<&SourceValue> {
        self.source.get("frmtrm_nm")
    }

    /// Returns source field `frmtrm_q_amount` when present.
    #[must_use]
    pub fn prior_interim_period_amount(&self) -> Option<&SourceValue> {
        self.source.get("frmtrm_q_amount")
    }

    /// Returns source field `frmtrm_q_nm` when present.
    #[must_use]
    pub fn prior_interim_period_name(&self) -> Option<&SourceValue> {
        self.source.get("frmtrm_q_nm")
    }

    /// Returns source field `ord` when present.
    #[must_use]
    pub fn display_order(&self) -> Option<&SourceValue> {
        self.source.get("ord")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `reprt_code` when present.
    #[must_use]
    pub fn report_code(&self) -> Option<&SourceValue> {
        self.source.get("reprt_code")
    }

    /// Returns source field `sj_div` when present.
    #[must_use]
    pub fn statement_type(&self) -> Option<&SourceValue> {
        self.source.get("sj_div")
    }

    /// Returns source field `sj_nm` when present.
    #[must_use]
    pub fn statement_name(&self) -> Option<&SourceValue> {
        self.source.get("sj_nm")
    }

    /// Returns source field `thstrm_add_amount` when present.
    #[must_use]
    pub fn current_period_cumulative_amount(&self) -> Option<&SourceValue> {
        self.source.get("thstrm_add_amount")
    }

    /// Returns source field `thstrm_amount` when present.
    #[must_use]
    pub fn current_period_amount(&self) -> Option<&SourceValue> {
        self.source.get("thstrm_amount")
    }

    /// Returns source field `thstrm_nm` when present.
    #[must_use]
    pub fn current_period_name(&self) -> Option<&SourceValue> {
        self.source.get("thstrm_nm")
    }
}

/// Opaque response for physical operation `get_fnlttSinglAcntAll_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct CompanyFinancialStatementsJsonResponse {
    source: SourceValue,
}

impl CompanyFinancialStatementsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = CompanyFinancialStatementLine<'_>> + '_ {
        items(&self.source, "list", false).map(CompanyFinancialStatementLine::new)
    }
}

fn decode_company_financial_statements_json_response(
    source: SourceValue,
) -> Result<CompanyFinancialStatementsJsonResponse, ResponseDecodeError> {
    decode_company_financial_statements_json_response_root(&source, "$".to_owned())?;
    Ok(CompanyFinancialStatementsJsonResponse { source })
}

fn decode_company_financial_statements_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_company_financial_statements_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_company_financial_statements_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_company_financial_statements_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_company_financial_statements_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_fnlttSinglAcntAll_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct CompanyFinancialStatementsXmlResponse {
    source: SourceValue,
}

impl CompanyFinancialStatementsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = CompanyFinancialStatementLine<'_>> + '_ {
        items(&self.source, "list", true).map(CompanyFinancialStatementLine::new)
    }
}

fn decode_company_financial_statements_xml_response(
    source: SourceValue,
) -> Result<CompanyFinancialStatementsXmlResponse, ResponseDecodeError> {
    decode_company_financial_statements_xml_response_root(&source, "$".to_owned())?;
    Ok(CompanyFinancialStatementsXmlResponse { source })
}

fn decode_company_financial_statements_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_company_financial_statements_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_company_financial_statements_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_company_financial_statements_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_company_financial_statements_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
