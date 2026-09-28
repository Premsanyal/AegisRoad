import { Server } from 'socket.io';
import { createServer } from 'http';
import Redis from 'ioredis';
import { connect } from 'nats.ws';
import { Kafka } from 'kafkajs';
import pino from 'pino';

const logger = pino({ level: process.env.LOG_LEVEL || 'info' });

// Configuration
const PORT = parseInt(process.env.PORT || '3001');
const REDIS_URL = process.env.REDIS_URL || 'redis://localhost:6379';
const NATS_URL = process.env.NATS_URL || 'nats://localhost:4222';
const KAFKA_BROKERS = (process.env.KAFKA_BROKERS || 'localhost:9092').split(',');

// Initialize connections
const redis = new Redis(REDIS_URL);
const kafka = new Kafka({ clientId: 'realtime-server', brokers: KAFKA_BROKERS });
const consumer = kafka.consumer({ groupId: 'realtime-server' });

// Socket.io server
const httpServer = createServer();
const io = new Server(httpServer, {
  cors: { origin: '*', methods: ['GET', 'POST'] },
  pingInterval: 25000,
  pingTimeout: 20000,
});

// Namespaces
const publicNs = io.of('/public');
const emergencyNs = io.of('/emergency');
const operatorNs = io.of('/operator');

// Redis pub/sub for horizontal scaling
const redisSub = new Redis(REDIS_URL);
const redisPub = new Redis(REDIS_URL);

// NATS connection for lightweight messaging
let natsConn: any = null;

async function connectNATS() {
  try {
    natsConn = await connect({ servers: NATS_URL });
    logger.info('Connected to NATS');
  } catch (err) {
    logger.warn({ err }, 'NATS connection failed, continuing without');
  }
}

// Kafka consumer for real-time events
async function startKafkaConsumer() {
  await consumer.connect();
  await consumer.subscribe({ topic: 'traffic.speeds', fromBeginning: false });
  await consumer.subscribe({ topic: 'traffic.incidents', fromBeginning: false });
  await consumer.subscribe({ topic: 'vehicle.gps', fromBeginning: false });
  await consumer.subscribe({ topic: 'routing.updates', fromBeginning: false });

  await consumer.run({
    eachMessage: async ({ topic, partition, message }) => {
      if (!message.value) return;
      const data = JSON.parse(message.value.toString());

      switch (topic) {
        case 'traffic.speeds':
          broadcastToPublic('congestion_update', data);
          broadcastToOperator('congestion_update', data);
          break;
        case 'traffic.incidents':
          broadcastToPublic('incident_update', data);
          broadcastToEmergency('incident_dispatch', data);
          broadcastToOperator('incident_update', data);
          break;
        case 'vehicle.gps':
          broadcastToVehicle(data.vehicle_id, 'location_update', data);
          broadcastToOperator('vehicle_update', data);
          break;
        case 'routing.updates':
          broadcastToUser(data.user_id, 'route_update', data);
          break;
      }
    },
  });

  logger.info('Kafka consumer started');
}

// Redis subscriber for cross-instance messages
async function startRedisSubscriber() {
  await redisSub.subscribe('aegisroad:broadcast', 'aegisroad:role:public', 'aegisroad:role:emergency', 'aegisroad:role:operator');
  redisSub.on('message', (channel, message) => {
    try {
      const { type, data } = JSON.parse(message);
      const role = channel.split(':').pop();
      if (role === 'broadcast') {
        broadcastAll(type, data);
      } else {
        broadcastToRole(role, type, data);
      }
    } catch (err) {
      logger.warn({ err, channel }, 'Invalid Redis message');
    }
  });
  logger.info('Redis subscriber started');
}

// Broadcast functions
function broadcastAll(event: string, data: any) {
  publicNs.emit(event, data);
  emergencyNs.emit(event, data);
  operatorNs.emit(event, data);
}

function broadcastToRole(role: string, event: string, data: any) {
  switch (role) {
    case 'public':
      publicNs.emit(event, data);
      break;
    case 'emergency':
      emergencyNs.emit(event, data);
      break;
    case 'operator':
      operatorNs.emit(event, data);
      break;
  }
}

function broadcastToPublic(event: string, data: any) {
  publicNs.emit(event, data);
}

function broadcastToEmergency(event: string, data: any) {
  emergencyNs.emit(event, data);
}

function broadcastToOperator(event: string, data: any) {
  operatorNs.emit(event, data);
}

function broadcastToVehicle(vehicleId: string, event: string, data: any) {
  // In production, maintain vehicle->socket mapping
  operatorNs.to(`vehicle:${vehicleId}`).emit(event, data);
  emergencyNs.to(`vehicle:${vehicleId}`).emit(event, data);
}

