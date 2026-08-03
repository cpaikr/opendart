use super::*;

/// Inputs for logical operation `DS005-2020023`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct PaidInCapitalIncreaseDecisionsInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl PaidInCapitalIncreaseDecisionsInput {
    /// Creates inputs from all required source-shaped values.
    #[must_use]
    pub fn new(company_code: String, start_date: String, end_date: String) -> Self {
        Self {
            company_code,
            start_date,
            end_date,
        }
    }

    fn parameters(
        &self,
        operation: OperationIdentity,
    ) -> Result<Vec<crate::request::QueryParameter<'_>>, PrepareError> {
        let mut query = Query::new(operation);
        let value = CompanyCode::new(operation, "corp_code", &self.company_code)?;
        query.required("corp_code", value.as_str())?;
        let value = CompactDate::new(operation, "bgn_de", &self.start_date)?;
        query.required("bgn_de", value.as_str())?;
        let value = CompactDate::new(operation, "end_de", &self.end_date)?;
        query.required("end_de", value.as_str())?;
        Ok(query.finish())
    }

    /// Prepares physical operation `get_piicDecsn_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<PaidInCapitalIncreaseDecisionsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_piicDecsn_json", "DS005-2020023");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/piicDecsn.json", operation, &parameters),
            decode_paid_in_capital_increase_decisions_json_response,
        ))
    }

    /// Prepares physical operation `get_piicDecsn_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<PaidInCapitalIncreaseDecisionsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_piicDecsn_xml", "DS005-2020023");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/piicDecsn.xml", operation, &parameters, "result"),
            decode_paid_in_capital_increase_decisions_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct PaidInCapitalIncreaseDecision<'a> {
    source: &'a SourceValue,
}

impl<'a> PaidInCapitalIncreaseDecision<'a> {
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

    /// Returns source field `bfic_tisstk_estk` when present.
    #[must_use]
    pub fn other_shares_issued_before_increase(&self) -> Option<&SourceValue> {
        self.source.get("bfic_tisstk_estk")
    }

    /// Returns source field `bfic_tisstk_ostk` when present.
    #[must_use]
    pub fn common_shares_issued_before_increase(&self) -> Option<&SourceValue> {
        self.source.get("bfic_tisstk_ostk")
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

    /// Returns source field `fdpp_bsninh` when present.
    #[must_use]
    pub fn business_acquisition_funds(&self) -> Option<&SourceValue> {
        self.source.get("fdpp_bsninh")
    }

    /// Returns source field `fdpp_dtrp` when present.
    #[must_use]
    pub fn debt_repayment_funds(&self) -> Option<&SourceValue> {
        self.source.get("fdpp_dtrp")
    }

    /// Returns source field `fdpp_etc` when present.
    #[must_use]
    pub fn other_funds(&self) -> Option<&SourceValue> {
        self.source.get("fdpp_etc")
    }

    /// Returns source field `fdpp_fclt` when present.
    #[must_use]
    pub fn facility_funds(&self) -> Option<&SourceValue> {
        self.source.get("fdpp_fclt")
    }

    /// Returns source field `fdpp_ocsa` when present.
    #[must_use]
    pub fn other_corporation_securities_acquisition_funds(&self) -> Option<&SourceValue> {
        self.source.get("fdpp_ocsa")
    }

    /// Returns source field `fdpp_op` when present.
    #[must_use]
    pub fn operating_funds(&self) -> Option<&SourceValue> {
        self.source.get("fdpp_op")
    }

    /// Returns source field `fv_ps` when present.
    #[must_use]
    pub fn par_value_per_share(&self) -> Option<&SourceValue> {
        self.source.get("fv_ps")
    }

    /// Returns source field `ic_mthn` when present.
    #[must_use]
    pub fn capital_increase_method(&self) -> Option<&SourceValue> {
        self.source.get("ic_mthn")
    }

    /// Returns source field `nstk_estk_cnt` when present.
    #[must_use]
    pub fn new_other_share_count(&self) -> Option<&SourceValue> {
        self.source.get("nstk_estk_cnt")
    }

    /// Returns source field `nstk_ostk_cnt` when present.
    #[must_use]
    pub fn new_common_share_count(&self) -> Option<&SourceValue> {
        self.source.get("nstk_ostk_cnt")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `ssl_at` when present.
    #[must_use]
    pub fn short_sale_status(&self) -> Option<&SourceValue> {
        self.source.get("ssl_at")
    }

    /// Returns source field `ssl_bgd` when present.
    #[must_use]
    pub fn short_sale_start_date(&self) -> Option<&SourceValue> {
        self.source.get("ssl_bgd")
    }

    /// Returns source field `ssl_edd` when present.
    #[must_use]
    pub fn short_sale_end_date(&self) -> Option<&SourceValue> {
        self.source.get("ssl_edd")
    }
}

/// Opaque response for physical operation `get_piicDecsn_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct PaidInCapitalIncreaseDecisionsJsonResponse {
    source: SourceValue,
}

impl PaidInCapitalIncreaseDecisionsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = PaidInCapitalIncreaseDecision<'_>> + '_ {
        items(&self.source, "list", false).map(PaidInCapitalIncreaseDecision::new)
    }
}

fn decode_paid_in_capital_increase_decisions_json_response(
    source: SourceValue,
) -> Result<PaidInCapitalIncreaseDecisionsJsonResponse, ResponseDecodeError> {
    decode_paid_in_capital_increase_decisions_json_response_root(&source, "$".to_owned())?;
    Ok(PaidInCapitalIncreaseDecisionsJsonResponse { source })
}

fn decode_paid_in_capital_increase_decisions_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_paid_in_capital_increase_decisions_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_paid_in_capital_increase_decisions_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_paid_in_capital_increase_decisions_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_paid_in_capital_increase_decisions_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_piicDecsn_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct PaidInCapitalIncreaseDecisionsXmlResponse {
    source: SourceValue,
}

impl PaidInCapitalIncreaseDecisionsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = PaidInCapitalIncreaseDecision<'_>> + '_ {
        items(&self.source, "list", true).map(PaidInCapitalIncreaseDecision::new)
    }
}

fn decode_paid_in_capital_increase_decisions_xml_response(
    source: SourceValue,
) -> Result<PaidInCapitalIncreaseDecisionsXmlResponse, ResponseDecodeError> {
    decode_paid_in_capital_increase_decisions_xml_response_root(&source, "$".to_owned())?;
    Ok(PaidInCapitalIncreaseDecisionsXmlResponse { source })
}

fn decode_paid_in_capital_increase_decisions_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_paid_in_capital_increase_decisions_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_paid_in_capital_increase_decisions_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_paid_in_capital_increase_decisions_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_paid_in_capital_increase_decisions_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
