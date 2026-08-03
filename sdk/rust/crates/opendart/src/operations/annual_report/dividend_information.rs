use super::*;

/// Inputs for logical operation `DS002-2019005`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct DividendInformationInput {
    company_code: String,
    business_year: String,
    report_code: String,
}

impl DividendInformationInput {
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

    /// Prepares physical operation `get_alotMatter_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<DividendInformationJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_alotMatter_json", "DS002-2019005");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/alotMatter.json", operation, &parameters),
            decode_dividend_information_json_response,
        ))
    }

    /// Prepares physical operation `get_alotMatter_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<DividendInformationXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_alotMatter_xml", "DS002-2019005");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/alotMatter.xml", operation, &parameters, "result"),
            decode_dividend_information_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct DividendEntry<'a> {
    source: &'a SourceValue,
}

impl<'a> DividendEntry<'a> {
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

    /// Returns source field `frmtrm` when present.
    #[must_use]
    pub fn prior_period(&self) -> Option<&SourceValue> {
        self.source.get("frmtrm")
    }

    /// Returns source field `lwfr` when present.
    #[must_use]
    pub fn two_periods_prior(&self) -> Option<&SourceValue> {
        self.source.get("lwfr")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
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

    /// Returns source field `stock_knd` when present.
    #[must_use]
    pub fn share_class(&self) -> Option<&SourceValue> {
        self.source.get("stock_knd")
    }

    /// Returns source field `thstrm` when present.
    #[must_use]
    pub fn current_period(&self) -> Option<&SourceValue> {
        self.source.get("thstrm")
    }
}

/// Opaque response for physical operation `get_alotMatter_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct DividendInformationJsonResponse {
    source: SourceValue,
}

impl DividendInformationJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = DividendEntry<'_>> + '_ {
        items(&self.source, "list", false).map(DividendEntry::new)
    }
}

fn decode_dividend_information_json_response(
    source: SourceValue,
) -> Result<DividendInformationJsonResponse, ResponseDecodeError> {
    decode_dividend_information_json_response_root(&source, "$".to_owned())?;
    Ok(DividendInformationJsonResponse { source })
}

fn decode_dividend_information_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional("list", decode_dividend_information_json_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_dividend_information_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_dividend_information_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_dividend_information_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_alotMatter_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct DividendInformationXmlResponse {
    source: SourceValue,
}

impl DividendInformationXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = DividendEntry<'_>> + '_ {
        items(&self.source, "list", true).map(DividendEntry::new)
    }
}

fn decode_dividend_information_xml_response(
    source: SourceValue,
) -> Result<DividendInformationXmlResponse, ResponseDecodeError> {
    decode_dividend_information_xml_response_root(&source, "$".to_owned())?;
    Ok(DividendInformationXmlResponse { source })
}

fn decode_dividend_information_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional("list", decode_dividend_information_xml_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_dividend_information_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_dividend_information_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_dividend_information_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
