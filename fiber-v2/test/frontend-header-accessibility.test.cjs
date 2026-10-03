const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { test } = require('node:test');

const components = path.join(__dirname, '..', 'static', 'html', 'components');
const header = fs.readFileSync(path.join(components, 'frontend-header.jet'), 'utf8');
const init = fs.readFileSync(path.join(components, 'frontend-header-init.jet'), 'utf8')
  .replace(/^\s*<script>\s*/, '').replace(/\s*<\/script>\s*$/, '');

function tag(source, expression) {
  const match = source.match(expression);
  assert.ok(match, `Missing markup: ${expression}`);
  return match[0];
}

function attribute(markup, name) {
  return markup.match(new RegExp(`\\b${name}="([^"]*)"`))?.[1];
}

function assertMarkup(source) {
  const search = tag(source, /<button\b[^>]*id="nvMobileSearchBtn"[^>]*>/);
  const menu = tag(source, /<button\b[^>]*class="mobile-nav__btn mobile-nav__toggler"[^>]*>/);
  const menuPanel = tag(source, /<div\b[^>]*id="nvMobileNav"[^>]*>/);
  const searchPanel = tag(source, /<div\b[^>]*id="nvMobileSearchBar"[^>]*>/);
  for (const [button, panel, id] of [
    [search, searchPanel, 'nvMobileSearchBar'],
    [menu, menuPanel, 'nvMobileNav'],
  ]) {
    assert.equal(attribute(button, 'type'), 'button');
    assert.ok(attribute(button, 'aria-label'));
    assert.equal(attribute(button, 'aria-controls'), id);
    assert.equal(attribute(button, 'aria-expanded'), 'false');
    assert.match(panel, /\shidden(?:\s|>)/);
  }
  assert.match(source, /<form\b[^>]*action="\/arama"[^>]*method="get"/);
  assert.match(source, /<input\b[^>]*id="nvMobileSearchInput"[^>]*name="q"/);
  assert.match(source, /<button\b[^>]*type="submit"[^>]*aria-label="Ara"/);
  // P2-4: kalıcı birincil CTA tasarım sistemi butonu ve tek ad ("Randevu Talebi Oluştur")
  assert.match(source, /<a href="\/randevu" class="nv-btn nv-btn--primary[^"]*">\s*Randevu Talebi Oluştur\s*<\/a>/);
  // Mobil menü tam ekran modal iletişim kutusu
  assert.match(source, /<div\b[^>]*id="nvMobileNav"[^>]*role="dialog"[^>]*aria-modal="true"/);
  const order = ['kurumsal', 'subeler', 'tibbi_birimler', 'tedkikler', 'diger']
    .map(type => source.indexOf(`HeaderButtons.ButtonType == "${type}"`));
  assert.ok(order.every((position, index) => position >= 0 && (!index || position > order[index - 1])));
  assert.match(source, /href="{{ HeaderButtons.Url }}" target="{{ HeaderButtons.Target }}"/);
  assert.match(source, /href="\/tibbi-birimler"/);
  assert.match(source, /href="\/tetkikler"/);
  return { search, menu };
}

class Element {
  constructor(id, classes = '') {
    this.id = id;
    this.hidden = false;
    this.parent = null;
    this.children = [];
    this.attributes = {};
    this.className = classes;
    const names = new Set(classes.split(' ').filter(Boolean));
    this.classList = {
      add: (...values) => values.forEach(value => names.add(value)),
      remove: (...values) => values.forEach(value => names.delete(value)),
      contains: value => names.has(value),
    };
    this.style = { setProperty() {} };
  }
  append(child) { child.parent = this; this.children.push(child); return child; }
  setAttribute(name, value) { this.attributes[name] = value; }
  getAttribute(name) { return this.attributes[name]; }
  contains(node) {
    for (let current = node; current; current = current.parent) if (current === this) return true;
    return false;
  }
  closest(selector) {
    for (let current = this; current; current = current.parent) {
      if (selector === '.mobile-nav__toggler' && current.classList.contains('mobile-nav__toggler')) return current;
      if (selector === '#nvMobileSearchBtn' && current.id === 'nvMobileSearchBtn') return current;
    }
    return null;
  }
  querySelector(selector) {
    if (selector === 'i') return this.children.find(child => child.id === 'icon') || null;
    if (selector === '.mobile-nav__container a') return this.children.find(child => child.id === 'menuLink') || null;
    if (selector === '.mobile-nav__content button') return this.children.find(child => child.id === 'close') || null;
    return null;
  }
  focus() { this.ownerDocument.activeElement = this; }
}

