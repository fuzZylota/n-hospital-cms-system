/**
 * Haber Ekle (Add News) Page JavaScript
 * Handles form submission, file uploads, validation, and date formatting
 */

class HaberFormHandler {
    constructor() {
        this.form = document.getElementById('haberForm');
        this.submitBtn = document.getElementById('submitBtn');
        this.isSubmitting = false;
        
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
        if (!this.form) return;

        this.form.addEventListener('submit', (e) => {
            e.preventDefault();
            
            if (this.isSubmitting) return;
            
            if (this.validateForm()) {
                this.submitForm();
            }
        });
    }

    setupFileUploads() {
        const fileInputs = document.querySelectorAll('.file-input');
        
        fileInputs.forEach(input => {
            const target = input.getAttribute('data-target') || input.id;
            const uploadArea = document.querySelector(`[data-target="${target}"]`);
            const content = uploadArea.querySelector('.file-upload-content');
            const preview = uploadArea.querySelector('.file-preview');
            const previewImage = preview.querySelector('.preview-image');
            const removeBtn = preview.querySelector('.file-remove');

            // File input change
            input.addEventListener('change', (e) => {
                if (e.target.files && e.target.files[0]) {
                    this.handleFileSelect(input, e.target.files[0], content, preview, previewImage);
                }
            });

            // Drag and drop
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
                
                if (e.dataTransfer.files && e.dataTransfer.files[0]) {
                    input.files = e.dataTransfer.files;
                    this.handleFileSelect(input, e.dataTransfer.files[0], content, preview, previewImage);
                }
            });

            // Remove file
            removeBtn.addEventListener('click', (e) => {
                e.preventDefault();
                this.removeFile(input, content, preview);
            });

            // Click to browse
            uploadArea.addEventListener('click', (e) => {
                if (e.target === uploadArea || e.target.closest('.file-upload-content')) {
                    input.click();
                }
            });
        });
    }

    handleFileSelect(input, file, content, preview, previewImage) {
        // Validate file type
        const allowedTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp'];
        if (!allowedTypes.includes(file.type)) {
            this.showErrorModal('Geçersiz dosya türü. Lütfen PNG, JPG veya WEBP dosyası seçin.');
            return;
        }

        // Validate file size (5MB)
        const maxSize = 5 * 1024 * 1024;
        if (file.size > maxSize) {
            this.showErrorModal('Dosya boyutu 5MB\'dan büyük olamaz.');
            return;
        }

        // Show preview
        const reader = new FileReader();
        reader.onload = (e) => {
            previewImage.src = e.target.result;
            content.style.display = 'none';
            preview.style.display = 'block';
        };
        reader.readAsDataURL(file);
    }

    removeFile(input, content, preview) {
        input.value = '';
        preview.style.display = 'none';
        content.style.display = 'block';
    }

    setupFormValidation() {
        const inputs = this.form.querySelectorAll('input, textarea');
        
        inputs.forEach(input => {
            // Real-time validation
            input.addEventListener('blur', () => {
                this.validateField(input);
            });

            input.addEventListener('input', () => {
                // Clear error state on input
                if (input.classList.contains('error')) {
                    this.setFieldState(input, true, '');
                }
            });
        });
    }

    setupToggleSwitches() {
        const toggles = document.querySelectorAll('.toggle-switch input[type="checkbox"]');
        
        toggles.forEach(toggle => {
            const updateToggleText = () => {
                const label = toggle.closest('.toggle-switch').querySelector('.toggle-text');
                if (label) {
                    if (toggle.checked) {
                        label.textContent = label.textContent.replace('Değil', '').trim();
                    } else {
                        if (!label.textContent.includes('Değil')) {
                            label.textContent = label.textContent + ' Değil';
                        }
                    }
                }
            };

            toggle.addEventListener('change', updateToggleText);
            updateToggleText(); // Initial state
        });
    }

    setupNameToUrlSync() {
        const titleInput = document.getElementById('title');
        const urlInput = document.getElementById('url_name');
        
        if (titleInput && urlInput) {
            titleInput.addEventListener('input', debounce(() => {
                if (urlInput.value === '' || urlInput.value === this.generateUrlName(titleInput.value)) {
                    urlInput.value = this.generateUrlName(titleInput.value);
                }
            }, 500));
        }
    }

    generateUrlName(name) {
        return name
            .toLowerCase()
            .replace(/[çÇ]/g, 'c')
            .replace(/[ğĞ]/g, 'g')
            .replace(/[ıİ]/g, 'i')
            .replace(/[öÖ]/g, 'o')
            .replace(/[şŞ]/g, 's')
            .replace(/[üÜ]/g, 'u')
            .replace(/[^a-z0-9\s-]/g, '')
            .replace(/\s+/g, '-')
            .replace(/-+/g, '-')
            .replace(/^-|-$/g, '');
    }

    validateField(field) {
        const value = field.value.trim();
        const isRequired = field.hasAttribute('required');
        
        if (isRequired && value === '') {
            this.setFieldState(field, false, 'Bu alan zorunludur');
            return false;
        }

        // Email validation
        if (field.type === 'email' && value !== '') {
            return this.validateEmail(field);
        }

        // URL validation
        if (field.type === 'url' && value !== '') {
            return this.validateUrl(field);
        }

        // Phone validation
        if (field.type === 'tel' && value !== '') {
            return this.validatePhone(field);
        }

        this.setFieldState(field, true, '');
        return true;
    }

    validateEmail(field) {
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        if (!emailRegex.test(field.value)) {
            this.setFieldState(field, false, 'Geçerli bir e-posta adresi girin');
            return false;
        }
        this.setFieldState(field, true, '');
        return true;
    }

    validateUrl(field) {
        try {
            new URL(field.value);
            this.setFieldState(field, true, '');
            return true;
        } catch {
            this.setFieldState(field, false, 'Geçerli bir URL girin');
            return false;
        }
    }

    validatePhone(field) {
        const phoneRegex = /^[\+]?[0-9\s\-\(\)]{10,}$/;
        if (!phoneRegex.test(field.value)) {
            this.setFieldState(field, false, 'Geçerli bir telefon numarası girin');
            return false;
        }
        this.setFieldState(field, true, '');
        return true;
    }

    setFieldState(field, isValid, errorMessage) {
        const errorElement = field.parentNode.querySelector('.field-error');
        
        field.classList.remove('error', 'success');
        
        if (isValid) {
            field.classList.add('success');
            if (errorElement) {
                errorElement.remove();
            }
        } else {
            field.classList.add('error');
            if (errorMessage) {
                if (errorElement) {
                    errorElement.textContent = errorMessage;
                } else {
                    const errorDiv = document.createElement('div');
                    errorDiv.className = 'field-error';
                    errorDiv.innerHTML = `<i class="icon-alert-circle"></i> ${errorMessage}`;
                    field.parentNode.appendChild(errorDiv);
                }
            }
        }
    }

    validateForm() {
        const requiredFields = this.form.querySelectorAll('[required]');
        let isValid = true;

        requiredFields.forEach(field => {
            if (!this.validateField(field)) {
                isValid = false;
            }
        });

        return isValid;
    }

    async submitForm() {
        if (this.isSubmitting) return;

        this.isSubmitting = true;
        this.setLoadingState(true);

        try {
            // Format publish_date before submission
            const publishDateInput = document.getElementById('publish_date');
            if (publishDateInput && publishDateInput.value) {
                publishDateInput.value = this.convertDateForBackend(publishDateInput.value);
            }

            const formData = new FormData(this.form);
            
            const response = await fetch(this.form.action, {
                method: 'POST',
                body: formData
            });

            if (response.ok) {
                const result = await response.text();
                if (result.includes('redirect')) {
                    // Extract redirect URL from response
                    const urlMatch = result.match(/href="([^"]+)"/);
                    if (urlMatch) {
                        window.location.href = urlMatch[1];
                    } else {
                        window.location.href = '/panel/haberler';
                    }
                } else {
                    this.showSuccessModal();
                }
            } else {
                this.showErrorModal('Server Hatası: Lütfen daha sonra tekrar deneyin.');
            }
        } catch (error) {
            console.error('Form submission error:', error);
            this.showErrorModal('Bir hata oluştu. Lütfen tekrar deneyin.');
        } finally {
            this.isSubmitting = false;
            this.setLoadingState(false);
        }
    }

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

    setLoadingState(loading) {
        const btnText = this.submitBtn.querySelector('.btn-text');
        const btnLoader = this.submitBtn.querySelector('.btn-loader');
        
        if (loading) {
            this.submitBtn.disabled = true;
            btnText.style.display = 'none';
            btnLoader.style.display = 'flex';
        } else {
            this.submitBtn.disabled = false;
            btnText.style.display = 'inline';
            btnLoader.style.display = 'none';
        }
    }

    showSuccessModal() {
        const modal = document.getElementById('successModal');
        if (modal) {
            modal.style.display = 'flex';
            setTimeout(() => modal.classList.add('show'), 10);
        }
    }

    showErrorModal(message) {
        const modal = document.getElementById('errorModal');
        const messageElement = document.getElementById('errorMessage');
        
        if (modal && messageElement) {
            messageElement.textContent = message;
            modal.style.display = 'flex';
            setTimeout(() => modal.classList.add('show'), 10);
        }
    }

    resetForm() {
        this.form.reset();
        
        // Reset file uploads
        const filePreviews = document.querySelectorAll('.file-preview');
        const fileContents = document.querySelectorAll('.file-upload-content');
        
        filePreviews.forEach(preview => {
            preview.style.display = 'none';
        });
        
        fileContents.forEach(content => {
            content.style.display = 'block';
        });
        
        // Clear validation states
        const inputs = this.form.querySelectorAll('input, textarea');
        inputs.forEach(input => {
            input.classList.remove('error', 'success');
            const errorElement = input.parentNode.querySelector('.field-error');
            if (errorElement) {
                errorElement.remove();
            }
        });
    }
}

// Global functions for modal interactions
function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
    }
}

function formatBytes(bytes, decimals = 2) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new HaberFormHandler();
});

// Export for potential external use
if (typeof module !== 'undefined' && module.exports) {
    module.exports = HaberFormHandler;
}
