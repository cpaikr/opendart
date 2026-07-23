use clap::{
    Arg, Command,
    error::{ContextKind, ContextValue, ErrorKind},
};

pub(crate) struct InvocationError {
    pub(crate) reason: &'static str,
    pub(crate) argument: Option<&'static str>,
    pub(crate) allowed: Vec<String>,
    pub(crate) help: Vec<String>,
}

pub(crate) enum ParseOutcome {
    Matches(clap::ArgMatches),
    PlainText(clap::Error),
    Usage(InvocationError),
}

pub(crate) fn parse<I, T>(args: I) -> ParseOutcome
where
    I: IntoIterator<Item = T>,
    T: Into<std::ffi::OsString> + Clone,
{
    let arguments: Vec<std::ffi::OsString> = args.into_iter().map(Into::into).collect();
    match crate::generated::command::command().try_get_matches_from(arguments.clone()) {
        Ok(matches) => ParseOutcome::Matches(matches),
        Err(error)
            if matches!(
                error.kind(),
                ErrorKind::DisplayHelp | ErrorKind::DisplayVersion
            ) =>
        {
            ParseOutcome::PlainText(error)
        }
        Err(error) => ParseOutcome::Usage(invocation_error(&arguments, &error)),
    }
}

fn invocation_error(arguments: &[std::ffi::OsString], error: &clap::Error) -> InvocationError {
    let operation = requested_operation(arguments);
    let argument = safe_argument(arguments, error, operation);
    let allowed = if argument == Some("--representation") {
        if operation.is_some() {
            operation
                .into_iter()
                .flat_map(|operation| operation.representations)
                .map(|representation| representation.name.to_owned())
                .collect()
        } else if is_operations_list(arguments) {
            ["json", "xml", "zip"].map(str::to_owned).to_vec()
        } else {
            Vec::new()
        }
    } else if matches!(
        error.kind(),
        ErrorKind::InvalidSubcommand
            | ErrorKind::MissingSubcommand
            | ErrorKind::DisplayHelpOnMissingArgumentOrSubcommand
    ) {
        valid_subcommands(arguments, error)
    } else {
        Vec::new()
    };
    InvocationError {
        reason: invocation_reason(arguments, error.kind()),
        argument,
        help: usage_help(arguments, error.kind(), &allowed),
        allowed,
    }
}

fn invocation_reason(arguments: &[std::ffi::OsString], kind: ErrorKind) -> &'static str {
    if unknown_root_command(arguments) {
        return "unknown_command";
    }
    match kind {
        ErrorKind::MissingRequiredArgument => "missing_required_argument",
        ErrorKind::MissingSubcommand | ErrorKind::DisplayHelpOnMissingArgumentOrSubcommand => {
            "missing_subcommand"
        }
        ErrorKind::UnknownArgument => "unknown_argument",
        ErrorKind::InvalidSubcommand => "unknown_command",
        ErrorKind::InvalidValue | ErrorKind::ValueValidation => "invalid_value",
        ErrorKind::ArgumentConflict => "argument_conflict",
        ErrorKind::NoEquals
        | ErrorKind::TooManyValues
        | ErrorKind::TooFewValues
        | ErrorKind::WrongNumberOfValues => "invalid_value_count",
        ErrorKind::InvalidUtf8 => "invalid_utf8",
        _ => "invalid_invocation",
    }
}

fn unknown_root_command(arguments: &[std::ffi::OsString]) -> bool {
    arguments.get(1).is_some_and(|value| {
        value
            .to_str()
            .is_some_and(|value| !value.starts_with('-') && !matches!(value, "operations" | "call"))
    })
}

fn valid_subcommands(arguments: &[std::ffi::OsString], error: &clap::Error) -> Vec<String> {
    let contextual = context_strings(error, ContextKind::ValidSubcommand).unwrap_or_default();
    let invalid = match (error.kind(), error.get(ContextKind::InvalidSubcommand)) {
        (ErrorKind::InvalidSubcommand, Some(ContextValue::String(value))) => Some(value.as_str()),
        (ErrorKind::InvalidSubcommand, _) => return Vec::new(),
        _ => None,
    };
    let root = crate::generated::command::command();
    let mut parent = &root;
    for argument in arguments.iter().skip(1) {
        let Some(argument) = argument.to_str() else {
            return Vec::new();
        };
        if invalid == Some(argument) {
            break;
        }
        let Some(subcommand) = parent.find_subcommand(argument) else {
            return Vec::new();
        };
        parent = subcommand;
    }
    parent
        .get_subcommands()
        .filter(|subcommand| !subcommand.is_hide_set())
        .filter(|subcommand| {
            contextual.is_empty()
                || subcommand
                    .get_name_and_visible_aliases()
                    .into_iter()
                    .any(|name| contextual.iter().any(|value| value == name))
        })
        .map(|subcommand| subcommand.get_name().to_owned())
        .collect()
}

