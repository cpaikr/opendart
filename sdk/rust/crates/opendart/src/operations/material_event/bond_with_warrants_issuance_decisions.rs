use super::*;

/// Inputs for logical operation `DS005-2020034`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct BondWithWarrantsIssuanceDecisionsInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl BondWithWarrantsIssuanceDecisionsInput {
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

    /// Prepares physical operation `get_bdwtIsDecsn_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<BondWithWarrantsIssuanceDecisionsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_bdwtIsDecsn_json", "DS005-2020034");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/bdwtIsDecsn.json", operation, &parameters),
            decode_bond_with_warrants_issuance_decisions_json_response,
        ))
    }

    /// Prepares physical operation `get_bdwtIsDecsn_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<BondWithWarrantsIssuanceDecisionsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_bdwtIsDecsn_xml", "DS005-2020034");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/bdwtIsDecsn.xml", operation, &parameters, "result"),
            decode_bond_with_warrants_issuance_decisions_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct BondWithWarrantsIssuanceDecision<'a> {
    source: &'a SourceValue,
}

impl<'a> BondWithWarrantsIssuanceDecision<'a> {
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

    /// Returns source field `abmg` when present.
    #[must_use]
    pub fn merger_matters(&self) -> Option<&SourceValue> {
        self.source.get("abmg")
    }

    /// Returns source field `act_mktprcfl_cvprc_lwtrsprc` when present.
    #[must_use]
    pub fn lowest_adjusted_exercise_price(&self) -> Option<&SourceValue> {
        self.source.get("act_mktprcfl_cvprc_lwtrsprc")
    }

    /// Returns source field `act_mktprcfl_cvprc_lwtrsprc_bs` when present.
    #[must_use]
    pub fn lowest_adjusted_exercise_price_basis(&self) -> Option<&SourceValue> {
        self.source.get("act_mktprcfl_cvprc_lwtrsprc_bs")
    }

    /// Returns source field `adt_a_atn` when present.
    #[must_use]
    pub fn auditor_attendance_status(&self) -> Option<&SourceValue> {
        self.source.get("adt_a_atn")
    }

    /// Returns source field `atcsc_rmislmt` when present.
    #[must_use]
    pub fn remaining_articles_issuance_limit(&self) -> Option<&SourceValue> {
        self.source.get("atcsc_rmislmt")
    }

    /// Returns source field `bd_fta` when present.
    #[must_use]
    pub fn bond_face_value_total(&self) -> Option<&SourceValue> {
        self.source.get("bd_fta")
    }

    /// Returns source field `bd_intr_ex` when present.
    #[must_use]
    pub fn bond_coupon_interest_rate(&self) -> Option<&SourceValue> {
        self.source.get("bd_intr_ex")
    }

    /// Returns source field `bd_intr_sf` when present.
    #[must_use]
    pub fn bond_maturity_interest_rate(&self) -> Option<&SourceValue> {
        self.source.get("bd_intr_sf")
    }

    /// Returns source field `bd_knd` when present.
    #[must_use]
    pub fn bond_type(&self) -> Option<&SourceValue> {
        self.source.get("bd_knd")
    }

    /// Returns source field `bd_mtd` when present.
    #[must_use]
    pub fn bond_maturity_date(&self) -> Option<&SourceValue> {
        self.source.get("bd_mtd")
    }

    /// Returns source field `bd_tm` when present.
    #[must_use]
    pub fn bond_series(&self) -> Option<&SourceValue> {
        self.source.get("bd_tm")
    }

    /// Returns source field `bddd` when present.
    #[must_use]
    pub fn board_resolution_date(&self) -> Option<&SourceValue> {
        self.source.get("bddd")
    }

    /// Returns source field `bdis_mthn` when present.
    #[must_use]
    pub fn bond_issuance_method(&self) -> Option<&SourceValue> {
        self.source.get("bdis_mthn")
    }

