<script lang="ts">
	import { Dialog as DialogPrimitive } from "bits-ui";
	import { cn, type WithElementRef } from "$lib/utils.js";
	
	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: {
		ref?: any;
		class?: string;
		children?: import("svelte").Snippet;
	} & DialogPrimitive.TitleProps = $props();
</script>

<DialogPrimitive.Title
	bind:this={ref}
	class={cn("text-lg font-semibold leading-none tracking-tight", className)}
	{...restProps}
>
	{@render children?.()}
</DialogPrimitive.Title>