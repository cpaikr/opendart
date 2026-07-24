use std::io::Write;

pub(crate) fn json(value: &impl serde::Serialize) -> Result<(), ()> {
    write(encode(value)?)
}

pub(crate) fn encode(value: &impl serde::Serialize) -> Result<Vec<u8>, ()> {
    let mut encoded = serde_json::to_vec(value).map_err(|_| ())?;
    encoded.push(b'\n');
    Ok(encoded)
}

pub(crate) fn write(encoded: Vec<u8>) -> Result<(), ()> {
    std::io::stdout().lock().write_all(&encoded).map_err(|_| ())
}

#[cfg(test)]
mod tests {
    use serde::Serialize;

    struct FailsToSerialize;

    impl Serialize for FailsToSerialize {
        fn serialize<S>(&self, _serializer: S) -> Result<S::Ok, S::Error>
        where
            S: serde::Serializer,
        {
            Err(serde::ser::Error::custom("fixture failure"))
        }
    }

    #[test]
    fn encoding_finishes_one_document_before_stdout_write() {
        assert_eq!(super::encode(&["complete"]).unwrap(), b"[\"complete\"]\n");
        assert!(super::encode(&FailsToSerialize).is_err());
    }
}