    /// Returns source field `bdwt_div_atn` when present.
    #[must_use]
    pub fn bond_and_warrant_separable(&self) -> Option<&SourceValue> {
        self.source.get("bdwt_div_atn")
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

    /// Returns source field `ex_prc` when present.
    #[must_use]
    pub fn warrant_exercise_price_per_share(&self) -> Option<&SourceValue> {
        self.source.get("ex_prc")
    }

    /// Returns source field `ex_prc_dmth` when present.
    #[must_use]
    pub fn warrant_exercise_price_determination_method(&self) -> Option<&SourceValue> {
        self.source.get("ex_prc_dmth")
    }

    /// Returns source field `ex_rt` when present.
    #[must_use]
    pub fn warrant_exercise_ratio(&self) -> Option<&SourceValue> {
        self.source.get("ex_rt")
    }

    /// Returns source field `ex_sm_r` when present.
    #[must_use]
    pub fn registration_statement_exemption_reason(&self) -> Option<&SourceValue> {
        self.source.get("ex_sm_r")
    }

    /// Returns source field `expd_bgd` when present.
    #[must_use]
    pub fn warrant_exercise_period_start_date(&self) -> Option<&SourceValue> {
        self.source.get("expd_bgd")
    }

    /// Returns source field `expd_edd` when present.
    #[must_use]
    pub fn warrant_exercise_period_end_date(&self) -> Option<&SourceValue> {
        self.source.get("expd_edd")
    }

    /// Returns source field `fdpp_bsninh` when present.
    #[must_use]
    pub fn business_acquisition_funding_amount(&self) -> Option<&SourceValue> {
        self.source.get("fdpp_bsninh")
    }

    /// Returns source field `fdpp_dtrp` when present.
    #[must_use]
    pub fn debt_repayment_funding_amount(&self) -> Option<&SourceValue> {
        self.source.get("fdpp_dtrp")
    }

    /// Returns source field `fdpp_etc` when present.
    #[must_use]
    pub fn other_funding_amount(&self) -> Option<&SourceValue> {
        self.source.get("fdpp_etc")
    }

    /// Returns source field `fdpp_fclt` when present.
    #[must_use]
    pub fn facility_funding_amount(&self) -> Option<&SourceValue> {
        self.source.get("fdpp_fclt")
    }

    /// Returns source field `fdpp_ocsa` when present.
    #[must_use]
    pub fn other_corporation_securities_acquisition_funding_amount(&self) -> Option<&SourceValue> {
        self.source.get("fdpp_ocsa")
    }

    /// Returns source field `fdpp_op` when present.
    #[must_use]
    pub fn operating_funding_amount(&self) -> Option<&SourceValue> {
        self.source.get("fdpp_op")
    }

    /// Returns source field `ftc_stt_atn` when present.
    #[must_use]
    pub fn fair_trade_commission_reporting_required(&self) -> Option<&SourceValue> {
        self.source.get("ftc_stt_atn")
    }

    /// Returns source field `grint` when present.
    #[must_use]
    pub fn guarantor(&self) -> Option<&SourceValue> {
        self.source.get("grint")
    }

    /// Returns source field `nstk_isstk_cnt` when present.
    #[must_use]
    pub fn warrant_exercise_share_count(&self) -> Option<&SourceValue> {
        self.source.get("nstk_isstk_cnt")
    }

    /// Returns source field `nstk_isstk_knd` when present.
    #[must_use]
    pub fn warrant_exercise_share_type(&self) -> Option<&SourceValue> {
        self.source.get("nstk_isstk_knd")
    }

    /// Returns source field `nstk_isstk_tisstk_vs` when present.
    #[must_use]
    pub fn warrant_exercise_shares_to_total_ratio(&self) -> Option<&SourceValue> {
        self.source.get("nstk_isstk_tisstk_vs")
    }

    /// Returns source field `nstk_pym_mth` when present.
    #[must_use]
    pub fn new_share_payment_method(&self) -> Option<&SourceValue> {
        self.source.get("nstk_pym_mth")
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

    /// Returns source field `ovis_fta` when present.
    #[must_use]
    pub fn overseas_issuance_face_value_total(&self) -> Option<&SourceValue> {
        self.source.get("ovis_fta")
    }

    /// Returns source field `ovis_fta_crn` when present.
    #[must_use]
    pub fn overseas_issuance_currency(&self) -> Option<&SourceValue> {
        self.source.get("ovis_fta_crn")
    }

    /// Returns source field `ovis_isar` when present.
    #[must_use]
    pub fn overseas_issuance_region(&self) -> Option<&SourceValue> {
        self.source.get("ovis_isar")
    }

    /// Returns source field `ovis_ltdtl` when present.
    #[must_use]
    pub fn overseas_issuance_linked_securities_lending_details(&self) -> Option<&SourceValue> {
        self.source.get("ovis_ltdtl")
    }

    /// Returns source field `ovis_mktnm` when present.
    #[must_use]
    pub fn overseas_listing_market_name(&self) -> Option<&SourceValue> {
        self.source.get("ovis_mktnm")
    }

    /// Returns source field `ovis_ster` when present.
    #[must_use]
    pub fn overseas_issuance_exchange_rate_basis(&self) -> Option<&SourceValue> {
        self.source.get("ovis_ster")
    }

    /// Returns source field `pymd` when present.
    #[must_use]
    pub fn payment_date(&self) -> Option<&SourceValue> {
        self.source.get("pymd")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `rmislmt_lt70p` when present.
    #[must_use]
    pub fn remaining_below_seventy_percent_adjustment_limit(&self) -> Option<&SourceValue> {
        self.source.get("rmislmt_lt70p")
    }

    /// Returns source field `rpmcmp` when present.
    #[must_use]
    pub fn lead_underwriter(&self) -> Option<&SourceValue> {
        self.source.get("rpmcmp")
    }

    /// Returns source field `rs_sm_atn` when present.
    #[must_use]
    pub fn securities_registration_statement_required(&self) -> Option<&SourceValue> {
        self.source.get("rs_sm_atn")
    }

    /// Returns source field `sbd` when present.
    #[must_use]
    pub fn subscription_date(&self) -> Option<&SourceValue> {
        self.source.get("sbd")
    }
}

/// Opaque response for physical operation `get_bdwtIsDecsn_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct BondWithWarrantsIssuanceDecisionsJsonResponse {
    source: SourceValue,
}

impl BondWithWarrantsIssuanceDecisionsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = BondWithWarrantsIssuanceDecision<'_>> + '_ {
        items(&self.source, "list", false).map(BondWithWarrantsIssuanceDecision::new)
    }
}

fn decode_bond_with_warrants_issuance_decisions_json_response(
    source: SourceValue,
) -> Result<BondWithWarrantsIssuanceDecisionsJsonResponse, ResponseDecodeError> {
    decode_bond_with_warrants_issuance_decisions_json_response_root(&source, "$".to_owned())?;
    Ok(BondWithWarrantsIssuanceDecisionsJsonResponse { source })
}

fn decode_bond_with_warrants_issuance_decisions_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_bond_with_warrants_issuance_decisions_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_bond_with_warrants_issuance_decisions_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_bond_with_warrants_issuance_decisions_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_bond_with_warrants_issuance_decisions_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_bdwtIsDecsn_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct BondWithWarrantsIssuanceDecisionsXmlResponse {
    source: SourceValue,
}

impl BondWithWarrantsIssuanceDecisionsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = BondWithWarrantsIssuanceDecision<'_>> + '_ {
        items(&self.source, "list", true).map(BondWithWarrantsIssuanceDecision::new)
    }
}

fn decode_bond_with_warrants_issuance_decisions_xml_response(
    source: SourceValue,
) -> Result<BondWithWarrantsIssuanceDecisionsXmlResponse, ResponseDecodeError> {
    decode_bond_with_warrants_issuance_decisions_xml_response_root(&source, "$".to_owned())?;
    Ok(BondWithWarrantsIssuanceDecisionsXmlResponse { source })
}

fn decode_bond_with_warrants_issuance_decisions_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_bond_with_warrants_issuance_decisions_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_bond_with_warrants_issuance_decisions_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_bond_with_warrants_issuance_decisions_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_bond_with_warrants_issuance_decisions_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
