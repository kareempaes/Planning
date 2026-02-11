import { auth } from './auth.svelte';
import type {
	AuthResponse,
	Tokens,
	Conversation,
	ConversationSummary,
	Message,
	PublicProfile,
	Pagination,
} from './types';

class ApiClient {
	private async request<T>(path: string, options: RequestInit = {}): Promise<T> {
		const headers = new Headers(options.headers);
		headers.set('Content-Type', 'application/json');

		const token = auth.getAccessToken();
		if (token) {
			headers.set('Authorization', `Bearer ${token}`);
		}

		let res = await fetch(path, { ...options, headers });

		if (res.status === 401 && token) {
			const refreshed = await auth.refreshAccessToken();
			if (refreshed) {
				const newToken = auth.getAccessToken();
				if (newToken) {
					headers.set('Authorization', `Bearer ${newToken}`);
					res = await fetch(path, { ...options, headers });
				}
			}
		}

		if (!res.ok) {
			const body = await res.json().catch(() => ({}));
			throw new Error(body?.error?.message ?? `Request failed: ${res.status}`);
		}

		if (res.status === 204) return undefined as T;
		return res.json();
	}

	async register(email: string, password: string, displayName: string): Promise<AuthResponse> {
		return this.request('/api/v1/auth/register', {
			method: 'POST',
			body: JSON.stringify({ email, password, display_name: displayName }),
		});
	}

	async login(email: string, password: string): Promise<AuthResponse> {
		return this.request('/api/v1/auth/login', {
			method: 'POST',
			body: JSON.stringify({ email, password }),
		});
	}

	async searchUsers(query: string): Promise<{ users: PublicProfile[]; pagination: Pagination }> {
		return this.request(`/api/v1/users?q=${encodeURIComponent(query)}`);
	}

	async listConversations(): Promise<{ conversations: ConversationSummary[]; pagination: Pagination }> {
		return this.request('/api/v1/conversations');
	}

	async createConversation(participantIds: string[]): Promise<Conversation> {
		return this.request('/api/v1/conversations', {
			method: 'POST',
			body: JSON.stringify({ type: 'direct', participant_ids: participantIds }),
		});
	}

	async getMessages(conversationId: string, cursor?: string): Promise<{ messages: Message[]; pagination: Pagination }> {
		const params = cursor ? `?cursor=${cursor}&limit=50` : '?limit=50';
		return this.request(`/api/v1/conversations/${conversationId}/messages${params}`);
	}

	async sendMessage(conversationId: string, body: string): Promise<Message> {
		return this.request(`/api/v1/conversations/${conversationId}/messages`, {
			method: 'POST',
			body: JSON.stringify({ body }),
		});
	}
}

export const api = new ApiClient();
