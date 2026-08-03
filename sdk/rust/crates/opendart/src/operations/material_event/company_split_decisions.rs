use super::*;

/// Inputs for logical operation `DS005-2020051`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct CompanySplitDecisionsInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl CompanySplitDecisionsInput {
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

    /// Prepares physical operation `get_cmpDvDecsn_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<CompanySplitDecisionsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_cmpDvDecsn_json", "DS005-2020051");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/cmpDvDecsn.json", operation, &parameters),
            decode_company_split_decisions_json_response,
        ))
    }

    /// Prepares physical operation `get_cmpDvDecsn_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<CompanySplitDecisionsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_cmpDvDecsn_xml", "DS005-2020051");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/cmpDvDecsn.xml", operation, &parameters, "result"),
            decode_company_split_decisions_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct CompanySplitDecision<'a> {
    source: &'a SourceValue,
}

impl<'a> CompanySplitDecision<'a> {
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

    /// Returns source field `abcr_crrt` when present.
    #[must_use]
    pub fn capital_reduction_ratio(&self) -> Option<&SourceValue> {
        self.source.get("abcr_crrt")
    }

    /// Returns source field `abcr_nstkascnd` when present.
    #[must_use]
    pub fn capital_reduction_new_share_allocation_conditions(&self) -> Option<&SourceValue> {
        self.source.get("abcr_nstkascnd")
    }

    /// Returns source field `abcr_nstkasstd` when present.
    #[must_use]
    pub fn capital_reduction_new_share_record_date(&self) -> Option<&SourceValue> {
        self.source.get("abcr_nstkasstd")
    }

    /// Returns source field `abcr_nstkdlprd` when present.
    #[must_use]
    pub fn planned_capital_reduction_new_share_certificate_delivery_date(
        &self,
    ) -> Option<&SourceValue> {
        self.source.get("abcr_nstkdlprd")
    }

    /// Returns source field `abcr_nstklstprd` when present.
    #[must_use]
    pub fn planned_capital_reduction_new_share_listing_date(&self) -> Option<&SourceValue> {
        self.source.get("abcr_nstklstprd")
    }

    /// Returns source field `abcr_osprpd_bgd` when present.
    #[must_use]
    pub fn capital_reduction_old_share_certificate_submission_start_date(
        &self,
    ) -> Option<&SourceValue> {
        self.source.get("abcr_osprpd_bgd")
    }

    /// Returns source field `abcr_osprpd_edd` when present.
    #[must_use]
    pub fn capital_reduction_old_share_certificate_submission_end_date(
        &self,
    ) -> Option<&SourceValue> {
        self.source.get("abcr_osprpd_edd")
    }

    /// Returns source field `abcr_shstkcnt_rt_at_rs` when present.
    #[must_use]
    pub fn capital_reduction_proportional_allocation_and_reason(&self) -> Option<&SourceValue> {
        self.source.get("abcr_shstkcnt_rt_at_rs")
    }

    /// Returns source field `abcr_trspprpd_bgd` when present.
    #[must_use]
    pub fn capital_reduction_trading_suspension_start_date(&self) -> Option<&SourceValue> {
        self.source.get("abcr_trspprpd_bgd")
    }

    /// Returns source field `abcr_trspprpd_edd` when present.
    #[must_use]
    pub fn capital_reduction_trading_suspension_end_date(&self) -> Option<&SourceValue> {
        self.source.get("abcr_trspprpd_edd")
    }

    /// Returns source field `adt_a_atn` when present.
    #[must_use]
    pub fn auditor_attendance_status(&self) -> Option<&SourceValue> {
        self.source.get("adt_a_atn")
    }

    /// Returns source field `atdv_excmp_atdv_lstmn_atn` when present.
    #[must_use]
    pub fn surviving_company_remains_listed(&self) -> Option<&SourceValue> {
        self.source.get("atdv_excmp_atdv_lstmn_atn")
    }

