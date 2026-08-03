use super::*;

/// Inputs for logical operation `DS005-2020038`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct TreasuryStockAcquisitionDecisionsInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl TreasuryStockAcquisitionDecisionsInput {
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

    /// Prepares physical operation `get_tsstkAqDecsn_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<TreasuryStockAcquisitionDecisionsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_tsstkAqDecsn_json", "DS005-2020038");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/tsstkAqDecsn.json", operation, &parameters),
            decode_treasury_stock_acquisition_decisions_json_response,
        ))
    }

    /// Prepares physical operation `get_tsstkAqDecsn_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<TreasuryStockAcquisitionDecisionsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_tsstkAqDecsn_xml", "DS005-2020038");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/tsstkAqDecsn.xml", operation, &parameters, "result"),
            decode_treasury_stock_acquisition_decisions_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct TreasuryStockAcquisitionDecision<'a> {
    source: &'a SourceValue,
}

impl<'a> TreasuryStockAcquisitionDecision<'a> {
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

    /// Returns source field `adt_a_atn` when present.
    #[must_use]
    pub fn auditor_attendance_status(&self) -> Option<&SourceValue> {
        self.source.get("adt_a_atn")
    }

    /// Returns source field `aq_dd` when present.
    #[must_use]
    pub fn acquisition_decision_date(&self) -> Option<&SourceValue> {
        self.source.get("aq_dd")
    }

    /// Returns source field `aq_mth` when present.
    #[must_use]
    pub fn acquisition_method(&self) -> Option<&SourceValue> {
        self.source.get("aq_mth")
    }

    /// Returns source field `aq_pp` when present.
    #[must_use]
    pub fn acquisition_purpose(&self) -> Option<&SourceValue> {
        self.source.get("aq_pp")
    }

    /// Returns source field `aq_wtn_div_estk` when present.
    #[must_use]
    pub fn pre_acquisition_distributable_profit_other_share_count(&self) -> Option<&SourceValue> {
        self.source.get("aq_wtn_div_estk")
    }

    /// Returns source field `aq_wtn_div_estk_rt` when present.
    #[must_use]
    pub fn pre_acquisition_distributable_profit_other_share_ratio(&self) -> Option<&SourceValue> {
        self.source.get("aq_wtn_div_estk_rt")
    }

    /// Returns source field `aq_wtn_div_ostk` when present.
    #[must_use]
    pub fn pre_acquisition_distributable_profit_common_share_count(&self) -> Option<&SourceValue> {
        self.source.get("aq_wtn_div_ostk")
    }

    /// Returns source field `aq_wtn_div_ostk_rt` when present.
    #[must_use]
    pub fn pre_acquisition_distributable_profit_common_share_ratio(&self) -> Option<&SourceValue> {
        self.source.get("aq_wtn_div_ostk_rt")
    }

    /// Returns source field `aqexpd_bgd` when present.
    #[must_use]
    pub fn expected_acquisition_start_date(&self) -> Option<&SourceValue> {
        self.source.get("aqexpd_bgd")
    }

    /// Returns source field `aqexpd_edd` when present.
    #[must_use]
    pub fn expected_acquisition_end_date(&self) -> Option<&SourceValue> {
        self.source.get("aqexpd_edd")
    }

    /// Returns source field `aqpln_prc_estk` when present.
    #[must_use]
    pub fn planned_other_share_acquisition_amount(&self) -> Option<&SourceValue> {
        self.source.get("aqpln_prc_estk")
    }

    /// Returns source field `aqpln_prc_ostk` when present.
    #[must_use]
    pub fn planned_common_share_acquisition_amount(&self) -> Option<&SourceValue> {
        self.source.get("aqpln_prc_ostk")
    }

    /// Returns source field `aqpln_stk_estk` when present.
    #[must_use]
    pub fn planned_other_share_acquisition_count(&self) -> Option<&SourceValue> {
        self.source.get("aqpln_stk_estk")
    }