function broadcastToUser(userId: string, event: string, data: any) {
  publicNs.to(`user:${userId}`).emit(event, data);
  emergencyNs.to(`user:${userId}`).emit(event, data);
}

// Authentication middleware for namespaces
function authMiddleware(namespace: any, allowedRoles: string[]) {
  namespace.use((socket: any, next: any) => {
    const token = socket.handshake.auth.token;
    const role = socket.handshake.auth.role;

    if (!token || !role || !allowedRoles.includes(role)) {
      return next(new Error('Authentication required'));
    }

    // In production: validate JWT token
    socket.data.userId = socket.handshake.auth.userId;
    socket.data.role = role;
    socket.data.vehicleId = socket.handshake.auth.vehicleId;

    // Join rooms
    socket.join(`role:${role}`);
    if (socket.data.userId) socket.join(`user:${socket.data.userId}`);
    if (socket.data.vehicleId) socket.join(`vehicle:${socket.data.vehicleId}`);

    next();
  });
}

// Apply auth
authMiddleware(publicNs, ['public']);
authMiddleware(emergencyNs, ['emergency']);
authMiddleware(operatorNs, ['operator']);

// Connection handlers
publicNs.on('connection', (socket) => {
  logger.info({ userId: socket.data.userId }, 'Public client connected');

  socket.on('subscribe_channel', (channel: string) => {
    socket.join(`channel:${channel}`);
  });

  socket.on('unsubscribe_channel', (channel: string) => {
    socket.leave(`channel:${channel}`);
  });

  socket.on('disconnect', () => {
    logger.info({ userId: socket.data.userId }, 'Public client disconnected');
  });
});

emergencyNs.on('connection', (socket) => {
  logger.info({ userId: socket.data.userId, vehicleId: socket.data.vehicleId }, 'Emergency client connected');

  socket.on('accept_dispatch', (data: { incidentId: string }) => {
    // Forward to Kafka for processing
    kafka.producer().send({
      topic: 'dispatch.accepted',
      messages: [{ value: JSON.stringify({ ...data, vehicleId: socket.data.vehicleId, userId: socket.data.userId }) }],
    });
  });

  socket.on('request_preemption', (data: { intersectionId: string }) => {
    kafka.producer().send({
      topic: 'signal.preemption_request',
      messages: [{ value: JSON.stringify({ ...data, vehicleId: socket.data.vehicleId }) }],
    });
  });

  socket.on('disconnect', () => {
    logger.info({ userId: socket.data.userId }, 'Emergency client disconnected');
  });
});

operatorNs.on('connection', (socket) => {
  logger.info({ userId: socket.data.userId }, 'Operator client connected');

  socket.on('subscribe_incidents', (filter?: any) => {
    socket.join('incidents:all');
  });

  socket.on('subscribe_vehicles', (filter?: any) => {
    socket.join('vehicles:all');
  });

  socket.on('subscribe_intersections', (ids?: string[]) => {
    if (ids) {
      ids.forEach((id: string) => socket.join(`intersection:${id}`));
    } else {
      socket.join('intersections:all');
    }
  });

  socket.on('signal_override', (data: { intersectionId: string; phasePlan: any }) => {
    kafka.producer().send({
      topic: 'signal.override',
      messages: [{ value: JSON.stringify(data) }],
    });
  });

  socket.on('broadcast_alert', (data: any) => {
    kafka.producer().send({
      topic: 'alerts.broadcast',
      messages: [{ value: JSON.stringify({ ...data, createdBy: socket.data.userId }) }],
    });
  });

  socket.on('disconnect', () => {
    logger.info({ userId: socket.data.userId }, 'Operator client disconnected');
  });
});

// Health check endpoint
httpServer.on('request', (req, res) => {
  if (req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ status: 'ok', service: 'realtime-server' }));
  }
});

// Start server
async function start() {
  await connectNATS();
  await startKafkaConsumer();
  await startRedisSubscriber();

  httpServer.listen(PORT, () => {
    logger.info(`Real-time server listening on port ${PORT}`);
  });
}

// Graceful shutdown
process.on('SIGTERM', async () => {
  logger.info('Shutting down...');
  await consumer.disconnect();
  await redis.quit();
  await redisSub.quit();
  await redisPub.quit();
  if (natsConn) await natsConn.close();
  httpServer.close();
  process.exit(0);
});

start().catch((err) => {
  logger.fatal({ err }, 'Failed to start server');
  process.exit(1);
});