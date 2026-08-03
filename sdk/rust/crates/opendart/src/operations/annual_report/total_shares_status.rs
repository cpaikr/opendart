use super::*;

/// Inputs for logical operation `DS002-2020002`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct TotalSharesStatusInput {
    company_code: String,
    business_year: String,
    report_code: String,
}

impl TotalSharesStatusInput {
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

    /// Prepares physical operation `get_stockTotqySttus_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<TotalSharesStatusJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_stockTotqySttus_json", "DS002-2020002");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/stockTotqySttus.json", operation, &parameters),
            decode_total_shares_status_json_response,
        ))
    }

    /// Prepares physical operation `get_stockTotqySttus_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<TotalSharesStatusXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_stockTotqySttus_xml", "DS002-2020002");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/stockTotqySttus.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_total_shares_status_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct ShareCountSummary<'a> {
    source: &'a SourceValue,
}

impl<'a> ShareCountSummary<'a> {
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

    /// Returns source field `distb_stock_co` when present.
    #[must_use]
    pub fn shares_in_circulation(&self) -> Option<&SourceValue> {
        self.source.get("distb_stock_co")
    }

    /// Returns source field `etc` when present.
    #[must_use]
    pub fn other_reductions(&self) -> Option<&SourceValue> {
        self.source.get("etc")
    }

    /// Returns source field `istc_totqy` when present.
    #[must_use]
    pub fn issued_share_count(&self) -> Option<&SourceValue> {
        self.source.get("istc_totqy")
    }

    /// Returns source field `isu_stock_totqy` when present.
    #[must_use]
    pub fn authorized_share_count(&self) -> Option<&SourceValue> {
        self.source.get("isu_stock_totqy")
    }

    /// Returns source field `now_to_dcrs_stock_totqy` when present.
    #[must_use]
    pub fn cumulative_reduced_share_count(&self) -> Option<&SourceValue> {
        self.source.get("now_to_dcrs_stock_totqy")
    }

    /// Returns source field `now_to_isu_stock_totqy` when present.
    #[must_use]
    pub fn cumulative_issued_share_count(&self) -> Option<&SourceValue> {
        self.source.get("now_to_isu_stock_totqy")
    }

    /// Returns source field `profit_incnr` when present.
    #[must_use]
    pub fn profit_retirement_share_count(&self) -> Option<&SourceValue> {
        self.source.get("profit_incnr")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `rdmstk_repy` when present.
    #[must_use]
    pub fn redeemed_share_count(&self) -> Option<&SourceValue> {
        self.source.get("rdmstk_repy")
    }

    /// Returns source field `redc` when present.
    #[must_use]
    pub fn capital_reduction_share_count(&self) -> Option<&SourceValue> {
        self.source.get("redc")
    }

    /// Returns source field `se` when present.
    #[must_use]
    pub fn category(&self) -> Option<&SourceValue> {
        self.source.get("se")
    }

    /// Returns source field `stlm_dt` when present.
    #[must_use]
    pub fn fiscal_period_end_date(&self) -> Option<&SourceValue> {
        self.source.get("stlm_dt")
    }

    /// Returns source field `tesstk_co` when present.
    #[must_use]
    pub fn treasury_share_count(&self) -> Option<&SourceValue> {
        self.source.get("tesstk_co")
    }
}

/// Opaque response for physical operation `get_stockTotqySttus_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct TotalSharesStatusJsonResponse {
    source: SourceValue,
}

impl TotalSharesStatusJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = ShareCountSummary<'_>> + '_ {
        items(&self.source, "list", false).map(ShareCountSummary::new)
    }
}

fn decode_total_shares_status_json_response(
    source: SourceValue,
) -> Result<TotalSharesStatusJsonResponse, ResponseDecodeError> {
    decode_total_shares_status_json_response_root(&source, "$".to_owned())?;
    Ok(TotalSharesStatusJsonResponse { source })
}

fn decode_total_shares_status_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional("list", decode_total_shares_status_json_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_total_shares_status_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_total_shares_status_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_total_shares_status_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_stockTotqySttus_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct TotalSharesStatusXmlResponse {
    source: SourceValue,
}

impl TotalSharesStatusXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = ShareCountSummary<'_>> + '_ {
        items(&self.source, "list", true).map(ShareCountSummary::new)
    }
}

fn decode_total_shares_status_xml_response(
    source: SourceValue,
) -> Result<TotalSharesStatusXmlResponse, ResponseDecodeError> {
    decode_total_shares_status_xml_response_root(&source, "$".to_owned())?;
    Ok(TotalSharesStatusXmlResponse { source })
}

fn decode_total_shares_status_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional("list", decode_total_shares_status_xml_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_total_shares_status_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_total_shares_status_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_total_shares_status_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
