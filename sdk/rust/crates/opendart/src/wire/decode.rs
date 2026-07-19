use std::collections::BTreeMap;

use super::{ResponseDecodeError, SourceStatus, SourceValue, SourceValueKind, SourceValueRepr};

pub(crate) type Decoder<T> = fn(SourceValue, String) -> Result<T, ResponseDecodeError>;

pub(crate) struct ObjectDecoder {
    fields: BTreeMap<String, SourceValue>,
    path: String,
}

impl ObjectDecoder {
    pub(crate) fn new(value: SourceValue, path: String) -> Result<Self, ResponseDecodeError> {
        let actual = value.kind();
        let SourceValueRepr::Object(fields) = value.0 else {
            return Err(ResponseDecodeError::WrongKind {
                path,
                expected: SourceValueKind::Object,
                actual,
            });
        };
        Ok(Self { fields, path })
    }

    // Canonical inputs currently mark response fields optional, but generated
    // decoders use this as soon as a future contract establishes a required field.
    #[allow(dead_code)]
    pub(crate) fn required<T>(
        &mut self,
        name: &'static str,
        decoder: Decoder<T>,
    ) -> Result<T, ResponseDecodeError> {
        let path = child_path(&self.path, name);
        let value = self
            .fields
            .remove(name)
            .ok_or_else(|| ResponseDecodeError::MissingRequired { path: path.clone() })?;
        decoder(value, path)
    }

    pub(crate) fn optional<T>(
        &mut self,
        name: &'static str,
        decoder: Decoder<T>,
    ) -> Result<Option<T>, ResponseDecodeError> {
        let Some(value) = self.fields.remove(name) else {
            return Ok(None);
        };
        decoder(value, child_path(&self.path, name)).map(Some)
    }

    pub(crate) fn finish(self) -> BTreeMap<String, SourceValue> {
        self.fields
    }
}

pub(crate) fn decode_array<T>(
    value: SourceValue,
    path: String,
    decoder: Decoder<T>,
) -> Result<Vec<T>, ResponseDecodeError> {
    let actual = value.kind();
    let SourceValueRepr::Array(values) = value.0 else {
        return Err(ResponseDecodeError::WrongKind {
            path,
            expected: SourceValueKind::Array,
            actual,
        });
    };
    values
        .into_iter()
        .enumerate()
        .map(|(index, value)| decoder(value, format!("{path}/{index}")))
        .collect()
}

pub(crate) fn decode_xml_array<T>(
    value: SourceValue,
    path: String,
    decoder: Decoder<T>,
) -> Result<Vec<T>, ResponseDecodeError> {
    match value.0 {
        SourceValueRepr::Array(values) => values
            .into_iter()
            .enumerate()
            .map(|(index, value)| decoder(value, format!("{path}/{index}")))
            .collect(),
        value => decoder(SourceValue(value), format!("{path}/0")).map(|value| vec![value]),
    }
}

pub(crate) fn decode_string(
    value: SourceValue,
    path: String,
) -> Result<String, ResponseDecodeError> {
    let actual = value.kind();
    let SourceValueRepr::String(value) = value.0 else {
        return Err(ResponseDecodeError::WrongKind {
            path,
            expected: SourceValueKind::String,
            actual,
        });
    };
    Ok(value)
}

pub(crate) fn decode_source_status(
    value: SourceValue,
    path: String,
) -> Result<SourceStatus, ResponseDecodeError> {
    decode_string(value, path).map(SourceStatus::new)
}

pub(crate) fn decode_source_value(
    value: SourceValue,
    _path: String,
) -> Result<SourceValue, ResponseDecodeError> {
    Ok(value)
}

fn child_path(parent: &str, name: &str) -> String {
    format!("{parent}/{}", name.replace('~', "~0").replace('/', "~1"))
}
