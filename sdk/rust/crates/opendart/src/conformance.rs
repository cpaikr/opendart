//! Private handwritten pilot and mutation controls for the Rust-native gate.

use crate::{
    Authentication, OperationIdentity, PrepareError, PreparedBinaryRequest, PreparedRequest,
    Representation, ResponseDecodeError, SourceStatus, SourceValue, SourceValueKind,
    request::{QueryParameter, QueryValue, RequestParts},
    validation::{require_allowed, require_format, require_length},
};

const JSON: &[Representation] = &[Representation::Json];
const XML: &[Representation] = &[Representation::Xml];
const ZIP_OR_XML: &[Representation] = &[Representation::Zip, Representation::Xml];
const HANDWRITTEN_CONTRACT: &str = "handwritten-conformance-v1";

struct CompanyOverviewInput {
    company_code: String,
}

impl CompanyOverviewInput {
    fn new(company_code: impl Into<String>) -> Self {
        Self {
            company_code: company_code.into(),
        }
    }

    fn prepare_json(&self) -> Result<PreparedRequest<CompanyOverviewJsonResponse>, PrepareError> {
        let parts = self.prepare_parts(
            OperationIdentity::new("get_company_json", "DS001-2019002"),
            "/api/company.json",
            JSON,
            None,
        )?;
        Ok(PreparedRequest::new(parts, decode_company_overview_json))
    }

    fn prepare_xml(&self) -> Result<PreparedRequest<CompanyOverviewXmlResponse>, PrepareError> {
        let parts = self.prepare_parts(
            OperationIdentity::new("get_company_xml", "DS001-2019002"),
            "/api/company.xml",
            XML,
            Some("result"),
        )?;
        Ok(PreparedRequest::new(parts, decode_company_overview_xml))
    }

    fn prepare_parts(
        &self,
        identity: OperationIdentity,
        path: &'static str,
        representation: &'static [Representation],
        xml_root: Option<&'static str>,
    ) -> Result<RequestParts, PrepareError> {
        require_nonempty(identity, "corp_code", &self.company_code)?;
        require_length(identity, "corp_code", &self.company_code, 8, 8)?;
        require_format(
            identity,
            "corp_code",
            &self.company_code,
            "opendart-corp-code",
        )?;
        let parameters = [QueryParameter {
            name: "corp_code",
            value: QueryValue::Scalar(&self.company_code),
        }];
        Ok(RequestParts::new(
            path,
            identity,
            &parameters,
            representation,
            xml_root,
            0,
            HANDWRITTEN_CONTRACT,
        ))
    }
}

struct AnnualReportRequest {
    company_code: String,
    business_year: String,
    report_code: String,
}

#[derive(Clone, Copy)]
enum ValidationFault {
    None,
    IgnoreRequiredness,
    IgnoreAllowedValues,
}

impl AnnualReportRequest {
    fn new(
        company_code: impl Into<String>,
        business_year: impl Into<String>,
        report_code: impl Into<String>,
    ) -> Self {
        Self {
            company_code: company_code.into(),
            business_year: business_year.into(),
            report_code: report_code.into(),
        }
    }

    fn prepare_json(&self) -> Result<PreparedRequest<CompanyOverviewJsonResponse>, PrepareError> {
        self.prepare_json_with_fault(ValidationFault::None)
    }

