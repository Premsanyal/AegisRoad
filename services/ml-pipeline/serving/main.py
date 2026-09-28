"""
ML Pipeline Serving - FastAPI service for traffic prediction models
"""
import os
import asyncio
import logging
from contextlib import asynccontextmanager
from typing import List, Optional

import torch
import numpy as np
import pandas as pd
from fastapi import FastAPI, HTTPException, BackgroundTasks
from pydantic import BaseModel, Field
from prometheus_client import Counter, Histogram, generate_latest
from starlette.responses import Response

import redis
from kafka import KafkaConsumer, KafkaProducer
import json

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Prometheus metrics
PREDICTION_REQUESTS = Counter('ml_prediction_requests_total', 'Total prediction requests', ['model', 'status'])
PREDICTION_LATENCY = Histogram('ml_prediction_latency_seconds', 'Prediction latency', ['model'])

# Global model store
models = {}
redis_client = None
kafka_producer = None
kafka_consumer = None


class ETAPredictionRequest(BaseModel):
    route_geometry: List[List[float]] = Field(..., description="Route as [[lon, lat], ...]")
    current_speeds: List[float] = Field(..., description="Current speed per segment (km/h)")
    segment_lengths: List[float] = Field(..., description="Segment lengths (meters)")
    time_of_day: int = Field(..., description="Hour of day (0-23)")
    day_of_week: int = Field(..., description="Day of week (0-6)")
    weather: Optional[str] = Field("clear", description="Weather condition")


class ETAPredictionResponse(BaseModel):
    segment_etas: List[float] = Field(..., description="ETA per segment (seconds)")
    total_eta: float = Field(..., description="Total ETA (seconds)")
    confidence_intervals: List[List[float]] = Field(..., description="[lower, upper] per segment")


class CongestionForecastRequest(BaseModel):
    bbox: List[float] = Field(..., description="[min_lon, min_lat, max_lon, max_lat]")
    horizon_minutes: int = Field(30, description="Forecast horizon in minutes")


class CongestionForecastResponse(BaseModel):
    timestamp: str
    predictions: List[dict] = Field(..., description="List of {segment_id, congestion_level, confidence}")


class IncidentDetectionRequest(BaseModel):
    camera_frames: List[str] = Field(..., description="Base64 encoded frames")
    location: List[float] = Field(..., description="[lon, lat]")


class IncidentDetectionResponse(BaseModel):
    incident_probability: float
    incident_type: str
    confidence: float
    bounding_boxes: List[List[float]]


@asynccontextmanager
async def lifespan(app: FastAPI):
    global models, redis_client, kafka_producer, kafka_consumer

    # Initialize Redis
    redis_client = redis.Redis.from_url(os.getenv("REDIS_URL", "redis://localhost:6379"), decode_responses=True)
    await redis_client.ping()
    logger.info("Connected to Redis")

    # Initialize Kafka
    kafka_producer = KafkaProducer(
        bootstrap_servers=os.getenv("KAFKA_BROKERS", "localhost:9092").split(","),
        value_serializer=lambda v: json.dumps(v).encode('utf-8')
    )
    logger.info("Kafka producer initialized")

    # Load models
    await load_models()
    logger.info("Models loaded")

    # Start background tasks
    asyncio.create_task(consume_traffic_data())

    yield

    # Cleanup
    await redis_client.close()
    kafka_producer.close()
    logger.info("Shutdown complete")


app = FastAPI(title="AegisRoad ML Pipeline", lifespan=lifespan)


async def load_models():
    """Load all ML models from model store"""
    model_store = os.getenv("MODEL_STORE", "/models")

    # ETA Predictor (LSTM)
    models['eta_predictor'] = ETAPredictorModel()
    await models['eta_predictor'].load(f"{model_store}/eta_predictor.pt")

    # Congestion Forecaster (GNN)
    models['congestion_forecaster'] = CongestionForecasterModel()
    await models['congestion_forecaster'].load(f"{model_store}/congestion_forecaster.pt")

    # Incident Detector (CNN)
    models['incident_detector'] = IncidentDetectorModel()
    await models['incident_detector'].load(f"{model_store}/incident_detector.pt")


class BaseModel:
    def __init__(self):
        self.device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')

    async def load(self, path: str):
        if os.path.exists(path):
            self.model = torch.jit.load(path, map_location=self.device)
            self.model.eval()
            logger.info(f"Loaded model from {path}")
        else:
            # Create default model for development
            self.model = self._create_default()
            logger.warning(f"Model not found at {path}, using default")

    def _create_default(self):
        raise NotImplementedError


