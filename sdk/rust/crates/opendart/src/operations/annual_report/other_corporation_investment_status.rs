use super::*;

/// Inputs for logical operation `DS002-2019015`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct OtherCorporationInvestmentStatusInput {
    company_code: String,
    business_year: String,
    report_code: String,
}

impl OtherCorporationInvestmentStatusInput {
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

    /// Prepares physical operation `get_otrCprInvstmntSttus_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<OtherCorporationInvestmentStatusJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_otrCprInvstmntSttus_json", "DS002-2019015");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/otrCprInvstmntSttus.json", operation, &parameters),
            decode_other_corporation_investment_status_json_response,
        ))
    }

    /// Prepares physical operation `get_otrCprInvstmntSttus_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<OtherCorporationInvestmentStatusXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_otrCprInvstmntSttus_xml", "DS002-2019015");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/otrCprInvstmntSttus.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_other_corporation_investment_status_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct OtherCorporationInvestment<'a> {
    source: &'a SourceValue,
}

impl<'a> OtherCorporationInvestment<'a> {
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

    /// Returns source field `bsis_blce_acntbk_amount` when present.
    #[must_use]
    pub fn opening_book_value(&self) -> Option<&SourceValue> {
        self.source.get("bsis_blce_acntbk_amount")
    }

    /// Returns source field `bsis_blce_qota_rt` when present.
    #[must_use]
    pub fn opening_ownership_percentage(&self) -> Option<&SourceValue> {
        self.source.get("bsis_blce_qota_rt")
    }

    /// Returns source field `bsis_blce_qy` when present.
    #[must_use]
    pub fn opening_share_count(&self) -> Option<&SourceValue> {
        self.source.get("bsis_blce_qy")
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

    /// Returns source field `frst_acqs_amount` when present.
    #[must_use]
    pub fn initial_acquisition_amount(&self) -> Option<&SourceValue> {
        self.source.get("frst_acqs_amount")
    }

    /// Returns source field `frst_acqs_de` when present.
    #[must_use]
    pub fn initial_acquisition_date(&self) -> Option<&SourceValue> {
        self.source.get("frst_acqs_de")
    }

    /// Returns source field `incrs_dcrs_acqs_dsps_amount` when present.
    #[must_use]
    pub fn acquisition_disposal_amount_change(&self) -> Option<&SourceValue> {
        self.source.get("incrs_dcrs_acqs_dsps_amount")
    }

    /// Returns source field `incrs_dcrs_acqs_dsps_qy` when present.
    #[must_use]
    pub fn acquisition_disposal_share_count_change(&self) -> Option<&SourceValue> {
        self.source.get("incrs_dcrs_acqs_dsps_qy")
    }

    /// Returns source field `incrs_dcrs_evl_lstmn` when present.
    #[must_use]
    pub fn valuation_gain_or_loss(&self) -> Option<&SourceValue> {
        self.source.get("incrs_dcrs_evl_lstmn")
    }

    /// Returns source field `inv_prm` when present.
    #[must_use]
    pub fn investee_name(&self) -> Option<&SourceValue> {
        self.source.get("inv_prm")
    }

    /// Returns source field `invstmnt_purps` when present.
    #[must_use]
    pub fn investment_purpose(&self) -> Option<&SourceValue> {
        self.source.get("invstmnt_purps")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `recent_bsns_year_fnnr_sttus_thstrm_ntpf` when present.
    #[must_use]
    pub fn recent_year_net_income(&self) -> Option<&SourceValue> {
        self.source.get("recent_bsns_year_fnnr_sttus_thstrm_ntpf")
    }

    /// Returns source field `recent_bsns_year_fnnr_sttus_tot_assets` when present.
    #[must_use]
    pub fn recent_year_total_assets(&self) -> Option<&SourceValue> {
        self.source.get("recent_bsns_year_fnnr_sttus_tot_assets")
    }

    /// Returns source field `stlm_dt` when present.
    #[must_use]
    pub fn fiscal_period_end_date(&self) -> Option<&SourceValue> {
        self.source.get("stlm_dt")
    }

    /// Returns source field `trmend_blce_acntbk_amount` when present.
    #[must_use]
    pub fn closing_book_value(&self) -> Option<&SourceValue> {
        self.source.get("trmend_blce_acntbk_amount")
    }

    /// Returns source field `trmend_blce_qota_rt` when present.
    #[must_use]
    pub fn closing_ownership_percentage(&self) -> Option<&SourceValue> {
        self.source.get("trmend_blce_qota_rt")
    }

    /// Returns source field `trmend_blce_qy` when present.
    #[must_use]
    pub fn closing_share_count(&self) -> Option<&SourceValue> {
        self.source.get("trmend_blce_qy")
    }
}

/// Opaque response for physical operation `get_otrCprInvstmntSttus_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct OtherCorporationInvestmentStatusJsonResponse {
    source: SourceValue,
}

impl OtherCorporationInvestmentStatusJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = OtherCorporationInvestment<'_>> + '_ {
        items(&self.source, "list", false).map(OtherCorporationInvestment::new)
    }
}

fn decode_other_corporation_investment_status_json_response(
    source: SourceValue,
) -> Result<OtherCorporationInvestmentStatusJsonResponse, ResponseDecodeError> {
    decode_other_corporation_investment_status_json_response_root(&source, "$".to_owned())?;
    Ok(OtherCorporationInvestmentStatusJsonResponse { source })
}

fn decode_other_corporation_investment_status_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_other_corporation_investment_status_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_other_corporation_investment_status_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_other_corporation_investment_status_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_other_corporation_investment_status_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_otrCprInvstmntSttus_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct OtherCorporationInvestmentStatusXmlResponse {
    source: SourceValue,
}

impl OtherCorporationInvestmentStatusXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = OtherCorporationInvestment<'_>> + '_ {
        items(&self.source, "list", true).map(OtherCorporationInvestment::new)
    }
}

fn decode_other_corporation_investment_status_xml_response(
    source: SourceValue,
) -> Result<OtherCorporationInvestmentStatusXmlResponse, ResponseDecodeError> {
    decode_other_corporation_investment_status_xml_response_root(&source, "$".to_owned())?;
    Ok(OtherCorporationInvestmentStatusXmlResponse { source })
}

fn decode_other_corporation_investment_status_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_other_corporation_investment_status_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_other_corporation_investment_status_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_other_corporation_investment_status_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_other_corporation_investment_status_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
