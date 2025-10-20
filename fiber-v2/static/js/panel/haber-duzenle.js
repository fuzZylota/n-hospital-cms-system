/**
 * Haber Duzenle (News Edit) Page JavaScript
 * Handles general settings form, picture uploads, and picture deletion
 */

class HaberEditHandler {
    constructor() {
        console.log('[HaberEditHandler] constructor');
        this.hid = window.pageData.hid;
        console.log('[HaberEditHandler] hid:', this.hid);
        this.generalForm = document.getElementById('generalForm');
        this.pictureForm = document.getElementById('pictureForm');
        this.generalSubmitBtn = document.getElementById('generalSubmitBtn');
        this.pictureSubmitBtn = document.getElementById('pictureSubmitBtn');
        this.currentDeleteTarget = null;
        this.originalValues = {};
        
        this.init();
    }

    init() {
        console.log('[HaberEditHandler] init() start');
        try {
            this.initializeOriginalValues();
            if (!this.generalForm) {
                throw new Error('Genel form bulunamadı (id="generalForm").');
            }
            if (!this.pictureForm) {
                console.warn('Picture form bulunamadı (id="pictureForm").');
            }
            this.setupGeneralForm();
            if (this.pictureForm) this.setupPictureForm();
            this.setupFileUploads();
            this.setupFormValidation();
            this.setupToggleSwitches();
            this.setupPictureDeletion();
            this.setupUrlGeneration();
            console.log('[HaberEditHandler] init() complete');
        } catch (err) {
            console.error('Initialization error:', err);
            this.ensureModalExists('error');
            this.showErrorModal('Sayfa başlatılırken hata: ' + (err?.message || err));
        }
    }

    /**
     * Store original values for change detection
     */
    initializeOriginalValues() {
        const altTextInput = document.getElementById('haber_media_alt_text');
        const titleInput = document.getElementById('haber_media_title');
        
        if (altTextInput) {
            this.originalValues.altText = altTextInput.value;
        }
        if (titleInput) {
            this.originalValues.title = titleInput.value;
        }
    }

