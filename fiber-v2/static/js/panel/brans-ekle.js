/**
 * Brans Ekle (Branch Add) Page JavaScript
 * Handles form submission, file uploads, drag & drop, and validation
 */

class BransFormHandler {
    constructor() {
        this.form = document.getElementById('bransForm');
        this.submitBtn = document.getElementById('submitBtn');
        this.init();
    }

    init() {
        this.setupFormSubmission();
        this.setupFileUploads();
        this.setupFormValidation();
        this.setupToggleSwitches();
        this.setupNameToUrlSync();
        this.setupServicesSync();
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

            // Process services input before submission
            this.processServicesInput();

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

        // Validate file size (5MB max for branch images)
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
     * Setup services input synchronization
     */
    setupServicesSync() {
        const servicesInput = document.getElementById('services_input');
        if (!servicesInput) return;

        // Sync on input change with debouncing
        const debouncedSync = debounce(() => {
            this.processServicesInput();
        }, 500);

        servicesInput.addEventListener('input', debouncedSync);
        servicesInput.addEventListener('paste', () => {
            // Process immediately after paste
            setTimeout(() => {
                this.processServicesInput();
            }, 100);
        });
    }

    /**
     * Generate URL-friendly name from branch name
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
     * Process services input and convert to JSON array
     */
    processServicesInput() {
        const servicesInput = document.getElementById('services_input');
        const servicesField = document.getElementById('services');
        
        if (!servicesInput || !servicesField) return;

        const servicesText = servicesInput.value.trim();
        
        if (!servicesText) {
            // If empty, set empty JSON array
            servicesField.value = '[]';
            return;
        }

        // Split by newlines and process each service
        const servicesArray = servicesText
            .split('\n')
            .map(service => service.trim())
            .filter(service => service.length > 0) // Remove empty lines
            .map(service => {
                // Clean up the service name
                return service
                    .replace(/^[-•\*\+]\s*/, '') // Remove bullet points
                    .replace(/\s+/g, ' ') // Normalize whitespace
                    .trim();
            })
            .filter(service => service.length > 0); // Remove empty services after cleaning

        // Convert to JSON string and set as hidden input value
        const servicesJson = JSON.stringify(servicesArray);
        servicesField.value = servicesJson;
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
        
        // Reset services field
        const servicesInput = document.getElementById('services_input');
        const servicesField = document.getElementById('services');
        if (servicesInput) servicesInput.value = '';
        if (servicesField) servicesField.value = '[]';
        
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
        
        // If success modal, redirect to branches list
        if (modalId === 'successModal') {
            window.location.href = '/panel/branslar';
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
    new BransFormHandler();
    
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
window.BransFormHandler = BransFormHandler;
