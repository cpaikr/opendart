use super::*;

/// Inputs for logical operation `DS004-2019022`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct InsiderShareholdingsInput {
    company_code: String,
}

impl InsiderShareholdingsInput {
    /// Creates inputs from all required source-shaped values.
    #[must_use]
    pub fn new(company_code: String) -> Self {
        Self { company_code }
    }

    fn parameters(
        &self,
        operation: OperationIdentity,
    ) -> Result<Vec<crate::request::QueryParameter<'_>>, PrepareError> {
        let mut query = Query::new(operation);
        let value = CompanyCode::new(operation, "corp_code", &self.company_code)?;
        query.required("corp_code", value.as_str())?;
        Ok(query.finish())
    }

    /// Prepares physical operation `get_elestock_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<InsiderShareholdingsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_elestock_json", "DS004-2019022");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/elestock.json", operation, &parameters),
            decode_insider_shareholdings_json_response,
        ))
    }

    /// Prepares physical operation `get_elestock_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<InsiderShareholdingsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_elestock_xml", "DS004-2019022");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/elestock.xml", operation, &parameters, "result"),
            decode_insider_shareholdings_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct InsiderShareholding<'a> {
    source: &'a SourceValue,
}

impl<'a> InsiderShareholding<'a> {
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

    /// Returns source field `isu_exctv_ofcps` when present.
    #[must_use]
    pub fn executive_position(&self) -> Option<&SourceValue> {
        self.source.get("isu_exctv_ofcps")
    }

    /// Returns source field `isu_exctv_rgist_at` when present.
    #[must_use]
    pub fn registered_executive_status(&self) -> Option<&SourceValue> {
        self.source.get("isu_exctv_rgist_at")
    }

    /// Returns source field `isu_main_shrholdr` when present.
    #[must_use]
    pub fn major_shareholder_status(&self) -> Option<&SourceValue> {
        self.source.get("isu_main_shrholdr")
    }

    /// Returns source field `rcept_dt` when present.
    #[must_use]
    pub fn receipt_date(&self) -> Option<&SourceValue> {
        self.source.get("rcept_dt")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `repror` when present.
    #[must_use]
    pub fn reporting_person(&self) -> Option<&SourceValue> {
        self.source.get("repror")
    }

    /// Returns source field `sp_stock_lmp_cnt` when present.
    #[must_use]
    pub fn specific_securities_held(&self) -> Option<&SourceValue> {
        self.source.get("sp_stock_lmp_cnt")
    }

    /// Returns source field `sp_stock_lmp_irds_cnt` when present.
    #[must_use]
    pub fn specific_securities_change(&self) -> Option<&SourceValue> {
        self.source.get("sp_stock_lmp_irds_cnt")
    }

    /// Returns source field `sp_stock_lmp_irds_rate` when present.
    #[must_use]
    pub fn specific_securities_percentage_change(&self) -> Option<&SourceValue> {
        self.source.get("sp_stock_lmp_irds_rate")
    }

    /// Returns source field `sp_stock_lmp_rate` when present.
    #[must_use]
    pub fn specific_securities_percentage(&self) -> Option<&SourceValue> {
        self.source.get("sp_stock_lmp_rate")
    }
}

/// Opaque response for physical operation `get_elestock_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct InsiderShareholdingsJsonResponse {
    source: SourceValue,
}

impl InsiderShareholdingsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = InsiderShareholding<'_>> + '_ {
        items(&self.source, "list", false).map(InsiderShareholding::new)
    }
}

fn decode_insider_shareholdings_json_response(
    source: SourceValue,
) -> Result<InsiderShareholdingsJsonResponse, ResponseDecodeError> {
    decode_insider_shareholdings_json_response_root(&source, "$".to_owned())?;
    Ok(InsiderShareholdingsJsonResponse { source })
}

fn decode_insider_shareholdings_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional("list", decode_insider_shareholdings_json_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_insider_shareholdings_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_insider_shareholdings_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_insider_shareholdings_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_elestock_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct InsiderShareholdingsXmlResponse {
    source: SourceValue,
}

impl InsiderShareholdingsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = InsiderShareholding<'_>> + '_ {
        items(&self.source, "list", true).map(InsiderShareholding::new)
    }
}

fn decode_insider_shareholdings_xml_response(
    source: SourceValue,
) -> Result<InsiderShareholdingsXmlResponse, ResponseDecodeError> {
    decode_insider_shareholdings_xml_response_root(&source, "$".to_owned())?;
    Ok(InsiderShareholdingsXmlResponse { source })
}

fn decode_insider_shareholdings_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional("list", decode_insider_shareholdings_xml_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_insider_shareholdings_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_insider_shareholdings_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_insider_shareholdings_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
