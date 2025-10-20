/**
 * Respond to Job Application Page JavaScript
 * Handles form interactions, validation, and AJAX submission
 */

class RespondToJobApplicationManager {
    constructor() {
        this.respondData = window.respondData || {};
        this.selectedApplication = null;
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupFormValidation();
        this.setupApplicationSelection();
    }

    /**
     * Setup all event listeners
     */
    setupEventListeners() {
        // Form submission
        const form = document.getElementById('respondToJobApplicationForm');
        if (form) {
            form.addEventListener('submit', (e) => {
                e.preventDefault();
                this.handleSubmit();
            });
        }

        // Application selection change
        const select = document.getElementById('job_application_select');
        if (select) {
            select.addEventListener('change', (e) => {
                this.handleApplicationSelection(e.target.value);
            });
        }

        // Real-time validation
        const inputs = document.querySelectorAll('input[required], textarea[required], select[required]');
        inputs.forEach(input => {
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
        const form = document.getElementById('respondToJobApplicationForm');
        if (!form) return;

        // Add validation attributes
        const requiredFields = form.querySelectorAll('[required]');
        requiredFields.forEach(field => {
            field.setAttribute('aria-required', 'true');
        });
    }

    /**
     * Setup application selection functionality
     */
    setupApplicationSelection() {
        const select = document.getElementById('job_application_select');
        if (!select) return;

        // Options are already populated by the template, no need to re-populate
        // Just check if there's a query parameter for auto-selection
        this.checkAutoSelect();
    }

    /**
     * Check for auto-selection from query parameters
     */
    checkAutoSelect() {
        const params = new URLSearchParams(window.location.search);
        const jaid = params.get('jaid');
        
        if (jaid) {
            const select = document.getElementById('job_application_select');
            if (select) {
                select.value = jaid;
                const evt = new Event('change');
                select.dispatchEvent(evt);
            }
        }
    }

    /**
     * Handle application selection
     */
    handleApplicationSelection(jaid) {
        const select = document.getElementById('job_application_select');
        const selectedOption = select.querySelector(`option[value="${jaid}"]`);
        
        if (!selectedOption || jaid === '') {
            this.hideApplicationInfo();
            return;
        }

        this.selectedApplication = {
            jaid: jaid,
            name: selectedOption.getAttribute('data-name'),
            position: selectedOption.getAttribute('data-position'),
            status: selectedOption.getAttribute('data-status')
        };

        this.showApplicationInfo();
    }

    /**
     * Show selected application information
     */
    showApplicationInfo() {
        const infoDiv = document.getElementById('selectedApplicationInfo');
        const nameSpan = document.getElementById('selectedName');
        const positionSpan = document.getElementById('selectedPosition');
        const statusSpan = document.getElementById('selectedStatus');

        if (!infoDiv || !this.selectedApplication) return;

        if (nameSpan) nameSpan.textContent = this.selectedApplication.name;
        if (positionSpan) positionSpan.textContent = this.selectedApplication.position;
        if (statusSpan) {
            const statusText = this.getStatusText(this.selectedApplication.status);
            statusSpan.textContent = statusText;
        }

        infoDiv.style.display = 'block';
    }

    /**
     * Hide application information
     */
    hideApplicationInfo() {
        const infoDiv = document.getElementById('selectedApplicationInfo');
        if (infoDiv) {
            infoDiv.style.display = 'none';
        }
        this.selectedApplication = null;
    }

    /**
     * Get status text in Turkish
     */
    getStatusText(status) {
        const statusMap = {
            'yeni': 'Yeni',
            'inceleniyor': 'İnceleniyor',
            'mulakat': 'Mülakat',
            'kabul': 'Kabul Edilmiş',
            'red': 'Reddedilmiş',
            'beklemede': 'Beklemede'
        };
        return statusMap[status] || status;
    }

    /**
     * Handle form submission
     */
    async handleSubmit() {
        const form = document.getElementById('respondToJobApplicationForm');
        const submitBtn = document.getElementById('submitResponseBtn');
        
        if (!form || !submitBtn) return;

        // Validate form
        if (!this.validateForm()) {
            return;
        }

        this.setButtonLoading(submitBtn, true);

        try {
            const formData = new FormData(form);
            const jaid = formData.get('jaid');
            
            if (!jaid) {
                this.showAlert('Lütfen bir iş başvurusu seçin.', 'error');
                this.setButtonLoading(submitBtn, false);
                return;
            }
            
            const data = {
                title: formData.get('title'),
                responder_name: formData.get('responder_name') || '',
                response_text: formData.get('response_text')
            };

            const response = await fetch(`/backend/job-application/${jaid}/respond`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                },
                body: JSON.stringify(data)
            });

            const result = await response.json();

            if (result.status === 201) {
                this.showAlert('Cevap başarıyla gönderildi.', 'success');
                this.resetForm();
                setTimeout(() => {
                    window.location.href = '/panel/is-basvurulari';
                }, 2000);
            } else {
                this.showAlert(result.message || 'Cevap gönderilirken hata oluştu.', 'error');
            }
        } catch (error) {
            console.error('Error submitting response:', error);
            this.showAlert('Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
        } finally {
            this.setButtonLoading(submitBtn, false);
        }
    }

    /**
     * Validate entire form
     */
    validateForm() {
        const form = document.getElementById('respondToJobApplicationForm');
        if (!form) return false;

        const requiredFields = form.querySelectorAll('[required]');
        let isValid = true;

        requiredFields.forEach(field => {
            if (!this.validateField(field)) {
                isValid = false;
            }
        });

        // Check if application is selected
        if (!this.selectedApplication) {
            const select = document.getElementById('job_application_select');
            if (select) {
                this.showFieldError(select, 'Lütfen bir iş başvurusu seçin.');
                isValid = false;
            }
        }

        return isValid;
    }

    /**
     * Validate individual field
     */
    validateField(field) {
        if (!field) return false;

        const value = field.value.trim();
        const isRequired = field.hasAttribute('required');

        if (isRequired && value === '') {
            this.showFieldError(field, 'Bu alan zorunludur.');
            return false;
        }

        if (value !== '') {
            this.clearFieldError(field);
        }

        return true;
    }

    /**
     * Show field error
     */
    showFieldError(field, message) {
        this.clearFieldError(field);
        
        field.classList.add('error');
        
        const errorDiv = document.createElement('div');
        errorDiv.className = 'field-error';
        errorDiv.textContent = message;
        errorDiv.style.color = '#dc2626';
        errorDiv.style.fontSize = '12px';
        errorDiv.style.marginTop = '4px';
        
        field.parentNode.appendChild(errorDiv);
    }

    /**
     * Clear field error
     */
    clearFieldError(field) {
        field.classList.remove('error');
        
        const errorDiv = field.parentNode.querySelector('.field-error');
        if (errorDiv) {
            errorDiv.remove();
        }
    }

    /**
     * Reset form
     */
    resetForm() {
        const form = document.getElementById('respondToJobApplicationForm');
        if (form) {
            form.reset();
        }
        
        this.hideApplicationInfo();
        
        // Clear all field errors
        const errorDivs = document.querySelectorAll('.field-error');
        errorDivs.forEach(div => div.remove());
        
        const errorFields = document.querySelectorAll('.error');
        errorFields.forEach(field => field.classList.remove('error'));
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
    if (window.respondToJobApplicationManager) {
        window.respondToJobApplicationManager.closeAlert(alertId);
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.respondToJobApplicationManager = new RespondToJobApplicationManager();
});

