use thiserror::Error;

#[derive(Error, Debug)]
pub enum RoutingError {
    #[error("OSRM error: {0}")]
    OsrmError(String),

    #[error("No route found between origin and destination")]
    NoRouteFound,

    #[error("Invalid coordinates: {0}")]
    InvalidCoordinates(String),

    #[error("Priority level not supported: {0}")]
    UnsupportedPriority(i32),

    #[error("Corridor reservation failed: {0}")]
    CorridorReservationFailed(String),

    #[error("Corridor not found: {0}")]
    CorridorNotFound(String),

    #[error("Database error: {0}")]
    DatabaseError(#[from] sqlx::Error),

    #[error("Serialization error: {0}")]
    SerializationError(#[from] serde_json::Error),

    #[error("Internal error: {0}")]
    Internal(String),
}

pub type Result<T> = std::result::Result<T, RoutingError>;