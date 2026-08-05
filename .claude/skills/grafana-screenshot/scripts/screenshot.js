// usage: NODE_PATH=/opt/node22/lib/node_modules node screenshot.js <url> <out.png> [<url> <out.png> ...]
const { chromium } = require('playwright');

// Grafana の UI はポーリングし続けるので networkidle は永遠に来ない。パネルのデータ取得だけを
// 見て、それが止まってから撮る。
const DATA_REQUEST = /\/api\/(ds\/query|prometheus|alertmanager|frames)/;
const QUIET_MS = 750;
const FLOOR_MS = 3000;
const CAP_MS = 15000;

async function shoot(page, url, out) {
  let lastData = Date.now();
  const track = (req) => {
    if (DATA_REQUEST.test(req.url())) lastData = Date.now();
  };
  page.on('requestfinished', track);
  await page.goto(url, { waitUntil: 'domcontentloaded' });

  const start = Date.now();
  for (;;) {
    const elapsed = Date.now() - start;
    if (elapsed >= CAP_MS) break;
    // クエリが飛び始める前に「静か」と誤判定しないよう、最初の数秒は無条件で待つ。
    if (elapsed >= FLOOR_MS && Date.now() - lastData >= QUIET_MS) break;
    await page.waitForTimeout(250);
  }
  page.off('requestfinished', track);
  await page.screenshot({ path: out, fullPage: true });
}

(async () => {
  const args = process.argv.slice(2);
  // ホストの locale が POSIX のままだと Grafana の bootstrap が Intl で落ち、
  // "failed to load its application files" の画面しか撮れない。
  const browser = await chromium.launch({ args: ['--lang=en-US'] });
  const page = await browser.newPage({
    viewport: { width: 1500, height: 1000 },
    locale: 'en-US',
    // 等倍だと文字が潰れて読めないスクショになる。
    deviceScaleFactor: 2,
  });
  for (let i = 0; i + 1 < args.length; i += 2) {
    await shoot(page, args[i], args[i + 1]);
  }
  await browser.close();
})();
