/**
 * Header Tusu Duzenle (Header Button Edit) Page JavaScript
 * Handles header button information form submission and validation
 */

class HeaderButtonEditHandler {
    constructor() {
        this.hbid = window.pageData.hbid;
        this.headerButtonForm = document.getElementById('headerButtonForm');
        this.headerButtonSubmitBtn = document.getElementById('headerButtonSubmitBtn');
        
        this.init();
    }

    init() {
        this.setupHeaderButtonForm();
        this.setupFormValidation();
        this.setupToggleSwitches();
        this.setupTargetHandling();
        this.setupIconPreview();
    }

    /**
     * Setup header button form (AJAX JSON submission)
     */
    setupHeaderButtonForm() {
        this.headerButtonForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            if (!this.validateHeaderButtonForm()) {
                return;
            }

            this.setHeaderButtonLoadingState(true);

            try {
                // Collect all form data
                const data = {};
                const allInputs = this.headerButtonForm.querySelectorAll('input, textarea, select');

                allInputs.forEach(input => {
                    if (!input.name) return;
                    if (input.type === 'checkbox') {
                        data[input.name] = input.checked;
                        return;
                    }

                    let value = input.value;
                    if (input.type === 'hidden') {
                        if (value === 'true') value = true;
                        else if (value === 'false') value = false;
                    }
                    data[input.name] = value;
                });

                // Normalize numeric fields
                if (data.sort_order !== undefined) {
                    data.sort_order = parseInt(data.sort_order, 10);
                }
                if (data.old_sort_order !== undefined) {
                    data.old_sort_order = parseInt(data.old_sort_order, 10);
                }
                if (data.parent_id !== undefined) {
                    data.parent_id = (data.parent_id === '' ? null : parseInt(data.parent_id, 10));
                }
                if (data.old_parent_id !== undefined) {
                    data.old_parent_id = (data.old_parent_id === '' ? null : parseInt(data.old_parent_id, 10));
                }

                const response = await fetch(`/backend/header-button/${this.hbid}/edit`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-Requested-With': 'XMLHttpRequest'
                    },
                    body: JSON.stringify(data)
                });

                const result = await response.json().catch(() => ({}));

