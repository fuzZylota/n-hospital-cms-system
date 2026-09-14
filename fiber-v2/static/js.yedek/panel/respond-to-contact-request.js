/**
 * Respond to Contact Request Page JavaScript
 * Handles form validation, contact request selection, and AJAX submission
 */

class RespondToContactRequestManager {
    constructor() {
        this.respondData = window.respondData || {};
        this.form = null;
        this.submitBtn = null;
        
        this.init();
    }

    init() {
        this.setupElements();
        this.setupEventListeners();
        this.setupFormValidation();
    }

    /**
     * Setup DOM elements
     */
    setupElements() {
        this.form = document.getElementById('respondToContactRequestForm');
        this.submitBtn = document.getElementById('submitResponseBtn');
        this.contactRequestSelect = document.getElementById('contact_request_select');
        this.selectedRequestInfo = document.getElementById('selectedRequestInfo');
        this.selectedSubject = document.getElementById('selectedSubject');
        this.selectedStatus = document.getElementById('selectedStatus');
    }

    /**
     * Setup all event listeners
     */
    setupEventListeners() {
        // Contact request selection change
        if (this.contactRequestSelect) {
            this.contactRequestSelect.addEventListener('change', (e) => {
                this.handleContactRequestSelection(e);
            });
        }

        // Form submission
        if (this.form) {
            this.form.addEventListener('submit', (e) => {
                e.preventDefault();
                this.handleFormSubmission();
            });
        }

        // Real-time validation
        const inputs = this.form?.querySelectorAll('input, select, textarea');
        inputs?.forEach(input => {
            input.addEventListener('blur', () => {
                this.validateField(input);
            });
            
            input.addEventListener('input', () => {
                this.clearFieldError(input);
            });
        });
    }

    /**
     * Setup form validation
     */
    setupFormValidation() {
        // Add required field indicators
        const requiredFields = this.form?.querySelectorAll('[required]');
        requiredFields?.forEach(field => {
            const label = this.form?.querySelector(`label[for="${field.id}"]`);
            if (label && !label.classList.contains('required')) {
                label.classList.add('required');
            }
        });
    }

    /**
     * Handle contact request selection change
     */
    handleContactRequestSelection(event) {
        const selectedOption = event.target.options[event.target.selectedIndex];
        
        if (selectedOption.value) {
            const subject = selectedOption.dataset.subject;
            const isReplied = selectedOption.dataset.isReplied === 'true';
            
            // Update selected request info
            this.selectedSubject.textContent = subject;
            this.selectedStatus.textContent = isReplied ? 'Cevaplanmış' : 'Cevaplanmamış';
            this.selectedStatus.style.color = isReplied ? 'var(--warning-color)' : 'var(--success-color)';
            
            // Show selected request info
            this.selectedRequestInfo.style.display = 'block';
            
            // Clear any previous errors
            this.clearFieldError(this.contactRequestSelect);
        } else {
            // Hide selected request info
            this.selectedRequestInfo.style.display = 'none';
        }
    }

    /**
     * Handle form submission
     */
    async handleFormSubmission() {
        if (!this.validateForm()) {
            return;
        }

        this.setButtonLoading(this.submitBtn, true);

        try {
            const formData = new FormData(this.form);
            const data = {
                crid: formData.get('crid'),
                title: formData.get('title'),
                responder_name: formData.get('responder_name'),
                response_text: formData.get('response_text')
            };

            const response = await fetch('/backend/contact-request/' + data.crid + '/respond', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                },
                body: JSON.stringify(data)
            });

            const responseData = await response.json().catch(() => ({}));

