use super::*;

/// Inputs for logical operation `DS002-2020004`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct CommercialPaperOutstandingBalanceInput {
    company_code: String,
    business_year: String,
    report_code: String,
}

impl CommercialPaperOutstandingBalanceInput {
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

    /// Prepares physical operation `get_entrprsBilScritsNrdmpBlce_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<CommercialPaperOutstandingBalanceJsonResponse>, PrepareError> {
        let operation =
            OperationIdentity::new("get_entrprsBilScritsNrdmpBlce_json", "DS002-2020004");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json(
                "/api/entrprsBilScritsNrdmpBlce.json",
                operation,
                &parameters,
            ),
            decode_commercial_paper_outstanding_balance_json_response,
        ))
    }

    /// Prepares physical operation `get_entrprsBilScritsNrdmpBlce_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<CommercialPaperOutstandingBalanceXmlResponse>, PrepareError> {
        let operation =
            OperationIdentity::new("get_entrprsBilScritsNrdmpBlce_xml", "DS002-2020004");
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/entrprsBilScritsNrdmpBlce.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_commercial_paper_outstanding_balance_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct CommercialPaperMaturityBalance<'a> {
    source: &'a SourceValue,
}

impl<'a> CommercialPaperMaturityBalance<'a> {
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

    /// Returns source field `de10_below` when present.
    #[must_use]
    pub fn up_to_ten_days(&self) -> Option<&SourceValue> {
        self.source.get("de10_below")
    }

    /// Returns source field `de10_excess_de30_below` when present.
    #[must_use]
    pub fn over_ten_up_to_thirty_days(&self) -> Option<&SourceValue> {
        self.source.get("de10_excess_de30_below")
    }

    /// Returns source field `de180_excess_yy1_below` when present.
    #[must_use]
    pub fn over_one_hundred_eighty_days_up_to_one_year(&self) -> Option<&SourceValue> {
        self.source.get("de180_excess_yy1_below")
    }

    /// Returns source field `de30_excess_de90_below` when present.
    #[must_use]
    pub fn over_thirty_up_to_ninety_days(&self) -> Option<&SourceValue> {
        self.source.get("de30_excess_de90_below")
    }

    /// Returns source field `de90_excess_de180_below` when present.
    #[must_use]
    pub fn over_ninety_up_to_one_hundred_eighty_days(&self) -> Option<&SourceValue> {
        self.source.get("de90_excess_de180_below")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `remndr_exprtn1` when present.
    #[must_use]
    pub fn remaining_maturity_category(&self) -> Option<&SourceValue> {
        self.source.get("remndr_exprtn1")
    }

    /// Returns source field `remndr_exprtn2` when present.
    #[must_use]
    pub fn remaining_maturity_detail(&self) -> Option<&SourceValue> {
        self.source.get("remndr_exprtn2")
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

    /// Returns source field `yy1_excess_yy2_below` when present.
    #[must_use]
    pub fn over_one_up_to_two_years(&self) -> Option<&SourceValue> {
        self.source.get("yy1_excess_yy2_below")
    }

    /// Returns source field `yy2_excess_yy3_below` when present.
    #[must_use]
    pub fn over_two_up_to_three_years(&self) -> Option<&SourceValue> {
        self.source.get("yy2_excess_yy3_below")
    }

    /// Returns source field `yy3_excess` when present.
    #[must_use]
    pub fn over_three_years(&self) -> Option<&SourceValue> {
        self.source.get("yy3_excess")
    }
}

/// Opaque response for physical operation `get_entrprsBilScritsNrdmpBlce_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct CommercialPaperOutstandingBalanceJsonResponse {
    source: SourceValue,
}

impl CommercialPaperOutstandingBalanceJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = CommercialPaperMaturityBalance<'_>> + '_ {
        items(&self.source, "list", false).map(CommercialPaperMaturityBalance::new)
    }
}

fn decode_commercial_paper_outstanding_balance_json_response(
    source: SourceValue,
) -> Result<CommercialPaperOutstandingBalanceJsonResponse, ResponseDecodeError> {
    decode_commercial_paper_outstanding_balance_json_response_root(&source, "$".to_owned())?;
    Ok(CommercialPaperOutstandingBalanceJsonResponse { source })
}

fn decode_commercial_paper_outstanding_balance_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_commercial_paper_outstanding_balance_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_commercial_paper_outstanding_balance_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_commercial_paper_outstanding_balance_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_commercial_paper_outstanding_balance_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_entrprsBilScritsNrdmpBlce_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct CommercialPaperOutstandingBalanceXmlResponse {
    source: SourceValue,
}

impl CommercialPaperOutstandingBalanceXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = CommercialPaperMaturityBalance<'_>> + '_ {
        items(&self.source, "list", true).map(CommercialPaperMaturityBalance::new)
    }
}

fn decode_commercial_paper_outstanding_balance_xml_response(
    source: SourceValue,
) -> Result<CommercialPaperOutstandingBalanceXmlResponse, ResponseDecodeError> {
    decode_commercial_paper_outstanding_balance_xml_response_root(&source, "$".to_owned())?;
    Ok(CommercialPaperOutstandingBalanceXmlResponse { source })
}

fn decode_commercial_paper_outstanding_balance_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_commercial_paper_outstanding_balance_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_commercial_paper_outstanding_balance_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_commercial_paper_outstanding_balance_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_commercial_paper_outstanding_balance_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