    fn prepare_json_with_fault(
        &self,
        fault: ValidationFault,
    ) -> Result<PreparedRequest<CompanyOverviewJsonResponse>, PrepareError> {
        let identity =
            OperationIdentity::new("get_accnutAdtorNmNdAdtOpinion_json", "DS002-2020009");
        if !matches!(fault, ValidationFault::IgnoreRequiredness) {
            require_nonempty(identity, "corp_code", &self.company_code)?;
            require_nonempty(identity, "bsns_year", &self.business_year)?;
            require_nonempty(identity, "reprt_code", &self.report_code)?;
        }
        if !self.company_code.is_empty() {
            require_length(identity, "corp_code", &self.company_code, 8, 8)?;
            require_format(
                identity,
                "corp_code",
                &self.company_code,
                "opendart-corp-code",
            )?;
        }
        if !self.business_year.is_empty() {
            require_length(identity, "bsns_year", &self.business_year, 4, 4)?;
            require_format(identity, "bsns_year", &self.business_year, "opendart-year")?;
        }
        if !self.report_code.is_empty() && !matches!(fault, ValidationFault::IgnoreAllowedValues) {
            require_allowed(
                identity,
                "reprt_code",
                &self.report_code,
                &["11011", "11012", "11013", "11014"],
            )?;
        }
        let parameters = [
            QueryParameter {
                name: "corp_code",
                value: QueryValue::Scalar(&self.company_code),
            },
            QueryParameter {
                name: "bsns_year",
                value: QueryValue::Scalar(&self.business_year),
            },
            QueryParameter {
                name: "reprt_code",
                value: QueryValue::Scalar(&self.report_code),
            },
        ];
        Ok(PreparedRequest::new(
            RequestParts::new(
                "/api/accnutAdtorNmNdAdtOpinion.json",
                identity,
                &parameters,
                JSON,
                None,
                0,
                HANDWRITTEN_CONTRACT,
            ),
            decode_company_overview_json,
        ))
    }
}

struct CompanyCodesInput;

impl CompanyCodesInput {
    fn prepare_archive(&self) -> PreparedBinaryRequest {
        PreparedBinaryRequest::new(RequestParts::new(
            "/api/corpCode.xml",
            OperationIdentity::new("get_corpCode_xml", "DS001-2019018"),
            &[],
            ZIP_OR_XML,
            Some("result"),
            0,
            HANDWRITTEN_CONTRACT,
        ))
    }
}

macro_rules! company_overview_response {
    ($name:ident) => {
        #[derive(Clone, Debug, PartialEq)]
        struct $name {
            source: SourceValue,
        }

        impl $name {
            fn source(&self) -> &SourceValue {
                &self.source
            }

            fn status(&self) -> Option<SourceStatus> {
                self.source
                    .get("status")
                    .and_then(SourceValue::as_str)
                    .map(SourceStatus::new)
            }

            fn message(&self) -> Option<&SourceValue> {
                self.source.get("message")
            }

            fn legal_name(&self) -> Option<&str> {
                self.source.get("corp_name").and_then(SourceValue::as_str)
            }

            fn field(&self, name: &str) -> Option<&SourceValue> {
                self.source.get(name)
            }
        }

        #[cfg(feature = "serde-json")]
        impl serde::Serialize for $name {
            fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
            where
                S: serde::Serializer,
            {
                self.source.serialize(serializer)
            }
        }
    };
}

company_overview_response!(CompanyOverviewJsonResponse);
company_overview_response!(CompanyOverviewXmlResponse);

fn validate_company_overview(value: &SourceValue) -> Result<(), ResponseDecodeError> {
    if value.kind() != SourceValueKind::Object {
        return Err(ResponseDecodeError::WrongKind {
            path: "$".to_owned(),
            expected: SourceValueKind::Object,
            actual: value.kind(),
        });
    }
    let company_name =
        value
            .get("corp_name")
            .ok_or_else(|| ResponseDecodeError::MissingRequired {
                path: "$/corp_name".to_owned(),
            })?;
    if company_name.kind() != SourceValueKind::String {
        return Err(ResponseDecodeError::WrongKind {
            path: "$/corp_name".to_owned(),
            expected: SourceValueKind::String,
            actual: company_name.kind(),
        });
    }
    Ok(())
}

fn decode_company_overview_json(
    value: SourceValue,
) -> Result<CompanyOverviewJsonResponse, ResponseDecodeError> {
    validate_company_overview(&value)?;
    Ok(CompanyOverviewJsonResponse { source: value })
}