    /**
     * Setup general settings form (AJAX JSON submission)
     */
    setupGeneralForm() {
        console.log('[setupGeneralForm] attaching submit handler');
        this.generalForm.addEventListener('submit', async (e) => {
            console.log('[generalForm] submit event');
            e.preventDefault();
            
            const valid = this.validateGeneralForm();
            console.log('[generalForm] validateGeneralForm =>', valid);
            if (!valid) {
                this.showErrorModal('Lütfen zorunlu alanları doldurun.');
                return;
            }

            this.setGeneralLoadingState(true);
            console.log('[generalForm] set loading state true');

            try {
                // Show immediate feedback while submitting (use existing modal only)
                const pendingEl = document.getElementById('successMessage');
                if (pendingEl) pendingEl.textContent = 'Gönderiliyor...';
                const successModal = document.getElementById('successModal');
                if (successModal) {
                    successModal.style.setProperty('display', 'flex', 'important');
                    document.body.style.overflow = 'hidden';
                    console.log('[generalForm] success modal opened (pending)');
                }

                // Collect all form data
                const data = {};
                const allInputs = this.generalForm.querySelectorAll('input, textarea, select');
                
                allInputs.forEach(input => {
                    if (input.type === 'checkbox') {
                        data[input.name] = input.checked;
                    } else if (input.type === 'number') {
                        data[input.name] = parseFloat(input.value) || 0;
                    } else if (input.type === 'hidden') {
                        if (input.value === 'true') {
                            data[input.name] = true;
                        } else if (input.value === 'false') {
                            data[input.name] = false;
                        } else if (input.name.includes('old_') && document.querySelector(`[name="${input.name.replace('old_', '')}"]`)?.type === 'number') {
                            data[input.name] = parseFloat(input.value) || 0;
                        } else {
                            data[input.name] = input.value;
                        }
                    } else {
                        data[input.name] = input.value;
                    }
                });
                console.log('[generalForm] collected data:', JSON.stringify(data));

                // Format publish_date for backend
                if (data.publish_date) {
                    data.publish_date = this.convertDateForBackend(data.publish_date);
                }
                if (data.old_publish_date) {
                    data.old_publish_date = this.convertDateForBackend(data.old_publish_date);
                }
                console.log('[generalForm] formatted dates:', { publish_date: data.publish_date, old_publish_date: data.old_publish_date });

                const response = await fetch(`/backend/news/${this.hid}/edit`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(data)
                });
                console.log('[generalForm] fetch done - response.ok:', response.ok, 'status:', response.status);

                let result = {};
                try {
                    result = await response.json();
                } catch (_) {
                    // Non-JSON response; proceed using HTTP status
                }
                console.log('[generalForm] parsed result:', result);

                const statusVal = Number(result?.status);
                const isSuccess = response.ok || statusVal === 200 || statusVal === 201;
                console.log('[generalForm] isSuccess:', isSuccess);

                if (isSuccess) {
                    this.showSuccessModal(result?.message || 'Haber bilgileri başarıyla güncellendi.');
                    console.log('[generalForm] showSuccessModal invoked');
                } else {
                    throw new Error(result?.message || `Güncelleme işlemi başarısız oldu. (HTTP ${response.status})`);
                }

            } catch (error) {
                console.error('General form submission error:', error);
                this.showErrorModal('Haber bilgileri güncellenirken bir hata oluştu: ' + error.message);
                console.log('[generalForm] showErrorModal invoked');
            } finally {
                this.setGeneralLoadingState(false);
                console.log('[generalForm] set loading state false');
            }
        });
    }

    /**
     * Setup picture form (FormData submission)
     */
    setupPictureForm() {
        console.log('[setupPictureForm] attaching submit handler');
        this.pictureForm.addEventListener('submit', async (e) => {
            console.log('[pictureForm] submit event');
            e.preventDefault();
            
            if (!this.validatePictureForm()) {
                this.showErrorModal('Lütfen bir değişiklik yapın.');
                console.log('[pictureForm] validation failed');
                return;
            }

            this.setPictureLoadingState(true);
            console.log('[pictureForm] set loading state true');

            try {
                const formData = new FormData(this.pictureForm);
                
                // Add old values for comparison
                formData.append('old_haber_media_alt_text', this.originalValues.altText || '');
                formData.append('old_haber_media_title', this.originalValues.title || '');
                console.log('[pictureForm] formData prepared');

                const response = await fetch(`/backend/news/${this.hid}/update-picture`, {
                    method: 'POST',
                    body: formData
                });
                console.log('[pictureForm] fetch done - response.ok:', response.ok, 'status:', response.status);

                let result = {};
                try {
                    result = await response.json();
                } catch (_) {
                    // Non-JSON response; proceed using HTTP status
                }
                console.log('[pictureForm] parsed result:', result);

                const statusVal = Number(result?.status);
                const isSuccess = response.ok || statusVal === 200 || statusVal === 201;
                console.log('[pictureForm] isSuccess:', isSuccess);

                if (isSuccess) {
                    this.showSuccessModal(result?.message || 'Haber görseli başarıyla güncellendi.');
                    console.log('[pictureForm] showSuccessModal invoked');
                    setTimeout(() => {
                        window.location.reload();
                    }, 1200);
                } else {
                    throw new Error(result?.message || `Görsel güncelleme işlemi başarısız oldu. (HTTP ${response.status})`);
                }

            } catch (error) {
                console.error('Picture form submission error:', error);
                this.showErrorModal('Haber görseli güncellenirken bir hata oluştu: ' + error.message);
                console.log('[pictureForm] showErrorModal invoked');
            } finally {
                this.setPictureLoadingState(false);
                console.log('[pictureForm] set loading state false');
            }
        });
    }

    /**
     * Setup file upload areas with drag & drop
     */
    setupFileUploads() {
        const fileInput = document.getElementById('haber_media_path');
        const uploadArea = document.querySelector('.file-upload-area');
        const preview = document.querySelector('.file-preview');
        const previewImage = preview.querySelector('.preview-image');
        const removeBtn = preview.querySelector('.file-remove');
        const content = document.querySelector('.file-upload-content');

        // Click to browse
        uploadArea.addEventListener('click', () => {
            fileInput.click();
        });

        // File selection
        fileInput.addEventListener('change', (e) => {
            const file = e.target.files[0];
            if (file) {
                this.handleFileSelect(fileInput, file, content, preview, previewImage, uploadArea);
            }
        });

        // Drag & drop
        uploadArea.addEventListener('dragover', (e) => {
            e.preventDefault();
            uploadArea.classList.add('dragover');
        });

        uploadArea.addEventListener('dragleave', (e) => {
            e.preventDefault();
            uploadArea.classList.remove('dragover');
        });

        uploadArea.addEventListener('drop', (e) => {
            e.preventDefault();
            uploadArea.classList.remove('dragover');
            
            const files = e.dataTransfer.files;
            if (files.length > 0) {
                const file = files[0];
                if (file.type.startsWith('image/')) {
                    fileInput.files = files;
                    this.handleFileSelect(fileInput, file, content, preview, previewImage, uploadArea);
                } else {
                    this.showErrorModal('Lütfen sadece resim dosyası seçin.');
                }
            }
        });

        // Remove file
        removeBtn.addEventListener('click', () => {
            this.removeFile(fileInput, content, preview, uploadArea);
        });
    }

    /**
     * Handle file selection and preview
     */
    handleFileSelect(input, file, content, preview, previewImage, area) {
        // Validate file type
        if (!file.type.startsWith('image/')) {
            this.showErrorModal('Lütfen sadece resim dosyası seçin.');
            return;
        }

        // Validate file size (5MB)
        if (file.size > 5 * 1024 * 1024) {
            this.showErrorModal('Dosya boyutu 5MB\'dan büyük olamaz.');
            return;
        }

        // Show preview
        const reader = new FileReader();
        reader.onload = (e) => {
            previewImage.src = e.target.result;
            preview.style.display = 'block';
            content.style.display = 'none';
            area.classList.add('has-file');
        };
        reader.readAsDataURL(file);
    }

    /**
     * Remove selected file
     */
    removeFile(input, content, preview, area) {
        input.value = '';
        preview.style.display = 'none';
        content.style.display = 'block';
        area.classList.remove('has-file');
    }

    /**
     * Setup form validation
     */
    setupFormValidation() {
        // Real-time validation for required fields
        const requiredFields = this.generalForm.querySelectorAll('[required]');
        requiredFields.forEach(field => {
            field.addEventListener('blur', () => {
                this.validateField(field);
            });
        });

        // Email validation
        const emailField = document.getElementById('email');
        if (emailField) {
            emailField.addEventListener('blur', () => {
                this.validateEmail(emailField);
            });
        }

        // URL validation
        const websiteField = document.getElementById('website');
        if (websiteField) {
            websiteField.addEventListener('blur', () => {
                this.validateUrl(websiteField);
            });
        }
    }

    /**
     * Setup toggle switches
     */
    setupToggleSwitches() {
        const toggles = document.querySelectorAll('.toggle-switch input[type="checkbox"]');
        
        toggles.forEach(toggle => {
            const updateToggleText = () => {
                const label = toggle.nextElementSibling;
                const textSpan = label.querySelector('.toggle-text');
                
                if (toggle.id === 'is_featured') {
                    textSpan.textContent = toggle.checked ? 'Öne Çıkan' : 'Normal';
                } else if (toggle.id === 'is_published') {
                    textSpan.textContent = toggle.checked ? 'Yayınlanmış' : 'Taslak';
                }
            };

            toggle.addEventListener('change', updateToggleText);
            updateToggleText(); // Initial call
        });
    }

    /**
     * Setup picture deletion
     */
    setupPictureDeletion() {
        console.log('[setupPictureDeletion] wiring delete buttons');
        const deleteButtons = document.querySelectorAll('.btn-delete-media');
        const confirmBtn = document.getElementById('confirmDeletePictureBtn');
        const deleteModal = document.getElementById('deletePictureModal');

        deleteButtons.forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.preventDefault();
                this.currentDeleteTarget = btn.dataset.hid;
                console.log('[delete] clicked, target hid:', this.currentDeleteTarget);
                if (deleteModal) {
                    this.openGenericModal('deletePictureModal');
                    console.log('[delete] opened delete modal via openGenericModal');
                } else {
                    // If modal not present, ask with confirm and proceed
                    const ok = window.confirm('Bu görseli silmek istediğinizden emin misiniz?');
                    if (ok && this.currentDeleteTarget) {
                        console.log('[delete] window.confirm ok, proceed deletion');
                        this.deletePicture(this.currentDeleteTarget);
                    }
                }
            });
        });

        if (confirmBtn) {
            confirmBtn.addEventListener('click', () => {
                if (this.currentDeleteTarget) {
                    console.log('[delete] confirm click -> deletePicture');
                    this.deletePicture(this.currentDeleteTarget);
                }
            });
        }
    }

    /**
     * Open a generic modal by id with forced visible styles
     */
    openGenericModal(modalId) {
        const modal = document.getElementById(modalId);
        if (!modal) {
            console.warn('[modal] openGenericModal: modal not found', modalId);
            return;
        }
        modal.style.setProperty('display', 'flex', 'important');
        modal.style.setProperty('position', 'fixed', 'important');
        modal.style.setProperty('inset', '0', 'important');
        modal.style.setProperty('opacity', '1', 'important');
        modal.style.setProperty('visibility', 'visible', 'important');
        modal.style.setProperty('pointer-events', 'auto', 'important');
        modal.style.setProperty('z-index', '100000', 'important');
        modal.setAttribute('role', 'dialog');
        modal.setAttribute('aria-modal', 'true');
        document.body.style.overflow = 'hidden';

        try {
            const cs = window.getComputedStyle(modal);
            const rect = modal.getBoundingClientRect();
            console.log('[modal] openGenericModal computed styles:', {
                display: cs.display,
                visibility: cs.visibility,
                opacity: cs.opacity,
                width: rect.width,
                height: rect.height
            });
        } catch (_) {}
    }

    /**
     * Setup URL generation from title
     */
    setupUrlGeneration() {
        const titleInput = document.getElementById('title');
        const urlInput = document.getElementById('url_name');

        if (titleInput && urlInput) {
            titleInput.addEventListener('input', () => {
                if (!urlInput.value || urlInput.dataset.autoGenerated === 'true') {
                    const urlFriendly = this.generateUrlFriendlyString(titleInput.value);
                    urlInput.value = urlFriendly;
                    urlInput.dataset.autoGenerated = 'true';
                }
            });

            urlInput.addEventListener('input', () => {
                urlInput.dataset.autoGenerated = 'false';
            });
        }
    }

    /**
     * Generate URL-friendly string
     */
    generateUrlFriendlyString(text) {
        return text
            .toLowerCase()
            .replace(/[çğıöşü]/g, (match) => {
                const map = {
                    'ç': 'c', 'ğ': 'g', 'ı': 'i', 'ö': 'o', 'ş': 's', 'ü': 'u'
                };
                return map[match] || match;
            })
            .replace(/[^a-z0-9\s-]/g, '')
            .replace(/\s+/g, '-')
            .replace(/-+/g, '-')
            .trim();
    }

    /**
     * Delete picture via AJAX
     */
    async deletePicture(target) {
        const confirmBtn = document.getElementById('confirmDeletePictureBtn');
        this.setDeletePictureLoadingState(true);

        try {
            const response = await fetch(`/backend/news/${target}/delete-picture`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });

            let result = {};
            try {
                result = await response.json();
            } catch (_) {
                // Non-JSON response; proceed using HTTP status
            }

            const statusVal = Number(result?.status);
            const isSuccess = response.ok || statusVal === 200 || statusVal === 201;

            if (isSuccess) {
                this.showSuccessModal(result?.message || 'Haber görseli başarıyla silindi.');
                setTimeout(() => {
                    window.location.reload();
                }, 1200);
            } else {
                throw new Error(result?.message || `Görsel silme işlemi başarısız oldu. (HTTP ${response.status})`);
            }

        } catch (error) {
            console.error('Delete picture error:', error);
            this.showErrorModal('Haber görseli silinirken bir hata oluştu: ' + error.message);
        } finally {
            this.setDeletePictureLoadingState(false);
            this.closeModal('deletePictureModal');
        }
    }

    /**
     * Validate individual field
     */
    validateField(field) {
        const value = field.value.trim();
        const isValid = value !== '';
        
        this.setFieldState(field, isValid, isValid ? '' : 'Bu alan zorunludur.');
        return isValid;
    }

    /**
     * Validate email field
     */
    validateEmail(field) {
        const value = field.value.trim();
        if (value === '') return true; // Optional field
        
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        const isValid = emailRegex.test(value);
        
        this.setFieldState(field, isValid, isValid ? '' : 'Geçerli bir e-posta adresi girin.');
        return isValid;
    }

    /**
     * Validate URL field
     */
    validateUrl(field) {
        const value = field.value.trim();
        if (value === '') return true; // Optional field
        
        try {
            new URL(value);
            this.setFieldState(field, true, '');
            return true;
        } catch {
            this.setFieldState(field, false, 'Geçerli bir URL girin.');
            return false;
        }
    }

    /**
     * Set field validation state
     */
    setFieldState(field, isValid, errorMessage) {
        field.classList.remove('error', 'success');
        field.classList.add(isValid ? 'success' : 'error');
        
        // Remove existing error message
        const existingError = field.parentNode.querySelector('.field-error');
        if (existingError) {
            existingError.remove();
        }
        
        // Add new error message if invalid
        if (!isValid && errorMessage) {
            const errorDiv = document.createElement('div');
            errorDiv.className = 'field-error';
            errorDiv.innerHTML = `<i class="fas fa-exclamation-circle"></i> ${errorMessage}`;
            field.parentNode.appendChild(errorDiv);
        }
    }

    /**
     * Validate general form
     */
    validateGeneralForm() {
        let isValid = true;
        
        // Validate required fields
        const requiredFields = this.generalForm.querySelectorAll('[required]');
        requiredFields.forEach(field => {
            if (!this.validateField(field)) {
                isValid = false;
            }
        });

        // Validate email if provided
        const emailField = document.getElementById('email');
        if (emailField && emailField.value.trim() !== '') {
            if (!this.validateEmail(emailField)) {
                isValid = false;
            }
        }

        // Validate website if provided
        const websiteField = document.getElementById('website');
        if (websiteField && websiteField.value.trim() !== '') {
            if (!this.validateUrl(websiteField)) {
                isValid = false;
            }
        }

        return isValid;
    }

    /**
     * Validate picture form
     */
    validatePictureForm() {
        const fileInput = document.getElementById('haber_media_path');
        const altTextInput = document.getElementById('haber_media_alt_text');
        const titleInput = document.getElementById('haber_media_title');
        
        // Check if file is selected or if metadata has changed
        const hasFile = fileInput.files.length > 0;
        const altTextChanged = altTextInput.value !== this.originalValues.altText;
        const titleChanged = titleInput.value !== this.originalValues.title;
        
        if (!hasFile && !altTextChanged && !titleChanged) {
            this.showErrorModal('Lütfen bir değişiklik yapın.');
            return false;
        }
        
        return true;
    }

    /**
     * Set general form loading state
     */
    setGeneralLoadingState(loading) {
        this.generalSubmitBtn.disabled = loading;
        const btnText = this.generalSubmitBtn.querySelector('.btn-text');
        const btnLoader = this.generalSubmitBtn.querySelector('.btn-loader');
        
        if (loading) {
            btnText.style.display = 'none';
            btnLoader.style.display = 'flex';
        } else {
            btnText.style.display = 'inline';
            btnLoader.style.display = 'none';
        }
    }

    /**
     * Set picture form loading state
     */
    setPictureLoadingState(loading) {
        this.pictureSubmitBtn.disabled = loading;
        const btnText = this.pictureSubmitBtn.querySelector('.btn-text');
        const btnLoader = this.pictureSubmitBtn.querySelector('.btn-loader');
        
        if (loading) {
            btnText.style.display = 'none';
            btnLoader.style.display = 'flex';
        } else {
            btnText.style.display = 'inline';
            btnLoader.style.display = 'none';
        }
    }

    /**
     * Set delete picture loading state
     */
    setDeletePictureLoadingState(loading) {
        const confirmBtn = document.getElementById('confirmDeletePictureBtn');
        const btnText = confirmBtn.querySelector('.btn-text');
        const btnLoader = confirmBtn.querySelector('.btn-loader');
        
        confirmBtn.disabled = loading;
        
        if (loading) {
            btnText.style.display = 'none';
            btnLoader.style.display = 'flex';
        } else {
            btnText.style.display = 'inline';
            btnLoader.style.display = 'none';
        }
    }

    /**
     * Show success modal
     */
    showSuccessModal(message) {
        console.log('[modal] showSuccessModal called with:', message);
        this.openModalOrAlert('successModal', 'successMessage', message, 'success');
        this.updateInlineNotice(message, false);
    }

    /**
     * Show error modal
     */
    showErrorModal(message) {
        console.log('[modal] showErrorModal called with:', message);
        this.openModalOrAlert('errorModal', 'errorMessage', message, 'error');
        this.updateInlineNotice(message, true);
    }

    /**
     * Fallback toast/modal renderer in case modal elements are missing
     */
    renderFallbackToast(message, type) {
        const overlay = document.createElement('div');
        overlay.style.position = 'fixed';
        overlay.style.inset = '0';
        overlay.style.background = 'rgba(0,0,0,0.72)';
        overlay.style.display = 'flex';
        overlay.style.alignItems = 'center';
        overlay.style.justifyContent = 'center';
        overlay.style.zIndex = '10000';

        const box = document.createElement('div');
        box.style.maxWidth = '520px';
        box.style.width = 'calc(100% - 32px)';
        box.style.background = 'var(--bg-primary, #fff)';
        box.style.color = 'var(--text-primary, #111827)';
        box.style.borderRadius = '12px';
        box.style.boxShadow = '0 10px 25px rgba(0,0,0,0.25)';
        box.style.padding = '24px';
        box.style.textAlign = 'left';

        const title = document.createElement('h3');
        title.style.margin = '0 0 8px 0';
        title.style.fontSize = '18px';
        title.style.fontWeight = '700';
        title.textContent = type === 'success' ? 'Başarılı!' : 'Hata!';

        const msg = document.createElement('p');
        msg.style.margin = '0 0 16px 0';
        msg.style.color = 'var(--text-secondary, #6b7280)';
        msg.textContent = message;

        const actions = document.createElement('div');
        actions.style.display = 'flex';
        actions.style.justifyContent = 'flex-end';

        const btn = document.createElement('button');
        btn.textContent = 'Tamam';
        btn.style.padding = '10px 14px';
        btn.style.borderRadius = '8px';
        btn.style.border = 'none';
        btn.style.cursor = 'pointer';
        btn.style.background = type === 'success' ? '#2563eb' : '#dc2626';
        btn.style.color = '#fff';

        btn.addEventListener('click', () => document.body.removeChild(overlay));

        actions.appendChild(btn);
        box.appendChild(title);
        box.appendChild(msg);
        box.appendChild(actions);
        overlay.appendChild(box);

        document.body.appendChild(overlay);
    }

    /**
     * Inline notice updater
     */
    updateInlineNotice(message, isError) {
        const el = document.getElementById('inlineNotice');
        if (!el) return;
        el.classList.remove('error');
        if (isError) el.classList.add('error');
        el.textContent = message;
        el.style.display = 'block';
    }

    /**
     * Ensure success/error modals exist in DOM; create if missing
     */
    ensureModalExists(type) {
        console.log('[modal] ensureModalExists:', type);
        if (type === 'success') {
            if (!document.getElementById('successModal')) {
                console.log('[modal] creating successModal');
                const wrapper = document.createElement('div');
                wrapper.id = 'successModal';
                wrapper.className = 'modal';
                wrapper.style.display = 'none';
                wrapper.innerHTML = `
                    <div class="modal-content success">
                        <div class="modal-icon"><i class="fas fa-check-circle"></i></div>
                        <h3 class="modal-title">Başarılı!</h3>
                        <p class="modal-message" id="successMessage">İşlem başarıyla tamamlandı.</p>
                        <div class="modal-actions">
                            <button type="button" class="btn btn-primary" onclick="closeModal('successModal')">Tamam</button>
                        </div>
                    </div>`;
                document.body.appendChild(wrapper);
            }
        } else if (type === 'error') {
            if (!document.getElementById('errorModal')) {
                console.log('[modal] creating errorModal');
                const wrapper = document.createElement('div');
                wrapper.id = 'errorModal';
                wrapper.className = 'modal';
                wrapper.style.display = 'none';
                wrapper.innerHTML = `
                    <div class="modal-content error">
                        <div class="modal-icon"><i class="fas fa-exclamation-circle"></i></div>
                        <h3 class="modal-title">Hata!</h3>
                        <p class="modal-message" id="errorMessage">Bir hata oluştu. Lütfen tekrar deneyin.</p>
                        <div class="modal-actions">
                            <button type="button" class="btn btn-danger" onclick="closeModal('errorModal')">Tamam</button>
                        </div>
                    </div>`;
                document.body.appendChild(wrapper);
            }
        }
    }

    /**
     * Attempt to open the modal; if still not visible, fallback to alert()
     */
    openModalOrAlert(modalId, messageId, message, type) {
        console.log('[modal] openModalOrAlert:', { modalId, messageId, type, message });
        const modal = document.getElementById(modalId);
        if (!modal) {
            console.warn('[modal] not found, skipping open');
            return;
        }
        const msgEl = document.getElementById(messageId);
        if (msgEl) msgEl.textContent = message;
        modal.style.setProperty('display', 'flex', 'important');
        modal.style.setProperty('position', 'fixed', 'important');
        modal.style.setProperty('inset', '0', 'important');
        modal.style.setProperty('opacity', '1', 'important');
        modal.style.setProperty('visibility', 'visible', 'important');
        modal.style.setProperty('pointer-events', 'auto', 'important');
        modal.style.setProperty('z-index', '100000', 'important');
        modal.setAttribute('role', 'dialog');
        modal.setAttribute('aria-modal', 'true');
        document.body.style.overflow = 'hidden';

        // Log computed visibility for debugging
        try {
            const cs = window.getComputedStyle(modal);
            const rect = modal.getBoundingClientRect();
            const visible = cs.display !== 'none' && cs.visibility !== 'hidden' && parseFloat(cs.opacity || '1') > 0 && rect.width > 50 && rect.height > 50;
            console.log('[modal] computed visible:', visible, {
                display: cs.display,
                visibility: cs.visibility,
                opacity: cs.opacity,
                width: rect.width,
                height: rect.height
            });
            // If not visible, we'll rely on existing page behavior; no emergency overlay
        } catch (_) {
            // swallow
        }

        if (modalId === 'successModal') {
            window.clearTimeout(this.__successAutoCloseT);
            this.__successAutoCloseT = window.setTimeout(() => closeModal('successModal'), 1600);
        }
    }

    /**
     * Show delete picture modal
     */
    showDeletePictureModal() {
        const modal = document.getElementById('deletePictureModal');
        modal.style.display = 'flex';
    }

    /**
     * Close modal
     */
    closeModal(modalId) {
        const modal = document.getElementById(modalId);
        modal.style.display = 'none';
    }

    /**
     * Convert date for backend format
     */
    convertDateForBackend(dateValue) {
        if (!dateValue || dateValue === '') {
            return '0001-01-01T00:00:00Z';
        }

        if (/^\d{4}-\d{2}-\d{2}$/.test(dateValue)) {
            return dateValue + 'T00:00:00Z';
        }

        if (String(dateValue).includes('00:00, 01/01/0001')) {
            return '0001-01-01T00:00:00Z';
        }

        const date = new Date(dateValue);
        if (!isNaN(date.getTime())) {
            return date.toISOString();
        }

        return '0001-01-01T00:00:00Z';
    }
}

