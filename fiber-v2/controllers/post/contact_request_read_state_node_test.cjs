// Execute the existing panel methods with synthetic DOM/fetch objects only.
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

function panel(file, className, reply, current = false) {
    const calls = [], alerts = [], loading = [], timers = [], mutations = [];
    let reloads = 0, statistics = 0, closed = 0;
    const badge = () => ({ className: '', innerHTML: '' });
    const rowBadge = badge(), cardBadge = badge();
    const element = (kind, readBadge) => ({
        classList: {
            add: value => mutations.push([kind, 'add', value]),
            remove: value => mutations.push([kind, 'remove', value])
        },
        querySelector: selector => {
            assert.equal(selector, '.read-badge');
            return readBadge;
        }
    });
    const row = element('row', rowBadge), card = element('card', cardBadge);
    const button = {
        getAttribute: name => ({ 'data-id': '7', 'data-is-read': String(current) })[name]
    };
    const context = vm.createContext({
        window: { location: { reload: () => reloads++ } },
        document: {
            addEventListener() {}, // Do not start unrelated constructor/UI work.
            getElementById: id => {
                assert.equal(id, 'toggleReadStatusBtn');
                return button;
            },
            querySelector: selector => {
                if (selector === 'tr[data-id="7"]') return row;
                assert.equal(selector, '.contact-request-card[data-id="7"]');
                return card;
            }
        },
        fetch: async (url, options) => {
            calls.push({ url, method: options.method, headers: options.headers, body: JSON.parse(options.body) });
            if (reply === 'network') throw new Error('synthetic network failure');
            return {
                status: reply === 201 ? 200 : reply,
                json: async () => {
                    if (reply === 'invalid-json') throw new Error('synthetic invalid JSON');
                    return { status: reply, message: 'Synthetic reply' };
                }
            };
        },
        console: { error() {} },
        setTimeout: (fn, delay) => timers.push({ fn, delay })
    });
    const source = fs.readFileSync(path.join(__dirname, '../../static/js/panel', file), 'utf8');
    vm.runInContext(source + `\nglobalThis.TestManager = ${className};`, context);
    const manager = Object.create(context.TestManager.prototype);
    manager.showAlert = (message, type) => alerts.push({ message, type });
    manager.setButtonLoading = (buttonArg, value) => {
        assert.equal(buttonArg, button);
        loading.push(value);
    };
    manager.closeAllDropdowns = () => closed++;
    manager.updateStatistics = () => statistics++;
    return {
        manager, calls, alerts, loading, timers, mutations, rowBadge, cardBadge,
        counts: () => ({ reloads, statistics, closed })
    };
}

function assertRequest(f, desired) {
    assert.equal(f.calls.length, 1);
    assert.equal(f.calls[0].url, '/backend/contact-request/7/set-as-read');
    assert.equal(f.calls[0].method, 'POST');
    assert.equal(f.calls[0].headers['Content-Type'], 'application/json');
    assert.equal(f.calls[0].headers['X-Requested-With'], 'XMLHttpRequest');
    assert.deepEqual(f.calls[0].body, { is_read: desired });
}

for (const desired of [true, false]) {
    test(`detail sends target ${desired} and handles HTTP 200 / JSON 201`, async () => {
        const f = panel('contact-request.js', 'ContactRequestViewManager', 201, !desired);
        await f.manager.toggleReadStatus();
        assertRequest(f, desired);
        assert.deepEqual(f.loading, [true, false]);
        assert.deepEqual(f.alerts, [{ message: 'Synthetic reply', type: 'success' }]);
        assert.equal(f.timers.length, 1);
        assert.equal(f.timers[0].delay, 1500);
        assert.equal(f.counts().reloads, 0);
        f.timers[0].fn();
        assert.equal(f.counts().reloads, 1);
    });

    test(`list sends target ${desired} and updates row/card on JSON 201`, async () => {
        const f = panel('contact-requests.js', 'ContactRequestsListManager', 201);
        await f.manager[desired ? 'markAsRead' : 'markAsUnread']('7');
        assertRequest(f, desired);
        assert.equal(f.alerts.length, 1);
        assert.equal(f.alerts[0].type, 'success');
        assert.deepEqual(f.mutations, [
            ['row', desired ? 'remove' : 'add', 'unread-row'],
            ['card', desired ? 'remove' : 'add', 'unread-card']
        ]);
        for (const badge of [f.rowBadge, f.cardBadge]) {
            assert.equal(badge.className, desired ? 'read-badge read' : 'read-badge unread');
            assert.ok(badge.innerHTML.includes(desired ? 'Okundu' : 'Okunmamış'));
        }
        assert.deepEqual(f.counts(), { reloads: 0, statistics: 1, closed: 1 });
    });
}

for (const reply of [400, 403, 404, 503, 200, 'invalid-json', 'network']) {
    test(`detail rejection/failure ${reply} restores button without reload`, async () => {
        const f = panel('contact-request.js', 'ContactRequestViewManager', reply);
        await f.manager.toggleReadStatus();
        assertRequest(f, true);
        assert.deepEqual(f.loading, [true, false]);
        assert.equal(f.alerts.length, 1);
        assert.equal(f.alerts[0].type, 'error');
        assert.equal(f.timers.length, 0);
        assert.equal(f.counts().reloads, 0);
    });
    for (const desired of [true, false]) {
        test(`list ${desired} rejection/failure ${reply} leaves DOM/statistics unchanged`, async () => {
            const f = panel('contact-requests.js', 'ContactRequestsListManager', reply);
            await f.manager[desired ? 'markAsRead' : 'markAsUnread']('7');
            assertRequest(f, desired);
            assert.equal(f.alerts.length, 1);
            assert.equal(f.alerts[0].type, 'error');
            assert.deepEqual(f.mutations, []);
            assert.equal(f.rowBadge.className, '');
            assert.equal(f.cardBadge.className, '');
            assert.deepEqual(f.counts(), { reloads: 0, statistics: 0, closed: 0 });
        });
    }
}
