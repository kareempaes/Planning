import { auth } from './auth.svelte';
import type { WSEvent } from './types';

type EventHandler = (data: unknown) => void;

let socket = $state<WebSocket | null>(null);
let connected = $state(false);

const listeners: Map<string, Set<EventHandler>> = new Map();
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let pingTimer: ReturnType<typeof setInterval> | null = null;
let reconnectDelay = 1000;

function connect() {
	const token = auth.getAccessToken();
	if (!token || socket) return;

	const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
	const wsUrl = `${protocol}//${window.location.host}/api/v1/ws?token=${token}`;

	const ws = new WebSocket(wsUrl);

	ws.onopen = () => {
		socket = ws;
		connected = true;
		reconnectDelay = 1000;

		pingTimer = setInterval(() => {
			if (ws.readyState === WebSocket.OPEN) {
				ws.send(JSON.stringify({ type: 'ping' }));
			}
		}, 30_000);
	};

	ws.onmessage = (event) => {
		try {
			const parsed: WSEvent = JSON.parse(event.data);
			const handlers = listeners.get(parsed.type);
			if (handlers) {
				handlers.forEach((fn) => fn(parsed.data));
			}
		} catch {
			// ignore malformed frames
		}
	};

	ws.onclose = () => {
		cleanup();
		scheduleReconnect();
	};

	ws.onerror = () => {
		ws.close();
	};
}

function cleanup() {
	socket = null;
	connected = false;
	if (pingTimer) {
		clearInterval(pingTimer);
		pingTimer = null;
	}
}

function scheduleReconnect() {
	if (!auth.isAuthenticated) return;
	reconnectTimer = setTimeout(() => {
		connect();
		reconnectDelay = Math.min(reconnectDelay * 2, 30_000);
	}, reconnectDelay);
}

function disconnect() {
	if (reconnectTimer) {
		clearTimeout(reconnectTimer);
		reconnectTimer = null;
	}
	if (socket) {
		socket.close();
	}
	cleanup();
}

function on(type: string, handler: EventHandler): () => void {
	if (!listeners.has(type)) listeners.set(type, new Set());
	listeners.get(type)!.add(handler);
	return () => {
		listeners.get(type)?.delete(handler);
	};
}

function send(event: WSEvent) {
	if (socket?.readyState === WebSocket.OPEN) {
		socket.send(JSON.stringify(event));
	}
}

export const ws = {
	get connected() { return connected; },
	connect,
	disconnect,
	on,
	send,
};
