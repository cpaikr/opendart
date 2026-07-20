use serde::Serialize;

#[derive(Serialize)]
pub(crate) struct ErrorEnvelope {
    kind: &'static str,
    operation: Option<OperationIdentity>,
    error: ErrorBody,
}

#[derive(Serialize)]
struct OperationIdentity {
    name: &'static str,
    logical_id: &'static str,
    physical_id: &'static str,
    representation: &'static str,
}

#[derive(Serialize)]
struct ErrorBody {
    code: &'static str,
    message: &'static str,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    help: Vec<String>,
}

impl ErrorEnvelope {
    pub(crate) fn usage(help: Vec<String>) -> Self {
        Self {
            kind: "error",
            operation: None,
            error: ErrorBody {
                code: "invalid_invocation",
                message: "the command invocation is invalid",
                help,
            },
        }
    }

    pub(crate) fn invalid_request() -> Self {
        Self {
            kind: "error",
            operation: None,
            error: ErrorBody {
                code: "invalid_request",
                message: "the operation inputs cannot prepare a valid SDK request",
                help: vec![
                    "Inspect operations describe and retry with valid input values".to_owned(),
                ],
            },
        }
    }

    pub(crate) fn missing_api_key() -> Self {
        Self {
            kind: "error",
            operation: None,
            error: ErrorBody {
                code: "missing_api_key",
                message: "OPENDART_API_KEY is required",
                help: vec!["Set OPENDART_API_KEY and retry the same command".to_owned()],
            },
        }
    }

    pub(crate) fn client_initialization() -> Self {
        Self {
            kind: "error",
            operation: None,
            error: ErrorBody {
                code: "client_initialization",
                message: "the OpenDART client could not be initialized",
                help: Vec::new(),
            },
        }
    }
}
