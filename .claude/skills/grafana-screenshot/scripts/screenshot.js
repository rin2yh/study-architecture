// usage: NODE_PATH=/opt/node22/lib/node_modules node screenshot.js <url> <out.png> [waitMs]
//
// Grafana / Prometheus の UI はポーリングし続けるので networkidle は永遠に来ない。
const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');

// Playwright のブラウザはバージョン付きディレクトリに入るため、決め打ちせず拾う。
function chromiumPath() {
  const root = process.env.PLAYWRIGHT_BROWSERS_PATH || '/opt/pw-browsers';
  const dir = fs.readdirSync(root).find((d) => /^chromium-\d+$/.test(d));
  if (!dir) throw new Error(`chromium not found under ${root}`);
  return path.join(root, dir, 'chrome-linux', 'chrome');
}

(async () => {
  const [url, out, waitMs] = process.argv.slice(2);
  const browser = await chromium.launch({ executablePath: chromiumPath() });
  const page = await browser.newPage({
    viewport: { width: 1500, height: 1000 },
    // Retina 相当。等倍だと文字が潰れて読めないスクショになる。
    deviceScaleFactor: 2,
  });
  await page.goto(url, { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(Number(waitMs) || 9000);
  await page.screenshot({ path: out, fullPage: true });
  await browser.close();
  console.log(out);
})();