    /// Returns source field `aqpln_stk_ostk` when present.
    #[must_use]
    pub fn planned_common_share_acquisition_count(&self) -> Option<&SourceValue> {
        self.source.get("aqpln_stk_ostk")
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

    /// Returns source field `cs_iv_bk` when present.
    #[must_use]
    pub fn entrusted_investment_broker(&self) -> Option<&SourceValue> {
        self.source.get("cs_iv_bk")
    }

    /// Returns source field `d1_prodlm_estk` when present.
    #[must_use]
    pub fn daily_other_share_purchase_order_limit(&self) -> Option<&SourceValue> {
        self.source.get("d1_prodlm_estk")
    }

    /// Returns source field `d1_prodlm_ostk` when present.
    #[must_use]
    pub fn daily_common_share_purchase_order_limit(&self) -> Option<&SourceValue> {
        self.source.get("d1_prodlm_ostk")
    }

    /// Returns source field `eaq_estk` when present.
    #[must_use]
    pub fn pre_acquisition_other_other_share_count(&self) -> Option<&SourceValue> {
        self.source.get("eaq_estk")
    }

    /// Returns source field `eaq_estk_rt` when present.
    #[must_use]
    pub fn pre_acquisition_other_other_share_ratio(&self) -> Option<&SourceValue> {
        self.source.get("eaq_estk_rt")
    }

    /// Returns source field `eaq_ostk` when present.
    #[must_use]
    pub fn pre_acquisition_other_common_share_count(&self) -> Option<&SourceValue> {
        self.source.get("eaq_ostk")
    }

    /// Returns source field `eaq_ostk_rt` when present.
    #[must_use]
    pub fn pre_acquisition_other_common_share_ratio(&self) -> Option<&SourceValue> {
        self.source.get("eaq_ostk_rt")
    }

    /// Returns source field `hdexpd_bgd` when present.
    #[must_use]
    pub fn expected_holding_start_date(&self) -> Option<&SourceValue> {
        self.source.get("hdexpd_bgd")
    }

    /// Returns source field `hdexpd_edd` when present.
    #[must_use]
    pub fn expected_holding_end_date(&self) -> Option<&SourceValue> {
        self.source.get("hdexpd_edd")
    }

    /// Returns source field `od_a_at_b` when present.
    #[must_use]
    pub fn outside_directors_absent_count(&self) -> Option<&SourceValue> {
        self.source.get("od_a_at_b")
    }

    /// Returns source field `od_a_at_t` when present.
    #[must_use]
    pub fn outside_directors_present_count(&self) -> Option<&SourceValue> {
        self.source.get("od_a_at_t")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }
}

/// Opaque response for physical operation `get_tsstkAqDecsn_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct TreasuryStockAcquisitionDecisionsJsonResponse {
    source: SourceValue,
}

impl TreasuryStockAcquisitionDecisionsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = TreasuryStockAcquisitionDecision<'_>> + '_ {
        items(&self.source, "list", false).map(TreasuryStockAcquisitionDecision::new)
    }
}

fn decode_treasury_stock_acquisition_decisions_json_response(
    source: SourceValue,
) -> Result<TreasuryStockAcquisitionDecisionsJsonResponse, ResponseDecodeError> {
    decode_treasury_stock_acquisition_decisions_json_response_root(&source, "$".to_owned())?;
    Ok(TreasuryStockAcquisitionDecisionsJsonResponse { source })
}

fn decode_treasury_stock_acquisition_decisions_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_treasury_stock_acquisition_decisions_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_treasury_stock_acquisition_decisions_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_treasury_stock_acquisition_decisions_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_treasury_stock_acquisition_decisions_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_tsstkAqDecsn_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct TreasuryStockAcquisitionDecisionsXmlResponse {
    source: SourceValue,
}

impl TreasuryStockAcquisitionDecisionsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = TreasuryStockAcquisitionDecision<'_>> + '_ {
        items(&self.source, "list", true).map(TreasuryStockAcquisitionDecision::new)
    }
}

fn decode_treasury_stock_acquisition_decisions_xml_response(
    source: SourceValue,
) -> Result<TreasuryStockAcquisitionDecisionsXmlResponse, ResponseDecodeError> {
    decode_treasury_stock_acquisition_decisions_xml_response_root(&source, "$".to_owned())?;
    Ok(TreasuryStockAcquisitionDecisionsXmlResponse { source })
}

fn decode_treasury_stock_acquisition_decisions_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_treasury_stock_acquisition_decisions_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_treasury_stock_acquisition_decisions_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_treasury_stock_acquisition_decisions_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_treasury_stock_acquisition_decisions_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
