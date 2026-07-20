use opendart::{OperationIdentity, PreparedBinaryRequest, PreparedRequest};

pub(crate) enum PreparedCall {
    Structured(Box<dyn PreparedOperation>),
    Binary(PreparedBinaryRequest),
}

pub(crate) trait PreparedOperation {
    fn identity(&self) -> OperationIdentity;
}

impl<T> PreparedOperation for PreparedRequest<T> {
    fn identity(&self) -> OperationIdentity {
        self.identity()
    }
}

impl PreparedCall {
    pub(crate) fn identity(&self) -> OperationIdentity {
        match self {
            Self::Structured(request) => request.identity(),
            Self::Binary(request) => request.identity(),
        }
    }
}

pub(crate) fn structured<T: 'static>(
    _operation: &'static crate::discovery::OperationSpec,
    request: PreparedRequest<T>,
) -> PreparedCall {
    PreparedCall::Structured(Box::new(request))
}

pub(crate) fn binary(
    _operation: &'static crate::discovery::OperationSpec,
    request: PreparedBinaryRequest,
) -> PreparedCall {
    PreparedCall::Binary(request)
}
