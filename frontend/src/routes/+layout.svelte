<script lang="ts">
	import '../app.css';
	import { setClient } from '@apollo/client';
	import { client } from '$lib/graphql/client';
	import { authStore } from '$lib/stores/auth';
	import { onMount } from 'svelte';
	
	let { children } = $props();

	// Set Apollo Client
	setClient(client);

	// Initialize auth store on mount
	onMount(() => {
		authStore.init();
	});
</script>

{@render children()}
