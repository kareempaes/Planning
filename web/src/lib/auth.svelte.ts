import type { User, Tokens } from './types';

let accessToken = $state<string | null>(null);
let refreshToken = $state<string | null>(null);
let user = $state<User | null>(null);
let initializing = $state(true);

const isAuthenticated = $derived(accessToken !== null && user !== null);

function setSession(u: User, tokens: Tokens) {
	user = u;
	accessToken = tokens.access_token;
	refreshToken = tokens.refresh_token;
	localStorage.setItem('refresh_token', tokens.refresh_token);
}

function clearSession() {
	user = null;
	accessToken = null;
	refreshToken = null;
	localStorage.removeItem('refresh_token');
}

function getAccessToken(): string | null {
	return accessToken;
}

async function refreshAccessToken(): Promise<boolean> {
	const rt = refreshToken ?? localStorage.getItem('refresh_token');
	if (!rt) return false;

	try {
		const res = await fetch('/api/v1/auth/refresh', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ refresh_token: rt }),
		});
		if (!res.ok) {
			clearSession();
			return false;
		}
		const tokens: Tokens = await res.json();
		accessToken = tokens.access_token;
		refreshToken = tokens.refresh_token;
		localStorage.setItem('refresh_token', tokens.refresh_token);
		return true;
	} catch {
		clearSession();
		return false;
	}
}

async function initialize() {
	const stored = localStorage.getItem('refresh_token');
	if (!stored) {
		initializing = false;
		return;
	}
	refreshToken = stored;
	const ok = await refreshAccessToken();
	if (ok && accessToken) {
		const res = await fetch('/api/v1/users/me', {
			headers: { Authorization: `Bearer ${accessToken}` },
		});
		if (res.ok) {
			user = await res.json();
		} else {
			clearSession();
		}
	}
	initializing = false;
}

export const auth = {
	get user() { return user; },
	get isAuthenticated() { return isAuthenticated; },
	get initializing() { return initializing; },
	getAccessToken,
	refreshAccessToken,
	setSession,
	clearSession,
	initialize,
};
