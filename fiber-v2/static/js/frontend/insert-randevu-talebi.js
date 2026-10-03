// Randevu talebi gönderimi
document.addEventListener('DOMContentLoaded', () => {
    const form = document.querySelector('.appointment-one__form');
    if (!form) return;

    form.setAttribute('novalidate', 'true');
    if (typeof jQuery !== 'undefined') {
        try {
            jQuery(form).off();
            if (jQuery.fn.validate) jQuery(form).validate().destroy();
        } catch(e) {}
    }

    const submitBtn = document.getElementById('appointmentSubmitBtn');
    if (submitBtn) {
        submitBtn.removeAttribute('disabled');
        submitBtn.classList.remove('is-disabled');
    }

    const showToast = (type, message) => {
        const existing = document.getElementById('randevu-toast');
        if (existing) existing.remove();
        const toast = document.createElement('div');
        toast.id = 'randevu-toast';
        const isSuccess = type === 'success';
        toast.style.cssText = `
            position: fixed; bottom: 32px; left: 50%; transform: translateX(-50%);
            background: ${isSuccess ? '#fff' : '#fff'};
            color: ${isSuccess ? '#0a2241' : '#0a2241'};
            border-left: 4px solid ${isSuccess ? '#16a34a' : '#dc2626'};
            padding: 14px 20px 14px 16px;
            border-radius: 8px;
            box-shadow: 0 4px 24px rgba(0,0,0,0.13);
            display: flex; align-items: center; gap: 12px;
            font-size: 14px; font-family: inherit;
            z-index: 99999; min-width: 280px; max-width: 420px;
            animation: randevuToastIn 0.3s ease;
        `;
        toast.innerHTML = `
            <i class="fas ${isSuccess ? 'fa-check-circle' : 'fa-exclamation-circle'}" 
               style="color:${isSuccess ? '#16a34a' : '#dc2626'};font-size:18px;flex-shrink:0;"></i>
            <span>${message}</span>
        `;
        if (!document.getElementById('randevu-toast-style')) {
            const style = document.createElement('style');
            style.id = 'randevu-toast-style';
            style.textContent = '@keyframes randevuToastIn{from{opacity:0;transform:translateX(-50%) translateY(16px)}to{opacity:1;transform:translateX(-50%) translateY(0)}}';
            document.head.appendChild(style);
        }
        document.body.appendChild(toast);
        setTimeout(() => {
            toast.style.animation = 'randevuToastIn 0.3s ease reverse';
            setTimeout(() => toast.remove(), 300);
        }, 5000);
    };

    const showModal = () => {};
    const hideModal = () => {};
    const updateModalStatus = (type, message) => {
        if (type === 'loading') return;
        if (type === 'success') showToast('success', message);
        if (type === 'error') showToast('error', message);
    };

    const convertDateForBackend = (dateValue) => {
        if (!dateValue || dateValue === '') return '0001-01-01T00:00:00Z';
        if (/^\d{4}-\d{2}-\d{2}$/.test(dateValue)) return dateValue + 'T00:00:00Z';
        const d = new Date(dateValue);
        if (!isNaN(d.getTime())) return d.toISOString();
        return '0001-01-01T00:00:00Z';
    };

    const doSubmit = async () => {
        const nameInput = form.querySelector('input[name="patient_first_name"]');
        const surnameInput = form.querySelector('input[name="patient_last_name"]');
        const emailInput = form.querySelector('input[name="patient_email"]');
        const phoneInput = form.querySelector('input[name="patient_phone"]');
        const dateInput = form.querySelector('input[name="preferred_date"]');
        const messageInput = form.querySelector('textarea[name="message"]');
        const sidSelect = form.querySelector('select[name="sid"]');
        const kvkkCheckbox = form.querySelector('#appointmentKVKK');
        const recaptchaWidget = form.querySelector('.g-recaptcha') || document.getElementById('randevuRecaptchaWidget');

        if (kvkkCheckbox && !kvkkCheckbox.checked) {
            updateModalStatus('error', 'Lütfen KVKK onay kutusunu işaretleyin.');
            showModal();
            return;
        }

        let recaptchaToken = '';
        if (recaptchaWidget) {
            if (typeof grecaptcha === 'undefined') {
                updateModalStatus('error', 'Güvenlik doğrulaması yükleniyor, lütfen bekleyin...');
                showModal();
                return;
            }
            // Global token'ı kullan (onRecaptchaVerified callback'i set ediyor)
            recaptchaToken = window._randevuCaptchaToken || grecaptcha.getResponse() || '';
            if (!recaptchaToken) {
                updateModalStatus('error', 'Lütfen güvenlik doğrulamasını tamamlayın.');
                showModal();
                return;
            }
        }

        if (!nameInput?.value || !surnameInput?.value || !phoneInput?.value) {
            updateModalStatus('error', 'Lütfen ad, soyad ve telefon alanlarını doldurun.');
            showModal();
            return;
        }

        const payload = {
            patient_first_name: nameInput.value,
            patient_last_name: surnameInput.value,
            patient_phone: phoneInput.value,
            patient_email: emailInput?.value || '',
            preferred_date: convertDateForBackend((dateInput?.value || '').trim()),
            sid: sidSelect?.value || '',
            message: messageInput?.value || '',
            recaptcha_token: recaptchaToken,
            website: form.querySelector('input[name="website"]')?.value || '',
        };

        showModal();
        updateModalStatus('loading', 'Randevu İsteği Gönderiliyor...');

        try {
            const res = await fetch('/backend/add-randevu-request', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
            });
            const data = await res.json();
            if (data && (data.status === 200 || data.status === 201)) {
                updateModalStatus('success', 'Randevu talebiniz başarıyla gönderildi. En kısa sürede sizinle iletişime geçeceğiz.');
                form.reset();
                if (typeof grecaptcha !== 'undefined' && recaptchaWidget) grecaptcha.reset();
            } else {
                updateModalStatus('error', data?.message || 'Sunucu Hatası: Lütfen daha sonra tekrar deneyin.');
            }
        } catch (err) {
            updateModalStatus('error', 'Sunucuya bağlanırken bir hata oluştu.');
        }
    };

    form.addEventListener('submit', (e) => {
        e.preventDefault();
        e.stopPropagation();
        e.stopImmediatePropagation();
        doSubmit();
    }, true);

    if (submitBtn) {
        submitBtn.addEventListener('click', (e) => {
            e.preventDefault();
            doSubmit();
        }, true);
    }
});
