import { test, expect } from '@playwright/test';

const BACKEND_URL = process.env.BACKEND_URL ?? 'http://localhost:8080';

test.describe('Health checks', () => {
	test('frontend is reachable', async ({ page }) => {
		const response = await page.goto('/login');
		expect(response?.status()).toBe(200);
	});

	test('backend healthz endpoint', async ({ request }) => {
		const response = await request.get(`${BACKEND_URL}/healthz`);
		expect(response.status()).toBe(200);

		const body = await response.json();
		expect(body.status).toBe('ok');
	});
});
