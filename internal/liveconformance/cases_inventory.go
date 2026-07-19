package liveconformance

import (
	"strconv"
	"strings"
)

type inventoryEntry struct {
	stem       string
	logicalID  AssertionID
	detailType string
	aliases    []string
}

// primaryLogicalCases is deliberately explicit: each row is the reviewed
// public coordinate and semantic-policy identity for one logical operation.
func primaryLogicalCases() []logicalCase {
	annual := parameters("corp_code", samsungCorpCode, "bsns_year", "2024", "reprt_code", "11011")
	eventPlaceholder := parameters("corp_code", samsungCorpCode, "bgn_de", "20250101", "end_de", "20250331")
	cases := []logicalCase{
		{group: "ds001", stem: "list", assertion: "DS001-2019001", parameters: parameters("corp_code", samsungCorpCode, "bgn_de", "20240101", "end_de", "20241231", "page_no", "1", "page_count", "100")},
		{group: "ds001", stem: "company", assertion: "DS001-2019002", parameters: parameters("corp_code", samsungCorpCode)},
	}

	ds002 := []inventoryEntry{
		{stem: "accnutAdtorNmNdAdtOpinion", logicalID: "DS002-2020009"},
		{stem: "accnutAdtorNonAdtServcCnclsSttus", logicalID: "DS002-2020011"},
		{stem: "adtServcCnclsSttus", logicalID: "DS002-2020010"},
		{stem: "alotMatter", logicalID: "DS002-2019005"},
		{stem: "cndlCaplScritsNrdmpBlce", logicalID: "DS002-2020008"},
		{stem: "cprndNrdmpBlce", logicalID: "DS002-2020006"},
		{stem: "detScritsIsuAcmslt", logicalID: "DS002-2020003"},
		{stem: "drctrAdtAllMendngSttusGmtsckConfmAmount", logicalID: "DS002-2020014"},
		{stem: "drctrAdtAllMendngSttusMendngPymntamtTyCl", logicalID: "DS002-2020015"},
		{stem: "empSttus", logicalID: "DS002-2019011"},
		{stem: "entrprsBilScritsNrdmpBlce", logicalID: "DS002-2020004"},
		{stem: "exctvSttus", logicalID: "DS002-2019010"},
		{stem: "hmvAuditAllSttus", logicalID: "DS002-2019013"},
		{stem: "hmvAuditIndvdlBySttus", logicalID: "DS002-2019012"},
		{stem: "hmvAuditIndvdlBySttusV2", logicalID: "DS002-2026001"},
		{stem: "hyslrChgSttus", logicalID: "DS002-2019008"},
		{stem: "hyslrSttus", logicalID: "DS002-2019007"},
		{stem: "indvdlByPay", logicalID: "DS002-2019014"},
		{stem: "indvdlByPayV2", logicalID: "DS002-2026002"},
		{stem: "irdsSttus", logicalID: "DS002-2019004"},
		{stem: "mrhlSttus", logicalID: "DS002-2019009"},
		{stem: "newCaplScritsNrdmpBlce", logicalID: "DS002-2020007"},
		{stem: "otrCprInvstmntSttus", logicalID: "DS002-2019015"},
		{stem: "outcmpnyDrctrNdChangeSttus", logicalID: "DS002-2020012"},
		{stem: "prvsrpCptalUseDtls", logicalID: "DS002-2020017"},
		{stem: "pssrpCptalUseDtls", logicalID: "DS002-2020016"},
		{stem: "srtpdPsndbtNrdmpBlce", logicalID: "DS002-2020005"},
		{stem: "stockTotqySttus", logicalID: "DS002-2020002"},
		{stem: "tesstkAcqsDspsSttus", logicalID: "DS002-2019006"},
		{stem: "unrstExctvMendngSttus", logicalID: "DS002-2020013"},
	}
	for _, entry := range ds002 {
		cases = append(cases, logicalCase{group: "ds002", stem: entry.stem, assertion: entry.logicalID, parameters: cloneParameters(annual)})
	}

	cases = append(cases,
		logicalCase{group: "ds003", stem: "fnlttCmpnyIndx", assertion: "DS003-2022002", parameters: parameters("corp_code", samsungCorpCode, "bsns_year", "2024", "reprt_code", "11011", "idx_cl_code", "M210000")},
		logicalCase{group: "ds003", stem: "fnlttMultiAcnt", assertion: "DS003-2019017", parameters: map[string][]string{"corp_code": {"00334624", samsungCorpCode}, "bsns_year": {"2018"}, "reprt_code": {"11011"}}},
		logicalCase{group: "ds003", stem: "fnlttSinglAcnt", assertion: "DS003-2019016", parameters: parameters("corp_code", samsungCorpCode, "bsns_year", "2018", "reprt_code", "11011")},
		logicalCase{group: "ds003", stem: "fnlttSinglAcntAll", assertion: "DS003-2019020", parameters: parameters("corp_code", samsungCorpCode, "bsns_year", "2018", "reprt_code", "11011", "fs_div", "CFS")},
		logicalCase{group: "ds003", stem: "fnlttSinglIndx", assertion: "DS003-2022001", parameters: parameters("corp_code", samsungCorpCode, "bsns_year", "2024", "reprt_code", "11011", "idx_cl_code", "M210000")},
		logicalCase{group: "ds003", stem: "xbrlTaxonomy", assertion: "DS003-2020001", parameters: parameters("sj_div", "BS1")},
		logicalCase{group: "ds004", stem: "elestock", assertion: "DS004-2019022", parameters: parameters("corp_code", samsungCorpCode)},
		logicalCase{group: "ds004", stem: "majorstock", assertion: "DS004-2019021", parameters: parameters("corp_code", samsungCorpCode)},
	)

	ds005 := []inventoryEntry{
		{stem: "astInhtrfEtcPtbkOpt", logicalID: "DS005-2020018", aliases: []string{"자산양수도(기타)", "풋백옵션", "자산양수도(기타), 풋백옵션"}},
		{stem: "bdwtIsDecsn", logicalID: "DS005-2020034", aliases: []string{"신주인수권부사채권 발행결정"}},
		{stem: "bnkMngtPcbg", logicalID: "DS005-2020027", aliases: []string{"채권은행 등의 관리절차 개시"}},
		{stem: "bnkMngtPcsp", logicalID: "DS005-2020036", aliases: []string{"채권은행 등의 관리절차 중단"}},
		{stem: "bsnInhDecsn", logicalID: "DS005-2020042", aliases: []string{"영업양수 결정"}},
		{stem: "bsnSp", logicalID: "DS005-2020020", aliases: []string{"영업정지"}},
		{stem: "bsnTrfDecsn", logicalID: "DS005-2020043", aliases: []string{"영업양도 결정"}},
		{stem: "cmpDvDecsn", logicalID: "DS005-2020051", aliases: []string{"회사분할 결정"}},
		{stem: "cmpDvmgDecsn", logicalID: "DS005-2020052", aliases: []string{"회사분할합병 결정"}},
		{stem: "cmpMgDecsn", logicalID: "DS005-2020050", aliases: []string{"회사합병 결정"}},
		{stem: "crDecsn", logicalID: "DS005-2020026", aliases: []string{"감자 결정"}},
		{stem: "ctrcvsBgrq", logicalID: "DS005-2020021", aliases: []string{"회생절차 개시신청"}},
		{stem: "cvbdIsDecsn", logicalID: "DS005-2020033", aliases: []string{"전환사채권 발행결정"}},
		{stem: "dfOcr", logicalID: "DS005-2020019", aliases: []string{"부도발생"}},
		{stem: "dsRsOcr", logicalID: "DS005-2020022", aliases: []string{"해산사유 발생"}},
		{stem: "exbdIsDecsn", logicalID: "DS005-2020035", aliases: []string{"교환사채권 발행결정"}},
		{stem: "fricDecsn", logicalID: "DS005-2020024", aliases: []string{"무상증자 결정"}},
		{stem: "lwstLg", logicalID: "DS005-2020028", aliases: []string{"소송 등의 제기"}},
		{stem: "otcprStkInvscrInhDecsn", logicalID: "DS005-2020046", aliases: []string{"타법인 주식 및 출자증권 양수결정"}},
		{stem: "otcprStkInvscrTrfDecsn", logicalID: "DS005-2020047", aliases: []string{"타법인 주식 및 출자증권 양도결정"}},
		{stem: "ovDlst", logicalID: "DS005-2020032", aliases: []string{"해외 증권시장 주권등 상장폐지"}},
		{stem: "ovDlstDecsn", logicalID: "DS005-2020030", aliases: []string{"해외 증권시장 주권등 상장폐지 결정"}},
		{stem: "ovLst", logicalID: "DS005-2020031", aliases: []string{"해외 증권시장 주권등 상장"}},
		{stem: "ovLstDecsn", logicalID: "DS005-2020029", aliases: []string{"해외 증권시장 주권등 상장 결정"}},
		{stem: "pifricDecsn", logicalID: "DS005-2020025", aliases: []string{"유무상증자 결정"}},
		{stem: "piicDecsn", logicalID: "DS005-2020023", aliases: []string{"유상증자 결정"}},
		{stem: "stkExtrDecsn", logicalID: "DS005-2020053", aliases: []string{"주식교환·이전 결정"}},
		{stem: "stkrtbdInhDecsn", logicalID: "DS005-2020048", aliases: []string{"주권 관련 사채권 양수 결정"}},
		{stem: "stkrtbdTrfDecsn", logicalID: "DS005-2020049", aliases: []string{"주권 관련 사채권 양도 결정"}},
		{stem: "tgastInhDecsn", logicalID: "DS005-2020044", aliases: []string{"유형자산 양수 결정"}},
		{stem: "tgastTrfDecsn", logicalID: "DS005-2020045", aliases: []string{"유형자산 양도 결정"}},
		{stem: "tsstkAqDecsn", logicalID: "DS005-2020038", aliases: []string{"자기주식 취득 결정"}},
		{stem: "tsstkAqTrctrCcDecsn", logicalID: "DS005-2020041", aliases: []string{"자기주식취득 신탁계약 해지 결정"}},
		{stem: "tsstkAqTrctrCnsDecsn", logicalID: "DS005-2020040", aliases: []string{"자기주식취득 신탁계약 체결 결정"}},
		{stem: "tsstkDpDecsn", logicalID: "DS005-2020039", aliases: []string{"자기주식 처분 결정"}},
		{stem: "wdCocobdIsDecsn", logicalID: "DS005-2020037", aliases: []string{"상각형 조건부자본증권 발행결정"}},
	}
	for _, entry := range ds005 {
		cases = append(cases, logicalCase{group: "ds005", stem: entry.stem, assertion: entry.logicalID, parameters: cloneParameters(eventPlaceholder), discovery: "rare-disclosures", detailTypes: []string{"B001"}, aliases: entry.aliases})
	}

	ds006 := []inventoryEntry{
		{stem: "bdRs", logicalID: "DS006-2020055", detailType: "C002", aliases: []string{"채무증권", "증권발행실적보고서"}},
		{stem: "dvRs", logicalID: "DS006-2020059", detailType: "C004", aliases: []string{"분할"}},
		{stem: "estkRs", logicalID: "DS006-2020054", detailType: "C001", aliases: []string{"지분증권", "증권발행실적보고서"}},
		{stem: "extrRs", logicalID: "DS006-2020058", detailType: "C004", aliases: []string{"주식의포괄적교환·이전"}},
		{stem: "mgRs", logicalID: "DS006-2020057", detailType: "C004", aliases: []string{"합병"}},
		{stem: "stkdpRs", logicalID: "DS006-2020056", detailType: "C005", aliases: []string{"증권예탁증권"}},
	}
	for _, entry := range ds006 {
		cases = append(cases, logicalCase{group: "ds006", stem: entry.stem, assertion: entry.logicalID, parameters: cloneParameters(eventPlaceholder), discovery: "rare-disclosures", detailTypes: []string{entry.detailType}, aliases: entry.aliases})
	}
	return cases
}

