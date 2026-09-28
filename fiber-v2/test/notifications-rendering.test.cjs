const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const listeners = {};
let htmlWrites;

class Element {
    constructor(tagName) {
        this.tagName = tagName.toUpperCase();
        this.children = [];
        this.listeners = {};
        this.style = {};
        this.textContent = '';
        this.className = '';
    }
    set innerHTML(value) { htmlWrites.push(value); }
    get firstChild() { return this.children[0]; }
    appendChild(child) { this.children.push(child); return child; }
    insertBefore(child) { this.children.unshift(child); return child; }
    querySelector() { return null; }
    setAttribute(name, value) { this[name] = value; }
    addEventListener(name, listener) { this.listeners[name] = listener; }
    dispatch(name, event = {}) { this.listeners[name]?.(event); }
}

global.document = {
    createElement: tag => new Element(tag),
    getElementById: () => null,
    addEventListener: (name, listener) => { listeners[name] = listener; }
};
global.window = { location: { origin: 'https://hospital.example', protocol: 'https:', host: 'hospital.example', href: 'https://hospital.example/panel' } };
const NotificationHandler = require('../static/js/panel/notifications.js');

function render(payload) {
    htmlWrites = [];
    const handler = Object.create(NotificationHandler.prototype);
    handler.notificationList = new Element('div');
    handler.addNotificationToDropdown(payload);
    return handler.notificationList.firstChild;
}

function child(parent, className) {
    return parent.children.find(node => node.className.split(' ').includes(className));
}

test('HTML and event attributes stay text; type maps to a fixed class', () => {
    const payload = '<img src=x onerror=alert(1)><svg onload=alert(2)>';
    const item = render({ message: payload, notification_type: 'info" onclick="alert(3)' });
    assert.deepEqual(htmlWrites, []);
    const icon = child(item, 'notification-icon');
    assert.equal(icon.className, 'notification-icon info');
    assert.equal(child(child(item, 'notification-content'), 'notification-text').textContent, payload);
    assert.equal(child(child(item, 'notification-content'), 'notification-time').textContent, 'Az önce');
    assert.equal(child(icon, 'fas').className, 'fas fa-info-circle');
});

test('valid panel link opens; unsafe links do not create clickable navigation', () => {
    for (const link of ['javascript:alert(1)', '//evil.example/panel', 'https://evil.example/panel', 'https://hospital.example/panel', '/panel/../admin', '/panel%2fadmin', '/panel/%zz', '/panel\\evil.example']) {
        window.location.href = 'https://hospital.example/panel';
        const item = render({ message: 'Yapay bildirim', request_link: link });
        item.dispatch('click');
        assert.equal(window.location.href, 'https://hospital.example/panel', link);
        assert.equal(item.listeners.click, undefined, link);
    }
    const item = render({ message: 'Yapay bildirim', notification_type: 'success', request_link: '/panel/randevu-talepleri/id?notification=true' });
    assert.equal(child(item, 'notification-icon').className, 'notification-icon success');
    item.dispatch('click');
    assert.equal(window.location.href, '/panel/randevu-talepleri/id?notification=true');
    window.location.href = 'https://hospital.example/panel';
    let prevented = false;
    item.dispatch('keydown', { key: 'Enter', preventDefault: () => { prevented = true; } });
    assert.equal(prevented, true);
    assert.equal(window.location.href, '/panel/randevu-talepleri/id?notification=true');
});

test('rrid is compared as data, never interpolated into a selector', () => {
    const row = { getAttribute: () => '42' };
    const selectors = [];
    document.querySelectorAll = selector => {
        selectors.push(selector);
        return selector === 'tr[data-id]' ? [row] : [];
    };
    const handler = Object.create(NotificationHandler.prototype);
    assert.equal(handler.findRequestElement('tr[data-id]', '42'), row);
    assert.equal(handler.findRequestElement('tr[data-id]', '42"], body'), null);
    assert.deepEqual(selectors, ['tr[data-id]', 'tr[data-id]']);
});

test('WebSocket notification keeps count and badge', () => {
    const handler = Object.create(NotificationHandler.prototype);
    handler.notificationList = new Element('div');
    handler.notificationCount = new Element('span');
    handler.notificationCount.textContent = '2';
    handler.notificationBtn = { classList: { contains: () => false, add: value => { handler.badge = value; } } };
    htmlWrites = [];
    handler.handleNotification(JSON.stringify({ message: 'Yapay bildirim', notification_type: 'warning' }));
    assert.equal(handler.notificationCount.textContent, 3);
    assert.equal(handler.badge, 'has-notification');
});

test('one saved request event updates the row and notification counter once', () => {
    const handler = Object.create(NotificationHandler.prototype);
    handler.notificationList = new Element('div');
    handler.notificationCount = new Element('span');
    handler.notificationCount.textContent = '0';
    handler.notificationBtn = { classList: { contains: () => false, add: value => { handler.badge = value; } } };
    let rows = 0;
    window._addNewRandevuRow = () => { rows++; };
    htmlWrites = [];
    handler.handleNotification(JSON.stringify({
        type: 'new_randevu_talebi', rrid: 'saved-1', sid: 'branch-a',
        message: 'Yapay şikâyet', notification_message: 'Yapay talep alındı',
        request_link: '/panel/randevu-talepleri/saved-1?notification=true'
    }));
    assert.equal(rows, 1);
    assert.equal(handler.notificationCount.textContent, 1);
    assert.equal(handler.notificationList.children.length, 1);
    assert.equal(child(child(handler.notificationList.firstChild, 'notification-content'), 'notification-text').textContent, 'Yapay talep alındı');
});

test('saved request is the only appointment notification producer', () => {
    const root = path.resolve(__dirname, '..');
    const requestSource = fs.readFileSync(path.join(root, 'controllers/post/randevular/randevular.go'), 'utf8');
    const socketSource = fs.readFileSync(path.join(root, 'controllers/post/post.go'), 'utf8');
    const formSource = fs.readFileSync(path.join(root, 'static/js/frontend/fast-randevu.js'), 'utf8');
    const requestHandler = requestSource.split('func AddRandevuRequest(')[1].split('func DeleteRandevuRequest(')[0];
    const socketHandler = socketSource.split('func NotificationWebsocket(')[1];
    assert.match(requestHandler, /Table\("notifications"\)/, 'mutation must persist notification');
    assert.doesNotMatch(socketHandler, /event == notificationws\.Appointment/, 'client appointment frame must not produce notification');
    assert.doesNotMatch(formSource, /new WebSocket\([^\n]*['"]randevu['"]/, 'quick form must not send a producer frame');
});
