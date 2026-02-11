<script lang="ts">
	import { api } from '$lib/api';
	import { auth } from '$lib/auth.svelte';
	import { goto } from '$app/navigation';

	let mode = $state<'login' | 'register'>('login');
	let email = $state('');
	let password = $state('');
	let displayName = $state('');
	let error = $state('');
	let loading = $state(false);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		error = '';
		loading = true;
		try {
			const result = mode === 'login'
				? await api.login(email, password)
				: await api.register(email, password, displayName);
			auth.setSession(result.user, result.tokens);
			goto('/');
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Something went wrong';
		} finally {
			loading = false;
		}
	}

	function toggleMode() {
		mode = mode === 'login' ? 'register' : 'login';
		error = '';
	}
</script>

<div class="auth-page">
	<h1>{mode === 'login' ? 'Log In' : 'Register'}</h1>
	<form onsubmit={submit}>
		{#if mode === 'register'}
			<input type="text" bind:value={displayName} placeholder="Display Name" required />
		{/if}
		<input type="email" bind:value={email} placeholder="Email" required />
		<input type="password" bind:value={password} placeholder="Password" required />
		{#if error}
			<p class="error">{error}</p>
		{/if}
		<button type="submit" disabled={loading}>
			{loading ? '...' : mode === 'login' ? 'Log In' : 'Register'}
		</button>
	</form>
	<button class="toggle" onclick={toggleMode}>
		{mode === 'login' ? 'Need an account? Register' : 'Have an account? Log In'}
	</button>
</div>
