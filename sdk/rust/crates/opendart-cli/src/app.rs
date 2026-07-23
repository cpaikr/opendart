use std::ffi::OsString;
use std::path::{Path, PathBuf};

use clap::ArgMatches;

use crate::artifact::TargetError;
use crate::command::ParseOutcome;
use crate::discovery::{Home, Operation, Operations};
use crate::error::ErrorEnvelope;
use crate::execution::{ClientOverrides, Executor};
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
        ParseOutcome::Usage(error) => emit(&ErrorEnvelope::invocation(error), 2),
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
        Some(("list", matches)) => emit(
            &Operations::new(
                catalog::OPERATIONS,
                matches.get_one::<String>("query").map(String::as_str),
                matches.get_one::<String>("group").map(String::as_str),
                matches
                    .get_one::<String>("representation")
                    .map(String::as_str),
            ),
            0,
        ),
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
    let operation_spec =
        catalog::operation(operation_name).expect("clap operation has catalog entry");
    let Some(requested_operation) = requested_operation_context(operation_spec, matches) else {
        return emit(&ErrorEnvelope::sdk_contract_mismatch(None), 1);
    };
    let prepared = match crate::generated::dispatch::prepare_call(matches) {
        Ok(prepared) => prepared,
        Err(error) => {
            return emit(
                &ErrorEnvelope::invalid_request(requested_operation, operation_spec, error),
                2,
            );
        }
    };
    let operation = match prepared.operation_context() {
        Ok(operation) => operation,
        Err(error) => return emit(&error, 1),
    };
    if operation != requested_operation {
        return emit(
            &ErrorEnvelope::sdk_contract_mismatch(Some(requested_operation)),
            1,
        );
    }
    let artifact = match prepared.artifact_target(matches, operation) {
        Ok(artifact) => artifact,
        Err(TargetError::Usage(error)) => return emit(&error, 2),
        Err(TargetError::Execution(error)) => return emit(&error, 1),
    };
    let overrides = ClientOverrides::from_matches(matches);
    let Some(key) = std::env::var_os("OPENDART_API_KEY") else {
        return emit(&ErrorEnvelope::missing_api_key(operation), 1);
    };
    if key.is_empty() {
        return emit(&ErrorEnvelope::missing_api_key(operation), 1);
    }
    let Ok(key) = key.into_string() else {
        return emit(
            &ErrorEnvelope::invalid_client_configuration(operation, "non_text_api_key"),
            1,
        );
    };
    let key = match opendart::ApiKey::new(key) {
        Ok(key) => key,
        Err(opendart::AuthorizationError::EmptyApiKey) => {
            return emit(
                &ErrorEnvelope::invalid_client_configuration(operation, "whitespace_only_api_key"),
                1,
            );
        }
        Err(opendart::AuthorizationError::ControlCharacterApiKey) => {
            return emit(
                &ErrorEnvelope::invalid_client_configuration(
                    operation,
                    "control_character_api_key",
                ),
                1,
            );
        }
        Err(_) => {
            return emit(
                &ErrorEnvelope::invalid_client_configuration(operation, "invalid_api_key"),
                1,
            );
        }
    };
    let executor = match Executor::new(key, overrides, operation) {
        Ok(executor) => executor,
        Err(error) => return emit(&error, 1),
    };
    match prepared.execute(&executor, artifact, operation) {
        Ok(output) => match crate::output::write(output.bytes) {
            Ok(()) => output.exit,
            Err(()) => 1,
        },
        Err(error) => emit(&error, 1),
    }
}

fn requested_operation_context(
    operation: &'static crate::discovery::OperationSpec,
    matches: &ArgMatches,
) -> Option<crate::execution::OperationContext> {
    let operation_matches = matches.subcommand().map(|(_, matches)| matches)?;
    let representation = if operation.representations.len() == 1 {
        operation.representations.first()?
    } else {
        let name = operation_matches
            .get_one::<String>("representation")
            .map(String::as_str)?;
        operation
            .representations
            .iter()
            .find(|representation| representation.name == name)?
    };
    Some(crate::execution::OperationContext::new(
        operation.name,
        operation.logical_id,
        representation.physical_id,
        representation.name,
    ))
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
