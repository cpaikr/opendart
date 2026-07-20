use std::ffi::OsString;
use std::path::{Path, PathBuf};

use clap::ArgMatches;

use crate::command::ParseOutcome;
use crate::discovery::{Home, Operation, Operations};
use crate::error::ErrorEnvelope;
use crate::generated::catalog;

pub(crate) fn run<I, T>(args: I) -> u8
where
    I: IntoIterator<Item = T>,
    T: Into<OsString> + Clone,
{
    match crate::command::parse(args) {
        ParseOutcome::Matches(matches) => dispatch(&matches),
        ParseOutcome::PlainText(error) => {
            if error.print().is_ok() {
                0
            } else {
                1
            }
        }
        ParseOutcome::Usage(help) => emit(&ErrorEnvelope::usage(help), 2),
    }
}

fn dispatch(matches: &ArgMatches) -> u8 {
    // Compile every generated projection identity into the consumer binary;
    // repository freshness verifies the corresponding generated headers.
    let _generated_identity = (
        crate::generated::GENERATOR_SCHEMA,
        crate::generated::PROJECTION_CHECKSUM,
    );
    match matches.subcommand() {
        None => home(),
        Some(("operations", matches)) => operations(matches),
        Some(("call", matches)) => call(matches),
        _ => emit(&ErrorEnvelope::usage(Vec::new()), 2),
    }
}

fn home() -> u8 {
    let Ok(executable) = std::env::current_exe() else {
        return emit(&ErrorEnvelope::executable_resolution(), 1);
    };
    let display = display_path(&executable);
    emit(&Home::new(&executable, display), 0)
}

fn operations(matches: &ArgMatches) -> u8 {
    match matches.subcommand() {
        Some(("list", _)) => emit(&Operations::new(catalog::OPERATIONS), 0),
        Some(("describe", matches)) => {
            let name = matches
                .get_one::<String>("operation")
                .expect("clap requires operation");
            match catalog::operation(name) {
                Some(operation) => emit(&Operation::new(operation), 0),
                None => emit(
                    &ErrorEnvelope::usage(vec![
                        "Use operations list to choose a canonical name or logical ID".to_owned(),
                    ]),
                    2,
                ),
            }
        }
        _ => emit(&ErrorEnvelope::usage(Vec::new()), 2),
    }
}

fn call(matches: &ArgMatches) -> u8 {
    let operation_name = matches
        .subcommand_name()
        .expect("clap requires generated operation");
    // Work 4 attaches this generated catalog identity to the execution outcome.
    let _operation = catalog::operation(operation_name).expect("clap operation has catalog entry");
    let prepared = match crate::generated::dispatch::prepare_call(matches) {
        Ok(prepared) => prepared,
        Err(_) => return emit(&ErrorEnvelope::invalid_request(), 2),
    };
    // Force the prepared SDK identity through the erased dispatch seam before
    // credential access; Work 4 validates and emits it during execution.
    let _identity = prepared.identity();
    let Some(key) = std::env::var_os("OPENDART_API_KEY") else {
        return emit(&ErrorEnvelope::missing_api_key(), 1);
    };
    if key.is_empty() {
        return emit(&ErrorEnvelope::missing_api_key(), 1);
    }
    let Ok(key) = key.into_string() else {
        return emit(&ErrorEnvelope::invalid_client_configuration(), 1);
    };
    if opendart::ApiKey::new(key).is_err() {
        return emit(&ErrorEnvelope::invalid_client_configuration(), 1);
    }
    emit(&ErrorEnvelope::client_initialization(), 1)
}

fn emit(value: &impl serde::Serialize, success: u8) -> u8 {
    if crate::output::json(value).is_ok() {
        success
    } else {
        1
    }
}

fn display_path(path: &Path) -> String {
    let Some(home) = std::env::var_os("HOME").map(PathBuf::from) else {
        return path.display().to_string();
    };
    match path.strip_prefix(home) {
        Ok(relative) => Path::new("~").join(relative).display().to_string(),
        Err(_) => path.display().to_string(),
    }
}
