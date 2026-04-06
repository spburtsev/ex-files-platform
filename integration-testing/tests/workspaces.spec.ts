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

test.describe('Workspaces', () => {
	test('manager can create a workspace', async ({ page }) => {
		await login(page, MANAGER.email, MANAGER.password);
		await page.goto('/workspaces');

		// Click create button (the "+" icon button)
		await page.getByRole('button', { name: /create|new/i }).click();

		// Fill in the workspace name in the dialog
		const dialog = page.getByRole('dialog');
		await expect(dialog).toBeVisible();
		const nameInput = dialog.getByRole('textbox');
		const wsName = `E2E Workspace ${Date.now()}`;
		await nameInput.fill(wsName);

		// Submit
		await dialog.getByRole('button', { name: /create/i }).click();

		// Should navigate to the new workspace detail page
		await expect(page).toHaveURL(/\/workspaces\/\d+/);
	});

	test('employee does not see create workspace button', async ({ page }) => {
		await login(page, EMPLOYEE.email, EMPLOYEE.password);
		await page.goto('/workspaces');

		// The create button should not be present for employees
		await expect(page.getByRole('button', { name: /create|new/i })).not.toBeVisible();
	});
});