fn context_strings(error: &clap::Error, kind: ContextKind) -> Option<Vec<String>> {
    match error.get(kind) {
        Some(ContextValue::String(value)) => Some(vec![value.clone()]),
        Some(ContextValue::Strings(values)) => Some(values.clone()),
        _ => None,
    }
}

fn requested_operation(
    arguments: &[std::ffi::OsString],
) -> Option<&'static crate::discovery::OperationSpec> {
    arguments
        .get(1)
        .filter(|value| value == &&std::ffi::OsString::from("call"))
        .and_then(|_| arguments.get(2))
        .and_then(|value| value.to_str())
        .and_then(crate::generated::catalog::operation)
}

fn safe_argument(
    arguments: &[std::ffi::OsString],
    error: &clap::Error,
    operation: Option<&'static crate::discovery::OperationSpec>,
) -> Option<&'static str> {
    known_arguments(arguments, operation)
        .into_iter()
        .filter(|argument| match error.get(ContextKind::InvalidArg) {
            Some(ContextValue::String(value)) => safe_context_matches(value, argument),
            Some(ContextValue::Strings(values)) => values
                .iter()
                .any(|value| safe_context_matches(value, argument)),
            _ => false,
        })
        .max_by_key(|argument| argument.len())
}

fn safe_context_matches(context: &str, argument: &str) -> bool {
    context == argument
        || context.strip_prefix(argument).is_some_and(|suffix| {
            suffix
                .chars()
                .next()
                .is_some_and(|character| character.is_whitespace() || character == '=')
        })
}

fn known_arguments(
    command_line: &[std::ffi::OsString],
    operation: Option<&'static crate::discovery::OperationSpec>,
) -> Vec<&'static str> {
    valid_flag_names(command_line, operation)
}

fn valid_flag_names(
    command_line: &[std::ffi::OsString],
    operation: Option<&'static crate::discovery::OperationSpec>,
) -> Vec<&'static str> {
    let mut flags = operation
        .into_iter()
        .flat_map(|operation| operation.flags)
        .map(|flag| flag.name)
        .collect::<Vec<_>>();
    if let Some(operation) = operation {
        if operation
            .representations
            .iter()
            .any(|representation| !representation.selector_argv.is_empty())
        {
            flags.push("--representation");
        }
        if operation
            .representations
            .iter()
            .any(|representation| representation.name == "zip")
        {
            flags.extend(["--output", "--artifact-limit-bytes"]);
        }
        flags.extend(crate::discovery::CALL_FLAGS.iter().map(|flag| flag.name));
    } else if is_operations_list(command_line) {
        flags.extend(["--query", "--group", "--representation"]);
    }
    flags
}

fn is_operations_list(arguments: &[std::ffi::OsString]) -> bool {
    matches!(
        (
            arguments.get(1).and_then(|value| value.to_str()),
            arguments.get(2).and_then(|value| value.to_str())
        ),
        (Some("operations"), Some("list"))
    )
}

fn usage_help(
    arguments: &[std::ffi::OsString],
    kind: ErrorKind,
    allowed: &[String],
) -> Vec<String> {
    if matches!(
        kind,
        ErrorKind::InvalidSubcommand
            | ErrorKind::MissingSubcommand
            | ErrorKind::DisplayHelpOnMissingArgumentOrSubcommand
    ) && !allowed.is_empty()
    {
        return vec![format!(
            "Valid commands: {}, --help, --version",
            allowed.join(", ")
        )];
    }
    let operation = requested_operation(arguments);
    if operation.is_none() && !is_operations_list(arguments) {
        return vec!["Valid commands: operations, call, --help, --version".to_owned()];
    }

    let mut flags = valid_flag_names(arguments, operation);
    flags.extend(["--help", "--version"]);
    vec![format!("Valid flags: {}", flags.join(", "))]
}

pub(crate) fn execution_arguments(command: Command, binary: bool) -> Command {
    let mut command = command;
    for flag in crate::discovery::CALL_FLAGS {
        let name = flag
            .name
            .strip_prefix("--")
            .expect("call flag names use their CLI spelling");
        command = command.arg(positive_integer(name));
    }
    if binary {
        command = command.arg(positive_integer("artifact-limit-bytes"));
    }
    command
}

fn positive_integer(name: &'static str) -> Arg {
    let (value_name, help) = match name {
        "connect-timeout-ms" => ("MILLISECONDS", "Set the connection timeout in milliseconds"),
        "read-timeout-ms" => (
            "MILLISECONDS",
            "Set the response-read timeout in milliseconds",
        ),
        "total-timeout-ms" => (
            "MILLISECONDS",
            "Set the total request timeout in milliseconds",
        ),
        "envelope-limit-bytes" => (
            "BYTES",
            "Set the maximum buffered structured-response size in bytes",
        ),
        "artifact-limit-bytes" => ("BYTES", "Set the maximum binary artifact size in bytes"),
        _ => (
            "POSITIVE_INTEGER",
            "Set a positive integer execution control",
        ),
    };
    Arg::new(name)
        .long(name)
        .value_name(value_name)
        .help(help)
        .num_args(1)
        .value_parser(clap::value_parser!(u64).range(1..))
}
