// Job application form submission
document.addEventListener('DOMContentLoaded', () => {
    const form = document.getElementById('jobApplicationForm');
    if (!form) return;

    // Modal oluştur
    const createModal = () => {
        const modal = document.createElement('div');
        modal.id = 'job-application-modal';
        modal.className = 'randevu-modal';
        modal.innerHTML = `
            <div class="randevu-modal__overlay">
                <div class="randevu-modal__content">
                    <div class="randevu-modal__header">
                        <h3 class="randevu-modal__title">İş Başvurusu</h3>
                        <button class="randevu-modal__close" type="button">
                            <i class="icon-close"></i>
                        </button>
                    </div>
                    <div class="randevu-modal__body">
                        <div class="randevu-modal__status" id="job-application-status">
                            <div class="randevu-modal__loading">
                                <div class="randevu-modal__spinner"></div>
                                <p class="randevu-modal__message">İş Başvurusu Gönderiliyor...</p>
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
        let modal = document.getElementById('job-application-modal');
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
        const modal = document.getElementById('job-application-modal');
        if (modal) {
            modal.style.display = 'none';
            document.body.style.overflow = '';
        }
    };

    // Modal durumunu güncelle
    const updateModalStatus = (type, message) => {
        const statusDiv = document.getElementById('job-application-status');
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
                    <button class="randevu-modal__btn randevu-modal__btn--primary" onclick="document.getElementById('job-application-modal').style.display='none'; document.body.style.overflow='';">
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
                    <button class="randevu-modal__btn randevu-modal__btn--secondary" onclick="document.getElementById('job-application-modal').style.display='none'; document.body.style.overflow='';">
                        Tekrar Dene
                    </button>
                </div>
            `;
        }
    };

    form.addEventListener('submit', async (e) => {
        e.preventDefault();

        // Get form data
        const formData = new FormData(form);
        
        // Validate required fields
        const requiredFields = ['first_name', 'last_name', 'email', 'city', 'cv_file', 'cover_letter'];
        for (const field of requiredFields) {
            if (!formData.get(field) || formData.get(field).toString().trim() === '') {
                updateModalStatus('error', 'Lütfen tüm zorunlu alanları doldurun.');
                showModal();
                return;
            }
        }

        // Validate KVKK approval
        if (!formData.get('kvkk_approval')) {
            updateModalStatus('error', 'KVKK onayını vermelisiniz.');
            showModal();
            return;
        }

        // Modal'ı göster ve loading durumunu ayarla
        showModal();
        updateModalStatus('loading', 'İş Başvurusu Gönderiliyor...');

        try {
            const res = await fetch('/backend/add-job-application', {
                method: 'POST',
                body: formData,
            });

            const data = await res.json();
            if (data && (data.status === 200 || data.status === 201)) {
                updateModalStatus('success', 'Başvurunuz başarıyla gönderildi. En kısa sürede sizinle iletişime geçeceğiz.');
                
                // WebSocket bildirimini gönder
                try {
                    const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws';
                    const url = `${scheme}://${window.location.host}/backend/notifications`;
                    const ws = new WebSocket(url, 'is-basvurusu');
                    
                    ws.addEventListener('open', () => {
                        // Create message with all form data except cv_file and cover_letter
                        const messageData = {
                            first_name: formData.get('first_name'),
                            last_name: formData.get('last_name'),
                            email: formData.get('email'),
                            city: formData.get('city'),
                            languages: formData.get('languages'),
                            work_references: formData.get('work_references'),
                            jaid: data.jaid
                        };

                        const message = {
                            uid: null,
                            message: JSON.stringify(messageData),
                        };
                        ws.send(JSON.stringify(message));
                        // Close connection after sending
                        setTimeout(() => { try { ws.close(); } catch (_) {} }, 500);
                    });
                } catch (_) {
                    // Silently handle WebSocket errors
                }
                
                form.reset();
            } else {
                updateModalStatus('error', data?.message || 'Sunucu Hatası: Lütfen daha sonra tekrar deneyin.');
            }
        } catch (err) {
            updateModalStatus('error', 'Sunucuya bağlanırken bir hata oluştu. Lütfen internet bağlantınızı kontrol edin.');
        }
    });
});

