const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

const script = fs.readFileSync(path.join(__dirname, '..', 'static', 'js', 'panel', 'homepage-content-duzenle.js'), 'utf8');

function panelWithUploadResponse(responseData) {
    const onReady = [];
    const fields = {
        content_html: { value: '' },
        content_css: { value: '' },
        bannerPreviewImg: { src: '' },
        mobileBannerPreviewImg: { src: '' },
        bannerPreviewPath: { textContent: '' },
        mobileBannerPreviewPath: { textContent: '' },
        bannerPreview: { style: {} },
        mobileBannerPreview: { style: {} },
        bannerCurrentImg: { style: {} },
        mobileBannerCurrentImg: { style: {} },
        bannerUploadArea: { style: {} },
        mobileBannerUploadArea: { style: {} }
    };
    const inputs = {};
    for (const id of ['bannerFileInput', 'mobileBannerFileInput']) {
        const replacement = {
            files: [{ name: 'synthetic.txt' }],
            addEventListener(event, callback) { if (event === 'change') this.onChange = callback; }
        };
        inputs[id] = {
            cloneNode() { return replacement; },
            parentNode: { replaceChild() { inputs[id] = replacement; } }
        };
    }
    const document = {
        addEventListener(event, callback) { if (event === 'DOMContentLoaded') onReady.push(callback); },
        getElementById(id) { return inputs[id] || fields[id] || null; },
        querySelector() { return null; },
        querySelectorAll() { return []; }
    };
    const requests = [];
    const alerts = [];
    const context = {
        document,
        window: {},
        FormData: class { append() {} },
        fetch: async (url) => {
            requests.push(url);
            return { text: async () => JSON.stringify(responseData) };
        },
        alert: (message) => alerts.push(message),
        console: { error() {} }
    };
    vm.runInNewContext(script, context);
    for (const ready of onReady.slice(1)) ready();
    return { fields, inputs, requests, alerts };
}

test('homepage banner fields use the add-file site URL for preview and saved paths', async () => {
    const page = panelWithUploadResponse({ status: 201, data: [{ name: 'banner image.txt', url: '/uploads/banner%20image.txt' }] });
    await page.inputs.bannerFileInput.onChange({ target: page.inputs.bannerFileInput });
    await page.inputs.mobileBannerFileInput.onChange({ target: page.inputs.mobileBannerFileInput });
    assert.deepEqual(page.requests, ['/backend/add-file', '/backend/add-file']);
    assert.equal(page.fields.content_html.value, 'uploads/banner%20image.txt');
    assert.equal(page.fields.content_css.value, 'uploads/banner%20image.txt');
    assert.equal(page.fields.bannerPreviewImg.src, '/uploads/banner%20image.txt');
    assert.equal(page.fields.mobileBannerPreviewImg.src, '/uploads/banner%20image.txt');
    assert.equal(page.fields.bannerPreviewPath.textContent, 'uploads/banner%20image.txt');
    assert.equal(page.fields.mobileBannerPreviewPath.textContent, 'uploads/banner%20image.txt');
    assert.equal(page.alerts.length, 2);
});
