//! Executable handwritten conformance controls.

use crate::{
    Authentication, OperationIdentity, PrepareError, PreparedRequest, Representation,
    ResponseDecodeError, SourceReply, SourceStatus, SourceValue, SourceValueKind,
    operations::{
        annual_report::AccountingAuditorNameAndAuditOpinionInput,
        disclosure::{CompanyOverviewInput, DisclosureSearchInput},
        financial_statement::{
            CompaniesFinancialIndicatorsInput, CompanyFinancialIndicatorsInput,
            CompanyFinancialStatementsInput,
        },
    },
    request::{QueryParameter, QueryValue, RequestParts},
};

const JSON: &[Representation] = &[Representation::Json];
const XML: &[Representation] = &[Representation::Xml];

#[cfg(test)]
mod tests {
    use std::collections::BTreeSet;

    use super::*;
    use crate::{ApiKey, RequestMethod, ResponseInterpretError, WireInspector};

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

    struct ExecutableCase {
        operation_id: &'static str,
        logical_id: &'static str,
        run: fn(),
    }

    #[derive(Clone, Copy, Debug, Eq, PartialEq)]
    enum ResponseTopology {
        Root,
        List,
        GroupList,
    }

    macro_rules! structured_executable_case {
        ($physical:literal, $logical:literal, $topology:expr, $prepare:expr) => {
            ExecutableCase {
                operation_id: $physical,
                logical_id: $logical,
                run: || {
                    let prepared = $prepare.expect("reviewed inputs must prepare");
                    assert_eq!(prepared.method(), RequestMethod::Get);
                    assert_eq!(prepared.authentication(), Authentication::ApiKeyQuery);
                    assert_eq!(prepared.identity().physical(), $physical);
                    assert_eq!(prepared.identity().logical(), $logical);
                    assert_eq!(prepared.relative_path(), expected_relative_path($physical));
                    assert_eq!(prepared.encoded_query(), expected_query($logical));
                    let expected = if $physical.ends_with("_json") {
                        Representation::Json
                    } else {
                        Representation::Xml
                    };
                    assert!(prepared.expected_representations().contains(&expected));
                    let body = universal_response_body(expected, $topology);
                    let inspector = WireInspector::new(1024).unwrap();
                    let key = ApiKey::new("fixture-key").unwrap();
                    assert!(matches!(
                        prepared
                            .interpret_response(&inspector, &key, 200, body)
                            .expect("universal reviewed response must interpret"),
                        SourceReply::Success(_)
                    ));
                    assert!(matches!(
                        prepared.decode(wrong_kind_root()),
                        Err(ResponseDecodeError::WrongKind {
                            ref path,
                            expected: SourceValueKind::Object,
                            actual: SourceValueKind::Array,
                        }) if path == "$"
                    ));
                },
            }
        };
    }

    macro_rules! executable_case {
        ($physical:literal, $logical:literal, $prepare:expr) => {
            structured_executable_case!($physical, $logical, ResponseTopology::List, $prepare)
        };
    }

    macro_rules! root_executable_case {
        ($physical:literal, $logical:literal, $prepare:expr) => {
            structured_executable_case!($physical, $logical, ResponseTopology::Root, $prepare)
        };
    }

    macro_rules! group_list_executable_case {
        ($physical:literal, $logical:literal, $prepare:expr) => {
            structured_executable_case!($physical, $logical, ResponseTopology::GroupList, $prepare)
        };
    }

