import type { TestRunnerConfig } from "@storybook/test-runner";
import { toMatchImageSnapshot } from "jest-image-snapshot";

const THEMES = ["light", "dark"] as const;

const config: TestRunnerConfig = {
  setup() {
    expect.extend({ toMatchImageSnapshot });
  },
  async postVisit(page, context) {
    // CI と手元でのレンダリング差を抑えるため。
    await page.emulateMedia({ reducedMotion: "reduce" });
    // useCart は mount 後の effect で ready になる (localStorage 読み)。それ以前は PageLoading。
    await page.waitForTimeout(200);
    const root = page.locator("#storybook-root");

    for (const theme of THEMES) {
      await page.evaluate((t) => {
        document.documentElement.classList.toggle("dark", t === "dark");
      }, theme);
      // FontFaceSet はそのまま返すと直列化に失敗するため void に畳む。
      await page.evaluate(() => document.fonts.ready.then(() => undefined));

      const image = await root.screenshot();
      expect(image).toMatchImageSnapshot({
        customSnapshotsDir: `${process.cwd()}/__vrt__/__snapshots__`,
        customSnapshotIdentifier: `${context.id}-${theme}`,
        // 同一イメージ内でもサブピクセル AA は揺れるため僅かな差は許容する。
        failureThreshold: 0.01,
        failureThresholdType: "percent",
      });
    }
  },
};

export default config;
