/**
 * Traffic generator for the ex-files app.
 *
 * Drives the real UI as the seeded users (Playwright, like a human) to produce
 * audit events — logins, workspace + member + issue creation, PDF uploads, new
 * versions, comments, and review actions (request-changes / approve / reject).
 * Sleeps between operations so the activity spreads over time (nicer time-series
 * on the Grafana "Audit" dashboard) and looks organic.
 *
 * Usage:
 *   cd integration-testing
 *   bunx playwright install chromium        # once, if browsers aren't installed
 *   BASE_URL=https://exfiles.spbware.xyz ROUNDS=3 bun scripts/generate-traffic.ts
 *
 * Env vars:
 *   BASE_URL        target app origin (default https://exfiles.spbware.xyz)
 *   ROUNDS          number of full issue lifecycles to run (default 3)
 *   HEADLESS        "false" to watch the browser (default true)
 *   SEED_PASSWORD   password for the seeded demo users (default password123)
 *   MIN_MS / MAX_MS sleep range between operations in ms (default 800 / 3000)
 */
import { chromium, type Page } from 'playwright';

const BASE_URL = (process.env.BASE_URL ?? 'https://exfiles.spbware.xyz').replace(/\/$/, '');
const ROUNDS = Number(process.env.ROUNDS ?? '3');
const HEADLESS = process.env.HEADLESS !== 'false';
const PASSWORD = process.env.SEED_PASSWORD ?? 'password123';
const MIN_MS = Number(process.env.MIN_MS ?? '800');
const MAX_MS = Number(process.env.MAX_MS ?? '3000');

// Seeded demo users (ex-files-backend/seed/seed.go). The root/admin user is
// intentionally excluded — these are the regular accounts.
const MANAGER = { email: 's.martinez@acme.org', name: 'Sofia Martinez' };
const EMPLOYEES = [
	{ email: 'a.johnson@acme.org', name: 'Alex Johnson' },
	{ email: 'm.chen@acme.org', name: 'Maria Chen' },
	{ email: 'j.wilson@acme.org', name: 'James Wilson' }
];

const rand = (min: number, max: number) => Math.floor(Math.random() * (max - min + 1)) + min;
const pick = <T>(arr: T[]): T => arr[rand(0, arr.length - 1)];
const sleep = (min = MIN_MS, max = MAX_MS) => new Promise((r) => setTimeout(r, rand(min, max)));
const log = (...a: unknown[]) => console.log(new Date().toISOString(), ...a);

const WS_NOUNS = ['Contracts', 'Compliance', 'Vendor Onboarding', 'Policies', 'Audit Pack', 'NDAs'];
const DOC_TOPICS = ['vendor agreement', 'NDA', 'SOW', 'data-processing addendum', 'service contract'];
const COMMENTS = [
	'Section 3 pricing terms need a second look.',
	'Please confirm the effective date matches the cover page.',
	'Signature block looks good to me.',
	'Can we clarify the termination clause here?',
	'Numbers reconcile with the finance sheet — thanks.',
	'Typo in the counterparty name, please fix.'
];

/** Build a small, unique, valid 1-page PDF (unique nonce => unique hash, no 409). */
function makePdf(label: string): Buffer {
	const nonce = `${Date.now()}-${Math.random().toString(36).slice(2)}`;
	const stream = `BT /F1 13 Tf 36 720 Td (${label}) Tj 0 -22 Td (ref ${nonce}) Tj ET`;
	const pdf = `%PDF-1.4
1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj
2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj
3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 612 792]/Contents 4 0 R/Resources<</Font<</F1 5 0 R>>>>>>endobj
4 0 obj<</Length ${stream.length}>>stream
${stream}
endstream endobj
5 0 obj<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>endobj
trailer<</Root 1 0 R/Size 6>>
startxref
0
%%EOF`;
	return Buffer.from(pdf, 'utf8');
}

async function login(page: Page, user: { email: string; name: string }) {
	await page.goto(`${BASE_URL}/login`);
	await page.locator('#email').fill(user.email);
	await page.locator('#password').fill(PASSWORD);
	await page.locator('button[type="submit"]').click();
	await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 20000 });
	log(`  login: ${user.name}`);
	await sleep();
}

