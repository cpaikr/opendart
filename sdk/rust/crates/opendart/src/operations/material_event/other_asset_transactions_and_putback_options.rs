use super::*;

/// Inputs for logical operation `DS005-2020018`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct OtherAssetTransactionsAndPutbackOptionsInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl OtherAssetTransactionsAndPutbackOptionsInput {
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

    /// Prepares physical operation `get_astInhtrfEtcPtbkOpt_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<OtherAssetTransactionsAndPutbackOptionsJsonResponse>, PrepareError>
    {
        let operation = OperationIdentity::new("get_astInhtrfEtcPtbkOpt_json", "DS005-2020018");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/astInhtrfEtcPtbkOpt.json", operation, &parameters),
            decode_other_asset_transactions_and_putback_options_json_response,
        ))
    }

    /// Prepares physical operation `get_astInhtrfEtcPtbkOpt_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<OtherAssetTransactionsAndPutbackOptionsXmlResponse>, PrepareError>
    {
        let operation = OperationIdentity::new("get_astInhtrfEtcPtbkOpt_xml", "DS005-2020018");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/astInhtrfEtcPtbkOpt.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_other_asset_transactions_and_putback_options_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct OtherAssetTransactionAndPutbackOption<'a> {
    source: &'a SourceValue,
}

impl<'a> OtherAssetTransactionAndPutbackOption<'a> {
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

    /// Returns source field `ast_inhtrf_prc` when present.
    #[must_use]
    pub fn asset_transfer_price(&self) -> Option<&SourceValue> {
        self.source.get("ast_inhtrf_prc")
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

    /// Returns source field `rp_rsn` when present.
    #[must_use]
    pub fn report_reason(&self) -> Option<&SourceValue> {
        self.source.get("rp_rsn")
    }
}

/// Opaque response for physical operation `get_astInhtrfEtcPtbkOpt_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct OtherAssetTransactionsAndPutbackOptionsJsonResponse {
    source: SourceValue,
}

impl OtherAssetTransactionsAndPutbackOptionsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = OtherAssetTransactionAndPutbackOption<'_>> + '_ {
        items(&self.source, "list", false).map(OtherAssetTransactionAndPutbackOption::new)
    }
}

fn decode_other_asset_transactions_and_putback_options_json_response(
    source: SourceValue,
) -> Result<OtherAssetTransactionsAndPutbackOptionsJsonResponse, ResponseDecodeError> {
    decode_other_asset_transactions_and_putback_options_json_response_root(
        &source,
        "$".to_owned(),
    )?;
    Ok(OtherAssetTransactionsAndPutbackOptionsJsonResponse { source })
}

fn decode_other_asset_transactions_and_putback_options_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_other_asset_transactions_and_putback_options_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_other_asset_transactions_and_putback_options_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_other_asset_transactions_and_putback_options_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_other_asset_transactions_and_putback_options_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_astInhtrfEtcPtbkOpt_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct OtherAssetTransactionsAndPutbackOptionsXmlResponse {
    source: SourceValue,
}

impl OtherAssetTransactionsAndPutbackOptionsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = OtherAssetTransactionAndPutbackOption<'_>> + '_ {
        items(&self.source, "list", true).map(OtherAssetTransactionAndPutbackOption::new)
    }
}

fn decode_other_asset_transactions_and_putback_options_xml_response(
    source: SourceValue,
) -> Result<OtherAssetTransactionsAndPutbackOptionsXmlResponse, ResponseDecodeError> {
    decode_other_asset_transactions_and_putback_options_xml_response_root(&source, "$".to_owned())?;
    Ok(OtherAssetTransactionsAndPutbackOptionsXmlResponse { source })
}

fn decode_other_asset_transactions_and_putback_options_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_other_asset_transactions_and_putback_options_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_other_asset_transactions_and_putback_options_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_other_asset_transactions_and_putback_options_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_other_asset_transactions_and_putback_options_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
