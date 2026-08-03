use super::*;

/// Inputs for logical operation `DS005-2020031`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct OverseasMarketListingsInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl OverseasMarketListingsInput {
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

    /// Prepares physical operation `get_ovLst_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<OverseasMarketListingsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_ovLst_json", "DS005-2020031");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/ovLst.json", operation, &parameters),
            decode_overseas_market_listings_json_response,
        ))
    }

    /// Prepares physical operation `get_ovLst_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<OverseasMarketListingsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_ovLst_xml", "DS005-2020031");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/ovLst.xml", operation, &parameters, "result"),
            decode_overseas_market_listings_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct OverseasMarketListing<'a> {
    source: &'a SourceValue,
}

impl<'a> OverseasMarketListing<'a> {
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

    /// Returns source field `cfd` when present.
    #[must_use]
    pub fn confirmation_date(&self) -> Option<&SourceValue> {
        self.source.get("cfd")
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

    /// Returns source field `lstd` when present.
    #[must_use]
    pub fn listing_date(&self) -> Option<&SourceValue> {
        self.source.get("lstd")
    }

    /// Returns source field `lstex_nt` when present.
    #[must_use]
    pub fn listing_exchange_and_country(&self) -> Option<&SourceValue> {
        self.source.get("lstex_nt")
    }

    /// Returns source field `lststk_estk_cnt` when present.
    #[must_use]
    pub fn listed_other_share_count(&self) -> Option<&SourceValue> {
        self.source.get("lststk_estk_cnt")
    }

    /// Returns source field `lststk_ostk_cnt` when present.
    #[must_use]
    pub fn listed_common_share_count(&self) -> Option<&SourceValue> {
        self.source.get("lststk_ostk_cnt")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `stk_cd` when present.
    #[must_use]
    pub fn security_name_and_code(&self) -> Option<&SourceValue> {
        self.source.get("stk_cd")
    }
}

/// Opaque response for physical operation `get_ovLst_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct OverseasMarketListingsJsonResponse {
    source: SourceValue,
}

impl OverseasMarketListingsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = OverseasMarketListing<'_>> + '_ {
        items(&self.source, "list", false).map(OverseasMarketListing::new)
    }
}

fn decode_overseas_market_listings_json_response(
    source: SourceValue,
) -> Result<OverseasMarketListingsJsonResponse, ResponseDecodeError> {
    decode_overseas_market_listings_json_response_root(&source, "$".to_owned())?;
    Ok(OverseasMarketListingsJsonResponse { source })
}

fn decode_overseas_market_listings_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_overseas_market_listings_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_overseas_market_listings_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_overseas_market_listings_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_overseas_market_listings_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_ovLst_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct OverseasMarketListingsXmlResponse {
    source: SourceValue,
}

impl OverseasMarketListingsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = OverseasMarketListing<'_>> + '_ {
        items(&self.source, "list", true).map(OverseasMarketListing::new)
    }
}

fn decode_overseas_market_listings_xml_response(
    source: SourceValue,
) -> Result<OverseasMarketListingsXmlResponse, ResponseDecodeError> {
    decode_overseas_market_listings_xml_response_root(&source, "$".to_owned())?;
    Ok(OverseasMarketListingsXmlResponse { source })
}

fn decode_overseas_market_listings_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_overseas_market_listings_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_overseas_market_listings_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_overseas_market_listings_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_overseas_market_listings_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
