<script lang="ts">
	import { cn, type WithElementRef } from "$lib/utils.js";
	import type { HTMLAttributes } from "svelte/elements";

	let {
		ref = $bindable(null),
		class: className,
		placeholder,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLSpanElement>> & {
		placeholder?: string;
	} = $props();
</script>

<span
	bind:this={ref}
	data-slot="select-value"
	class={cn("pointer-events-none", className)}
	{...restProps}
>
	{#if children}
		{@render children()}
	{:else if placeholder}
		{placeholder}
	{/if}
</span>