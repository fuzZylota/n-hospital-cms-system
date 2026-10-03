/* P3-1 — /randevu talep sihirbazı.
 * - 4 adım; her adım ilerlemeden önce doğrulanır, hata özeti + alan altı hata.
 * - Adım değişiminde: metin göstergesi, aria-live duyurusu, odak adım başlığına.
 * - Enter yalnız son adımda gönderir; önceki adımlarda "İleri" gibi davranır.
 * - Hasta verisi localStorage/sessionStorage'a YAZILMAZ (proje gizlilik kararı).
 * - Yanıt sözleşmesi: JSON {status, message, rrid?}; başarı = status 200/201.
 *   HTTP 429 = form hız sınırı; JSON status 429 (HTTP 200) = aynı telefonla 24 saat kuralı.
 */
(function () {
    'use strict';

    var SERVER_ERROR = 'Server Hatası: Lütfen daha sonra tekrar deneyin.';
    var TOTAL = 4;

    var form = document.getElementById('nvRandevuForm');
    if (!form) return;

    var app = document.getElementById('nvRandevuApp');
    var done = document.getElementById('nvRandevuDone');
    var live = document.getElementById('nvStepLive');
    var stepText = document.getElementById('nvStepText');
    var steps = Array.prototype.slice.call(form.querySelectorAll('.nv-step'));
    var dots = Array.prototype.slice.call(document.querySelectorAll('[data-step-dot]'));
    var submitBtn = document.getElementById('nvSubmit');
    var submitLabel = submitBtn ? submitBtn.querySelector('.js-nv-submit-label') : null;
    var result = document.getElementById('nvResult');
    var current = 1;
    var sending = false;

    function $(id) { return document.getElementById(id); }
    function checked(name) { return form.querySelector('input[name="' + name + '"]:checked'); }
    // Başlığa anında konumlan + odakla (temanın "smooth" kaydırması uzun sayfada yorucu).
    function focusTop(h) {
        if (!h) return;
        try { h.scrollIntoView({ block: 'start', behavior: 'instant' }); } catch (err) { h.scrollIntoView(true); }
        try { h.focus({ preventScroll: true }); } catch (err) { h.focus(); }
    }
    function el(tag, cls, text) {
        var n = document.createElement(tag);
        if (cls) n.className = cls;
        if (text != null) n.textContent = text;
        return n;
    }

    /* ---------------- Telefon ---------------- */
    // Yinelenen talep kuralı telefonu metin olarak karşılaştırdığı için tek biçime çevrilir.
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
    function prettyPhone(n) {
        return /^0\d{10}$/.test(n) ? n.replace(/^(\d{4})(\d{3})(\d{2})(\d{2})$/, '$1 $2 $3 $4') : n;
    }

    var phone = $('nvPhone');
    var phoneCount = $('nvPhoneCount');
    function updatePhoneCount() {
        if (!phone || !phoneCount) return;
        var v = phone.value.trim();
        if (!v || v.charAt(0) === '+') { phoneCount.textContent = ''; return; }
        var digits = normalizePhone(v).length;
        phoneCount.textContent = 'Yazılan hane: ' + digits + ' / 11';
    }
    if (phone) {
        phone.addEventListener('input', updatePhoneCount);
        phone.addEventListener('blur', function () {
            var n = normalizePhone(phone.value);
            if (isValidPhone(n)) phone.value = prettyPhone(n);
            updatePhoneCount();
        });
    }

    /* ---------------- Tarih ---------------- */
    function isoDate(d) {
        var m = String(d.getMonth() + 1), day = String(d.getDate());
        return d.getFullYear() + '-' + (m.length < 2 ? '0' + m : m) + '-' + (day.length < 2 ? '0' + day : day);
    }
    var dateInput = $('nvDate');
    var today = new Date();
    var maxDay = new Date(today.getFullYear(), today.getMonth(), today.getDate() + 90);
    if (dateInput) {
        dateInput.min = isoDate(today);
        dateInput.max = isoDate(maxDay);
    }
    function trDate(iso) {
        var p = iso.split('-');
        return p.length === 3 ? p[2] + '.' + p[1] + '.' + p[0] : iso;
    }

    /* ---------------- Merkeze göre seçenekler ---------------- */
    function selectedSid() { var r = checked('sid'); return r ? r.value : ''; }

    function syncCall() {
        var r = checked('sid');
        document.querySelectorAll('.js-nv-call').forEach(function (a) {
            var num = a.querySelector('.js-nv-call-number');
            if (r && r.getAttribute('data-phone-href')) {
                a.href = 'tel:' + r.getAttribute('data-phone-href');
                if (num) num.textContent = r.getAttribute('data-phone');
            } else {
                a.href = a.getAttribute('data-default-href');
                if (num) num.textContent = a.getAttribute('data-default-number');
            }
        });
    }

    function syncOptions() {
        var sid = selectedSid();
        form.querySelectorAll('#nvBolumGroup [data-for-sid]').forEach(function (label) {
            var show = label.getAttribute('data-for-sid') === sid;
            label.hidden = !show;
            var input = label.querySelector('input');
            if (!show && input.checked) input.checked = false;
        });
        var bolum = checked('bolum');
        var brid = bolum && bolum.value !== 'emin-degilim' ? bolum.value : '';
        var visible = 0;
        form.querySelectorAll('#nvDoktorGroup [data-for-sid]').forEach(function (label) {
            var show = label.getAttribute('data-for-sid') === sid && (!brid || label.getAttribute('data-brid') === brid);
            label.hidden = !show;
            if (show) visible++;
            var input = label.querySelector('input');
            if (!show && input.checked) {
                input.checked = false;
                var any = form.querySelector('input[name="doktor"][value="fark-etmez"]');
                if (any) any.checked = true;
            }
        });
        var empty = $('nvDoktorEmpty');
        if (empty) empty.hidden = visible > 0;
    }

    form.addEventListener('change', function (e) {
        var name = e.target && e.target.name;
        if (name === 'sid') { syncOptions(); syncCall(); clearFieldError('sid'); loadRecaptcha(); }
        if (name === 'bolum') { syncOptions(); clearFieldError('bolum'); }
        if (name === 'kvkk' && e.target.checked) clearFieldError('kvkk');
    });

    /* ---------------- Hata gösterimi ---------------- */
    var FIELDS = {
        sid:        { error: 'nvSidError', target: function () { return form.querySelector('input[name="sid"]'); }, group: 'nvSidGroup' },
        bolum:      { error: 'nvBolumError', target: function () { return form.querySelector('input[name="bolum"]'); }, group: 'nvBolumGroup' },
        date:       { error: 'nvDateError', target: function () { return dateInput; } },
        first_name: { error: 'nvFirstNameError', target: function () { return $('nvFirstName'); } },
        last_name:  { error: 'nvLastNameError', target: function () { return $('nvLastName'); } },
        phone:      { error: 'nvPhoneError', target: function () { return phone; } },
        email:      { error: 'nvEmailError', target: function () { return $('nvEmail'); } },
        kvkk:       { error: 'nvKvkkError', target: function () { return $('nvKvkk'); } },
        captcha:    { error: 'nvCaptchaError', target: function () { return $('nvRecaptcha'); } }
    };

    function setFieldError(key, msg) {
        var f = FIELDS[key];
        if (!f) return;
        var err = $(f.error);
        if (err) err.textContent = msg;
        var t = f.target();
        if (f.group) {
            var g = $(f.group);
            if (g) g.classList.add('nv-choices--error');
            form.querySelectorAll('#' + f.group + ' input').forEach(function (i) { i.setAttribute('aria-invalid', 'true'); });
        } else if (t && t.tagName !== 'DIV') {
            t.setAttribute('aria-invalid', 'true');
            var field = t.closest('.nv-field');
            if (field) field.classList.add('nv-field--error');
        }
    }

    function clearFieldError(key) {
        var f = FIELDS[key];
        if (!f) return;
        var err = $(f.error);
        if (err) err.textContent = '';
        if (f.group) {
            var g = $(f.group);
            if (g) g.classList.remove('nv-choices--error');
            form.querySelectorAll('#' + f.group + ' input').forEach(function (i) { i.removeAttribute('aria-invalid'); });
        } else {
            var t = f.target();
            if (t && t.removeAttribute) {
                t.removeAttribute('aria-invalid');
                var field = t.closest && t.closest('.nv-field');
                if (field) field.classList.remove('nv-field--error');
            }
        }
    }

    function hideSummary(n) {
        var s = $('nvStep' + n + 'Summary');
        if (s) { s.hidden = true; s.textContent = ''; }
    }

    function showSummary(n, errors) {
        var s = $('nvStep' + n + 'Summary');
        if (!s) return;
        s.textContent = '';
        s.appendChild(el('p', 'nv-alert__title', errors.length === 1 ? 'Devam etmeden önce şunu düzeltin:' : 'Devam etmeden önce şunları düzeltin:'));
        var ul = el('ul', 'nv-step__summary-list');
        errors.forEach(function (e) {
            var li = el('li');
            var f = FIELDS[e.key];
            var t = f && f.target();
            var anchor = f && f.group ? f.group : (t && t.id);
            if (anchor) {
                var a = el('a', null, e.msg);
                a.href = '#' + anchor;
                a.addEventListener('click', function (ev) {
                    ev.preventDefault();
                    var focusable = t;
                    if (f.group) {
                        // Radyo grubu: seçili ya da görünen ilk seçenek
                        focusable = form.querySelector('#' + f.group + ' input:checked') ||
                                    form.querySelector('#' + f.group + ' .nv-choice:not([hidden]) input');
                    } else if (t.tagName === 'DIV') {
                        focusable = t.querySelector('iframe, button') || t;
                    }
                    if (focusable && focusable.focus) focusable.focus();
                });
                li.appendChild(a);
            } else {
                li.textContent = e.msg;
            }
            ul.appendChild(li);
        });
        s.appendChild(ul);
        s.hidden = false;
        s.focus();
    }

    // Alan değiştirilince o alanın eski hata metni kalkar (yeniden denetim gönderimde/İleri'de).
    var INPUT_KEYS = { nvFirstName: 'first_name', nvLastName: 'last_name', nvPhone: 'phone', nvEmail: 'email', nvDate: 'date' };
    form.addEventListener('input', function (e) {
        var key = e.target && INPUT_KEYS[e.target.id];
        if (key && e.target.getAttribute('aria-invalid') === 'true') clearFieldError(key);
    });

    /* ---------------- Doğrulama ---------------- */
    function validate(n) {
        var errors = [];
        function add(key, msg) { errors.push({ key: key, msg: msg }); setFieldError(key, msg); }
        if (n === 1) {
            clearFieldError('sid');
            if (!form.querySelector('input[name="sid"]')) {
                errors.push({ key: '', msg: 'Şu an çevrim içi talep alamıyoruz. Lütfen bizi telefonla arayın.' });
            } else if (!selectedSid()) {
                add('sid', 'Bir merkez seçin.');
            }
        }
        if (n === 2) {
            clearFieldError('bolum');
            if (!checked('bolum')) add('bolum', 'Bir bölüm seçin. Emin değilseniz “Emin değilim, merkez yönlendirsin” seçeneğini işaretleyin.');
        }
        if (n === 3 && dateInput) {
            clearFieldError('date');
            var v = dateInput.value;
            if (dateInput.validity && dateInput.validity.badInput) {
                add('date', 'Tarih tam değil. Gün, ay ve yılı yazın ya da alanı boş bırakın.');
            } else if (v && v < dateInput.min) {
                add('date', 'Geçmiş bir gün seçtiniz. Bugün ya da daha ileri bir gün seçin veya alanı boş bırakın.');
            } else if (v && v > dateInput.max) {
                add('date', 'En fazla 3 ay sonrası için gün seçebilirsiniz. Daha ileri bir tarih için not alanına yazın.');
            }
        }
        if (n === 4) {
            ['first_name', 'last_name', 'phone', 'email', 'kvkk', 'captcha'].forEach(clearFieldError);
            if (!$('nvFirstName').value.trim()) add('first_name', 'Adınızı yazın.');
            if (!$('nvLastName').value.trim()) add('last_name', 'Soyadınızı yazın.');
            var p = normalizePhone(phone.value);
            if (!phone.value.trim()) add('phone', 'Telefon numaranızı yazın. Sizi bu numaradan arayacağız.');
            else if (!isValidPhone(p)) add('phone', 'Telefon numarası eksik ya da hatalı görünüyor. 0 ile başlayan 11 haneyle yazın, örneğin 0532 123 45 67.');
            var email = $('nvEmail').value.trim();
            if (email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) add('email', 'E-posta adresi eksik görünüyor. Örnek: ad@ornek.com. İsterseniz boş bırakabilirsiniz.');
            if (!$('nvKvkk').checked) add('kvkk', 'Devam etmek için aydınlatma metnini okuduğunuzu onaylayın.');
            if (captchaRequired()) {
                if (typeof window.grecaptcha === 'undefined' || captchaWidget === null) {
                    add('captcha', 'Güvenlik doğrulaması henüz yüklenmedi. Birkaç saniye bekleyin; yüklenmezse sayfayı yenileyin ya da bizi arayın.');
                } else if (!window.grecaptcha.getResponse(captchaWidget)) {
                    add('captcha', '“Ben robot değilim” kutusunu işaretleyin.');
                }
            }
        }
        if (errors.length) showSummary(n, errors); else hideSummary(n);
        return errors.length === 0;
    }

    /* ---------------- Adım geçişi ---------------- */
    function fillSummary() {
        function put(key, text) { var d = form.querySelector('[data-sum="' + key + '"]'); if (d) d.textContent = text; }
        var sid = checked('sid'), bolum = checked('bolum'), doktor = checked('doktor'), saat = checked('saat');
        put('merkez', sid ? sid.getAttribute('data-name') : '—');
        put('bolum', bolum ? bolum.getAttribute('data-name') : '—');
        put('doktor', doktor ? doktor.getAttribute('data-name') : 'Fark etmez');
        var gun = dateInput && dateInput.value ? trDate(dateInput.value) : 'Gün fark etmez';
        put('zaman', gun + ', ' + (saat ? saat.value.toLocaleLowerCase('tr') : 'fark etmez'));
    }

    function show(n, moveFocus) {
        current = n;
        steps.forEach(function (s) { s.hidden = Number(s.getAttribute('data-step')) !== n; });
        dots.forEach(function (d) {
            var k = Number(d.getAttribute('data-step-dot'));
            d.classList.toggle('is-current', k === n);
            d.classList.toggle('is-done', k < n);
        });
        var step = steps[n - 1];
        var name = step.getAttribute('data-step-name');
        stepText.querySelector('.nv-steps__count').textContent = 'Adım ' + n + ' / ' + TOTAL + ':';
        stepText.querySelector('.js-nv-step-name').textContent = name;
        if (n === 4) { fillSummary(); renderRecaptcha(); }
        if (n >= 2) loadRecaptcha();
        if (moveFocus) {
            focusTop(step.querySelector('.nv-step__title'));
            // Odak duyurusundan sonra adım bilgisini kibarca yinele.
            live.textContent = '';
            setTimeout(function () { live.textContent = 'Adım ' + n + ' / ' + TOTAL + ': ' + name; }, 150);
        }
    }

    function next() {
        if (current < TOTAL && validate(current)) show(current + 1, true);
    }
    function prev() {
        if (current > 1) { hideSummary(current); show(current - 1, true); }
    }

    form.addEventListener('click', function (e) {
        var b = e.target.closest('button');
        if (!b || !form.contains(b)) return;
        if (b.hasAttribute('data-next')) { e.preventDefault(); next(); }
        else if (b.hasAttribute('data-prev')) { e.preventDefault(); prev(); }
        else if (b.hasAttribute('data-goto')) { e.preventDefault(); hideSummary(current); show(Number(b.getAttribute('data-goto')), true); }
    });

    // Enter: son adım dışında gönderim yapmaz, "İleri" gibi çalışır.
    form.addEventListener('keydown', function (e) {
        if (e.key !== 'Enter' || current === TOTAL) return;
        var t = e.target;
        if (t.tagName === 'INPUT' && t.type !== 'checkbox') { e.preventDefault(); next(); }
    });

    /* ---------------- reCAPTCHA (yalnız anahtar varsa) ---------------- */
    var siteKey = form.getAttribute('data-recaptcha-key') || '';
    var captchaWidget = null;
    var captchaLoading = false;
    function captchaRequired() { return !!siteKey && !!$('nvRecaptcha'); }
    function loadRecaptcha() {
        if (!captchaRequired() || captchaLoading || typeof window.grecaptcha !== 'undefined') return;
        captchaLoading = true;
        window.nvRandevuRecaptchaLoad = function () { if (current === TOTAL) renderRecaptcha(); };
        var s = document.createElement('script');
        s.src = 'https://www.google.com/recaptcha/api.js?onload=nvRandevuRecaptchaLoad&render=explicit&hl=tr';
        s.async = true;
        s.defer = true;
        document.head.appendChild(s);
    }
    function renderRecaptcha() {
        if (!captchaRequired() || captchaWidget !== null) return;
        if (typeof window.grecaptcha === 'undefined' || typeof window.grecaptcha.render !== 'function') return;
        try {
            captchaWidget = window.grecaptcha.render('nvRecaptcha', {
                sitekey: siteKey,
                callback: function () { clearFieldError('captcha'); },
                'expired-callback': function () { setFieldError('captcha', 'Doğrulamanın süresi doldu. Kutuyu yeniden işaretleyin.'); }
            });
        } catch (err) { captchaWidget = null; }
    }
    function resetRecaptcha() {
        if (captchaWidget !== null && window.grecaptcha) { try { window.grecaptcha.reset(captchaWidget); } catch (err) {} }
    }

    /* ---------------- Gönderim ---------------- */
    function callButton() {
        var src = document.querySelector('.nv-randevu__call .js-nv-call');
        if (!src) return null;
        var a = src.cloneNode(true);
        a.classList.remove('nv-btn--block');
        return a;
    }

    function showResult(kind, title, text, withCall) {
        result.className = 'nv-randevu__result nv-alert nv-alert--' + kind;
        result.setAttribute('role', kind === 'danger' ? 'alert' : 'status');
        result.textContent = '';
        result.appendChild(el('p', 'nv-alert__title', title));
        if (text) result.appendChild(el('p', null, text));
        if (withCall) {
            var a = callButton();
            if (a) { var p = el('p', 'nv-randevu__result-call'); p.appendChild(a); result.appendChild(p); }
        }
        result.hidden = false;
        result.focus();
    }

    function setSending(on) {
        sending = on;
        form.setAttribute('aria-busy', on ? 'true' : 'false');
        if (submitBtn) submitBtn.disabled = on;
        if (submitLabel) submitLabel.textContent = on ? 'Gönderiliyor…' : 'Talebi Gönder';
    }

    function buildMessage() {
        var lines = [];
        var bolum = checked('bolum'), doktor = checked('doktor'), saat = checked('saat');
        lines.push('Bölüm tercihi: ' + (bolum ? bolum.getAttribute('data-name') : 'Belirtilmedi'));
        lines.push('Doktor tercihi: ' + (doktor ? doktor.getAttribute('data-name') : 'Fark etmez'));
        lines.push('Saat tercihi: ' + (saat ? saat.value : 'Fark etmez'));
        var note = ($('nvNote').value || '').trim();
        if (note) lines.push('Not: ' + note);
        lines.push('(Randevu sayfası, 4 adımlı talep formu)');
        return lines.join('\n');
    }

    function succeed(data) {
        var ref = data && data.rrid != null && String(data.rrid) !== '' ? String(data.rrid) : '';
        var refBox = $('nvDoneRef');
        if (ref && refBox) { refBox.querySelector('.js-nv-ref').textContent = ref; refBox.hidden = false; }
        syncCall();
        form.reset();
        app.hidden = true;
        done.hidden = false;
        focusTop($('nvDoneTitle'));
    }

    function handleError(httpStatus, data) {
        var msg = data && typeof data.message === 'string' ? data.message : '';
        if (httpStatus === 429) {
            showResult('warning', 'Kısa sürede çok fazla deneme yapıldı', (msg || 'Lütfen birkaç dakika sonra tekrar deneyin.') + ' Beklemek istemezseniz bizi arayabilirsiniz.', true);
            return;
        }
        var status = data ? Number(data.status) : 0;
        if (status === 429) {
            showResult('info', 'Bu telefon numarasıyla bekleyen bir talebiniz var',
                'Son 24 saat içinde bu numarayla gönderilen bir talep bize ulaştı. Merkezimiz sizi arayacak; yeniden göndermenize gerek yok. Beklemek istemezseniz hemen arayabilirsiniz.', true);
            return;
        }
        if (status === 400) {
            if (/recaptcha/i.test(msg)) {
                setFieldError('captcha', 'Güvenlik doğrulaması geçmedi. Kutuyu yeniden işaretleyip tekrar gönderin.');
                showSummary(4, [{ key: 'captcha', msg: 'Güvenlik doğrulaması geçmedi. Kutuyu yeniden işaretleyip tekrar gönderin.' }]);
                return;
            }
            if (/zorunlu/i.test(msg)) { validate(4); return; }
            showSummary(4, [{ key: '', msg: msg || 'Bilgilerinizden biri kabul edilmedi. Lütfen kontrol edip tekrar deneyin.' }]);
            return;
        }
        showResult('danger', SERVER_ERROR, '', true);
    }

    form.addEventListener('submit', function (e) {
        e.preventDefault();
        if (current !== TOTAL) { next(); return; }
        if (sending) return;
        result.hidden = true;
        if (!validate(4)) return;

        var payload = {
            patient_first_name: $('nvFirstName').value.trim(),
            patient_last_name: $('nvLastName').value.trim(),
            patient_phone: normalizePhone(phone.value),
            patient_email: $('nvEmail').value.trim(),
            sid: selectedSid(),
            message: buildMessage(),
            website: (form.querySelector('input[name="website"]') || {}).value || ''
        };
        if (dateInput && dateInput.value) payload.preferred_date = dateInput.value + 'T00:00:00Z';
        if (captchaRequired() && captchaWidget !== null) payload.recaptcha_token = window.grecaptcha.getResponse(captchaWidget);

        setSending(true);
        var httpStatus = 0;
        fetch(form.getAttribute('data-endpoint'), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', 'Accept': 'application/json' },
            credentials: 'same-origin',
            body: JSON.stringify(payload)
        }).then(function (res) {
            httpStatus = res.status;
            return res.json().catch(function () { return null; });
        }).then(function (data) {
            if (data && (Number(data.status) === 200 || Number(data.status) === 201)) {
                succeed(data);
                return;
            }
            resetRecaptcha();
            setSending(false);
            handleError(httpStatus, data);
        }).catch(function () {
            resetRecaptcha();
            setSending(false);
            showResult('danger', SERVER_ERROR, '', true);
        });
    });

    // Sayfa geri gelindiğinde (bfcache) düğme takılı kalmasın.
    window.addEventListener('pageshow', function () { if (sending && !done.hidden) return; setSending(false); });

    syncOptions();
    syncCall();
    updatePhoneCount();
})();
