import { test, expect } from '@playwright/test';

// Seed data from ex-files-backend/seed/seed.go
const MANAGER = { email: 's.martinez@acme.org', password: 'password123', name: 'Sofia Martinez' };
const EMPLOYEE = { email: 'a.johnson@acme.org', password: 'password123', name: 'Alex Johnson' };

test.describe('Authentication', () => {
	test('unauthenticated user is redirected to login', async ({ page }) => {
		await page.goto('/');
		await expect(page).toHaveURL(/\/login/);
	});

	test('login with invalid credentials shows error', async ({ page }) => {
		await page.goto('/login');
		await page.locator('#email').fill('nobody@example.com');
		await page.locator('#password').fill('wrongpassword');
		await page.locator('button[type="submit"]').click();

		await expect(page.locator('.bg-destructive\\/10')).toBeVisible();
	});

	test('login with valid employee credentials', async ({ page }) => {
		await page.goto('/login');
		await page.locator('#email').fill(EMPLOYEE.email);
		await page.locator('#password').fill(EMPLOYEE.password);
		await page.locator('button[type="submit"]').click();

		// Should land on dashboard with greeting
		await expect(page).not.toHaveURL(/\/login/);
		await expect(page.locator('h1')).toContainText('Alex');
	});

	test('login with valid manager credentials', async ({ page }) => {
		await page.goto('/login');
		await page.locator('#email').fill(MANAGER.email);
		await page.locator('#password').fill(MANAGER.password);
		await page.locator('button[type="submit"]').click();

		await expect(page).not.toHaveURL(/\/login/);
		await expect(page.locator('h1')).toContainText('Sofia');
	});

	test('logout redirects to login', async ({ page }) => {
		// Login first
		await page.goto('/login');
		await page.locator('#email').fill(EMPLOYEE.email);
		await page.locator('#password').fill(EMPLOYEE.password);
		await page.locator('button[type="submit"]').click();
		await expect(page).not.toHaveURL(/\/login/);

		// Open user dropdown and logout
		await page.getByRole('button', { name: EMPLOYEE.name }).click();
		await page.getByRole('menuitem', { name: /log\s*out/i }).click();

		await expect(page).toHaveURL(/\/login/);
	});

	test('signup with new user', async ({ page }) => {
		const unique = `e2e-${Date.now()}@test.org`;

		await page.goto('/signup');
		await page.locator('#name').fill('E2E Tester');
		await page.locator('#email').fill(unique);
		await page.locator('#password').fill('testpassword123');
		await page.locator('button[type="submit"]').click();

		// Should land on dashboard
		await expect(page).not.toHaveURL(/\/signup/);
		await expect(page.locator('h1')).toContainText('E2E');
	});
});
