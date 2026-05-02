import { defineConfig } from '@hey-api/openapi-ts';

export default defineConfig({
	input: '../api/openapi.yaml',
	output: {
		path: 'src/lib/api',
		format: 'prettier',
		clean: true
	},
	plugins: [
		{
			name: '@hey-api/client-fetch',
			runtimeConfigPath: './src/lib/api-client.ts'
		},
		'@hey-api/typescript',
		{ name: '@hey-api/sdk', asClass: false }
	]
});
