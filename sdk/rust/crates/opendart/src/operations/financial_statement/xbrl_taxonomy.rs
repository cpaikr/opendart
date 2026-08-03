use super::*;

/// Inputs for logical operation `DS003-2020001`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct XbrlTaxonomyInput {
    statement_type: String,
}

impl XbrlTaxonomyInput {
    /// Creates inputs from all required source-shaped values.
    #[must_use]
    pub fn new(statement_type: String) -> Self {
        Self { statement_type }
    }

    fn parameters(
        &self,
        operation: OperationIdentity,
    ) -> Result<Vec<crate::request::QueryParameter<'_>>, PrepareError> {
        let mut query = Query::new(operation);
        query.required("sj_div", &self.statement_type)?;
        Ok(query.finish())
    }

    /// Prepares physical operation `get_xbrlTaxonomy_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(&self) -> Result<PreparedRequest<XbrlTaxonomyJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_xbrlTaxonomy_json", "DS003-2020001");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/xbrlTaxonomy.json", operation, &parameters),
            decode_xbrl_taxonomy_json_response,
        ))
    }

    /// Prepares physical operation `get_xbrlTaxonomy_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(&self) -> Result<PreparedRequest<XbrlTaxonomyXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_xbrlTaxonomy_xml", "DS003-2020001");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/xbrlTaxonomy.xml", operation, &parameters, "result"),
            decode_xbrl_taxonomy_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct XbrlTaxonomyAccount<'a> {
    source: &'a SourceValue,
}

impl<'a> XbrlTaxonomyAccount<'a> {
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

    /// Returns source field `account_id` when present.
    #[must_use]
    pub fn account_id(&self) -> Option<&SourceValue> {
        self.source.get("account_id")
    }

    /// Returns source field `account_nm` when present.
    #[must_use]
    pub fn account_name(&self) -> Option<&SourceValue> {
        self.source.get("account_nm")
    }

    /// Returns source field `bsns_de` when present.
    #[must_use]
    pub fn reference_date(&self) -> Option<&SourceValue> {
        self.source.get("bsns_de")
    }

    /// Returns source field `data_tp` when present.
    #[must_use]
    pub fn data_type(&self) -> Option<&SourceValue> {
        self.source.get("data_tp")
    }

    /// Returns source field `ifrs_ref` when present.
    #[must_use]
    pub fn ifrs_reference(&self) -> Option<&SourceValue> {
        self.source.get("ifrs_ref")
    }

    /// Returns source field `label_eng` when present.
    #[must_use]
    pub fn english_label(&self) -> Option<&SourceValue> {
        self.source.get("label_eng")
    }

    /// Returns source field `label_kor` when present.
    #[must_use]
    pub fn korean_label(&self) -> Option<&SourceValue> {
        self.source.get("label_kor")
    }

    /// Returns source field `sj_div` when present.
    #[must_use]
    pub fn statement_type(&self) -> Option<&SourceValue> {
        self.source.get("sj_div")
    }
}

/// Opaque response for physical operation `get_xbrlTaxonomy_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct XbrlTaxonomyJsonResponse {
    source: SourceValue,
}

impl XbrlTaxonomyJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = XbrlTaxonomyAccount<'_>> + '_ {
        items(&self.source, "list", false).map(XbrlTaxonomyAccount::new)
    }
}

fn decode_xbrl_taxonomy_json_response(
    source: SourceValue,
) -> Result<XbrlTaxonomyJsonResponse, ResponseDecodeError> {
    decode_xbrl_taxonomy_json_response_root(&source, "$".to_owned())?;
    Ok(XbrlTaxonomyJsonResponse { source })
}

fn decode_xbrl_taxonomy_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional("list", decode_xbrl_taxonomy_json_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_xbrl_taxonomy_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_xbrl_taxonomy_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_xbrl_taxonomy_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_xbrlTaxonomy_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct XbrlTaxonomyXmlResponse {
    source: SourceValue,
}

impl XbrlTaxonomyXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = XbrlTaxonomyAccount<'_>> + '_ {
        items(&self.source, "list", true).map(XbrlTaxonomyAccount::new)
    }
}

fn decode_xbrl_taxonomy_xml_response(
    source: SourceValue,
) -> Result<XbrlTaxonomyXmlResponse, ResponseDecodeError> {
    decode_xbrl_taxonomy_xml_response_root(&source, "$".to_owned())?;
    Ok(XbrlTaxonomyXmlResponse { source })
}

fn decode_xbrl_taxonomy_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional("list", decode_xbrl_taxonomy_xml_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_xbrl_taxonomy_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_xbrl_taxonomy_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_xbrl_taxonomy_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
