import type { CodegenConfig } from '@graphql-codegen/cli';

const config: CodegenConfig = {
  overwrite: true,
  schema: 'http://localhost:8080/query',
  documents: 'src/**/*.{ts,svelte}',
  generates: {
    'src/lib/generated/': {
      preset: 'client',
      plugins: [],
      config: {
        useTypeImports: true,
      },
    },
    'src/lib/generated/schema.ts': {
      plugins: ['typescript'],
    },
  },
};

export default config;