// Global modal close function
function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) modal.style.setProperty('display', 'none', 'important');
    document.body.style.overflow = '';
}

// Initialize when DOM is loaded
function __initHaberEdit() {
    const handler = new HaberEditHandler();
    window.__haberEditHandlerInstance = handler;

    ['successModal', 'errorModal', 'deletePictureModal'].forEach(id => {
        const m = document.getElementById(id);
        if (!m) return;
        m.addEventListener('click', (e) => {
            if (e.target === m) {
                closeModal(id);
            }
        });
    });

    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            ['successModal', 'errorModal', 'deletePictureModal'].forEach(id => closeModal(id));
        }
    });
}

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', __initHaberEdit);
} else {
    // DOM already ready (script likely at the end); init immediately
    __initHaberEdit();
}

// Global error handlers to surface runtime issues to the user as modals
window.addEventListener('error', (e) => {
    try {
        const handler = window.__haberEditHandlerInstance || null;
        if (handler && typeof handler.showErrorModal === 'function') {
            handler.showErrorModal('Bir hata oluştu: ' + (e?.message || 'Bilinmeyen hata'));
        }
    } catch (_) {}
});

window.addEventListener('unhandledrejection', (e) => {
    try {
        const handler = window.__haberEditHandlerInstance || null;
        if (handler && typeof handler.showErrorModal === 'function') {
            handler.showErrorModal('Beklenmeyen bir hata oluştu: ' + (e?.reason?.message || 'Bilinmeyen hata'));
        }
    } catch (_) {}
});
