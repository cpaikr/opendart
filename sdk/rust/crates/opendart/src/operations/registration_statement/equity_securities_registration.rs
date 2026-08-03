use super::*;

/// Inputs for logical operation `DS006-2020054`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct EquitySecuritiesRegistrationInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl EquitySecuritiesRegistrationInput {
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

    /// Prepares physical operation `get_estkRs_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<EquitySecuritiesRegistrationJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_estkRs_json", "DS006-2020054");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/estkRs.json", operation, &parameters),
            decode_equity_securities_registration_json_response,
        ))
    }

    /// Prepares physical operation `get_estkRs_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<EquitySecuritiesRegistrationXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_estkRs_xml", "DS006-2020054");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/estkRs.xml", operation, &parameters, "result"),
            decode_equity_securities_registration_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.group[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct EquitySecuritiesRegistrationSection<'a> {
    source: &'a SourceValue,
    xml: bool,
}

impl<'a> EquitySecuritiesRegistrationSection<'a> {
    fn new(source: &'a SourceValue, xml: bool) -> Self {
        Self { source, xml }
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

    /// Returns source field `title` when present.
    #[must_use]
    pub fn title(&self) -> Option<&SourceValue> {
        self.source.get("title")
    }

    /// Iterates the reviewed `$.group[].list[]` child views.
    pub fn items(&self) -> impl Iterator<Item = EquitySecuritiesRegistrationEntry<'a>> + '_ {
        items(self.source, "list", self.xml).map(EquitySecuritiesRegistrationEntry::new)
    }
}

/// Borrowed semantic view over `$.group[].list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct EquitySecuritiesRegistrationEntry<'a> {
    source: &'a SourceValue,
}

impl<'a> EquitySecuritiesRegistrationEntry<'a> {
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

    /// Returns source field `actnmn` when present.
    #[must_use]
    pub fn underwriter_name(&self) -> Option<&SourceValue> {
        self.source.get("actnmn")
    }

    /// Returns source field `actsen` when present.
    #[must_use]
    pub fn underwriter_type(&self) -> Option<&SourceValue> {
        self.source.get("actsen")
    }

    /// Returns source field `amt` when present.
    #[must_use]
    pub fn amount(&self) -> Option<&SourceValue> {
        self.source.get("amt")
    }

    /// Returns source field `asand` when present.
    #[must_use]
    pub fn allocation_notice_date(&self) -> Option<&SourceValue> {
        self.source.get("asand")
    }

    /// Returns source field `asstd` when present.
    #[must_use]
    pub fn allocation_record_date(&self) -> Option<&SourceValue> {
        self.source.get("asstd")
    }

    /// Returns source field `atsl_hdstk` when present.
    #[must_use]
    pub fn holdings_after_sale(&self) -> Option<&SourceValue> {
        self.source.get("atsl_hdstk")
    }

