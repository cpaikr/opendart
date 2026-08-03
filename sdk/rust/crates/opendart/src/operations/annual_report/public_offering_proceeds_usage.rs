use super::*;

/// Inputs for logical operation `DS002-2020016`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct PublicOfferingProceedsUsageInput {
    company_code: String,
    business_year: String,
    report_code: String,
}

impl PublicOfferingProceedsUsageInput {
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

    /// Prepares physical operation `get_pssrpCptalUseDtls_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<PublicOfferingProceedsUsageJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_pssrpCptalUseDtls_json", "DS002-2020016");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/pssrpCptalUseDtls.json", operation, &parameters),
            decode_public_offering_proceeds_usage_json_response,
        ))
    }

    /// Prepares physical operation `get_pssrpCptalUseDtls_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<PublicOfferingProceedsUsageXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_pssrpCptalUseDtls_xml", "DS002-2020016");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/pssrpCptalUseDtls.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_public_offering_proceeds_usage_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct PublicOfferingProceedsUsageEntry<'a> {
    source: &'a SourceValue,
}

impl<'a> PublicOfferingProceedsUsageEntry<'a> {
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

    /// Returns source field `dffrnc_occrrnc_resn` when present.
    #[must_use]
    pub fn variance_reason(&self) -> Option<&SourceValue> {
        self.source.get("dffrnc_occrrnc_resn")
    }

    /// Returns source field `on_dclrt_cptal_use_plan` when present.
    #[must_use]
    pub fn disclosed_proceeds_use_plan(&self) -> Option<&SourceValue> {
        self.source.get("on_dclrt_cptal_use_plan")
    }

    /// Returns source field `pay_amount` when present.
    #[must_use]
    pub fn payment_amount(&self) -> Option<&SourceValue> {
        self.source.get("pay_amount")
    }

    /// Returns source field `pay_de` when present.
    #[must_use]
    pub fn payment_date(&self) -> Option<&SourceValue> {
        self.source.get("pay_de")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `real_cptal_use_dtls_amount` when present.
    #[must_use]
    pub fn actual_use_amount(&self) -> Option<&SourceValue> {
        self.source.get("real_cptal_use_dtls_amount")
    }

    /// Returns source field `real_cptal_use_dtls_cn` when present.
    #[must_use]
    pub fn actual_use_description(&self) -> Option<&SourceValue> {
        self.source.get("real_cptal_use_dtls_cn")
    }

    /// Returns source field `real_cptal_use_sttus` when present.
    #[must_use]
    pub fn actual_proceeds_usage(&self) -> Option<&SourceValue> {
        self.source.get("real_cptal_use_sttus")
    }

    /// Returns source field `rs_cptal_use_plan_prcure_amount` when present.
    #[must_use]
    pub fn registration_statement_planned_amount(&self) -> Option<&SourceValue> {
        self.source.get("rs_cptal_use_plan_prcure_amount")
    }

    /// Returns source field `rs_cptal_use_plan_useprps` when present.
    #[must_use]
    pub fn registration_statement_planned_use(&self) -> Option<&SourceValue> {
        self.source.get("rs_cptal_use_plan_useprps")
    }

    /// Returns source field `se_nm` when present.
    #[must_use]
    pub fn category(&self) -> Option<&SourceValue> {
        self.source.get("se_nm")
    }

    /// Returns source field `stlm_dt` when present.
    #[must_use]
    pub fn fiscal_period_end_date(&self) -> Option<&SourceValue> {
        self.source.get("stlm_dt")
    }

    /// Returns source field `tm` when present.
    #[must_use]
    pub fn offering_round(&self) -> Option<&SourceValue> {
        self.source.get("tm")
    }
}

/// Opaque response for physical operation `get_pssrpCptalUseDtls_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct PublicOfferingProceedsUsageJsonResponse {
    source: SourceValue,
}

impl PublicOfferingProceedsUsageJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = PublicOfferingProceedsUsageEntry<'_>> + '_ {
        items(&self.source, "list", false).map(PublicOfferingProceedsUsageEntry::new)
    }
}

fn decode_public_offering_proceeds_usage_json_response(
    source: SourceValue,
) -> Result<PublicOfferingProceedsUsageJsonResponse, ResponseDecodeError> {
    decode_public_offering_proceeds_usage_json_response_root(&source, "$".to_owned())?;
    Ok(PublicOfferingProceedsUsageJsonResponse { source })
}

fn decode_public_offering_proceeds_usage_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_public_offering_proceeds_usage_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_public_offering_proceeds_usage_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_public_offering_proceeds_usage_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_public_offering_proceeds_usage_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_pssrpCptalUseDtls_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct PublicOfferingProceedsUsageXmlResponse {
    source: SourceValue,
}

impl PublicOfferingProceedsUsageXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = PublicOfferingProceedsUsageEntry<'_>> + '_ {
        items(&self.source, "list", true).map(PublicOfferingProceedsUsageEntry::new)
    }
}

fn decode_public_offering_proceeds_usage_xml_response(
    source: SourceValue,
) -> Result<PublicOfferingProceedsUsageXmlResponse, ResponseDecodeError> {
    decode_public_offering_proceeds_usage_xml_response_root(&source, "$".to_owned())?;
    Ok(PublicOfferingProceedsUsageXmlResponse { source })
}

fn decode_public_offering_proceeds_usage_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_public_offering_proceeds_usage_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_public_offering_proceeds_usage_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_public_offering_proceeds_usage_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_public_offering_proceeds_usage_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
