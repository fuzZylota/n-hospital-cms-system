const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

const script = fs.readFileSync(path.join(__dirname, '..', 'static', 'js', 'panel', 'randevu-duzenle.js'), 'utf8');
const legacyKey = 'randevuEditAutoSave';
const sample = {
    patient_first_name: 'YapayAd',
    patient_last_name: 'YapaySoyad',
    patient_phone: '00000000000',
    patient_tc_kimlik: '11111111111',
    complaint: 'YapaySikayet',
    notes: 'YapayNot'
};

function storage(entries = {}) {
    const values = new Map(Object.entries(entries));
    const writes = [];
    const removals = [];
    return {
        values, writes, removals,
        getItem(key) { return values.get(key) ?? null; },
        setItem(key, value) { writes.push(key); values.set(key, String(value)); },
        removeItem(key) { removals.push(key); values.delete(key); }
    };
}

function noPatientStorage(local, session) {
    for (const store of [local, session]) {
        assert.deepEqual(store.writes, []);
        assert.equal(store.getItem(legacyKey), null);
        for (const value of store.values.values()) {
            for (const patientValue of Object.values(sample)) {
                assert.equal(String(value).includes(patientValue), false);
            }
        }
    }
}

function makePage(local, session, initialFields = sample) {
    const documentEvents = new Map();
    const windowEvents = new Map();
    const formEvents = new Map();
    const saveEvents = new Map();
    const fields = Object.entries(initialFields).map(([name, value]) => ({
        name, value, type: 'text', checked: false,
        addEventListener() {}
    }));
    const form = {
        fields,
        querySelectorAll() { return fields; },
        addEventListener(name, callback) { formEvents.set(name, callback); }
    };
    const saveBtn = { addEventListener(name, callback) { saveEvents.set(name, callback); } };
    const document = {
        head: { appendChild() {} },
        body: { appendChild() {} },
        createElement() { return { style: {}, textContent: '' }; },
        getElementById(id) {
            return id === 'randevuEditForm' ? form : id === 'saveRandevuBtn' ? saveBtn : null;
        },
        querySelectorAll(selector) { return selector === 'input, select, textarea' ? fields : []; },
        addEventListener(name, callback) { documentEvents.set(name, callback); }
    };
    const window = {
        randevuData: { id: 'synthetic-appointment' },
        location: { href: '' },
        history: { back() {} },
        addEventListener(name, callback) { windowEvents.set(name, callback); }
    };
    const requests = [];
    const alerts = [];
    const timers = [];
    let response = { status: 500, message: 'Yapay hata' };
    class FakeFormData {
        constructor(target) { this.target = target; }
        *entries() {
            for (const field of this.target.fields) yield [field.name, field.value];
        }
    }
    const context = vm.createContext({
        document, window, localStorage: local, sessionStorage: session,
        FormData: FakeFormData,
        fetch: async (url, options) => {
            requests.push({ url, options });
            return { json: async () => response };
        },
        setTimeout(callback, delay) { timers.push({ callback, delay }); return timers.length; },
        clearTimeout() {},
        console
    });
    vm.runInContext(script + '\nRandevuEditManager.prototype.setupDropdowns = function() {}; RandevuEditManager.prototype.setupTimeInput = function() {};', context);
    documentEvents.get('DOMContentLoaded')();
    const manager = window.randevuEditManager;
    manager.validateAllForms = () => true;
    manager.showAlert = (message, type) => alerts.push({ message, type });
    manager.setButtonLoading = () => {};
    return {
        manager, fields, formEvents, saveEvents, documentEvents, windowEvents, window,
        requests, alerts, timers,
        setResponse(value) { response = value; }
    };
}

test('editing and leaving keep synthetic patient fields out of browser storage', () => {
    const local = storage({ [legacyKey]: JSON.stringify(sample), otherAppSetting: 'preserve-me' });
    const session = storage({ otherSessionSetting: 'preserve-this-too' });
    const page = makePage(local, session);
    assert.deepEqual(local.removals, [legacyKey]);
    assert.equal(local.getItem('otherAppSetting'), 'preserve-me');
    assert.equal(session.getItem('otherSessionSetting'), 'preserve-this-too');

    page.fields[0].value = 'YapayYeniAd';
    page.formEvents.get('input')();
    assert.equal(page.manager.hasUnsavedChanges(), true);
    assert.deepEqual(page.timers, []);
    const leave = { preventDefault() { this.prevented = true; } };
    page.windowEvents.get('beforeunload')(leave);
    assert.equal(leave.prevented, true);
    noPatientStorage(local, session);

    const otherAccountPage = makePage(local, session, { ...sample, patient_first_name: 'DigerYapayAd' });
    assert.equal(otherAccountPage.fields[0].value, 'DigerYapayAd');
    assert.deepEqual(local.removals, [legacyKey, legacyKey]);
    noPatientStorage(local, session);
});

test('failed save keeps edits in the open form; Ctrl/Cmd+S retries without storage', async () => {
    const local = storage({ otherAppSetting: 'preserve-me' });
    const session = storage();
    const page = makePage(local, session);
    page.fields[0].value = 'YapayDuzeltilmisAd';
    await page.manager.handleSave({ preventDefault() {} });
    assert.equal(page.requests.length, 1);
    assert.equal(JSON.parse(page.requests[0].options.body).patient_first_name, 'YapayDuzeltilmisAd');
    assert.equal(page.fields[0].value, 'YapayDuzeltilmisAd');
    assert.equal(page.manager.isSubmitting, false);
    assert.equal(page.window.location.href, '');
    assert.equal(page.alerts.at(-1).type, 'error');
    noPatientStorage(local, session);

    page.setResponse({ status: 201 });
    for (const shortcut of [{ ctrlKey: true }, { metaKey: true }]) {
        let prevented = false;
        page.documentEvents.get('keydown')({
            ...shortcut, key: 's', preventDefault() { prevented = true; }
        });
        await new Promise(resolve => setImmediate(resolve));
        assert.equal(prevented, true);
    }
    assert.equal(page.requests.length, 3);
    assert.equal(page.alerts.at(-1).type, 'success');
    assert.equal(page.timers.some(timer => timer.delay === 1500), true);
    assert.equal(local.getItem('otherAppSetting'), 'preserve-me');
    noPatientStorage(local, session);
});

test('save button retains the validation guard without storing patient fields', async () => {
    const local = storage();
    const session = storage();
    const page = makePage(local, session);
    let validationCalls = 0;
    page.manager.validateAllForms = () => { validationCalls++; return false; };
    let prevented = false;
    page.saveEvents.get('click')({ preventDefault() { prevented = true; } });
    await new Promise(resolve => setImmediate(resolve));
    assert.equal(prevented, true);
    assert.equal(validationCalls, 1);
    assert.equal(page.requests.length, 0);
    assert.equal(page.alerts.at(-1).type, 'error');
    noPatientStorage(local, session);
});

test('blocked localStorage cleanup does not prevent in-memory editing', () => {
    const local = storage();
    local.removeItem = () => { throw new Error('storage unavailable'); };
    const page = makePage(local, storage());
    page.fields[0].value = 'YapayYeniAd';
    page.formEvents.get('input')();
    assert.equal(page.manager.hasUnsavedChanges(), true);
});