    /// Returns source field `bfsl_hdstk` when present.
    #[must_use]
    pub fn holdings_before_sale(&self) -> Option<&SourceValue> {
        self.source.get("bfsl_hdstk")
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

    /// Returns source field `exavivr` when present.
    #[must_use]
    pub fn eligible_exercise_investors(&self) -> Option<&SourceValue> {
        self.source.get("exavivr")
    }

    /// Returns source field `expd` when present.
    #[must_use]
    pub fn warrant_exercise_period(&self) -> Option<&SourceValue> {
        self.source.get("expd")
    }

    /// Returns source field `exprc` when present.
    #[must_use]
    pub fn warrant_exercise_price(&self) -> Option<&SourceValue> {
        self.source.get("exprc")
    }

    /// Returns source field `exstk` when present.
    #[must_use]
    pub fn warrant_underlying_security(&self) -> Option<&SourceValue> {
        self.source.get("exstk")
    }

    /// Returns source field `fv` when present.
    #[must_use]
    pub fn face_value(&self) -> Option<&SourceValue> {
        self.source.get("fv")
    }

    /// Returns source field `grtcnt` when present.
    #[must_use]
    pub fn granted_quantity(&self) -> Option<&SourceValue> {
        self.source.get("grtcnt")
    }

    /// Returns source field `grtrs` when present.
    #[must_use]
    pub fn grant_reason(&self) -> Option<&SourceValue> {
        self.source.get("grtrs")
    }

    /// Returns source field `hdr` when present.
    #[must_use]
    pub fn holder(&self) -> Option<&SourceValue> {
        self.source.get("hdr")
    }

    /// Returns source field `pymd` when present.
    #[must_use]
    pub fn payment_date(&self) -> Option<&SourceValue> {
        self.source.get("pymd")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `rl_cmp` when present.
    #[must_use]
    pub fn relationship_to_company(&self) -> Option<&SourceValue> {
        self.source.get("rl_cmp")
    }

    /// Returns source field `rpt_rcpn` when present.
    #[must_use]
    pub fn material_event_receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rpt_rcpn")
    }

    /// Returns source field `sband` when present.
    #[must_use]
    pub fn subscription_notice_date(&self) -> Option<&SourceValue> {
        self.source.get("sband")
    }

    /// Returns source field `sbd` when present.
    #[must_use]
    pub fn subscription_date(&self) -> Option<&SourceValue> {
        self.source.get("sbd")
    }

    /// Returns source field `se` when present.
    #[must_use]
    pub fn category(&self) -> Option<&SourceValue> {
        self.source.get("se")
    }

    /// Returns source field `slmthn` when present.
    #[must_use]
    pub fn offering_method(&self) -> Option<&SourceValue> {
        self.source.get("slmthn")
    }

    /// Returns source field `slprc` when present.
    #[must_use]
    pub fn offering_price(&self) -> Option<&SourceValue> {
        self.source.get("slprc")
    }

    /// Returns source field `slstk` when present.
    #[must_use]
    pub fn securities_sold(&self) -> Option<&SourceValue> {
        self.source.get("slstk")
    }

    /// Returns source field `slta` when present.
    #[must_use]
    pub fn offering_total(&self) -> Option<&SourceValue> {
        self.source.get("slta")
    }

    /// Returns source field `stkcnt` when present.
    #[must_use]
    pub fn security_quantity(&self) -> Option<&SourceValue> {
        self.source.get("stkcnt")
    }

    /// Returns source field `stksen` when present.
    #[must_use]
    pub fn security_type(&self) -> Option<&SourceValue> {
        self.source.get("stksen")
    }

    /// Returns source field `udtamt` when present.
    #[must_use]
    pub fn underwriting_amount(&self) -> Option<&SourceValue> {
        self.source.get("udtamt")
    }

    /// Returns source field `udtcnt` when present.
    #[must_use]
    pub fn underwriting_quantity(&self) -> Option<&SourceValue> {
        self.source.get("udtcnt")
    }

    /// Returns source field `udtmth` when present.
    #[must_use]
    pub fn underwriting_method(&self) -> Option<&SourceValue> {
        self.source.get("udtmth")
    }

    /// Returns source field `udtprc` when present.
    #[must_use]
    pub fn underwriting_consideration(&self) -> Option<&SourceValue> {
        self.source.get("udtprc")
    }
}

/// Opaque response for physical operation `get_estkRs_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct EquitySecuritiesRegistrationJsonResponse {
    source: SourceValue,
}

impl EquitySecuritiesRegistrationJsonResponse {
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

    /// Iterates the reviewed `$.group[]` item views.
    pub fn items(&self) -> impl Iterator<Item = EquitySecuritiesRegistrationSection<'_>> + '_ {
        items(&self.source, "group", false)
            .map(|source| EquitySecuritiesRegistrationSection::new(source, false))
    }
}

fn decode_equity_securities_registration_json_response(
    source: SourceValue,
) -> Result<EquitySecuritiesRegistrationJsonResponse, ResponseDecodeError> {
    decode_equity_securities_registration_json_response_root(&source, "$".to_owned())?;
    Ok(EquitySecuritiesRegistrationJsonResponse { source })
}

fn decode_equity_securities_registration_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "group",
        decode_equity_securities_registration_json_response_root_group,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_equity_securities_registration_json_response_root_group(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_equity_securities_registration_json_response_root_group_item,
    )?;
    Ok(())
}

fn decode_equity_securities_registration_json_response_root_group_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_equity_securities_registration_json_response_root_group_item_list,
    )?;
    Ok(())
}

fn decode_equity_securities_registration_json_response_root_group_item_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_equity_securities_registration_json_response_root_group_item_list_item,
    )?;
    Ok(())
}

fn decode_equity_securities_registration_json_response_root_group_item_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_estkRs_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct EquitySecuritiesRegistrationXmlResponse {
    source: SourceValue,
}

impl EquitySecuritiesRegistrationXmlResponse {
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

    /// Iterates the reviewed `$.group[]` item views.
    pub fn items(&self) -> impl Iterator<Item = EquitySecuritiesRegistrationSection<'_>> + '_ {
        items(&self.source, "group", true)
            .map(|source| EquitySecuritiesRegistrationSection::new(source, true))
    }
}

fn decode_equity_securities_registration_xml_response(
    source: SourceValue,
) -> Result<EquitySecuritiesRegistrationXmlResponse, ResponseDecodeError> {
    decode_equity_securities_registration_xml_response_root(&source, "$".to_owned())?;
    Ok(EquitySecuritiesRegistrationXmlResponse { source })
}

fn decode_equity_securities_registration_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "group",
        decode_equity_securities_registration_xml_response_root_group,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_equity_securities_registration_xml_response_root_group(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_equity_securities_registration_xml_response_root_group_item,
    )?;
    Ok(())
}

fn decode_equity_securities_registration_xml_response_root_group_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_equity_securities_registration_xml_response_root_group_item_list,
    )?;
    Ok(())
}

fn decode_equity_securities_registration_xml_response_root_group_item_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_equity_securities_registration_xml_response_root_group_item_list_item,
    )?;
    Ok(())
}

fn decode_equity_securities_registration_xml_response_root_group_item_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
