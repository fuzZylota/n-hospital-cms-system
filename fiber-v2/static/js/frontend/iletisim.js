// Contact form submission with modal
document.addEventListener('DOMContentLoaded', () => {
    const form = document.getElementById('contactForm');
    if (!form) return;

    // Modal oluştur
    const createModal = () => {
        const modal = document.createElement('div');
        modal.id = 'contact-modal';
        modal.className = 'randevu-modal';
        modal.innerHTML = `
            <div class="randevu-modal__overlay">
                <div class="randevu-modal__content">
                    <div class="randevu-modal__header">
                        <h3 class="randevu-modal__title">İletişim Talebi</h3>
                        <button class="randevu-modal__close" type="button">
                            <i class="icon-close"></i>
                        </button>
                    </div>
                    <div class="randevu-modal__body">
                        <div class="randevu-modal__status" id="contact-status">
                            <div class="randevu-modal__loading">
                                <div class="randevu-modal__spinner"></div>
                                <p class="randevu-modal__message">İletişim Talebi Gönderiliyor...</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;
        document.body.appendChild(modal);
        return modal;
    };

    let modalA11yCleanup = null;

    // Modal'ı göster
    const showModal = () => {
        let modal = document.getElementById('contact-modal');
        if (!modal) {
            modal = createModal();
        }
        modal.style.display = 'flex';
        document.body.style.overflow = 'hidden';
        
        // Close button event
        const closeBtn = modal.querySelector('.randevu-modal__close');
        closeBtn.removeEventListener('click', hideModal);
        closeBtn.addEventListener('click', hideModal);
        
        // Overlay click to close
        const overlay = modal.querySelector('.randevu-modal__overlay');
        overlay.onclick = (e) => {
            if (e.target === overlay) {
                hideModal();
            }
        };

        // Erişilebilirlik (A11y) İyileştirmeleri
        if (modalA11yCleanup) modalA11yCleanup();
        
        const previousFocus = document.activeElement;
        const escHandler = (e) => {
            if (e.key === 'Escape' || e.keyCode === 27) {
                hideModal();
            }
        };
        document.addEventListener('keydown', escHandler);
        
        // Modal açıldığında ilk interaktif elemana focus ol
        setTimeout(() => {
            const focusable = modal.querySelector('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])');
            if (focusable) focusable.focus();
        }, 50);
        
        modalA11yCleanup = () => {
            document.removeEventListener('keydown', escHandler);
            if (previousFocus && typeof previousFocus.focus === 'function') {
                previousFocus.focus();
            }
        };
    };

    // Modal'ı gizle
    const hideModal = () => {
        const modal = document.getElementById('contact-modal');
        if (modal) {
            modal.style.display = 'none';
            document.body.style.overflow = '';
            
            if (modalA11yCleanup) {
                modalA11yCleanup();
                modalA11yCleanup = null;
            }
        }
    };

    // Modal durumunu güncelle
    const updateModalStatus = (type, message) => {
        const statusDiv = document.getElementById('contact-status');
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
                    <button class="randevu-modal__btn randevu-modal__btn--primary" onclick="document.getElementById('contact-modal').style.display='none'; document.body.style.overflow='';">
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
                    <button class="randevu-modal__btn randevu-modal__btn--secondary" onclick="document.getElementById('contact-modal').style.display='none'; document.body.style.overflow='';">
                        Tekrar Dene
                    </button>
                </div>
            `;
        }
    };

    // Set dynamic color for icons
    const setupIconColor = () => {
        const iletisimIcon = document.querySelector('.iletisim-icon');
        if (iletisimIcon) {
            const rootStyle = getComputedStyle(document.documentElement);
            const primaryColor = rootStyle.getPropertyValue('--theme-primary-color').trim();
            if (primaryColor) {
                iletisimIcon.style.color = primaryColor;
            }
        }
        
        const contactInfoIcon = document.querySelector('.contact-info-icon');
        if (contactInfoIcon) {
            const rootStyle = getComputedStyle(document.documentElement);
            const primaryColor = rootStyle.getPropertyValue('--theme-primary-color').trim();
            if (primaryColor) {
                contactInfoIcon.style.color = primaryColor;
            }
        }
    };

    form.addEventListener('submit', async (e) => {
        e.preventDefault();

        // Get form data
        const formData = new FormData(form);
        
        // Validate required fields
        const requiredFields = ['first_name', 'last_name', 'email', 'subject', 'message'];
        for (const field of requiredFields) {
            if (!formData.get(field) || formData.get(field).toString().trim() === '') {
                updateModalStatus('error', 'Lütfen tüm zorunlu alanları doldurun.');
                showModal();
                return;
            }
        }

        // Validate KVKK approval
        const kvkkCheckbox = form.querySelector('#kvkk_approval');
        if (kvkkCheckbox && !kvkkCheckbox.checked) {
            updateModalStatus('error', 'Lütfen KVKK onay kutusunu işaretleyin.');
            showModal();
            return;
        }

        // Validate reCAPTCHA only if it exists
        const recaptchaWidget = form.querySelector('.g-recaptcha');
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
            
            // Add token to form data
            const recaptchaTokenInput = form.querySelector('input[name="recaptcha_token"]');
            if (recaptchaTokenInput) {
                formData.set('recaptcha_token', recaptchaTokenInput.value || recaptchaResponse);
            } else {
                formData.set('recaptcha_token', recaptchaResponse);
            }
        }

        // Modal'ı göster ve loading durumunu ayarla
        showModal();
        updateModalStatus('loading', 'İletişim Talebi Gönderiliyor...');

        try {
            const res = await fetch('/backend/add-contact-request', {
                method: 'POST',
                body: formData,
            });

            const data = await res.json();
            if (data && (data.status === 200 || data.status === 201)) {
                updateModalStatus('success', 'Mesajınız başarıyla gönderildi. En kısa sürede size dönüş yapacağız.');
                
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
                    // Reset reCAPTCHA verification state
                    if (typeof recaptchaVerifiedContact !== 'undefined') {
                        recaptchaVerifiedContact = false;
                    }
                }
                // Update button state
                if (typeof updateContactSubmitButtonState === 'function') {
                    updateContactSubmitButtonState();
                }
            } else {
                updateModalStatus('error', data?.message || 'Sunucu Hatası: Lütfen daha sonra tekrar deneyin.');
            }
        } catch (err) {
            updateModalStatus('error', 'Sunucuya bağlanırken bir hata oluştu. Lütfen internet bağlantınızı kontrol edin.');
        }
    });

    // Initialize icon colors
    setupIconColor();
});

