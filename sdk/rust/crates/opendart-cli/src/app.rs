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
    let (encoded, exit) = encode_for_emit(value, success);
    match crate::output::write(encoded) {
        Ok(()) => exit,
        Err(()) => 1,
    }
}

fn encode_for_emit(value: &impl serde::Serialize, success: u8) -> (Vec<u8>, u8) {
    match crate::output::encode(value) {
        Ok(encoded) => (encoded, success),
        Err(()) => (
            crate::output::encode(&ErrorEnvelope::output_encode_global())
                .expect("the static global output-encoding error must serialize"),
            1,
        ),
    }
}

fn display_path(path: &Path) -> String {
    let home = home_path_with(|name| std::env::var_os(name));
    display_path_with_home(path, home.as_deref())
}

fn display_path_with_home(path: &Path, home: Option<&Path>) -> String {
    let Some(home) = home else {
        return path.display().to_string();
    };
    match path.strip_prefix(home) {
        Ok(relative) if relative.as_os_str().is_empty() => "~".to_owned(),
        Ok(relative) => Path::new("~").join(relative).display().to_string(),
        Err(_) => path.display().to_string(),
    }
}

fn home_path_with(mut environment: impl FnMut(&str) -> Option<OsString>) -> Option<PathBuf> {
    #[cfg(windows)]
    {
        usable_home(environment("USERPROFILE")).or_else(|| {
            let drive = environment("HOMEDRIVE")?;
            let path = environment("HOMEPATH")?;
            if drive.is_empty() || path.is_empty() {
                return None;
            }
            let mut combined = PathBuf::from(drive);
            combined.push(path);
            usable_home(Some(combined.into_os_string()))
        })
    }
    #[cfg(not(windows))]
    {
        usable_home(environment("HOME"))
    }
}

fn usable_home(value: Option<OsString>) -> Option<PathBuf> {
    let path = PathBuf::from(value?);
    (!path.as_os_str().is_empty() && path.is_absolute()).then_some(path)
}

#[cfg(test)]
mod tests {
    use std::ffi::OsString;
    use std::path::{Path, PathBuf};

    use serde::Serialize;

    struct FailsToSerialize;

    impl Serialize for FailsToSerialize {
        fn serialize<S>(&self, _serializer: S) -> Result<S::Ok, S::Error>
        where
            S: serde::Serializer,
        {
            Err(serde::ser::Error::custom("fixture failure"))
        }
    }

    #[test]
    fn emit_encoding_failure_becomes_one_static_global_error() {
        let (encoded, exit) = super::encode_for_emit(&FailsToSerialize, 0);
        assert_eq!(exit, 1);
        assert_eq!(
            encoded,
            b"{\"kind\":\"error\",\"error\":{\"code\":\"output_encode\",\"message\":\"the structured result could not be encoded safely\"}}\n"
        );
    }

    #[test]
    fn display_path_collapses_only_true_home_prefixes() {
        let root = PathBuf::from(std::path::MAIN_SEPARATOR.to_string());
        let home = root.join("home").join("person");
        let executable = home.join("bin").join("opendart");
        assert_eq!(
            super::display_path_with_home(&executable, Some(&home)),
            Path::new("~")
                .join("bin")
                .join("opendart")
                .display()
                .to_string()
        );
        assert_eq!(super::display_path_with_home(&home, Some(&home)), "~");

        let sibling = root.join("home").join("person-other").join("opendart");
        assert_eq!(
            super::display_path_with_home(&sibling, Some(&home)),
            sibling.display().to_string()
        );
        assert_eq!(
            super::display_path_with_home(&executable, None),
            executable.display().to_string()
        );
    }

    #[cfg(not(windows))]
    #[test]
    fn unix_home_resolver_rejects_missing_empty_and_relative_values() {
        assert_eq!(super::home_path_with(|_| None), None);
        assert_eq!(
            super::home_path_with(|name| (name == "HOME").then(OsString::new)),
            None
        );
        assert_eq!(
            super::home_path_with(|name| (name == "HOME").then(|| OsString::from("relative"))),
            None
        );
    }

    #[cfg(unix)]
    #[test]
    fn unix_home_resolver_preserves_non_utf8_paths() {
        use std::os::unix::ffi::OsStringExt;

        let home = PathBuf::from(OsString::from_vec(b"/tmp/home-\xff".to_vec()));
        assert_eq!(
            super::home_path_with(|name| {
                (name == "HOME").then(|| home.clone().into_os_string())
            }),
            Some(home)
        );
    }

    #[cfg(windows)]
    #[test]
    fn windows_home_resolver_prefers_userprofile_then_native_parts() {
        use std::os::windows::ffi::OsStringExt;

        assert_eq!(super::home_path_with(|_| None), None);
        assert_eq!(
            super::home_path_with(|name| (name == "USERPROFILE").then(OsString::new)),
            None
        );
        assert_eq!(
            super::home_path_with(|name| {
                (name == "USERPROFILE").then(|| OsString::from("relative"))
            }),
            None
        );

        let profile = PathBuf::from(r"C:\Users\profile");
        assert_eq!(
            super::home_path_with(|name| match name {
                "USERPROFILE" => Some(profile.clone().into_os_string()),
                "HOMEDRIVE" => Some(OsString::from("D:")),
                "HOMEPATH" => Some(OsString::from(r"\Users\fallback")),
                _ => None,
            }),
            Some(profile)
        );
        assert_eq!(
            super::home_path_with(|name| match name {
                "HOMEDRIVE" => Some(OsString::from("D:")),
                "HOMEPATH" => Some(OsString::from(r"\Users\fallback")),
                _ => None,
            }),
            Some(PathBuf::from(r"D:\Users\fallback"))
        );

        let non_utf8 =
            OsString::from_wide(&[u16::from(b'C'), u16::from(b':'), u16::from(b'\\'), 0xd800]);
        let non_utf8 = PathBuf::from(non_utf8);
        assert_eq!(
            super::home_path_with(|name| {
                (name == "USERPROFILE").then(|| non_utf8.clone().into_os_string())
            }),
            Some(non_utf8)
        );
    }
}
