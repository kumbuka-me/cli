// A generic, opt-in browser-module export host for offline static builds.
// Plugin code is loaded only from the validated plugin package's copied assets.
import { createRequire } from 'node:module';
import { pathToFileURL } from 'node:url';
import path from 'node:path';
import fs from 'node:fs/promises';
import readline from 'node:readline';

const require = createRequire(pathToFileURL(path.resolve(process.cwd(), 'package.json')));
let playwright;
try {
  playwright = require('playwright');
} catch {
  process.stderr.write('Static browser export requires Playwright in the build project: npm install --save-dev playwright && npx playwright install chromium\n');
  process.exit(1);
}

const origin = 'http://kumbuka-static-export.invalid';
let browser;
try {
  browser = await playwright.chromium.launch({
    headless: true,
    ...(process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE } : {}),
  });
} catch (error) {
  process.stderr.write(`Unable to start Chromium for static exports: ${error.message}\n`);
  process.exit(1);
}

async function render(job) {
  const assetDirectory = path.resolve(job.asset_directory);
  const script = job.javascript;
  const theme = job.theme;
  if (theme !== 'light' && theme !== 'dark') throw new Error('Invalid export theme');
  if (!script || path.isAbsolute(script) || script.split('/').includes('..')) {
    throw new Error('Invalid browser module asset path');
  }

  const context = await browser.newContext({ serviceWorkers: 'block', acceptDownloads: false });
  try {
    // All traffic is intercepted; network requests and external assets are refused.
    await context.route('**/*', async (route) => {
      const url = new URL(route.request().url());
      if (url.origin !== origin) {
        await route.abort();
        return;
      }
      if (url.pathname === '/') {
        await route.fulfill({ status: 200, contentType: 'text/html', body: '<!doctype html><html><head></head><body></body></html>' });
        return;
      }
      if (!url.pathname.startsWith('/assets/')) {
        await route.abort();
        return;
      }
      const filename = path.resolve(assetDirectory, decodeURIComponent(url.pathname.slice('/assets/'.length)));
      if (!filename.startsWith(assetDirectory + path.sep)) {
        await route.abort();
        return;
      }
      try {
        const data = await fs.readFile(filename);
        await route.fulfill({ status: 200, contentType: 'text/javascript', body: data });
      } catch {
        await route.fulfill({ status: 404, body: 'asset not found' });
      }
    });
    const page = await context.newPage();
    await page.goto(origin + '/');
    await page.addScriptTag({ url: origin + '/assets/' + script });
    return await page.evaluate(async (input) => {
      const plugin = globalThis.kumbukaPlugin;
      if (typeof plugin?.exportStatic !== 'function') {
        throw new Error('plugin declared static exports without exportStatic(root, context)');
      }
      const root = document.createElement('div');
      root.innerHTML = input.markup;
      document.body.append(root);
      try {
        const result = await plugin.exportStatic(root, { theme: input.theme });
        if (result === null) return null;
        if (!result || typeof result.content !== 'string') {
          throw new Error('plugin static export returned an invalid artifact');
        }
        return result;
      } finally {
        root.remove();
      }
    }, job);
  } finally {
    await context.close();
  }
}

try {
  for await (const line of readline.createInterface({ input: process.stdin, crlfDelay: Infinity })) {
    let response;
    try {
      response = { result: await render(JSON.parse(line)) };
    } catch (error) {
      response = { error: String(error?.message ?? error) };
    }
    process.stdout.write(JSON.stringify(response) + '\n');
  }
} finally {
  await browser.close();
}
