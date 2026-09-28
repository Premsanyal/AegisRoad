fn main() -> Result<(), Box<dyn std::error::Error>> {
    let proto_file = "proto/routing.proto";
    let proto_dir = "proto";

    tonic_build::configure()
        .build_server(true)
        .build_client(true)
        .compile(&[proto_file], &[proto_dir])?;

    Ok(())
}