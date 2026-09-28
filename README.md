# AegisRoad - Smart Traffic Management System

A real-time traffic management platform for Bengaluru that prioritizes emergency vehicles (ambulance, fire, police) on optimal routes while redirecting public traffic to alternative routes.

## 🎯 Features

- **Priority Routing**: Emergency vehicles get guaranteed optimal routes with corridor reservation
- **Public Redirection**: Common traffic automatically routed to 2nd/3rd best alternatives
- **Three Interfaces**:
  - 📱 **Public PWA** - Mobile-first navigation with offline support
  - 🚑 **Emergency App** - Tablet-optimized for ambulance/fire/police
  - 🖥️ **Operator Dashboard** - Multi-monitor traffic control center
- **Real-time Data**: Live congestion, incidents, camera feeds
- **SCATS Integration**: Signal preemption for emergency vehicles
- **ML Predictions**: ETA forecasting, congestion prediction, incident detection

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        AegisRoad Platform                        │
├─────────────────┬─────────────────┬─────────────────────────────┤
│   Public PWA    │  Emergency App  │    Operator Dashboard       │
│   (React/TS)    │   (React/TS)    │      (React/TS)             │
└────────┬────────┴────────┬────────┴────────────┬───────────────┘
         │                 │                     │
         └─────────────────┼─────────────────────┘
                           ▼
              ┌─────────────────────────┐
              │      API Gateway         │
              │      (Go + gRPC)         │
              └───────────┬──────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        ▼                 ▼                 ▼
┌───────────────┐ ┌───────────────┐ ┌───────────────┐
│ Routing Engine│ │ Realtime Server│ │  SCATS Emul.  │
│   (Rust)      │ │  (Node.js)     │ │    (Go)       │
└───────┬───────┘ └───────┬───────┘ └───────┬───────┘
        │                 │                 │
        └─────────────────┼─────────────────┘
                          ▼
         ┌────────────────────────────┐
         │     Data Layer             │
         │ PostgreSQL + PostGIS       │
         │ TimescaleDB + Redis        │
         │ Kafka + NATS               │
         └────────────────────────────┘
```

## 🛠️ Tech Stack

| Layer | Technology |
|-------|------------|
| **Routing** | Rust + OSRM (custom priority profiles) |
| **API Gateway** | Go (Gin) + gRPC + WebSocket |
| **Real-time** | Node.js + Socket.io + NATS |
| **Signal Control** | Go (SCATS/SCOOT compatible) |
| **ML Pipeline** | Python (FastAPI) + PyTorch |
| **Maps** | OpenMapTiles + MapLibre GL JS |
| **Database** | PostgreSQL + PostGIS + TimescaleDB |
| **Cache/Queue** | Redis Cluster + Apache Kafka |
| **Frontend** | React 18 + TypeScript + Vite + PWA |
| **Deployment** | Docker Compose (dev) → Kubernetes (prod) |

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Git
- 8GB+ RAM recommended

### Development Setup

```bash
# Clone repository
git clone https://github.com/Premsanyal/AegisRoad.git
cd AegisRoad

# Copy environment template
cp .env.example .env
# Edit .env with your API keys (TomTom, HERE, etc.)

# Start all services
make dev

# Or start infrastructure only
make up

