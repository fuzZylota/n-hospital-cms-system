/**
 * Custom Content Ekle (Custom Content Add) Page JavaScript
 * Handles form submission, CodeMirror integration, and validation
 */

class CustomContentFormHandler {
    constructor() {
        this.form = document.getElementById('customContentForm');
        this.submitBtn = document.getElementById('submitBtn');
        this.codeEditors = {};
        this.init();
    }

    init() {
        this.setupFormSubmission();
        this.setupCodeEditors();
        this.setupFormValidation();
        this.setupToggleSwitches();
        this.setupNameToUrlSync();
    }

    /**
     * Setup form submission (non-AJAX). Validate and allow native submit.
     */
    setupFormSubmission() {
        this.form.addEventListener('submit', (e) => {
            // Validate before submit; block submit if invalid
            if (!this.validateForm()) {
                e.preventDefault();
                return;
            }

            // Show loading state
            this.setLoadingState(true);

            // Allow natural form submission to the action URL
        });
    }

    /**
     * Setup CodeMirror editors for HTML, CSS, and JavaScript
     */
    setupCodeEditors() {
        // HTML Editor
        const htmlTextarea = document.getElementById('content_html');
        if (htmlTextarea) {
            this.codeEditors.html = CodeMirror.fromTextArea(htmlTextarea, {
                mode: 'htmlmixed',
                theme: 'monokai',
                lineNumbers: true,
                autoCloseBrackets: true,
                matchBrackets: true,
                indentUnit: 2,
                tabSize: 2,
                lineWrapping: true,
                extraKeys: {
                    "Ctrl-Space": "autocomplete"
                }
            });
        }

        // CSS Editor
        const cssTextarea = document.getElementById('content_css');
        if (cssTextarea) {
            this.codeEditors.css = CodeMirror.fromTextArea(cssTextarea, {
                mode: 'css',
                theme: 'monokai',
                lineNumbers: true,
                autoCloseBrackets: true,
                matchBrackets: true,
                indentUnit: 2,
                tabSize: 2,
                lineWrapping: true,
                extraKeys: {
                    "Ctrl-Space": "autocomplete"
                }
            });
        }

        // JavaScript Editor
        const jsTextarea = document.getElementById('content_javascript');
        if (jsTextarea) {
            this.codeEditors.javascript = CodeMirror.fromTextArea(jsTextarea, {
                mode: 'javascript',
                theme: 'monokai',
                lineNumbers: true,
                autoCloseBrackets: true,
                matchBrackets: true,
                indentUnit: 2,
                tabSize: 2,
                lineWrapping: true,
                extraKeys: {
                    "Ctrl-Space": "autocomplete"
                }
            });
        }

        // Sync CodeMirror content with form data
        Object.keys(this.codeEditors).forEach(key => {
            this.codeEditors[key].on('change', () => {
                this.codeEditors[key].save();
            });
        });
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

        // URL validation
        const urlFields = this.form.querySelectorAll('input[type="url"]');
        urlFields.forEach(field => {
            field.addEventListener('blur', () => {
                this.validateUrl(field);
            });
        });

        // Number validation
        const numberFields = this.form.querySelectorAll('input[type="number"]');
        numberFields.forEach(field => {
            field.addEventListener('blur', () => {
                this.validateNumber(field);
            });
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
     * Setup name to URL name synchronization
     */
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

            urlField.addEventListener('input', () => {
                if (urlField.value) {
                    urlField.dataset.autoGenerated = 'false';
                }
            });
        }
    }

    /**
     * Generate URL-friendly name from content name
     */
    generateUrlName(name) {
        return name
            .toLowerCase()
            .replace(/ç/g, 'c')
            .replace(/ğ/g, 'g')
            .replace(/ı/g, 'i')
            .replace(/ö/g, 'o')
            .replace(/ş/g, 's')
            .replace(/ü/g, 'u')
            .replace(/[^a-z0-9]+/g, '-')
            .replace(/^-+|-+$/g, '');
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
        if (field.hasAttribute('minlength') && value.length < field.getAttribute('minlength')) {
            isValid = false;
            errorMessage = `En az ${field.getAttribute('minlength')} karakter olmalıdır.`;
        }

        this.setFieldState(field, isValid, errorMessage);
        return isValid;
    }

    /**
     * Validate URL field
     */
    validateUrl(field) {
        const url = field.value.trim();
        if (!url) return true; // Not required
        
        const domainLikeRegex = /^(?:[a-zA-Z]+:\/\/)?[^\s.]+(?:\.[^\s.]+)+(?:[\/#?].*)?$/;
        const isValid = domainLikeRegex.test(url);
        this.setFieldState(field, isValid, isValid ? '' : "Geçerli bir URL girin (örn: ornek.com)");
        return isValid;
    }

    /**
     * Validate number field
     */
    validateNumber(field) {
        const value = field.value.trim();
        if (!value) return true; // Not required
        
        const num = parseFloat(value);
        const isValid = !isNaN(num) && num >= 0;
        
        this.setFieldState(field, isValid, isValid ? '' : 'Geçerli bir sayı girin.');
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
            errorElement.innerHTML = `<i class="icon-alert-circle"></i> ${errorMessage}`;
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

        // Validate URL fields
        const urlFields = this.form.querySelectorAll('input[type="url"]');
        urlFields.forEach(field => {
            if (!this.validateUrl(field)) {
                isValid = false;
            }
        });

        // Validate number fields
        const numberFields = this.form.querySelectorAll('input[type="number"]');
        numberFields.forEach(field => {
            if (!this.validateNumber(field)) {
                isValid = false;
            }
        });

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
     * Reset form to initial state
     */
    resetForm() {
        this.form.reset();
        
        // Reset CodeMirror editors
        Object.keys(this.codeEditors).forEach(key => {
            this.codeEditors[key].setValue('');
        });
        
        // Reset validation states
        const fields = this.form.querySelectorAll('.form-input, .form-textarea, .form-select');
        fields.forEach(field => {
            field.classList.remove('error', 'success');
        });
        
        // Remove error messages
        const errorElements = this.form.querySelectorAll('.field-error');
        errorElements.forEach(element => element.remove());
        
        // Reset toggle switches
        this.setupToggleSwitches();
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
        
        // If success modal, redirect to custom contents list
        if (modalId === 'successModal') {
            window.location.href = '/panel/ozel-icerikler';
        }
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

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new CustomContentFormHandler();
    
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
window.CustomContentFormHandler = CustomContentFormHandler;

