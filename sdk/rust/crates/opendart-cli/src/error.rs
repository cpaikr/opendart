use opendart::{ClientBuildError, ClientError, ResponseMetadata, TransportFailureKind};
use serde::Serialize;

use crate::execution::OperationContext;

#[derive(Serialize)]
pub(crate) struct ErrorEnvelope {
    kind: &'static str,
    #[serde(skip_serializing_if = "Option::is_none")]
    operation: Option<OperationContext>,
    #[serde(skip_serializing_if = "Option::is_none")]
    metadata: Option<Box<ResponseMetadata>>,
    error: Box<ErrorBody>,
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
            metadata: None,
            error: Box::new(ErrorBody {
                code: "invalid_invocation",
                message: "the command invocation is invalid",
                help,
            }),
        }
    }

    pub(crate) fn invalid_request() -> Self {
        Self {
            kind: "error",
            operation: None,
            metadata: None,
            error: Box::new(ErrorBody {
                code: "invalid_request",
                message: "the operation inputs cannot prepare a valid SDK request",
                help: vec![
                    "Inspect operations describe and retry with valid input values".to_owned(),
                ],
            }),
        }
    }

    pub(crate) fn missing_api_key() -> Self {
        Self {
            kind: "error",
            operation: None,
            metadata: None,
            error: Box::new(ErrorBody {
                code: "missing_api_key",
                message: "OPENDART_API_KEY is required",
                help: vec!["Set OPENDART_API_KEY and retry the same command".to_owned()],
            }),
        }
    }

    pub(crate) fn invalid_client_configuration() -> Self {
        Self {
            kind: "error",
            operation: None,
            metadata: None,
            error: Box::new(ErrorBody {
                code: "invalid_client_configuration",
                message: "OPENDART_API_KEY is not a valid environment value",
                help: vec!["Set OPENDART_API_KEY to a valid non-empty text value".to_owned()],
            }),
        }
    }

    pub(crate) fn executable_resolution() -> Self {
        Self {
            kind: "error",
            operation: None,
            metadata: None,
            error: Box::new(ErrorBody {
                code: "executable_resolution",
                message: "the current executable path could not be resolved",
                help: Vec::new(),
            }),
        }
    }

    pub(crate) fn invalid_client_setting(operation: OperationContext) -> Self {
        Self::execution(
            operation,
            None,
            "invalid_client_configuration",
            "a client setting cannot be represented safely",
        )
    }

    pub(crate) fn client_initialization_for(operation: OperationContext) -> Self {
        Self::execution(
            operation,
            None,
            "client_initialization",
            "the OpenDART client could not be initialized",
        )
    }

    pub(crate) fn client_build(operation: OperationContext, error: ClientBuildError) -> Self {
        match error {
            ClientBuildError::InvalidConfiguration { .. } => {
                Self::invalid_client_setting(operation)
            }
            ClientBuildError::TransportSetup => Self::client_initialization_for(operation),
            _ => Self::client_initialization_for(operation),
        }
    }

    pub(crate) fn client(operation: OperationContext, error: ClientError) -> Self {
        let metadata = error.metadata().cloned().map(Box::new);
        let (code, message) = match error {
            ClientError::Transport(error) => transport_fields(error.kind()),
            ClientError::BodyLimit { .. } => (
                "body_limit",
                "the OpenDART response exceeded the configured envelope limit",
            ),
            ClientError::Envelope { .. } => (
                "malformed_envelope",
                "the OpenDART response is not a valid supported envelope",
            ),
            ClientError::ResponseDecode { .. } => (
                "response_decode",
                "the OpenDART success response does not match the generated SDK contract",
            ),
            ClientError::Representation { .. } => (
                "sdk_contract_mismatch",
                "the prepared SDK request does not match generated CLI discovery",
            ),
            _ => (
                "sdk_contract_mismatch",
                "the prepared SDK request does not match generated CLI discovery",
            ),
        };
        Self::execution(operation, metadata, code, message)
    }

    pub(crate) fn body_stream(
        operation: OperationContext,
        metadata: ResponseMetadata,
        kind: TransportFailureKind,
    ) -> Self {
        let (code, message) = transport_fields(kind);
        Self::execution(operation, Some(Box::new(metadata)), code, message)
    }

    pub(crate) fn destination_exists(
        operation: OperationContext,
        metadata: Option<ResponseMetadata>,
    ) -> Self {
        Self::execution(
            operation,
            metadata.map(Box::new),
            "destination_exists",
            "the artifact destination already exists",
        )
    }

    pub(crate) fn artifact_limit(operation: OperationContext, metadata: ResponseMetadata) -> Self {
        Self::execution(
            operation,
            Some(Box::new(metadata)),
            "artifact_limit",
            "the binary response exceeded the configured artifact limit",
        )
    }

    pub(crate) fn artifact_io(
        operation: OperationContext,
        metadata: Option<ResponseMetadata>,
    ) -> Self {
        Self::execution(
            operation,
            metadata.map(Box::new),
            "artifact_io",
            "the artifact could not be written or published safely",
        )
    }

    pub(crate) fn output_encode(operation: OperationContext) -> Self {
        Self::execution(
            operation,
            None,
            "output_encode",
            "the structured result could not be encoded safely",
        )
    }

    pub(crate) fn output_encode_with_metadata(
        operation: OperationContext,
        metadata: ResponseMetadata,
    ) -> Self {
        Self::execution(
            operation,
            Some(Box::new(metadata)),
            "output_encode",
            "the structured result could not be encoded safely",
        )
    }

    pub(crate) fn sdk_contract_mismatch(operation: Option<OperationContext>) -> Self {
        Self {
            kind: "error",
            operation,
            metadata: None,
            error: Box::new(ErrorBody {
                code: "sdk_contract_mismatch",
                message: "the prepared SDK request does not match generated CLI discovery",
                help: Vec::new(),
            }),
        }
    }

    fn execution(
        operation: OperationContext,
        metadata: Option<Box<ResponseMetadata>>,
        code: &'static str,
        message: &'static str,
    ) -> Self {
        Self {
            kind: "error",
            operation: Some(operation),
            metadata,
            error: Box::new(ErrorBody {
                code,
                message,
                help: Vec::new(),
            }),
        }
    }
}

