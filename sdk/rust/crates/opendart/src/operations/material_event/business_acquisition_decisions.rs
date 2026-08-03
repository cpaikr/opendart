use super::*;

/// Inputs for logical operation `DS005-2020042`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct BusinessAcquisitionDecisionsInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl BusinessAcquisitionDecisionsInput {
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

    /// Prepares physical operation `get_bsnInhDecsn_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<BusinessAcquisitionDecisionsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_bsnInhDecsn_json", "DS005-2020042");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/bsnInhDecsn.json", operation, &parameters),
            decode_business_acquisition_decisions_json_response,
        ))
    }

    /// Prepares physical operation `get_bsnInhDecsn_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<BusinessAcquisitionDecisionsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_bsnInhDecsn_xml", "DS005-2020042");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/bsnInhDecsn.xml", operation, &parameters, "result"),
            decode_business_acquisition_decisions_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct BusinessAcquisitionDecision<'a> {
    source: &'a SourceValue,
}

impl<'a> BusinessAcquisitionDecision<'a> {
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

    /// Returns source field `absn_inh_atn` when present.
    #[must_use]
    pub fn acquires_entire_business(&self) -> Option<&SourceValue> {
        self.source.get("absn_inh_atn")
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

    /// Returns source field `ast_cmp_all` when present.
    #[must_use]
    pub fn company_total_assets(&self) -> Option<&SourceValue> {
        self.source.get("ast_cmp_all")
    }

    /// Returns source field `ast_inh_bsn` when present.
    #[must_use]
    pub fn target_business_assets(&self) -> Option<&SourceValue> {
        self.source.get("ast_inh_bsn")
    }

    /// Returns source field `ast_rt` when present.
    #[must_use]
    pub fn target_assets_ratio(&self) -> Option<&SourceValue> {
        self.source.get("ast_rt")
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

    /// Returns source field `dbt_cmp_all` when present.
    #[must_use]
    pub fn company_total_liabilities(&self) -> Option<&SourceValue> {
        self.source.get("dbt_cmp_all")
    }

    /// Returns source field `dbt_inh_bsn` when present.
    #[must_use]
    pub fn target_business_liabilities(&self) -> Option<&SourceValue> {
        self.source.get("dbt_inh_bsn")
    }

    /// Returns source field `dbt_rt` when present.
    #[must_use]
    pub fn target_liabilities_ratio(&self) -> Option<&SourceValue> {
        self.source.get("dbt_rt")
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

    /// Returns source field `inh_af` when present.
    #[must_use]
    pub fn acquisition_effect(&self) -> Option<&SourceValue> {
        self.source.get("inh_af")
    }

    /// Returns source field `inh_bsn` when present.
    #[must_use]
    pub fn acquired_business(&self) -> Option<&SourceValue> {
        self.source.get("inh_bsn")
    }

    /// Returns source field `inh_bsn_mc` when present.
    #[must_use]
    pub fn acquired_business_details(&self) -> Option<&SourceValue> {
        self.source.get("inh_bsn_mc")
    }

    /// Returns source field `inh_pp` when present.
    #[must_use]
    pub fn acquisition_purpose(&self) -> Option<&SourceValue> {
        self.source.get("inh_pp")
    }

    /// Returns source field `inh_prc` when present.
    #[must_use]
    pub fn acquisition_price(&self) -> Option<&SourceValue> {
        self.source.get("inh_prc")
    }

    /// Returns source field `inh_prd_ctr_cnsd` when present.
    #[must_use]
    pub fn acquisition_contract_date(&self) -> Option<&SourceValue> {
        self.source.get("inh_prd_ctr_cnsd")
    }

    /// Returns source field `inh_prd_inh_std` when present.
    #[must_use]
    pub fn acquisition_reference_date(&self) -> Option<&SourceValue> {
        self.source.get("inh_prd_inh_std")
    }

    /// Returns source field `inh_pym` when present.
    #[must_use]
    pub fn acquisition_payment_terms(&self) -> Option<&SourceValue> {
        self.source.get("inh_pym")
    }

    /// Returns source field `n6m_tpai_plann` when present.
    #[must_use]
    pub fn third_party_allotment_plan_within_six_months(&self) -> Option<&SourceValue> {
        self.source.get("n6m_tpai_plann")
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

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `sl_cmp_all` when present.
    #[must_use]
    pub fn company_total_revenue(&self) -> Option<&SourceValue> {
        self.source.get("sl_cmp_all")
    }

    /// Returns source field `sl_inh_bsn` when present.
    #[must_use]
    pub fn target_business_revenue(&self) -> Option<&SourceValue> {
        self.source.get("sl_inh_bsn")
    }

    /// Returns source field `sl_rt` when present.
    #[must_use]
    pub fn target_revenue_ratio(&self) -> Option<&SourceValue> {
        self.source.get("sl_rt")
    }
}

/// Opaque response for physical operation `get_bsnInhDecsn_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct BusinessAcquisitionDecisionsJsonResponse {
    source: SourceValue,
}

impl BusinessAcquisitionDecisionsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = BusinessAcquisitionDecision<'_>> + '_ {
        items(&self.source, "list", false).map(BusinessAcquisitionDecision::new)
    }
}

fn decode_business_acquisition_decisions_json_response(
    source: SourceValue,
) -> Result<BusinessAcquisitionDecisionsJsonResponse, ResponseDecodeError> {
    decode_business_acquisition_decisions_json_response_root(&source, "$".to_owned())?;
    Ok(BusinessAcquisitionDecisionsJsonResponse { source })
}

fn decode_business_acquisition_decisions_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_business_acquisition_decisions_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_business_acquisition_decisions_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_business_acquisition_decisions_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_business_acquisition_decisions_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_bsnInhDecsn_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct BusinessAcquisitionDecisionsXmlResponse {
    source: SourceValue,
}

impl BusinessAcquisitionDecisionsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = BusinessAcquisitionDecision<'_>> + '_ {
        items(&self.source, "list", true).map(BusinessAcquisitionDecision::new)
    }
}

fn decode_business_acquisition_decisions_xml_response(
    source: SourceValue,
) -> Result<BusinessAcquisitionDecisionsXmlResponse, ResponseDecodeError> {
    decode_business_acquisition_decisions_xml_response_root(&source, "$".to_owned())?;
    Ok(BusinessAcquisitionDecisionsXmlResponse { source })
}

fn decode_business_acquisition_decisions_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_business_acquisition_decisions_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_business_acquisition_decisions_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_business_acquisition_decisions_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_business_acquisition_decisions_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
