<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { alertVariants, type AlertProps } from "./alert.js";

	type $$Props = AlertProps & {
		class?: string;
		children?: import('svelte').Snippet;
	};

	let { 
		variant,
		class: className,
		children,
		...restProps
	}: $$Props = $props();
</script>

<div class={cn(alertVariants({ variant }), className)} role="alert" {...restProps}>
	{@render children?.()}
</div>