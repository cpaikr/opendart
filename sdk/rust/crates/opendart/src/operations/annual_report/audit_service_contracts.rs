use super::*;

/// Inputs for logical operation `DS002-2020010`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct AuditServiceContractsInput {
    company_code: String,
    business_year: String,
    report_code: String,
}

impl AuditServiceContractsInput {
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

    /// Prepares physical operation `get_adtServcCnclsSttus_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<AuditServiceContractsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_adtServcCnclsSttus_json", "DS002-2020010");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/adtServcCnclsSttus.json", operation, &parameters),
            decode_audit_service_contracts_json_response,
        ))
    }

    /// Prepares physical operation `get_adtServcCnclsSttus_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<AuditServiceContractsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_adtServcCnclsSttus_xml", "DS002-2020010");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/adtServcCnclsSttus.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_audit_service_contracts_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct AuditServiceContract<'a> {
    source: &'a SourceValue,
}

impl<'a> AuditServiceContract<'a> {
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

    /// Returns source field `adt_cntrct_dtls_mendng` when present.
    #[must_use]
    pub fn audit_contract_compensation(&self) -> Option<&SourceValue> {
        self.source.get("adt_cntrct_dtls_mendng")
    }

    /// Returns source field `adt_cntrct_dtls_time` when present.
    #[must_use]
    pub fn audit_contract_hours(&self) -> Option<&SourceValue> {
        self.source.get("adt_cntrct_dtls_time")
    }

    /// Returns source field `adtor` when present.
    #[must_use]
    pub fn accounting_auditor(&self) -> Option<&SourceValue> {
        self.source.get("adtor")
    }

    /// Returns source field `bsns_year` when present.
    #[must_use]
    pub fn business_year(&self) -> Option<&SourceValue> {
        self.source.get("bsns_year")
    }

    /// Returns source field `cn` when present.
    #[must_use]
    pub fn service_description(&self) -> Option<&SourceValue> {
        self.source.get("cn")
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

    /// Returns source field `mendng` when present.
    #[must_use]
    pub fn compensation(&self) -> Option<&SourceValue> {
        self.source.get("mendng")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `real_exc_dtls_mendng` when present.
    #[must_use]
    pub fn actual_compensation(&self) -> Option<&SourceValue> {
        self.source.get("real_exc_dtls_mendng")
    }

    /// Returns source field `real_exc_dtls_time` when present.
    #[must_use]
    pub fn actual_hours(&self) -> Option<&SourceValue> {
        self.source.get("real_exc_dtls_time")
    }

    /// Returns source field `stlm_dt` when present.
    #[must_use]
    pub fn fiscal_period_end_date(&self) -> Option<&SourceValue> {
        self.source.get("stlm_dt")
    }

    /// Returns source field `tot_reqre_time` when present.
    #[must_use]
    pub fn total_hours(&self) -> Option<&SourceValue> {
        self.source.get("tot_reqre_time")
    }
}

/// Opaque response for physical operation `get_adtServcCnclsSttus_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct AuditServiceContractsJsonResponse {
    source: SourceValue,
}

impl AuditServiceContractsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = AuditServiceContract<'_>> + '_ {
        items(&self.source, "list", false).map(AuditServiceContract::new)
    }
}

fn decode_audit_service_contracts_json_response(
    source: SourceValue,
) -> Result<AuditServiceContractsJsonResponse, ResponseDecodeError> {
    decode_audit_service_contracts_json_response_root(&source, "$".to_owned())?;
    Ok(AuditServiceContractsJsonResponse { source })
}

fn decode_audit_service_contracts_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_audit_service_contracts_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_audit_service_contracts_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_audit_service_contracts_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_audit_service_contracts_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_adtServcCnclsSttus_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct AuditServiceContractsXmlResponse {
    source: SourceValue,
}

impl AuditServiceContractsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = AuditServiceContract<'_>> + '_ {
        items(&self.source, "list", true).map(AuditServiceContract::new)
    }
}

fn decode_audit_service_contracts_xml_response(
    source: SourceValue,
) -> Result<AuditServiceContractsXmlResponse, ResponseDecodeError> {
    decode_audit_service_contracts_xml_response_root(&source, "$".to_owned())?;
    Ok(AuditServiceContractsXmlResponse { source })
}

fn decode_audit_service_contracts_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_audit_service_contracts_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_audit_service_contracts_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_audit_service_contracts_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_audit_service_contracts_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
