use super::*;

/// Inputs for logical operation `DS002-2020015`.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct DirectorAuditorCompensationByTypeInput {
    company_code: String,
    business_year: String,
    report_code: String,
}

impl DirectorAuditorCompensationByTypeInput {
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

    /// Prepares physical operation `get_drctrAdtAllMendngSttusMendngPymntamtTyCl_json`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_json(
        &self,
    ) -> Result<PreparedRequest<DirectorAuditorCompensationByTypeJsonResponse>, PrepareError> {
        let operation = OperationIdentity::new(
            "get_drctrAdtAllMendngSttusMendngPymntamtTyCl_json",
            "DS002-2020015",
        );
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_json(
                "/api/drctrAdtAllMendngSttusMendngPymntamtTyCl.json",
                operation,
                &parameters,
            ),
            decode_director_auditor_compensation_by_type_json_response,
        ))
    }

    /// Prepares physical operation `get_drctrAdtAllMendngSttusMendngPymntamtTyCl_xml`.
    ///
    /// # Errors
    /// Returns a stable [`PrepareError`] when supplied input violates canonical OpenAPI constraints.
    pub fn prepare_xml(
        &self,
    ) -> Result<PreparedRequest<DirectorAuditorCompensationByTypeXmlResponse>, PrepareError> {
        let operation = OperationIdentity::new(
            "get_drctrAdtAllMendngSttusMendngPymntamtTyCl_xml",
            "DS002-2020015",
        );
        let parameters = self.parameters(operation)?;
        Ok(PreparedRequest::new(
            RequestParts::structured_xml(
                "/api/drctrAdtAllMendngSttusMendngPymntamtTyCl.xml",
                operation,
                &parameters,
                "result",
            ),
            decode_director_auditor_compensation_by_type_xml_response,
        ))
    }
}

/// Borrowed semantic view over `$.list[]`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct DirectorAuditorCompensationByTypeEntry<'a> {
    source: &'a SourceValue,
}

impl<'a> DirectorAuditorCompensationByTypeEntry<'a> {
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

    /// Returns source field `fscl_year` when present.
    #[must_use]
    pub fn fiscal_year(&self) -> Option<&SourceValue> {
        self.source.get("fscl_year")
    }

    /// Returns source field `nmpr` when present.
    #[must_use]
    pub fn person_count(&self) -> Option<&SourceValue> {
        self.source.get("nmpr")
    }

    /// Returns source field `othr_stk_bsd_cmpn_mkt_vl` when present.
    #[must_use]
    pub fn other_unpaid_stock_compensation_market_value(&self) -> Option<&SourceValue> {
        self.source.get("othr_stk_bsd_cmpn_mkt_vl")
    }

    /// Returns source field `othr_stk_bsd_cmpn_unpyd_qty` when present.
    #[must_use]
    pub fn other_unpaid_stock_compensation_quantity(&self) -> Option<&SourceValue> {
        self.source.get("othr_stk_bsd_cmpn_unpyd_qty")
    }

    /// Returns source field `psn1_avrg_pymntamt` when present.
    #[must_use]
    pub fn average_compensation_per_person(&self) -> Option<&SourceValue> {
        self.source.get("psn1_avrg_pymntamt")
    }

    /// Returns source field `pymnt_totamt` when present.
    #[must_use]
    pub fn total_compensation(&self) -> Option<&SourceValue> {
        self.source.get("pymnt_totamt")
    }

    /// Returns source field `rcept_no` when present.
    #[must_use]
    pub fn receipt_number(&self) -> Option<&SourceValue> {
        self.source.get("rcept_no")
    }

    /// Returns source field `rm` when present.
    #[must_use]
    pub fn remarks(&self) -> Option<&SourceValue> {
        self.source.get("rm")
    }

    /// Returns source field `se` when present.
    #[must_use]
    pub fn category(&self) -> Option<&SourceValue> {
        self.source.get("se")
    }

    /// Returns source field `stk_bsd_pd_mendng_totamt` when present.
    #[must_use]
    pub fn stock_based_compensation_paid(&self) -> Option<&SourceValue> {
        self.source.get("stk_bsd_pd_mendng_totamt")
    }

