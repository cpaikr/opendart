use crate::{OperationIdentity, PrepareError};

pub(crate) fn require_length(
    operation: OperationIdentity,
    parameter: &'static str,
    value: &str,
    minimum: usize,
    maximum: usize,
) -> Result<(), PrepareError> {
    if !(minimum..=maximum).contains(&value.chars().count()) {
        return Err(PrepareError::InvalidLength {
            operation,
            parameter,
            minimum,
            maximum,
        });
    }
    Ok(())
}

pub(crate) fn require_allowed(
    operation: OperationIdentity,
    parameter: &'static str,
    value: &str,
    allowed: &'static [&'static str],
) -> Result<(), PrepareError> {
    if !allowed.contains(&value) {
        return Err(PrepareError::InvalidAllowedValue {
            operation,
            parameter,
        });
    }
    Ok(())
}

pub(crate) fn require_format(
    operation: OperationIdentity,
    parameter: &'static str,
    value: &str,
    format: &'static str,
) -> Result<(), PrepareError> {
    let valid = match format {
        "opendart-corp-code" => value.len() == 8 && value.bytes().all(|byte| byte.is_ascii_digit()),
        "opendart-year" => value.len() == 4 && value.bytes().all(|byte| byte.is_ascii_digit()),
        "opendart-date" => valid_compact_date(value),
        _ => false,
    };
    if !valid {
        return Err(PrepareError::InvalidFormat {
            operation,
            parameter,
            format,
        });
    }
    Ok(())
}

pub(crate) fn require_decimal_range(
    operation: OperationIdentity,
    parameter: &'static str,
    value: &str,
    minimum: u64,
    maximum: Option<u64>,
) -> Result<(), PrepareError> {
    let parsed = if value.bytes().all(|byte| byte.is_ascii_digit()) {
        value.parse::<u64>().ok()
    } else {
        None
    };
    if parsed.is_none_or(|number| number < minimum || maximum.is_some_and(|limit| number > limit)) {
        return Err(PrepareError::InvalidDecimalRange {
            operation,
            parameter,
            minimum,
            maximum,
        });
    }
    Ok(())
}

fn valid_compact_date(value: &str) -> bool {
    if value.len() != 8 || !value.bytes().all(|byte| byte.is_ascii_digit()) {
        return false;
    }
    let year = value[0..4].parse::<u32>().expect("digits checked");
    if year == 0 {
        return false;
    }
    let month = value[4..6].parse::<u32>().expect("digits checked");
    let day = value[6..8].parse::<u32>().expect("digits checked");
    let maximum_day = match month {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        2 if year % 400 == 0 || year % 4 == 0 && year % 100 != 0 => 29,
        2 => 28,
        _ => return false,
    };
    day != 0 && day <= maximum_day
}

#[cfg(test)]
mod tests {
    use super::*;

    const IDENTITY: OperationIdentity = OperationIdentity::new("physical", "logical");

    #[test]
    fn compact_dates_validate_calendar_days() {
        for valid in ["20240229", "20000229", "20261231"] {
            assert!(require_format(IDENTITY, "date", valid, "opendart-date").is_ok());
        }
        for invalid in [
            "00000229",
            "20230229",
            "19000229",
            "20261301",
            "20260010",
            "20260100",
            "２０２６０１０１",
        ] {
            assert!(matches!(
                require_format(IDENTITY, "date", invalid, "opendart-date"),
                Err(PrepareError::InvalidFormat { .. })
            ));
        }
    }

    #[test]
    fn decimal_ranges_reject_signs_zero_overflow_and_excess() {
        for invalid in ["0", "+1", "-1", "101", "18446744073709551616"] {
            assert!(matches!(
                require_decimal_range(IDENTITY, "page_count", invalid, 1, Some(100)),
                Err(PrepareError::InvalidDecimalRange { .. })
            ));
        }
        assert!(require_decimal_range(IDENTITY, "page_count", "100", 1, Some(100)).is_ok());
    }
}
