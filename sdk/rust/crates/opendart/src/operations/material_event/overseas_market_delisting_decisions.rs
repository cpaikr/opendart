use super::*;

/// Inputs for logical operation `DS005-2020030`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct OverseasMarketDelistingDecisionsInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl OverseasMarketDelistingDecisionsInput {
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

    /// Prepares physical operation `get_ovDlstDecsn_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<OverseasMarketDelistingDecisionsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_ovDlstDecsn_json", "DS005-2020030");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/ovDlstDecsn.json", operation, &parameters),
            decode_overseas_market_delisting_decisions_json_response,
        ))
    }

    /// Prepares physical operation `get_ovDlstDecsn_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<OverseasMarketDelistingDecisionsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_ovDlstDecsn_xml", "DS005-2020030");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/ovDlstDecsn.xml", operation, &parameters, "result"),
            decode_overseas_market_delisting_decisions_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct OverseasMarketDelistingDecision<'a> {
    source: &'a SourceValue,
}

impl<'a> OverseasMarketDelistingDecision<'a> {
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

    /// Returns source field `dlst_prd` when present.
    #[must_use]
    pub fn delisting_date(&self) -> Option<&SourceValue> {
        self.source.get("dlst_prd")
    }

    /// Returns source field `dlst_rs` when present.
    #[must_use]
    pub fn delisting_reason(&self) -> Option<&SourceValue> {
        self.source.get("dlst_rs")
    }

    /// Returns source field `dlstrq_prd` when present.
    #[must_use]
    pub fn delisting_application_planned_date(&self) -> Option<&SourceValue> {
        self.source.get("dlstrq_prd")
    }

    /// Returns source field `dlststk_estk_cnt` when present.
    #[must_use]
    pub fn delisted_other_share_count(&self) -> Option<&SourceValue> {
        self.source.get("dlststk_estk_cnt")
    }

    /// Returns source field `dlststk_ostk_cnt` when present.
    #[must_use]
    pub fn delisted_common_share_count(&self) -> Option<&SourceValue> {
        self.source.get("dlststk_ostk_cnt")
    }

    /// Returns source field `lstex_nt` when present.
    #[must_use]
    pub fn listing_exchange_and_country(&self) -> Option<&SourceValue> {
        self.source.get("lstex_nt")
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

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }
}

/// Opaque response for physical operation `get_ovDlstDecsn_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct OverseasMarketDelistingDecisionsJsonResponse {
    source: SourceValue,
}

impl OverseasMarketDelistingDecisionsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = OverseasMarketDelistingDecision<'_>> + '_ {
        items(&self.source, "list", false).map(OverseasMarketDelistingDecision::new)
    }
}

fn decode_overseas_market_delisting_decisions_json_response(
    source: SourceValue,
) -> Result<OverseasMarketDelistingDecisionsJsonResponse, ResponseDecodeError> {
    decode_overseas_market_delisting_decisions_json_response_root(&source, "$".to_owned())?;
    Ok(OverseasMarketDelistingDecisionsJsonResponse { source })
}

fn decode_overseas_market_delisting_decisions_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_overseas_market_delisting_decisions_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_overseas_market_delisting_decisions_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_overseas_market_delisting_decisions_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_overseas_market_delisting_decisions_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_ovDlstDecsn_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct OverseasMarketDelistingDecisionsXmlResponse {
    source: SourceValue,
}

impl OverseasMarketDelistingDecisionsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = OverseasMarketDelistingDecision<'_>> + '_ {
        items(&self.source, "list", true).map(OverseasMarketDelistingDecision::new)
    }
}

fn decode_overseas_market_delisting_decisions_xml_response(
    source: SourceValue,
) -> Result<OverseasMarketDelistingDecisionsXmlResponse, ResponseDecodeError> {
    decode_overseas_market_delisting_decisions_xml_response_root(&source, "$".to_owned())?;
    Ok(OverseasMarketDelistingDecisionsXmlResponse { source })
}

fn decode_overseas_market_delisting_decisions_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_overseas_market_delisting_decisions_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_overseas_market_delisting_decisions_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_overseas_market_delisting_decisions_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_overseas_market_delisting_decisions_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
