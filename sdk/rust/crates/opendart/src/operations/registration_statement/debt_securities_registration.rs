use super::*;

/// Inputs for logical operation `DS006-2020055`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct DebtSecuritiesRegistrationInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl DebtSecuritiesRegistrationInput {
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

    /// Prepares physical operation `get_bdRs_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<DebtSecuritiesRegistrationJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_bdRs_json", "DS006-2020055");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/bdRs.json", operation, &parameters),
            decode_debt_securities_registration_json_response,
        ))
    }

    /// Prepares physical operation `get_bdRs_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<DebtSecuritiesRegistrationXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_bdRs_xml", "DS006-2020055");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/bdRs.xml", operation, &parameters, "result"),
            decode_debt_securities_registration_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.group[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct DebtSecuritiesRegistrationSection<'a> {
    source: &'a SourceValue,
    xml: bool,
}

impl<'a> DebtSecuritiesRegistrationSection<'a> {
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
    pub fn items(&self) -> impl Iterator<Item = DebtSecuritiesRegistrationEntry<'a>> + '_ {
        items(self.source, "list", self.xml).map(DebtSecuritiesRegistrationEntry::new)
    }
}

/// Borrowed semantic view over `$.group[].list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct DebtSecuritiesRegistrationEntry<'a> {
    source: &'a SourceValue,
}

impl<'a> DebtSecuritiesRegistrationEntry<'a> {
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

    /// Returns source field `bdnmn` when present.
    #[must_use]
    pub fn debt_security_name(&self) -> Option<&SourceValue> {
        self.source.get("bdnmn")
    }

    /// Returns source field `bfsl_hdstk` when present.
    #[must_use]
    pub fn holdings_before_sale(&self) -> Option<&SourceValue> {
        self.source.get("bfsl_hdstk")
    }