# Run database migrations
make db-migrate
make db-seed
```

### Access Points

| Service | URL |
|---------|-----|
| Public PWA | http://localhost:5173 |
| Emergency App | http://localhost:5174 |
| Operator Dashboard | http://localhost:5175 |
| API Gateway | http://localhost:8081 |
| Routing Engine (gRPC) | localhost:50051 |
| Real-time Server | ws://localhost:3001 |
| SCATS Emulator | http://localhost:8082 |
| ML Pipeline | http://localhost:8000 |
| Tile Server | http://localhost:8080 |

## 📁 Project Structure

```
AegisRoad/
├── apps/
│   ├── public-pwa/          # Public navigation PWA
│   ├── emergency-app/       # Emergency vehicle tablet app
│   └── operator-dashboard/  # Traffic operator control center
├── services/
│   ├── routing-engine/      # Rust + OSRM priority routing
│   ├── api-gateway/         # Go REST/gRPC/WebSocket API
│   ├── realtime-server/     # Node.js Socket.io server
│   ├── scats-emulator/      # SCATS/SCOOT signal emulator
│   ├── ml-pipeline/         # Python ML serving (ETA, congestion, incidents)
│   ├── tile-server/         # OpenMapTiles vector tiles
│   └── data-ingestion/      # Kafka connectors for traffic APIs
├── packages/
│   ├── shared-types/        # TypeScript types from protobuf
│   ├── ui-components/       # Shared React components
│   └── map-utils/           # MapLibre utilities
├── database/
│   ├── migrations/          # SQL migrations
│   └── seeds/               # Bengaluru sample data
├── infra/
│   ├── k8s/                 # Kubernetes manifests
│   └── terraform/           # Cloud infrastructure
├── docker/
│   └── docker-compose.yml   # Local development stack
└── scripts/
    └── import-osm.sh        # Bengaluru OSM import
```

## 🗺️ Bengaluru Prototype

The prototype covers central Bengaluru with:
- **Road Network**: OSM import (primary, secondary, residential roads)
- **Key Junctions**: Silk Board, Marathahalli, Hebbal, Koramangala, Electronic City
- **Emergency Corridors**: Hospital/fire station access roads
- **Traffic Cameras**: 5 simulated BBMP camera feeds
- **Sample Data**: 6 emergency vehicles, 8 users, live traffic speeds

## 🔌 API Endpoints

### Routes
```
POST   /api/v1/routes           # Calculate route
GET    /api/v1/routes/:id       # Get route details
POST   /api/v1/routes/:id/accept # Accept route (start navigation)
```

### Incidents
```
POST   /api/v1/incidents        # Report incident
GET    /api/v1/incidents        # List incidents
PUT    /api/v1/incidents/:id    # Update incident
POST   /api/v1/incidents/:id/assign # Assign emergency vehicles
```

### Vehicles
```
GET    /api/v1/vehicles/:id     # Get vehicle status
PUT    /api/v1/vehicles/:id/location # Update GPS location
PUT    /api/v1/vehicles/:id/status   # Update status
```

### WebSocket (Real-time)
```
WS /ws/v1/live
Channels: traffic, incidents, vehicles, alerts
```

## 📊 Priority Routing Logic

| Priority Level | User Type | Route Access |
|----------------|-----------|--------------|
| 0 | Public | 2nd/3rd best routes only |
| 1 | Operator | Standard + reserved lanes (peak) |
| 2 | Emergency | Optimal route + corridor reservation + signal preemption |
| 3 | VIP | Near-optimal + limited reservation |

## 🧪 Testing

```bash
# Run all tests
make test

# Lint all services
make lint

# Type check
make typecheck
```

## 📦 Deployment

### Production (Kubernetes)
```bash
# Build and push images
make build
make push

# Deploy to dev cluster
make deploy-dev

# Or use kubectl directly
kubectl apply -k infra/k8s/overlays/prod
```

### Environment Variables
Key variables in `.env`:
- `DB_PASSWORD` - PostgreSQL password
- `JWT_SECRET` - 32-char secret for tokens
- `TOMTOM_API_KEY` / `HERE_API_KEY` - Traffic data
- `FIREBASE_*` - Push notifications
- `GHCR_TOKEN` - Container registry

## 🤝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

- OpenStreetMap contributors for Bengaluru road data
- OSRM Project for routing engine
- MapLibre for open-source mapping
- BBMP for traffic camera feeds (simulated)
- TimescaleDB for time-series traffic data

---

**Built with ❤️ for safer, smarter Bengaluru traffic**