    /// Returns source field `atdv_excmp_cmpnm` when present.
    #[must_use]
    pub fn surviving_company_name(&self) -> Option<&SourceValue> {
        self.source.get("atdv_excmp_cmpnm")
    }

    /// Returns source field `atdv_excmp_exbsn_rsl` when present.
    #[must_use]
    pub fn surviving_business_segment_prior_year_revenue(&self) -> Option<&SourceValue> {
        self.source.get("atdv_excmp_exbsn_rsl")
    }

    /// Returns source field `atdv_excmp_mbsn` when present.
    #[must_use]
    pub fn surviving_company_principal_business(&self) -> Option<&SourceValue> {
        self.source.get("atdv_excmp_mbsn")
    }

    /// Returns source field `atdvfdtl_cpt` when present.
    #[must_use]
    pub fn surviving_company_capital(&self) -> Option<&SourceValue> {
        self.source.get("atdvfdtl_cpt")
    }

    /// Returns source field `atdvfdtl_std` when present.
    #[must_use]
    pub fn surviving_company_financials_as_of(&self) -> Option<&SourceValue> {
        self.source.get("atdvfdtl_std")
    }

    /// Returns source field `atdvfdtl_tast` when present.
    #[must_use]
    pub fn surviving_company_total_assets(&self) -> Option<&SourceValue> {
        self.source.get("atdvfdtl_tast")
    }

    /// Returns source field `atdvfdtl_tdbt` when present.
    #[must_use]
    pub fn surviving_company_total_liabilities(&self) -> Option<&SourceValue> {
        self.source.get("atdvfdtl_tdbt")
    }

    /// Returns source field `atdvfdtl_teqt` when present.
    #[must_use]
    pub fn surviving_company_total_equity(&self) -> Option<&SourceValue> {
        self.source.get("atdvfdtl_teqt")
    }

    /// Returns source field `bddd` when present.
    #[must_use]
    pub fn board_resolution_date(&self) -> Option<&SourceValue> {
        self.source.get("bddd")
    }

    /// Returns source field `cdobprpd_bgd` when present.
    #[must_use]
    pub fn creditor_objection_start_date(&self) -> Option<&SourceValue> {
        self.source.get("cdobprpd_bgd")
    }

