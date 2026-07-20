use std::io::Write;

pub(crate) fn json(value: &impl serde::Serialize) -> Result<(), ()> {
    let mut encoded = serde_json::to_vec(value).map_err(|_| ())?;
    encoded.push(b'\n');
    std::io::stdout().lock().write_all(&encoded).map_err(|_| ())
}