    macro_rules! binary_executable_case {
        ($physical:literal, $logical:literal, $prepare:expr) => {
            ExecutableCase {
                operation_id: $physical,
                logical_id: $logical,
                run: || {
                    let prepared = $prepare.expect("reviewed inputs must prepare");
                    assert_eq!(prepared.method(), RequestMethod::Get);
                    assert_eq!(prepared.authentication(), Authentication::ApiKeyQuery);
                    assert_eq!(prepared.identity().physical(), $physical);
                    assert_eq!(prepared.identity().logical(), $logical);
                    assert_eq!(prepared.relative_path(), expected_relative_path($physical));
                    assert_eq!(prepared.encoded_query(), expected_query($logical));
                    assert!(
                        prepared
                            .expected_representations()
                            .contains(&Representation::Zip)
                    );
                    #[cfg(all(
                        opendart_compat,
                        feature = "client-reqwest",
                        not(target_family = "wasm")
                    ))]
                    exercise_binary_alternate_status(&prepared);
                },
            }
        };
    }

    #[cfg(all(
        opendart_compat,
        feature = "client-reqwest",
        not(target_family = "wasm")
    ))]
    fn exercise_binary_alternate_status(prepared: &crate::PreparedBinaryRequest) {
        use tokio::{
            io::{AsyncReadExt, AsyncWriteExt},
            net::TcpListener,
        };

        tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .build()
            .expect("binary conformance runtime must build")
            .block_on(async {
                let body = b"<result><status>010</status><message>no file</message></result>";
                let listener = TcpListener::bind("127.0.0.1:0")
                    .await
                    .expect("binary conformance listener must bind");
                let address = listener
                    .local_addr()
                    .expect("binary conformance listener has an address");
                let server = tokio::spawn(async move {
                    let (mut socket, _) = listener
                        .accept()
                        .await
                        .expect("binary conformance connection must arrive");
                    let mut request = Vec::new();
                    let mut buffer = [0; 1024];
                    while !request.windows(4).any(|window| window == b"\r\n\r\n") {
                        let read = socket
                            .read(&mut buffer)
                            .await
                            .expect("binary conformance request must be readable");
                        if read == 0 {
                            break;
                        }
                        request.extend_from_slice(&buffer[..read]);
                    }
                    let headers = format!(
                        "HTTP/1.1 200 OK\r\nContent-Type: application/xml\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
                        body.len()
                    );
                    socket
                        .write_all(headers.as_bytes())
                        .await
                        .expect("binary conformance headers must be writable");
                    socket
                        .write_all(body)
                        .await
                        .expect("binary conformance body must be writable");
                });
                let client = crate::Client::builder(ApiKey::new("fixture-key").unwrap())
                    .compatibility_origin(format!("http://{address}"))
                    .build()
                    .expect("binary conformance client must build");
                let response = client
                    .execute_binary(prepared)
                    .await
                    .expect("prepared binary request must execute");
                let crate::BinaryReply::Status(status) = response.reply else {
                    panic!("alternate XML body must select the status branch");
                };
                assert_eq!(status.code.as_str(), "010");
                server.await.expect("binary conformance server must finish");
            });
    }

    fn expected_relative_path(operation_id: &str) -> String {
        let target = operation_id
            .strip_prefix("get_")
            .expect("reviewed physical IDs use the get_ prefix");
        if let Some(stem) = target.strip_suffix("_json") {
            format!("/api/{stem}.json")
        } else {
            let stem = target
                .strip_suffix("_xml")
                .expect("reviewed physical IDs use a representation suffix");
            format!("/api/{stem}.xml")
        }
    }

    fn expected_query(logical_id: &str) -> &'static str {
        if logical_id.starts_with("DS002-") {
            return "corp_code=00126380&bsns_year=2025&reprt_code=11011";
        }
        if logical_id.starts_with("DS005-") || logical_id.starts_with("DS006-") {
            return "corp_code=00126380&bgn_de=20250101&end_de=20251231";
        }
        if logical_id.starts_with("DS004-") {
            return "corp_code=00126380";
        }
        match logical_id {
            "DS001-2019001" | "DS001-2019018" => "",
            "DS001-2019002" => "corp_code=00126380",
            "DS001-2019003" => "rcept_no=20250101000001",
            "DS003-2019016" | "DS003-2019017" => {
                "corp_code=00126380&bsns_year=2025&reprt_code=11011"
            }
            "DS003-2019019" => "rcept_no=20250101000001&reprt_code=11011",
            "DS003-2019020" => "corp_code=00126380&bsns_year=2025&reprt_code=11011&fs_div=CFS",
            "DS003-2020001" => "sj_div=BS1",
            "DS003-2022001" | "DS003-2022002" => {
                "corp_code=00126380&bsns_year=2025&reprt_code=11011&idx_cl_code=M210000"
            }
            _ => panic!("unreviewed logical operation {logical_id}"),
        }
    }

    fn universal_response_body(
        representation: Representation,
        topology: ResponseTopology,
    ) -> &'static [u8] {
        match (representation, topology) {
            (Representation::Json, ResponseTopology::Root) => {
                br#"{"status":"000","message":"ok","list":"wrong-kind","group":"wrong-kind"}"#
            }
            (Representation::Json, ResponseTopology::List) => {
                br#"{"status":"000","message":"ok","list":[{}],"group":"wrong-kind"}"#
            }
            (Representation::Json, ResponseTopology::GroupList) => {
                br#"{"status":"000","message":"ok","list":"wrong-kind","group":[{"list":[{}]}]}"#
            }
            (Representation::Xml, ResponseTopology::Root) => b"<result><status>000</status><message>ok</message><list>wrong-kind</list><group>wrong-kind</group></result>",
            (Representation::Xml, ResponseTopology::List) => b"<result><status>000</status><message>ok</message><list><future>kept</future></list><group>wrong-kind</group></result>",
            (Representation::Xml, ResponseTopology::GroupList) => b"<result><status>000</status><message>ok</message><list>wrong-kind</list><group><list><future>kept</future></list></group></result>",
            (Representation::Zip, _) => unreachable!("binary cases use their lifecycle seam"),
        }
    }

    fn wrong_kind_root() -> SourceValue {
        SourceValue::array(Vec::new())
    }

    fn executable_cases() -> Vec<ExecutableCase> {
        vec![
        executable_case!("get_irdsSttus_json", "DS002-2019004", crate::operations::annual_report::CapitalIncreaseAndReductionStatusInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_irdsSttus_xml", "DS002-2019004", crate::operations::annual_report::CapitalIncreaseAndReductionStatusInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_alotMatter_json", "DS002-2019005", crate::operations::annual_report::DividendInformationInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_alotMatter_xml", "DS002-2019005", crate::operations::annual_report::DividendInformationInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_tesstkAcqsDspsSttus_json", "DS002-2019006", crate::operations::annual_report::TreasuryStockTransactionsInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_tesstkAcqsDspsSttus_xml", "DS002-2019006", crate::operations::annual_report::TreasuryStockTransactionsInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_hyslrSttus_json", "DS002-2019007", crate::operations::annual_report::LargestShareholderStatusInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_hyslrSttus_xml", "DS002-2019007", crate::operations::annual_report::LargestShareholderStatusInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_hyslrChgSttus_json", "DS002-2019008", crate::operations::annual_report::LargestShareholderChangesInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_hyslrChgSttus_xml", "DS002-2019008", crate::operations::annual_report::LargestShareholderChangesInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_mrhlSttus_json", "DS002-2019009", crate::operations::annual_report::MinorityShareholderStatusInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_mrhlSttus_xml", "DS002-2019009", crate::operations::annual_report::MinorityShareholderStatusInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_exctvSttus_json", "DS002-2019010", crate::operations::annual_report::ExecutiveStatusInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_exctvSttus_xml", "DS002-2019010", crate::operations::annual_report::ExecutiveStatusInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_empSttus_json", "DS002-2019011", crate::operations::annual_report::EmployeeStatusInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_empSttus_xml", "DS002-2019011", crate::operations::annual_report::EmployeeStatusInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_hmvAuditIndvdlBySttus_json", "DS002-2019012", crate::operations::annual_report::DirectorAuditorIndividualCompensationInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_hmvAuditIndvdlBySttus_xml", "DS002-2019012", crate::operations::annual_report::DirectorAuditorIndividualCompensationInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_hmvAuditAllSttus_json", "DS002-2019013", crate::operations::annual_report::DirectorAuditorTotalCompensationInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_hmvAuditAllSttus_xml", "DS002-2019013", crate::operations::annual_report::DirectorAuditorTotalCompensationInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_indvdlByPay_json", "DS002-2019014", crate::operations::annual_report::TopFiveIndividualCompensationInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_indvdlByPay_xml", "DS002-2019014", crate::operations::annual_report::TopFiveIndividualCompensationInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_otrCprInvstmntSttus_json", "DS002-2019015", crate::operations::annual_report::OtherCorporationInvestmentStatusInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_otrCprInvstmntSttus_xml", "DS002-2019015", crate::operations::annual_report::OtherCorporationInvestmentStatusInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_stockTotqySttus_json", "DS002-2020002", crate::operations::annual_report::TotalSharesStatusInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_stockTotqySttus_xml", "DS002-2020002", crate::operations::annual_report::TotalSharesStatusInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_detScritsIsuAcmslt_json", "DS002-2020003", crate::operations::annual_report::DebtSecuritiesIssuanceResultsInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_detScritsIsuAcmslt_xml", "DS002-2020003", crate::operations::annual_report::DebtSecuritiesIssuanceResultsInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_entrprsBilScritsNrdmpBlce_json", "DS002-2020004", crate::operations::annual_report::CommercialPaperOutstandingBalanceInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_entrprsBilScritsNrdmpBlce_xml", "DS002-2020004", crate::operations::annual_report::CommercialPaperOutstandingBalanceInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_srtpdPsndbtNrdmpBlce_json", "DS002-2020005", crate::operations::annual_report::ShortTermBondOutstandingBalanceInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_srtpdPsndbtNrdmpBlce_xml", "DS002-2020005", crate::operations::annual_report::ShortTermBondOutstandingBalanceInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_cprndNrdmpBlce_json", "DS002-2020006", crate::operations::annual_report::CorporateBondOutstandingBalanceInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_cprndNrdmpBlce_xml", "DS002-2020006", crate::operations::annual_report::CorporateBondOutstandingBalanceInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_newCaplScritsNrdmpBlce_json", "DS002-2020007", crate::operations::annual_report::HybridCapitalSecuritiesOutstandingBalanceInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_newCaplScritsNrdmpBlce_xml", "DS002-2020007", crate::operations::annual_report::HybridCapitalSecuritiesOutstandingBalanceInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_cndlCaplScritsNrdmpBlce_json", "DS002-2020008", crate::operations::annual_report::ContingentCapitalSecuritiesOutstandingBalanceInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_cndlCaplScritsNrdmpBlce_xml", "DS002-2020008", crate::operations::annual_report::ContingentCapitalSecuritiesOutstandingBalanceInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_accnutAdtorNmNdAdtOpinion_json", "DS002-2020009", crate::operations::annual_report::AccountingAuditorNameAndAuditOpinionInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_accnutAdtorNmNdAdtOpinion_xml", "DS002-2020009", crate::operations::annual_report::AccountingAuditorNameAndAuditOpinionInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_adtServcCnclsSttus_json", "DS002-2020010", crate::operations::annual_report::AuditServiceContractsInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_adtServcCnclsSttus_xml", "DS002-2020010", crate::operations::annual_report::AuditServiceContractsInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_accnutAdtorNonAdtServcCnclsSttus_json", "DS002-2020011", crate::operations::annual_report::NonAuditServiceContractsInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_accnutAdtorNonAdtServcCnclsSttus_xml", "DS002-2020011", crate::operations::annual_report::NonAuditServiceContractsInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_outcmpnyDrctrNdChangeSttus_json", "DS002-2020012", crate::operations::annual_report::OutsideDirectorStatusAndChangesInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_outcmpnyDrctrNdChangeSttus_xml", "DS002-2020012", crate::operations::annual_report::OutsideDirectorStatusAndChangesInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_unrstExctvMendngSttus_json", "DS002-2020013", crate::operations::annual_report::UnregisteredExecutiveCompensationInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_unrstExctvMendngSttus_xml", "DS002-2020013", crate::operations::annual_report::UnregisteredExecutiveCompensationInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_drctrAdtAllMendngSttusGmtsckConfmAmount_json", "DS002-2020014", crate::operations::annual_report::ShareholderApprovedDirectorAuditorCompensationInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_drctrAdtAllMendngSttusGmtsckConfmAmount_xml", "DS002-2020014", crate::operations::annual_report::ShareholderApprovedDirectorAuditorCompensationInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_drctrAdtAllMendngSttusMendngPymntamtTyCl_json", "DS002-2020015", crate::operations::annual_report::DirectorAuditorCompensationByTypeInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_drctrAdtAllMendngSttusMendngPymntamtTyCl_xml", "DS002-2020015", crate::operations::annual_report::DirectorAuditorCompensationByTypeInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_pssrpCptalUseDtls_json", "DS002-2020016", crate::operations::annual_report::PublicOfferingProceedsUsageInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_pssrpCptalUseDtls_xml", "DS002-2020016", crate::operations::annual_report::PublicOfferingProceedsUsageInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_prvsrpCptalUseDtls_json", "DS002-2020017", crate::operations::annual_report::PrivatePlacementProceedsUsageInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_prvsrpCptalUseDtls_xml", "DS002-2020017", crate::operations::annual_report::PrivatePlacementProceedsUsageInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        group_list_executable_case!("get_hmvAuditIndvdlBySttusV2_json", "DS002-2026001", crate::operations::annual_report::DirectorAuditorIndividualCompensationV2Input::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        group_list_executable_case!("get_hmvAuditIndvdlBySttusV2_xml", "DS002-2026001", crate::operations::annual_report::DirectorAuditorIndividualCompensationV2Input::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        group_list_executable_case!("get_indvdlByPayV2_json", "DS002-2026002", crate::operations::annual_report::TopFiveIndividualCompensationV2Input::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        group_list_executable_case!("get_indvdlByPayV2_xml", "DS002-2026002", crate::operations::annual_report::TopFiveIndividualCompensationV2Input::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_list_json", "DS001-2019001", crate::operations::disclosure::DisclosureSearchInput::new().prepare_json()),
        executable_case!("get_list_xml", "DS001-2019001", crate::operations::disclosure::DisclosureSearchInput::new().prepare_xml()),
        root_executable_case!("get_company_json", "DS001-2019002", crate::operations::disclosure::CompanyOverviewInput::new("00126380".to_owned()).prepare_json()),
        root_executable_case!("get_company_xml", "DS001-2019002", crate::operations::disclosure::CompanyOverviewInput::new("00126380".to_owned()).prepare_xml()),
        binary_executable_case!("get_corpCode_xml", "DS001-2019018", crate::operations::disclosure::CompanyCodesInput::new().prepare_archive()),
        binary_executable_case!("get_document_xml", "DS001-2019003", crate::operations::disclosure::DisclosureDocumentInput::new("20250101000001".to_owned()).prepare_archive()),
        executable_case!("get_fnlttSinglAcnt_json", "DS003-2019016", crate::operations::financial_statement::CompanyKeyAccountsInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_fnlttSinglAcnt_xml", "DS003-2019016", crate::operations::financial_statement::CompanyKeyAccountsInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        executable_case!("get_fnlttMultiAcnt_json", "DS003-2019017", crate::operations::financial_statement::CompaniesKeyAccountsInput::new(vec!["00126380".to_owned()], "2025".to_owned(), "11011".to_owned()).prepare_json()),
        executable_case!("get_fnlttMultiAcnt_xml", "DS003-2019017", crate::operations::financial_statement::CompaniesKeyAccountsInput::new(vec!["00126380".to_owned()], "2025".to_owned(), "11011".to_owned()).prepare_xml()),
        binary_executable_case!("get_fnlttXbrl_xml", "DS003-2019019", crate::operations::financial_statement::XbrlFinancialStatementsInput::new("20250101000001".to_owned(), "11011".to_owned()).prepare_archive()),
        executable_case!("get_fnlttSinglAcntAll_json", "DS003-2019020", crate::operations::financial_statement::CompanyFinancialStatementsInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned(), "CFS".to_owned()).prepare_json()),
        executable_case!("get_fnlttSinglAcntAll_xml", "DS003-2019020", crate::operations::financial_statement::CompanyFinancialStatementsInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned(), "CFS".to_owned()).prepare_xml()),
        executable_case!("get_xbrlTaxonomy_json", "DS003-2020001", crate::operations::financial_statement::XbrlTaxonomyInput::new("BS1".to_owned()).prepare_json()),
        executable_case!("get_xbrlTaxonomy_xml", "DS003-2020001", crate::operations::financial_statement::XbrlTaxonomyInput::new("BS1".to_owned()).prepare_xml()),
        executable_case!("get_fnlttSinglIndx_json", "DS003-2022001", crate::operations::financial_statement::CompanyFinancialIndicatorsInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned(), "M210000".to_owned()).prepare_json()),
        executable_case!("get_fnlttSinglIndx_xml", "DS003-2022001", crate::operations::financial_statement::CompanyFinancialIndicatorsInput::new("00126380".to_owned(), "2025".to_owned(), "11011".to_owned(), "M210000".to_owned()).prepare_xml()),
        executable_case!("get_fnlttCmpnyIndx_json", "DS003-2022002", crate::operations::financial_statement::CompaniesFinancialIndicatorsInput::new(vec!["00126380".to_owned()], "2025".to_owned(), "11011".to_owned(), "M210000".to_owned()).prepare_json()),
        executable_case!("get_fnlttCmpnyIndx_xml", "DS003-2022002", crate::operations::financial_statement::CompaniesFinancialIndicatorsInput::new(vec!["00126380".to_owned()], "2025".to_owned(), "11011".to_owned(), "M210000".to_owned()).prepare_xml()),
        executable_case!("get_astInhtrfEtcPtbkOpt_json", "DS005-2020018", crate::operations::material_event::OtherAssetTransactionsAndPutbackOptionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_astInhtrfEtcPtbkOpt_xml", "DS005-2020018", crate::operations::material_event::OtherAssetTransactionsAndPutbackOptionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_bnkMngtPcsp_json", "DS005-2020036", crate::operations::material_event::CreditorBankManagementDiscontinuationsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_bnkMngtPcsp_xml", "DS005-2020036", crate::operations::material_event::CreditorBankManagementDiscontinuationsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_wdCocobdIsDecsn_json", "DS005-2020037", crate::operations::material_event::WriteDownContingentCapitalSecuritiesIssuanceDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_wdCocobdIsDecsn_xml", "DS005-2020037", crate::operations::material_event::WriteDownContingentCapitalSecuritiesIssuanceDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_tsstkAqDecsn_json", "DS005-2020038", crate::operations::material_event::TreasuryStockAcquisitionDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_tsstkAqDecsn_xml", "DS005-2020038", crate::operations::material_event::TreasuryStockAcquisitionDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_tsstkDpDecsn_json", "DS005-2020039", crate::operations::material_event::TreasuryStockDisposalDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_tsstkDpDecsn_xml", "DS005-2020039", crate::operations::material_event::TreasuryStockDisposalDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_tsstkAqTrctrCnsDecsn_json", "DS005-2020040", crate::operations::material_event::TreasuryStockAcquisitionTrustContractExecutionDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_tsstkAqTrctrCnsDecsn_xml", "DS005-2020040", crate::operations::material_event::TreasuryStockAcquisitionTrustContractExecutionDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_tsstkAqTrctrCcDecsn_json", "DS005-2020041", crate::operations::material_event::TreasuryStockAcquisitionTrustContractTerminationDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_tsstkAqTrctrCcDecsn_xml", "DS005-2020041", crate::operations::material_event::TreasuryStockAcquisitionTrustContractTerminationDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_bsnInhDecsn_json", "DS005-2020042", crate::operations::material_event::BusinessAcquisitionDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_bsnInhDecsn_xml", "DS005-2020042", crate::operations::material_event::BusinessAcquisitionDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_bsnTrfDecsn_json", "DS005-2020043", crate::operations::material_event::BusinessTransferDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_bsnTrfDecsn_xml", "DS005-2020043", crate::operations::material_event::BusinessTransferDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_tgastInhDecsn_json", "DS005-2020044", crate::operations::material_event::TangibleAssetAcquisitionDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_tgastInhDecsn_xml", "DS005-2020044", crate::operations::material_event::TangibleAssetAcquisitionDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_tgastTrfDecsn_json", "DS005-2020045", crate::operations::material_event::TangibleAssetTransferDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_tgastTrfDecsn_xml", "DS005-2020045", crate::operations::material_event::TangibleAssetTransferDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_otcprStkInvscrInhDecsn_json", "DS005-2020046", crate::operations::material_event::OtherCompanyEquitySecuritiesAcquisitionDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_otcprStkInvscrInhDecsn_xml", "DS005-2020046", crate::operations::material_event::OtherCompanyEquitySecuritiesAcquisitionDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_otcprStkInvscrTrfDecsn_json", "DS005-2020047", crate::operations::material_event::OtherCompanyEquitySecuritiesTransferDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_otcprStkInvscrTrfDecsn_xml", "DS005-2020047", crate::operations::material_event::OtherCompanyEquitySecuritiesTransferDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_stkrtbdInhDecsn_json", "DS005-2020048", crate::operations::material_event::StockRelatedBondAcquisitionDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_stkrtbdInhDecsn_xml", "DS005-2020048", crate::operations::material_event::StockRelatedBondAcquisitionDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_stkrtbdTrfDecsn_json", "DS005-2020049", crate::operations::material_event::StockRelatedBondTransferDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_stkrtbdTrfDecsn_xml", "DS005-2020049", crate::operations::material_event::StockRelatedBondTransferDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_cmpMgDecsn_json", "DS005-2020050", crate::operations::material_event::CompanyMergerDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_cmpMgDecsn_xml", "DS005-2020050", crate::operations::material_event::CompanyMergerDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_cmpDvDecsn_json", "DS005-2020051", crate::operations::material_event::CompanySplitDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_cmpDvDecsn_xml", "DS005-2020051", crate::operations::material_event::CompanySplitDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_cmpDvmgDecsn_json", "DS005-2020052", crate::operations::material_event::CompanySplitMergerDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_cmpDvmgDecsn_xml", "DS005-2020052", crate::operations::material_event::CompanySplitMergerDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_stkExtrDecsn_json", "DS005-2020053", crate::operations::material_event::ShareExchangeTransferDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_stkExtrDecsn_xml", "DS005-2020053", crate::operations::material_event::ShareExchangeTransferDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_dfOcr_json", "DS005-2020019", crate::operations::material_event::DefaultEventsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_dfOcr_xml", "DS005-2020019", crate::operations::material_event::DefaultEventsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_bsnSp_json", "DS005-2020020", crate::operations::material_event::BusinessSuspensionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_bsnSp_xml", "DS005-2020020", crate::operations::material_event::BusinessSuspensionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_ctrcvsBgrq_json", "DS005-2020021", crate::operations::material_event::RehabilitationProceedingApplicationsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_ctrcvsBgrq_xml", "DS005-2020021", crate::operations::material_event::RehabilitationProceedingApplicationsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_dsRsOcr_json", "DS005-2020022", crate::operations::material_event::DissolutionEventsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_dsRsOcr_xml", "DS005-2020022", crate::operations::material_event::DissolutionEventsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_piicDecsn_json", "DS005-2020023", crate::operations::material_event::PaidInCapitalIncreaseDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_piicDecsn_xml", "DS005-2020023", crate::operations::material_event::PaidInCapitalIncreaseDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_fricDecsn_json", "DS005-2020024", crate::operations::material_event::BonusShareIssueDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_fricDecsn_xml", "DS005-2020024", crate::operations::material_event::BonusShareIssueDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_pifricDecsn_json", "DS005-2020025", crate::operations::material_event::CombinedPaidInAndBonusShareIssueDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_pifricDecsn_xml", "DS005-2020025", crate::operations::material_event::CombinedPaidInAndBonusShareIssueDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_crDecsn_json", "DS005-2020026", crate::operations::material_event::CapitalReductionDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_crDecsn_xml", "DS005-2020026", crate::operations::material_event::CapitalReductionDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_bnkMngtPcbg_json", "DS005-2020027", crate::operations::material_event::CreditorBankManagementCommencementsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_bnkMngtPcbg_xml", "DS005-2020027", crate::operations::material_event::CreditorBankManagementCommencementsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_lwstLg_json", "DS005-2020028", crate::operations::material_event::LawsuitFilingsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_lwstLg_xml", "DS005-2020028", crate::operations::material_event::LawsuitFilingsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_ovLstDecsn_json", "DS005-2020029", crate::operations::material_event::OverseasMarketListingDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_ovLstDecsn_xml", "DS005-2020029", crate::operations::material_event::OverseasMarketListingDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_ovDlstDecsn_json", "DS005-2020030", crate::operations::material_event::OverseasMarketDelistingDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_ovDlstDecsn_xml", "DS005-2020030", crate::operations::material_event::OverseasMarketDelistingDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_ovLst_json", "DS005-2020031", crate::operations::material_event::OverseasMarketListingsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_ovLst_xml", "DS005-2020031", crate::operations::material_event::OverseasMarketListingsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_ovDlst_json", "DS005-2020032", crate::operations::material_event::OverseasMarketDelistingsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_ovDlst_xml", "DS005-2020032", crate::operations::material_event::OverseasMarketDelistingsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_cvbdIsDecsn_json", "DS005-2020033", crate::operations::material_event::ConvertibleBondIssuanceDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_cvbdIsDecsn_xml", "DS005-2020033", crate::operations::material_event::ConvertibleBondIssuanceDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_bdwtIsDecsn_json", "DS005-2020034", crate::operations::material_event::BondWithWarrantsIssuanceDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_bdwtIsDecsn_xml", "DS005-2020034", crate::operations::material_event::BondWithWarrantsIssuanceDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_exbdIsDecsn_json", "DS005-2020035", crate::operations::material_event::ExchangeableBondIssuanceDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        executable_case!("get_exbdIsDecsn_xml", "DS005-2020035", crate::operations::material_event::ExchangeableBondIssuanceDecisionsInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        executable_case!("get_majorstock_json", "DS004-2019021", crate::operations::ownership::MajorShareholdingsInput::new("00126380".to_owned()).prepare_json()),
        executable_case!("get_majorstock_xml", "DS004-2019021", crate::operations::ownership::MajorShareholdingsInput::new("00126380".to_owned()).prepare_xml()),
        executable_case!("get_elestock_json", "DS004-2019022", crate::operations::ownership::InsiderShareholdingsInput::new("00126380".to_owned()).prepare_json()),
        executable_case!("get_elestock_xml", "DS004-2019022", crate::operations::ownership::InsiderShareholdingsInput::new("00126380".to_owned()).prepare_xml()),
        group_list_executable_case!("get_estkRs_json", "DS006-2020054", crate::operations::registration_statement::EquitySecuritiesRegistrationInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        group_list_executable_case!("get_estkRs_xml", "DS006-2020054", crate::operations::registration_statement::EquitySecuritiesRegistrationInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        group_list_executable_case!("get_bdRs_json", "DS006-2020055", crate::operations::registration_statement::DebtSecuritiesRegistrationInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        group_list_executable_case!("get_bdRs_xml", "DS006-2020055", crate::operations::registration_statement::DebtSecuritiesRegistrationInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        group_list_executable_case!("get_stkdpRs_json", "DS006-2020056", crate::operations::registration_statement::DepositaryReceiptRegistrationInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        group_list_executable_case!("get_stkdpRs_xml", "DS006-2020056", crate::operations::registration_statement::DepositaryReceiptRegistrationInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        group_list_executable_case!("get_mgRs_json", "DS006-2020057", crate::operations::registration_statement::MergerRegistrationInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        group_list_executable_case!("get_mgRs_xml", "DS006-2020057", crate::operations::registration_statement::MergerRegistrationInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        group_list_executable_case!("get_extrRs_json", "DS006-2020058", crate::operations::registration_statement::ShareExchangeTransferRegistrationInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        group_list_executable_case!("get_extrRs_xml", "DS006-2020058", crate::operations::registration_statement::ShareExchangeTransferRegistrationInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        group_list_executable_case!("get_dvRs_json", "DS006-2020059", crate::operations::registration_statement::DemergerRegistrationInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_json()),
        group_list_executable_case!("get_dvRs_xml", "DS006-2020059", crate::operations::registration_statement::DemergerRegistrationInput::new("00126380".to_owned(), "20250101".to_owned(), "20251231".to_owned()).prepare_xml()),
        ]
    }

    #[test]
    fn executable_cases_cover_reviewed_operations() {
        let cases = executable_cases();
        assert_eq!(cases.len(), 167);
        let registered = cases
            .iter()
            .map(|case| case.operation_id)
            .collect::<BTreeSet<_>>();
        assert_eq!(registered.len(), 167, "registry contains a duplicate");
        assert_eq!(
            cases
                .iter()
                .map(|case| case.logical_id)
                .collect::<BTreeSet<_>>()
                .len(),
            85
        );

        for case in cases {
            (case.run)();
        }
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

    type SourceDecoder = fn(SourceValue) -> Result<SourceValue, ResponseDecodeError>;

    fn source_adapter(
        path: &'static str,
        identity: OperationIdentity,
        parameter_name: &'static str,
        parameter_value: &str,
        representations: &'static [Representation],
        xml_root: Option<&'static str>,
        decoder: SourceDecoder,
    ) -> PreparedRequest<SourceValue> {
        let parameters = [QueryParameter {
            name: parameter_name,
            value: QueryValue::Scalar(parameter_value),
        }];
        PreparedRequest::new(
            RequestParts::new(path, identity, &parameters, representations, xml_root),
            decoder,
        )
    }

    fn decode_identity(value: SourceValue) -> Result<SourceValue, ResponseDecodeError> {
        Ok(value)
    }

    fn decode_requiring_list(value: SourceValue) -> Result<SourceValue, ResponseDecodeError> {
        let list = value
            .get("list")
            .ok_or_else(|| ResponseDecodeError::MissingRequired {
                path: "$/list".to_owned(),
            })?;
        if list.kind() != SourceValueKind::Array {
            return Err(ResponseDecodeError::WrongKind {
                path: "$/list".to_owned(),
                expected: SourceValueKind::Array,
                actual: list.kind(),
            });
        }
        Ok(value)
    }

    fn decode_lossy(value: SourceValue) -> Result<SourceValue, ResponseDecodeError> {
        let mut retained = std::collections::BTreeMap::new();
        for name in ["status", "message", "corp_name"] {
            if let Some(field) = value.get(name) {
                retained.insert(name.to_owned(), field.clone());
            }
        }
        Ok(SourceValue::object(retained))
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
    fn wrapper_retains_one_complete_source_and_serializes_through_it() {
        let input = CompanyOverviewInput::new("00126380".to_owned());
        let prepared = input.prepare_json().unwrap();
        let inspector = WireInspector::new(1024).unwrap();
        let key = ApiKey::new("fixture-key").unwrap();
        let body =
            br#"{"status":"000","message":"ok","corp_name":"Example","future":{"nested":true}}"#;
        let SourceReply::Success(profile) = prepared
            .interpret_response(&inspector, &key, 200, body)
            .unwrap()
        else {
            panic!("payload fields should produce a success wrapper");
        };

        assert_eq!(
            profile.status().as_ref().map(SourceStatus::as_str),
            Some("000")
        );
        assert_eq!(profile.message().and_then(SourceValue::as_str), Some("ok"));
        assert_eq!(
            profile.legal_name().and_then(SourceValue::as_str),
            Some("Example")
        );
        assert_eq!(
            profile
                .field("future")
                .and_then(|value| value.get("nested"))
                .and_then(SourceValue::as_bool),
            Some(true)
        );
        let SourceReply::Success(expected) = inspector.inspect_json(body).unwrap() else {
            panic!("JSON fixture should be a success envelope");
        };
        assert_eq!(profile.source(), &expected);

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
        assert_eq!(
            xml.legal_name().and_then(SourceValue::as_str),
            Some("Example")
        );
        assert_eq!(
            xml.field("future").and_then(SourceValue::as_str),
            Some("kept")
        );
    }

    #[cfg(opendart_compat)]
    #[test]
    fn retained_fictional_protocol_evidence_crosses_the_handwritten_interpreter() {
        let input = CompanyOverviewInput::new("00126380".to_owned());
        let json = input.prepare_json().unwrap();
        let xml = input.prepare_xml().unwrap();
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
        let SourceReply::Success(expected_xml) = inspector.inspect_xml(&xml_body).unwrap() else {
            panic!("retained XML fixture should be a success envelope");
        };
        assert_eq!(xml_profile.source(), &expected_xml);

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
    }

    #[test]
    fn request_fault_adapters_fail_the_intended_public_expectation() {
        let identity = OperationIdentity::new("get_company_json", "DS001-2019002");
        let expected = expected_company_json("corp_code=00126380");

        let wrong_path = source_adapter(
            "/api/list.json",
            identity,
            "corp_code",
            "00126380",
            JSON,
            None,
            decode_identity,
        );
        assert_eq!(
            mismatch(&expected, &observe(&wrong_path)),
            Some(FailureDimension::Path)
        );

        let wrong_name = source_adapter(
            "/api/company.json",
            identity,
            "company_code",
            "00126380",
            JSON,
            None,
            decode_identity,
        );
        assert_eq!(
            mismatch(&expected, &observe(&wrong_name)),
            Some(FailureDimension::ParameterName)
        );

        let encoded = source_adapter(
            "/api/company.json",
            identity,
            "corp_code",
            "회사 /+",
            JSON,
            None,
            decode_identity,
        );
        assert_eq!(
            observe(&encoded),
            expected_company_json("corp_code=%ED%9A%8C%EC%82%AC+%2F%2B")
        );
        let wrong_encoding = source_adapter(
            "/api/company.json",
            identity,
            "corp_code",
            "%ED%9A%8C%EC%82%AC+%2F%2B",
            JSON,
            None,
            decode_identity,
        );
        assert_eq!(
            mismatch(
                &expected_company_json("corp_code=%ED%9A%8C%EC%82%AC+%2F%2B"),
                &observe(&wrong_encoding)
            ),
            Some(FailureDimension::Encoding)
        );

        let wrong_binding = source_adapter(
            "/api/company.json",
            OperationIdentity::new("get_company_xml", "DS001-2019002"),
            "corp_code",
            "00126380",
            JSON,
            None,
            decode_identity,
        );
        assert_eq!(
            mismatch(&expected, &observe(&wrong_binding)),
            Some(FailureDimension::OperationIdentity)
        );

        let inspector = WireInspector::new(1024).unwrap();
        let key = ApiKey::new("fixture-key").unwrap();
        let response = br#"{"status":"000","corp_name":"Example"}"#;
        let correctly_bound = CompanyOverviewInput::new("00126380".to_owned())
            .prepare_json()
            .unwrap();
        let incorrectly_bound = source_adapter(
            "/api/company.json",
            identity,
            "corp_code",
            "00126380",
            JSON,
            None,
            decode_requiring_list,
        );
        killed(
            FailureDimension::ResponseBinding,
            correctly_bound
                .interpret_response(&inspector, &key, 200, response)
                .is_ok(),
            incorrectly_bound
                .interpret_response(&inspector, &key, 200, response)
                .is_err(),
        );
    }

    #[test]
    fn validation_fault_adapters_are_killed_by_stable_categories() {
        let missing = AccountingAuditorNameAndAuditOpinionInput::new(
            String::new(),
            "2025".to_owned(),
            "11011".to_owned(),
        );
        assert!(matches!(
            missing.prepare_json(),
            Err(PrepareError::MissingInput {
                parameter: "corp_code",
                ..
            })
        ));
        let missing_mutant = source_adapter(
            "/api/accnutAdtorNmNdAdtOpinion.json",
            OperationIdentity::new("get_accnutAdtorNmNdAdtOpinion_json", "DS002-2020009"),
            "corp_code",
            "",
            JSON,
            None,
            decode_identity,
        );
        killed(
            FailureDimension::Requiredness,
            missing.prepare_json().is_err(),
            !missing_mutant.encoded_query().is_empty(),
        );

        let invalid = AccountingAuditorNameAndAuditOpinionInput::new(
            "00126380".to_owned(),
            "2025".to_owned(),
            "99999".to_owned(),
        );
        assert!(matches!(
            invalid.prepare_json(),
            Err(PrepareError::InvalidAllowedValue {
                parameter: "reprt_code",
                ..
            })
        ));
        let invalid_mutant = source_adapter(
            "/api/accnutAdtorNmNdAdtOpinion.json",
            OperationIdentity::new("get_accnutAdtorNmNdAdtOpinion_json", "DS002-2020009"),
            "reprt_code",
            "99999",
            JSON,
            None,
            decode_identity,
        );
        killed(
            FailureDimension::AllowedValue,
            invalid.prepare_json().is_err(),
            invalid_mutant.encoded_query() == "reprt_code=99999",
        );
    }

    #[test]
    fn operation_local_constraints_are_bound_to_each_public_input() {
        let invalid_scope = CompanyFinancialStatementsInput::new(
            "00126380".to_owned(),
            "2025".to_owned(),
            "11011".to_owned(),
            "INVALID".to_owned(),
        )
        .prepare_json()
        .unwrap_err();
        assert!(matches!(
            invalid_scope,
            PrepareError::InvalidAllowedValue {
                operation,
                parameter: "fs_div",
                ..
            } if operation.physical() == "get_fnlttSinglAcntAll_json"
                && operation.logical() == "DS003-2019020"
        ));

        let invalid_company_indicator = CompanyFinancialIndicatorsInput::new(
            "00126380".to_owned(),
            "2025".to_owned(),
            "11011".to_owned(),
            "INVALID".to_owned(),
        )
        .prepare_json()
        .unwrap_err();
        assert!(matches!(
            invalid_company_indicator,
            PrepareError::InvalidAllowedValue {
                operation,
                parameter: "idx_cl_code",
                ..
            } if operation.physical() == "get_fnlttSinglIndx_json"
                && operation.logical() == "DS003-2022001"
        ));

        let invalid_companies_indicator = CompaniesFinancialIndicatorsInput::new(
            vec!["00126380".to_owned()],
            "2025".to_owned(),
            "11011".to_owned(),
            "INVALID".to_owned(),
        )
        .prepare_json()
        .unwrap_err();
        assert!(matches!(
            invalid_companies_indicator,
            PrepareError::InvalidAllowedValue {
                operation,
                parameter: "idx_cl_code",
                ..
            } if operation.physical() == "get_fnlttCmpnyIndx_json"
                && operation.logical() == "DS003-2022002"
        ));

        let optional_allowed_errors = [
            (
                DisclosureSearchInput::new()
                    .with_final_reports_only("invalid".to_owned())
                    .prepare_json()
                    .unwrap_err(),
                "last_reprt_at",
            ),
            (
                DisclosureSearchInput::new()
                    .with_disclosure_type("invalid".to_owned())
                    .prepare_json()
                    .unwrap_err(),
                "pblntf_ty",
            ),
            (
                DisclosureSearchInput::new()
                    .with_company_class("invalid".to_owned())
                    .prepare_json()
                    .unwrap_err(),
                "corp_cls",
            ),
            (
                DisclosureSearchInput::new()
                    .with_sort_by("invalid".to_owned())
                    .prepare_json()
                    .unwrap_err(),
                "sort",
            ),
            (
                DisclosureSearchInput::new()
                    .with_sort_order("invalid".to_owned())
                    .prepare_json()
                    .unwrap_err(),
                "sort_mth",
            ),
        ];
        for (error, expected_parameter) in optional_allowed_errors {
            assert!(matches!(
                error,
                PrepareError::InvalidAllowedValue {
                    operation,
                    parameter,
                    ..
                } if operation.physical() == "get_list_json"
                    && operation.logical() == "DS001-2019001"
                    && parameter == expected_parameter
            ));
        }

        for (error, expected_parameter) in [
            (
                DisclosureSearchInput::new()
                    .with_page("0".to_owned())
                    .prepare_json()
                    .unwrap_err(),
                "page_no",
            ),
            (
                DisclosureSearchInput::new()
                    .with_page_size("101".to_owned())
                    .prepare_json()
                    .unwrap_err(),
                "page_count",
            ),
        ] {
            assert!(matches!(
                error,
                PrepareError::InvalidDecimalRange {
                    operation,
                    parameter,
                    ..
                } if operation.physical() == "get_list_json"
                    && operation.logical() == "DS001-2019001"
                    && parameter == expected_parameter
            ));
        }

        let detail = DisclosureSearchInput::new()
            .with_disclosure_detail_type("A001".to_owned())
            .prepare_json()
            .expect("source-defined disclosure detail remains accepted");
        assert_eq!(detail.encoded_query(), "pblntf_detail_ty=A001");
    }

    #[test]
    fn response_fault_adapters_fail_at_media_root_shape_and_retention() {
        let input = CompanyOverviewInput::new("00126380".to_owned());
        let json = input.prepare_json().unwrap();
        let xml = input.prepare_xml().unwrap();
        let inspector = WireInspector::new(2048).unwrap();
        let key = ApiKey::new("fixture-key").unwrap();

        let faulty_media = source_adapter(
            "/api/company.json",
            OperationIdentity::new("get_company_json", "DS001-2019002"),
            "corp_code",
            "00126380",
            XML,
            Some("result"),
            decode_identity,
        );
        let xml_body = br#"<result><status>000</status><corp_name>Example</corp_name></result>"#;
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
        assert!(matches!(
            xml.interpret_response(&inspector, &key, 200, wrong_root_body),
            Err(ResponseInterpretError::Decode {
                source: ResponseDecodeError::UnexpectedXmlRoot { expected: "result" },
                ..
            })
        ));
        let faulty_root = source_adapter(
            "/api/company.xml",
            OperationIdentity::new("get_company_xml", "DS001-2019002"),
            "corp_code",
            "00126380",
            XML,
            Some("wrong"),
            decode_identity,
        );
        killed(
            FailureDimension::XmlRoot,
            xml.interpret_response(&inspector, &key, 200, wrong_root_body)
                .is_err(),
            faulty_root
                .interpret_response(&inspector, &key, 200, wrong_root_body)
                .is_ok(),
        );

        let wrong_shape_body = br#"{"status":"000","list":{}}"#;
        let search = DisclosureSearchInput::new().prepare_json().unwrap();
        let faulty_shape = source_adapter(
            "/api/list.json",
            OperationIdentity::new("get_list_json", "DS001-2019001"),
            "corp_code",
            "00126380",
            JSON,
            None,
            decode_identity,
        );
        killed(
            FailureDimension::ResponseFieldShape,
            search
                .interpret_response(&inspector, &key, 200, wrong_shape_body)
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
        let faulty_retention = source_adapter(
            "/api/company.json",
            OperationIdentity::new("get_company_json", "DS001-2019002"),
            "corp_code",
            "00126380",
            JSON,
            None,
            decode_lossy,
        );
        let SourceReply::Success(faulty_profile) = faulty_retention
            .interpret_response(&inspector, &key, 200, retained_body)
            .unwrap()
        else {
            panic!("faulty decoder should still construct success evidence");
        };
        killed(
            FailureDimension::SourceRetention,
            profile.source().get("future").is_some(),
            faulty_profile.get("future").is_none(),
        );
    }

    #[cfg(all(
        opendart_compat,
        feature = "client-reqwest",
        not(target_family = "wasm")
    ))]
    #[tokio::test]
    async fn handwritten_zip_streams_exact_bytes_and_routes_xml_status() {
        use std::future::poll_fn;

        use bytes::Bytes;
        use futures_core::Stream;

        use crate::{BinaryReply, client::classify_binary_fixture};

        let prepared = crate::operations::disclosure::CompanyCodesInput::new()
            .prepare_archive()
            .unwrap();
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
        assert!(
            prepared
                .expected_representations()
                .contains(&Representation::Zip)
        );

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