async function logout(page: Page, user: { name: string }) {
	try {
		await page.getByRole('button', { name: user.name }).first().click();
		await page.getByRole('menuitem', { name: /log\s*out/i }).click();
		await page.waitForURL(/\/login/, { timeout: 15000 });
	} catch {
		// Fall back to clearing the session cookie if the menu isn't reachable.
		await page.context().clearCookies();
	}
	await sleep();
}

async function createWorkspace(page: Page, name: string): Promise<string> {
	await page.goto(`${BASE_URL}/workspaces`);
	await page.getByRole('button', { name: /new workspace/i }).click();
	const dialog = page.getByRole('dialog');
	await dialog.getByRole('textbox').first().fill(name);
	await dialog.getByRole('button', { name: /create/i }).click();
	await page.waitForURL(/\/workspaces\/\d+$/, { timeout: 20000 });
	log(`  workspace created: "${name}"`);
	await sleep();
	return page.url();
}

async function addMembers(page: Page, names: string[]) {
	await page.getByRole('tab', { name: 'Members' }).click();
	await sleep(400, 1200);
	await page.getByRole('button', { name: /add member/i }).click();
	const dialog = page.getByRole('dialog');
	// The assignable-members list loads async — wait for it before clicking.
	await dialog.getByText(names[0], { exact: false }).first().waitFor({ timeout: 10000 }).catch(() => {});
	for (const name of names) {
		try {
			await dialog.getByText(name, { exact: false }).first().click({ timeout: 5000 });
			log(`  member added: ${name}`);
			await sleep(400, 1200);
		} catch {
			log(`  (could not add ${name})`);
		}
	}
	// The dialog has two "Close" affordances (corner ✕ + footer button); Escape is unambiguous.
	await page.keyboard.press('Escape');
	await sleep();
}

async function createIssue(page: Page, title: string, assigneeName: string) {
	await page.getByRole('tab', { name: 'Issues' }).click();
	await sleep(400, 1200);
	await page.getByRole('button', { name: /new issue/i }).click();
	const dialog = page.getByRole('dialog');
	await dialog.getByPlaceholder(/review q2 report/i).fill(title);
	await dialog.getByPlaceholder(/optional details/i).fill('Auto-generated for review traffic.');
	await dialog.getByRole('button', { name: /select a member/i }).click();
	await page.getByRole('option', { name: assigneeName }).click();
	await dialog.getByRole('button', { name: /^create$/i }).click();
	await sleep(800, 1800);
	log(`  issue created: "${title}" -> ${assigneeName}`);
	// Open the newly listed issue and return its URL.
	await page.getByRole('link', { name: /view/i }).first().click();
	await page.waitForURL(/\/workspaces\/\d+\/issues\/\d+/, { timeout: 20000 });
	await sleep();
	return page.url();
}

async function uploadVersion(page: Page, label: string) {
	// Reveal the dropzone if a saved version already exists.
	const uploadBtn = page.getByRole('button', { name: /upload submission/i });
	if (await uploadBtn.isVisible().catch(() => false)) {
		await uploadBtn.click();
		await sleep(400, 1000);
	}
	const input = page.locator('input[type="file"]').first();
	await input.setInputFiles({
		name: `${label.replace(/[^a-z0-9]+/gi, '-').toLowerCase()}.pdf`,
		mimeType: 'application/pdf',
		buffer: makePdf(label)
	});
	await sleep(1000, 2000); // let pdf.js render the preview
	await page.getByRole('button', { name: /^save$/i }).click();
	await page.getByText(/saved/i).first().waitFor({ timeout: 12000 }).catch(() => {});
	log(`  uploaded version: "${label}"`);
	await sleep();
}

async function addComment(page: Page, text: string) {
	const canvas = page.locator('canvas').first();
	const box = await canvas.boundingBox();
	if (!box) return;
	await page.mouse.click(box.x + box.width * rand(25, 75) / 100, box.y + box.height * rand(20, 70) / 100);
	const ta = page.getByPlaceholder(/write your comment/i);
	if (!(await ta.isVisible().catch(() => false))) return;
	await ta.fill(text);
	await page.getByRole('button', { name: /add comment/i }).click();
	log(`  comment: "${text.slice(0, 40)}"`);
	await sleep();
}

