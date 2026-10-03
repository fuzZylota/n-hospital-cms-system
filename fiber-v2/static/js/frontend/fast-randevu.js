/* Hızlı randevu paneli (components/fast-randevu.jet) + ortak talep yardımcıları.
 * window.NvTalep: anasayfadaki hızlı randevu şeridi de (home/hizli-randevu-strip.jet)
 * aynı telefon biçimini ve aynı başarı/hata dilini kullansın diye burada.
 * /randevu sayfası kendi dosyasını (randevu.js) kullanır; dil orada da aynıdır.
 * Hasta verisi tarayıcı deposuna yazılmaz. */
window.NvTalep = (function () {
    var SERVER_ERROR = 'Server Hatası: Lütfen daha sonra tekrar deneyin.';

    // "24 saatte bir talep" kuralı telefonu metin olarak karşılaştırır; tek biçim gönderilir.
    function normalizePhone(value) {
        var raw = String(value || '').trim();
        var plus = raw.charAt(0) === '+';
        var d = raw.replace(/\D/g, '');
        if (d.length === 12 && d.indexOf('90') === 0) return '0' + d.slice(2);
        if (plus) return '+' + d;
        if (d.length === 10 && d.charAt(0) !== '0') return '0' + d;
        return d;
    }

    function isValidPhone(n) { return /^0\d{10}$/.test(n) || /^\+\d{10,15}$/.test(n); }

    function describe(httpStatus, data) {
        var st = data ? Number(data.status) : 0;
        var msg = data && typeof data.message === 'string' ? data.message : '';
        if (st === 200 || st === 201) {
            var ref = data.rrid != null && String(data.rrid) !== '' ? String(data.rrid) : '';
            return { ok: true, kind: 'success', title: 'Talebiniz alındı',
                text: 'Merkezimiz mesai saatleri içinde sizi arayacak.' + (ref ? ' Talep numaranız: ' + ref + '.' : '') +
                      ' Bu bir randevu onayı değildir; gün ve saati telefonda birlikte belirleriz.' };
        }
        if (httpStatus === 429) {
            return { kind: 'warning', title: 'Kısa sürede çok fazla deneme yapıldı',
                text: (msg || 'Lütfen birkaç dakika sonra tekrar deneyin.') + ' Beklemek istemezseniz bizi arayabilirsiniz.' };
        }
        if (st === 429) {
            return { kind: 'info', title: 'Bu telefon numarasıyla bekleyen bir talebiniz var',
                text: 'Son 24 saat içinde bu numarayla gönderilen bir talep bize ulaştı. Merkezimiz sizi arayacak; yeniden göndermenize gerek yok. Beklemek istemezseniz hemen arayabilirsiniz.' };
        }
        if (st === 400) {
            return { kind: 'danger', title: 'Talep gönderilemedi', captcha: /recaptcha/i.test(msg),
                text: /recaptcha/i.test(msg) ? 'Güvenlik doğrulaması geçmedi. Kutuyu yeniden işaretleyip tekrar gönderin.'
                                              : (msg || 'Bilgilerinizden biri kabul edilmedi. Lütfen kontrol edip tekrar deneyin.') };
        }
        return { kind: 'danger', title: SERVER_ERROR, text: '' };
    }

    function render(box, r) {
        if (!box) return;
        box.textContent = '';
        if (!r) { box.classList.remove('is-visible'); return; }
        var d = document.createElement('div');
        d.className = 'nv-alert nv-alert--' + r.kind;
        var t = document.createElement('p');
        t.className = 'nv-alert__title';
        t.textContent = r.title;
        d.appendChild(t);
        if (r.text) { var p = document.createElement('p'); p.textContent = r.text; d.appendChild(p); }
        box.appendChild(d);
        box.classList.add('is-visible');
    }

    // Alan altı hata: <p class="nv-error" id="..."> + aria-invalid
    function fieldError(input, errorEl, msg) {
        if (errorEl) errorEl.textContent = msg || '';
        if (input) {
            if (msg) input.setAttribute('aria-invalid', 'true'); else input.removeAttribute('aria-invalid');
        }
    }

    function post(payload) {
        var httpStatus = 0;
        return fetch('/backend/add-randevu-request', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', 'Accept': 'application/json' },
            credentials: 'same-origin',
            body: JSON.stringify(payload)
        }).then(function (res) {
            httpStatus = res.status;
            return res.json().catch(function () { return null; });
        }).then(function (data) {
            return describe(httpStatus, data);
        }, function () {
            return describe(0, null);
        });
    }

    return { SERVER_ERROR: SERVER_ERROR, normalizePhone: normalizePhone, isValidPhone: isValidPhone,
             describe: describe, render: render, fieldError: fieldError, post: post };
})();