fn decode_company_overview_xml(
    value: SourceValue,
) -> Result<CompanyOverviewXmlResponse, ResponseDecodeError> {
    validate_company_overview(&value)?;
    Ok(CompanyOverviewXmlResponse { source: value })
}

fn require_nonempty(
    operation: OperationIdentity,
    parameter: &'static str,
    value: &str,
) -> Result<(), PrepareError> {
    if value.is_empty() {
        return Err(PrepareError::MissingInput {
            operation,
            parameter,
        });
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{ApiKey, RequestMethod, ResponseInterpretError, SourceReply, WireInspector};

    const EXECUTABLE_CASES: &str = include_str!("../../../conformance/obligations.toml");

    type CompanyJsonDecoder =
        fn(SourceValue) -> Result<CompanyOverviewJsonResponse, ResponseDecodeError>;

    #[cfg(opendart_compat)]
    fn fixture(name: &str) -> Vec<u8> {
        std::fs::read(
            std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
                .join("../../../../openapi/fixtures/v1/bodies")
                .join(name),
        )
        .expect("repository contract fixture is readable")
    }

    #[derive(Clone, Debug, Eq, PartialEq)]
    struct RequestObservation {
        method: RequestMethod,
        path: String,
        query: String,
        authentication: Authentication,
        physical: String,
        logical: String,
        representations: Vec<Representation>,
    }

    #[derive(Clone, Copy, Debug, Eq, PartialEq)]
    enum FailureDimension {
        Path,
        ParameterName,
        Encoding,
        Requiredness,
        AllowedValue,
        OperationIdentity,
        ResponseBinding,
        MediaRouting,
        XmlRoot,
        ResponseFieldShape,
        SourceRetention,
    }

    fn observe<T>(prepared: &PreparedRequest<T>) -> RequestObservation {
        RequestObservation {
            method: prepared.method(),
            path: prepared.relative_path().to_owned(),
            query: prepared.encoded_query().to_owned(),
            authentication: prepared.authentication(),
            physical: prepared.identity().physical().to_owned(),
            logical: prepared.identity().logical().to_owned(),
            representations: prepared.expected_representations().to_vec(),
        }
    }

    fn company_json_adapter(
        path: &'static str,
        identity: OperationIdentity,
        parameter_name: &'static str,
        parameter_value: &str,
        representations: &'static [Representation],
        xml_root: Option<&'static str>,
        decoder: CompanyJsonDecoder,
    ) -> PreparedRequest<CompanyOverviewJsonResponse> {
        let parameters = [QueryParameter {
            name: parameter_name,
            value: QueryValue::Scalar(parameter_value),
        }];
        PreparedRequest::new(
            RequestParts::new(
                path,
                identity,
                &parameters,
                representations,
                xml_root,
                0,
                HANDWRITTEN_CONTRACT,
            ),
            decoder,
        )
    }

    fn expected_company_json(query: &str) -> RequestObservation {
        RequestObservation {
            method: RequestMethod::Get,
            path: "/api/company.json".to_owned(),
            query: query.to_owned(),
            authentication: Authentication::ApiKeyQuery,
            physical: "get_company_json".to_owned(),
            logical: "DS001-2019002".to_owned(),
            representations: JSON.to_vec(),
        }
    }

    fn decode_company_overview_json_without_shape_check(
        value: SourceValue,
    ) -> Result<CompanyOverviewJsonResponse, ResponseDecodeError> {
        Ok(CompanyOverviewJsonResponse { source: value })
    }

    fn decode_company_overview_json_lossy(
        value: SourceValue,
    ) -> Result<CompanyOverviewJsonResponse, ResponseDecodeError> {
        validate_company_overview(&value)?;
        let mut retained = std::collections::BTreeMap::new();
        for name in ["status", "message", "corp_name"] {
            if let Some(field) = value.get(name) {
                retained.insert(name.to_owned(), field.clone());
            }
        }
        Ok(CompanyOverviewJsonResponse {
            source: SourceValue::object(retained),
        })
    }

    fn decode_company_overview_json_with_wrong_binding(
        value: SourceValue,
    ) -> Result<CompanyOverviewJsonResponse, ResponseDecodeError> {
        validate_company_overview(&value)?;
        if value.get("list").is_none() {
            return Err(ResponseDecodeError::MissingRequired {
                path: "$/list".to_owned(),
            });
        }
        Ok(CompanyOverviewJsonResponse { source: value })
    }

    fn mismatch(
        expected: &RequestObservation,
        candidate: &RequestObservation,
    ) -> Option<FailureDimension> {
        if expected.path != candidate.path {
            return Some(FailureDimension::Path);
        }
        if expected.query != candidate.query {
            let expected_name = expected.query.split('=').next();
            let candidate_name = candidate.query.split('=').next();
            return Some(if expected_name != candidate_name {
                FailureDimension::ParameterName
            } else {
                FailureDimension::Encoding
            });
        }
        if expected.physical != candidate.physical
            || expected.logical != candidate.logical
            || expected.representations != candidate.representations
        {
            return Some(FailureDimension::OperationIdentity);
        }
        None
    }

    fn killed(
        dimension: FailureDimension,
        expected_behavior_observed: bool,
        mutant_behavior_observed: bool,
    ) {
        assert!(
            expected_behavior_observed,
            "conformer did not expose expected {dimension:?} behavior"
        );
        assert!(
            mutant_behavior_observed,
            "mutant did not expose faulty {dimension:?} behavior"
        );
    }

    #[test]
    fn executable_cases_cover_reviewed_operations() {
        let operation_ids = executable_operation_ids(EXECUTABLE_CASES);
        assert_eq!(operation_ids.len(), 3);
        for operation_id in operation_ids {
            match operation_id {
                "get_company_json" => executable_get_company_json(),
                "get_company_xml" => executable_get_company_xml(),
                "get_corpCode_xml" => executable_get_corp_code_xml(),
                other => panic!("unregistered executable operation {other}"),
            }
        }
    }

    fn executable_operation_ids(manifest: &str) -> Vec<&str> {
        let mut current = None;
        let mut executable = Vec::new();
        for line in manifest.lines().map(str::trim) {
            if line == "[[case]]" {
                current = None;
            } else if let Some(value) = line
                .strip_prefix("operation_id = \"")
                .and_then(|value| value.strip_suffix('"'))
            {
                current = Some(value);
            } else if line == "executable = true" {
                executable.push(current.expect("executable case must have an operation ID"));
            }
        }
        executable
    }

    fn executable_get_company_json() {
        let request = CompanyOverviewInput::new("00126380");
        let json = request.prepare_json().expect("JSON pilot should prepare");
        assert_eq!(json.method(), RequestMethod::Get);
        assert_eq!(json.relative_path(), "/api/company.json");
        assert_eq!(json.encoded_query(), "corp_code=00126380");
        assert_eq!(json.authentication(), Authentication::ApiKeyQuery);
        assert_eq!(json.identity().physical(), "get_company_json");
        assert_eq!(json.identity().logical(), "DS001-2019002");
        assert_eq!(json.expected_representations(), JSON);
        assert_eq!(json.generator_schema(), 0);
        assert_eq!(json.projection_identity(), HANDWRITTEN_CONTRACT);
    }

    fn executable_get_company_xml() {
        let request = CompanyOverviewInput::new("00126380");
        let xml = request.prepare_xml().expect("XML pilot should prepare");
        assert_eq!(xml.relative_path(), "/api/company.xml");
        assert_eq!(xml.identity().physical(), "get_company_xml");
        assert_eq!(xml.expected_representations(), XML);
    }

    fn executable_get_corp_code_xml() {
        let archive = CompanyCodesInput.prepare_archive();
        assert_eq!(archive.method(), RequestMethod::Get);
        assert_eq!(archive.relative_path(), "/api/corpCode.xml");
        assert_eq!(archive.encoded_query(), "");
        assert_eq!(archive.identity().physical(), "get_corpCode_xml");
        assert_eq!(archive.identity().logical(), "DS001-2019018");
        assert_eq!(archive.expected_representations(), ZIP_OR_XML);
    }

    #[test]
    fn wrapper_retains_one_complete_source_and_serializes_through_it() {
        let input = CompanyOverviewInput::new("00126380");
        let prepared = input.prepare_json().unwrap();
        let inspector = WireInspector::new(1024).unwrap();
        let key = ApiKey::new("fixture-key").unwrap();
        let json_body =
            br#"{"status":"000","message":"ok","corp_name":"Example","future":{"nested":true}}"#;
        let SourceReply::Success(profile) = prepared
            .interpret_response(&inspector, &key, 200, json_body)
            .unwrap()
        else {
            panic!("payload fields should produce a success wrapper");
        };

        assert_eq!(
            profile.status().as_ref().map(SourceStatus::as_str),
            Some("000")
        );
        assert_eq!(profile.message().and_then(SourceValue::as_str), Some("ok"));
        assert_eq!(profile.legal_name(), Some("Example"));
        assert_eq!(
            profile
                .field("future")
                .and_then(|value| value.get("nested"))
                .and_then(SourceValue::as_bool),
            Some(true)
        );
        let SourceReply::Success(expected_json) = inspector.inspect_json(json_body).unwrap() else {
            panic!("JSON fixture should be a success envelope");
        };
        assert_eq!(profile.source(), &expected_json);

        #[cfg(feature = "serde-json")]
        assert_eq!(
            serde_json::to_value(&profile).unwrap(),
            serde_json::json!({
                "status": "000",
                "message": "ok",
                "corp_name": "Example",
                "future": {"nested": true}
            })
        );

        let xml_body = b"<result><status>000</status><message>ok</message><corp_name>Example</corp_name><future>kept</future></result>";
        let SourceReply::Success(xml) = input
            .prepare_xml()
            .unwrap()
            .interpret_response(&inspector, &key, 200, xml_body)
            .unwrap()
        else {
            panic!("XML payload fields should produce its distinct wrapper");
        };
        let SourceReply::Success(expected_xml) = inspector.inspect_xml(xml_body).unwrap() else {
            panic!("XML fixture should be a success envelope");
        };
        assert_eq!(xml.source(), &expected_xml);
        assert_eq!(xml.status().as_ref().map(SourceStatus::as_str), Some("000"));
        assert_eq!(xml.message().and_then(SourceValue::as_str), Some("ok"));
        assert_eq!(xml.legal_name(), Some("Example"));
        assert_eq!(
            xml.field("future").and_then(SourceValue::as_str),
            Some("kept")
        );
    }

    #[cfg(opendart_compat)]
    #[test]
    fn retained_protocol_evidence_crosses_the_handwritten_interpreter() {
        let request = CompanyOverviewInput::new("00126380");
        let json = request.prepare_json().unwrap();
        let xml = request.prepare_xml().unwrap();
        let inspector = WireInspector::new(2048).unwrap();
        let key = ApiKey::new("fixture-key").unwrap();

        let json_body = fixture("company-success.json");
        let SourceReply::Success(json_profile) = json
            .interpret_response(&inspector, &key, 200, &json_body)
            .unwrap()
        else {
            panic!("retained JSON fixture should construct the wrapper");
        };
        assert_eq!(
            json_profile
                .field("future")
                .and_then(|value| value.get("exact"))
                .and_then(SourceValue::as_number_str),
            Some("9007199254740993")
        );
        assert_eq!(json_profile.legal_name(), Some("Example Corp"));

        let xml_body = fixture("company-success.xml");
        let SourceReply::Success(xml_profile) = xml
            .interpret_response(&inspector, &key, 200, &xml_body)
            .unwrap()
        else {
            panic!("retained XML fixture should construct the wrapper");
        };
        assert_eq!(
            xml_profile.field("future").and_then(SourceValue::as_str),
            Some("kept")
        );
        assert_eq!(
            xml_profile.status().as_ref().map(SourceStatus::as_str),
            Some("000")
        );
        assert_eq!(xml_profile.message(), None);
        let SourceReply::Success(expected_xml) = inspector.inspect_xml(&xml_body).unwrap() else {
            panic!("retained XML fixture should be a success envelope");
        };
        assert_eq!(xml_profile.source(), &expected_xml);
        assert_eq!(xml_profile.legal_name(), Some("Example Corp"));

        let SourceReply::Status(status) = json
            .interpret_response(&inspector, &key, 200, &fixture("status-013.json"))
            .unwrap()
        else {
            panic!("retained provider status should remain source evidence");
        };
        assert_eq!(status.code.as_str(), "013");
        assert_eq!(
            status.evidence.get("future").and_then(SourceValue::as_str),
            Some("kept")
        );

        assert!(matches!(
            json.interpret_response(&inspector, &key, 200, &fixture("malformed.json")),
            Err(ResponseInterpretError::Envelope { .. })
        ));
        assert!(matches!(
            xml.interpret_response(&inspector, &key, 200, &fixture("wrong-root.xml")),
            Err(ResponseInterpretError::Decode {
                source: ResponseDecodeError::UnexpectedXmlRoot { expected: "result" },
                ..
            })
        ));

        let SourceReply::Status(unknown) = json
            .interpret_response(
                &inspector,
                &key,
                200,
                br#"{"status":"future-status","message":"unknown"}"#,
            )
            .unwrap()
        else {
            panic!("unknown source status should remain representable");
        };
        assert_eq!(unknown.code.as_str(), "future-status");

        let reflected_key = ApiKey::new("secret value").unwrap();
        assert!(matches!(
            json.interpret_response(
                &inspector,
                &reflected_key,
                500,
                br#"{"status":"999","message":"secret value"}"#,
            ),
            Err(ResponseInterpretError::HttpStatus { evidence: None, .. })
        ));
    }

    #[test]
    fn request_fault_adapters_fail_the_intended_public_expectation() {
        let identity = OperationIdentity::new("get_company_json", "DS001-2019002");
        let expected = expected_company_json("corp_code=00126380");

        let wrong_path = company_json_adapter(
            "/api/list.json",
            identity,
            "corp_code",
            "00126380",
            JSON,
            None,
            decode_company_overview_json,
        );
        assert_eq!(
            mismatch(&expected, &observe(&wrong_path)),
            Some(FailureDimension::Path)
        );

        let wrong_name = company_json_adapter(
            "/api/company.json",
            identity,
            "company_code",
            "00126380",
            JSON,
            None,
            decode_company_overview_json,
        );
        assert_eq!(
            mismatch(&expected, &observe(&wrong_name)),
            Some(FailureDimension::ParameterName)
        );

        let encoded = company_json_adapter(
            "/api/company.json",
            identity,
            "corp_code",
            "회사 /+",
            JSON,
            None,
            decode_company_overview_json,
        );
        let expected_encoded = expected_company_json("corp_code=%ED%9A%8C%EC%82%AC+%2F%2B");
        assert_eq!(observe(&encoded), expected_encoded);
        let wrong_encoding = company_json_adapter(
            "/api/company.json",
            identity,
            "corp_code",
            "%ED%9A%8C%EC%82%AC+%2F%2B",
            JSON,
            None,
            decode_company_overview_json,
        );
        assert_eq!(
            mismatch(&expected_encoded, &observe(&wrong_encoding)),
            Some(FailureDimension::Encoding)
        );

        let wrong_binding = company_json_adapter(
            "/api/company.json",
            OperationIdentity::new("get_company_xml", "DS001-2019002"),
            "corp_code",
            "00126380",
            JSON,
            None,
            decode_company_overview_json,
        );
        assert_eq!(
            mismatch(&expected, &observe(&wrong_binding)),
            Some(FailureDimension::OperationIdentity)
        );

        let inspector = WireInspector::new(1024).unwrap();
        let key = ApiKey::new("fixture-key").unwrap();
        let response = br#"{"status":"000","corp_name":"Example"}"#;
        let correctly_bound = company_json_adapter(
            "/api/company.json",
            identity,
            "corp_code",
            "00126380",
            JSON,
            None,
            decode_company_overview_json,
        );
        let incorrectly_bound = company_json_adapter(
            "/api/company.json",
            identity,
            "corp_code",
            "00126380",
            JSON,
            None,
            decode_company_overview_json_with_wrong_binding,
        );
        let dimension = FailureDimension::ResponseBinding;
        assert!(
            correctly_bound
                .interpret_response(&inspector, &key, 200, response)
                .is_ok(),
            "conformer rejected valid {dimension:?}"
        );
        assert!(
            incorrectly_bound
                .interpret_response(&inspector, &key, 200, response)
                .is_err(),
            "wrong decoder survived {dimension:?} mutation"
        );
    }

    #[test]
    fn validation_fault_adapters_are_killed_by_stable_categories() {
        let missing = AnnualReportRequest::new("", "2025", "11011");
        assert!(matches!(
            missing.prepare_json(),
            Err(PrepareError::MissingInput {
                parameter: "corp_code",
                ..
            })
        ));
        assert!(
            missing
                .prepare_json_with_fault(ValidationFault::IgnoreRequiredness)
                .is_ok()
        );
        killed(
            FailureDimension::Requiredness,
            missing.prepare_json().is_err(),
            missing
                .prepare_json_with_fault(ValidationFault::IgnoreRequiredness)
                .is_ok(),
        );

        let invalid = AnnualReportRequest::new("00126380", "2025", "99999");
        assert!(matches!(
            invalid.prepare_json(),
            Err(PrepareError::InvalidAllowedValue {
                parameter: "reprt_code",
                ..
            })
        ));
        assert!(
            invalid
                .prepare_json_with_fault(ValidationFault::IgnoreAllowedValues)
                .is_ok()
        );
        killed(
            FailureDimension::AllowedValue,
            invalid.prepare_json().is_err(),
            invalid
                .prepare_json_with_fault(ValidationFault::IgnoreAllowedValues)
                .is_ok(),
        );
    }

    #[test]
    fn response_fault_adapters_fail_at_media_root_shape_and_retention() {
        let request = CompanyOverviewInput::new("00126380");
        let json = request.prepare_json().unwrap();
        let xml = request.prepare_xml().unwrap();
        let inspector = WireInspector::new(2048).unwrap();
        let key = ApiKey::new("fixture-key").unwrap();

        let faulty_media = company_json_adapter(
            "/api/company.json",
            OperationIdentity::new("get_company_json", "DS001-2019002"),
            "corp_code",
            "00126380",
            XML,
            Some("result"),
            decode_company_overview_json,
        );

        let xml_body = br#"<result><status>000</status><corp_name>Example</corp_name></result>"#;
        let media = json
            .interpret_response(&inspector, &key, 200, xml_body)
            .unwrap_err();
        assert!(matches!(media, ResponseInterpretError::Envelope { .. }));
        killed(
            FailureDimension::MediaRouting,
            json.interpret_response(&inspector, &key, 200, xml_body)
                .is_err(),
            faulty_media
                .interpret_response(&inspector, &key, 200, xml_body)
                .is_ok(),
        );

        let wrong_root_body =
            br#"<wrong><status>000</status><corp_name>Example</corp_name></wrong>"#;
        let root = xml
            .interpret_response(&inspector, &key, 200, wrong_root_body)
            .unwrap_err();
        assert!(matches!(
            root,
            ResponseInterpretError::Decode {
                source: ResponseDecodeError::UnexpectedXmlRoot { expected: "result" },
                ..
            }
        ));
        let faulty_root = PreparedRequest::new(
            RequestParts::new(
                "/api/company.xml",
                OperationIdentity::new("get_company_xml", "DS001-2019002"),
                &[QueryParameter {
                    name: "corp_code",
                    value: QueryValue::Scalar("00126380"),
                }],
                XML,
                Some("wrong"),
                0,
                HANDWRITTEN_CONTRACT,
            ),
            decode_company_overview_xml,
        );
        killed(
            FailureDimension::XmlRoot,
            xml.interpret_response(&inspector, &key, 200, wrong_root_body)
                .is_err(),
            faulty_root
                .interpret_response(&inspector, &key, 200, wrong_root_body)
                .is_ok(),
        );

        let wrong_shape_body = br#"{"status":"000","corp_name":[]}"#;
        let shape = json
            .interpret_response(&inspector, &key, 200, wrong_shape_body)
            .unwrap_err();
        assert!(matches!(
            shape,
            ResponseInterpretError::Decode {
                source: ResponseDecodeError::WrongKind { ref path, .. },
                ..
            } if path == "$/corp_name"
        ));
        let faulty_shape = company_json_adapter(
            "/api/company.json",
            OperationIdentity::new("get_company_json", "DS001-2019002"),
            "corp_code",
            "00126380",
            JSON,
            None,
            decode_company_overview_json_without_shape_check,
        );
        killed(
            FailureDimension::ResponseFieldShape,
            json.interpret_response(&inspector, &key, 200, wrong_shape_body)
                .is_err(),
            faulty_shape
                .interpret_response(&inspector, &key, 200, wrong_shape_body)
                .is_ok(),
        );

        let retained_body = br#"{"status":"000","corp_name":"Example","future":true}"#;
        let SourceReply::Success(profile) = json
            .interpret_response(&inspector, &key, 200, retained_body)
            .unwrap()
        else {
            panic!("expected success wrapper");
        };
        let faulty_retention = company_json_adapter(
            "/api/company.json",
            OperationIdentity::new("get_company_json", "DS001-2019002"),
            "corp_code",
            "00126380",
            JSON,
            None,
            decode_company_overview_json_lossy,
        );
        let SourceReply::Success(faulty_profile) = faulty_retention
            .interpret_response(&inspector, &key, 200, retained_body)
            .unwrap()
        else {
            panic!("faulty decoder should still construct a success wrapper");
        };
        let retained = profile.source().get("future").is_some();
        let faulty_retained = faulty_profile.source().get("future").is_some();
        killed(
            FailureDimension::SourceRetention,
            retained,
            !faulty_retained,
        );
    }

    #[cfg(all(
        opendart_compat,
        feature = "client-reqwest",
        not(target_family = "wasm")
    ))]
    #[tokio::test]
    async fn handwritten_zip_pilot_streams_exact_bytes_and_routes_xml_status() {
        use std::future::poll_fn;

        use bytes::Bytes;
        use futures_core::Stream;

        use crate::{BinaryReply, client::classify_binary_fixture};

        let prepared = CompanyCodesInput.prepare_archive();
        let archive_chunks = vec![
            Bytes::from_static(b"PK\x03"),
            Bytes::from_static(b"\x04body"),
        ];
        let BinaryReply::Archive(mut stream) = classify_binary_fixture(
            archive_chunks,
            WireInspector::new(1024).unwrap(),
            Some("result"),
        )
        .await
        else {
            panic!("ZIP signature should select the streaming archive branch");
        };
        let mut replayed = Vec::new();
        while let Some(chunk) =
            poll_fn(|context| std::pin::Pin::new(&mut stream).poll_next(context)).await
        {
            replayed.extend_from_slice(chunk.unwrap().as_bytes());
        }
        assert_eq!(replayed, b"PK\x03\x04body");
        assert_eq!(prepared.expected_representations(), ZIP_OR_XML);

        let BinaryReply::Status(status) = classify_binary_fixture(
            vec![Bytes::from(fixture("zip-error-010.xml"))],
            WireInspector::new(1024).unwrap(),
            Some("result"),
        )
        .await
        else {
            panic!("bounded XML status should select the alternate-status branch");
        };
        assert_eq!(status.code.as_str(), "010");
    }
}
