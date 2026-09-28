use serde::Deserialize;
use anyhow::Result;

#[derive(Debug, Clone, Deserialize)]
pub struct Config {
    pub grpc_addr: String,
    pub osrm_endpoint: String,
    pub database_url: String,
    pub redis_url: String,
    pub kafka_brokers: String,
    pub jwt_secret: String,
    pub default_profile: String,
    pub emergency_lookahead_meters: u32,
    pub max_alternatives: usize,
}

impl Config {
    pub fn load() -> Result<Self> {
        dotenvy::dotenv().ok();

        let config = config::Config::builder()
            .add_source(config::Environment::with_prefix("ROUTING"))
            .build()?;

        Ok(Self {
            grpc_addr: config.get_string("GRPC_ADDR").unwrap_or_else(|_| "[::]:50051".to_string()),
            osrm_endpoint: config.get_string("OSRM_ENDPOINT").unwrap_or_else(|_| "http://localhost:5000".to_string()),
            database_url: config.get_string("DATABASE_URL").unwrap_or_else(|_| "postgres://aegis:aegisroad_dev@localhost:5432/aegisroad".to_string()),
            redis_url: config.get_string("REDIS_URL").unwrap_or_else(|_| "redis://localhost:6379".to_string()),
            kafka_brokers: config.get_string("KAFKA_BROKERS").unwrap_or_else(|_| "localhost:9092".to_string()),
            jwt_secret: config.get_string("JWT_SECRET").unwrap_or_else(|_| "dev_secret".to_string()),
            default_profile: config.get_string("DEFAULT_PROFILE").unwrap_or_else(|_| "car".to_string()),
            emergency_lookahead_meters: config.get_int("EMERGENCY_LOOKAHEAD_METERS").unwrap_or(3000) as u32,
            max_alternatives: config.get_int("MAX_ALTERNATIVES").unwrap_or(3) as usize,
        })
    }
}