function fixture() {
  const markup = assertMarkup(header);
  const listeners = {};
  const body = new Element('body');
  const searchButton = new Element('nvMobileSearchBtn');
  const menuButton = new Element('menuButton', 'mobile-nav__toggler');
  const searchPanel = new Element('nvMobileSearchBar');
  const menuPanel = new Element('nvMobileNav');
  const searchInput = searchPanel.append(new Element('nvMobileSearchInput'));
  const submit = searchPanel.append(new Element('submit'));
  const icon = searchButton.append(new Element('icon'));
  const menuLink = menuPanel.append(new Element('menuLink'));
  const close = menuPanel.append(new Element('close', 'mobile-nav__toggler'));
  const outside = new Element('outside');
  searchPanel.hidden = menuPanel.hidden = true;
  for (const [element, source] of [[searchButton, markup.search], [menuButton, markup.menu]]) {
    for (const name of ['aria-controls', 'aria-expanded', 'aria-label']) element.setAttribute(name, attribute(source, name));
  }
  const ids = { nvMobileSearchBtn: searchButton, nvMobileSearchBar: searchPanel, nvMobileSearchInput: searchInput, nvMobileNav: menuPanel };
  const document = {
    body, activeElement: null,
    getElementById: id => ids[id],
    querySelector: selector => selector === '.main-header .mobile-nav__btn' ? menuButton : null,
    querySelectorAll: () => [],
    addEventListener: (event, callback) => { (listeners[event] ||= []).push(callback); },
  };
  for (const element of [body, searchButton, menuButton, searchPanel, menuPanel, searchInput, submit, icon, menuLink, close, outside]) element.ownerDocument = document;
  const window = { location: { href: 'https://local.test/arama' }, addEventListener() {} };
  vm.runInNewContext(init, { document, window }, { filename: 'frontend-header-init.jet' });
  listeners.DOMContentLoaded.forEach(callback => callback());

  function click(target) {
    const event = { target, preventDefault() {}, stopImmediatePropagation() {} };
    listeners.click.forEach(callback => callback(event));
  }
  function key(target, value) {
    if (value === 'Enter' || value === ' ') {
      // Models the browser's native click activation of a real button.
      assert.ok(target === searchButton || target === menuButton);
      click(target);
      return;
    }
    listeners.keydown.forEach(callback => callback({ target, key: value, preventDefault() {} }));
  }
  function tabStops() {
    return [searchButton, menuButton, menuLink, close, searchInput, submit].filter(element => {
      for (let current = element; current; current = current.parent) if (current.hidden) return false;
      return true;
    });
  }
  function assertState() {
    for (const [button, panel] of [[searchButton, searchPanel], [menuButton, menuPanel]]) {
      assert.equal(button.getAttribute('aria-controls'), panel.id);
      assert.equal(button.getAttribute('aria-expanded'), String(!panel.hidden));
    }
    if (menuPanel.hidden) {
      assert.ok(!tabStops().includes(menuLink));
      assert.ok(!menuPanel.contains(document.activeElement));
    }
    if (searchPanel.hidden) {
      assert.ok(!tabStops().includes(searchInput));
      assert.ok(!searchPanel.contains(document.activeElement));
    }
  }
  return { document, searchButton, menuButton, searchPanel, menuPanel, searchInput, menuLink, close, outside, click, key, tabStops, assertState };
}

test('markup keeps button, panel, search and menu contracts', () => assertMarkup(header));

test('actual init script synchronizes panels, Tab stops, Escape and focus', () => {
  const ui = fixture();
  ui.assertState();
  assert.deepEqual(ui.tabStops(), [ui.searchButton, ui.menuButton]);
  ui.key(ui.menuButton, 'Enter');
  ui.assertState();
  assert.equal(ui.document.activeElement, ui.menuLink);
  assert.ok(ui.tabStops().includes(ui.menuLink));
  ui.click(ui.menuButton);
  ui.assertState();
  assert.ok(!ui.tabStops().includes(ui.menuLink));
  ui.key(ui.menuButton, ' ');
  ui.key(ui.menuLink, 'Escape');
  ui.assertState();
  assert.equal(ui.document.activeElement, ui.menuButton);
  ui.key(ui.searchButton, ' ');
  ui.assertState();
  assert.equal(ui.document.activeElement, ui.searchInput);
  ui.key(ui.searchInput, 'Escape');
  ui.assertState();
  assert.equal(ui.document.activeElement, ui.searchButton);
  ui.key(ui.searchButton, 'Enter');
  ui.click(ui.menuButton);
  ui.assertState();
  assert.equal(ui.searchPanel.hidden, true);
  ui.click(ui.close);
  ui.assertState();
  assert.equal(ui.document.activeElement, ui.menuButton);
});

test('wrong ARIA relation and focusable closed panel fail the contract', () => {
  assert.throws(() => assertMarkup(header.replace('aria-controls="nvMobileNav"', 'aria-controls="wrongPanel"')));
  assert.throws(() => assertMarkup(header.replace('id="nvMobileNav" hidden', 'id="nvMobileNav"')));
  const ui = fixture();
  ui.menuButton.setAttribute('aria-controls', 'wrongPanel');
  assert.throws(() => ui.assertState());
  ui.menuButton.setAttribute('aria-controls', 'nvMobileNav');
  ui.menuLink.focus();
  assert.throws(() => ui.assertState());
});