fn transport_fields(kind: TransportFailureKind) -> (&'static str, &'static str) {
    match kind {
        TransportFailureKind::Timeout => (
            "transport_timeout",
            "the OpenDART request exceeded a configured deadline",
        ),
        TransportFailureKind::Connection => (
            "transport_connection",
            "the OpenDART connection could not be established or retained",
        ),
        TransportFailureKind::Body => (
            "transport_body",
            "the OpenDART response body could not be received",
        ),
        TransportFailureKind::Protocol => (
            "transport_protocol",
            "the OpenDART HTTP exchange violated the transport contract",
        ),
        TransportFailureKind::Other => ("transport_other", "the OpenDART HTTP exchange failed"),
        _ => ("transport_other", "the OpenDART HTTP exchange failed"),
    }
}

#[cfg(test)]
mod tests {
    use super::ErrorEnvelope;
    use crate::execution::OperationContext;

    #[test]
    fn executable_resolution_has_a_dedicated_stable_code() {
        let encoded = serde_json::to_string(&ErrorEnvelope::executable_resolution()).unwrap();
        assert_eq!(
            encoded,
            r#"{"kind":"error","error":{"code":"executable_resolution","message":"the current executable path could not be resolved"}}"#
        );
    }

    #[test]
    fn invalid_sdk_builder_configuration_has_the_stable_cli_code() {
        let operation =
            OperationContext::new("company", "DS001-2019002", "get_company_json", "json");
        let error = ErrorEnvelope::client_build(
            operation,
            opendart::ClientBuildError::InvalidConfiguration { setting: "fixture" },
        );
        let encoded = serde_json::to_string(&error).unwrap();
        assert!(encoded.contains(r#""code":"invalid_client_configuration""#));
        assert!(!encoded.contains("fixture"));
    }
}
