<script lang="ts">
	import { api } from '$lib/api';
	import { auth } from '$lib/auth.svelte';
	import { ws } from '$lib/ws.svelte';
	import { goto } from '$app/navigation';
	import type { ConversationSummary, PublicProfile, Message } from '$lib/types';

	let conversations = $state<ConversationSummary[]>([]);
	let searchQuery = $state('');
	let searchResults = $state<PublicProfile[]>([]);
	let searching = $state(false);

	$effect(() => {
		loadConversations();
	});

	$effect(() => {
		const unsub = ws.on('message', (data: unknown) => {
			const msg = data as Message;
			conversations = conversations.map((c) => {
				if (c.id === msg.conversation_id) {
					return {
						...c,
						last_message: {
							id: msg.id,
							body: msg.body,
							sender_id: msg.sender_id,
							created_at: msg.created_at,
						},
					};
				}
				return c;
			});
		});
		return unsub;
	});

	async function loadConversations() {
		try {
			const res = await api.listConversations();
			conversations = res.conversations ?? [];
		} catch {
			// ignore
		}
	}

	async function searchUsers() {
		if (!searchQuery.trim()) {
			searchResults = [];
			return;
		}
		searching = true;
		try {
			const res = await api.searchUsers(searchQuery);
			searchResults = (res.users ?? []).filter((u) => u.id !== auth.user?.id);
		} catch {
			// ignore
		}
		searching = false;
	}

	async function startConversation(userId: string) {
		try {
			const convo = await api.createConversation([userId]);
			goto(`/chat/${convo.id}`);
		} catch {
			// ignore
		}
	}

	function getConversationName(c: ConversationSummary): string {
		if (c.name) return c.name;
		const other = c.participants.find((p) => p.user_id !== auth.user?.id);
		return other?.display_name ?? 'Unknown';
	}
</script>

<div class="home">
	<section class="new-conversation">
		<h2>New Conversation</h2>
		<div class="search-bar">
			<input
				type="text"
				bind:value={searchQuery}
				placeholder="Search users by name..."
				oninput={searchUsers}
			/>
		</div>
		{#if searchResults.length > 0}
			<ul class="search-results">
				{#each searchResults as user (user.id)}
					<li>
						<button onclick={() => startConversation(user.id)}>
							{user.display_name}
						</button>
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	<section class="conversation-list">
		<h2>Conversations</h2>
		{#if conversations.length === 0}
			<p class="empty">No conversations yet. Search for a user to start chatting.</p>
		{:else}
			<ul>
				{#each conversations as convo (convo.id)}
					<li>
						<a href="/chat/{convo.id}">
							<strong>{getConversationName(convo)}</strong>
							{#if convo.last_message}
								<span class="preview">{convo.last_message.body}</span>
							{/if}
						</a>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</div>
