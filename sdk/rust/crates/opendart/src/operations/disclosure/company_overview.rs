use super::*;

/// Inputs for logical operation `DS001-2019002`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct CompanyOverviewInput {
    company_code: String,
}

impl CompanyOverviewInput {
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

    /// Prepares physical operation `get_company_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<CompanyOverviewJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_company_json", "DS001-2019002");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/company.json", operation, &parameters),
            decode_company_overview_json_response,
        ))
    }

    /// Prepares physical operation `get_company_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(&self) -> Result<PreparedRequest<CompanyOverviewXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_company_xml", "DS001-2019002");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/company.xml", operation, &parameters, "result"),
            decode_company_overview_xml_response,
        ))
    }
}

/// Opaque response for physical operation `get_company_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct CompanyOverviewJsonResponse {
    source: SourceValue,
}

impl CompanyOverviewJsonResponse {
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

    /// Returns source field `acc_mt` when present.
    #[must_use]
    pub fn fiscal_year_end_month(&self) -> Option<&SourceValue> {
        self.source.get("acc_mt")
    }

    /// Returns source field `adres` when present.
    #[must_use]
    pub fn address(&self) -> Option<&SourceValue> {
        self.source.get("adres")
    }

    /// Returns source field `bizr_no` when present.
    #[must_use]
    pub fn business_registration_number(&self) -> Option<&SourceValue> {
        self.source.get("bizr_no")
    }

    /// Returns source field `ceo_nm` when present.
    #[must_use]
    pub fn chief_executive_name(&self) -> Option<&SourceValue> {
        self.source.get("ceo_nm")
    }

    /// Returns source field `corp_cls` when present.
    #[must_use]
    pub fn company_class(&self) -> Option<&SourceValue> {
        self.source.get("corp_cls")
    }

    /// Returns source field `corp_name` when present.
    #[must_use]
    pub fn legal_name(&self) -> Option<&SourceValue> {
        self.source.get("corp_name")
    }

    /// Returns source field `corp_name_eng` when present.
    #[must_use]
    pub fn english_legal_name(&self) -> Option<&SourceValue> {
        self.source.get("corp_name_eng")
    }

    /// Returns source field `est_dt` when present.
    #[must_use]
    pub fn establishment_date(&self) -> Option<&SourceValue> {
        self.source.get("est_dt")
    }

    /// Returns source field `fax_no` when present.
    #[must_use]
    pub fn fax_number(&self) -> Option<&SourceValue> {
        self.source.get("fax_no")
    }

    /// Returns source field `hm_url` when present.
    #[must_use]
    pub fn website_url(&self) -> Option<&SourceValue> {
        self.source.get("hm_url")
    }

    /// Returns source field `induty_code` when present.
    #[must_use]
    pub fn industry_code(&self) -> Option<&SourceValue> {
        self.source.get("induty_code")
    }

    /// Returns source field `ir_url` when present.
    #[must_use]
    pub fn investor_relations_url(&self) -> Option<&SourceValue> {
        self.source.get("ir_url")
    }

    /// Returns source field `jurir_no` when present.
    #[must_use]
    pub fn corporate_registration_number(&self) -> Option<&SourceValue> {
        self.source.get("jurir_no")
    }

    /// Returns source field `phn_no` when present.
    #[must_use]
    pub fn phone_number(&self) -> Option<&SourceValue> {
        self.source.get("phn_no")
    }

    /// Returns source field `stock_code` when present.
    #[must_use]
    pub fn stock_code(&self) -> Option<&SourceValue> {
        self.source.get("stock_code")
    }

    /// Returns source field `stock_name` when present.
    #[must_use]
    pub fn display_name(&self) -> Option<&SourceValue> {
        self.source.get("stock_name")
    }

    /// Iterates response items; this object-shaped response defines none.
    pub fn items(&self) -> impl Iterator<Item = &SourceValue> {
        std::iter::empty()
    }
}

fn decode_company_overview_json_response(
    source: SourceValue,
) -> Result<CompanyOverviewJsonResponse, ResponseDecodeError> {
    decode_company_overview_json_response_root(&source, "$".to_owned())?;
    Ok(CompanyOverviewJsonResponse { source })
}

fn decode_company_overview_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

/// Opaque response for physical operation `get_company_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct CompanyOverviewXmlResponse {
    source: SourceValue,
}

impl CompanyOverviewXmlResponse {
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

    /// Returns source field `acc_mt` when present.
    #[must_use]
    pub fn fiscal_year_end_month(&self) -> Option<&SourceValue> {
        self.source.get("acc_mt")
    }

    /// Returns source field `adres` when present.
    #[must_use]
    pub fn address(&self) -> Option<&SourceValue> {
        self.source.get("adres")
    }

    /// Returns source field `bizr_no` when present.
    #[must_use]
    pub fn business_registration_number(&self) -> Option<&SourceValue> {
        self.source.get("bizr_no")
    }

    /// Returns source field `ceo_nm` when present.
    #[must_use]
    pub fn chief_executive_name(&self) -> Option<&SourceValue> {
        self.source.get("ceo_nm")
    }

    /// Returns source field `corp_cls` when present.
    #[must_use]
    pub fn company_class(&self) -> Option<&SourceValue> {
        self.source.get("corp_cls")
    }

    /// Returns source field `corp_name` when present.
    #[must_use]
    pub fn legal_name(&self) -> Option<&SourceValue> {
        self.source.get("corp_name")
    }

    /// Returns source field `corp_name_eng` when present.
    #[must_use]
    pub fn english_legal_name(&self) -> Option<&SourceValue> {
        self.source.get("corp_name_eng")
    }

    /// Returns source field `est_dt` when present.
    #[must_use]
    pub fn establishment_date(&self) -> Option<&SourceValue> {
        self.source.get("est_dt")
    }

    /// Returns source field `fax_no` when present.
    #[must_use]
    pub fn fax_number(&self) -> Option<&SourceValue> {
        self.source.get("fax_no")
    }

    /// Returns source field `hm_url` when present.
    #[must_use]
    pub fn website_url(&self) -> Option<&SourceValue> {
        self.source.get("hm_url")
    }

    /// Returns source field `induty_code` when present.
    #[must_use]
    pub fn industry_code(&self) -> Option<&SourceValue> {
        self.source.get("induty_code")
    }

    /// Returns source field `ir_url` when present.
    #[must_use]
    pub fn investor_relations_url(&self) -> Option<&SourceValue> {
        self.source.get("ir_url")
    }

    /// Returns source field `jurir_no` when present.
    #[must_use]
    pub fn corporate_registration_number(&self) -> Option<&SourceValue> {
        self.source.get("jurir_no")
    }

    /// Returns source field `phn_no` when present.
    #[must_use]
    pub fn phone_number(&self) -> Option<&SourceValue> {
        self.source.get("phn_no")
    }

    /// Returns source field `stock_code` when present.
    #[must_use]
    pub fn stock_code(&self) -> Option<&SourceValue> {
        self.source.get("stock_code")
    }

    /// Returns source field `stock_name` when present.
    #[must_use]
    pub fn display_name(&self) -> Option<&SourceValue> {
        self.source.get("stock_name")
    }

    /// Iterates response items; this object-shaped response defines none.
    pub fn items(&self) -> impl Iterator<Item = &SourceValue> {
        std::iter::empty()
    }
}

fn decode_company_overview_xml_response(
    source: SourceValue,
) -> Result<CompanyOverviewXmlResponse, ResponseDecodeError> {
    decode_company_overview_xml_response_root(&source, "$".to_owned())?;
    Ok(CompanyOverviewXmlResponse { source })
}

fn decode_company_overview_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}
