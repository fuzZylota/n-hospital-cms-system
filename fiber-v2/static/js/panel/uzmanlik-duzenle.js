/**
 * Uzmanlik Duzenle (Expertise Edit) Page JavaScript
 * Handles general settings form for expertise areas
 */

class UzmanlikEditHandler {
    constructor() {
        this.uzid = window.pageData.uzid;
        this.generalForm = document.getElementById('generalForm');
        this.generalSubmitBtn = document.getElementById('generalSubmitBtn');
        this.originalValues = {};
        
        this.init();
    }

    init() {
        this.initializeOriginalValues();
        this.setupGeneralForm();
        this.setupFormValidation();
        this.setupToggleSwitches();
        this.setupUrlGeneration();
    }

    /**
     * Store original values for change detection
     */
    initializeOriginalValues() {
        const nameInput = document.getElementById('name');
        const urlNameInput = document.getElementById('url_name');
        const descriptionInput = document.getElementById('description');
        const iconInput = document.getElementById('icon');
        const sortOrderInput = document.getElementById('sort_order');
        const isActiveInput = document.getElementById('is_active');
        
        if (nameInput) {
            this.originalValues.name = nameInput.value;
        }
        if (urlNameInput) {
            this.originalValues.urlName = urlNameInput.value;
        }
        if (descriptionInput) {
            this.originalValues.description = descriptionInput.value;
        }
        if (iconInput) {
            this.originalValues.icon = iconInput.value;
        }
        if (sortOrderInput) {
            this.originalValues.sortOrder = sortOrderInput.value;
        }
        if (isActiveInput) {
            this.originalValues.isActive = isActiveInput.checked;
        }
    }

    /**
     * Setup general settings form (AJAX JSON submission)
     */
    setupGeneralForm() {
        this.generalForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            if (!this.validateGeneralForm()) {
                return;
            }

            this.setGeneralLoadingState(true);

            try {
                // Collect all form data
                const data = {};
                const allInputs = this.generalForm.querySelectorAll('input, textarea, select');
                
                allInputs.forEach(input => {
                    if (input.type === 'checkbox') {
                        data[input.name] = input.checked;
                    } else if (input.type === 'number') {
                        data[input.name] = parseInt(input.value) || 0;
                    } else if (input.type === 'hidden') {
                        if (input.value === 'true') {
                            data[input.name] = true;
                        } else if (input.value === 'false') {
                            data[input.name] = false;
                        } else if (input.name.includes('old_') && document.querySelector(`[name="${input.name.replace('old_', '')}"]`)?.type === 'number') {
                            data[input.name] = parseInt(input.value) || 0;
                        } else {
                            data[input.name] = input.value;
                        }
                    } else {
                        data[input.name] = input.value;
                    }
                });

                const response = await fetch(`/backend/expertise/${this.uzid}/edit`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(data)
                });

                const result = await response.json();

                if (result.status === 201) {
                    this.showSuccessModal('Uzmanlık alanı bilgileri başarıyla güncellendi.');
                } else {
                    throw new Error(result.message || 'Güncelleme işlemi başarısız oldu.');
                }

            } catch (error) {
                console.error('General form submission error:', error);
                this.showErrorModal(error.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
            } finally {
                this.setGeneralLoadingState(false);
            }
        });
    }

    /**
     * Setup form validation
     */
    setupFormValidation() {
        // General form validation
        const requiredFields = this.generalForm.querySelectorAll('[required]');
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

        // Number validation for sort_order
        const sortOrderField = document.getElementById('sort_order');
        if (sortOrderField) {
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
     * Setup URL generation from name
     */
    setupUrlGeneration() {
        const nameInput = document.getElementById('name');
        const urlInput = document.getElementById('url_name');
        
        if (nameInput && urlInput) {
            nameInput.addEventListener('input', () => {
                if (!urlInput.value || urlInput.dataset.autoGenerated === 'true') {
                    const urlFriendly = this.generateUrlFriendlyString(nameInput.value);
                    urlInput.value = urlFriendly;
                    urlInput.dataset.autoGenerated = 'true';
                }
            });
            
            urlInput.addEventListener('input', () => {
                // Mark as manually edited if user types
                if (urlInput.value !== this.generateUrlFriendlyString(nameInput.value)) {
                    urlInput.dataset.autoGenerated = 'false';
                }
            });
        }
    }

    /**
     * Generate URL-friendly string from Turkish text
     */
    generateUrlFriendlyString(text) {
        const turkishMap = {
            'ç': 'c', 'Ç': 'C',
            'ğ': 'g', 'Ğ': 'G',
            'ı': 'i', 'I': 'I',
            'İ': 'I', 'i': 'i',
            'ö': 'o', 'Ö': 'O',
            'ş': 's', 'Ş': 'S',
            'ü': 'u', 'Ü': 'U'
        };
        
        return text
            .split('')
            .map(char => turkishMap[char] || char)
            .join('')
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, '-')
            .replace(/^-+|-+$/g, '')
            .substring(0, 50);
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
     * Validate sort order field
     */
    validateSortOrder(field) {
        const value = parseInt(field.value);
        if (isNaN(value)) return true; // Empty is ok
        
        const isValid = value >= 0;
        this.setFieldState(field, isValid, isValid ? '' : 'Sıralama 0 veya daha büyük olmalıdır.');
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

        // Validate sort order
        const sortOrderField = document.getElementById('sort_order');
        if (sortOrderField && !this.validateSortOrder(sortOrderField)) {
            isValid = false;
        }

        if (!isValid) {
            this.showErrorModal('Lütfen tüm alanları doğru şekilde doldurun.');
            
            // Scroll to first error
            const firstError = this.generalForm.querySelector('.error');
            if (firstError) {
                firstError.scrollIntoView({ behavior: 'smooth', block: 'center' });
                firstError.focus();
            }
        }

        return isValid;
    }

    /**
     * Set loading state for general form
     */
    setGeneralLoadingState(loading) {
        if (loading) {
            this.generalSubmitBtn.disabled = true;
            this.generalSubmitBtn.querySelector('.btn-text').style.opacity = '0';
            this.generalSubmitBtn.querySelector('.btn-loader').style.display = 'block';
            this.generalForm.classList.add('form-loading');
        } else {
            this.generalSubmitBtn.disabled = false;
            this.generalSubmitBtn.querySelector('.btn-text').style.opacity = '1';
            this.generalSubmitBtn.querySelector('.btn-loader').style.display = 'none';
            this.generalForm.classList.remove('form-loading');
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

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new UzmanlikEditHandler();
    
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
window.UzmanlikEditHandler = UzmanlikEditHandler;