class ETAPredictorModel(BaseModel):
    def _create_default(self):
        return torch.nn.Sequential(
            torch.nn.Linear(10, 64),
            torch.nn.ReLU(),
            torch.nn.Linear(64, 32),
            torch.nn.ReLU(),
            torch.nn.Linear(32, 1)
        ).to(self.device)

    async def predict(self, request: ETAPredictionRequest) -> ETAPredictionResponse:
        with PREDICTION_LATENCY.labels(model='eta_predictor').time():
            # Prepare features
            features = self._prepare_features(request)
            features_tensor = torch.tensor(features, dtype=torch.float32).unsqueeze(0).to(self.device)

            # Predict
            with torch.no_grad():
                predictions = self.model(features_tensor).cpu().numpy().flatten()

            # Convert to segment ETAs
            segment_etas = predictions.tolist()
            total_eta = sum(segment_etas)

            # Simple confidence intervals (±20%)
            confidence_intervals = [[eta * 0.8, eta * 1.2] for eta in segment_etas]

            PREDICTION_REQUESTS.labels(model='eta_predictor', status='success').inc()
            return ETAPredictionResponse(
                segment_etas=segment_etas,
                total_eta=total_eta,
                confidence_intervals=confidence_intervals
            )

    def _prepare_features(self, request: ETAPredictionRequest) -> np.ndarray:
        # Feature engineering
        n_segments = len(request.segment_lengths)
        features = []

        for i in range(n_segments):
            seg_features = [
                request.segment_lengths[i] / 1000.0,  # km
                request.current_speeds[i] / 100.0,    # normalized
                request.time_of_day / 24.0,
                request.day_of_week / 7.0,
                1.0 if request.weather == "rain" else 0.0,
                1.0 if request.weather == "fog" else 0.0,
            ]
            # Add historical averages (mock)
            features.extend(seg_features)

        # Pad or truncate to fixed size
        target_size = 10
        if len(features) > target_size:
            features = features[:target_size]
        else:
            features.extend([0.0] * (target_size - len(features)))

        return np.array(features, dtype=np.float32)


class CongestionForecasterModel(BaseModel):
    def _create_default(self):
        return torch.nn.Sequential(
            torch.nn.Linear(20, 128),
            torch.nn.ReLU(),
            torch.nn.Linear(128, 64),
            torch.nn.ReLU(),
            torch.nn.Linear(64, 6)  # 6 congestion levels
        ).to(self.device)

    async def predict(self, request: CongestionForecastRequest) -> CongestionForecastResponse:
        with PREDICTION_LATENCY.labels(model='congestion_forecaster').time():
            # In production: query road segments in bbox, run GNN
            # For now, return mock predictions
            predictions = [
                {"segment_id": 1001, "congestion_level": 2, "confidence": 0.85},
                {"segment_id": 1002, "congestion_level": 3, "confidence": 0.78},
            ]

            PREDICTION_REQUESTS.labels(model='congestion_forecaster', status='success').inc()
            return CongestionForecastResponse(
                timestamp=pd.Timestamp.now().isoformat(),
                predictions=predictions
            )


class IncidentDetectorModel(BaseModel):
    def _create_default(self):
        return torch.nn.Sequential(
            torch.nn.Conv2d(3, 32, 3, padding=1),
            torch.nn.ReLU(),
            torch.nn.AdaptiveAvgPool2d((1, 1)),
            torch.nn.Flatten(),
            torch.nn.Linear(32, 4)  # no_incident, accident, breakdown, hazard
        ).to(self.device)

    async def predict(self, request: IncidentDetectionRequest) -> IncidentDetectionResponse:
        with PREDICTION_LATENCY.labels(model='incident_detector').time():
            # In production: decode base64 frames, run CNN
            # For now, return mock
            PREDICTION_REQUESTS.labels(model='incident_detector', status='success').inc()
            return IncidentDetectionResponse(
                incident_probability=0.15,
                incident_type="none",
                confidence=0.9,
                bounding_boxes=[]
            )


async def consume_traffic_data():
    """Background task to consume traffic data from Kafka and update models"""
    consumer = KafkaConsumer(
        'traffic.speeds',
        bootstrap_servers=os.getenv("KAFKA_BROKERS", "localhost:9092").split(","),
        value_deserializer=lambda m: json.loads(m.decode('utf-8')),
        group_id='ml-pipeline-traffic',
        auto_offset_reset='latest'
    )

    for message in consumer:
        data = message.value
        # Update real-time features cache in Redis
        key = f"traffic:segment:{data['segment_id']}"
        await redis_client.hset(key, mapping={
            'speed': data['speed_kmh'],
            'confidence': data['confidence'],
            'timestamp': data['timestamp']
        })
        await redis_client.expire(key, 300)  # 5 min TTL


# API Endpoints
@app.post("/predict/eta", response_model=ETAPredictionResponse)
async def predict_eta(request: ETAPredictionRequest):
    try:
        return await models['eta_predictor'].predict(request)
    except Exception as e:
        PREDICTION_REQUESTS.labels(model='eta_predictor', status='error').inc()
        logger.error(f"ETA prediction failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/predict/congestion", response_model=CongestionForecastResponse)
async def predict_congestion(request: CongestionForecastRequest):
    try:
        return await models['congestion_forecaster'].predict(request)
    except Exception as e:
        PREDICTION_REQUESTS.labels(model='congestion_forecaster', status='error').inc()
        logger.error(f"Congestion prediction failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/detect/incident", response_model=IncidentDetectionResponse)
async def detect_incident(request: IncidentDetectionRequest):
    try:
        return await models['incident_detector'].predict(request)
    except Exception as e:
        PREDICTION_REQUESTS.labels(model='incident_detector', status='error').inc()
        logger.error(f"Incident detection failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/metrics")
async def metrics():
    return Response(content=generate_latest(), media_type="text/plain")


@app.get("/health")
async def health():
    return {"status": "healthy", "models_loaded": list(models.keys())}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)