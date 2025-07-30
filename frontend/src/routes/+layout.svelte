<script lang="ts">
	import '../app.css';
	import { authStore } from '$lib/stores/auth';
	import { onMount } from 'svelte';
	
	let { children } = $props();

	// Initialize auth store on mount
	onMount(() => {
		authStore.init();
	});
</script>

{@render children()}
