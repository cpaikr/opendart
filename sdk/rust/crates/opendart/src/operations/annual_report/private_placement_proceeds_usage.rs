use super::*;

/// Inputs for logical operation `DS002-2020017`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct PrivatePlacementProceedsUsageInput {
    company_code: String,
    business_year: String,
    report_code: String,
}

impl PrivatePlacementProceedsUsageInput {
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

    /// Prepares physical operation `get_prvsrpCptalUseDtls_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<PrivatePlacementProceedsUsageJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_prvsrpCptalUseDtls_json", "DS002-2020017");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/prvsrpCptalUseDtls.json", operation, &parameters),
            decode_private_placement_proceeds_usage_json_response,
        ))
    }

    /// Prepares physical operation `get_prvsrpCptalUseDtls_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<PrivatePlacementProceedsUsageXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_prvsrpCptalUseDtls_xml", "DS002-2020017");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/prvsrpCptalUseDtls.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_private_placement_proceeds_usage_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct PrivatePlacementProceedsUsageEntry<'a> {
    source: &'a SourceValue,
}

impl<'a> PrivatePlacementProceedsUsageEntry<'a> {
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

    /// Returns source field `cptal_use_plan` when present.
    #[must_use]
    pub fn proceeds_use_plan(&self) -> Option<&SourceValue> {
        self.source.get("cptal_use_plan")
    }

    /// Returns source field `dffrnc_occrrnc_resn` when present.
    #[must_use]
    pub fn variance_reason(&self) -> Option<&SourceValue> {
        self.source.get("dffrnc_occrrnc_resn")
    }

    /// Returns source field `mtrpt_cptal_use_plan_prcure_amount` when present.
    #[must_use]
    pub fn material_event_report_planned_amount(&self) -> Option<&SourceValue> {
        self.source.get("mtrpt_cptal_use_plan_prcure_amount")
    }

    /// Returns source field `mtrpt_cptal_use_plan_useprps` when present.
    #[must_use]
    pub fn material_event_report_planned_use(&self) -> Option<&SourceValue> {
        self.source.get("mtrpt_cptal_use_plan_useprps")
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

/// Opaque response for physical operation `get_prvsrpCptalUseDtls_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct PrivatePlacementProceedsUsageJsonResponse {
    source: SourceValue,
}

impl PrivatePlacementProceedsUsageJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = PrivatePlacementProceedsUsageEntry<'_>> + '_ {
        items(&self.source, "list", false).map(PrivatePlacementProceedsUsageEntry::new)
    }
}

fn decode_private_placement_proceeds_usage_json_response(
    source: SourceValue,
) -> Result<PrivatePlacementProceedsUsageJsonResponse, ResponseDecodeError> {
    decode_private_placement_proceeds_usage_json_response_root(&source, "$".to_owned())?;
    Ok(PrivatePlacementProceedsUsageJsonResponse { source })
}

fn decode_private_placement_proceeds_usage_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_private_placement_proceeds_usage_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_private_placement_proceeds_usage_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_private_placement_proceeds_usage_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_private_placement_proceeds_usage_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_prvsrpCptalUseDtls_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct PrivatePlacementProceedsUsageXmlResponse {
    source: SourceValue,
}

impl PrivatePlacementProceedsUsageXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = PrivatePlacementProceedsUsageEntry<'_>> + '_ {
        items(&self.source, "list", true).map(PrivatePlacementProceedsUsageEntry::new)
    }
}

fn decode_private_placement_proceeds_usage_xml_response(
    source: SourceValue,
) -> Result<PrivatePlacementProceedsUsageXmlResponse, ResponseDecodeError> {
    decode_private_placement_proceeds_usage_xml_response_root(&source, "$".to_owned())?;
    Ok(PrivatePlacementProceedsUsageXmlResponse { source })
}

fn decode_private_placement_proceeds_usage_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_private_placement_proceeds_usage_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_private_placement_proceeds_usage_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_private_placement_proceeds_usage_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_private_placement_proceeds_usage_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
