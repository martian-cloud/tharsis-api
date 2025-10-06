import type { CodegenConfig } from '@graphql-codegen/cli';

const config: CodegenConfig = {
  overwrite: true,
  schema: "http://localhost:8000/graphql",
  generates: {
    "./schema/schema.graphql": {
      plugins: [
        // This plugin generates our schema.graphql file.
        "schema-ast",
      ]
    }
  }
};

export default config;
