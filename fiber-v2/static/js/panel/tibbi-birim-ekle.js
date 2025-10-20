// Tibbi Birim Ekle Page JS - adapted from sube-ekle.js

class TibbiBirimFormHandler {
    constructor() {
        this.form = document.getElementById('tibbiBirimForm');
        this.submitBtn = document.getElementById('submitBtn');
        this.init();
    }

    init() {
        this.setupFormSubmission();
        this.setupFileUploads();
        this.setupFormValidation();
        this.setupToggleSwitches();
        this.setupNameToUrlSync();
    }

    setupFormSubmission() {
        this.form.addEventListener('submit', (e) => {
            if (!this.validateForm()) {
                e.preventDefault();
                return;
            }
            this.setLoadingState(true);
        });
    }

    setupFileUploads() {
        const fileUploadAreas = document.querySelectorAll('.file-upload-area');
        fileUploadAreas.forEach(area => {
            const input = area.querySelector('.file-input');
            const content = area.querySelector('.file-upload-content');
            const preview = area.querySelector('.file-preview');
            const previewImage = area.querySelector('.preview-image');
            const removeBtn = area.querySelector('.file-remove');
            const browseLink = area.querySelector('.file-browse');

            browseLink.addEventListener('click', (e) => {
                e.preventDefault();
                input.click();
            });

            area.addEventListener('click', (e) => {
                if (e.target === area || e.target === content) {
                    input.click();
                }
            });

            area.addEventListener('dragover', (e) => {
                e.preventDefault();
                area.classList.add('dragover');
            });

            area.addEventListener('dragleave', (e) => {
                e.preventDefault();
                if (!area.contains(e.relatedTarget)) {
                    area.classList.remove('dragover');
                }
            });

            area.addEventListener('drop', (e) => {
                e.preventDefault();
                area.classList.remove('dragover');
                const files = e.dataTransfer.files;
                if (files.length > 0) {
                    this.handleFileSelect(input, files[0], content, preview, previewImage);
                }
            });

            input.addEventListener('change', (e) => {
                if (e.target.files.length > 0) {
                    this.handleFileSelect(input, e.target.files[0], content, preview, previewImage);
                }
            });

            removeBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                this.removeFile(input, content, preview);
            });
        });
    }

    handleFileSelect(input, file, content, preview, previewImage) {
        const allowedTypes = input.accept.split(',').map(type => type.trim());
        const isValidType = allowedTypes.some(type => {
            if (type.startsWith('.')) return file.name.toLowerCase().endsWith(type.toLowerCase());
            if (type.includes('*')) return file.type.startsWith(type.split('/')[0]);
            return file.type === type;
        });
        if (!isValidType) {
            this.showErrorModal('Geçersiz dosya türü. Lütfen uygun bir dosya seçin.');
            return;
        }
        const maxSize = (typeof window !== 'undefined' && typeof maxUploadSize !== 'undefined')
            ? Number(maxUploadSize)
            : 5 * 1024 * 1024;
        if (file.size > maxSize) {
            const allowedMb = Math.max(1, Math.round(maxSize / (1024 * 1024)));
            this.showErrorModal(`Dosya boyutu ${allowedMb}MB'dan büyük olamaz.`);
            return;
        }
        const dt = new DataTransfer();
        dt.items.add(file);
        input.files = dt.files;
        
        // Show preview for both images and videos
        if (file.type.startsWith('image/')) {
            const reader = new FileReader();
            reader.onload = (e) => {
                previewImage.src = e.target.result;
                content.style.display = 'none';
                preview.style.display = 'flex';
            };
            reader.readAsDataURL(file);
        } else if (file.type.startsWith('video/')) {
            // For videos, show a video preview
            const reader = new FileReader();
            reader.onload = (e) => {
                // Create video element for preview
                const videoPreview = document.createElement('video');
                videoPreview.src = e.target.result;
                videoPreview.controls = true;
                videoPreview.style.maxWidth = '100%';
                videoPreview.style.maxHeight = '200px';
                
                // Replace the image preview with video
                previewImage.style.display = 'none';
                if (preview.querySelector('video')) {
                    preview.querySelector('video').remove();
                }
                preview.insertBefore(videoPreview, previewImage);
                
                content.style.display = 'none';
                preview.style.display = 'flex';
            };
            reader.readAsDataURL(file);
        } else {
            // For other file types, just show the file name
            content.style.display = 'none';
            preview.style.display = 'flex';
            previewImage.style.display = 'none';
            if (!preview.querySelector('.file-name-display')) {
                const fileNameDisplay = document.createElement('div');
                fileNameDisplay.className = 'file-name-display';
                fileNameDisplay.innerHTML = `<i class="icon-file"></i> ${file.name}`;
                preview.insertBefore(fileNameDisplay, previewImage);
            }
        }
    }

    removeFile(input, content, preview) {
        input.value = '';
        content.style.display = 'block';
        preview.style.display = 'none';
        
        // Clean up video preview and file name display
        const videoPreview = preview.querySelector('video');
        if (videoPreview) {
            videoPreview.remove();
        }
        
        const fileNameDisplay = preview.querySelector('.file-name-display');
        if (fileNameDisplay) {
            fileNameDisplay.remove();
        }
        
        // Reset image preview
        const previewImage = preview.querySelector('.preview-image');
        if (previewImage) {
            previewImage.src = '';
            previewImage.style.display = 'block';
        }
    }

    setupFormValidation() {
        const requiredFields = this.form.querySelectorAll('[required]');
        requiredFields.forEach(field => {
            field.addEventListener('blur', () => this.validateField(field));
            field.addEventListener('input', () => { if (field.classList.contains('error')) this.validateField(field); });
        });
    }

    setupToggleSwitches() {
        const toggles = this.form.querySelectorAll('.toggle-switch input[type="checkbox"]');
        toggles.forEach(toggle => {
            const updateToggleText = () => {
                const textElement = toggle.parentNode.querySelector('.toggle-text');
                if (textElement) textElement.textContent = toggle.checked ? 'Aktif' : 'Pasif';
            };
            toggle.addEventListener('change', updateToggleText);
            updateToggleText();
        });
    }

    setupNameToUrlSync() {
        const nameField = document.getElementById('name');
        const urlField = document.getElementById('url_name');
        if (nameField && urlField) {
            nameField.addEventListener('input', () => {
                if (!urlField.value || urlField.dataset.autoGenerated === 'true') {
                    const urlName = this.generateUrlName(nameField.value);
                    urlField.value = urlName;
                    urlField.dataset.autoGenerated = 'true';
                }
            });
            urlField.addEventListener('input', () => { if (urlField.value) urlField.dataset.autoGenerated = 'false'; });
        }
    }

    generateUrlName(name) {
        return name
            .toLowerCase()
            .replace(/ç/g, 'c').replace(/ğ/g, 'g').replace(/ı/g, 'i').replace(/ö/g, 'o').replace(/ş/g, 's').replace(/ü/g, 'u')
            .replace(/[^a-z0-9]+/g, '-')
            .replace(/^-+|-+$/g, '');
    }

    validateField(field) {
        const value = field.value.trim();
        let isValid = true; let errorMessage = '';
        if (field.hasAttribute('required') && !value) { isValid = false; errorMessage = 'Bu alan zorunludur.'; }
        this.setFieldState(field, isValid, errorMessage);
        return isValid;
    }

    validateForm() {
        let isValid = true;
        const requiredFields = this.form.querySelectorAll('[required]');
        requiredFields.forEach(field => { if (!this.validateField(field)) isValid = false; });
        if (!isValid) {
            this.showErrorModal('Lütfen tüm alanları doğru şekilde doldurun.');
            const firstError = this.form.querySelector('.error');
            if (firstError) { firstError.scrollIntoView({ behavior: 'smooth', block: 'center' }); firstError.focus(); }
        }
        return isValid;
    }

    setFieldState(field, isValid, errorMessage) {
        const fieldGroup = field.closest('.form-group');
        let errorElement = fieldGroup.querySelector('.field-error');
        if (errorElement) errorElement.remove();
        field.classList.remove('error', 'success');
        field.classList.add(isValid ? 'success' : 'error');
        if (!isValid && errorMessage) {
            errorElement = document.createElement('div');
            errorElement.className = 'field-error';
            errorElement.innerHTML = `<i class="icon-alert-circle"></i> ${errorMessage}`;
            fieldGroup.appendChild(errorElement);
        }
    }

    setLoadingState(loading) {
        if (loading) {
            this.submitBtn.disabled = true;
            this.submitBtn.querySelector('.btn-text').style.opacity = '0';
            this.submitBtn.querySelector('.btn-loader').style.display = 'block';
            this.form.classList.add('form-loading');
        } else {
            this.submitBtn.disabled = false;
            this.submitBtn.querySelector('.btn-text').style.opacity = '1';
            this.submitBtn.querySelector('.btn-loader').style.display = 'none';
            this.form.classList.remove('form-loading');
        }
    }

    showErrorModal(message) {
        const modal = document.getElementById('errorModal');
        const messageElement = document.getElementById('errorMessage');
        messageElement.textContent = message;
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
    }
}

function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    modal.classList.remove('show');
    setTimeout(() => { modal.style.display = 'none'; }, 300);
}

document.addEventListener('DOMContentLoaded', () => {
    new TibbiBirimFormHandler();
    document.addEventListener('click', (e) => {
        if (e.target.classList.contains('modal')) { closeModal(e.target.id); }
    });
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            const visibleModal = document.querySelector('.modal.show');
            if (visibleModal) closeModal(visibleModal.id);
        }
    });
});

window.TibbiBirimFormHandler = TibbiBirimFormHandler;

