/**
 * Secenek Ekle (Option Add) Page JavaScript
 * Handles form submission, file uploads, drag & drop, and validation
 */

class OptionFormHandler {
    constructor() {
        this.form = document.getElementById('optionForm');
        this.submitBtn = document.getElementById('submitBtn');
        this.init();
    }

    init() {
        this.setupFormSubmission();
        this.setupFileUploads();
        this.setupColorInputs();
        this.setupRangeInputs();
        this.setupFormValidation();
        this.setupToggleSwitches();
    }

    /**
     * Setup form submission (non-AJAX). Validate and adjust values, then allow native submit.
     */
    setupFormSubmission() {
        this.form.addEventListener('submit', (e) => {
            // Validate before submit; block submit if invalid
            if (!this.validateForm()) {
                e.preventDefault();
                return;
            }

            // Convert max_upload_size (MB) to bytes just before submit
            const maxUploadInput = this.form.querySelector('input[name="max_upload_size"]');
            if (maxUploadInput && maxUploadInput.value) {
                const mb = parseInt(maxUploadInput.value, 10);
                if (!Number.isNaN(mb) && mb > 0) {
                    maxUploadInput.value = String(mb * 1024 * 1024);
                }
            }

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
            this.showErrorModal('Geçersiz dosya türü. Lütfen uygun bir dosya seçin.');
            return;
        }

        // Validate file size (2MB for images, 1MB for favicon, 5MB for default page)
        let maxSize = 2 * 1024 * 1024; // Default 2MB
        if (input.id === 'site_favicon_mid') {
            maxSize = 1024 * 1024; // 1MB for favicon
        } else if (input.id === 'default_page_mid') {
            maxSize = 5 * 1024 * 1024; // 5MB for default page
        }
        if (file.size > maxSize) {
            const maxSizeMB = maxSize / (1024 * 1024);
            this.showErrorModal(`Dosya boyutu ${maxSizeMB}MB'dan büyük olamaz.`);
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
     * Setup color input synchronization
     */
    setupColorInputs() {
        const colorGroups = [
            { picker: 'primary_color', text: 'primary_color' },
            { picker: 'secondary_color', text: 'secondary_color' },
            { picker: 'accent_color', text: 'accent_color' }
        ];

        colorGroups.forEach(group => {
            const picker = document.getElementById(group.picker);
            const textInput = picker.parentNode.querySelector('.color-text');

            // Convert hex to rgba
            const hexToRgba = (hex, alpha = 1) => {
                const r = parseInt(hex.slice(1, 3), 16);
                const g = parseInt(hex.slice(3, 5), 16);
                const b = parseInt(hex.slice(5, 7), 16);
                return `rgba(${r}, ${g}, ${b}, ${alpha})`;
            };

            // Convert rgba to hex
            const rgbaToHex = (rgba) => {
                const match = rgba.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
                if (match) {
                    const r = parseInt(match[1]).toString(16).padStart(2, '0');
                    const g = parseInt(match[2]).toString(16).padStart(2, '0');
                    const b = parseInt(match[3]).toString(16).padStart(2, '0');
                    return `#${r}${g}${b}`;
                }
                return '#000000';
            };

            // Update text when picker changes
            picker.addEventListener('input', (e) => {
                const rgbaValue = hexToRgba(e.target.value);
                textInput.value = rgbaValue;
            });

            // Update picker when text changes
            textInput.addEventListener('input', (e) => {
                const hexValue = rgbaToHex(e.target.value);
                picker.value = hexValue;
            });
        });
    }

    /**
     * Setup range input value display
     */
    setupRangeInputs() {
        const rangeInputs = document.querySelectorAll('.range-slider');
        
        rangeInputs.forEach(input => {
            const valueDisplay = input.parentNode.querySelector('.range-value');
            
            // Update display
            const updateValue = () => {
                valueDisplay.textContent = `${input.value}px`;
            };
            
            input.addEventListener('input', updateValue);
            updateValue(); // Initial value
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

        // Email validation
        const emailFields = this.form.querySelectorAll('input[type="email"]');
        emailFields.forEach(field => {
            field.addEventListener('blur', () => {
                this.validateEmail(field);
            });
        });

        // URL validation
        const urlFields = this.form.querySelectorAll('input[type="url"]');
        urlFields.forEach(field => {
            field.addEventListener('blur', () => {
                this.validateUrl(field);
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
     * Validate URL field
     */
    validateUrl(field) {
        const url = field.value.trim();
        if (!url) return true; // Not required
        
        try {
            new URL(url);
            this.setFieldState(field, true, '');
            return true;
        } catch {
            this.setFieldState(field, false, 'Geçerli bir URL girin.');
            return false;
        }
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

        // Validate URL fields
        const urlFields = this.form.querySelectorAll('input[type="url"]');
        urlFields.forEach(field => {
            if (!this.validateUrl(field)) {
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
        
        // Reset color inputs
        this.setupColorInputs();
        
        // Reset range inputs
        this.setupRangeInputs();
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
        
        // If success modal, redirect to options list
        if (modalId === 'successModal') {
            window.location.href = '/panel/secenekler';
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
    new OptionFormHandler();
    
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
window.OptionFormHandler = OptionFormHandler;