    /// Returns source field `cdrt_int` when present.
    #[must_use]
    pub fn credit_rating_and_agency(&self) -> Option<&SourceValue> {
        self.source.get("cdrt_int")
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

    /// Returns source field `dpcr_amt` when present.
    #[must_use]
    pub fn denomination_issue_amount(&self) -> Option<&SourceValue> {
        self.source.get("dpcr_amt")
    }

    /// Returns source field `dpcrn` when present.
    #[must_use]
    pub fn denomination_currency(&self) -> Option<&SourceValue> {
        self.source.get("dpcrn")
    }

    /// Returns source field `drcb_at` when present.
    #[must_use]
    pub fn is_derivative_linked_bond(&self) -> Option<&SourceValue> {
        self.source.get("drcb_at")
    }

    /// Returns source field `drcb_mtd` when present.
    #[must_use]
    pub fn derivative_bond_maturity_date(&self) -> Option<&SourceValue> {
        self.source.get("drcb_mtd")
    }

    /// Returns source field `drcb_optknd` when present.
    #[must_use]
    pub fn option_type(&self) -> Option<&SourceValue> {
        self.source.get("drcb_optknd")
    }

    /// Returns source field `drcb_uast` when present.
    #[must_use]
    pub fn underlying_asset(&self) -> Option<&SourceValue> {
        self.source.get("drcb_uast")
    }

    /// Returns source field `estk_expd` when present.
    #[must_use]
    pub fn exercise_period(&self) -> Option<&SourceValue> {
        self.source.get("estk_expd")
    }

    /// Returns source field `estk_exprc` when present.
    #[must_use]
    pub fn exercise_price(&self) -> Option<&SourceValue> {
        self.source.get("estk_exprc")
    }

    /// Returns source field `estk_exrt` when present.
    #[must_use]
    pub fn exercise_ratio(&self) -> Option<&SourceValue> {
        self.source.get("estk_exrt")
    }

    /// Returns source field `estk_exstk` when present.
    #[must_use]
    pub fn linked_equity_security(&self) -> Option<&SourceValue> {
        self.source.get("estk_exstk")
    }

    /// Returns source field `fta` when present.
    #[must_use]
    pub fn aggregate_face_amount(&self) -> Option<&SourceValue> {
        self.source.get("fta")
    }

    /// Returns source field `grt_amt` when present.
    #[must_use]
    pub fn guaranteed_amount(&self) -> Option<&SourceValue> {
        self.source.get("grt_amt")
    }

    /// Returns source field `grt_int` when present.
    #[must_use]
    pub fn guarantor(&self) -> Option<&SourceValue> {
        self.source.get("grt_int")
    }

    /// Returns source field `hdr` when present.
    #[must_use]
    pub fn holder(&self) -> Option<&SourceValue> {
        self.source.get("hdr")
    }

    /// Returns source field `icmg_mgamt` when present.
    #[must_use]
    pub fn collateral_amount(&self) -> Option<&SourceValue> {
        self.source.get("icmg_mgamt")
    }

    /// Returns source field `icmg_mgknd` when present.
    #[must_use]
    pub fn collateral_type(&self) -> Option<&SourceValue> {
        self.source.get("icmg_mgknd")
    }

    /// Returns source field `intr` when present.
    #[must_use]
    pub fn interest_rate(&self) -> Option<&SourceValue> {
        self.source.get("intr")
    }

    /// Returns source field `isprc` when present.
    #[must_use]
    pub fn issue_price(&self) -> Option<&SourceValue> {
        self.source.get("isprc")
    }

    /// Returns source field `isrr` when present.
    #[must_use]
    pub fn issue_yield(&self) -> Option<&SourceValue> {
        self.source.get("isrr")
    }

    /// Returns source field `mngt_cmp` when present.
    #[must_use]
    pub fn bond_manager(&self) -> Option<&SourceValue> {
        self.source.get("mngt_cmp")
    }

    /// Returns source field `print_pymint` when present.
    #[must_use]
    pub fn principal_interest_payment_agent(&self) -> Option<&SourceValue> {
        self.source.get("print_pymint")
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

    /// Returns source field `rpd` when present.
    #[must_use]
    pub fn maturity_date(&self) -> Option<&SourceValue> {
        self.source.get("rpd")
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

    /// Returns source field `slmth` when present.
    #[must_use]
    pub fn offering_method(&self) -> Option<&SourceValue> {
        self.source.get("slmth")
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

    /// Returns source field `stksen` when present.
    #[must_use]
    pub fn security_type(&self) -> Option<&SourceValue> {
        self.source.get("stksen")
    }

    /// Returns source field `tm` when present.
    #[must_use]
    pub fn tranche(&self) -> Option<&SourceValue> {
        self.source.get("tm")
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

    /// Returns source field `udtintnm` when present.
    #[must_use]
    pub fn underwriting_institution_name(&self) -> Option<&SourceValue> {
        self.source.get("udtintnm")
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

    /// Returns source field `usarn` when present.
    #[must_use]
    pub fn use_region(&self) -> Option<&SourceValue> {
        self.source.get("usarn")
    }

    /// Returns source field `usntn` when present.
    #[must_use]
    pub fn use_country(&self) -> Option<&SourceValue> {
        self.source.get("usntn")
    }

    /// Returns source field `wnexpl_at` when present.
    #[must_use]
    pub fn won_exchange_planned(&self) -> Option<&SourceValue> {
        self.source.get("wnexpl_at")
    }
}

/// Opaque response for physical operation `get_bdRs_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct DebtSecuritiesRegistrationJsonResponse {
    source: SourceValue,
}

impl DebtSecuritiesRegistrationJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = DebtSecuritiesRegistrationSection<'_>> + '_ {
        items(&self.source, "group", false)
            .map(|source| DebtSecuritiesRegistrationSection::new(source, false))
    }
}

fn decode_debt_securities_registration_json_response(
    source: SourceValue,
) -> Result<DebtSecuritiesRegistrationJsonResponse, ResponseDecodeError> {
    decode_debt_securities_registration_json_response_root(&source, "$".to_owned())?;
    Ok(DebtSecuritiesRegistrationJsonResponse { source })
}

fn decode_debt_securities_registration_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "group",
        decode_debt_securities_registration_json_response_root_group,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_debt_securities_registration_json_response_root_group(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_debt_securities_registration_json_response_root_group_item,
    )?;
    Ok(())
}

fn decode_debt_securities_registration_json_response_root_group_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_debt_securities_registration_json_response_root_group_item_list,
    )?;
    Ok(())
}

fn decode_debt_securities_registration_json_response_root_group_item_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_debt_securities_registration_json_response_root_group_item_list_item,
    )?;
    Ok(())
}

fn decode_debt_securities_registration_json_response_root_group_item_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_bdRs_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct DebtSecuritiesRegistrationXmlResponse {
    source: SourceValue,
}

impl DebtSecuritiesRegistrationXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = DebtSecuritiesRegistrationSection<'_>> + '_ {
        items(&self.source, "group", true)
            .map(|source| DebtSecuritiesRegistrationSection::new(source, true))
    }
}

fn decode_debt_securities_registration_xml_response(
    source: SourceValue,
) -> Result<DebtSecuritiesRegistrationXmlResponse, ResponseDecodeError> {
    decode_debt_securities_registration_xml_response_root(&source, "$".to_owned())?;
    Ok(DebtSecuritiesRegistrationXmlResponse { source })
}

fn decode_debt_securities_registration_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "group",
        decode_debt_securities_registration_xml_response_root_group,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_debt_securities_registration_xml_response_root_group(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_debt_securities_registration_xml_response_root_group_item,
    )?;
    Ok(())
}

fn decode_debt_securities_registration_xml_response_root_group_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_debt_securities_registration_xml_response_root_group_item_list,
    )?;
    Ok(())
}

fn decode_debt_securities_registration_xml_response_root_group_item_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_debt_securities_registration_xml_response_root_group_item_list_item,
    )?;
    Ok(())
}

fn decode_debt_securities_registration_xml_response_root_group_item_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