let recaptchaVerifiedFast = false;

// Gönder düğmesi artık pasifleştirilmiyor: eksik bilgi gönderimde alan altında söyleniyor
// (sessiz başarısızlık yok). Ad, eski çağrılarla uyum için korunuyor.
function updateFastRandevuSubmitState() {}

function setFastRandevuFeedback(result) {
    window.NvTalep.render(document.getElementById('fastRandevuFeedback'), result);
}

window.onRecaptchaVerifiedFast = function(token) {
    recaptchaVerifiedFast = true;
    const tokenInput = document.getElementById('fastRandevuRecaptchaToken');
    if (tokenInput) {
        tokenInput.value = token;
    }
    window.NvTalep.fieldError(null, document.getElementById('fastRandevuCaptchaError'), '');
};

window.onRecaptchaExpiredFast = function() {
    recaptchaVerifiedFast = false;
    const tokenInput = document.getElementById('fastRandevuRecaptchaToken');
    if (tokenInput) {
        tokenInput.value = '';
    }
};

// Merkez listesi artık şablonda (Options.SubelerLinks) basılıyor; eski
// /backend/get-all-subeler isteği oturum gerektirdiği için ziyaretçide hiç dolmuyordu.
function loadSubeler() {}

function clearFastRandevuErrors() {
    ['Name', 'Surname', 'Phone', 'KVKK', 'Captcha'].forEach(function (k) {
        window.NvTalep.fieldError(document.getElementById('fastRandevu' + k), document.getElementById('fastRandevu' + k + 'Error'), '');
    });
}

function setFastRandevuOpenState(isOpen) {
    const container = document.getElementById('fastRandevu');
    const toggleButton = document.getElementById('fastRandevuToggle');
    const panel = container ? container.querySelector('.fast-randevu__panel') : null;

    if (!container || !toggleButton || !panel) return;

    if (isOpen) {
        container.classList.add('is-open');
        loadSubeler();
        if (typeof window.loadFastRecaptchaScript === 'function') {
            window.loadFastRecaptchaScript();
        }
    } else {
        container.classList.remove('is-open');
        // Formu ve geri bildirimi sıfırla (hasta verisi panelde kalmasın)
        const form = document.getElementById('fastRandevuForm');
        if (form) form.reset();
        setFastRandevuFeedback(null);
        clearFastRandevuErrors();
    }
    toggleButton.setAttribute('aria-expanded', isOpen ? 'true' : 'false');
    panel.setAttribute('aria-hidden', isOpen ? 'false' : 'true');
}

function handleFastRandevuOutsideInteraction(event) {
    const container = document.getElementById('fastRandevu');
    if (!container || !container.classList.contains('is-open')) return;

    const panel = container.querySelector('.fast-randevu__panel');
    const toggleButton = document.getElementById('fastRandevuToggle');
    const closeButton = document.getElementById('fastRandevuClose');
    const mobileLineButton = document.getElementById('fastRandevuMobileLine');

    const target = event.target;

    // Ignore clicks/taps inside panel or on known controls
    if (panel && panel.contains(target)) return;
    if (toggleButton && toggleButton.contains(target)) return;
    if (closeButton && closeButton.contains(target)) return;
    if (mobileLineButton && mobileLineButton.contains(target)) return;
    // Çerez bandı/tercih penceresi panelin üstünde açılabilir; onlara dokunmak paneli kapatmasın.
    if (target.closest && target.closest('#nvCookieBanner, #nvCookieModal, #nvCookieBackdrop')) return;

    setFastRandevuOpenState(false);
}

