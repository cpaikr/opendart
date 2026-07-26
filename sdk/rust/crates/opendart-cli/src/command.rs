use clap::{
    Arg, Command,
    error::{ContextKind, ContextValue, ErrorKind},
};
use std::ffi::{OsStr, OsString};

use crate::discovery::OperationSpec;

#[derive(Clone, Copy)]
enum CommandContext {
    Root,
    Operations,
    OperationsList,
    OperationsDescribe,
    Call,
    Operation(&'static OperationSpec),
}

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
    let arguments: Vec<OsString> = args.into_iter().map(Into::into).collect();
    let arguments = normalize_arguments(arguments);
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

fn invocation_error(arguments: &[OsString], error: &clap::Error) -> InvocationError {
    let context = command_context(arguments);
    let operation = operation(context);
    let argument = safe_argument(context, error, operation);
    let allowed = if argument == Some("--representation") {
        if let Some(operation) = operation {
            operation
                .representations
                .iter()
                .map(|representation| representation.name.to_owned())
                .collect()
        } else if matches!(context, CommandContext::OperationsList) {
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
        valid_subcommands(context)
    } else {
        Vec::new()
    };
    InvocationError {
        reason: invocation_reason(arguments, context, error.kind()),
        argument,
        help: usage_help(context, error.kind(), &allowed),
        allowed,
    }
}

fn invocation_reason(
    arguments: &[OsString],
    context: CommandContext,
    kind: ErrorKind,
) -> &'static str {
    if matches!(context, CommandContext::Root) && unknown_root_command(arguments) {
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

fn unknown_root_command(arguments: &[OsString]) -> bool {
    arguments.get(1).is_some_and(|value| {
        value
            .to_str()
            .is_some_and(|value| !value.starts_with('-') && !matches!(value, "operations" | "call"))
    })
}

fn command_context(arguments: &[OsString]) -> CommandContext {
    let root = crate::generated::command::command();
    let mut command = &root;
    let mut context = CommandContext::Root;
    for argument in arguments.iter().skip(1) {
        let Some(argument) = argument.to_str() else {
            break;
        };
        if argument.starts_with('-') {
            break;
        }
        let Some(subcommand) = command.find_subcommand(argument) else {
            break;
        };
        context = match context {
            CommandContext::Root if subcommand.get_name() == "operations" => {
                CommandContext::Operations
            }
            CommandContext::Root if subcommand.get_name() == "call" => CommandContext::Call,
            CommandContext::Operations if subcommand.get_name() == "list" => {
                CommandContext::OperationsList
            }
            CommandContext::Operations if subcommand.get_name() == "describe" => {
                CommandContext::OperationsDescribe
            }
            CommandContext::Call => crate::generated::catalog::operation(subcommand.get_name())
                .map(CommandContext::Operation)
                .unwrap_or(CommandContext::Call),
            _ => break,
        };
        command = subcommand;
    }
    context
}

fn valid_subcommands(context: CommandContext) -> Vec<String> {
    match context {
        CommandContext::Root => ["operations", "call"].map(str::to_owned).to_vec(),
        CommandContext::Operations => ["list", "describe"].map(str::to_owned).to_vec(),
        CommandContext::Call => crate::generated::catalog::OPERATIONS
            .iter()
            .map(|operation| operation.name.to_owned())
            .collect(),
        _ => Vec::new(),
    }
}

fn operation(context: CommandContext) -> Option<&'static OperationSpec> {
    match context {
        CommandContext::Operation(operation) => Some(operation),
        _ => None,
    }
}

fn safe_argument(
    context: CommandContext,
    error: &clap::Error,
    operation: Option<&'static crate::discovery::OperationSpec>,
) -> Option<&'static str> {
    known_arguments(context, operation)
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
    context: CommandContext,
    operation: Option<&'static OperationSpec>,
) -> Vec<&'static str> {
    valid_flag_names(context, operation)
}

fn valid_flag_names(
    context: CommandContext,
    operation: Option<&'static OperationSpec>,
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
    } else if matches!(context, CommandContext::OperationsList) {
        flags.extend(["--query", "--group", "--representation"]);
    }
    flags
}

fn usage_help(context: CommandContext, kind: ErrorKind, allowed: &[String]) -> Vec<String> {
    if matches!(
        kind,
        ErrorKind::InvalidSubcommand
            | ErrorKind::MissingSubcommand
            | ErrorKind::DisplayHelpOnMissingArgumentOrSubcommand
    ) && !allowed.is_empty()
    {
        if matches!(context, CommandContext::Call) {
            return vec![
                "Choose an operation from error.allowed or operations list, then retry opendart call <OPERATION> [OPTIONS]".to_owned(),
            ];
        }
        return vec![format!(
            "Valid commands: {}, --help, --version",
            allowed.join(", ")
        )];
    }
    match context {
        CommandContext::Root => {
            return vec!["Valid commands: operations, call, --help, --version".to_owned()];
        }
        CommandContext::Operations => {
            return vec!["Valid commands: list, describe, --help, --version".to_owned()];
        }
        CommandContext::OperationsDescribe => {
            return vec![
                "Usage: opendart operations describe <OPERATION>; choose a canonical name or logical ID with operations list".to_owned(),
            ];
        }
        CommandContext::Call => {
            return vec![
                "Choose an operation with operations list, then retry opendart call <OPERATION> [OPTIONS]".to_owned(),
            ];
        }
        CommandContext::OperationsList | CommandContext::Operation(_) => {}
    }

    let mut flags = valid_flag_names(context, operation(context));
    flags.extend(["--help", "--version"]);
    vec![format!("Valid flags: {}", flags.join(", "))]
}

fn normalize_arguments(mut arguments: Vec<OsString>) -> Vec<OsString> {
    let context = command_context(&arguments);
    let candidate = match context {
        CommandContext::OperationsList => Some(("--query", false)),
        CommandContext::Operation(operation)
            if operation
                .representations
                .iter()
                .any(|representation| representation.name == "zip") =>
        {
            Some(("--output", true))
        }
        _ => None,
    };
    let Some((candidate, reject_dash)) = candidate else {
        return arguments;
    };
    let mut recognized = valid_flag_names(context, operation(context));
    recognized.extend(["--help", "--version", "-h", "-V", "--"]);

    let mut index = 1;
    while index + 1 < arguments.len() {
        if arguments[index] == OsStr::new("--") {
            break;
        }
        if arguments[index] != OsStr::new(candidate) {
            index += 1;
            continue;
        }
        let value = &arguments[index + 1];
        if !value.as_encoded_bytes().starts_with(b"-")
            || reject_dash && value == OsStr::new("-")
            || recognized_option(value, &recognized)
        {
            index += 1;
            continue;
        }
        let mut joined = OsString::from(candidate);
        joined.push("=");
        joined.push(value);
        arguments[index] = joined;
        arguments.remove(index + 1);
        index += 1;
    }
    arguments
}

fn recognized_option(value: &OsStr, recognized: &[&str]) -> bool {
    let Some(value) = value.to_str() else {
        return false;
    };
    let name = value.split_once('=').map_or(value, |(name, _)| name);
    recognized.contains(&name)
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