            if (responseData.status === 201) {
                this.showAlert(responseData.message || 'Cevap başarıyla gönderildi.', 'success');
                // Reset form after successful submission
                setTimeout(() => {
                    this.form.reset();
                    this.selectedRequestInfo.style.display = 'none';
                }, 1500);
            } else {
                this.showAlert(responseData.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
            }
        } catch (error) {
            console.error('Submit error:', error);
            this.showAlert('Bağlantı hatası oluştu.', 'error');
        } finally {
            this.setButtonLoading(this.submitBtn, false);
        }
    }

    /**
     * Validate entire form
     */
    validateForm() {
        let isValid = true;
        const requiredFields = this.form?.querySelectorAll('[required]');
        
        requiredFields?.forEach(field => {
            if (!this.validateField(field)) {
                isValid = false;
            }
        });

        return isValid;
    }

    /**
     * Validate individual field
     */
    validateField(field) {
        const value = field.value.trim();
        const isRequired = field.hasAttribute('required');
        const fieldGroup = field.closest('.form-group');
        
        // Clear previous errors
        this.clearFieldError(field);
        
        // Check if required field is empty
        if (isRequired && !value) {
            this.setFieldError(field, 'Bu alan zorunludur.');
            return false;
        }
        
        // Additional validations
        if (field.type === 'email' && value && !this.isValidEmail(value)) {
            this.setFieldError(field, 'Geçerli bir e-posta adresi girin.');
            return false;
        }
        
        if (field.id === 'response_text' && value && value.length < 10) {
            this.setFieldError(field, 'Cevap metni en az 10 karakter olmalıdır.');
            return false;
        }
        
        return true;
    }

    /**
     * Set field error state
     */
    setFieldError(field, message) {
        const fieldGroup = field.closest('.form-group');
        if (!fieldGroup) return;
        
        fieldGroup.classList.add('error');
        
        // Remove existing error message
        const existingError = fieldGroup.querySelector('.field-error');
        if (existingError) {
            existingError.remove();
        }
        
        // Add new error message
        const errorDiv = document.createElement('div');
        errorDiv.className = 'field-error';
        errorDiv.style.color = 'var(--error-color)';
        errorDiv.style.fontSize = '12px';
        errorDiv.style.marginTop = '4px';
        errorDiv.textContent = message;
        
        fieldGroup.appendChild(errorDiv);
    }

    /**
     * Clear field error state
     */
    clearFieldError(field) {
        const fieldGroup = field.closest('.form-group');
        if (!fieldGroup) return;
        
        fieldGroup.classList.remove('error');
        
        const existingError = fieldGroup.querySelector('.field-error');
        if (existingError) {
            existingError.remove();
        }
    }

    /**
     * Validate email format
     */
    isValidEmail(email) {
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        return emailRegex.test(email);
    }

    /**
     * Set button loading state
     */
    setButtonLoading(button, loading) {
        if (!button) return;
        
        const textSpan = button.querySelector('.btn-text');
        const loader = button.querySelector('.btn-loader');
        
        if (loading) {
            button.disabled = true;
            if (textSpan) textSpan.style.display = 'none';
            if (loader) loader.style.display = 'inline-flex';
        } else {
            button.disabled = false;
            if (textSpan) textSpan.style.display = 'inline';
            if (loader) loader.style.display = 'none';
        }
    }

    /**
     * Show alert message
     */
    showAlert(message, type = 'success') {
        const alertId = type === 'success' ? 'successMessage' : 'errorMessage';
        const alert = document.getElementById(alertId);
        
        if (!alert) return;
        
        const textSpan = alert.querySelector('.alert-text');
        if (textSpan) {
            textSpan.textContent = message;
        }
        
        alert.style.display = 'flex';
        
        // Auto hide after 5 seconds
        setTimeout(() => {
            this.closeAlert(alertId);
        }, 5000);
    }

    /**
     * Close alert message
     */
    closeAlert(alertId) {
        const alert = document.getElementById(alertId);
        if (alert) {
            alert.style.display = 'none';
        }
    }
}

// Global functions for HTML onclick attributes
function closeAlert(alertId) {
    if (window.respondToContactRequestManager) {
        window.respondToContactRequestManager.closeAlert(alertId);
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.respondToContactRequestManager = new RespondToContactRequestManager();
});

