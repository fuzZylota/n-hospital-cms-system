// Kalite taraması: sayfa başına 390/1280px ekran görüntüsü, axe-core ihlalleri,
// yatay taşma, konsol/ağ hataları. Kullanım: node qa.mjs [baseURL] [--panel]
// Çıktı: tools/qa/out/<tarih>/ (PNG + report.json)
import { createRequire } from 'node:module';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_PATH || '/opt/node22/lib/node_modules/playwright');
const axeSource = fs.readFileSync(path.join(here, 'node_modules/axe-core/axe.min.js'), 'utf8');

const args = process.argv.slice(2);
const base = (args.find(a => a.startsWith('http')) || 'http://127.0.0.1:2000').replace(/\/$/, '');
const withPanel = args.includes('--panel');
const PUBLIC = ['/', '/randevu', '/iletisim', '/ulasim', '/tibbi-birimler', '/haberler', '/foto-galeri', '/video-galeri', '/giris'];
const PANEL = ['/panel', '/panel/randevu-talepleri', '/panel/randevular'];
const VIEWPORTS = [{ name: 'mobil', width: 390, height: 844 }, { name: 'masaustu', width: 1280, height: 900 }];

const stamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);
const out = path.join(here, 'out', stamp);
fs.mkdirSync(out, { recursive: true });

const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH || undefined });
const report = [];

async function login(context) {
  const page = await context.newPage();
  await page.goto(base + '/giris');
  await page.fill('#login_email_or_phone', process.env.QA_USER || 'admin@nhospital.com');
  await page.fill('#login_password', process.env.QA_PASS || 'DevOnly-Nivgoz-123');
  await Promise.all([page.waitForURL('**/panel**'), page.click('#nvLoginForm [type=submit]')]);
  await page.close();
}

for (const vp of VIEWPORTS) {
  const context = await browser.newContext({ viewport: { width: vp.width, height: vp.height }, locale: 'tr-TR' });
  if (withPanel) await login(context);
  for (const url of [...PUBLIC, ...(withPanel ? PANEL : [])]) {
    await new Promise(r => setTimeout(r, Number(process.env.QA_DELAY_MS || 1500))); // uygulamadaki 100 istek/dk sınırını aşmamak için
    const page = await context.newPage();
    const consoleErrors = [], failed = [];
    page.on('console', m => { if (m.type() === 'error') consoleErrors.push(m.text().slice(0, 200)); });
    page.on('requestfailed', r => failed.push(r.url().slice(0, 150)));
    page.on('response', r => { if (r.status() >= 400) failed.push(`${r.status()} ${r.url().slice(0, 150)}`); });
    let status = null;
    try {
      const resp = await page.goto(base + url, { waitUntil: 'networkidle', timeout: 30000 });
      status = resp && resp.status();
    } catch (e) { consoleErrors.push('goto: ' + e.message.slice(0, 120)); }
    await page.evaluate(axeSource).catch(() => {});
    const axe = await page.evaluate(async () => {
      if (!window.axe) return null;
      const r = await window.axe.run(document, { runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21aa', 'wcag22aa'] } });
      return r.violations.map(v => ({ id: v.id, impact: v.impact, nodes: v.nodes.length, help: v.help }));
    }).catch(() => null);
    const layout = await page.evaluate(() => ({
      overflowX: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1,
      h1: document.querySelectorAll('h1').length,
      bodyFont: getComputedStyle(document.body).fontSize,
      smallTargets: [...document.querySelectorAll('a,button,input,select,textarea,[role=button]')]
        .filter(e => { const r = e.getBoundingClientRect(); return r.width > 0 && r.height > 0 && (r.width < 44 || r.height < 44); }).length,
    })).catch(() => null);
    const file = `${vp.name}_${url === '/' ? 'anasayfa' : url.replace(/\W+/g, '-').replace(/^-|-$/g, '')}.png`;
    await page.screenshot({ path: path.join(out, file), fullPage: true }).catch(() => {});
    report.push({ viewport: vp.name, url, status, axe, layout, consoleErrors, failed: failed.slice(0, 8), screenshot: file });
    await page.close();
  }
  await context.close();
}
await browser.close();

fs.writeFileSync(path.join(out, 'report.json'), JSON.stringify(report, null, 2));
const sev = n => report.reduce((a, r) => a + (r.axe || []).filter(v => v.impact === n).reduce((x, v) => x + v.nodes, 0), 0);
console.log(`Rapor: ${out}`);
console.log(`Sayfa x viewport: ${report.length} | axe kritik=${sev('critical')} ciddi=${sev('serious')} orta=${sev('moderate')} hafif=${sev('minor')}`);
for (const r of report) {
  const v = (r.axe || []).filter(x => ['critical', 'serious'].includes(x.impact)).map(x => `${x.id}(${x.nodes})`).join(',');
  console.log(`${String(r.status).padEnd(4)} ${r.viewport.padEnd(9)} ${r.url.padEnd(20)} axe:${r.axe ? (v || 'temiz') : 'ölçülemedi'} taşma:${r.layout?.overflowX ? 'EVET' : 'hayır'} h1:${r.layout?.h1} <44px:${r.layout?.smallTargets} konsol:${r.consoleErrors.length} ağ:${r.failed.length}`);
}
