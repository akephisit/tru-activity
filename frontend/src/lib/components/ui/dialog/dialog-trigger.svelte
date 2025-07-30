<script lang="ts">
	import { Dialog as DialogPrimitive } from "bits-ui";
	
	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: {
		ref?: any;
		class?: string;
		children?: import("svelte").Snippet;
	} & DialogPrimitive.TriggerProps = $props();
</script>

<DialogPrimitive.Trigger
	bind:this={ref}
	class={className}
	{...restProps}
>
	{@render children?.()}
</DialogPrimitive.Trigger>