import { test, expect, type Page } from '@playwright/test';

const MANAGER = { email: 's.martinez@acme.org', password: 'password123' };
const EMPLOYEE = { email: 'a.johnson@acme.org', password: 'password123' };

async function login(page: Page, email: string, password: string) {
	await page.goto('/login');
	await page.locator('#email').fill(email);
	await page.locator('#password').fill(password);
	await page.locator('button[type="submit"]').click();
	await expect(page).not.toHaveURL(/\/login/);
}

test.describe('Sidebar navigation', () => {
	test('employee sees dashboard, workspaces, users', async ({ page }) => {
		await login(page, EMPLOYEE.email, EMPLOYEE.password);

		const sidebar = page.locator('[data-sidebar="sidebar"]');
		await expect(sidebar.getByRole('link', { name: /dashboard/i })).toBeVisible();
		await expect(sidebar.getByRole('link', { name: /workspaces/i })).toBeVisible();
		await expect(sidebar.getByRole('link', { name: /users/i })).toBeVisible();
	});

	test('manager also sees audit log', async ({ page }) => {
		await login(page, MANAGER.email, MANAGER.password);

		const sidebar = page.locator('[data-sidebar="sidebar"]');
		await expect(sidebar.getByRole('link', { name: /audit/i })).toBeVisible();
	});

	test('navigate to workspaces page', async ({ page }) => {
		await login(page, EMPLOYEE.email, EMPLOYEE.password);

		await page.locator('[data-sidebar="sidebar"]').getByRole('link', { name: /workspaces/i }).click();
		await expect(page).toHaveURL(/\/workspaces/);
	});
});
