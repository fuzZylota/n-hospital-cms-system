const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

test('successful job application uses HTTP success without a WebSocket producer', async () => {
    const source = fs.readFileSync(path.join(__dirname, '..', 'static', 'js', 'frontend', 'insert-job-application.js'), 'utf8');
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
            if (id === 'jobApplicationForm') return form;
            if (id === 'job-application-modal') return shown ? modal : null;
            if (id === 'job-application-status') return status;
            return null;
        }
    };
    class FormData {
        get(name) { return name === 'cv_file' ? { toString: () => 'synthetic.pdf' } : 'synthetic'; }
    }
    vm.runInNewContext(source, {
        document, FormData, setTimeout() {},
        async fetch(url, options) {
            requests++;
            assert.equal(url, '/backend/add-job-application');
            assert.equal(options.method, 'POST');
            return { async json() { return { status: 201, jaid: 'saved-7' }; } };
        },
        WebSocket: class { constructor() { sockets++; } }
    });
    ready();
    await submit({ preventDefault() {} });
    assert.equal(requests, 1);
    assert.equal(sockets, 0);
    assert.equal(resets, 1);
    assert.match(status.innerHTML, /Başvurunuz başarıyla gönderildi/);
});