async function requestChanges(page: Page, note: string) {
	await page.getByRole('button', { name: /request changes/i }).click();
	const dialog = page.getByRole('dialog');
	await dialog.getByPlaceholder(/describe the changes/i).fill(note);
	await dialog.getByRole('button', { name: /request changes/i }).click();
	log('  review: requested changes');
	await sleep();
}

async function approve(page: Page) {
	await page.getByRole('button', { name: /^approve$/i }).click();
	log('  review: approved');
	await sleep();
}

async function reject(page: Page, note: string) {
	await page.getByRole('button', { name: /^reject$/i }).click();
	const dialog = page.getByRole('dialog');
	const ta = dialog.getByRole('textbox').first();
	if (await ta.isVisible().catch(() => false)) await ta.fill(note);
	await dialog.getByRole('button', { name: /^reject$/i }).click();
	log('  review: rejected');
	await sleep();
}

async function runRound(page: Page, n: number) {
	const assignee = pick(EMPLOYEES);
	const others = EMPLOYEES.filter((e) => e.email !== assignee.email);
	const stamp = `${Date.now().toString().slice(-6)}`;
	const wsName = `${pick(WS_NOUNS)} ${stamp}`;
	const issueTitle = `Review ${pick(DOC_TOPICS)} ${stamp}`;

	// 1) Manager sets up the workspace + issue.
	await login(page, MANAGER);
	await createWorkspace(page, wsName);
	await addMembers(page, EMPLOYEES.map((e) => e.name));
	const issueUrl = await createIssue(page, issueTitle, assignee.name);
	await logout(page, MANAGER);

	// 2) Assignee uploads the first version + comments.
	await login(page, assignee);
	await page.goto(issueUrl);
	await sleep();
	await uploadVersion(page, `${issueTitle} v1`);
	await addComment(page, pick(COMMENTS));
	await logout(page, assignee);

	// 3) A colleague drops a comment.
	const colleague = pick(others);
	await login(page, colleague);
	await page.goto(issueUrl);
	await sleep();
	await addComment(page, pick(COMMENTS));
	await logout(page, colleague);

	// 4) Manager reviews. Half the time: request changes -> new version -> approve.
	//    Otherwise: approve (or occasionally reject) the first version directly.
	const path = Math.random();
	await login(page, MANAGER);
	await page.goto(issueUrl);
	await sleep();
	if (path < 0.6) {
		await requestChanges(page, 'Please revise per the comments, then re-upload.');
		await logout(page, MANAGER);

		await login(page, assignee);
		await page.goto(issueUrl);
		await sleep();
		await uploadVersion(page, `${issueTitle} v2`);
		await addComment(page, 'Updated per review feedback.');
		await logout(page, assignee);

		await login(page, MANAGER);
		await page.goto(issueUrl);
		await sleep();
		await approve(page);
	} else if (path < 0.85) {
		await approve(page);
	} else {
		await reject(page, 'Does not meet requirements; resubmit a new draft.');
	}
	await logout(page, MANAGER);
	log(`round ${n}/${ROUNDS} complete`);
}

async function main() {
	log(`traffic generator -> ${BASE_URL} (${ROUNDS} rounds, headless=${HEADLESS})`);
	const browser = await chromium.launch({ headless: HEADLESS });
	const context = await browser.newContext({ baseURL: BASE_URL });
	const page = await context.newPage();
	page.setDefaultTimeout(20000);

	for (let n = 1; n <= ROUNDS; n++) {
		log(`--- round ${n}/${ROUNDS} ---`);
		try {
			await runRound(page, n);
		} catch (err) {
			log(`round ${n} error (continuing):`, (err as Error).message);
			await page.context().clearCookies().catch(() => {});
		}
		await sleep(1500, 4000);
	}

	await browser.close();
	log('done');
}

main().catch((e) => {
	console.error(e);
	process.exit(1);
});