func disclosureDiscoveryRequests() []DiscoveryRequest {
	requests := make([]DiscoveryRequest, 0, 32)
	for month := 1; month <= 12; month++ {
		lastDay := 31
		if month == 2 {
			lastDay = 28
		} else if month == 4 || month == 6 || month == 9 || month == 11 {
			lastDay = 30
		}
		for half, days := range [][2]int{{1, 15}, {16, lastDay}} {
			id := strings.Join([]string{"b001", "2025", "m" + twoDigits(month), "h" + strconv.Itoa(half+1)}, "-")
			requests = append(requests, DiscoveryRequest{ID: id, Parameters: parameters("pblntf_detail_ty", "B001", "bgn_de", "2025"+twoDigits(month)+twoDigits(days[0]), "end_de", "2025"+twoDigits(month)+twoDigits(days[1]), "page_no", "1", "page_count", "100")})
		}
	}
	for _, detailType := range []string{"C001", "C002", "C004", "C005"} {
		for half, days := range [][2]int{{1, 15}, {16, 31}} {
			requests = append(requests, DiscoveryRequest{ID: strings.ToLower(detailType) + "-2025-m01-h" + strconv.Itoa(half+1), Parameters: parameters("pblntf_detail_ty", detailType, "bgn_de", "202501"+twoDigits(days[0]), "end_de", "202501"+twoDigits(days[1]), "page_no", "1", "page_count", "100")})
		}
	}
	return requests
}

func twoDigits(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}
