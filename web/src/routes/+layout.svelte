<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { ws } from '$lib/ws.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import '../app.css';

	let { children } = $props();

	$effect(() => {
		auth.initialize();
	});

	$effect(() => {
		if (!auth.initializing && !auth.isAuthenticated && page.url.pathname !== '/login') {
			goto('/login');
		}
	});

	$effect(() => {
		if (auth.isAuthenticated) {
			ws.connect();
			return () => ws.disconnect();
		}
	});

	function logout() {
		ws.disconnect();
		auth.clearSession();
		goto('/login');
	}
</script>

{#if auth.initializing}
	<div class="loading">Loading...</div>
{:else}
	{#if auth.isAuthenticated}
		<header>
			<span>Chat &mdash; {auth.user?.display_name}</span>
			<button onclick={logout}>Logout</button>
		</header>
	{/if}
	{@render children()}
{/if}
