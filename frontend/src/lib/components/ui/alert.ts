import { tv, type VariantProps } from "tailwind-variants";

export { default as Alert } from "./alert.svelte";
export { default as AlertDescription } from "./alert-description.svelte";
export { default as AlertTitle } from "./alert-title.svelte";

export const alertVariants = tv({
  base: "relative w-full rounded-lg border p-4 [&>svg~*]:pl-7 [&>svg+div]:translate-y-[-3px] [&>svg]:absolute [&>svg]:left-4 [&>svg]:top-4 [&>svg]:text-foreground",
  variants: {
    variant: {
      default: "bg-background text-foreground",
      destructive:
        "border-destructive/50 text-destructive dark:border-destructive [&>svg]:text-destructive",
    },
  },
  defaultVariants: {
    variant: "default",
  },
});

export type AlertProps = VariantProps<typeof alertVariants>;