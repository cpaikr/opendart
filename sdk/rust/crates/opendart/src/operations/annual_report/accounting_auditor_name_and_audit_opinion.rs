use super::*;

/// Inputs for logical operation `DS002-2020009`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct AccountingAuditorNameAndAuditOpinionInput {
    company_code: String,
    business_year: String,
    report_code: String,
}

impl AccountingAuditorNameAndAuditOpinionInput {
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

    /// Prepares physical operation `get_accnutAdtorNmNdAdtOpinion_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<AccountingAuditorNameAndAuditOpinionJsonResponse>, PrepareError>
    {
        let operation =
            OperationIdentity::new("get_accnutAdtorNmNdAdtOpinion_json", "DS002-2020009");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json(
                "/api/accnutAdtorNmNdAdtOpinion.json",
                operation,
                &parameters,
            ),
            decode_accounting_auditor_name_and_audit_opinion_json_response,
        ))
    }

    /// Prepares physical operation `get_accnutAdtorNmNdAdtOpinion_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<AccountingAuditorNameAndAuditOpinionXmlResponse>, PrepareError>
    {
        let operation =
            OperationIdentity::new("get_accnutAdtorNmNdAdtOpinion_xml", "DS002-2020009");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/accnutAdtorNmNdAdtOpinion.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_accounting_auditor_name_and_audit_opinion_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct AccountingAudit<'a> {
    source: &'a SourceValue,
}

impl<'a> AccountingAudit<'a> {
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

    /// Returns source field `adt_opinion` when present.
    #[must_use]
    pub fn audit_opinion(&self) -> Option<&SourceValue> {
        self.source.get("adt_opinion")
    }

    /// Returns source field `adt_reprt_spcmnt_matter` when present.
    #[must_use]
    pub fn audit_report_special_notes(&self) -> Option<&SourceValue> {
        self.source.get("adt_reprt_spcmnt_matter")
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

    /// Returns source field `core_adt_matter` when present.
    #[must_use]
    pub fn key_audit_matters(&self) -> Option<&SourceValue> {
        self.source.get("core_adt_matter")
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

    /// Returns source field `emphs_matter` when present.
    #[must_use]
    pub fn emphasis_of_matter(&self) -> Option<&SourceValue> {
        self.source.get("emphs_matter")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `stlm_dt` when present.
    #[must_use]
    pub fn fiscal_period_end_date(&self) -> Option<&SourceValue> {
        self.source.get("stlm_dt")
    }
}

/// Opaque response for physical operation `get_accnutAdtorNmNdAdtOpinion_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct AccountingAuditorNameAndAuditOpinionJsonResponse {
    source: SourceValue,
}

impl AccountingAuditorNameAndAuditOpinionJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = AccountingAudit<'_>> + '_ {
        items(&self.source, "list", false).map(AccountingAudit::new)
    }
}

fn decode_accounting_auditor_name_and_audit_opinion_json_response(
    source: SourceValue,
) -> Result<AccountingAuditorNameAndAuditOpinionJsonResponse, ResponseDecodeError> {
    decode_accounting_auditor_name_and_audit_opinion_json_response_root(&source, "$".to_owned())?;
    Ok(AccountingAuditorNameAndAuditOpinionJsonResponse { source })
}

fn decode_accounting_auditor_name_and_audit_opinion_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_accounting_auditor_name_and_audit_opinion_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_accounting_auditor_name_and_audit_opinion_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_accounting_auditor_name_and_audit_opinion_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_accounting_auditor_name_and_audit_opinion_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_accnutAdtorNmNdAdtOpinion_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct AccountingAuditorNameAndAuditOpinionXmlResponse {
    source: SourceValue,
}

impl AccountingAuditorNameAndAuditOpinionXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = AccountingAudit<'_>> + '_ {
        items(&self.source, "list", true).map(AccountingAudit::new)
    }
}

fn decode_accounting_auditor_name_and_audit_opinion_xml_response(
    source: SourceValue,
) -> Result<AccountingAuditorNameAndAuditOpinionXmlResponse, ResponseDecodeError> {
    decode_accounting_auditor_name_and_audit_opinion_xml_response_root(&source, "$".to_owned())?;
    Ok(AccountingAuditorNameAndAuditOpinionXmlResponse { source })
}

fn decode_accounting_auditor_name_and_audit_opinion_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_accounting_auditor_name_and_audit_opinion_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_accounting_auditor_name_and_audit_opinion_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_accounting_auditor_name_and_audit_opinion_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_accounting_auditor_name_and_audit_opinion_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
