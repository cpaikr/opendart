use super::*;

/// Inputs for logical operation `DS006-2020057`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct MergerRegistrationInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl MergerRegistrationInput {
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

    /// Prepares physical operation `get_mgRs_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<MergerRegistrationJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_mgRs_json", "DS006-2020057");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/mgRs.json", operation, &parameters),
            decode_merger_registration_json_response,
        ))
    }

    /// Prepares physical operation `get_mgRs_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<MergerRegistrationXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_mgRs_xml", "DS006-2020057");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/mgRs.xml", operation, &parameters, "result"),
            decode_merger_registration_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.group[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct MergerRegistrationSection<'a> {
    source: &'a SourceValue,
    xml: bool,
}

impl<'a> MergerRegistrationSection<'a> {
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
    pub fn items(&self) -> impl Iterator<Item = MergerRegistrationEntry<'a>> + '_ {
        items(self.source, "list", self.xml).map(MergerRegistrationEntry::new)
    }
}

/// Borrowed semantic view over `$.group[].list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct MergerRegistrationEntry<'a> {
    source: &'a SourceValue,
}

impl<'a> MergerRegistrationEntry<'a> {
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

    /// Returns source field `ap_gmtsck` when present.
    #[must_use]
    pub fn approval_shareholder_meeting_date(&self) -> Option<&SourceValue> {
        self.source.get("ap_gmtsck")
    }

    /// Returns source field `aprskh_pd_bgd` when present.
    #[must_use]
    pub fn appraisal_rights_start_date(&self) -> Option<&SourceValue> {
        self.source.get("aprskh_pd_bgd")
    }

    /// Returns source field `aprskh_pd_edd` when present.
    #[must_use]
    pub fn appraisal_rights_end_date(&self) -> Option<&SourceValue> {
        self.source.get("aprskh_pd_edd")
    }

    /// Returns source field `aprskh_prc` when present.
    #[must_use]
    pub fn appraisal_rights_price(&self) -> Option<&SourceValue> {
        self.source.get("aprskh_prc")
    }

    /// Returns source field `bddd` when present.
    #[must_use]
    pub fn board_resolution_date(&self) -> Option<&SourceValue> {
        self.source.get("bddd")
    }

    /// Returns source field `cmpnm` when present.
    #[must_use]
    pub fn counterparty_company_name(&self) -> Option<&SourceValue> {
        self.source.get("cmpnm")
    }

