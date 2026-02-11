// Auth
export interface AuthResponse {
	user: User;
	tokens: Tokens;
}

export interface Tokens {
	access_token: string;
	refresh_token: string;
	expires_in: number;
}

export interface User {
	id: string;
	email: string;
	display_name: string;
	avatar_url: string | null;
	status: string;
	created_at: string;
}

// Users
export interface PublicProfile {
	id: string;
	display_name: string;
	avatar_url: string | null;
	status: string;
}

// Conversations
export interface Conversation {
	id: string;
	type: 'direct' | 'group';
	name: string | null;
	participants: Participant[];
	created_at: string;
}

export interface ConversationSummary {
	id: string;
	type: 'direct' | 'group';
	name: string | null;
	last_message: MessagePreview | null;
	unread_count: number;
	participants: ParticipantMin[];
}

export interface Participant {
	user_id: string;
	display_name: string;
	role: string;
}

export interface ParticipantMin {
	user_id: string;
	display_name: string;
}

export interface MessagePreview {
	id: string;
	body: string;
	sender_id: string;
	created_at: string;
}

// Messages
export interface Message {
	id: string;
	conversation_id: string;
	sender_id: string;
	body: string;
	status: string;
	created_at: string;
}

// Pagination
export interface Pagination {
	next_cursor: string | null;
	has_more: boolean;
}

// WebSocket
export interface WSEvent {
	type: string;
	data?: unknown;
}
