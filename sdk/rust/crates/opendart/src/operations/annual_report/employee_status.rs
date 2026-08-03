use super::*;

/// Inputs for logical operation `DS002-2019011`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct EmployeeStatusInput {
    company_code: String,
    business_year: String,
    report_code: String,
}

impl EmployeeStatusInput {
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

    /// Prepares physical operation `get_empSttus_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<EmployeeStatusJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_empSttus_json", "DS002-2019011");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json("/api/empSttus.json", operation, &parameters),
            decode_employee_status_json_response,
        ))
    }

    /// Prepares physical operation `get_empSttus_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(&self) -> Result<PreparedRequest<EmployeeStatusXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new("get_empSttus_xml", "DS002-2019011");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml("/api/empSttus.xml", operation, &parameters, "result"),
            decode_employee_status_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct EmployeeGroup<'a> {
    source: &'a SourceValue,
}

impl<'a> EmployeeGroup<'a> {
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

    /// Returns source field `avrg_cnwk_sdytrn` when present.
    #[must_use]
    pub fn average_years_of_service(&self) -> Option<&SourceValue> {
        self.source.get("avrg_cnwk_sdytrn")
    }

    /// Returns source field `cnttk_abacpt_labrr_co` when present.
    #[must_use]
    pub fn part_time_contract_employee_count(&self) -> Option<&SourceValue> {
        self.source.get("cnttk_abacpt_labrr_co")
    }

    /// Returns source field `cnttk_co` when present.
    #[must_use]
    pub fn contract_employee_count(&self) -> Option<&SourceValue> {
        self.source.get("cnttk_co")
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

    /// Returns source field `fo_bbm` when present.
    #[must_use]
    pub fn business_segment(&self) -> Option<&SourceValue> {
        self.source.get("fo_bbm")
    }

    /// Returns source field `fyer_salary_totamt` when present.
    #[must_use]
    pub fn annual_salary_total(&self) -> Option<&SourceValue> {
        self.source.get("fyer_salary_totamt")
    }

    /// Returns source field `jan_salary_am` when present.
    #[must_use]
    pub fn average_salary_per_employee(&self) -> Option<&SourceValue> {
        self.source.get("jan_salary_am")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `reform_bfe_emp_co_cnttk` when present.
    #[must_use]
    pub fn pre_revision_contract_employee_count(&self) -> Option<&SourceValue> {
        self.source.get("reform_bfe_emp_co_cnttk")
    }

    /// Returns source field `reform_bfe_emp_co_etc` when present.
    #[must_use]
    pub fn pre_revision_other_employee_count(&self) -> Option<&SourceValue> {
        self.source.get("reform_bfe_emp_co_etc")
    }

    /// Returns source field `reform_bfe_emp_co_rgllbr` when present.
    #[must_use]
    pub fn pre_revision_regular_employee_count(&self) -> Option<&SourceValue> {
        self.source.get("reform_bfe_emp_co_rgllbr")
    }

    /// Returns source field `rgllbr_abacpt_labrr_co` when present.
    #[must_use]
    pub fn part_time_regular_employee_count(&self) -> Option<&SourceValue> {
        self.source.get("rgllbr_abacpt_labrr_co")
    }

    /// Returns source field `rgllbr_co` when present.
    #[must_use]
    pub fn regular_employee_count(&self) -> Option<&SourceValue> {
        self.source.get("rgllbr_co")
    }

    /// Returns source field `rm` when present.
    #[must_use]
    pub fn remarks(&self) -> Option<&SourceValue> {
        self.source.get("rm")
    }

    /// Returns source field `sexdstn` when present.
    #[must_use]
    pub fn gender(&self) -> Option<&SourceValue> {
        self.source.get("sexdstn")
    }

    /// Returns source field `sm` when present.
    #[must_use]
    pub fn total(&self) -> Option<&SourceValue> {
        self.source.get("sm")
    }

    /// Returns source field `stlm_dt` when present.
    #[must_use]
    pub fn fiscal_period_end_date(&self) -> Option<&SourceValue> {
        self.source.get("stlm_dt")
    }
}

/// Opaque response for physical operation `get_empSttus_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct EmployeeStatusJsonResponse {
    source: SourceValue,
}

impl EmployeeStatusJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = EmployeeGroup<'_>> + '_ {
        items(&self.source, "list", false).map(EmployeeGroup::new)
    }
}

fn decode_employee_status_json_response(
    source: SourceValue,
) -> Result<EmployeeStatusJsonResponse, ResponseDecodeError> {
    decode_employee_status_json_response_root(&source, "$".to_owned())?;
    Ok(EmployeeStatusJsonResponse { source })
}

fn decode_employee_status_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional("list", decode_employee_status_json_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_employee_status_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_employee_status_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_employee_status_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_empSttus_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct EmployeeStatusXmlResponse {
    source: SourceValue,
}

impl EmployeeStatusXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = EmployeeGroup<'_>> + '_ {
        items(&self.source, "list", true).map(EmployeeGroup::new)
    }
}

fn decode_employee_status_xml_response(
    source: SourceValue,
) -> Result<EmployeeStatusXmlResponse, ResponseDecodeError> {
    decode_employee_status_xml_response_root(&source, "$".to_owned())?;
    Ok(EmployeeStatusXmlResponse { source })
}

fn decode_employee_status_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional("list", decode_employee_status_xml_response_root_list)?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_employee_status_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_employee_status_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_employee_status_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
