use super::*;

/// Inputs for logical operation `DS005-2020047`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct OtherCompanyEquitySecuritiesTransferDecisionsInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl OtherCompanyEquitySecuritiesTransferDecisionsInput {
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

    /// Prepares physical operation `get_otcprStkInvscrTrfDecsn_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<
        PreparedRequest<OtherCompanyEquitySecuritiesTransferDecisionsJsonResponse>,
        PrepareError,
    > {
        let operation = OperationIdentity::new("get_otcprStkInvscrTrfDecsn_json", "DS005-2020047");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json(
                "/api/otcprStkInvscrTrfDecsn.json",
                operation,
                &parameters,
            ),
            decode_other_company_equity_securities_transfer_decisions_json_response,
        ))
    }

    /// Prepares physical operation `get_otcprStkInvscrTrfDecsn_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<
        PreparedRequest<OtherCompanyEquitySecuritiesTransferDecisionsXmlResponse>,
        PrepareError,
    > {
        let operation = OperationIdentity::new("get_otcprStkInvscrTrfDecsn_xml", "DS005-2020047");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/otcprStkInvscrTrfDecsn.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_other_company_equity_securities_transfer_decisions_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct OtherCompanyEquitySecuritiesTransferDecision<'a> {
    source: &'a SourceValue,
}

impl<'a> OtherCompanyEquitySecuritiesTransferDecision<'a> {
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

    /// Returns source field `attrf_eqrt` when present.
    #[must_use]
    pub fn ownership_percentage_after_transfer(&self) -> Option<&SourceValue> {
        self.source.get("attrf_eqrt")
    }

    /// Returns source field `attrf_owstkcnt` when present.
    #[must_use]
    pub fn owned_shares_after_transfer(&self) -> Option<&SourceValue> {
        self.source.get("attrf_owstkcnt")
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

    /// Returns source field `iscmp_cmpnm` when present.
    #[must_use]
    pub fn issuer_company_name(&self) -> Option<&SourceValue> {
        self.source.get("iscmp_cmpnm")
    }

    /// Returns source field `iscmp_cpt` when present.
    #[must_use]
    pub fn issuer_capital(&self) -> Option<&SourceValue> {
        self.source.get("iscmp_cpt")
    }

    /// Returns source field `iscmp_mbsn` when present.
    #[must_use]
    pub fn issuer_principal_business(&self) -> Option<&SourceValue> {
        self.source.get("iscmp_mbsn")
    }

    /// Returns source field `iscmp_nt` when present.
    #[must_use]
    pub fn issuer_nationality(&self) -> Option<&SourceValue> {
        self.source.get("iscmp_nt")
    }

    /// Returns source field `iscmp_rl_cmpn` when present.
    #[must_use]
    pub fn issuer_relationship_to_company(&self) -> Option<&SourceValue> {
        self.source.get("iscmp_rl_cmpn")
    }

    /// Returns source field `iscmp_rp` when present.
    #[must_use]
    pub fn issuer_representative(&self) -> Option<&SourceValue> {
        self.source.get("iscmp_rp")
    }

    /// Returns source field `iscmp_tisstk` when present.
    #[must_use]
    pub fn issuer_total_issued_shares(&self) -> Option<&SourceValue> {
        self.source.get("iscmp_tisstk")
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

    /// Returns source field `trf_pp` when present.
    #[must_use]
    pub fn transfer_purpose(&self) -> Option<&SourceValue> {
        self.source.get("trf_pp")
    }

    /// Returns source field `trf_prd` when present.
    #[must_use]
    pub fn planned_transfer_date(&self) -> Option<&SourceValue> {
        self.source.get("trf_prd")
    }

    /// Returns source field `trfdtl_ecpt` when present.
    #[must_use]
    pub fn equity(&self) -> Option<&SourceValue> {
        self.source.get("trfdtl_ecpt")
    }

    /// Returns source field `trfdtl_ecpt_vs` when present.
    #[must_use]
    pub fn transfer_amount_to_equity_ratio(&self) -> Option<&SourceValue> {
        self.source.get("trfdtl_ecpt_vs")
    }

    /// Returns source field `trfdtl_stkcnt` when present.
    #[must_use]
    pub fn transferred_share_count(&self) -> Option<&SourceValue> {
        self.source.get("trfdtl_stkcnt")
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

/// Opaque response for physical operation `get_otcprStkInvscrTrfDecsn_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct OtherCompanyEquitySecuritiesTransferDecisionsJsonResponse {
    source: SourceValue,
}

impl OtherCompanyEquitySecuritiesTransferDecisionsJsonResponse {
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
    pub fn items(
        &self,
    ) -> impl Iterator<Item = OtherCompanyEquitySecuritiesTransferDecision<'_>> + '_ {
        items(&self.source, "list", false).map(OtherCompanyEquitySecuritiesTransferDecision::new)
    }
}

fn decode_other_company_equity_securities_transfer_decisions_json_response(
    source: SourceValue,
) -> Result<OtherCompanyEquitySecuritiesTransferDecisionsJsonResponse, ResponseDecodeError> {
    decode_other_company_equity_securities_transfer_decisions_json_response_root(
        &source,
        "$".to_owned(),
    )?;
    Ok(OtherCompanyEquitySecuritiesTransferDecisionsJsonResponse { source })
}

fn decode_other_company_equity_securities_transfer_decisions_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_other_company_equity_securities_transfer_decisions_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_other_company_equity_securities_transfer_decisions_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_other_company_equity_securities_transfer_decisions_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_other_company_equity_securities_transfer_decisions_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_otcprStkInvscrTrfDecsn_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct OtherCompanyEquitySecuritiesTransferDecisionsXmlResponse {
    source: SourceValue,
}

impl OtherCompanyEquitySecuritiesTransferDecisionsXmlResponse {
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
    pub fn items(
        &self,
    ) -> impl Iterator<Item = OtherCompanyEquitySecuritiesTransferDecision<'_>> + '_ {
        items(&self.source, "list", true).map(OtherCompanyEquitySecuritiesTransferDecision::new)
    }
}

fn decode_other_company_equity_securities_transfer_decisions_xml_response(
    source: SourceValue,
) -> Result<OtherCompanyEquitySecuritiesTransferDecisionsXmlResponse, ResponseDecodeError> {
    decode_other_company_equity_securities_transfer_decisions_xml_response_root(
        &source,
        "$".to_owned(),
    )?;
    Ok(OtherCompanyEquitySecuritiesTransferDecisionsXmlResponse { source })
}

fn decode_other_company_equity_securities_transfer_decisions_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_other_company_equity_securities_transfer_decisions_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_other_company_equity_securities_transfer_decisions_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_other_company_equity_securities_transfer_decisions_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_other_company_equity_securities_transfer_decisions_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
