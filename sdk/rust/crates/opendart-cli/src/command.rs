use clap::{Arg, Command, error::ErrorKind};

pub(crate) enum ParseOutcome {
    Matches(clap::ArgMatches),
    PlainText(clap::Error),
    Usage(Vec<String>),
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
        Err(_) => ParseOutcome::Usage(usage_help(&arguments)),
    }
}

fn usage_help(arguments: &[std::ffi::OsString]) -> Vec<String> {
    let operation = arguments
        .get(1)
        .filter(|value| value == &&std::ffi::OsString::from("call"))
        .and_then(|_| arguments.get(2))
        .and_then(|value| value.to_str())
        .and_then(crate::generated::catalog::operation);
    let Some(operation) = operation else {
        return vec!["Valid commands: operations, call, --help, --version".to_owned()];
    };

    let mut flags: Vec<&str> = operation.flags.iter().map(|flag| flag.name).collect();
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
    flags.extend([
        "--connect-timeout-ms",
        "--read-timeout-ms",
        "--total-timeout-ms",
        "--envelope-limit-bytes",
        "--help",
        "--version",
    ]);
    vec![format!("Valid flags: {}", flags.join(", "))]
}

pub(crate) fn execution_arguments(command: Command, binary: bool) -> Command {
    let mut command = command
        .arg(positive_integer("connect-timeout-ms"))
        .arg(positive_integer("read-timeout-ms"))
        .arg(positive_integer("total-timeout-ms"))
        .arg(positive_integer("envelope-limit-bytes"));
    if binary {
        command = command.arg(positive_integer("artifact-limit-bytes"));
    }
    command
}

fn positive_integer(name: &'static str) -> Arg {
    Arg::new(name)
        .long(name)
        .num_args(1)
        .value_parser(clap::value_parser!(u64).range(1..))
}
