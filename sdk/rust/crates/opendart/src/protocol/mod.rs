//! Private protocol helpers shared by handwritten operations.

use crate::request::{QueryParameter, QueryValue};
use crate::{OperationIdentity, PrepareError, SourceStatus, SourceValue};

pub(crate) fn source_status(source: &SourceValue) -> Option<SourceStatus> {
    source
        .get("status")
        .and_then(SourceValue::as_str)
        .map(SourceStatus::new)
}

pub(crate) struct Query<'a> {
    operation: OperationIdentity,
    parameters: Vec<QueryParameter<'a>>,
}

impl<'a> Query<'a> {
    pub(crate) fn new(operation: OperationIdentity) -> Self {
        Self {
            operation,
            parameters: Vec::new(),
        }
    }

    pub(crate) fn required(
        &mut self,
        name: &'static str,
        value: &'a str,
    ) -> Result<(), PrepareError> {
        if value.is_empty() {
            return Err(PrepareError::MissingInput {
                operation: self.operation,
                parameter: name,
            });
        }
        self.parameters.push(QueryParameter {
            name,
            value: QueryValue::Scalar(value),
        });
        Ok(())
    }

    pub(crate) fn optional(
        &mut self,
        name: &'static str,
        value: Option<&'a str>,
    ) -> Result<(), PrepareError> {
        if let Some(value) = value {
            self.required(name, value)?;
        }
        Ok(())
    }

    pub(crate) fn required_list(
        &mut self,
        name: &'static str,
        values: &'a [String],
        minimum: usize,
        maximum: usize,
    ) -> Result<(), PrepareError> {
        if !(minimum..=maximum).contains(&values.len()) {
            return Err(PrepareError::InvalidCardinality {
                operation: self.operation,
                parameter: name,
                minimum,
                maximum,
            });
        }
        if values.iter().any(String::is_empty) {
            return Err(PrepareError::MissingInput {
                operation: self.operation,
                parameter: name,
            });
        }
        self.parameters.push(QueryParameter {
            name,
            value: QueryValue::CommaSeparated(values),
        });
        Ok(())
    }

    pub(crate) fn finish(self) -> Vec<QueryParameter<'a>> {
        self.parameters
    }
}

pub(crate) enum Items<'a> {
    Empty,
    One(Option<&'a SourceValue>),
    Many(std::slice::Iter<'a, SourceValue>),
}

impl<'a> Iterator for Items<'a> {
    type Item = &'a SourceValue;

    fn next(&mut self) -> Option<Self::Item> {
        match self {
            Self::Empty => None,
            Self::One(value) => value.take(),
            Self::Many(values) => values.next(),
        }
    }
}

pub(crate) fn items<'a>(source: &'a SourceValue, name: &str, xml: bool) -> Items<'a> {
    let Some(value) = source.get(name) else {
        return Items::Empty;
    };
    if let Some(values) = value.as_array() {
        return Items::Many(values.iter());
    }
    if xml {
        Items::One(Some(value))
    } else {
        Items::Empty
    }
}