    /// Returns source field `stk_opt_exrcsbl_qty` when present.
    #[must_use]
    pub fn exercisable_stock_option_quantity(&self) -> Option<&SourceValue> {
        self.source.get("stk_opt_exrcsbl_qty")
    }

    /// Returns source field `stk_opt_rmn_blce` when present.
    #[must_use]
    pub fn remaining_stock_option_value(&self) -> Option<&SourceValue> {
        self.source.get("stk_opt_rmn_blce")
    }

    /// Returns source field `stk_opt_unexrcsbl_qty` when present.
    #[must_use]
    pub fn unexercisable_stock_option_quantity(&self) -> Option<&SourceValue> {
        self.source.get("stk_opt_unexrcsbl_qty")
    }

    /// Returns source field `stlm_dt` when present.
    #[must_use]
    pub fn fiscal_period_end_date(&self) -> Option<&SourceValue> {
        self.source.get("stlm_dt")
    }
}

/// Opaque response for physical operation `get_drctrAdtAllMendngSttusMendngPymntamtTyCl_json`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct DirectorAuditorCompensationByTypeJsonResponse {
    source: SourceValue,
}

impl DirectorAuditorCompensationByTypeJsonResponse {
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
    pub fn items(&self) -> impl Iterator<Item = DirectorAuditorCompensationByTypeEntry<'_>> + '_ {
        items(&self.source, "list", false).map(DirectorAuditorCompensationByTypeEntry::new)
    }
}

fn decode_director_auditor_compensation_by_type_json_response(
    source: SourceValue,
) -> Result<DirectorAuditorCompensationByTypeJsonResponse, ResponseDecodeError> {
    decode_director_auditor_compensation_by_type_json_response_root(&source, "$".to_owned())?;
    Ok(DirectorAuditorCompensationByTypeJsonResponse { source })
}

fn decode_director_auditor_compensation_by_type_json_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new(value, path)?;
    object.optional(
        "list",
        decode_director_auditor_compensation_by_type_json_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_director_auditor_compensation_by_type_json_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_array(
        value,
        path,
        decode_director_auditor_compensation_by_type_json_response_root_list_item,
    )?;
    Ok(())
}

fn decode_director_auditor_compensation_by_type_json_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new(value, path)?;
    Ok(())
}

/// Opaque response for physical operation `get_drctrAdtAllMendngSttusMendngPymntamtTyCl_xml`.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]
#[cfg_attr(feature = "serde-json", serde(transparent))]
pub struct DirectorAuditorCompensationByTypeXmlResponse {
    source: SourceValue,
}

impl DirectorAuditorCompensationByTypeXmlResponse {
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
    pub fn items(&self) -> impl Iterator<Item = DirectorAuditorCompensationByTypeEntry<'_>> + '_ {
        items(&self.source, "list", true).map(DirectorAuditorCompensationByTypeEntry::new)
    }
}

fn decode_director_auditor_compensation_by_type_xml_response(
    source: SourceValue,
) -> Result<DirectorAuditorCompensationByTypeXmlResponse, ResponseDecodeError> {
    decode_director_auditor_compensation_by_type_xml_response_root(&source, "$".to_owned())?;
    Ok(DirectorAuditorCompensationByTypeXmlResponse { source })
}

fn decode_director_auditor_compensation_by_type_xml_response_root(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    let object = ObjectDecoder::new_xml(value, path)?;
    object.optional(
        "list",
        decode_director_auditor_compensation_by_type_xml_response_root_list,
    )?;
    object.optional("status", decode_source_status)?;
    Ok(())
}

fn decode_director_auditor_compensation_by_type_xml_response_root_list(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_xml_array(
        value,
        path,
        decode_director_auditor_compensation_by_type_xml_response_root_list_item,
    )?;
    Ok(())
}

fn decode_director_auditor_compensation_by_type_xml_response_root_list_item(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    ObjectDecoder::new_xml(value, path)?;
    Ok(())
}
