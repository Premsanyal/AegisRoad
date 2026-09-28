// Routing Engine - High-performance priority-based routing for AegisRoad
// Built with Rust + OSRM for sub-millisecond routing queries

mod config;
mod error;
mod models;
mod osrm_client;
mod priority_router;
mod corridor_manager;
mod grpc_server;

use std::sync::Arc;
use tracing::{info, error, warn};
use anyhow::Result;

use crate::config::Config;
use crate::grpc_server::RoutingServiceImpl;
use crate::osrm_client::OsrmClient;
use crate::corridor_manager::CorridorManager;
use crate::priority_router::PriorityRouter;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize tracing
    tracing_subscriber::fmt()
        .with_env_filter(tracing_subscriber::EnvFilter::from_default_env())
        .init();

    info!("Starting AegisRoad Routing Engine");

    // Load configuration
    let config = Config::load()?;
    info!("Configuration loaded: {:?}", config);

    // Initialize OSRM client
    let osrm_client = Arc::new(OsrmClient::new(&config.osrm_endpoint).await?);
    info!("OSRM client connected to {}", config.osrm_endpoint);

    // Initialize corridor manager
    let corridor_manager = Arc::new(CorridorManager::new());

    // Initialize priority router
    let router = Arc::new(PriorityRouter::new(osrm_client.clone(), corridor_manager.clone()));

    // Build gRPC service
    let routing_service = RoutingServiceImpl::new(router, corridor_manager);

    // Start gRPC server
    let addr = config.grpc_addr.parse()?;
    info!("gRPC server listening on {}", addr);

    tonic::transport::Server::builder()
        .add_service(routing_service.into_router())
        .serve(addr)
        .await?;

    Ok(())
}