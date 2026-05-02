import { browser } from '$app/environment';
import { getPdfjs } from '$lib/pdf/pdfjs';

export const load = () => {
	if (browser) void getPdfjs();
	return {};
};
