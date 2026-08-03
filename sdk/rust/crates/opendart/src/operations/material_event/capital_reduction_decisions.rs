use super::*;

/// Inputs for logical operation `DS005-2020026`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct CapitalReductionDecisionsInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl CapitalReductionDecisionsInput {
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

    /// Prepares physical operation `get_crDecsn_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<CapitalReductionDecisionsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_crDecsn_json", "DS005-2020026");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/crDecsn.json", operation, &parameters),
            decode_capital_reduction_decisions_json_response,
        ))
    }

    /// Prepares physical operation `get_crDecsn_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<CapitalReductionDecisionsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_crDecsn_xml", "DS005-2020026");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/crDecsn.xml", operation, &parameters, "result"),
            decode_capital_reduction_decisions_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct CapitalReductionDecision<'a> {
    source: &'a SourceValue,
}

impl<'a> CapitalReductionDecision<'a> {
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

    /// Returns source field `atcr_cpt` when present.
    #[must_use]
    pub fn capital_after_reduction(&self) -> Option<&SourceValue> {
        self.source.get("atcr_cpt")
    }

    /// Returns source field `atcr_tisstk_estk` when present.
    #[must_use]
    pub fn other_shares_issued_after_reduction(&self) -> Option<&SourceValue> {
        self.source.get("atcr_tisstk_estk")
    }

    /// Returns source field `atcr_tisstk_ostk` when present.
    #[must_use]
    pub fn common_shares_issued_after_reduction(&self) -> Option<&SourceValue> {
        self.source.get("atcr_tisstk_ostk")
    }

    /// Returns source field `bddd` when present.
    #[must_use]
    pub fn board_resolution_date(&self) -> Option<&SourceValue> {
        self.source.get("bddd")
    }

    /// Returns source field `bfcr_cpt` when present.
    #[must_use]
    pub fn capital_before_reduction(&self) -> Option<&SourceValue> {
        self.source.get("bfcr_cpt")
    }

    /// Returns source field `bfcr_tisstk_estk` when present.
    #[must_use]
    pub fn other_shares_issued_before_reduction(&self) -> Option<&SourceValue> {
        self.source.get("bfcr_tisstk_estk")
    }

    /// Returns source field `bfcr_tisstk_ostk` when present.
    #[must_use]
    pub fn common_shares_issued_before_reduction(&self) -> Option<&SourceValue> {
        self.source.get("bfcr_tisstk_ostk")
    }

    /// Returns source field `cdobprpd_bgd` when present.
    #[must_use]
    pub fn creditor_objection_submission_start_date(&self) -> Option<&SourceValue> {
        self.source.get("cdobprpd_bgd")
    }

