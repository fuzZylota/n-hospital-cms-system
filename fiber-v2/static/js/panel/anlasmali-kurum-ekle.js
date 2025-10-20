/**
 * Anlasmali Kurum Ekle (Contract Partner Add) Page JavaScript
 * Handles form submission, file uploads, drag & drop, and validation
 */

class AnlasmaliKurumFormHandler {
    constructor() {
        this.form = document.getElementById('anlasmaliKurumForm');
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

            // Convert date inputs to proper format before submission
            this.convertDateInputs();

            // Show loading state
            this.setLoadingState(true);

            // Allow natural form submission to the action URL
        });
    }

    /**
     * Setup file upload areas with drag & drop functionality
     */
    setupFileUploads() {
        const fileUploadAreas = document.querySelectorAll('.file-upload-area');
        
        fileUploadAreas.forEach(area => {
            const input = area.querySelector('.file-input');
            const content = area.querySelector('.file-upload-content');
            const preview = area.querySelector('.file-preview');
            const previewImage = area.querySelector('.preview-image');
            const removeBtn = area.querySelector('.file-remove');
            const browseLink = area.querySelector('.file-browse');

            // Click to browse
            browseLink.addEventListener('click', (e) => {
                e.preventDefault();
                input.click();
            });

            area.addEventListener('click', (e) => {
                if (e.target === area || e.target === content) {
                    input.click();
                }
            });

            // Drag & drop events
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

            // File input change
            input.addEventListener('change', (e) => {
                if (e.target.files.length > 0) {
                    this.handleFileSelect(input, e.target.files[0], content, preview, previewImage);
                }
            });

            // Remove file
            removeBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                this.removeFile(input, content, preview);
            });
        });
    }

    /**
     * Handle file selection and preview
     */
    handleFileSelect(input, file, content, preview, previewImage) {
        // Validate file type
        const allowedTypes = input.accept.split(',').map(type => type.trim());
        const isValidType = allowedTypes.some(type => {
            if (type.startsWith('.')) {
                return file.name.toLowerCase().endsWith(type.toLowerCase());
            } else if (type.includes('*')) {
                const mainType = type.split('/')[0];
                return file.type.startsWith(mainType);
            } else {
                return file.type === type;
            }
        });

        if (!isValidType) {
            this.showErrorModal('Geçersiz dosya türü. Lütfen uygun bir görsel dosyası seçin.');
            return;
        }

        // Validate file size (5MB max for contract partner logos)
        const maxSize = 5 * 1024 * 1024; // 5MB
        if (file.size > maxSize) {
            this.showErrorModal('Dosya boyutu 5MB\'dan büyük olamaz.');
            return;
        }

        // Create file list for input
        const dt = new DataTransfer();
        dt.items.add(file);
        input.files = dt.files;

        // Show preview
        if (file.type.startsWith('image/')) {
            const reader = new FileReader();
            reader.onload = (e) => {
                previewImage.src = e.target.result;
                content.style.display = 'none';
                preview.style.display = 'flex';
            };
            reader.readAsDataURL(file);
        }
    }

    /**
     * Remove selected file
     */
    removeFile(input, content, preview) {
        input.value = '';
        content.style.display = 'block';
        preview.style.display = 'none';
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

        // Email validation
        const emailFields = this.form.querySelectorAll('input[type="email"]');
        emailFields.forEach(field => {
            field.addEventListener('blur', () => {
                this.validateEmail(field);
            });
        });

        // Phone validation
        const phoneFields = this.form.querySelectorAll('input[type="tel"]');
        phoneFields.forEach(field => {
            field.addEventListener('blur', () => {
                this.validatePhone(field);
            });
        });

        // Date validation
        const dateFields = this.form.querySelectorAll('input[type="date"]');
        dateFields.forEach(field => {
            field.addEventListener('blur', () => {
                this.validateDate(field);
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
     * Convert date inputs to proper format before form submission
     */
    convertDateInputs() {
        const dateInputs = this.form.querySelectorAll('input[type="date"]');
        
        dateInputs.forEach(input => {
            if (input.value) {
                const convertedDate = this.convertDateForBackend(input.value);
                // Create a hidden input with the converted date
                const hiddenInput = document.createElement('input');
                hiddenInput.type = 'hidden';
                hiddenInput.name = input.name + '_converted';
                hiddenInput.value = convertedDate;
                this.form.appendChild(hiddenInput);
            }
        });
    }

    /**
     * Convert date from frontend format to backend format
     */
    convertDateForBackend(dateValue) {
        if (!dateValue || dateValue === '') {
            // Return Go's zero time format for empty dates
            return '0001-01-01T00:00:00Z';
        }
        
        // If it's already in the correct format (YYYY-MM-DD), convert to Go time format
        if (dateValue.match(/^\d{4}-\d{2}-\d{2}$/)) {
            // Convert YYYY-MM-DD to Go time format (2006-01-02T15:04:05Z07:00)
            return dateValue + 'T00:00:00Z';
        }
        
        // If it's in the old format "00:00, 01/01/0001", convert to Go zero time
        if (dateValue.includes('00:00, 01/01/0001')) {
            return '0001-01-01T00:00:00Z';
        }
        
        // Try to parse other date formats
        const date = new Date(dateValue);
        if (!isNaN(date.getTime())) {
            return date.toISOString();
        }
        
        // Return Go's zero time format for invalid dates
        return '0001-01-01T00:00:00Z';
    }

    /**
     * Generate URL-friendly name from contract partner name
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
     * Validate email field
     */
    validateEmail(field) {
        const email = field.value.trim();
        if (!email) return true; // Not required
        
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        const isValid = emailRegex.test(email);
        
        this.setFieldState(field, isValid, isValid ? '' : 'Geçerli bir e-posta adresi girin.');
        return isValid;
    }

    /**
     * Validate phone field
     */
    validatePhone(field) {
        const phone = field.value.trim();
        if (!phone) return true; // Not required
        
        // Basic phone validation - accepts various formats
        const phoneRegex = /^[\+]?[0-9\s\-\(\)]{10,}$/;
        const isValid = phoneRegex.test(phone);
        
        this.setFieldState(field, isValid, isValid ? '' : 'Geçerli bir telefon numarası girin.');
        return isValid;
    }

    /**
     * Validate date field
     */
    validateDate(field) {
        const date = field.value;
        if (!date) return true; // Not required
        
        const inputDate = new Date(date);
        const today = new Date();
        today.setHours(0, 0, 0, 0);
        
        let isValid = true;
        let errorMessage = '';

        // Check if date is valid
        if (isNaN(inputDate.getTime())) {
            isValid = false;
            errorMessage = 'Geçerli bir tarih girin.';
        } else {
            // Check if end date is after start date
            if (field.id === 'contract_end_date') {
                const startDateField = document.getElementById('contract_start_date');
                if (startDateField && startDateField.value) {
                    const startDate = new Date(startDateField.value);
                    if (inputDate <= startDate) {
                        isValid = false;
                        errorMessage = 'Bitiş tarihi başlangıç tarihinden sonra olmalıdır.';
                    }
                }
            }
        }
        
        this.setFieldState(field, isValid, errorMessage);
        return isValid;
    }

    /**
     * Validate number field
     */
    validateNumber(field) {
        const value = field.value.trim();
        if (!value) return true; // Not required
        
        const num = parseFloat(value);
        const min = field.getAttribute('min');
        const max = field.getAttribute('max');
        
        let isValid = true;
        let errorMessage = '';

        if (isNaN(num)) {
            isValid = false;
            errorMessage = 'Geçerli bir sayı girin.';
        } else {
            if (min !== null && num < parseFloat(min)) {
                isValid = false;
                errorMessage = `Değer en az ${min} olmalıdır.`;
            }
            if (max !== null && num > parseFloat(max)) {
                isValid = false;
                errorMessage = `Değer en fazla ${max} olmalıdır.`;
            }
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

        // Validate email fields
        const emailFields = this.form.querySelectorAll('input[type="email"]');
        emailFields.forEach(field => {
            if (!this.validateEmail(field)) {
                isValid = false;
            }
        });

        // Validate phone fields
        const phoneFields = this.form.querySelectorAll('input[type="tel"]');
        phoneFields.forEach(field => {
            if (!this.validatePhone(field)) {
                isValid = false;
            }
        });

        // Validate date fields
        const dateFields = this.form.querySelectorAll('input[type="date"]');
        dateFields.forEach(field => {
            if (!this.validateDate(field)) {
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
        
        // Reset file uploads
        const filePreviews = this.form.querySelectorAll('.file-preview');
        const fileContents = this.form.querySelectorAll('.file-upload-content');
        
        filePreviews.forEach(preview => preview.style.display = 'none');
        fileContents.forEach(content => content.style.display = 'block');
        
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
        
        // If success modal, redirect to contract partners list
        if (modalId === 'successModal') {
            window.location.href = '/panel/anlasmali-kurumlar';
        }
    }, 300);
}

/**
 * Utility functions
 */
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
    new AnlasmaliKurumFormHandler();
    
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
window.AnlasmaliKurumFormHandler = AnlasmaliKurumFormHandler;
