use super::*;

/// Inputs for logical operation `DS005-2020053`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct ShareExchangeTransferDecisionsInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl ShareExchangeTransferDecisionsInput {
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

    /// Prepares physical operation `get_stkExtrDecsn_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<ShareExchangeTransferDecisionsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_stkExtrDecsn_json", "DS005-2020053");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/stkExtrDecsn.json", operation, &parameters),
            decode_share_exchange_transfer_decisions_json_response,
        ))
    }

    /// Prepares physical operation `get_stkExtrDecsn_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<ShareExchangeTransferDecisionsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_stkExtrDecsn_xml", "DS005-2020053");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/stkExtrDecsn.xml", operation, &parameters, "result"),
            decode_share_exchange_transfer_decisions_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct ShareExchangeTransferDecision<'a> {
    source: &'a SourceValue,
}

impl<'a> ShareExchangeTransferDecision<'a> {
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

    /// Returns source field `aprskh_ctref` when present.
    #[must_use]
    pub fn appraisal_rights_effect_on_contract(&self) -> Option<&SourceValue> {
        self.source.get("aprskh_ctref")
    }

    /// Returns source field `aprskh_lmt` when present.
    #[must_use]
    pub fn appraisal_rights_restrictions(&self) -> Option<&SourceValue> {
        self.source.get("aprskh_lmt")
    }

    /// Returns source field `aprskh_plnprc` when present.
    #[must_use]
    pub fn appraisal_rights_expected_purchase_price(&self) -> Option<&SourceValue> {
        self.source.get("aprskh_plnprc")
    }

    /// Returns source field `aprskh_pym_plpd_mth` when present.
    #[must_use]
    pub fn appraisal_rights_payment_timing_and_method(&self) -> Option<&SourceValue> {
        self.source.get("aprskh_pym_plpd_mth")
    }

    /// Returns source field `atextr_cpcmpnm` when present.
    #[must_use]
    pub fn wholly_owning_parent_company_after_exchange_or_transfer(&self) -> Option<&SourceValue> {
        self.source.get("atextr_cpcmpnm")
    }

    /// Returns source field `bddd` when present.
    #[must_use]
    pub fn board_resolution_date(&self) -> Option<&SourceValue> {
        self.source.get("bddd")
    }