    /// Returns source field `cdobprpd_edd` when present.
    #[must_use]
    pub fn creditor_objection_submission_end_date(&self) -> Option<&SourceValue> {
        self.source.get("cdobprpd_edd")
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

    /// Returns source field `cr_mth` when present.
    #[must_use]
    pub fn capital_reduction_method(&self) -> Option<&SourceValue> {
        self.source.get("cr_mth")
    }

    /// Returns source field `cr_rs` when present.
    #[must_use]
    pub fn capital_reduction_reason(&self) -> Option<&SourceValue> {
        self.source.get("cr_rs")
    }

    /// Returns source field `cr_rt_estk` when present.
    #[must_use]
    pub fn other_share_reduction_ratio(&self) -> Option<&SourceValue> {
        self.source.get("cr_rt_estk")
    }

    /// Returns source field `cr_rt_ostk` when present.
    #[must_use]
    pub fn common_share_reduction_ratio(&self) -> Option<&SourceValue> {
        self.source.get("cr_rt_ostk")
    }

    /// Returns source field `cr_std` when present.
    #[must_use]
    pub fn capital_reduction_record_date(&self) -> Option<&SourceValue> {
        self.source.get("cr_std")
    }

    /// Returns source field `crsc_gmtsck_prd` when present.
    #[must_use]
    pub fn scheduled_general_meeting_date(&self) -> Option<&SourceValue> {
        self.source.get("crsc_gmtsck_prd")
    }

    /// Returns source field `crsc_nstkdlprd` when present.
    #[must_use]
    pub fn planned_new_share_certificate_delivery_date(&self) -> Option<&SourceValue> {
        self.source.get("crsc_nstkdlprd")
    }

    /// Returns source field `crsc_nstklstprd` when present.
    #[must_use]
    pub fn planned_new_share_listing_date(&self) -> Option<&SourceValue> {
        self.source.get("crsc_nstklstprd")
    }

    /// Returns source field `crsc_osprpd` when present.
    #[must_use]
    pub fn old_share_certificate_submission_period(&self) -> Option<&SourceValue> {
        self.source.get("crsc_osprpd")
    }

    /// Returns source field `crsc_osprpd_bgd` when present.
    #[must_use]
    pub fn old_share_certificate_submission_start_date(&self) -> Option<&SourceValue> {
        self.source.get("crsc_osprpd_bgd")
    }

    /// Returns source field `crsc_osprpd_edd` when present.
    #[must_use]
    pub fn old_share_certificate_submission_end_date(&self) -> Option<&SourceValue> {
        self.source.get("crsc_osprpd_edd")
    }

    /// Returns source field `crsc_trnmsppd` when present.
    #[must_use]
    pub fn transfer_registration_suspension_period(&self) -> Option<&SourceValue> {
        self.source.get("crsc_trnmsppd")
    }

    /// Returns source field `crsc_trspprpd` when present.
    #[must_use]
    pub fn planned_trading_suspension_period(&self) -> Option<&SourceValue> {
        self.source.get("crsc_trspprpd")
    }

    /// Returns source field `crsc_trspprpd_bgd` when present.
    #[must_use]
    pub fn planned_trading_suspension_start_date(&self) -> Option<&SourceValue> {
        self.source.get("crsc_trspprpd_bgd")
    }

    /// Returns source field `crsc_trspprpd_edd` when present.
    #[must_use]
    pub fn planned_trading_suspension_end_date(&self) -> Option<&SourceValue> {
        self.source.get("crsc_trspprpd_edd")
    }

    /// Returns source field `crstk_estk_cnt` when present.
    #[must_use]
    pub fn other_shares_subject_to_reduction(&self) -> Option<&SourceValue> {
        self.source.get("crstk_estk_cnt")
    }

    /// Returns source field `crstk_ostk_cnt` when present.
    #[must_use]
    pub fn common_shares_subject_to_reduction(&self) -> Option<&SourceValue> {
        self.source.get("crstk_ostk_cnt")
    }

    /// Returns source field `ftc_stt_atn` when present.
    #[must_use]
    pub fn fair_trade_commission_reporting_status(&self) -> Option<&SourceValue> {
        self.source.get("ftc_stt_atn")
    }

    /// Returns source field `fv_ps` when present.
    #[must_use]
    pub fn par_value_per_share(&self) -> Option<&SourceValue> {
        self.source.get("fv_ps")
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

    /// Returns source field `ospr_nstkdl_pl` when present.
    #[must_use]
    pub fn old_share_submission_and_new_share_delivery_location(&self) -> Option<&SourceValue> {
        self.source.get("ospr_nstkdl_pl")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }
}

/// Opaque response for physical operation `get_crDecsn_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct CapitalReductionDecisionsJsonResponse {
    source: SourceValue,
}

impl CapitalReductionDecisionsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = CapitalReductionDecision<'_>> + '_ {
        items(&self.source, "list", false).map(CapitalReductionDecision::new)
    }
}

fn decode_capital_reduction_decisions_json_response(
    source: SourceValue,
) -> Result<CapitalReductionDecisionsJsonResponse, ResponseDecodeError> {
    decode_capital_reduction_decisions_json_response_root(&source, "$".to_owned())?;
    Ok(CapitalReductionDecisionsJsonResponse { source })
}

fn decode_capital_reduction_decisions_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_capital_reduction_decisions_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_capital_reduction_decisions_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_capital_reduction_decisions_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_capital_reduction_decisions_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_crDecsn_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct CapitalReductionDecisionsXmlResponse {
    source: SourceValue,
}

impl CapitalReductionDecisionsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = CapitalReductionDecision<'_>> + '_ {
        items(&self.source, "list", true).map(CapitalReductionDecision::new)
    }
}

fn decode_capital_reduction_decisions_xml_response(
    source: SourceValue,
) -> Result<CapitalReductionDecisionsXmlResponse, ResponseDecodeError> {
    decode_capital_reduction_decisions_xml_response_root(&source, "$".to_owned())?;
    Ok(CapitalReductionDecisionsXmlResponse { source })
}

fn decode_capital_reduction_decisions_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_capital_reduction_decisions_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_capital_reduction_decisions_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_capital_reduction_decisions_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_capital_reduction_decisions_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
