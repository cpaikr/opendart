use super::*;

/// Inputs for logical operation `DS005-2020025`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct CombinedPaidInAndBonusShareIssueDecisionsInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl CombinedPaidInAndBonusShareIssueDecisionsInput {
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

    /// Prepares physical operation `get_pifricDecsn_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<CombinedPaidInAndBonusShareIssueDecisionsJsonResponse>, PrepareError>
    {
        let operation = OperationIdentity::new("get_pifricDecsn_json", "DS005-2020025");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/pifricDecsn.json", operation, &parameters),
            decode_combined_paid_in_and_bonus_share_issue_decisions_json_response,
        ))
    }

    /// Prepares physical operation `get_pifricDecsn_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<CombinedPaidInAndBonusShareIssueDecisionsXmlResponse>, PrepareError>
    {
        let operation = OperationIdentity::new("get_pifricDecsn_xml", "DS005-2020025");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/pifricDecsn.xml", operation, &parameters, "result"),
            decode_combined_paid_in_and_bonus_share_issue_decisions_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct CombinedPaidInAndBonusShareIssueDecision<'a> {
    source: &'a SourceValue,
}

impl<'a> CombinedPaidInAndBonusShareIssueDecision<'a> {
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

    /// Returns source field `fric_adt_a_atn` when present.
    #[must_use]
    pub fn bonus_issue_auditor_attendance_status(&self) -> Option<&SourceValue> {
        self.source.get("fric_adt_a_atn")
    }

    /// Returns source field `fric_bddd` when present.
    #[must_use]
    pub fn bonus_issue_board_resolution_date(&self) -> Option<&SourceValue> {
        self.source.get("fric_bddd")
    }

    /// Returns source field `fric_bfic_tisstk_estk` when present.
    #[must_use]
    pub fn bonus_issue_other_shares_issued_before_increase(&self) -> Option<&SourceValue> {
        self.source.get("fric_bfic_tisstk_estk")
    }

    /// Returns source field `fric_bfic_tisstk_ostk` when present.
    #[must_use]
    pub fn bonus_issue_common_shares_issued_before_increase(&self) -> Option<&SourceValue> {
        self.source.get("fric_bfic_tisstk_ostk")
    }

    /// Returns source field `fric_fv_ps` when present.
    #[must_use]
    pub fn bonus_issue_par_value_per_share(&self) -> Option<&SourceValue> {
        self.source.get("fric_fv_ps")
    }

    /// Returns source field `fric_nstk_ascnt_ps_estk` when present.
    #[must_use]
    pub fn bonus_issue_new_other_shares_allocated_per_share(&self) -> Option<&SourceValue> {
        self.source.get("fric_nstk_ascnt_ps_estk")
    }

    /// Returns source field `fric_nstk_ascnt_ps_ostk` when present.
    #[must_use]
    pub fn bonus_issue_new_common_shares_allocated_per_share(&self) -> Option<&SourceValue> {
        self.source.get("fric_nstk_ascnt_ps_ostk")
    }

    /// Returns source field `fric_nstk_asstd` when present.
    #[must_use]
    pub fn bonus_issue_new_share_allocation_record_date(&self) -> Option<&SourceValue> {
        self.source.get("fric_nstk_asstd")
    }

    /// Returns source field `fric_nstk_dividrk` when present.
    #[must_use]
    pub fn bonus_issue_new_share_dividend_accrual_date(&self) -> Option<&SourceValue> {
        self.source.get("fric_nstk_dividrk")
    }

    /// Returns source field `fric_nstk_dlprd` when present.
    #[must_use]
    pub fn bonus_issue_new_share_certificate_delivery_date(&self) -> Option<&SourceValue> {
        self.source.get("fric_nstk_dlprd")
    }

    /// Returns source field `fric_nstk_estk_cnt` when present.
    #[must_use]
    pub fn bonus_issue_new_other_share_count(&self) -> Option<&SourceValue> {
        self.source.get("fric_nstk_estk_cnt")
    }

    /// Returns source field `fric_nstk_lstprd` when present.
    #[must_use]
    pub fn bonus_issue_planned_new_share_listing_date(&self) -> Option<&SourceValue> {
        self.source.get("fric_nstk_lstprd")
    }

    /// Returns source field `fric_nstk_ostk_cnt` when present.
    #[must_use]
    pub fn bonus_issue_new_common_share_count(&self) -> Option<&SourceValue> {
        self.source.get("fric_nstk_ostk_cnt")
    }

    /// Returns source field `fric_od_a_at_b` when present.
    #[must_use]
    pub fn bonus_issue_outside_directors_absent(&self) -> Option<&SourceValue> {
        self.source.get("fric_od_a_at_b")
    }

    /// Returns source field `fric_od_a_at_t` when present.
    #[must_use]
    pub fn bonus_issue_outside_directors_present(&self) -> Option<&SourceValue> {
        self.source.get("fric_od_a_at_t")
    }

    /// Returns source field `piic_bfic_tisstk_estk` when present.
    #[must_use]
    pub fn paid_in_other_shares_issued_before_increase(&self) -> Option<&SourceValue> {
        self.source.get("piic_bfic_tisstk_estk")
    }

