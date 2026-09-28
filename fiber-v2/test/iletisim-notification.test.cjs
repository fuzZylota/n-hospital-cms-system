const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

test('successful contact form uses HTTP success without a WebSocket producer', async () => {
    const source = fs.readFileSync(path.join(__dirname, '..', 'static', 'js', 'frontend', 'iletisim.js'), 'utf8');
    let ready, submit, requests = 0, sockets = 0, resets = 0;
    const status = { innerHTML: '' };
    const closeButton = { removeEventListener() {}, addEventListener() {} };
    const overlay = {};
    const modal = {
        style: {},
        querySelector(selector) {
            if (selector === '.randevu-modal__close') return closeButton;
            if (selector === '.randevu-modal__overlay') return overlay;
            return null;
        }
    };
    let shown = false;
    const form = {
        addEventListener(event, callback) { if (event === 'submit') submit = callback; },
        querySelector(selector) { return selector === '#kvkk_approval' ? { checked: true } : null; },
        reset() { resets++; }
    };
    const document = {
        body: { style: {}, appendChild() { shown = true; } },
        activeElement: null,
        addEventListener(event, callback) { if (event === 'DOMContentLoaded') ready = callback; },
        removeEventListener() {},
        createElement() { return modal; },
        getElementById(id) {
            if (id === 'contactForm') return form;
            if (id === 'contact-modal') return shown ? modal : null;
            if (id === 'contact-status') return status;
            return null;
        },
        querySelector() { return null; }
    };
    class FormData {
        get() { return 'synthetic'; }
    }
    vm.runInNewContext(source, {
        document, FormData, setTimeout() {},
        async fetch(url, options) {
            requests++;
            assert.equal(url, '/backend/add-contact-request');
            assert.equal(options.method, 'POST');
            return { async json() { return { status: 201, crid: '42' }; } };
        },
        WebSocket: class { constructor() { sockets++; } }
    });
    ready();
    await submit({ preventDefault() {} });
    assert.equal(requests, 1);
    assert.equal(sockets, 0);
    assert.equal(resets, 1);
    assert.match(status.innerHTML, /Mesajınız başarıyla gönderildi/);
});
