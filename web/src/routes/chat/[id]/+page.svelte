<script lang="ts">
	import { api } from '$lib/api';
	import { auth } from '$lib/auth.svelte';
	import { ws } from '$lib/ws.svelte';
	import { page } from '$app/state';
	import type { Message } from '$lib/types';

	let messages = $state<Message[]>([]);
	let newMessage = $state('');
	let sending = $state(false);
	let messagesContainer: HTMLElement;

	const conversationId = $derived(page.params.id);

	$effect(() => {
		if (conversationId) {
			loadMessages(conversationId);
		}
	});

	$effect(() => {
		const unsub = ws.on('message', (data: unknown) => {
			const msg = data as Message;
			if (msg.conversation_id === conversationId) {
				if (!messages.some((m) => m.id === msg.id)) {
					messages = [...messages, msg];
					scrollToBottom();
				}
			}
		});
		return unsub;
	});

	async function loadMessages(convoId: string) {
		try {
			const res = await api.getMessages(convoId);
			messages = (res.messages ?? []).reverse();
			scrollToBottom();
		} catch {
			// ignore
		}
	}

	async function send(e: SubmitEvent) {
		e.preventDefault();
		if (!newMessage.trim() || !conversationId) return;
		sending = true;
		try {
			const msg = await api.sendMessage(conversationId, newMessage);
			messages = [...messages, msg];
			newMessage = '';
			scrollToBottom();
		} catch {
			// ignore
		}
		sending = false;
	}

	function scrollToBottom() {
		setTimeout(() => {
			if (messagesContainer) {
				messagesContainer.scrollTop = messagesContainer.scrollHeight;
			}
		}, 0);
	}

	function isOwnMessage(msg: Message): boolean {
		return msg.sender_id === auth.user?.id;
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' && !event.shiftKey) {
			event.preventDefault();
			const form = (event.target as HTMLElement).closest('form');
			if (form) form.requestSubmit();
		}
	}
</script>

<div class="chat-page">
	<div class="chat-header">
		<a href="/">&larr; Back</a>
	</div>

	<div class="messages" bind:this={messagesContainer}>
		{#each messages as msg (msg.id)}
			<div class="message" class:own={isOwnMessage(msg)}>
				<div class="bubble">
					<p>{msg.body}</p>
					<small>{new Date(msg.created_at).toLocaleTimeString()}</small>
				</div>
			</div>
		{/each}
	</div>

	<form class="send-bar" onsubmit={send}>
		<input
			type="text"
			bind:value={newMessage}
			placeholder="Type a message..."
			onkeydown={handleKeydown}
		/>
		<button type="submit" disabled={sending || !newMessage.trim()}>Send</button>
	</form>
</div>
