use super::*;

/// Inputs for logical operation `DS002-2026002`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct TopFiveIndividualCompensationV2Input {
    company_code: String,
    business_year: String,
    report_code: String,
}

impl TopFiveIndividualCompensationV2Input {
    /// Creates inputs from all required source-shaped values.
    #[must_use]
    pub fn new(company_code: String, business_year: String, report_code: String) -> Self {
        Self {
            company_code,
            business_year,
            report_code,
        }
    }

    fn parameters(
        &self,
        operation: OperationIdentity,
    ) -> Result<Vec<crate::request::QueryParameter<'_>>, PrepareError> {
        let mut query = Query::new(operation);
        let value = CompanyCode::new(operation, "corp_code", &self.company_code)?;
        query.required("corp_code", value.as_str())?;
        let value = BusinessYear::new(operation, "bsns_year", &self.business_year)?;
        query.required("bsns_year", value.as_str())?;
        let value = ReportCode::new(operation, "reprt_code", &self.report_code)?;
        query.required("reprt_code", value.as_str())?;
        Ok(query.finish())
    }

    /// Prepares physical operation `get_indvdlByPayV2_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<TopFiveIndividualCompensationV2JsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_indvdlByPayV2_json", "DS002-2026002");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/indvdlByPayV2.json", operation, &parameters),
            decode_top_five_individual_compensation_v2_json_response,
        ))
    }

    /// Prepares physical operation `get_indvdlByPayV2_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<TopFiveIndividualCompensationV2XmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_indvdlByPayV2_xml", "DS002-2026002");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/indvdlByPayV2.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_top_five_individual_compensation_v2_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.group[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct TopFiveCompensationRecipient<'a> {
    source: &'a SourceValue,
    xml: bool,
}

impl<'a> TopFiveCompensationRecipient<'a> {
    fn new(source: &'a SourceValue, xml: bool) -> Self {
        Self { source, xml }
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

    /// Returns source field `fscl_year` when present.
    #[must_use]
    pub fn fiscal_year(&self) -> Option<&SourceValue> {
        self.source.get("fscl_year")
    }

    /// Returns source field `mendng_totamt` when present.
    #[must_use]
    pub fn total_compensation(&self) -> Option<&SourceValue> {
        self.source.get("mendng_totamt")
    }

    /// Returns source field `nm` when present.
    #[must_use]
    pub fn name(&self) -> Option<&SourceValue> {
        self.source.get("nm")
    }

    /// Returns source field `ofcps` when present.
    #[must_use]
    pub fn position(&self) -> Option<&SourceValue> {
        self.source.get("ofcps")
    }

    /// Iterates the reviewed `$.group[].list[]` child views.
    pub fn items(&self) -> impl Iterator<Item = TopFiveEquityCompensationDetail<'a>> + '_ {
        items(self.source, "list", self.xml).map(TopFiveEquityCompensationDetail::new)
    }
}

/// Borrowed semantic view over `$.group[].list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct TopFiveEquityCompensationDetail<'a> {
    source: &'a SourceValue,
}

impl<'a> TopFiveEquityCompensationDetail<'a> {
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

    /// Returns source field `othr_stk_bsd_cmpn_mkt_vl` when present.
    #[must_use]
    pub fn other_unpaid_stock_compensation_market_value(&self) -> Option<&SourceValue> {
        self.source.get("othr_stk_bsd_cmpn_mkt_vl")
    }

    /// Returns source field `othr_stk_bsd_cmpn_unpyd_qty` when present.
    #[must_use]
    pub fn other_unpaid_stock_compensation_quantity(&self) -> Option<&SourceValue> {
        self.source.get("othr_stk_bsd_cmpn_unpyd_qty")
    }

    /// Returns source field `rm` when present.
    #[must_use]
    pub fn remarks(&self) -> Option<&SourceValue> {
        self.source.get("rm")
    }

    /// Returns source field `stk_bsd_pd_mendng_totamt_amt` when present.
    #[must_use]
    pub fn stock_based_compensation_amount(&self) -> Option<&SourceValue> {
        self.source.get("stk_bsd_pd_mendng_totamt_amt")
    }

    /// Returns source field `stk_bsd_pd_mendng_totamt_knd` when present.
    #[must_use]
    pub fn stock_based_compensation_type(&self) -> Option<&SourceValue> {
        self.source.get("stk_bsd_pd_mendng_totamt_knd")
    }

    /// Returns source field `stk_bsd_pd_mendng_totamt_qty` when present.
    #[must_use]
    pub fn stock_based_compensation_quantity(&self) -> Option<&SourceValue> {
        self.source.get("stk_bsd_pd_mendng_totamt_qty")
    }

