<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { textareaVariants, type TextareaProps } from "./textarea.js";

	type $$Props = TextareaProps & {
		value?: string;
		placeholder?: string;
		disabled?: boolean;
		readonly?: boolean;
		class?: string;
		rows?: number;
		cols?: number;
	};

	let { 
		value = $bindable(),
		placeholder,
		disabled = false,
		readonly = false,
		class: className,
		rows,
		cols,
		...restProps 
	}: $$Props = $props();
</script>

<textarea
	bind:value
	{placeholder}
	{disabled}
	{readonly}
	{rows}
	{cols}
	class={cn(textareaVariants(), className)}
	{...restProps}
></textarea>