/**
 * Header Tusu Ekle (Header Button Add) Page JavaScript
 * Handles form submission, validation, icon preview, and user interactions
 */

class HeaderButtonFormHandler {
    constructor() {
        this.form = document.getElementById('headerButtonForm');
        this.submitBtn = document.getElementById('submitBtn');
        this.titleInput = document.getElementById('title');
        this.urlInput = document.getElementById('url');
        this.iconInput = document.getElementById('icon');
        this.targetSelect = document.getElementById('target');
        this.parentSelect = document.getElementById('parent_id');
        this.sortOrderInput = document.getElementById('sort_order');
        this.iconPreview = document.getElementById('iconPreview');
        
        this.init();
    }

    init() {
        this.setupFormSubmission();
        this.setupFormValidation();
        this.setupToggleSwitches();
        this.setupIconPreview();
        this.setupTargetHelp();
        this.setupParentSelection();
        this.setupUrlValidation();
        this.handleUrlErrors();
    }

    /**
     * Setup form submission
     */
    setupFormSubmission() {
        this.form.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            if (!this.validateForm()) {
                return;
            }

            await this.submitForm();
        });
    }

    /**
     * Setup icon preview
     */
    setupIconPreview() {
        this.iconInput.addEventListener('input', () => {
            this.updateIconPreview();
        });
        
        // Initialize preview
        this.updateIconPreview();
    }

    /**
     * Update icon preview
     */
    updateIconPreview() {
        const iconValue = this.iconInput.value.trim();
        const iconElement = this.iconPreview.querySelector('i');
        const textElement = this.iconPreview.querySelector('span');
        
        if (iconValue) {
            // Validate FontAwesome icon format
            const iconRegex = /^(fas|far|fab|fal|fad)\s+fa-[\w-]+$/;
            
            if (iconRegex.test(iconValue)) {
                iconElement.className = iconValue;
                textElement.textContent = 'Geçerli ikon';
                this.iconPreview.classList.remove('invalid');
                this.iconPreview.classList.add('valid');
            } else {
                iconElement.className = 'fas fa-exclamation-triangle';
                textElement.textContent = 'Geçersiz ikon formatı';
                this.iconPreview.classList.remove('valid');
                this.iconPreview.classList.add('invalid');
            }
        } else {
            iconElement.className = 'fas fa-link';
            textElement.textContent = 'Önizleme';
            this.iconPreview.classList.remove('valid', 'invalid');
        }
    }

    /**
     * Setup target help text
     */
    setupTargetHelp() {
        this.targetSelect.addEventListener('change', () => {
            this.updateTargetHelp();
        });
        
        // Initialize help text
        this.updateTargetHelp();
    }

    /**
     * Update target help text
     */
    updateTargetHelp() {
        const helpText = document.getElementById('target-help-text');
        const targetValue = this.targetSelect.value;
        
        const descriptions = {
            '_self': 'Bağlantı aynı pencerede açılır (varsayılan davranış)',
            '_blank': 'Bağlantı yeni bir pencerede veya sekmede açılır'
        };
        
        if (helpText && descriptions[targetValue]) {
            helpText.textContent = descriptions[targetValue];
        }
    }

    /**
     * Setup parent selection
     */
    setupParentSelection() {
        this.parentSelect.addEventListener('change', () => {
            this.updateHierarchyInfo();
        });
        
        // Initialize hierarchy info
        this.updateHierarchyInfo();
    }

    /**
     * Update hierarchy information
     */
    updateHierarchyInfo() {
        const parentValue = this.parentSelect.value;
        const parentGroup = this.parentSelect.closest('.form-group');
        
        // Remove existing info
        const existingInfo = parentGroup.querySelector('.hierarchy-info');
        if (existingInfo) {
            existingInfo.remove();
        }
        
        if (parentValue) {
            const selectedOption = this.parentSelect.querySelector(`option[value="${parentValue}"]`);
            const parentName = selectedOption ? selectedOption.textContent : 'Seçilen Ana Menü';
            
            const info = document.createElement('div');
            info.className = 'hierarchy-info';
            info.innerHTML = `
                <i class="fas fa-level-down-alt"></i>
                Bu tuş "${parentName}" altında bir alt menü öğesi olacak
            `;
            
            parentGroup.appendChild(info);
        } else {
            const info = document.createElement('div');
            info.className = 'hierarchy-info';
            info.innerHTML = `
                <i class="fas fa-home"></i>
                Bu tuş ana menüde görünecek (üst seviye)
            `;
            
            parentGroup.appendChild(info);
        }
    }

    /**
     * Setup URL validation
     */
    setupUrlValidation() {
        this.urlInput.addEventListener('input', () => {
            this.validateUrl();
        });
        
        this.urlInput.addEventListener('blur', () => {
            this.validateUrl();
        });
    }

    /**
     * Validate URL field
     */
    validateUrl() {
        const urlValue = this.urlInput.value.trim();
        const urlGroup = this.urlInput.closest('.form-group');
        
        // Remove existing validation
        const existingValidation = urlGroup.querySelector('.url-validation');
        if (existingValidation) {
            existingValidation.remove();
        }
        
        if (urlValue) {
            let isValid = false;
            let message = '';
            
            // Check for internal URLs (starting with /)
            if (urlValue.startsWith('/')) {
                isValid = true;
                message = 'İç sayfa bağlantısı';
            } else {
                // Check for external URLs
                try {
                    new URL(urlValue);
                    isValid = true;
                    message = 'Geçerli dış bağlantı';
                } catch {
                    isValid = false;
                    message = 'Geçersiz URL formatı';
                }
            }
            
            const validation = document.createElement('div');
            validation.className = `url-validation ${isValid ? 'valid' : 'invalid'}`;
            validation.innerHTML = `
                <i class="fas fa-${isValid ? 'check' : 'times'}"></i>
                ${message}
            `;
            
            urlGroup.appendChild(validation);
            
            this.setFieldState(this.urlInput, isValid, isValid ? '' : 'Lütfen geçerli bir URL girin');
            return isValid;
        }
        
        return true; // URL is optional
    }

    /**
     * Setup form validation
     */
    setupFormValidation() {
        const requiredFields = this.form.querySelectorAll('[required]');
        
        requiredFields.forEach(field => {
            field.addEventListener('blur', () => {
                this.validateField(field);
            });
            
            field.addEventListener('input', () => {
                if (field.classList.contains('error')) {
                    this.validateField(field);
                }
            });
        });

        // Title validation
        this.titleInput.addEventListener('blur', () => {
            this.validateTitle();
        });

        // Sort order validation
        this.sortOrderInput.addEventListener('input', () => {
            this.validateSortOrder();
        });

        // Icon validation
        this.iconInput.addEventListener('blur', () => {
            this.validateIcon();
        });
    }

    /**
     * Setup toggle switches
     */
    setupToggleSwitches() {
        const toggles = document.querySelectorAll('.toggle-switch input[type="checkbox"]');
        
        toggles.forEach(toggle => {
            const updateToggleText = () => {
                const textElement = toggle.parentNode.querySelector('.toggle-text');
                if (textElement) {
                    textElement.textContent = toggle.checked ? 'Aktif' : 'Pasif';
                }
            };
            
            toggle.addEventListener('change', updateToggleText);
            updateToggleText(); // Initial state
        });
    }

    /**
     * Validate individual field
     */
    validateField(field) {
        const value = field.value.trim();
        let isValid = true;
        let errorMessage = '';

        // Required field validation
        if (field.hasAttribute('required') && !value) {
            isValid = false;
            errorMessage = 'Bu alan zorunludur.';
        }

        // Minimum length validation
        if (field.hasAttribute('minlength') && value.length > 0 && value.length < field.getAttribute('minlength')) {
            isValid = false;
            errorMessage = `En az ${field.getAttribute('minlength')} karakter olmalıdır.`;
        }

        // Maximum length validation
        if (field.hasAttribute('maxlength') && value.length > field.getAttribute('maxlength')) {
            isValid = false;
            errorMessage = `En fazla ${field.getAttribute('maxlength')} karakter olmalıdır.`;
        }

        this.setFieldState(field, isValid, errorMessage);
        return isValid;
    }

    /**
     * Validate title field
     */
    validateTitle() {
        const title = this.titleInput.value.trim();
        let isValid = true;
        let errorMessage = '';

        if (!title) {
            isValid = false;
            errorMessage = 'Başlık zorunludur.';
        } else if (title.length < 2) {
            isValid = false;
            errorMessage = 'Başlık en az 2 karakter olmalıdır.';
        } else if (title.length > 100) {
            isValid = false;
            errorMessage = 'Başlık en fazla 100 karakter olmalıdır.';
        }

        this.setFieldState(this.titleInput, isValid, errorMessage);
        return isValid;
    }

    /**
     * Validate sort order field
     */
    validateSortOrder() {
        const sortOrder = parseInt(this.sortOrderInput.value);
        let isValid = true;
        let errorMessage = '';

        if (isNaN(sortOrder)) {
            isValid = false;
            errorMessage = 'Sıra numarası sayı olmalıdır.';
        } else if (sortOrder < 1) {
            isValid = false;
            errorMessage = 'Sıra numarası 1 veya daha büyük olmalıdır.';
        } else if (sortOrder > 999) {
            isValid = false;
            errorMessage = 'Sıra numarası 999 veya daha küçük olmalıdır.';
        }

        this.setFieldState(this.sortOrderInput, isValid, errorMessage);
        return isValid;
    }

    /**
     * Validate icon field
     */
    validateIcon() {
        const icon = this.iconInput.value.trim();
        let isValid = true;
        let errorMessage = '';

        if (icon) {
            const iconRegex = /^(fas|far|fab|fal|fad)\s+fa-[\w-]+$/;
            if (!iconRegex.test(icon)) {
                isValid = false;
                errorMessage = 'Geçerli FontAwesome ikon formatı: "fas fa-home"';
            }
        }

        this.setFieldState(this.iconInput, isValid, errorMessage);
        return isValid;
    }

    /**
     * Set field validation state
     */
    setFieldState(field, isValid, errorMessage) {
        const fieldGroup = field.closest('.form-group');
        let errorElement = fieldGroup.querySelector('.field-error');

        // Remove existing error
        if (errorElement) {
            errorElement.remove();
        }

        // Update field classes
        field.classList.remove('error', 'success');
        field.classList.add(isValid ? 'success' : 'error');

        // Add error message
        if (!isValid && errorMessage) {
            errorElement = document.createElement('div');
            errorElement.className = 'field-error';
            errorElement.innerHTML = `<i class="fas fa-exclamation-circle"></i> ${errorMessage}`;
            fieldGroup.appendChild(errorElement);
        }
    }

    /**
     * Validate entire form
     */
    validateForm() {
        let isValid = true;
        
        // Validate required fields
        const requiredFields = this.form.querySelectorAll('[required]');
        requiredFields.forEach(field => {
            if (!this.validateField(field)) {
                isValid = false;
            }
        });

        // Validate title
        if (!this.validateTitle()) {
            isValid = false;
        }

        // Validate URL if provided
        if (this.urlInput.value.trim() && !this.validateUrl()) {
            isValid = false;
        }

        // Validate sort order
        if (!this.validateSortOrder()) {
            isValid = false;
        }

        // Validate icon if provided
        if (this.iconInput.value.trim() && !this.validateIcon()) {
            isValid = false;
        }

        if (!isValid) {
            this.showErrorModal('Lütfen tüm alanları doğru şekilde doldurun.');
            
            // Scroll to first error
            const firstError = this.form.querySelector('.error');
            if (firstError) {
                firstError.scrollIntoView({ behavior: 'smooth', block: 'center' });
                firstError.focus();
            }
        }

        return isValid;
    }

    /**
     * Submit form
     */
    async submitForm() {
        try {
            this.setLoadingState(true);

            const formData = new FormData(this.form);
            
            const response = await fetch('/backend/add-header-button', {
                method: 'POST',
                body: formData,
                headers: {
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const result = await response.json().catch(() => ({}));

            if (result.status === 201) {
                this.showSuccessModal();
            } else {
                this.showErrorModal(result.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
            }
        } catch (error) {
            console.error('Form submission error:', error);
            this.showErrorModal('Bağlantı hatası oluştu.');
        } finally {
            this.setLoadingState(false);
        }
    }

    /**
     * Set loading state for form submission
     */
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

    /**
     * Show success modal
     */
    showSuccessModal() {
        const modal = document.getElementById('successModal');
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
    }

    /**
     * Show error modal
     */
    showErrorModal(message) {
        const modal = document.getElementById('errorModal');
        const messageElement = document.getElementById('errorMessage');
        messageElement.textContent = message;
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
    }

    /**
     * Handle URL error parameters
     */
    handleUrlErrors() {
        const urlParams = new URLSearchParams(window.location.search);
        const error = urlParams.get('error');
        
        if (error) {
            const errorMessage = this.getErrorMessage(error);
            this.showErrorModal(errorMessage);
            
            // Clear URL parameters to prevent showing error on page refresh
            const newUrl = window.location.pathname;
            window.history.replaceState({}, document.title, newUrl);
            
            // Focus on relevant field if applicable
            this.focusOnErrorField(error);
        }
    }

    /**
     * Get user-friendly error message from error code
     */
    getErrorMessage(errorCode) {
        const errorMessages = {
            'title_required': 'Başlık alanı zorunludur.',
            'title_too_short': 'Başlık çok kısa. En az 2 karakter olmalıdır.',
            'title_too_long': 'Başlık çok uzun. En fazla 100 karakter olmalıdır.',
            'invalid_url': 'Geçersiz URL formatı. Lütfen geçerli bir URL girin.',
            'invalid_icon': 'Geçersiz ikon formatı. FontAwesome ikon sınıfı kullanın.',
            'invalid_sort_order': 'Geçersiz sıra numarası. 1-999 arası bir sayı girin.',
            'parent_not_found': 'Seçilen ana menü öğesi bulunamadı.',
            'internal_server_error': 'Sunucu hatası oluştu. Lütfen daha sonra tekrar deneyin.',
            'title_already_exists': 'Bu başlık zaten kullanılıyor. Farklı bir başlık seçin.',
            'max_menu_items_reached': 'Maksimum menü öğesi sayısına ulaşıldı.'
        };
        
        return errorMessages[errorCode] || 'Bilinmeyen bir hata oluştu. Lütfen tekrar deneyin.';
    }

    /**
     * Focus on relevant field based on error type
     */
    focusOnErrorField(errorCode) {
        let fieldToFocus = null;
        
        switch (errorCode) {
            case 'title_required':
            case 'title_too_short':
            case 'title_too_long':
            case 'title_already_exists':
                fieldToFocus = this.titleInput;
                break;
            case 'invalid_url':
                fieldToFocus = this.urlInput;
                break;
            case 'invalid_icon':
                fieldToFocus = this.iconInput;
                break;
            case 'invalid_sort_order':
                fieldToFocus = this.sortOrderInput;
                break;
            case 'parent_not_found':
                fieldToFocus = this.parentSelect;
                break;
        }
        
        if (fieldToFocus) {
            setTimeout(() => {
                // Add error class to highlight the field
                fieldToFocus.classList.add('error');
                
                // Focus and scroll to field
                fieldToFocus.focus();
                fieldToFocus.scrollIntoView({ behavior: 'smooth', block: 'center' });
                
                // Remove error class after a delay
                setTimeout(() => {
                    fieldToFocus.classList.remove('error');
                }, 3000);
            }, 500); // Delay to allow modal to show first
        }
    }

    /**
     * Reset form to initial state
     */
    resetForm() {
        this.form.reset();
        
        // Reset validation states
        const fields = this.form.querySelectorAll('.form-input, .form-textarea, .form-select');
        fields.forEach(field => {
            field.classList.remove('error', 'success');
        });
        
        // Remove error messages
        const errorElements = this.form.querySelectorAll('.field-error');
        errorElements.forEach(element => element.remove());
        
        // Remove validation indicators
        const validationElements = this.form.querySelectorAll('.url-validation, .hierarchy-info');
        validationElements.forEach(element => element.remove());
        
        // Reset icon preview
        this.updateIconPreview();
        
        // Reset toggle switches
        this.setupToggleSwitches();
        
        // Reset help texts
        this.updateTargetHelp();
        this.updateHierarchyInfo();
    }
}

/**
 * Modal management functions
 */
function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    modal.classList.remove('show');
    setTimeout(() => {
        modal.style.display = 'none';
        
        // If success modal, redirect to header buttons list
        if (modalId === 'successModal') {
            window.location.href = '/panel/header-tuslari';
        }
    }, 300);
}

/**
 * Utility functions
 */
function generateSlug(text) {
    return text
        .toLowerCase()
        .replace(/[çÇ]/g, 'c')
        .replace(/[ğĞ]/g, 'g')
        .replace(/[ıİ]/g, 'i')
        .replace(/[öÖ]/g, 'o')
        .replace(/[şŞ]/g, 's')
        .replace(/[üÜ]/g, 'u')
        .replace(/[^a-z0-9]/g, '-')
        .replace(/-+/g, '-')
        .replace(/^-|-$/g, '');
}

function validateFontAwesome(iconClass) {
    const iconRegex = /^(fas|far|fab|fal|fad)\s+fa-[\w-]+$/;
    return iconRegex.test(iconClass);
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
    new HeaderButtonFormHandler();
    
    // Close modals when clicking outside
    document.addEventListener('click', (e) => {
        if (e.target.classList.contains('modal')) {
            const modalId = e.target.id;
            closeModal(modalId);
        }
    });
    
    // Close modals with Escape key
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            const visibleModal = document.querySelector('.modal.show');
            if (visibleModal) {
                closeModal(visibleModal.id);
            }
        }
    });
});

// Export for potential external use
window.HeaderButtonFormHandler = HeaderButtonFormHandler;