document.addEventListener('DOMContentLoaded', () => {
    const container = document.getElementById('fastRandevu');
    const toggleButton = document.getElementById('fastRandevuToggle');
    const form = document.getElementById('fastRandevuForm');
    const closeButton = document.getElementById('fastRandevuClose');
    const mobileLineButton = document.getElementById('fastRandevuMobileLine');

    if (container && toggleButton) {
        toggleButton.addEventListener('click', (event) => {
            event.preventDefault();
            const isOpen = container.classList.contains('is-open');
            setFastRandevuOpenState(!isOpen);
        });
    }

    if (closeButton) {
        closeButton.addEventListener('click', (event) => {
            event.preventDefault();
            setFastRandevuOpenState(false);
        });
    }

    if (mobileLineButton && container) {
        mobileLineButton.addEventListener('click', (event) => {
            event.preventDefault();
            const isOpen = container.classList.contains('is-open');
            setFastRandevuOpenState(!isOpen);
        });
    }

    // Close when clicking/touching outside the panel
    document.addEventListener('click', handleFastRandevuOutsideInteraction);
    document.addEventListener('touchstart', handleFastRandevuOutsideInteraction, { passive: true });

    if (!form) {
        return;
    }

    let sending = false;
    const T = window.NvTalep;

    form.addEventListener('submit', async (event) => {
        event.preventDefault();
        if (sending) return;

        const firstNameInput = document.getElementById('fastRandevuName');
        const lastNameInput = document.getElementById('fastRandevuSurname');
        const phoneInput = document.getElementById('fastRandevuPhone');
        const subeSelect = document.getElementById('fastRandevuSube');
        const kvkkCheckbox = document.getElementById('fastRandevuKVKK');
        const recaptchaWidget = document.querySelector('#fastRandevu .g-recaptcha');
        const tokenInput = document.getElementById('fastRandevuRecaptchaToken');
        const submitButton = document.getElementById('fastRandevuSubmit');
        const err = (k) => document.getElementById('fastRandevu' + k + 'Error');

        clearFastRandevuErrors();
        setFastRandevuFeedback(null);
        let firstInvalid = null;
        const fail = (input, k, msg) => { T.fieldError(input, err(k), msg); if (!firstInvalid) firstInvalid = input; };

        const phone = T.normalizePhone(phoneInput.value);
        if (!firstNameInput.value.trim()) fail(firstNameInput, 'Name', 'Adınızı yazın.');
        if (!lastNameInput.value.trim()) fail(lastNameInput, 'Surname', 'Soyadınızı yazın.');
        if (!phoneInput.value.trim()) fail(phoneInput, 'Phone', 'Telefon numaranızı yazın.');
        else if (!T.isValidPhone(phone)) fail(phoneInput, 'Phone', 'Telefon numarası eksik görünüyor. Örnek: 0532 123 45 67.');
        if (kvkkCheckbox && !kvkkCheckbox.checked) fail(kvkkCheckbox, 'KVKK', 'Aydınlatma metnini okuduğunuzu onaylayın.');

        let recaptchaToken = null;
        if (recaptchaWidget) {
            if (typeof grecaptcha === 'undefined') {
                fail(recaptchaWidget, 'Captcha', 'Güvenlik doğrulaması yükleniyor. Birkaç saniye bekleyin.');
            } else {
                recaptchaToken = (recaptchaVerifiedFast && tokenInput && tokenInput.value) ? tokenInput.value : '';
                if (!recaptchaToken) {
                    try { recaptchaToken = grecaptcha.getResponse() || ''; } catch (e) { recaptchaToken = ''; }
                }
                if (!recaptchaToken) fail(recaptchaWidget, 'Captcha', '“Ben robot değilim” kutusunu işaretleyin.');
                else if (tokenInput) tokenInput.value = recaptchaToken;
            }
        }

        if (firstInvalid) {
            if (firstInvalid.focus && firstInvalid.tagName !== 'DIV') firstInvalid.focus();
            return;
        }

        const payload = {
            patient_first_name: firstNameInput.value.trim(),
            patient_last_name: lastNameInput.value.trim(),
            patient_phone: phone,
            message: 'Hızlı randevu formu üzerinden iletilmiştir.',
            website: (form.querySelector('input[name="website"]') || {}).value || ''
        };
        if (subeSelect && subeSelect.value) payload.sid = subeSelect.value;
        if (recaptchaToken) payload.recaptcha_token = recaptchaToken;

        sending = true;
        submitButton.disabled = true;
        submitButton.textContent = 'Gönderiliyor…';

        const result = await T.post(payload);
        sending = false;
        submitButton.disabled = false;
        submitButton.textContent = 'Talebi Gönder';
        setFastRandevuFeedback(result);
        if (result.ok) {
            form.reset();
            recaptchaVerifiedFast = false;
        } else if (result.captcha) {
            T.fieldError(null, err('Captcha'), result.text);
        }
        // Jeton tek kullanımlık: her yanıttan sonra doğrulama kutusu sıfırlanır.
        if (typeof grecaptcha !== 'undefined' && recaptchaWidget) {
            try { grecaptcha.reset(); } catch (e) {}
            recaptchaVerifiedFast = false;
        }
        // Geri bildirim kendiliğinden kapanmaz (yavaş okuyan ziyaretçi için).
    });
});
