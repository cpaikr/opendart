use super::{ResponseDecodeError, SourceValue, SourceValueKind};

pub(crate) type Decoder = fn(&SourceValue, String) -> Result<(), ResponseDecodeError>;

/// A borrowed object-shape validator that never duplicates retained source data.
pub(crate) struct ObjectDecoder<'a> {
    value: &'a SourceValue,
    path: String,
}

impl<'a> ObjectDecoder<'a> {
    pub(crate) fn new(value: &'a SourceValue, path: String) -> Result<Self, ResponseDecodeError> {
        let actual = value.kind();
        if actual != SourceValueKind::Object {
            return Err(ResponseDecodeError::WrongKind {
                path,
                expected: SourceValueKind::Object,
                actual,
            });
        }
        Ok(Self { value, path })
    }

    pub(crate) fn new_xml(
        value: &'a SourceValue,
        path: String,
    ) -> Result<Self, ResponseDecodeError> {
        let actual = value.kind();
        if actual != SourceValueKind::Object && value.as_str() != Some("") {
            return Err(ResponseDecodeError::WrongKind {
                path,
                expected: SourceValueKind::Object,
                actual,
            });
        }
        Ok(Self { value, path })
    }

    pub(crate) fn optional(
        &self,
        name: &'static str,
        decoder: Decoder,
    ) -> Result<(), ResponseDecodeError> {
        let Some(value) = self.value.get(name) else {
            return Ok(());
        };
        decoder(value, child_path(&self.path, name))
    }
}

pub(crate) fn decode_array(
    value: &SourceValue,
    path: String,
    decoder: Decoder,
) -> Result<(), ResponseDecodeError> {
    let actual = value.kind();
    let Some(values) = value.as_array() else {
        return Err(ResponseDecodeError::WrongKind {
            path,
            expected: SourceValueKind::Array,
            actual,
        });
    };
    for (index, value) in values.iter().enumerate() {
        decoder(value, format!("{path}/{index}"))?;
    }
    Ok(())
}

pub(crate) fn decode_xml_array(
    value: &SourceValue,
    path: String,
    decoder: Decoder,
) -> Result<(), ResponseDecodeError> {
    if let Some(values) = value.as_array() {
        for (index, value) in values.iter().enumerate() {
            decoder(value, format!("{path}/{index}"))?;
        }
        return Ok(());
    }
    decoder(value, format!("{path}/0"))
}

pub(crate) fn decode_string(value: &SourceValue, path: String) -> Result<(), ResponseDecodeError> {
    let actual = value.kind();
    if actual != SourceValueKind::String {
        return Err(ResponseDecodeError::WrongKind {
            path,
            expected: SourceValueKind::String,
            actual,
        });
    }
    Ok(())
}

pub(crate) fn decode_source_status(
    value: &SourceValue,
    path: String,
) -> Result<(), ResponseDecodeError> {
    decode_string(value, path)
}

fn child_path(parent: &str, name: &str) -> String {
    format!("{parent}/{}", name.replace('~', "~0").replace('/', "~1"))
}
