use opendart::{OperationIdentity, PreparedBinaryRequest, PreparedRequest, Representation};

use clap::ArgMatches;

use crate::artifact::{ArtifactTarget, TargetError};
use crate::discovery::{OperationSpec, OutputSpec, RepresentationSpec};
use crate::execution::{BufferedOutput, Executor, OperationContext};

pub(crate) enum PreparedCall {
    Structured(Box<dyn PreparedOperation>),
    Binary(BinaryOperation),
}

pub(crate) trait PreparedOperation {
    fn identity(&self) -> OperationIdentity;
    fn operation(&self) -> &'static OperationSpec;
    fn expected_representations(&self) -> &'static [Representation];
    fn execute(
        self: Box<Self>,
        executor: &Executor,
    ) -> Result<BufferedOutput, crate::error::ErrorEnvelope>;
}

struct StructuredOperation<T> {
    operation: &'static OperationSpec,
    request: PreparedRequest<T>,
}

impl<T> PreparedOperation for StructuredOperation<T>
where
    T: serde::Serialize + 'static,
{
    fn identity(&self) -> OperationIdentity {
        self.request.identity()
    }

    fn operation(&self) -> &'static OperationSpec {
        self.operation
    }

    fn expected_representations(&self) -> &'static [Representation] {
        self.request.expected_representations()
    }

    fn execute(
        self: Box<Self>,
        executor: &Executor,
    ) -> Result<BufferedOutput, crate::error::ErrorEnvelope> {
        let context = operation_context(
            self.operation,
            self.request.identity(),
            self.request.expected_representations(),
        )?;
        executor.execute(self.request, context)
    }
}

pub(crate) struct BinaryOperation {
    operation: &'static OperationSpec,
    request: PreparedBinaryRequest,
}

impl PreparedCall {
    pub(crate) fn operation_context(
        &self,
    ) -> Result<OperationContext, crate::error::ErrorEnvelope> {
        match self {
            Self::Structured(request) => operation_context(
                request.operation(),
                request.identity(),
                request.expected_representations(),
            ),
            Self::Binary(request) => binary_operation_context(request),
        }
    }

    pub(crate) fn artifact_target(
        &self,
        matches: &ArgMatches,
        operation: OperationContext,
    ) -> Result<Option<ArtifactTarget>, TargetError> {
        match self {
            Self::Structured(_) => Ok(None),
            Self::Binary(prepared) => {
                let representation =
                    binary_representation(prepared).map_err(TargetError::Execution)?;
                let OutputSpec::Artifact {
                    default_limit_bytes,
                    ..
                } = representation.output
                else {
                    return Err(TargetError::Execution(
                        crate::error::ErrorEnvelope::sdk_contract_mismatch(Some(operation)),
                    ));
                };
                ArtifactTarget::from_matches(matches, default_limit_bytes, operation).map(Some)
            }
        }
    }

    pub(crate) fn execute(
        self,
        executor: &Executor,
        artifact: Option<ArtifactTarget>,
        operation: OperationContext,
    ) -> Result<BufferedOutput, crate::error::ErrorEnvelope> {
        match (self, artifact) {
            (Self::Structured(request), None) => request.execute(executor),
            (Self::Binary(request), Some(artifact)) => {
                executor.execute_binary(request.request, operation, artifact)
            }
            _ => Err(crate::error::ErrorEnvelope::sdk_contract_mismatch(Some(
                operation,
            ))),
        }
    }
}

pub(crate) fn structured<T: serde::Serialize + 'static>(
    operation: &'static OperationSpec,
    request: PreparedRequest<T>,
) -> PreparedCall {
    PreparedCall::Structured(Box::new(StructuredOperation { operation, request }))
}

pub(crate) fn binary(
    operation: &'static OperationSpec,
    request: PreparedBinaryRequest,
) -> PreparedCall {
    PreparedCall::Binary(BinaryOperation { operation, request })
}

fn operation_context(
    operation: &'static OperationSpec,
    identity: OperationIdentity,
    expected_representations: &'static [Representation],
) -> Result<OperationContext, crate::error::ErrorEnvelope> {
    if operation.logical_id != identity.logical() {
        return Err(crate::error::ErrorEnvelope::sdk_contract_mismatch(None));
    }
    let mut representations = operation
        .representations
        .iter()
        .filter(|item| item.physical_id == identity.physical());
    let Some(representation) = representations.next() else {
        return Err(crate::error::ErrorEnvelope::sdk_contract_mismatch(None));
    };
    if representations.next().is_some()
        || !matches!(representation.name, "json" | "xml")
        || !matches!(representation.output, OutputSpec::Stdout)
        || match representation.name {
            "json" => expected_representations != [Representation::Json],
            "xml" => expected_representations != [Representation::Xml],
            _ => true,
        }
    {
        return Err(crate::error::ErrorEnvelope::sdk_contract_mismatch(None));
    }
    Ok(OperationContext::new(
        operation.name,
        operation.logical_id,
        representation.physical_id,
        representation.name,
    ))
}

fn binary_operation_context(
    prepared: &BinaryOperation,
) -> Result<OperationContext, crate::error::ErrorEnvelope> {
    let representation = binary_representation(prepared)?;
    Ok(OperationContext::new(
        prepared.operation.name,
        prepared.operation.logical_id,
        representation.physical_id,
        representation.name,
    ))
}

fn binary_representation(
    prepared: &BinaryOperation,
) -> Result<&'static RepresentationSpec, crate::error::ErrorEnvelope> {
    let identity = prepared.request.identity();
    if prepared.operation.logical_id != identity.logical()
        || prepared.request.expected_representations() != [Representation::Zip, Representation::Xml]
    {
        return Err(crate::error::ErrorEnvelope::sdk_contract_mismatch(None));
    }
    let mut representations = prepared
        .operation
        .representations
        .iter()
        .filter(|item| item.physical_id == identity.physical());
    let Some(representation) = representations.next() else {
        return Err(crate::error::ErrorEnvelope::sdk_contract_mismatch(None));
    };
    if representations.next().is_some()
        || representation.name != "zip"
        || !matches!(representation.output, OutputSpec::Artifact { .. })
    {
        return Err(crate::error::ErrorEnvelope::sdk_contract_mismatch(None));
    }
    Ok(representation)
}
