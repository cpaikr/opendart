use super::*;

/// Inputs for logical operation `DS005-2020019`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct DefaultEventsInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl DefaultEventsInput {
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

    /// Prepares physical operation `get_dfOcr_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(&self) -> Result<PreparedRequest<DefaultEventsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_dfOcr_json", "DS005-2020019");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/dfOcr.json", operation, &parameters),
            decode_default_events_json_response,
        ))
    }

    /// Prepares physical operation `get_dfOcr_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(&self) -> Result<PreparedRequest<DefaultEventsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_dfOcr_xml", "DS005-2020019");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/dfOcr.xml", operation, &parameters, "result"),
            decode_default_events_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct DefaultEvent<'a> {
    source: &'a SourceValue,
}

impl<'a> DefaultEvent<'a> {
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

    /// Returns source field `df_amt` when present.
    #[must_use]
    pub fn default_amount(&self) -> Option<&SourceValue> {
        self.source.get("df_amt")
    }

    /// Returns source field `df_bnk` when present.
    #[must_use]
    pub fn default_bank(&self) -> Option<&SourceValue> {
        self.source.get("df_bnk")
    }

    /// Returns source field `df_cn` when present.
    #[must_use]
    pub fn default_details(&self) -> Option<&SourceValue> {
        self.source.get("df_cn")
    }

    /// Returns source field `df_rs` when present.
    #[must_use]
    pub fn default_reason_and_circumstances(&self) -> Option<&SourceValue> {
        self.source.get("df_rs")
    }

    /// Returns source field `dfd` when present.
    #[must_use]
    pub fn final_default_date(&self) -> Option<&SourceValue> {
        self.source.get("dfd")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }
}

/// Opaque response for physical operation `get_dfOcr_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct DefaultEventsJsonResponse {
    source: SourceValue,
}

impl DefaultEventsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = DefaultEvent<'_>> + '_ {
        items(&self.source, "list", false).map(DefaultEvent::new)
    }
}

fn decode_default_events_json_response(
    source: SourceValue,
) -> Result<DefaultEventsJsonResponse, ResponseDecodeError> {
    decode_default_events_json_response_root(&source, "$".to_owned())?;
    Ok(DefaultEventsJsonResponse { source })
}

fn decode_default_events_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional("list", decode_default_events_json_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_default_events_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_default_events_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_default_events_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_dfOcr_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct DefaultEventsXmlResponse {
    source: SourceValue,
}

impl DefaultEventsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = DefaultEvent<'_>> + '_ {
        items(&self.source, "list", true).map(DefaultEvent::new)
    }
}

fn decode_default_events_xml_response(
    source: SourceValue,
) -> Result<DefaultEventsXmlResponse, ResponseDecodeError> {
    decode_default_events_xml_response_root(&source, "$".to_owned())?;
    Ok(DefaultEventsXmlResponse { source })
}

fn decode_default_events_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional("list", decode_default_events_xml_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_default_events_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_default_events_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_default_events_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