    /// Returns source field `stk_opt_exrc_pr` when present.
    #[must_use]
    pub fn stock_option_exercise_price(&self) -> Option<&SourceValue> {
        self.source.get("stk_opt_exrc_pr")
    }

    /// Returns source field `stk_opt_exrcsbl_qty` when present.
    #[must_use]
    pub fn exercisable_stock_option_quantity(&self) -> Option<&SourceValue> {
        self.source.get("stk_opt_exrcsbl_qty")
    }

    /// Returns source field `stk_opt_rmn_blce` when present.
    #[must_use]
    pub fn remaining_stock_option_value(&self) -> Option<&SourceValue> {
        self.source.get("stk_opt_rmn_blce")
    }

    /// Returns source field `stk_opt_unexrcsbl_qty` when present.
    #[must_use]
    pub fn unexercisable_stock_option_quantity(&self) -> Option<&SourceValue> {
        self.source.get("stk_opt_unexrcsbl_qty")
    }
}

/// Opaque response for physical operation `get_indvdlByPayV2_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct TopFiveIndividualCompensationV2JsonResponse {
    source: SourceValue,
}

impl TopFiveIndividualCompensationV2JsonResponse {
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

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `stlm_dt` when present.
    #[must_use]
    pub fn fiscal_period_end_date(&self) -> Option<&SourceValue> {
        self.source.get("stlm_dt")
    }

    /// Iterates the reviewed `$.group[]` item views.
    pub fn items(&self) -> impl Iterator<Item = TopFiveCompensationRecipient<'_>> + '_ {
        items(&self.source, "group", false)
            .map(|source| TopFiveCompensationRecipient::new(source, false))
    }
}

fn decode_top_five_individual_compensation_v2_json_response(
    source: SourceValue,
) -> Result<TopFiveIndividualCompensationV2JsonResponse, ResponseDecodeError> {
    decode_top_five_individual_compensation_v2_json_response_root(&source, "$".to_owned())?;
    Ok(TopFiveIndividualCompensationV2JsonResponse { source })
}

fn decode_top_five_individual_compensation_v2_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "group",
        decode_top_five_individual_compensation_v2_json_response_root_group,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_top_five_individual_compensation_v2_json_response_root_group(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_top_five_individual_compensation_v2_json_response_root_group_item,
    )?;
    Ok(())
}

fn decode_top_five_individual_compensation_v2_json_response_root_group_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_top_five_individual_compensation_v2_json_response_root_group_item_list,
    )?;
    Ok(())
}

fn decode_top_five_individual_compensation_v2_json_response_root_group_item_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_top_five_individual_compensation_v2_json_response_root_group_item_list_item,
    )?;
    Ok(())
}

fn decode_top_five_individual_compensation_v2_json_response_root_group_item_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_indvdlByPayV2_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct TopFiveIndividualCompensationV2XmlResponse {
    source: SourceValue,
}

impl TopFiveIndividualCompensationV2XmlResponse {
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

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `stlm_dt` when present.
    #[must_use]
    pub fn fiscal_period_end_date(&self) -> Option<&SourceValue> {
        self.source.get("stlm_dt")
    }

    /// Iterates the reviewed `$.group[]` item views.
    pub fn items(&self) -> impl Iterator<Item = TopFiveCompensationRecipient<'_>> + '_ {
        items(&self.source, "group", true)
            .map(|source| TopFiveCompensationRecipient::new(source, true))
    }
}

fn decode_top_five_individual_compensation_v2_xml_response(
    source: SourceValue,
) -> Result<TopFiveIndividualCompensationV2XmlResponse, ResponseDecodeError> {
    decode_top_five_individual_compensation_v2_xml_response_root(&source, "$".to_owned())?;
    Ok(TopFiveIndividualCompensationV2XmlResponse { source })
}

fn decode_top_five_individual_compensation_v2_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "group",
        decode_top_five_individual_compensation_v2_xml_response_root_group,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_top_five_individual_compensation_v2_xml_response_root_group(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_top_five_individual_compensation_v2_xml_response_root_group_item,
    )?;
    Ok(())
}

fn decode_top_five_individual_compensation_v2_xml_response_root_group_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_top_five_individual_compensation_v2_xml_response_root_group_item_list,
    )?;
    Ok(())
}

fn decode_top_five_individual_compensation_v2_xml_response_root_group_item_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_top_five_individual_compensation_v2_xml_response_root_group_item_list_item,
    )?;
    Ok(())
}

fn decode_top_five_individual_compensation_v2_xml_response_root_group_item_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