                if (result.status === 201) {
                    this.showSuccessModal('Header button bilgileri başarıyla güncellendi.');
                    // Optionally redirect or update UI
                    setTimeout(() => {
                        window.location.href = `/panel/header-tuslari/${this.hbid}`;
                    }, 2000);
                } else {
                    this.showErrorModal(result.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
                }

            } catch (error) {
                console.error('Header button form submission error:', error);
                this.showErrorModal('Bağlantı hatası oluştu.');
            } finally {
                this.setHeaderButtonLoadingState(false);
            }
        });
    }

    /**
     * Setup form validation
     */
    setupFormValidation() {
        // Required fields validation
        const requiredFields = this.headerButtonForm.querySelectorAll('[required]');
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

        // URL validation
        const urlField = this.headerButtonForm.querySelector('input[name="url"]');
        if (urlField) {
            urlField.addEventListener('blur', () => {
                this.validateUrl(urlField);
            });
            
            urlField.addEventListener('input', () => {
                if (urlField.classList.contains('error')) {
                    this.validateUrl(urlField);
                }
            });
        }

        // Title validation
        const titleField = this.headerButtonForm.querySelector('input[name="title"]');
        if (titleField) {
            titleField.addEventListener('input', () => {
                this.validateTitle(titleField);
            });
            
            titleField.addEventListener('blur', () => {
                this.validateTitle(titleField);
            });
        }

        // Icon validation
        const iconField = this.headerButtonForm.querySelector('input[name="icon"]');
        if (iconField) {
            iconField.addEventListener('input', () => {
                this.validateIcon(iconField);
            });
            
            iconField.addEventListener('blur', () => {
                this.validateIcon(iconField);
            });
        }

        // Sort order validation
        const sortOrderField = this.headerButtonForm.querySelector('input[name="sort_order"]');
        if (sortOrderField) {
            sortOrderField.addEventListener('input', () => {
                this.validateSortOrder(sortOrderField);
            });
            
            sortOrderField.addEventListener('blur', () => {
                this.validateSortOrder(sortOrderField);
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
     * Setup target handling and information
     */
    setupTargetHandling() {
        const targetField = this.headerButtonForm.querySelector('#target');
        if (targetField) {
            targetField.addEventListener('change', () => {
                this.updateTargetInfo(targetField.value);
                this.validateField(targetField);
            });
            
            // Initialize target info
            this.updateTargetInfo(targetField.value);
        }
    }

    /**
     * Update target information
     */
    updateTargetInfo(target) {
        const targetDescriptions = {
            '_self': 'Link aynı sekmede açılır',
            '_blank': 'Link yeni sekmede açılır'
        };
        
        // Update help text
        const helpText = this.headerButtonForm.querySelector('#target').parentNode.querySelector('.form-help');
        if (helpText && targetDescriptions[target]) {
            helpText.textContent = targetDescriptions[target];
        } else if (helpText) {
            helpText.textContent = 'Linkin nasıl açılacağını belirler';
        }
    }

    /**
     * Setup icon preview
     */
    setupIconPreview() {
        const iconField = this.headerButtonForm.querySelector('input[name="icon"]');
        if (iconField) {
            iconField.addEventListener('input', () => {
                this.updateIconPreview(iconField.value);
            });
            
            // Initial preview
            this.updateIconPreview(iconField.value);
        }
    }

    /**
     * Update icon preview
     */
    updateIconPreview(iconName) {
        const iconPreview = this.headerButtonForm.querySelector('.icon-preview');
        if (iconPreview) {
            if (iconName && iconName.trim()) {
                iconPreview.innerHTML = `<i class="fas fa-${iconName.trim()}"></i> Önizleme: ${iconName.trim()}`;
                iconPreview.style.display = 'block';
            } else {
                iconPreview.style.display = 'none';
            }
        }
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
    validateTitle(field) {
        const value = field.value.trim();
        let isValid = true;
        let errorMessage = '';

        // Required validation
        if (!value) {
            isValid = false;
            errorMessage = 'Başlık zorunludur.';
        } else if (value.length < 2) {
            isValid = false;
            errorMessage = 'En az 2 karakter olmalıdır.';
        } else if (value.length > 100) {
            isValid = false;
            errorMessage = 'En fazla 100 karakter olmalıdır.';
        }

        this.setFieldState(field, isValid, errorMessage);
        return isValid;
    }

    /**
     * Validate URL field
     */
    validateUrl(field) {
        const url = field.value.trim();
        let isValid = true;
        let errorMessage = '';
        
        if (url && url.length > 0) {
            if (url.length > 255) {
                isValid = false;
                errorMessage = 'URL çok uzun.';
            } else {
                // Basic URL validation
                // Accept either:
                // - absolute URLs starting with http:// or https://
                // - relative paths starting with / followed by allowed URL path chars (no spaces, exclude '!')
                const urlRegex = /^(https?:\/\/[^\s]+|\/[A-Za-z0-9._~:\/?#\[\]@$&'()*+,;=\-]*)$/;
                if (!urlRegex.test(url)) {
                    isValid = false;
                    errorMessage = 'Geçerli bir URL girin (http:// veya https:// ile başlamalı).';
                }
            }
        }
        
        this.setFieldState(field, isValid, errorMessage);
        return isValid;
    }

    /**
     * Validate icon field
     */
    validateIcon(field) {
        const icon = field.value.trim();
        let isValid = true;
        let errorMessage = '';
        
        if (icon && icon.length > 0) {
            if (icon.length > 50) {
                isValid = false;
                errorMessage = 'İkon adı çok uzun.';
            } else if (!/^[a-zA-Z0-9\-_]+$/.test(icon)) {
                isValid = false;
                errorMessage = 'İkon adı sadece harf, rakam, tire ve alt çizgi içerebilir.';
            }
        }
        
        this.setFieldState(field, isValid, errorMessage);
        return isValid;
    }

    /**
     * Validate sort order field
     */
    validateSortOrder(field) {
        const sortOrder = parseInt(field.value);
        let isValid = true;
        let errorMessage = '';
        
        if (!sortOrder || sortOrder < 1) {
            isValid = false;
            errorMessage = 'Sıra numarası 1\'den büyük olmalıdır.';
        } else if (sortOrder > 999) {
            isValid = false;
            errorMessage = 'Sıra numarası çok büyük.';
        }
        
        this.setFieldState(field, isValid, errorMessage);
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
     * Validate entire header button form
     */
    validateHeaderButtonForm() {
        let isValid = true;
        
        // Validate required fields
        const requiredFields = this.headerButtonForm.querySelectorAll('[required]');
        requiredFields.forEach(field => {
            if (!this.validateField(field)) {
                isValid = false;
            }
        });

        // Validate title
        const titleField = this.headerButtonForm.querySelector('input[name="title"]');
        if (titleField && !this.validateTitle(titleField)) {
            isValid = false;
        }

        // Validate URL
        const urlField = this.headerButtonForm.querySelector('input[type="url"]');
        if (urlField && !this.validateUrl(urlField)) {
            isValid = false;
        }

        // Validate icon
        const iconField = this.headerButtonForm.querySelector('input[name="icon"]');
        if (iconField && !this.validateIcon(iconField)) {
            isValid = false;
        }

        // Validate sort order
        const sortOrderField = this.headerButtonForm.querySelector('input[name="sort_order"]');
        if (sortOrderField && !this.validateSortOrder(sortOrderField)) {
            isValid = false;
        }

        if (!isValid) {
            this.showErrorModal('Lütfen tüm alanları doğru şekilde doldurun.');
            
            // Scroll to first error
            const firstError = this.headerButtonForm.querySelector('.error');
            if (firstError) {
                firstError.scrollIntoView({ behavior: 'smooth', block: 'center' });
                firstError.focus();
            }
        }

        return isValid;
    }

    /**
     * Set loading state for header button form
     */
    setHeaderButtonLoadingState(loading) {
        if (loading) {
            this.headerButtonSubmitBtn.disabled = true;
            this.headerButtonSubmitBtn.querySelector('.btn-text').style.opacity = '0';
            this.headerButtonSubmitBtn.querySelector('.btn-loader').style.display = 'block';
            this.headerButtonForm.classList.add('form-loading');
        } else {
            this.headerButtonSubmitBtn.disabled = false;
            this.headerButtonSubmitBtn.querySelector('.btn-text').style.opacity = '1';
            this.headerButtonSubmitBtn.querySelector('.btn-loader').style.display = 'none';
            this.headerButtonForm.classList.remove('form-loading');
        }
    }

    /**
     * Show success modal
     */
    showSuccessModal(message) {
        const modal = document.getElementById('successModal');
        const messageElement = document.getElementById('successMessage');
        messageElement.textContent = message;
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
     * Close modal
     */
    closeModal(modalId) {
        const modal = document.getElementById(modalId);
        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
    }

    /**
     * Reset form to initial state
     */
    resetForm() {
        this.headerButtonForm.reset();
        
        // Reset validation states
        const fields = this.headerButtonForm.querySelectorAll('.form-input, .form-textarea, .form-select');
        fields.forEach(field => {
            field.classList.remove('error', 'success');
        });
        
        // Remove error messages
        const errorElements = this.headerButtonForm.querySelectorAll('.field-error');
        errorElements.forEach(element => element.remove());
        
        // Reset toggle switches
        this.setupToggleSwitches();
        
        // Reset target info
        const targetField = this.headerButtonForm.querySelector('#target');
        if (targetField) {
            this.updateTargetInfo(targetField.value);
        }
        
        // Reset icon preview
        const iconField = this.headerButtonForm.querySelector('input[name="icon"]');
        if (iconField) {
            this.updateIconPreview(iconField.value);
        }
    }

    /**
     * Get form data as object
     */
    getFormData() {
        const data = {};
        const inputs = this.headerButtonForm.querySelectorAll('input, textarea, select');
        
        inputs.forEach(input => {
            if (input.type === 'checkbox') {
                data[input.name] = input.checked;
            } else if (input.type !== 'submit' && input.type !== 'button') {
                data[input.name] = input.value;
            }
        });
        
        return data;
    }

    /**
     * Set form data from object
     */
    setFormData(data) {
        Object.keys(data).forEach(key => {
            const input = this.headerButtonForm.querySelector(`[name="${key}"]`);
            if (input) {
                if (input.type === 'checkbox') {
                    input.checked = Boolean(data[key]);
                } else {
                    input.value = data[key] || '';
                }
            }
        });
        
        // Update toggle switches
        this.setupToggleSwitches();
        
        // Update target info
        const targetField = this.headerButtonForm.querySelector('#target');
        if (targetField) {
            this.updateTargetInfo(targetField.value);
        }
        
        // Update icon preview
        const iconField = this.headerButtonForm.querySelector('input[name="icon"]');
        if (iconField) {
            this.updateIconPreview(iconField.value);
        }
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
    }, 300);
}

/**
 * Utility functions
 */
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

function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString('tr-TR', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
}

function validateUrl(url) {
    // Basic URL validation
    const urlRegex = /^https?:\/\/.+/;
    return urlRegex.test(url);
}

function validateIconName(iconName) {
    // FontAwesome icon name validation
    const iconRegex = /^[a-zA-Z0-9\-_]+$/;
    return iconRegex.test(iconName);
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    const headerButtonEditHandler = new HeaderButtonEditHandler();
    window.headerButtonEditHandler = headerButtonEditHandler;
    
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

    // Add keyboard shortcuts
    document.addEventListener('keydown', (e) => {
        // Ctrl/Cmd + S for save
        if ((e.ctrlKey || e.metaKey) && e.key === 's') {
            e.preventDefault();
            const submitBtn = document.getElementById('headerButtonSubmitBtn');
            if (submitBtn && !submitBtn.disabled) {
                submitBtn.click();
            }
        }
        
        // Ctrl/Cmd + R for reset
        if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
            e.preventDefault();
            if (confirm('Formu sıfırlamak istediğinizden emin misiniz?')) {
                headerButtonEditHandler.resetForm();
            }
        }
    });
});

// Export for potential external use
window.HeaderButtonEditHandler = HeaderButtonEditHandler;
