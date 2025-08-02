import adapter from "@sveltejs/adapter-node";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

/** @type {import('@sveltejs/kit').Config} */
const config = {
  // Consult https://svelte.dev/docs/kit/integrations
  // for more information about preprocessors
  preprocess: vitePreprocess(),

  kit: {
    // Node.js adapter for Cloud Run deployment
    adapter: adapter({
      out: 'build'
    }),
    alias: {
      "$lib": "./src/lib",
    },
    prerender: {
      handleHttpError: 'warn',
      entries: ['/']
    }
  },
};

export default config;
