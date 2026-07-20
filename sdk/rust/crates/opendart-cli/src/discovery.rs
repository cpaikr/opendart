use std::path::Path;

use serde::Serialize;

pub(crate) const GLOBAL_FLAGS: &[GlobalFlag] = &[
    GlobalFlag::positive_integer("--connect-timeout-ms", None),
    GlobalFlag::positive_integer("--read-timeout-ms", None),
    GlobalFlag::positive_integer("--total-timeout-ms", None),
    GlobalFlag::positive_integer("--envelope-limit-bytes", None),
    GlobalFlag::positive_integer("--artifact-limit-bytes", Some(536_870_912)),
];

#[derive(Clone, Copy, Serialize)]
pub(crate) struct GlobalFlag {
    pub(crate) name: &'static str,
    required: bool,
    shape: &'static str,
    #[serde(skip_serializing_if = "Option::is_none")]
    default: Option<u64>,
}

impl GlobalFlag {
    const fn positive_integer(name: &'static str, default: Option<u64>) -> Self {
        Self {
            name,
            required: false,
            shape: "positive_integer",
            default,
        }
    }
}

#[derive(Clone, Copy, Serialize)]
pub(crate) struct OperationSpec {
    pub(crate) name: &'static str,
    pub(crate) logical_id: &'static str,
    pub(crate) group: &'static str,
    pub(crate) api_id: &'static str,
    pub(crate) guide_url: &'static str,
    pub(crate) description: &'static str,
    pub(crate) invocation: InvocationSpec,
    pub(crate) global_flags: &'static [GlobalFlag],
    pub(crate) flags: &'static [FlagSpec],
    pub(crate) representations: &'static [RepresentationSpec],
}

#[derive(Clone, Copy, Serialize)]
pub(crate) struct InvocationSpec {
    pub(crate) argv_prefix: &'static [&'static str],
    pub(crate) required_env: &'static [&'static str],
}

#[derive(Clone, Copy, Serialize)]
pub(crate) struct FlagSpec {
    pub(crate) name: &'static str,
    #[serde(skip)]
    pub(crate) id: &'static str,
    pub(crate) sdk_field: &'static str,
    pub(crate) description: &'static str,
    pub(crate) required: bool,
    pub(crate) value_kind: &'static str,
    pub(crate) occurrence: &'static str,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) min_items: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) max_items: Option<i64>,
}

#[derive(Clone, Copy, Serialize)]
pub(crate) struct RepresentationSpec {
    pub(crate) name: &'static str,
    pub(crate) physical_id: &'static str,
    pub(crate) response_type: &'static str,
    pub(crate) response_shape: ResponseShape,
    pub(crate) selector_argv: &'static [&'static str],
    pub(crate) output: OutputSpec,
}

#[derive(Clone, Copy, Serialize)]
#[serde(tag = "kind", rename_all = "snake_case")]
pub(crate) enum OutputSpec {
    Stdout,
    Artifact {
        argument_argv: &'static [&'static str],
        required: bool,
        existing_destination: &'static str,
        limit_argument_argv: &'static [&'static str],
        limit_required: bool,
        default_limit_bytes: u64,
    },
}

#[derive(Clone, Copy, Serialize)]
#[serde(tag = "kind", rename_all = "snake_case")]
pub(crate) enum ResponseShape {
    Object {
        additional_fields: bool,
        fields: &'static [ResponseField],
    },
    Array {
        items: &'static ResponseShape,
    },
    SourceValue,
    SourceStatus,
    Binary,
}

#[derive(Clone, Copy, Serialize)]
pub(crate) struct ResponseField {
    pub(crate) name: &'static str,
    pub(crate) required: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) description: Option<&'static str>,
    pub(crate) shape: ResponseShape,
}

#[derive(Serialize)]
pub(crate) struct Home<'a> {
    kind: &'static str,
    executable: Executable<'a>,
    description: &'static str,
    commands: Commands,
    authentication: Authentication,
    global_flags: &'static [GlobalFlag],
}

#[derive(Serialize)]
struct Executable<'a> {
    path: &'a Path,
    display: String,
}

#[derive(Serialize)]
struct Commands {
    list: CommandPrefix,
    describe: CommandPrefix,
    call: CommandPrefix,
}

#[derive(Serialize)]
struct CommandPrefix {
    argv: &'static [&'static str],
}

#[derive(Serialize)]
struct Authentication {
    environment: &'static str,
    required_for: &'static [&'static str],
}

impl<'a> Home<'a> {
    pub(crate) fn new(executable: &'a Path, display: String) -> Self {
        Self {
            kind: "home",
            executable: Executable {
                path: executable,
                display,
            },
            description: "Call OpenDART through the typed Rust SDK.",
            commands: Commands {
                list: CommandPrefix {
                    argv: &["operations", "list"],
                },
                describe: CommandPrefix {
                    argv: &["operations", "describe", "<operation>"],
                },
                call: CommandPrefix {
                    argv: &["call", "<operation>"],
                },
            },
            authentication: Authentication {
                environment: "OPENDART_API_KEY",
                required_for: &["call"],
            },
            global_flags: GLOBAL_FLAGS,
        }
    }
}

#[derive(Serialize)]
pub(crate) struct Operations<'a> {
    kind: &'static str,
    operations: Vec<CompactOperation<'a>>,
}

#[derive(Serialize)]
struct CompactOperation<'a> {
    name: &'a str,
    logical_id: &'a str,
    group: &'a str,
    api_id: &'a str,
    representations: Vec<&'a str>,
}

impl<'a> Operations<'a> {
    pub(crate) fn new(operations: &'a [OperationSpec]) -> Self {
        Self {
            kind: "operations",
            operations: operations
                .iter()
                .map(|operation| CompactOperation {
                    name: operation.name,
                    logical_id: operation.logical_id,
                    group: operation.group,
                    api_id: operation.api_id,
                    representations: operation
                        .representations
                        .iter()
                        .map(|item| item.name)
                        .collect(),
                })
                .collect(),
        }
    }
}

#[derive(Serialize)]
pub(crate) struct Operation<'a> {
    kind: &'static str,
    operation: &'a OperationSpec,
}

impl<'a> Operation<'a> {
    pub(crate) const fn new(operation: &'a OperationSpec) -> Self {
        Self {
            kind: "operation",
            operation,
        }
    }
}