    /// Returns source field `bdlst_atn` when present.
    #[must_use]
    pub fn backdoor_listing_applicable(&self) -> Option<&SourceValue> {
        self.source.get("bdlst_atn")
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

    /// Returns source field `ex_sm_r` when present.
    #[must_use]
    pub fn securities_registration_statement_exemption_reason(&self) -> Option<&SourceValue> {
        self.source.get("ex_sm_r")
    }

    /// Returns source field `exevl_atn` when present.
    #[must_use]
    pub fn external_appraisal_conducted(&self) -> Option<&SourceValue> {
        self.source.get("exevl_atn")
    }

    /// Returns source field `exevl_bs_rs` when present.
    #[must_use]
    pub fn external_appraisal_basis_and_reason(&self) -> Option<&SourceValue> {
        self.source.get("exevl_bs_rs")
    }

    /// Returns source field `exevl_intn` when present.
    #[must_use]
    pub fn external_appraisal_institution(&self) -> Option<&SourceValue> {
        self.source.get("exevl_intn")
    }

    /// Returns source field `exevl_op` when present.
    #[must_use]
    pub fn external_appraisal_opinion(&self) -> Option<&SourceValue> {
        self.source.get("exevl_op")
    }

    /// Returns source field `exevl_pd` when present.
    #[must_use]
    pub fn external_appraisal_period(&self) -> Option<&SourceValue> {
        self.source.get("exevl_pd")
    }

    /// Returns source field `extr_pp` when present.
    #[must_use]
    pub fn exchange_or_transfer_purpose(&self) -> Option<&SourceValue> {
        self.source.get("extr_pp")
    }

    /// Returns source field `extr_rt` when present.
    #[must_use]
    pub fn exchange_or_transfer_ratio(&self) -> Option<&SourceValue> {
        self.source.get("extr_rt")
    }

    /// Returns source field `extr_rt_bs` when present.
    #[must_use]
    pub fn exchange_or_transfer_ratio_basis(&self) -> Option<&SourceValue> {
        self.source.get("extr_rt_bs")
    }

    /// Returns source field `extr_sen` when present.
    #[must_use]
    pub fn exchange_or_transfer_classification(&self) -> Option<&SourceValue> {
        self.source.get("extr_sen")
    }

    /// Returns source field `extr_stn` when present.
    #[must_use]
    pub fn exchange_or_transfer_form(&self) -> Option<&SourceValue> {
        self.source.get("extr_stn")
    }

    /// Returns source field `extr_tgcmp_cmpnm` when present.
    #[must_use]
    pub fn target_company_name(&self) -> Option<&SourceValue> {
        self.source.get("extr_tgcmp_cmpnm")
    }

    /// Returns source field `extr_tgcmp_mbsn` when present.
    #[must_use]
    pub fn target_company_principal_business(&self) -> Option<&SourceValue> {
        self.source.get("extr_tgcmp_mbsn")
    }

    /// Returns source field `extr_tgcmp_rl_cmpn` when present.
    #[must_use]
    pub fn target_company_relationship_to_company(&self) -> Option<&SourceValue> {
        self.source.get("extr_tgcmp_rl_cmpn")
    }

    /// Returns source field `extr_tgcmp_rp` when present.
    #[must_use]
    pub fn target_company_representative(&self) -> Option<&SourceValue> {
        self.source.get("extr_tgcmp_rp")
    }

    /// Returns source field `extr_tgcmp_tisstk_cstk` when present.
    #[must_use]
    pub fn target_company_total_class_shares(&self) -> Option<&SourceValue> {
        self.source.get("extr_tgcmp_tisstk_cstk")
    }

    /// Returns source field `extr_tgcmp_tisstk_ostk` when present.
    #[must_use]
    pub fn target_company_total_common_shares(&self) -> Option<&SourceValue> {
        self.source.get("extr_tgcmp_tisstk_ostk")
    }

    /// Returns source field `extrsc_aprskh_expd_bgd` when present.
    #[must_use]
    pub fn exchange_or_transfer_appraisal_rights_exercise_start_date(
        &self,
    ) -> Option<&SourceValue> {
        self.source.get("extrsc_aprskh_expd_bgd")
    }

    /// Returns source field `extrsc_aprskh_expd_edd` when present.
    #[must_use]
    pub fn exchange_or_transfer_appraisal_rights_exercise_end_date(&self) -> Option<&SourceValue> {
        self.source.get("extrsc_aprskh_expd_edd")
    }

    /// Returns source field `extrsc_extrctrd` when present.
    #[must_use]
    pub fn exchange_or_transfer_contract_date(&self) -> Option<&SourceValue> {
        self.source.get("extrsc_extrctrd")
    }

    /// Returns source field `extrsc_extrdt` when present.
    #[must_use]
    pub fn exchange_or_transfer_date(&self) -> Option<&SourceValue> {
        self.source.get("extrsc_extrdt")
    }

    /// Returns source field `extrsc_extrop_rcpd_bgd` when present.
    #[must_use]
    pub fn exchange_or_transfer_dissent_notice_period_start_date(&self) -> Option<&SourceValue> {
        self.source.get("extrsc_extrop_rcpd_bgd")
    }

    /// Returns source field `extrsc_extrop_rcpd_edd` when present.
    #[must_use]
    pub fn exchange_or_transfer_dissent_notice_period_end_date(&self) -> Option<&SourceValue> {
        self.source.get("extrsc_extrop_rcpd_edd")
    }

    /// Returns source field `extrsc_gmtsck_prd` when present.
    #[must_use]
    pub fn planned_exchange_or_transfer_shareholders_meeting_date(&self) -> Option<&SourceValue> {
        self.source.get("extrsc_gmtsck_prd")
    }

    /// Returns source field `extrsc_nstkdlprd` when present.
    #[must_use]
    pub fn planned_exchange_or_transfer_new_share_certificate_delivery_date(
        &self,
    ) -> Option<&SourceValue> {
        self.source.get("extrsc_nstkdlprd")
    }

    /// Returns source field `extrsc_nstklstprd` when present.
    #[must_use]
    pub fn planned_exchange_or_transfer_new_share_listing_date(&self) -> Option<&SourceValue> {
        self.source.get("extrsc_nstklstprd")
    }

    /// Returns source field `extrsc_osprpd_bgd` when present.
    #[must_use]
    pub fn exchange_or_transfer_old_share_certificate_submission_start_date(
        &self,
    ) -> Option<&SourceValue> {
        self.source.get("extrsc_osprpd_bgd")
    }

    /// Returns source field `extrsc_osprpd_edd` when present.
    #[must_use]
    pub fn exchange_or_transfer_old_share_certificate_submission_end_date(
        &self,
    ) -> Option<&SourceValue> {
        self.source.get("extrsc_osprpd_edd")
    }

    /// Returns source field `extrsc_shclspd_bgd` when present.
    #[must_use]
    pub fn exchange_or_transfer_shareholder_registry_closure_start_date(
        &self,
    ) -> Option<&SourceValue> {
        self.source.get("extrsc_shclspd_bgd")
    }

    /// Returns source field `extrsc_shclspd_edd` when present.
    #[must_use]
    pub fn exchange_or_transfer_shareholder_registry_closure_end_date(
        &self,
    ) -> Option<&SourceValue> {
        self.source.get("extrsc_shclspd_edd")
    }

    /// Returns source field `extrsc_shddstd` when present.
    #[must_use]
    pub fn exchange_or_transfer_shareholder_record_date(&self) -> Option<&SourceValue> {
        self.source.get("extrsc_shddstd")
    }

    /// Returns source field `extrsc_trspprpd` when present.
    #[must_use]
    pub fn exchange_or_transfer_trading_suspension_period(&self) -> Option<&SourceValue> {
        self.source.get("extrsc_trspprpd")
    }

    /// Returns source field `extrsc_trspprpd_bgd` when present.
    #[must_use]
    pub fn exchange_or_transfer_trading_suspension_start_date(&self) -> Option<&SourceValue> {
        self.source.get("extrsc_trspprpd_bgd")
    }

    /// Returns source field `extrsc_trspprpd_edd` when present.
    #[must_use]
    pub fn exchange_or_transfer_trading_suspension_end_date(&self) -> Option<&SourceValue> {
        self.source.get("extrsc_trspprpd_edd")
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

    /// Returns source field `otcpr_bdlst_sf_atn` when present.
    #[must_use]
    pub fn other_company_meets_backdoor_listing_requirements(&self) -> Option<&SourceValue> {
        self.source.get("otcpr_bdlst_sf_atn")
    }

    /// Returns source field `popt_ctr_atn` when present.
    #[must_use]
    pub fn put_option_contract_exists(&self) -> Option<&SourceValue> {
        self.source.get("popt_ctr_atn")
    }

    /// Returns source field `popt_ctr_cn` when present.
    #[must_use]
    pub fn put_option_contract_terms(&self) -> Option<&SourceValue> {
        self.source.get("popt_ctr_cn")
    }

    /// Returns source field `rbsnfdtl_cpt` when present.
    #[must_use]
    pub fn target_company_capital(&self) -> Option<&SourceValue> {
        self.source.get("rbsnfdtl_cpt")
    }

    /// Returns source field `rbsnfdtl_tast` when present.
    #[must_use]
    pub fn target_company_total_assets(&self) -> Option<&SourceValue> {
        self.source.get("rbsnfdtl_tast")
    }

    /// Returns source field `rbsnfdtl_tdbt` when present.
    #[must_use]
    pub fn target_company_total_liabilities(&self) -> Option<&SourceValue> {
        self.source.get("rbsnfdtl_tdbt")
    }

    /// Returns source field `rbsnfdtl_teqt` when present.
    #[must_use]
    pub fn target_company_total_equity(&self) -> Option<&SourceValue> {
        self.source.get("rbsnfdtl_teqt")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `rs_sm_atn` when present.
    #[must_use]
    pub fn securities_registration_statement_required(&self) -> Option<&SourceValue> {
        self.source.get("rs_sm_atn")
    }
}

/// Opaque response for physical operation `get_stkExtrDecsn_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct ShareExchangeTransferDecisionsJsonResponse {
    source: SourceValue,
}

impl ShareExchangeTransferDecisionsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = ShareExchangeTransferDecision<'_>> + '_ {
        items(&self.source, "list", false).map(ShareExchangeTransferDecision::new)
    }
}

fn decode_share_exchange_transfer_decisions_json_response(
    source: SourceValue,
) -> Result<ShareExchangeTransferDecisionsJsonResponse, ResponseDecodeError> {
    decode_share_exchange_transfer_decisions_json_response_root(&source, "$".to_owned())?;
    Ok(ShareExchangeTransferDecisionsJsonResponse { source })
}

fn decode_share_exchange_transfer_decisions_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_share_exchange_transfer_decisions_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_share_exchange_transfer_decisions_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_share_exchange_transfer_decisions_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_share_exchange_transfer_decisions_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_stkExtrDecsn_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct ShareExchangeTransferDecisionsXmlResponse {
    source: SourceValue,
}

impl ShareExchangeTransferDecisionsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = ShareExchangeTransferDecision<'_>> + '_ {
        items(&self.source, "list", true).map(ShareExchangeTransferDecision::new)
    }
}

fn decode_share_exchange_transfer_decisions_xml_response(
    source: SourceValue,
) -> Result<ShareExchangeTransferDecisionsXmlResponse, ResponseDecodeError> {
    decode_share_exchange_transfer_decisions_xml_response_root(&source, "$".to_owned())?;
    Ok(ShareExchangeTransferDecisionsXmlResponse { source })
}

fn decode_share_exchange_transfer_decisions_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_share_exchange_transfer_decisions_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_share_exchange_transfer_decisions_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_share_exchange_transfer_decisions_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_share_exchange_transfer_decisions_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
