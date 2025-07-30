<script lang="ts">
	import { Dialog as DialogPrimitive } from "bits-ui";
	
	let {
		open = $bindable(false),
		children,
		...restProps
	}: {
		open?: boolean;
		children?: import("svelte").Snippet;
	} & DialogPrimitive.RootProps = $props();
</script>

<DialogPrimitive.Root bind:open {...restProps}>
	{@render children?.()}
</DialogPrimitive.Root>