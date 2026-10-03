/*
 * header.js — P2-4 header davranışları
 *
 * 1) Erişilebilirlik paneli (disclosure): yüksek kontrast + metin boyutu.
 *    Seçim <html data-contrast / data-text> özniteliklerine yazılır ve
 *    localStorage'da saklanır. Sayfa açılışında aynı tercihi main.jet
 *    <head> içindeki küçük satır içi betik ilk boyamadan önce uygular.
 * 2) Masaüstü menü: üst öğeler disclosure düğmesi. Yalnız tıklama/klavye
 *    ile açılır (fareyle üstünden geçince açılıp kaybolan menü yok —
 *    titreme ve büyüteç kullanan ziyaretçi için). Esc kapatır, odak
 *    düğmeye döner.
 * 3) Mobil menü içindeki akordeonlar.
 * 4) Geçerli sayfa bağlantısına aria-current="page".
 * Mobil menü (dialog) ve arama paneli frontend-header-init.jet'te.
 */
(function () {
    'use strict';

    var STORE_KEY = 'nv_a11y_v1';
    var root = document.documentElement;

    function readPrefs() {
        try {
            var raw = window.localStorage.getItem(STORE_KEY);
            var p = raw ? JSON.parse(raw) : null;
            return p && typeof p === 'object' ? p : {};
        } catch (e) { return {}; }
    }
    function writePrefs(p) {
        try { window.localStorage.setItem(STORE_KEY, JSON.stringify(p)); } catch (e) { /* gizli pencere vb. */ }
    }
    function applyPrefs(p) {
        if (p.contrast === 'high') root.setAttribute('data-contrast', 'high');
        else root.removeAttribute('data-contrast');
        if (p.text === 'large' || p.text === 'xlarge') root.setAttribute('data-text', p.text);
        else root.removeAttribute('data-text');
    }

    /* ---- Disclosure yardımcıları ------------------------------------- */
    function setOpen(button, panel, open) {
        button.setAttribute('aria-expanded', open ? 'true' : 'false');
        panel.hidden = !open;
    }

    /* ---- 1) Erişilebilirlik paneli ------------------------------------ */
    function initA11y() {
        var toggle = document.getElementById('nvA11yToggle');
        var panel = document.getElementById('nvA11yPanel');
        if (!toggle || !panel) return;
        var wrap = toggle.parentNode;
        var prefs = readPrefs();

        function sync() {
            var c = panel.querySelector('input[name="nvA11yContrast"][value="' + (prefs.contrast === 'high' ? 'high' : 'normal') + '"]');
            var t = panel.querySelector('input[name="nvA11yText"][value="' + (prefs.text === 'large' || prefs.text === 'xlarge' ? prefs.text : 'normal') + '"]');
            if (c) c.checked = true;
            if (t) t.checked = true;
        }
        function close(returnFocus) {
            if (panel.hidden) return;
            setOpen(toggle, panel, false);
            if (returnFocus) toggle.focus();
        }

        sync();
        toggle.addEventListener('click', function () {
            var willOpen = panel.hidden;
            setOpen(toggle, panel, willOpen);
            if (willOpen) {
                var checked = panel.querySelector('input:checked');
                if (checked) checked.focus();
            }
        });
        panel.addEventListener('change', function (e) {
            var t = e.target;
            if (t.name === 'nvA11yContrast') prefs.contrast = t.value === 'high' ? 'high' : 'normal';
            if (t.name === 'nvA11yText') prefs.text = t.value;
            applyPrefs(prefs);
            writePrefs(prefs);
        });
        var closeBtn = panel.querySelector('.nv-a11y__close');
        if (closeBtn) closeBtn.addEventListener('click', function () { close(true); });

        wrap.addEventListener('keydown', function (e) {
            if ((e.key === 'Escape' || e.key === 'Esc') && !panel.hidden) {
                e.stopPropagation();
                close(true);
            }
        });
        document.addEventListener('click', function (e) {
            if (!panel.hidden && !wrap.contains(e.target)) close(false);
        });
        wrap.addEventListener('focusout', function (e) {
            if (!panel.hidden && e.relatedTarget && !wrap.contains(e.relatedTarget)) close(false);
        });
    }

    /* ---- 2) Masaüstü menü ---------------------------------------------- */
    function initDesktopNav() {
        var nav = document.querySelector('.nv-nav');
        if (!nav) return;
        var toggles = Array.prototype.slice.call(nav.querySelectorAll('.nv-nav__toggle'));

        function panelOf(btn) { return document.getElementById(btn.getAttribute('aria-controls')); }
        function closeAll(except) {
            toggles.forEach(function (b) {
                if (b !== except) { var p = panelOf(b); if (p) setOpen(b, p, false); }
            });
        }
        toggles.forEach(function (btn) {
            var panel = panelOf(btn);
            if (!panel) return;
            btn.addEventListener('click', function () {
                var open = panel.hidden;
                closeAll(btn);
                setOpen(btn, panel, open);
            });
            btn.parentNode.addEventListener('keydown', function (e) {
                if ((e.key === 'Escape' || e.key === 'Esc') && !panel.hidden) {
                    e.stopPropagation();
                    setOpen(btn, panel, false);
                    btn.focus();
                }
            });
            btn.parentNode.addEventListener('focusout', function (e) {
                if (!panel.hidden && e.relatedTarget && !btn.parentNode.contains(e.relatedTarget)) setOpen(btn, panel, false);
            });
        });
        document.addEventListener('click', function (e) {
            if (!nav.contains(e.target)) closeAll(null);
        });
    }

    /* ---- 3) Mobil menü akordeonları ----------------------------------- */
    function initMobileAccordions() {
        document.querySelectorAll('.nv-mnav__toggle').forEach(function (btn) {
            var panel = document.getElementById(btn.getAttribute('aria-controls'));
            if (!panel) return;
            btn.addEventListener('click', function () { setOpen(btn, panel, panel.hidden); });
        });
    }

    /* ---- 4) Geçerli sayfa ---------------------------------------------- */
    function markCurrent() {
        var here = window.location.pathname.replace(/\/+$/, '') || '/';
        document.querySelectorAll('.nv-nav a[href], .nv-mnav a[href]').forEach(function (a) {
            var path;
            try { path = new URL(a.getAttribute('href'), window.location.href).pathname.replace(/\/+$/, '') || '/'; } catch (e) { return; }
            if (path !== here) return;
            a.setAttribute('aria-current', 'page');
            var item = a.closest('.nv-nav__item');
            if (item) item.classList.add('is-current');
        });
    }

    function init() {
        initA11y();
        initDesktopNav();
        initMobileAccordions();
        markCurrent();
    }
    if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init);
    else init();
})();