    /// Returns source field `cdobprpd_edd` when present.
    #[must_use]
    pub fn creditor_objection_end_date(&self) -> Option<&SourceValue> {
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

    /// Returns source field `dv_impef` when present.
    #[must_use]
    pub fn split_material_impact_and_effect(&self) -> Option<&SourceValue> {
        self.source.get("dv_impef")
    }

    /// Returns source field `dv_mth` when present.
    #[must_use]
    pub fn split_method(&self) -> Option<&SourceValue> {
        self.source.get("dv_mth")
    }

    /// Returns source field `dv_rt` when present.
    #[must_use]
    pub fn split_ratio(&self) -> Option<&SourceValue> {
        self.source.get("dv_rt")
    }

    /// Returns source field `dv_trfbsnprt_cn` when present.
    #[must_use]
    pub fn business_and_property_transferred_by_split(&self) -> Option<&SourceValue> {
        self.source.get("dv_trfbsnprt_cn")
    }

    /// Returns source field `dvdt` when present.
    #[must_use]
    pub fn split_date(&self) -> Option<&SourceValue> {
        self.source.get("dvdt")
    }

    /// Returns source field `dvfcmp_cmpnm` when present.
    #[must_use]
    pub fn new_split_company_name(&self) -> Option<&SourceValue> {
        self.source.get("dvfcmp_cmpnm")
    }

    /// Returns source field `dvfcmp_mbsn` when present.
    #[must_use]
    pub fn new_split_company_principal_business(&self) -> Option<&SourceValue> {
        self.source.get("dvfcmp_mbsn")
    }

    /// Returns source field `dvfcmp_nbsn_rsl` when present.
    #[must_use]
    pub fn new_business_segment_prior_year_revenue(&self) -> Option<&SourceValue> {
        self.source.get("dvfcmp_nbsn_rsl")
    }

    /// Returns source field `dvfcmp_rlst_atn` when present.
    #[must_use]
    pub fn new_split_company_relisting_application_status(&self) -> Option<&SourceValue> {
        self.source.get("dvfcmp_rlst_atn")
    }

    /// Returns source field `dvrgsprd` when present.
    #[must_use]
    pub fn planned_split_registration_date(&self) -> Option<&SourceValue> {
        self.source.get("dvrgsprd")
    }

    /// Returns source field `ex_sm_r` when present.
    #[must_use]
    pub fn securities_registration_statement_exemption_reason(&self) -> Option<&SourceValue> {
        self.source.get("ex_sm_r")
    }

    /// Returns source field `ffdtl_cpt` when present.
    #[must_use]
    pub fn new_split_company_capital(&self) -> Option<&SourceValue> {
        self.source.get("ffdtl_cpt")
    }

    /// Returns source field `ffdtl_std` when present.
    #[must_use]
    pub fn new_split_company_financials_as_of(&self) -> Option<&SourceValue> {
        self.source.get("ffdtl_std")
    }

    /// Returns source field `ffdtl_tast` when present.
    #[must_use]
    pub fn new_split_company_total_assets(&self) -> Option<&SourceValue> {
        self.source.get("ffdtl_tast")
    }

    /// Returns source field `ffdtl_tdbt` when present.
    #[must_use]
    pub fn new_split_company_total_liabilities(&self) -> Option<&SourceValue> {
        self.source.get("ffdtl_tdbt")
    }

    /// Returns source field `ffdtl_teqt` when present.
    #[must_use]
    pub fn new_split_company_total_equity(&self) -> Option<&SourceValue> {
        self.source.get("ffdtl_teqt")
    }

    /// Returns source field `gmtsck_prd` when present.
    #[must_use]
    pub fn planned_shareholders_meeting_date(&self) -> Option<&SourceValue> {
        self.source.get("gmtsck_prd")
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

    /// Returns source field `rs_sm_atn` when present.
    #[must_use]
    pub fn securities_registration_statement_required(&self) -> Option<&SourceValue> {
        self.source.get("rs_sm_atn")
    }
}

/// Opaque response for physical operation `get_cmpDvDecsn_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct CompanySplitDecisionsJsonResponse {
    source: SourceValue,
}

impl CompanySplitDecisionsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = CompanySplitDecision<'_>> + '_ {
        items(&self.source, "list", false).map(CompanySplitDecision::new)
    }
}

fn decode_company_split_decisions_json_response(
    source: SourceValue,
) -> Result<CompanySplitDecisionsJsonResponse, ResponseDecodeError> {
    decode_company_split_decisions_json_response_root(&source, "$".to_owned())?;
    Ok(CompanySplitDecisionsJsonResponse { source })
}

fn decode_company_split_decisions_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_company_split_decisions_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_company_split_decisions_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_company_split_decisions_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_company_split_decisions_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_cmpDvDecsn_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct CompanySplitDecisionsXmlResponse {
    source: SourceValue,
}

impl CompanySplitDecisionsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = CompanySplitDecision<'_>> + '_ {
        items(&self.source, "list", true).map(CompanySplitDecision::new)
    }
}

fn decode_company_split_decisions_xml_response(
    source: SourceValue,
) -> Result<CompanySplitDecisionsXmlResponse, ResponseDecodeError> {
    decode_company_split_decisions_xml_response_root(&source, "$".to_owned())?;
    Ok(CompanySplitDecisionsXmlResponse { source })
}

fn decode_company_split_decisions_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_company_split_decisions_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_company_split_decisions_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_company_split_decisions_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_company_split_decisions_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
