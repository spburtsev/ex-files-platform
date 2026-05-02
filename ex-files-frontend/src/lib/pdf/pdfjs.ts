import { browser } from '$app/environment';
import type * as PdfjsLib from 'pdfjs-dist';

let pdfjsPromise: Promise<typeof PdfjsLib> | null = null;

export function getPdfjs(): Promise<typeof PdfjsLib> {
	if (!browser) {
		return Promise.reject(new Error('pdfjs-dist is browser-only'));
	}
	if (!pdfjsPromise) {
		pdfjsPromise = (async () => {
			const lib = await import('pdfjs-dist');
			lib.GlobalWorkerOptions.workerSrc = new URL(
				'pdfjs-dist/build/pdf.worker.mjs',
				import.meta.url
			).href;
			return lib;
		})();
	}
	return pdfjsPromise;
}