    /// Returns source field `piic_bfic_tisstk_ostk` when present.
    #[must_use]
    pub fn paid_in_common_shares_issued_before_increase(&self) -> Option<&SourceValue> {
        self.source.get("piic_bfic_tisstk_ostk")
    }

    /// Returns source field `piic_fdpp_bsninh` when present.
    #[must_use]
    pub fn paid_in_business_acquisition_funds(&self) -> Option<&SourceValue> {
        self.source.get("piic_fdpp_bsninh")
    }

    /// Returns source field `piic_fdpp_dtrp` when present.
    #[must_use]
    pub fn paid_in_debt_repayment_funds(&self) -> Option<&SourceValue> {
        self.source.get("piic_fdpp_dtrp")
    }

    /// Returns source field `piic_fdpp_etc` when present.
    #[must_use]
    pub fn paid_in_other_funds(&self) -> Option<&SourceValue> {
        self.source.get("piic_fdpp_etc")
    }

    /// Returns source field `piic_fdpp_fclt` when present.
    #[must_use]
    pub fn paid_in_facility_funds(&self) -> Option<&SourceValue> {
        self.source.get("piic_fdpp_fclt")
    }

    /// Returns source field `piic_fdpp_ocsa` when present.
    #[must_use]
    pub fn paid_in_other_corporation_securities_acquisition_funds(&self) -> Option<&SourceValue> {
        self.source.get("piic_fdpp_ocsa")
    }

    /// Returns source field `piic_fdpp_op` when present.
    #[must_use]
    pub fn paid_in_operating_funds(&self) -> Option<&SourceValue> {
        self.source.get("piic_fdpp_op")
    }

    /// Returns source field `piic_fv_ps` when present.
    #[must_use]
    pub fn paid_in_par_value_per_share(&self) -> Option<&SourceValue> {
        self.source.get("piic_fv_ps")
    }

    /// Returns source field `piic_ic_mthn` when present.
    #[must_use]
    pub fn paid_in_capital_increase_method(&self) -> Option<&SourceValue> {
        self.source.get("piic_ic_mthn")
    }

    /// Returns source field `piic_nstk_estk_cnt` when present.
    #[must_use]
    pub fn paid_in_new_other_share_count(&self) -> Option<&SourceValue> {
        self.source.get("piic_nstk_estk_cnt")
    }

    /// Returns source field `piic_nstk_ostk_cnt` when present.
    #[must_use]
    pub fn paid_in_new_common_share_count(&self) -> Option<&SourceValue> {
        self.source.get("piic_nstk_ostk_cnt")
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

/// Opaque response for physical operation `get_pifricDecsn_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct CombinedPaidInAndBonusShareIssueDecisionsJsonResponse {
    source: SourceValue,
}

impl CombinedPaidInAndBonusShareIssueDecisionsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = CombinedPaidInAndBonusShareIssueDecision<'_>> + '_ {
        items(&self.source, "list", false).map(CombinedPaidInAndBonusShareIssueDecision::new)
    }
}

fn decode_combined_paid_in_and_bonus_share_issue_decisions_json_response(
    source: SourceValue,
) -> Result<CombinedPaidInAndBonusShareIssueDecisionsJsonResponse, ResponseDecodeError> {
    decode_combined_paid_in_and_bonus_share_issue_decisions_json_response_root(
        &source,
        "$".to_owned(),
    )?;
    Ok(CombinedPaidInAndBonusShareIssueDecisionsJsonResponse { source })
}

fn decode_combined_paid_in_and_bonus_share_issue_decisions_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_combined_paid_in_and_bonus_share_issue_decisions_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_combined_paid_in_and_bonus_share_issue_decisions_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_combined_paid_in_and_bonus_share_issue_decisions_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_combined_paid_in_and_bonus_share_issue_decisions_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_pifricDecsn_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct CombinedPaidInAndBonusShareIssueDecisionsXmlResponse {
    source: SourceValue,
}

impl CombinedPaidInAndBonusShareIssueDecisionsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = CombinedPaidInAndBonusShareIssueDecision<'_>> + '_ {
        items(&self.source, "list", true).map(CombinedPaidInAndBonusShareIssueDecision::new)
    }
}

fn decode_combined_paid_in_and_bonus_share_issue_decisions_xml_response(
    source: SourceValue,
) -> Result<CombinedPaidInAndBonusShareIssueDecisionsXmlResponse, ResponseDecodeError> {
    decode_combined_paid_in_and_bonus_share_issue_decisions_xml_response_root(
        &source,
        "$".to_owned(),
    )?;
    Ok(CombinedPaidInAndBonusShareIssueDecisionsXmlResponse { source })
}

fn decode_combined_paid_in_and_bonus_share_issue_decisions_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_combined_paid_in_and_bonus_share_issue_decisions_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_combined_paid_in_and_bonus_share_issue_decisions_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_combined_paid_in_and_bonus_share_issue_decisions_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_combined_paid_in_and_bonus_share_issue_decisions_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