    /// Returns source field `cnt` when present.
    #[must_use]
    pub fn quantity(&self) -> Option<&SourceValue> {
        self.source.get("cnt")
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

    /// Returns source field `cpt` when present.
    #[must_use]
    pub fn paid_in_capital(&self) -> Option<&SourceValue> {
        self.source.get("cpt")
    }

    /// Returns source field `ctrd` when present.
    #[must_use]
    pub fn contract_date(&self) -> Option<&SourceValue> {
        self.source.get("ctrd")
    }

    /// Returns source field `exevl_int` when present.
    #[must_use]
    pub fn external_valuation_firm(&self) -> Option<&SourceValue> {
        self.source.get("exevl_int")
    }

    /// Returns source field `fv` when present.
    #[must_use]
    pub fn face_value(&self) -> Option<&SourceValue> {
        self.source.get("fv")
    }

    /// Returns source field `gmtsck_shddstd` when present.
    #[must_use]
    pub fn shareholder_record_date(&self) -> Option<&SourceValue> {
        self.source.get("gmtsck_shddstd")
    }

    /// Returns source field `grtmn_etc` when present.
    #[must_use]
    pub fn cash_or_other_consideration(&self) -> Option<&SourceValue> {
        self.source.get("grtmn_etc")
    }

    /// Returns source field `isstk_cnt` when present.
    #[must_use]
    pub fn issued_share_count(&self) -> Option<&SourceValue> {
        self.source.get("isstk_cnt")
    }

    /// Returns source field `isstk_knd` when present.
    #[must_use]
    pub fn issued_share_type(&self) -> Option<&SourceValue> {
        self.source.get("isstk_knd")
    }

    /// Returns source field `kndn` when present.
    #[must_use]
    pub fn security_type(&self) -> Option<&SourceValue> {
        self.source.get("kndn")
    }

    /// Returns source field `mgdt_etc` when present.
    #[must_use]
    pub fn transaction_effective_date(&self) -> Option<&SourceValue> {
        self.source.get("mgdt_etc")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `rpt_rcpn` when present.
    #[must_use]
    pub fn material_event_receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rpt_rcpn")
    }

    /// Returns source field `rt_vl` when present.
    #[must_use]
    pub fn ratio_or_value(&self) -> Option<&SourceValue> {
        self.source.get("rt_vl")
    }

    /// Returns source field `sen` when present.
    #[must_use]
    pub fn party_type(&self) -> Option<&SourceValue> {
        self.source.get("sen")
    }

    /// Returns source field `slprc` when present.
    #[must_use]
    pub fn offering_price(&self) -> Option<&SourceValue> {
        self.source.get("slprc")
    }

    /// Returns source field `slta` when present.
    #[must_use]
    pub fn offering_total(&self) -> Option<&SourceValue> {
        self.source.get("slta")
    }

    /// Returns source field `stn` when present.
    #[must_use]
    pub fn transaction_form(&self) -> Option<&SourceValue> {
        self.source.get("stn")
    }

    /// Returns source field `tast` when present.
    #[must_use]
    pub fn total_assets(&self) -> Option<&SourceValue> {
        self.source.get("tast")
    }
}

/// Opaque response for physical operation `get_mgRs_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct MergerRegistrationJsonResponse {
    source: SourceValue,
}

impl MergerRegistrationJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = MergerRegistrationSection<'_>> + '_ {
        items(&self.source, "group", false)
            .map(|source| MergerRegistrationSection::new(source, false))
    }
}

fn decode_merger_registration_json_response(
    source: SourceValue,
) -> Result<MergerRegistrationJsonResponse, ResponseDecodeError> {
    decode_merger_registration_json_response_root(&source, "$".to_owned())?;
    Ok(MergerRegistrationJsonResponse { source })
}

fn decode_merger_registration_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional("group", decode_merger_registration_json_response_root_group)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_merger_registration_json_response_root_group(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_merger_registration_json_response_root_group_item,
    )?;
    Ok(())
}

fn decode_merger_registration_json_response_root_group_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_merger_registration_json_response_root_group_item_list,
    )?;
    Ok(())
}

fn decode_merger_registration_json_response_root_group_item_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_merger_registration_json_response_root_group_item_list_item,
    )?;
    Ok(())
}

fn decode_merger_registration_json_response_root_group_item_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_mgRs_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct MergerRegistrationXmlResponse {
    source: SourceValue,
}

impl MergerRegistrationXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = MergerRegistrationSection<'_>> + '_ {
        items(&self.source, "group", true)
            .map(|source| MergerRegistrationSection::new(source, true))
    }
}

fn decode_merger_registration_xml_response(
    source: SourceValue,
) -> Result<MergerRegistrationXmlResponse, ResponseDecodeError> {
    decode_merger_registration_xml_response_root(&source, "$".to_owned())?;
    Ok(MergerRegistrationXmlResponse { source })
}

fn decode_merger_registration_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional("group", decode_merger_registration_xml_response_root_group)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_merger_registration_xml_response_root_group(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_merger_registration_xml_response_root_group_item,
    )?;
    Ok(())
}

fn decode_merger_registration_xml_response_root_group_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_merger_registration_xml_response_root_group_item_list,
    )?;
    Ok(())
}

fn decode_merger_registration_xml_response_root_group_item_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_merger_registration_xml_response_root_group_item_list_item,
    )?;
    Ok(())
}

fn decode_merger_registration_xml_response_root_group_item_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
