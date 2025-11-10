// Randevu talebi gönderimi
document.addEventListener('DOMContentLoaded', () => {
    const form = document.querySelector('.appointment-one__form');
    if (!form) return;

    // Modal oluştur
    const createModal = () => {
        const modal = document.createElement('div');
        modal.id = 'randevu-modal';
        modal.className = 'randevu-modal';
        modal.innerHTML = `
            <div class="randevu-modal__overlay">
                <div class="randevu-modal__content">
                    <div class="randevu-modal__header">
                        <h3 class="randevu-modal__title">Randevu Talebi</h3>
                        <button class="randevu-modal__close" type="button">
                            <i class="icon-close"></i>
                        </button>
                    </div>
                    <div class="randevu-modal__body">
                        <div class="randevu-modal__status" id="randevu-status">
                            <div class="randevu-modal__loading">
                                <div class="randevu-modal__spinner"></div>
                                <p class="randevu-modal__message">Randevu İsteği Gönderiliyor...</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;
        document.body.appendChild(modal);
        return modal;
    };

    // Modal'ı göster
    const showModal = () => {
        let modal = document.getElementById('randevu-modal');
        if (!modal) {
            modal = createModal();
        }
        modal.style.display = 'flex';
        document.body.style.overflow = 'hidden';
        
        // Close button event
        const closeBtn = modal.querySelector('.randevu-modal__close');
        closeBtn.addEventListener('click', hideModal);
        
        // Overlay click to close
        const overlay = modal.querySelector('.randevu-modal__overlay');
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) {
                hideModal();
            }
        });
    };

    // Modal'ı gizle
    const hideModal = () => {
        const modal = document.getElementById('randevu-modal');
        if (modal) {
            modal.style.display = 'none';
            document.body.style.overflow = '';
        }
    };

    // Modal durumunu güncelle
    const updateModalStatus = (type, message) => {
        const statusDiv = document.getElementById('randevu-status');
        if (!statusDiv) return;

        if (type === 'loading') {
            statusDiv.innerHTML = `
                <div class="randevu-modal__loading">
                    <div class="randevu-modal__spinner"></div>
                    <p class="randevu-modal__message">${message}</p>
                </div>
            `;
        } else if (type === 'success') {
            statusDiv.innerHTML = `
                <div class="randevu-modal__success">
                    <div class="randevu-modal__icon">
                        <i class="icon-check"></i>
                    </div>
                    <p class="randevu-modal__message">${message}</p>
                    <button class="randevu-modal__btn randevu-modal__btn--primary" onclick="document.getElementById('randevu-modal').style.display='none'; document.body.style.overflow='';">
                        Tamam
                    </button>
                </div>
            `;
        } else if (type === 'error') {
            statusDiv.innerHTML = `
                <div class="randevu-modal__error">
                    <div class="randevu-modal__icon">
                        <i class="icon-close"></i>
                    </div>
                    <p class="randevu-modal__message">${message}</p>
                    <button class="randevu-modal__btn randevu-modal__btn--secondary" onclick="document.getElementById('randevu-modal').style.display='none'; document.body.style.overflow='';">
                        Tekrar Dene
                    </button>
                </div>
            `;
        }
    };

    const convertDateForBackend = (dateValue) => {
        if (!dateValue || dateValue === '') {
            return '0001-01-01T00:00:00Z';
        }
        if (/^\d{4}-\d{2}-\d{2}$/.test(dateValue)) {
            return dateValue + 'T00:00:00Z';
        }
        if (String(dateValue).includes('00:00, 01/01/0001')) {
            return '0001-01-01T00:00:00Z';
        }
        const d = new Date(dateValue);
        if (!isNaN(d.getTime())) {
            return d.toISOString();
        }
        return '0001-01-01T00:00:00Z';
    };

    form.addEventListener('submit', async (e) => {
        e.preventDefault();

        const nameInput = form.querySelector('input[name="patient_first_name"]');
        const surnameInput = form.querySelector('input[name="patient_last_name"]');
        const emailInput = form.querySelector('input[name="patient_email"]');
        const phoneInput = form.querySelector('input[name="patient_phone"]');
        const dateInput = form.querySelector('input[name="preferred_date"]');
        const messageInput = form.querySelector('textarea[name="message"]');
        const sidSelect = form.querySelector('select[name="sid"]');
        const kvkkCheckbox = form.querySelector('#appointmentKVKK') || form.querySelector('#homepageAppointmentKVKK');
        const recaptchaTokenInput = form.querySelector('input[name="recaptcha_token"]');
        const recaptchaWidget = form.querySelector('.g-recaptcha');

        const payload = {
            patient_first_name: nameInput?.value || '',
            patient_last_name: surnameInput?.value || '',
            patient_phone: phoneInput?.value || '',
            patient_email: emailInput?.value || '',
            preferred_date: convertDateForBackend((dateInput?.value || '').trim()),
            sid: sidSelect?.value || '',
            message: messageInput?.value || '',
        };

        if (kvkkCheckbox && !kvkkCheckbox.checked) {
            updateModalStatus('error', 'Lütfen KVKK onay kutusunu işaretleyin.');
            showModal();
            return;
        }

        // Validate reCAPTCHA only if it exists
        if (recaptchaWidget) {
            if (typeof grecaptcha === 'undefined') {
                updateModalStatus('error', 'Güvenlik doğrulaması yükleniyor, lütfen bekleyin...');
                showModal();
                return;
            }
            const recaptchaResponse = grecaptcha.getResponse();
            if (!recaptchaResponse || recaptchaResponse === '') {
                updateModalStatus('error', 'Lütfen güvenlik doğrulamasını tamamlayın.');
                showModal();
                return;
            }
            
            // Add token to payload
            if (recaptchaTokenInput) {
                payload.recaptcha_token = recaptchaTokenInput.value || recaptchaResponse;
            } else {
                payload.recaptcha_token = recaptchaResponse;
            }
        }

        if (!payload.patient_first_name || !payload.patient_last_name || !payload.patient_phone || !payload.patient_email) {
            updateModalStatus('error', 'Lütfen ad, soyad, telefon ve e-posta alanlarını doldurun.');
            showModal();
            return;
        }

        // Modal'ı göster ve loading durumunu ayarla
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
                
                // WebSocket bildirimini gönder
                try {
                    const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws';
                    const url = `${scheme}://${window.location.host}/backend/notifications`;
                    const ws = new WebSocket(url, 'randevu');
                    payload.rrid = data.rrid;
                    ws.addEventListener('open', () => {
                        const message = {
                            uid: null,
                            //insert_form: 'randevu',
                            message: JSON.stringify(payload),
                        };
                        ws.send(JSON.stringify(message));
                        // kısa bir süre sonra kapat
                        setTimeout(() => { try { ws.close(); } catch (_) {} }, 500);
                    });
                } catch (_) {
                    // sessizce geç
                }
                form.reset();
                // Reset reCAPTCHA only if it exists
                const recaptchaWidget = form.querySelector('.g-recaptcha');
                if (recaptchaWidget) {
                    if (typeof grecaptcha !== 'undefined') {
                        grecaptcha.reset();
                    }
                    // Reset reCAPTCHA token input
                    const recaptchaTokenInput = form.querySelector('input[name="recaptcha_token"]');
                    if (recaptchaTokenInput) {
                        recaptchaTokenInput.value = '';
                    }
                    // Reset reCAPTCHA verification state (both standalone and homepage)
                    if (typeof recaptchaVerified !== 'undefined') {
                        recaptchaVerified = false;
                    }
                    if (typeof recaptchaVerifiedHomepage !== 'undefined') {
                        recaptchaVerifiedHomepage = false;
                    }
                }
                // Update button states
                if (typeof updateSubmitButtonState === 'function') {
                    updateSubmitButtonState();
                }
                if (typeof updateHomepageSubmitButtonState === 'function') {
                    updateHomepageSubmitButtonState();
                }
            } else {
                updateModalStatus('error', data?.message || 'Sunucu Hatası: Lütfen daha sonra tekrar deneyin.');
            }
        } catch (err) {
            updateModalStatus('error', 'Sunucuya bağlanırken bir hata oluştu. Lütfen internet bağlantınızı kontrol edin.');
        }
    });
});


