use super::*;

/// Inputs for logical operation `DS005-2020020`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct BusinessSuspensionsInput {
    company_code: String,
    start_date: String,
    end_date: String,
}

impl BusinessSuspensionsInput {
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

    /// Prepares physical operation `get_bsnSp_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<BusinessSuspensionsJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_bsnSp_json", "DS005-2020020");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/bsnSp.json", operation, &parameters),
            decode_business_suspensions_json_response,
        ))
    }

    /// Prepares physical operation `get_bsnSp_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<BusinessSuspensionsXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_bsnSp_xml", "DS005-2020020");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/bsnSp.xml", operation, &parameters, "result"),
            decode_business_suspensions_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct BusinessSuspension<'a> {
    source: &'a SourceValue,
}

impl<'a> BusinessSuspension<'a> {
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

    /// Returns source field `adt_a_atn` when present.
    #[must_use]
    pub fn auditor_attendance_status(&self) -> Option<&SourceValue> {
        self.source.get("adt_a_atn")
    }

    /// Returns source field `bddd` when present.
    #[must_use]
    pub fn board_resolution_date(&self) -> Option<&SourceValue> {
        self.source.get("bddd")
    }

    /// Returns source field `bsnsp_af` when present.
    #[must_use]
    pub fn suspension_impact(&self) -> Option<&SourceValue> {
        self.source.get("bsnsp_af")
    }

    /// Returns source field `bsnsp_amt` when present.
    #[must_use]
    pub fn suspended_business_amount(&self) -> Option<&SourceValue> {
        self.source.get("bsnsp_amt")
    }

    /// Returns source field `bsnsp_cn` when present.
    #[must_use]
    pub fn suspension_details(&self) -> Option<&SourceValue> {
        self.source.get("bsnsp_cn")
    }

    /// Returns source field `bsnsp_rm` when present.
    #[must_use]
    pub fn suspended_business_area(&self) -> Option<&SourceValue> {
        self.source.get("bsnsp_rm")
    }

    /// Returns source field `bsnsp_rs` when present.
    #[must_use]
    pub fn suspension_reason(&self) -> Option<&SourceValue> {
        self.source.get("bsnsp_rs")
    }

    /// Returns source field `bsnspd` when present.
    #[must_use]
    pub fn suspension_date(&self) -> Option<&SourceValue> {
        self.source.get("bsnspd")
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

    /// Returns source field `ft_ctp` when present.
    #[must_use]
    pub fn future_measures(&self) -> Option<&SourceValue> {
        self.source.get("ft_ctp")
    }

    /// Returns source field `krx_stt_atn` when present.
    #[must_use]
    pub fn stock_exchange_mandatory_disclosure_status(&self) -> Option<&SourceValue> {
        self.source.get("krx_stt_atn")
    }

    /// Returns source field `ls_atn` when present.
    #[must_use]
    pub fn large_corporation_status(&self) -> Option<&SourceValue> {
        self.source.get("ls_atn")
    }

    /// Returns source field `od_a_at_b` when present.
    #[must_use]
    pub fn outside_directors_absent_count(&self) -> Option<&SourceValue> {
        self.source.get("od_a_at_b")
    }

    /// Returns source field `od_a_at_t` when present.
    #[must_use]
    pub fn outside_directors_present_count(&self) -> Option<&SourceValue> {
        self.source.get("od_a_at_t")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `rsl` when present.
    #[must_use]
    pub fn recent_total_sales(&self) -> Option<&SourceValue> {
        self.source.get("rsl")
    }

    /// Returns source field `sl_vs` when present.
    #[must_use]
    pub fn percentage_of_sales(&self) -> Option<&SourceValue> {
        self.source.get("sl_vs")
    }
}

/// Opaque response for physical operation `get_bsnSp_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct BusinessSuspensionsJsonResponse {
    source: SourceValue,
}

impl BusinessSuspensionsJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = BusinessSuspension<'_>> + '_ {
        items(&self.source, "list", false).map(BusinessSuspension::new)
    }
}

fn decode_business_suspensions_json_response(
    source: SourceValue,
) -> Result<BusinessSuspensionsJsonResponse, ResponseDecodeError> {
    decode_business_suspensions_json_response_root(&source, "$".to_owned())?;
    Ok(BusinessSuspensionsJsonResponse { source })
}

fn decode_business_suspensions_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional("list", decode_business_suspensions_json_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_business_suspensions_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_business_suspensions_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_business_suspensions_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_bsnSp_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct BusinessSuspensionsXmlResponse {
    source: SourceValue,
}

impl BusinessSuspensionsXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = BusinessSuspension<'_>> + '_ {
        items(&self.source, "list", true).map(BusinessSuspension::new)
    }
}

fn decode_business_suspensions_xml_response(
    source: SourceValue,
) -> Result<BusinessSuspensionsXmlResponse, ResponseDecodeError> {
    decode_business_suspensions_xml_response_root(&source, "$".to_owned())?;
    Ok(BusinessSuspensionsXmlResponse { source })
}

fn decode_business_suspensions_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional("list", decode_business_suspensions_xml_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_business_suspensions_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_business_suspensions_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_business_suspensions_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
