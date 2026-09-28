use reqwest::Client;
use serde::{Deserialize, Serialize};
use geo::{Point, LineString};
use anyhow::Result;
use std::sync::Arc;
use tracing::{debug, warn};

#[derive(Debug, Clone, Serialize, Deserialize)]
struct OsrmRouteResponse {
    code: String,
    routes: Vec<OsrmRoute>,
    waypoints: Vec<OsrmWaypoint>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct OsrmRoute {
    geometry: OsrmGeometry,
    distance: f64,
    duration: f64,
    weight: f64,
    legs: Vec<OsrmLeg>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct OsrmGeometry {
    coordinates: Vec<Vec<f64>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct OsrmWaypoint {
    location: Vec<f64>,
    name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct OsrmLeg {
    steps: Vec<OsrmStep>,
    summary: String,
    distance: f64,
    duration: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct OsrmStep {
    geometry: OsrmGeometry,
    distance: f64,
    duration: f64,
    name: String,
    intersections: Vec<OsrmIntersection>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct OsrmIntersection {
    location: Vec<f64>,
    bearings: Vec<i32>,
    entry: Vec<bool>,
    in_: i32,
    out: i32,
}

pub struct OsrmClient {
    client: Client,
    base_url: String,
    profiles: Vec<String>,
}

impl OsrmClient {
    pub async fn new(base_url: &str) -> Result<Arc<Self>> {
        let client = Client::builder()
            .timeout(std::time::Duration::from_secs(5))
            .build()?;

        // Verify OSRM is reachable
        let health_url = format!("{}/health", base_url.trim_end_matches('/'));
        match client.get(&health_url).send().await {
            Ok(resp) if resp.status().is_success() => {
                debug!("OSRM health check passed");
            }
            Ok(resp) => {
                warn!("OSRM health check returned: {}", resp.status());
            }
            Err(e) => {
                warn!("OSRM health check failed: {}", e);
            }
        }

        Ok(Arc::new(Self {
            client,
            base_url: base_url.trim_end_matches('/').to_string(),
            profiles: vec!["car".to_string(), "bike".to_string(), "foot".to_string(), "emergency".to_string()],
        }))
    }

    pub async fn route(
        &self,
        origin: Point,
        destination: Point,
        profile: &str,
        alternatives: bool,
        steps: bool,
    ) -> Result<OsrmRouteResponse> {
        let coords = format!("{},{};{},{}", origin.x(), origin.y(), destination.x(), destination.y());
        let url = format!("{}/route/v1/{}/{}", self.base_url, profile, coords);

        let mut query = vec![
            ("overview", "full"),
            ("geometries", "geojson"),
            ("steps", if steps { "true" } else { "false" }),
            ("alternatives", if alternatives { "true" } else { "false" }),
            ("annotations", "nodes,distance,duration,speed"),
        ];

        let resp = self.client.get(&url).query(&query).send().await?;

        if !resp.status().is_success() {
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow::anyhow!("OSRM error: {}", text));
        }

        let osrm_resp: OsrmRouteResponse = resp.json().await?;

        if osrm_resp.code != "Ok" || osrm_resp.routes.is_empty() {
            return Err(anyhow::anyhow!("No route found"));
        }

        Ok(osrm_resp)
    }

    pub async fn table(
        &self,
        sources: &[Point],
        destinations: &[Point],
        profile: &str,
    ) -> Result<OsrmTableResponse> {
        let coords: Vec<String> = sources
            .iter()
            .chain(destinations.iter())
            .map(|p| format!("{},{}", p.x(), p.y()))
            .collect();

        let url = format!("{}/table/v1/{}/{}", self.base_url, profile, coords.join(";"));

        let resp = self.client
            .get(&url)
            .query(&[
                ("sources", &sources.iter().map(|_| "0").collect::<Vec<_>>().join(";")),
                ("destinations", &destinations.iter().map(|_| "0").collect::<Vec<_>>().join(";")),
                ("annotations", "duration,distance"),
            ])
            .send()
            .await?;

        if !resp.status().is_success() {
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow::anyhow!("OSRM table error: {}", text));
        }

        Ok(resp.json().await?)
    }

    pub fn available_profiles(&self) -> &[String] {
        &self.profiles
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct OsrmTableResponse {
    code: String,
    durations: Vec<Vec<f64>>,
    distances: Vec<Vec<f64>>,
    sources: Vec<OsrmWaypoint>,
    destinations: Vec<OsrmWaypoint>,
}

impl OsrmClient {
    pub fn convert_geometry(&self, geom: &OsrmGeometry) -> LineString<f64> {
        LineString::from(geom.coordinates.iter().map(|c| Point::new(c[0], c[1])).collect::<Vec<_>>())
    }

    pub fn extract_segments(&self, route: &OsrmRoute) -> Vec<SegmentInfo> {
        route.legs.iter().flat_map(|leg| {
            leg.steps.iter().enumerate().map(move |(i, step)| {
                SegmentInfo {
                    segment_id: 0, // Will be filled by router
                    osm_id: 0,
                    name: step.name.clone(),
                    highway_type: "unknown".to_string(),
                    max_speed: 50,
                    current_speed: (step.distance / step.duration.max(0.1) * 3.6) as i32,
                    congestion_level: 0,
                    length_meters: step.distance as i64,
                    duration_seconds: step.duration as i32,
                    is_reserved: false,
                    is_oneway: false,
                }
            })
        }).collect()
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct SegmentInfo {
    segment_id: i64,
    osm_id: i64,
    name: String,
    highway_type: String,
    max_speed: i32,
    current_speed: i32,
    congestion_level: i32,
    length_meters: i64,
    duration_seconds: i32,
    is_reserved: bool,
    is_oneway: bool,
}