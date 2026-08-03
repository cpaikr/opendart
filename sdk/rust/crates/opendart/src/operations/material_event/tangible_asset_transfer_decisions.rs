use super::*;

/// Inputs for logical operation `DS005-2020045`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct TangibleAssetTransferDecisionsInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl TangibleAssetTransferDecisionsInput {
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

    /// Prepares physical operation `get_tgastTrfDecsn_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<TangibleAssetTransferDecisionsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_tgastTrfDecsn_json", "DS005-2020045");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/tgastTrfDecsn.json", operation, &parameters),
            decode_tangible_asset_transfer_decisions_json_response,
        ))
    }

    /// Prepares physical operation `get_tgastTrfDecsn_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<TangibleAssetTransferDecisionsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_tgastTrfDecsn_xml", "DS005-2020045");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/tgastTrfDecsn.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_tangible_asset_transfer_decisions_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct TangibleAssetTransferDecision<'a> {
    source: &'a SourceValue,
}

impl<'a> TangibleAssetTransferDecision<'a> {
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

    /// Returns source field `aprskh_ex_pc_mth_pd_pl` when present.
    #[must_use]
    pub fn appraisal_rights_exercise_procedure_method_period_and_place(
        &self,
    ) -> Option<&SourceValue> {
        self.source.get("aprskh_ex_pc_mth_pd_pl")
    }

    /// Returns source field `aprskh_exrq` when present.
    #[must_use]
    pub fn appraisal_rights_exercise_requirements(&self) -> Option<&SourceValue> {
        self.source.get("aprskh_exrq")
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

    /// Returns source field `ast_nm` when present.
    #[must_use]
    pub fn asset_name(&self) -> Option<&SourceValue> {
        self.source.get("ast_nm")
    }

    /// Returns source field `ast_sen` when present.
    #[must_use]
    pub fn asset_classification(&self) -> Option<&SourceValue> {
        self.source.get("ast_sen")
    }

    /// Returns source field `bddd` when present.
    #[must_use]
    pub fn board_resolution_date(&self) -> Option<&SourceValue> {
        self.source.get("bddd")
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

    /// Returns source field `dl_pym` when present.
    #[must_use]
    pub fn transaction_payment_terms(&self) -> Option<&SourceValue> {
        self.source.get("dl_pym")
    }

    /// Returns source field `dlptn_cmpnm` when present.
    #[must_use]
    pub fn counterparty_name(&self) -> Option<&SourceValue> {
        self.source.get("dlptn_cmpnm")
    }

    /// Returns source field `dlptn_cpt` when present.
    #[must_use]
    pub fn counterparty_capital(&self) -> Option<&SourceValue> {
        self.source.get("dlptn_cpt")
    }

    /// Returns source field `dlptn_hoadd` when present.
    #[must_use]
    pub fn counterparty_headquarters_address(&self) -> Option<&SourceValue> {
        self.source.get("dlptn_hoadd")
    }

    /// Returns source field `dlptn_mbsn` when present.
    #[must_use]
    pub fn counterparty_principal_business(&self) -> Option<&SourceValue> {
        self.source.get("dlptn_mbsn")
    }

    /// Returns source field `dlptn_rl_cmpn` when present.
    #[must_use]
    pub fn counterparty_relationship_to_company(&self) -> Option<&SourceValue> {
        self.source.get("dlptn_rl_cmpn")
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

    /// Returns source field `ftc_stt_atn` when present.
    #[must_use]
    pub fn fair_trade_commission_filing_required(&self) -> Option<&SourceValue> {
        self.source.get("ftc_stt_atn")
    }

    /// Returns source field `gmtsck_prd` when present.
    #[must_use]
    pub fn planned_shareholders_meeting_date(&self) -> Option<&SourceValue> {
        self.source.get("gmtsck_prd")
    }

    /// Returns source field `gmtsck_spd_atn` when present.
    #[must_use]
    pub fn shareholders_meeting_special_resolution_required(&self) -> Option<&SourceValue> {
        self.source.get("gmtsck_spd_atn")
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

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `trf_af` when present.
    #[must_use]
    pub fn transfer_effect(&self) -> Option<&SourceValue> {
        self.source.get("trf_af")
    }

    /// Returns source field `trf_pp` when present.
    #[must_use]
    pub fn transfer_purpose(&self) -> Option<&SourceValue> {
        self.source.get("trf_pp")
    }

    /// Returns source field `trf_prd_ctr_cnsd` when present.
    #[must_use]
    pub fn transfer_contract_date(&self) -> Option<&SourceValue> {
        self.source.get("trf_prd_ctr_cnsd")
    }

    /// Returns source field `trf_prd_rgs_prd` when present.
    #[must_use]
    pub fn registration_date(&self) -> Option<&SourceValue> {
        self.source.get("trf_prd_rgs_prd")
    }

    /// Returns source field `trf_prd_trf_std` when present.
    #[must_use]
    pub fn transfer_reference_date(&self) -> Option<&SourceValue> {
        self.source.get("trf_prd_trf_std")
    }

    /// Returns source field `trfdtl_tast` when present.
    #[must_use]
    pub fn total_assets(&self) -> Option<&SourceValue> {
        self.source.get("trfdtl_tast")
    }

    /// Returns source field `trfdtl_tast_vs` when present.
    #[must_use]
    pub fn transfer_amount_to_total_assets_ratio(&self) -> Option<&SourceValue> {
        self.source.get("trfdtl_tast_vs")
    }

    /// Returns source field `trfdtl_trfprc` when present.
    #[must_use]
    pub fn transfer_amount(&self) -> Option<&SourceValue> {
        self.source.get("trfdtl_trfprc")
    }
}

/// Opaque response for physical operation `get_tgastTrfDecsn_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct TangibleAssetTransferDecisionsJsonResponse {
    source: SourceValue,
}

impl TangibleAssetTransferDecisionsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = TangibleAssetTransferDecision<'_>> + '_ {
        items(&self.source, "list", false).map(TangibleAssetTransferDecision::new)
    }
}

fn decode_tangible_asset_transfer_decisions_json_response(
    source: SourceValue,
) -> Result<TangibleAssetTransferDecisionsJsonResponse, ResponseDecodeError> {
    decode_tangible_asset_transfer_decisions_json_response_root(&source, "$".to_owned())?;
    Ok(TangibleAssetTransferDecisionsJsonResponse { source })
}

fn decode_tangible_asset_transfer_decisions_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_tangible_asset_transfer_decisions_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_tangible_asset_transfer_decisions_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_tangible_asset_transfer_decisions_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_tangible_asset_transfer_decisions_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_tgastTrfDecsn_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct TangibleAssetTransferDecisionsXmlResponse {
    source: SourceValue,
}

impl TangibleAssetTransferDecisionsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = TangibleAssetTransferDecision<'_>> + '_ {
        items(&self.source, "list", true).map(TangibleAssetTransferDecision::new)
    }
}

fn decode_tangible_asset_transfer_decisions_xml_response(
    source: SourceValue,
) -> Result<TangibleAssetTransferDecisionsXmlResponse, ResponseDecodeError> {
    decode_tangible_asset_transfer_decisions_xml_response_root(&source, "$".to_owned())?;
    Ok(TangibleAssetTransferDecisionsXmlResponse { source })
}

fn decode_tangible_asset_transfer_decisions_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_tangible_asset_transfer_decisions_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_tangible_asset_transfer_decisions_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_tangible_asset_transfer_decisions_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_tangible_asset_transfer_decisions_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
