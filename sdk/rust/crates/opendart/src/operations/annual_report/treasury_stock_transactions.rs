use super::*;

/// Inputs for logical operation `DS002-2019006`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct TreasuryStockTransactionsInput {
    company_code: String,
    business_year: String,
    report_code: String,
}

impl TreasuryStockTransactionsInput {
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

    /// Prepares physical operation `get_tesstkAcqsDspsSttus_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<TreasuryStockTransactionsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_tesstkAcqsDspsSttus_json", "DS002-2019006");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/tesstkAcqsDspsSttus.json", operation, &parameters),
            decode_treasury_stock_transactions_json_response,
        ))
    }

    /// Prepares physical operation `get_tesstkAcqsDspsSttus_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<TreasuryStockTransactionsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_tesstkAcqsDspsSttus_xml", "DS002-2019006");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/tesstkAcqsDspsSttus.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_treasury_stock_transactions_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct TreasuryStockTransaction<'a> {
    source: &'a SourceValue,
}

impl<'a> TreasuryStockTransaction<'a> {
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

    /// Returns source field `acqs_mth1` when present.
    #[must_use]
    pub fn acquisition_method_category(&self) -> Option<&SourceValue> {
        self.source.get("acqs_mth1")
    }

    /// Returns source field `acqs_mth2` when present.
    #[must_use]
    pub fn acquisition_method_subcategory(&self) -> Option<&SourceValue> {
        self.source.get("acqs_mth2")
    }

    /// Returns source field `acqs_mth3` when present.
    #[must_use]
    pub fn acquisition_method_detail(&self) -> Option<&SourceValue> {
        self.source.get("acqs_mth3")
    }

    /// Returns source field `bsis_qy` when present.
    #[must_use]
    pub fn opening_share_count(&self) -> Option<&SourceValue> {
        self.source.get("bsis_qy")
    }

    /// Returns source field `change_qy_acqs` when present.
    #[must_use]
    pub fn acquired_share_count(&self) -> Option<&SourceValue> {
        self.source.get("change_qy_acqs")
    }

    /// Returns source field `change_qy_dsps` when present.
    #[must_use]
    pub fn disposed_share_count(&self) -> Option<&SourceValue> {
        self.source.get("change_qy_dsps")
    }

    /// Returns source field `change_qy_incnr` when present.
    #[must_use]
    pub fn retired_share_count(&self) -> Option<&SourceValue> {
        self.source.get("change_qy_incnr")
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

    /// Returns source field `rm` when present.
    #[must_use]
    pub fn remarks(&self) -> Option<&SourceValue> {
        self.source.get("rm")
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

    /// Returns source field `trmend_qy` when present.
    #[must_use]
    pub fn closing_share_count(&self) -> Option<&SourceValue> {
        self.source.get("trmend_qy")
    }
}

/// Opaque response for physical operation `get_tesstkAcqsDspsSttus_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct TreasuryStockTransactionsJsonResponse {
    source: SourceValue,
}

impl TreasuryStockTransactionsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = TreasuryStockTransaction<'_>> + '_ {
        items(&self.source, "list", false).map(TreasuryStockTransaction::new)
    }
}

fn decode_treasury_stock_transactions_json_response(
    source: SourceValue,
) -> Result<TreasuryStockTransactionsJsonResponse, ResponseDecodeError> {
    decode_treasury_stock_transactions_json_response_root(&source, "$".to_owned())?;
    Ok(TreasuryStockTransactionsJsonResponse { source })
}

fn decode_treasury_stock_transactions_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_treasury_stock_transactions_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_treasury_stock_transactions_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_treasury_stock_transactions_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_treasury_stock_transactions_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_tesstkAcqsDspsSttus_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct TreasuryStockTransactionsXmlResponse {
    source: SourceValue,
}

impl TreasuryStockTransactionsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = TreasuryStockTransaction<'_>> + '_ {
        items(&self.source, "list", true).map(TreasuryStockTransaction::new)
    }
}

fn decode_treasury_stock_transactions_xml_response(
    source: SourceValue,
) -> Result<TreasuryStockTransactionsXmlResponse, ResponseDecodeError> {
    decode_treasury_stock_transactions_xml_response_root(&source, "$".to_owned())?;
    Ok(TreasuryStockTransactionsXmlResponse { source })
}

fn decode_treasury_stock_transactions_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_treasury_stock_transactions_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_treasury_stock_transactions_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_treasury_stock_transactions_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_treasury_stock_transactions_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
