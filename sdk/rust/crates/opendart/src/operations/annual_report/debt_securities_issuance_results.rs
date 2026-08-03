use super::*;

/// Inputs for logical operation `DS002-2020003`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct DebtSecuritiesIssuanceResultsInput {
    company_code: String,
    business_year: String,
    report_code: String,
}

impl DebtSecuritiesIssuanceResultsInput {
    /// Creates inputs from all required source-shaped values.
    #[must_use]
    pub fn new(company_code: String, business_year: String, report_code: String) -> Self {
        Self {
            company_code,
            business_year,
            report_code,
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
        Ok(query.finish())
    }

    /// Prepares physical operation `get_detScritsIsuAcmslt_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<DebtSecuritiesIssuanceResultsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_detScritsIsuAcmslt_json", "DS002-2020003");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/detScritsIsuAcmslt.json", operation, &parameters),
            decode_debt_securities_issuance_results_json_response,
        ))
    }

    /// Prepares physical operation `get_detScritsIsuAcmslt_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<DebtSecuritiesIssuanceResultsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_detScritsIsuAcmslt_xml", "DS002-2020003");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/detScritsIsuAcmslt.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_debt_securities_issuance_results_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct DebtSecuritiesIssuance<'a> {
    source: &'a SourceValue,
}

impl<'a> DebtSecuritiesIssuance<'a> {
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

    /// Returns source field `corp_cls` when present.
    #[must_use]
    pub fn company_class(&self) -> Option<&SourceValue> {
        self.source.get("corp_cls")
    }

    /// Returns source field `corp_code` when present.
    #[must_use]
    pub fn company_code(&self) -> Option<&SourceValue> {
        self.source.get("corp_code")
    }

    /// Returns source field `corp_name` when present.
    #[must_use]
    pub fn company_name(&self) -> Option<&SourceValue> {
        self.source.get("corp_name")
    }

    /// Returns source field `evl_grad_instt` when present.
    #[must_use]
    pub fn credit_rating_and_agency(&self) -> Option<&SourceValue> {
        self.source.get("evl_grad_instt")
    }

    /// Returns source field `facvalu_totamt` when present.
    #[must_use]
    pub fn total_face_value(&self) -> Option<&SourceValue> {
        self.source.get("facvalu_totamt")
    }

    /// Returns source field `intrt` when present.
    #[must_use]
    pub fn interest_rate(&self) -> Option<&SourceValue> {
        self.source.get("intrt")
    }

    /// Returns source field `isu_cmpny` when present.
    #[must_use]
    pub fn issuer_name(&self) -> Option<&SourceValue> {
        self.source.get("isu_cmpny")
    }

    /// Returns source field `isu_de` when present.
    #[must_use]
    pub fn issuance_date(&self) -> Option<&SourceValue> {
        self.source.get("isu_de")
    }

    /// Returns source field `isu_mth_nm` when present.
    #[must_use]
    pub fn issuance_method(&self) -> Option<&SourceValue> {
        self.source.get("isu_mth_nm")
    }

    /// Returns source field `mngt_cmpny` when present.
    #[must_use]
    pub fn lead_manager(&self) -> Option<&SourceValue> {
        self.source.get("mngt_cmpny")
    }

    /// Returns source field `mtd` when present.
    #[must_use]
    pub fn maturity_date(&self) -> Option<&SourceValue> {
        self.source.get("mtd")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `repy_at` when present.
    #[must_use]
    pub fn repayment_status(&self) -> Option<&SourceValue> {
        self.source.get("repy_at")
    }

    /// Returns source field `scrits_knd_nm` when present.
    #[must_use]
    pub fn security_type(&self) -> Option<&SourceValue> {
        self.source.get("scrits_knd_nm")
    }

    /// Returns source field `stlm_dt` when present.
    #[must_use]
    pub fn fiscal_period_end_date(&self) -> Option<&SourceValue> {
        self.source.get("stlm_dt")
    }
}

/// Opaque response for physical operation `get_detScritsIsuAcmslt_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct DebtSecuritiesIssuanceResultsJsonResponse {
    source: SourceValue,
}

impl DebtSecuritiesIssuanceResultsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = DebtSecuritiesIssuance<'_>> + '_ {
        items(&self.source, "list", false).map(DebtSecuritiesIssuance::new)
    }
}

fn decode_debt_securities_issuance_results_json_response(
    source: SourceValue,
) -> Result<DebtSecuritiesIssuanceResultsJsonResponse, ResponseDecodeError> {
    decode_debt_securities_issuance_results_json_response_root(&source, "$".to_owned())?;
    Ok(DebtSecuritiesIssuanceResultsJsonResponse { source })
}

fn decode_debt_securities_issuance_results_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_debt_securities_issuance_results_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_debt_securities_issuance_results_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_debt_securities_issuance_results_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_debt_securities_issuance_results_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_detScritsIsuAcmslt_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct DebtSecuritiesIssuanceResultsXmlResponse {
    source: SourceValue,
}

impl DebtSecuritiesIssuanceResultsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = DebtSecuritiesIssuance<'_>> + '_ {
        items(&self.source, "list", true).map(DebtSecuritiesIssuance::new)
    }
}

fn decode_debt_securities_issuance_results_xml_response(
    source: SourceValue,
) -> Result<DebtSecuritiesIssuanceResultsXmlResponse, ResponseDecodeError> {
    decode_debt_securities_issuance_results_xml_response_root(&source, "$".to_owned())?;
    Ok(DebtSecuritiesIssuanceResultsXmlResponse { source })
}

fn decode_debt_securities_issuance_results_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_debt_securities_issuance_results_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_debt_securities_issuance_results_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_debt_securities_issuance_results_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_debt_securities_issuance_results_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
