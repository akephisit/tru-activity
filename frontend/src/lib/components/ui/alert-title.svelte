<script lang="ts">
	import { cn } from "$lib/utils.js";

	type $$Props = {
		class?: string;
		children?: import('svelte').Snippet;
	};

	let { 
		class: className,
		children,
		...restProps
	}: $$Props = $props();
</script>

<h5 class={cn("mb-1 font-medium leading-none tracking-tight", className)} {...restProps}>
	{@render children?.()}
</h5>